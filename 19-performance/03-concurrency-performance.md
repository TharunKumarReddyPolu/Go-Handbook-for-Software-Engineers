# Concurrency performance

## Why Does This Matter?

Concurrency fixes *structure*, not automatically *speed*. The same
goroutine that parallelizes your work can serialize it on a hot channel
or a shared mutex — and the failure mode is subtle because the code
looks concurrent. This chapter is about the performance physics of
concurrent Go: where the costs live, how to see them, and the patterns
that scale.

## Mental Model

Three taxes every concurrent design pays:

1. **Coordination tax** — channel ops, mutex acquisitions, atomics: each
   is cheap alone, expensive multiplied by hot-path call count.
2. **Contention tax** — the same coordination point hit by many
   goroutines serializes them; time waiting is invisible to CPU
   profiles.
3. **Structure tax** — goroutine startups, stack growth, scheduling
   hops, GC scanning more live objects.

The design question is always: which tax dominates *this* workload, and
which structural change removes it?

## Channel costs — the numbers that matter

A buffered channel send/receive costs roughly 50-100ns uncontended —
fine at request rates, expensive at item rates. Rules of thumb:

- **Don't channel-hop per item** in hot loops. A pipeline that sends
  10M items through 3 stages pays 30M+ channel ops; batching changes
  that to 30M/batchSize.
- **Uncontended ≠ under load.** Every sender+receiver pair added to a
  hot channel adds scheduling wakeups; measure with `b.RunParallel`.
- **`select` grows with case count** — keep hot selects small.

```go
// PER-ITEM: 10M channel ops
for item := range in {
	out <- transform(item)
}

// BATCHED: 10M/batch channel ops — same data, fraction of the tax
var batch []Item
for item := range in {
	batch = append(batch, transform(item))
	if len(batch) == 256 {
		out <- batch       // hand off the slice; receiver must not mutate after
		batch = make([]Item, 0, 256)
	}
}
if len(batch) > 0 {
	out <- batch
}
```

Slice batches are zero-copy handoffs — ownership transfers with the
channel; see [02-memory-and-allocations](02-memory-and-allocations.md)
for why that beats copying.

## Lock contention — the invisible killer

Uncontended `Lock/Unlock` is ~15-25ns. Contended locks wait on the
scheduler: the *wait* is 1000x the *operation*. Symptoms: p99 climbs
while CPU is idle; mutex profiles show one symbol.

### Finding it

```bash
go test -mutexprofile=mutex.out ./...
go tool pprof mutex.out        # who holds and waits
go test -blockprofile=block.out ./...
go tool pprof block.out        # where goroutines wait (channels, locks, I/O)
```

### Fixing it — in order of preference

1. **Shard**: split one hot lock into N (by key hash). 16 shards ≈ 16x
   less contention, no semantic change.
2. **Reduce hold time**: move I/O, allocation, and computation outside
   the critical section; lock for memory operations only.
3. **Change the model**: give one goroutine ownership (channel the
   *requests*, not the data); readers get a snapshot, writers get
   sequencing. Removes the lock entirely.
4. **Atomic when single-word**: a hot counter under mutex becomes
   `atomic.Int64` — but two related values still need one lock (see the
   sync chapter's invariant rule).

```go
// SHARDED MAP — the standard fix for hot registries/caches
type ShardedMap struct {
	shards [16]struct {
		sync.RWMutex
		m map[string]any
	}
}

func (s *ShardedMap) shardFor(key string) *shard { ... } // fnv/hash key % 16

func (s *ShardedMap) Get(key string) (any, bool) {
	sh := s.shardFor(key)
	sh.RLock()
	defer sh.RUnlock()
	v, ok := sh.m[key]
	return v, ok
}
```

## Goroutine economics

- **Start**: ~2KB stack, sub-microsecond — *per-unit* spawning is fine
  for per-request/per-connection work.
- **Per-item spawning** in hot loops: the structure tax adds up —
  scheduling churn, GC roots. Pool instead (the worker-pool pattern
  exists for this).
- **Blocked goroutines are cheap but not free** — 100k parked
  goroutines hold real memory and GC scan time; bound concurrency at
  the intake (stage 4 pattern in the concurrency section).

## Batching — the universal performance tool

Almost every latency/throughput trade in systems is a batching decision:

| Layer | Batch win | Cost |
|---|---|---|
| DB writes | 1 round-trip vs N | added latency for the first item |
| Network I/O | fewer syscalls | buffering memory |
| GC | fewer live objects churned | batch lifetime |
| Kafka produce | high throughput | linger latency (see 18-kafka) |

The shape: accumulate to a threshold OR a deadline — whichever first —
then flush. The `select`-with-ticker flusher in the concurrency chapter
is the canonical implementation.

## Zero-copy techniques — with the safety tax

```go
// COPY: string <-> []byte conversions duplicate memory
s := string(b)

// ZERO-COPY (read-only): unsafe.String shares the buffer
s := unsafe.String(&b[0], len(b)) // b must not be mutated while s lives
```

Legitimate in audited hot paths (protocol decoders); dangerous as a
habit because aliasing rules are manual and the race detector will not
catch buffer misuse through `unsafe`. Standard-library zero-copy wins
without the risk: `strings.Builder` (no copy between builds),
`io.WriterTo`/`io.ReaderFrom` implementations (skips intermediate
buffers), `http.Client` with reused transports (connection pooling is
zero-copy for the protocol machinery).

## Caching — the final boss

A cache converts CPU/IO cost into a memory + invalidation cost. Rules:

- Bound it (LRU, size-capped) — unbounded caches are memory incidents
  (see [07-pitfalls in 08-concurrency](../08-concurrency/07-pitfalls.md)).
- Singleflight the misses — 10k concurrent cache misses on one key must
  be one backend call, not 10k (`golang.org/x/sync/singleflight`).
- Metrics on hit rate and fill latency — a cache you can't see is a
  liability.
- Sharded locks or sync.Map per the access-shape rules (sync chapter).

## Basic Example — channel vs mutex at the same job

The repo's `examples/channels-vs-mutex` benchmark (run
`go test ./19-performance/... -bench=Counter`):

