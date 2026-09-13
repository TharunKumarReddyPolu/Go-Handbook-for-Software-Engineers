# 09 · Memory & Runtime

**Status: in depth.** Five chapters covering the runtime's memory
model from the engineer's seat: where values live, why the compiler
puts them there, what the collector does about it, how the
scheduler makes concurrency cheap, and the five ways Go programs
leak. The runtime's *architecture* (what the pieces are) lives in
[23 §3](../23-go-internals/03-runtime-architecture.md); the
*applied optimization* workflow lives in [19
§2](../19-performance/02-memory-and-allocations.md); this section
is the connective depth between them.

## Chapters

1. **[Stack & heap](01-stack-and-heap.md)**: growable goroutine
   stacks, the allocator's size classes, pointer vs value semantics
   as the memory decision
2. **[Escape analysis](02-escape-analysis.md)**: the rules that
   force escapes, reading `-m` diagnostics, boxing costs, and the
   API-shape insight (`leaking param` is your callers' problem)
3. **[The garbage collector](03-garbage-collector.md)**: tri-color
   concurrent mark-sweep, write barriers, pacing, the assist tax,
   GOGC/GOMEMLIMIT mechanics, and the gctrace reading guide
4. **[The scheduler](04-scheduler-internals.md)**: work stealing,
   async preemption, syscall handoff, the netpoller, schedtrace
   decoding, and the syscall-thread math behind worker pools
5. **[Memory leaks in Go](05-memory-leaks.md)**: the four-shape
   taxonomy (goroutines, pinned arrays, unbounded maps, un-released
   resources), each with an ownership rule and a detection metric

## The example

[examples/memwatch](examples/memwatch/) is the leak taxonomy as a
specimen jar: `Tracker` (owned vs unowned goroutines, with
`LiveWorkers` as the self-reporting metric), `MapGrowth`
(bounded vs unbounded collections), and `Big`/`BigFix` (the pinned
backing array and the copy fix), with structural and lifecycle
tests asserting each lesson.

```bash
go test ./09-memory-runtime/... -race -v
```

## The one-page summary

| Question | Answer | Chapter |
|---|---|---|
| Where does my value live? | Stack if the compiler can prove frame-locality | 1, 2 |
| What does allocation cost? | Per-P cache → central → page heap → GC work | 1 |
| When does the GC run and what does it cost? | Pacing by GOGC/GOMEMLIMIT; the cost is assist CPU, not pauses | 3 |
| Why is goroutine-per-connection viable? | Netpoller + parking: blocked Gs cost memory, not threads | 4 |
| Why is memory growing? | Something reachable that should not be: four shapes | 5 |
