# FAQ & quick reference

Rapid answers for review-time decisions; every entry links to the deep
chapter.

## Decision tables

### Channel or mutex?

| Situation | Answer |
|---|---|
| Hand a value to one goroutine, transfer ownership | channel |
| Signal "done"/"data ready" | channel (close or struct{}) |
| Protect a map/struct read+written in place | mutex |
| Count something | atomic |
| Many goroutines, one queue | buffered channel + pool |
| Cache with hot keys | mutex + map (or sharded) |

### Which sync primitive?

| Need | Primitive |
|---|---|
| Mutual exclusion | `sync.Mutex` |
| Read-mostly protection | `sync.RWMutex` (measure!) |
| One-time init | `sync.Once` |
| Wait for N tasks | `sync.WaitGroup` / `errgroup` |
| Single word, lock-free | `sync/atomic` |
| Special access shapes | `sync.Map` (see ch. 04 for the shapes) |

### Bounded or not?

| Traffic shape | Design |
|---|---|
| Steady, producers ≈ consumers | unbuffered handoff |
| Bursty producers | bounded buffer sized to burst |
| Untrusted/large payloads | bounded + drop/shed with metric |
| Must never lose work | bounded + persistent queue upstream |

## Quick answers

**Why does my program panic with `all goroutines are asleep - deadlock!`?**
Every goroutine is blocked on channel/lock operations and none can
progress — the runtime detects total deadlock. Partial deadlocks (one
component blocked, others fine) hang instead; find them with the
goroutine profile. See [07-pitfalls](07-pitfalls.md).

**Why `defer cancel()` on contexts I never cancel explicitly?**
The context holds a timer and a parent link until cancelled; on hot
paths that's a real leak. See [03-context](03-context.md).

**When is it safe to close a channel?**
When you can prove no send will happen again: you own all sends (owner
closes), or every sender is joined (`wg.Wait()` before close). See
[01-goroutines-and-channels](01-goroutines-and-channels.md).

**My select has two ready cases — which runs?**
Random, uniformly. This prevents starvation; don't build ordering on it.
See [02-select-and-timeouts](02-select-and-timeouts.md).

**Buffered channel as a lock-free queue?**
It's a bounded queue, not a lock-free anything — full buffers block
sends (that's backpressure, the feature). Sync-free semantics still come
from channel rules, not atomics.

**How many goroutines is too many?**
When memory, scheduler latency, or contention profiles say so. Workers
pool at #cores (CPU-bound) or #deps (I/O-bound); per-connection
goroutines in servers are normal at 100k+. Measure; don't folklore.

**Goroutine leaks — how do I check in CI?**
goleak, or the runtime.NumGoroutine baseline pattern in
[07-pitfalls](07-pitfalls.md).

**Do I need a mutex to read a value written once at startup?**
Only if the write happens after readers start. Initialize before
spawning goroutines, or use `sync.Once`; the memory model requires an
edge.

**errgroup vs WaitGroup?**
`errgroup.Group` = WaitGroup + first-error capture + ctx linking
(`WithContext`). Default to it for task fan-out that can fail; plain
WaitGroup for pure joins. See [05-patterns](05-patterns.md).

**Why does `sync.Map` exist if maps+mutex work?**
Two specific shapes: write-once-read-many keys, and disjoint key sets
per goroutine. Outside those, mutex+map wins. See
[04-sync-primitives](04-sync-primitives.md).

**Can a channel be nil? What happens?**
Yes — every operation blocks forever. Used deliberately to disable
select cases; accidentally, it's a deadlock. See
[05-zero-values chapter of 01](../01-go-fundamentals/05-zero-values.md).

**How do I run something exactly once across a cluster, not just a
process?**
`sync.Once` is per-process. Distributed once needs consensus or lease
acquisition — see [16-distributed-systems](../16-distributed-systems/).

## Anti-pattern gallery

```go
// 1. Fire-and-forget goroutine (no exit path, panics kill the process)
go doWork()

// 2. time.After in a hot loop (allocates, resets deadline each item)
select {
case v := <-ch:
	process(v)
case <-time.After(timeout):
	return
}

// 3. Exported mutex (callers can lock it, deadlock you)
type Server struct {
	Mu sync.Mutex // MISTAKE: unexported mu
}

// 4. Mutex copy (value receiver on struct with a mutex)
func (s Server) Snapshot() Server { return s } // copies a lock

// 5. Blocking send in a worker without ctx
results <- process(job) // hangs forever if consumer died
```

Each fixed form lives in the chapters above; this gallery exists so a
reviewer can link it in a PR comment.

## Interview rapid-fire

- Buffered vs unbuffered? → rendezvous vs capacity-decoupled.
- Mutex vs RWMutex? → measure; RWMutex wins only read-heavy.
- Select with default? → non-blocking attempt.
- `close(ch)` semantics? → drains, then zero values; broadcast signal.
- ctx values for business data? → no; request metadata only.
- Atomic for two related fields? → no; one invariant needs one lock.
- Race detector at runtime or compile time? → runtime, instrumented.
- WaitGroup.Add inside the goroutine? → no; race with Wait.

## Further Reading

- [Go memory model](https://go.dev/ref/mem) — the ground truth for all of it
- [Practical Go: Concurrency](https://dave.cheney.net/practical-go/presentations/qcon-china.html) — Cheney's concurrency counsel
- [Concurrency in Go](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/) — the book this section most agrees with
