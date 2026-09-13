# 23 · Go Internals

**Status: in depth.** Seven chapters explaining why Go behaves the
way it does: the compiler's decisions, the runtime's architecture,
and the data-structure internals behind daily surprises. The goal
is not compiler engineers: it is engineers who can read a
diagnostic flag, explain a panic, and predict what the machine
will do with clear code.

## Chapters

1. **[The compiler pipeline](01-compiler-pipeline.md)**: source →
   AST → SSA → machine code, and where each optimization lives
2. **[SSA & optimizations](02-ssa-and-optimizations.md)**: escape
   analysis, inlining, and bounds-check elimination with real
   diagnostic transcripts from the example package
3. **[Runtime architecture](03-runtime-architecture.md)**: the
   runtime as a program: G/M/P, the netpoller, allocator, GC,
   growable stacks
4. **[Channels & maps under the hood](04-channels-and-maps.md)**:
   hchan fast paths and direct handoff; bucket layout, incremental
   growth, randomized iteration
5. **[Interfaces, slices & strings](05-interfaces-slices-strings.md)**:
   the two-word interface (typed nil, mechanically), the slice
   header (aliasing by construction), string immutability
6. **[The memory model](06-memory-model.md)**: happens-before
   edges, the complete practical table, and what -race really
   checks
7. **[Reflection & assembly](07-reflection-and-assembly.md)**: the
   cost ladder, the cached-plan pattern, and reading the compiler's
   output

## The example

[examples/internals](examples/internals/) makes internals
testable:

- escape decisions pinned with `testing.AllocsPerRun`
  (`Escaped` allocates; `StackLocal` does not)
- the typed-nil trap and its `Normalize` fix, unit-tested
- `SortedKeys` pinning the map-order discipline over 100 runs

The compiler-flag transcripts quoted in chapters 1, 2, and 7 were
captured against this package: rerun the commands yourself, they
will match.

## Where this sits

- The *engineering* view of memory and GC lives in
  [09](../09-memory-runtime/); this section provides the
  architecture beneath it.
- The *applied* optimization workflow lives in
  [19](../19-performance/); chapters 1-2 here explain the
  mechanism those benchmarks measure.
- The *behavioral* concurrency rules live in
  [08](../08-concurrency/); chapters 3-4-6 here explain why those
  rules hold.
