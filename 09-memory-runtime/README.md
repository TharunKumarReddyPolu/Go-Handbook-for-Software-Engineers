# 09 · Memory & Runtime

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The engineering-facing half already
exists in [19 §2 Memory & allocations](../19-performance/02-memory-and-allocations.md);
this section adds the runtime depth beneath it.

## Planned chapters

1. **Stack & heap** — goroutine stacks (growable, 2-8KB), allocation
   paths, the runtime allocator's size classes
2. **Escape analysis** — the compiler's rules with worked `-gcflags=-m`
   examples (extends
   [19 §2](../19-performance/02-memory-and-allocations.md))
3. **Garbage collector** — tri-color concurrent mark & sweep, write
   barriers, GC pacing, GOGC/GOMEMLIMIT mechanics
4. **Pointer vs value semantics** — how the choice shapes escapes,
   caches, and API design
5. **Interfaces & allocations** — boxing costs, the devirtualization
   story from [19 §4](../19-performance/04-compiler-and-pgo.md)
6. **The scheduler** — G/M/P in depth, work stealing, preemption,
   syscalls; extends
   [08 §6](../08-concurrency/06-concurrency-vs-parallelism.md)
7. **runtime package** — GOMAXPROCS, metrics, MemStats, the knobs that
   exist and the ones that don't
8. **Memory leaks in Go** — the taxonomy: goroutines, unbounded
   collections, pinned references, runtime-level leaks
