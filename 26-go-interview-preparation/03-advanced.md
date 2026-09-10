# Advanced track

Goroutines, channels, synchronization, memory, GC, scheduler. This is
where systems interviews at infrastructure companies are decided.

## Goroutines & scheduler

**Q: What are G, M, P? What happens on a blocking syscall?**

Strong answer: G = goroutine (stack + state), M = OS thread, P =
scheduling context required to run Go code (GOMAXPROCS of them).
Blocking syscall: the M blocks in the kernel; its P is handed to
another M so other goroutines keep running. Contrast: blocking in Go
code (channel/mutex) parks the G without involving a thread. —
[06-concurrency-vs-parallelism](../08-concurrency/06-concurrency-vs-parallelism.md)

**Q: A container has a 2-CPU quota on a 64-core host. What's the
GOMAXPROCS problem and the fix?**

Strong answer: defaults to 64 → scheduler oversubscribes the quota →
throttling, latency spikes, GC runs starved. Fix: set GOMAXPROCS to
the quota (or automaxprocs-class libs). The follow-up is GC interplay:
with GOMAXPROCS 64 under a 2-CPU cgroup, GC assist behaves erratically.

**Q: How does async preemption work and when did it land?**

Strong answer: Go 1.14 — the runtime preempts running goroutines at
safe points (based on signals) instead of waiting for function calls,
so tight loops can't starve the scheduler/GC. Still: correctness must
not depend on preemption timing.

## Channels

**Q: Send on closed channel? Receive on closed? Close twice?**

Strong answer: panic / drain-then-zero-values (immediate) / panic.
Then the design question they're really asking: *who closes?* — the
owner (sole sender), or after wg.Wait() when senders are joined. —
[01-goroutines-and-channels](../08-concurrency/01-goroutines-and-channels.md)

**Q: Select with multiple ready cases?**

Strong answer: uniform random pick — starvation prevention. Priority
needs nested selects (high-channel first with default), and the cost
is a fairness trade you should justify.

**Q: Where does backpressure come from in a pipeline?**

Strong answer: blocking sends on unbuffered/small channels — the
slowest stage throttles upstream. Buffers *hide* pressure (until
memory dies); they don't remove it. The policy choice: block, shed
(select+default), or buffer knowingly with a reason. —
[05-patterns](../08-concurrency/05-patterns.md)

## Synchronization

**Q: Mutex vs channel — decide for (a) a shared cache (b) work
distribution (c) a shutdown signal.**

Strong answer: (a) mutex/RWMutex (protect in place), (b) channel +
worker pool (transfer ownership), (c) closed channel broadcast or
context. The reasoning: channels coordinate and transfer; mutexes
protect. — [04-sync-primitives](../08-concurrency/04-sync-primitives.md)

**Q: What does the race detector actually detect? What misses?**

Strong answer: unsynchronized conflicting memory accesses *at runtime,
during instrumented execution* (happens-before analysis). Misses:
logical race conditions without memory races (check-then-act on well-
synchronized state), races that didn't occur this run, and anything in
untested paths. It's a dynamic tool, not a proof. —
[07-pitfalls](../08-concurrency/07-pitfalls.md)

**Q: Two atomics modeling one state — why wrong?**

Strong answer: torn reads across the pair; no single happens-before
edge covers both. One invariant needs one lock (or one word). CAS
loops exist but are the last resort, not the style.

**Q: Write deadlock-prone code, then fix it three ways.**

Strong answer: two goroutines, two locks, opposite order. Fixes: (1)
global lock ordering (by ID), (2) restructure to one lock, (3) channel
ownership so no shared lock exists. Bonus: mention the runtime panics
only on *total* deadlock; partial hangs silently.

## Memory & GC

**Q: What decides stack vs heap? How do you check?**

Strong answer: escape analysis at compile time — values that outlive
the frame (returned pointers, interface boxing, closures, channel
sends) heap-allocate; the rest stay on the per-goroutine stack. Check:
`go build -gcflags=-m`. — [02-memory-and-allocations](../19-performance/02-memory-and-allocations.md)

**Q: Your Go service's p99 creeps up as traffic grows. GC suspicion?
Walk me through it.**

Strong answer: check allocation rate first (heap profile,
alloc_objects) — GC CPU is proportional to churn; gctrace for cycle
frequency/pauses; GOMEMLIMIT for container safety; fix the top
allocation site before touching GOGC. The graded insight: *rate*, not
heap size, is the usual latency killer.

**Q: How does GOGC interact with GOMEMLIMIT?**

Strong answer: GOGC sets the growth target relative to live heap
(default 100% → collect at 2x live); GOMEMLIMIT (1.19+) is a soft
absolute ceiling the GC runs harder to respect — the container-friendly
knob. Modern posture: GOMEMLIMIT at ~90% of the container, GOGC raised
or default.

**Q: Goroutine leaks — name three causes and the CI detection.**

Strong answer: producer with no receiver; receive-from-never-closed
channel; blocking call without ctx. Detection: goleak / NumGoroutine
baselines in tests, pprof goroutine profile in prod (identical stacks
= blocked at same line). — [07-pitfalls](../08-concurrency/07-pitfalls.md)

## Live-code drills

1. Worker pool with cancellation — they grade: select on ctx in
   send/receive, owner closes, no leaks. (The 08-concurrency stage 3
   is exactly this; write it cold.)
2. Rate-limited API client — token bucket from channels or
   x/time/rate; discuss burst vs sustained.
3. Fan-in N channels with clean close — WaitGroup + closer goroutine;
   they'll probe leaks if you forget the closer.
4. concurrent-safe LRU — hash map + doubly-linked list; then discuss
   sharding under contention.

## Red flags interviewers watch for

- "Just add goroutines" without identifying the serialization point.
- Channels-everywhere (a mutex-shaped problem solved with channels).
- Believing buffered channels fix consumer slowness.
- No mention of `-race` in any concurrency answer.
- Optimizing without a benchmark in the performance story.
