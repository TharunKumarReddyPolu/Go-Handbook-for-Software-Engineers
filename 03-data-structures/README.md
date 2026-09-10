# 03 · Data Structures

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).**

## Planned chapters

Each chapter covers: implementation, complexity, memory characteristics,
idiomatic Go implementation, when to use, when NOT to use, and a
production example.

1. **Arrays, slices & maps in depth** — the built-ins' real costs
2. **Sets via maps** — `map[T]struct{}` idiom and why
3. **Stacks & queues** — slice-backed, slice-as-stack idioms
4. **Heaps & priority queues** — `container/heap`, real scheduling uses
5. **Linked lists** — `container/list` and why raw slices usually win
6. **Trees** — binary trees, tries; when Go's map + sort covers you
7. **Graphs** — adjacency maps, BFS/DFS in idiomatic Go
8. **Ring buffers** — fixed-size circular queues for streams
9. **LRU cache** — the classic; bounded-memory pattern reused by the
   risk-engine windows in [25 §4](../25-fintech-with-go/04-risk-and-compliance.md)

## Cross-references

- Concurrency-safe variants land here with links to the sync chapter:
  [08 §4](../08-concurrency/04-sync-primitives.md)
- The LRU chapter pairs with the singleflight/caching discussion in
  [19 §3](../19-performance/03-concurrency-performance.md)
