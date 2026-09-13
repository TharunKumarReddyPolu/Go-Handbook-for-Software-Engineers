# 23 · Go Internals

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The goal: engineers who understand *why
Go behaves the way it does*: not compiler engineers. The runtime
material already drafted across [09](../09-memory-runtime/) (memory,
GC, scheduler) and [19 §4](../19-performance/04-compiler-and-pgo.md)
(compiler) lands here in full depth.

## Planned chapters

1. **The compiler pipeline**: source → lexer → parser → AST → SSA →
   machine code, with go/ast as the tooling-grade introduction;
   where each optimization lives
2. **SSA & optimizations**: inlining, bounds-check elimination,
   escape analysis as passes with observable flags (extends
   [19 §4](../19-performance/04-compiler-and-pgo.md))
3. **Runtime architecture**: the runtime as a program: memory
   allocator, GC, scheduler, netpoller; G/M/P and preemption at
   internals depth (the engineering view lives in
   [09](../09-memory-runtime/), the concurrency view in
   [08 §6](../08-concurrency/06-concurrency-vs-parallelism.md))
4. **Channels & maps under the hood**: hchan (lock, buffer,
   sendq/recvq, direct handoff) and map internals (buckets,
   overflow, growth, randomized order)
5. **Interfaces, slices & strings under the hood**: itab and dynamic
   dispatch (the nil traps, mechanically), header layouts, growth
   strategy, string immutability
6. **The memory model**: happens-before, the sync primitives'
   guarantees ([08 §8](../08-concurrency/08-faq-notes.md) pairs)
7. **Reflection & assembly**: reflect.Type/Value mechanics, costs,
   when to refuse it; reading GOASM output for hot functions

(Consolidated from 12 planned topics: data-structure internals
grouped into two chapters, reflection and assembly paired.)
