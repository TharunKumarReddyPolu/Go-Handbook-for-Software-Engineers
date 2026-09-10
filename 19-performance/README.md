# 19 · Performance

Optimization is a measurement discipline, not an intuition contest. This
section's iron rule: **measure first, change one thing, measure again.**
Every claim here is backed by a runnable benchmark in `examples/`, and
the before/after numbers in the chapters were produced by them.

## Objectives

By the end of this section you can:

- Build benchmarks that survive scrutiny (benchstat, sinks, b.Loop)
- Read CPU, heap, and goroutine pprof profiles and act on them
- Find and fix the usual culprits: allocations, lock contention, chatty I/O
- Use escape analysis output to understand where your memory goes
- Batch, pool, and cache with evidence, and know when NOT to
- Configure GC (GOGC/GOMEMLIMIT) and PGO deliberately

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Measure first](01-measure-first.md) | Benchmarks, pprof, trace, the methodology |
| 2 | [Memory & allocations](02-memory-and-allocations.md) | Escape analysis, pooling, GC pressure |
| 3 | [Concurrency performance](03-concurrency-performance.md) | Contention, channel costs, batching |
| 4 | [Compiler & PGO](04-compiler-and-pgo.md) | What the compiler does for you, PGO workflow |

## Examples

`examples/` carries the worked measurements:

- `examples/join/`: string building: O(n²) vs O(n), with the actual
  output from this repo's CI-capable machine
- `examples/profiling/`: a deliberately slow service handler with the
  pprof workflow documented in code comments

Run the benchmarks:

```bash
go test ./19-performance/... -bench=. -benchmem
```

## Progress checklist

- [x] Benchmarking methodology
- [x] CPU / memory / allocation profiling with pprof
- [x] Execution tracing
- [x] Escape analysis with real examples
- [x] GC behavior and tuning (GOGC, GOMEMLIMIT)
- [x] Concurrency performance: lock contention, channel costs
- [x] Batching and pooling
- [x] Zero-copy techniques
- [x] PGO
- [x] When optimization is premature