```text
BenchmarkCounter/Mutex-8        89142102    13.4 ns/op
BenchmarkCounter/Channel-8       9344987   128.0 ns/op
BenchmarkCounter/Atomic-8      291928475     4.1 ns/op
```

Same job, three structures: atomic 3x faster than mutex, channel 10x
slower — *for this job*. That's the point: the structure follows the
problem (protect one word → atomic; protect an invariant → mutex;
transfer ownership → channel), and the benchmark proves the fit.

## Common Mistakes

- **Adding goroutines to fix latency** — if the bottleneck is one
  contended lock, more workers deepen the queue. Profile first.
- **Buffered channels as "performance"** — buffering trades latency for
  throughput and hides backpressure; the fix for a slow consumer is
  rarely a bigger buffer.
- **Sharding by iteration count, not measurement** — shard counts from
  folklore (always 16) can be wrong by an order of magnitude for your
  keys; benchmark 4/16/64.
- **Ignoring GC interplay** — pooling that keeps objects alive longer
  can *increase* GC work (bigger live set). Measure the whole system,
  not the allocation counter alone.

## Idiomatic Go

- The standard library first: `sync.Pool`, `singleflight`,
  `httputil.ReverseProxy`'s pooling — decades of tuning, one import away.
- Comment hot paths with their benchmark names; the next reader needs
  the evidence trail.

## Concurrency Considerations

Every fix here interacts with correctness: sharded maps change iteration
semantics (no global snapshot); singleflight collapses calls (misses
share errors too); batching changes failure granularity (a lost batch is
N items — the [25-fintech](../25-fintech-with-go/) sections treat this
as an integrity question, not just a performance one).

## Security Considerations

- Per-key/per-source rate limits are also contention shields: without
  them, one tenant's traffic can monopolize your shared locks (a
  cross-tenant DoS; see [21-security](../21-security/)).
- Batching PII across requests in shared buffers needs strict reset
  discipline — the pooled-buffer leak again.

## Testing Strategy

- `b.RunParallel` benchmarks for every sharded/locked structure, run in
  CI nightly: contention regressions are statistical, not instant.
- Mutex/block profiles captured in load tests, diffed release-over-
  release like CPU profiles.

## Interview Questions

1. *A mutex shows 40% of wall time in profiles. Options?* — Shard,
   shrink hold time, change ownership model, atomic-if-single-word — in
   that order, each with measurement.
2. *When is a channel faster than a mutex?* — When it *removes* work:
   ownership transfer eliminates lock bookkeeping; for pure protection
   of shared state, never.
3. *Design a 1M-item/s ingestion pipeline.* — Batched handoffs, sharded
   intake, per-stage buffering, backpressure at the edge; grade on
   where the coordination points ended up.

## Practice Exercises

1. Write the three-counter benchmark from this chapter; verify the
   ordering on your machine and explain deviations (frequency, cache
   lines, core count).
2. Take a contended map in a load test; shard it 4/16/64 ways and plot
   p99 vs shard count — find your workload's knee.
3. Add singleflight to a cache-miss path; measure backend QPS before/
   after under a thundering-herd load test.

## Further Reading

- [Introducing the Go Race Detector](https://go.dev/blog/race-detector) — profiling pair for contention
- [Advanced Go Concurrency Patterns](https://talks.golang.org/2013/advconc.slide) — the cancellation/loading discipline
- [runtime/metrics](https://pkg.go.dev/runtime/metrics) — scheduler and GC quantiles as code
