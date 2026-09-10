# 23 · Go Internals

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The goal: engineers who understand *why
Go behaves the way it does*: not compiler engineers. The runtime
material already drafted across [09](../09-memory-runtime/) (memory,
GC, scheduler) and [19 §4](../19-performance/04-compiler-and-pgo.md)
(compiler) lands here in full depth.

## Planned chapters

1. **Compiler pipeline**: source → AST → SSA → machine code; where
   each optimization lives
2. **Lexer, parser & AST**: reading Go's own AST with go/ast (the
   tooling-grade introduction)
3. **SSA & optimizations**: inlining, BCE, escape analysis as passes
   with observable flags (extends
   [19 §4](../19-performance/04-compiler-and-pgo.md))
4. **Runtime architecture**: the runtime as a program: memory
   allocator, GC, scheduler, netpoller
5. **Goroutines & the scheduler**: G/M/P internals, preemption,
   stack management (extends
   [08 §6](../08-concurrency/06-concurrency-vs-parallelism.md))
6. **Channels**: hchan internals: lock, buffer, sendq/recvq, the
   direct-handoff fast path
7. **Maps**: buckets, overflow chains, growth/evacuation, why order
   is randomized
8. **Interfaces**: itab layout, dynamic dispatch, the nil traps'
   mechanical explanation
9. **Slices & strings**: header layout, growth strategy, immutability
10. **The memory model**: happens-before, the sync primitives' guarantees
    ([08 §8](../08-concurrency/08-faq-notes.md) pairs)
11. **Reflection**: reflect.Type/Value mechanics, costs, when to
    refuse it
12. **Assembly**: reading GOASM output for hot functions; what the
    compiler actually emitted
