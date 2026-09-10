# Sync primitives

## Why Does This Matter?

Channels get the attention; `sync` gets the work done. Real services
protect shared state — caches, registries, connection pools, counters —
and the vocabulary for that is the `sync` package. Choosing the wrong
primitive is how systems acquire lock contention at 10k RPS and nobody
can explain why.

## Mental Model

One decision table first — *protection in place vs transfer of
ownership*:

| You are... | Use |
|---|---|
| Protecting a struct's fields from concurrent access | `Mutex` / `RWMutex` |
| Counting / simple flags under contention | `atomic` |
| One-time initialization | `sync.Once` |
| Waiting for N tasks | `WaitGroup` |
| Handing values between goroutines | channels |
| Many-key shared cache with hot keys | sharded mutexes or `sync.Map` |

A mutex protects *memory*, a channel *coordinates* — most designs need
both, and knowing which problem you are solving is half the skill.

## Mutex — protect the invariant, not the line of code

```go
type Counter struct {
	mu sync.Mutex
	n  int64
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock() // unlock on every path, including panics
	c.n++
}

func (c *Counter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
```

Rules with teeth:

- **Lock/defer-unlock adjacent.** Reviewers read the pairing instantly.
- **Never call out of your own mutex** (no method calls that might lock
  again — that's how deadlocks happen).
- **The mutex guards an invariant**, not a statement: if `a` and `b` must
  be consistent, one lock covers both.
- **Zero value is ready.** `var mu sync.Mutex` — never copy it after use
  (`go vet` catches copying).

## RWMutex — only when reads dominate *measurably*

```go
type Registry struct {
	mu   sync.RWMutex
	svc  map[string]Endpoint
}

func (r *Registry) Get(name string) (Endpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.svc[name]
	return e, ok
}

func (r *Registry) Set(name string, e Endpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.svc[name] = e
}
```

RWMutex is not free: writer starvation protection, more bookkeeping, and
RLock is slower than Mutex's fast path. Below high read contention
(measure!), plain Mutex often wins. Contended single-key locks are also
a shard boundary signal — see below.

## sync.Once — initialization as an idempotent fact

```go
type Client struct {
	once    sync.Once
	conn    *Conn
	initErr error
}

func (c *Client) Conn(ctx context.Context) (*Conn, error) {
	c.once.Do(func() {
		c.conn, c.initErr = dial(ctx) // first caller wins; others wait
	})
	return c.conn, c.initErr
}
```

`Once` runs exactly one function, and other callers block until it
completes. Two caveats: a panicking `Do` still counts as done (the
failure is sticky), and `Do` cannot take arguments — capture them in the
closure.

## WaitGroup — the collector

```go
var wg sync.WaitGroup
for _, job := range jobs {
	wg.Add(1) // before starting, not inside the goroutine
	go func(j Job) {
		defer wg.Done()
		process(j)
	}(job)
}
wg.Wait()
```

Since Go 1.25, `wg.Go(f)` replaces the Add/go/defer-Done triplet:

```go
wg := sync.WaitGroup{}
for _, job := range jobs {
	wg.Go(func() { process(job) }) // Add/Done handled for you
}
wg.Wait()
```

(Loop variables are per-iteration since Go 1.22, so the old `job := job`
capture hack is gone — see [meta/versioning.md](../meta/versioning.md).)

## atomic — the scalpel

```go
type Metrics struct {
	requests atomic.Int64
	errors   atomic.Int64
}

func (m *Metrics) Record(err error) {
	m.requests.Add(1)
	if err != nil {
		m.errors.Add(1)
	}
}
```

Atomics are lock-free single-word operations: increment, load, store,
compare-and-swap. Two hard limits:

1. **One value.** Two related atomics are not one consistent state — if
   `requests` and `errors` must be read consistently, they are a struct
   under one Mutex, not two atomics.
2. **No compound invariants.** "Move 10 from A to B" is not atomic as two
   atomic ops.

CAS loops (compare-and-swap) are the advanced form:

```go
for {
	old := c.state.Load()
	new := computeState(old)
	if c.state.CompareAndSwap(old, new) {
		break // won the race; otherwise retry with fresh state
	}
}
```

## sync.Map — the specialty tool

Optimized for two specific shapes: (1) keys written once, read many
(caches, registries), (2) disjoint goroutines touching disjoint keys.
It is *slower* than `Mutex + map` for write-heavy or overlapping access.

```go
var cache sync.Map // map[string]*Entry

if v, ok := cache.Load(key); ok { ... }
cache.Store(key, entry)
cache.LoadOrStore(key, buildEntry()) // one-shot memoization
```

Default to `Mutex + map` until a profile shows the contention; reach for
`sync.Map` when the access shape matches its optimization, and say the
shape out loud in a comment.

## Basic Example — all together: a rate-limited client registry

```go
type Registry struct {
	mu      sync.RWMutex
	clients map[string]*Client
	lookups atomic.Int64
}

func (r *Registry) Client(name string, dial func() (*Client, error)) (*Client, error) {
	r.mu.RLock()
	c, ok := r.clients[name]
	r.mu.RUnlock()
	if ok {
		r.lookups.Add(1)
		return c, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.clients[name]; ok { // re-check: someone won the race
		return c, nil
	}
	c, err := dial()
	if err != nil {
		return nil, err
	}
	r.clients[name] = c
	return c, nil
}
```

Note the double-checked pattern — cheap read path, serialized write path,
no double-dial.

## Common Mistakes

- **Copying a mutex** (value receiver on a struct holding one) — vet
  catches it; the copy silently guards nothing.
- **Locking around long I/O** — hold locks for memory operations;
  if you must wait on the network under a lock, redesign.
- **Two atomics for one invariant** — covered above; it *will* tear.
- **RWMutex by default** — measure; it's often slower than Mutex.
- **`wg.Add` inside the goroutine** — race with `wg.Wait()`: the counter
  may hit zero before Add runs. Add before `go`.
- **Reusing `sync.Once` for "run every N"** — Once is once, forever.

## Idiomatic Go

- Prefer designing away shared state (ownership transfer) — the best
  mutex is the one that doesn't exist.
- Keep locked sections tiny and boring; allocate/IO outside the lock.
- Expose locking via methods; never export a mutex (callers will misuse it).

## Performance Considerations

- Uncontended lock/unlock: ~15-25ns; under contention it's scheduler
  wait time that dominates, not the lock op.
- Contended mutexes degrade to OS-level waiting; if your pprof shows
  `sync.(*Mutex).Lock` hot, shard, batch, or re-own the data.
- Atomics under heavy contention serialize on the cacheline —
  sharded counters beat one hot atomic at scale.

## Concurrency Considerations

- All of `sync` provides happens-before edges (a successful Lock pairs
  with Unlock; Once completes before any Do returns). Race-free is not
  the same as *deadlock-free* — ordering of lock acquisition across
  multiple locks must be global (always A→B, never B→A).
- Go's mutexes are not reentrant: locking twice in one goroutine
  deadlocks. Design non-reentrant APIs.

## Security Considerations

- Long critical sections amplify DoS: a slow handler holding a lock
  queues every request behind it. Bound work *before* acquiring locks.
- Zero-value readiness means no uninitialized-window attacks; but
  double-checked patterns must re-verify invariants under the write lock.

## Testing Strategy

- Tests under `-race` on every run; lock bugs surface as reports, not
  hangs.
- Stress with `go test -count=100` and shuffled ordering.
- Contention benchmarks: benchmark with `b.RunParallel` to see the
  scaling shape, not just the single-threaded cost.

## Interview Questions

1. *Mutex vs channel — how do you choose?* — Protect-in-place vs
   transfer-ownership; both examples.
2. *Why is RWMutex sometimes slower than Mutex?* — RLock bookkeeping +
   writer-priority coordination; wins only under heavy read share.
3. *Write a concurrent-safe lazy singleton. What are the failure modes?* —
   sync.Once; sticky panic, arg capture, first-caller blocking.
4. *What breaks if two atomics model one state?* — Torn reads across the
   pair; needs one lock or one word.

## Practice Exercises

1. Build a sharded map (16 shards, key-hash to shard); benchmark it
   against Mutex+map and sync.Map with 70/30 read/write at 8 goroutines.
2. Implement a singleflight utility (collapse duplicate in-flight
   lookups) with sync.Once + mutex; test that 100 concurrent identical
   calls produce exactly one backend call.
3. Find the deadlock: write two goroutines each acquiring two mutexes in
   opposite order; fix with global ordering; prove it with a
   `timeout + goroutine dump` test.

## Further Reading

- [sync package docs](https://pkg.go.dev/sync) — every primitive's contract
- [Go memory model](https://go.dev/ref/mem) — the happens-before edges sync provides
- [Introducing sync.Map](https://go.dev/blog/sync-map) — official rationale for its niche
