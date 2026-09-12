# 03 · Data Structures

**Status: complete.** Nine chapters with a benchmark-backed example
suite. Every chapter follows the standard contract (see
[CONTRIBUTING](../CONTRIBUTING.md)): implementation, complexity,
memory behavior, when NOT to use it, and a production home.

The lens throughout: Go's built-ins (slice, map, array) are so good
that every custom structure must justify itself against them, and
complexity classes are the floor while cache behavior and allocation
rate are the ceiling engineers forget.

## Chapters

1. **[The built-ins in depth](01-builtins-in-depth.md)**: the measured
   cost table, scan-vs-map crossover, slices as stacks and queues,
   memory overhead ratios
2. **[Sets via maps](02-sets.md)**: the `map[T]struct{}` idiom, generic
   Set with algebra, ordered sets, bitsets
3. **[Stacks & queues](03-stacks-queues.md)**: LIFO/FIFO disciplines,
   the reslice-queue leak and its fixes, channels as queues
4. **[Heaps & priority queues](04-heaps-priority-queues.md)**:
   `container/heap` and a generic heap, the Idx bookkeeping that
   enables Fix/Remove, starvation policy
5. **[Linked lists](05-linked-lists.md)**: `container/list` honestly,
   when pointer-chasing wins (LRU), when slices win (almost always)
6. **[Trees](06-trees.md)**: BSTs and tries, map+sort as the
   production alternative, iterative traversal on untrusted input
7. **[Graphs](07-graphs.md)**: adjacency maps, BFS/DFS, cycle
   detection, topological order, Dijkstra with lazy deletion
8. **[Ring buffers](08-ring-buffers.md)**: fixed memory, wrap seams,
   drop-oldest vs block policies, the record-vs-query API trap
9. **[LRU cache](09-lru-cache.md)**: map + list, TTL and byte-budget
   decisions, the sharding ladder, singleflight pairing

## Runnable examples

- [examples/dsbench](examples/dsbench/): the benchmarks behind every
  cost table in this section (scan-vs-map crossover, append hints,
  queue shapes, list-vs-slice locality). Run:

  ```bash
  go test ./03-data-structures/examples/dsbench -bench . -benchmem
  ```

- [examples/ring](examples/ring/): fixed-capacity FIFO with
  drop-oldest eviction, wrap-seam tests, and the RecentKeys dedup
  shape (with the Seen/Contains record-vs-query split its own tests
  forced).
- [examples/lru](examples/lru/): the generic LRU with promotion,
  non-mutating Peek, eviction callbacks, metrics, a differential test
  against a reference implementation, and a concurrent storm test.

Cross-references: concurrency-safe variants are covered in
[08-concurrency/04-sync-primitives](../08-concurrency/04-sync-primitives.md);
the LRU pairs with the caching discussion in
[19-performance/03-concurrency-performance](../19-performance/03-concurrency-performance.md).
