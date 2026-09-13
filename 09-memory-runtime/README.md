# 09 · Memory & Runtime

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The engineering-facing half already
exists in [19 §2 Memory & allocations](../19-performance/02-memory-and-allocations.md);
this section adds the runtime depth beneath it.

## Planned chapters

1. **Stack & heap**: goroutine stacks (growable, 2-8KB), allocation
   paths, the runtime allocator's size classes; pointer vs value
   semantics as the escape decision
2. **Escape analysis**: the compiler's rules with worked
   `-gcflags=-m` examples, plus interface boxing costs (extends
   [19 §2](../19-performance/02-memory-and-allocations.md))
3. **The garbage collector**: tri-color concurrent mark & sweep,
   write barriers, GC pacing, GOGC/GOMEMLIMIT mechanics
4. **The scheduler**: G/M/P, work stealing, preemption, syscalls,
   netpoller, and the runtime knobs that matter (extends
   [08 §6](../08-concurrency/06-concurrency-vs-parallelism.md))
5. **Memory leaks in Go**: the taxonomy: goroutines, unbounded
   collections, pinned references, runtime-level leaks

(Consolidated from 8 planned topics: semantics folds into the
stack/heap chapter, interfaces/boxing into escape analysis, the
runtime package into the scheduler chapter.)
