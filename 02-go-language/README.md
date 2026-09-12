# 02 · The Go Language

**Status: complete.** Eight chapters with a compiled, tested example
package. Every chapter follows the standard contract (see
[CONTRIBUTING](../CONTRIBUTING.md)).

The core data model of Go: how values, headers, and pointers really
behave, and what that means for correctness, performance, and
concurrency. This is the section to read slowly; nearly every later
chapter assumes its mental models.

## Chapters

1. **[Arrays & slices](01-arrays-and-slices.md)**: header semantics,
   growth, the backing-array sharing model, aliasing and the leak
2. **[Maps](02-maps.md)**: invariants, iteration randomness,
   addressability, nil maps, the sync.Map pointer
3. **[Strings, runes & bytes](03-strings-runes-bytes.md)**: UTF-8
   reality, indexing vs ranging, the string vs []byte table
4. **[Structs](04-structs.md)**: zero-value design, tags,
   comparability, embedding
5. **[Pointers & receivers](05-pointers-and-receivers.md)**: value vs
   pointer semantics, method sets, escape analysis primer
6. **[Interfaces & embedding](06-interfaces-and-embedding.md)**: the
   two-word model, implicit satisfaction, the nil trap, decorators
7. **[Function types, closures & variadics](07-closures-functions.md)**:
   capture semantics, middleware, functional options
8. **[Multiple & named returns](08-multiple-returns.md)**: tuple
   unpacking, naked returns, the shadowing trap, defer-repair

## Runnable example

[examples/seqops](examples/seqops/): generic chunking (aliasing and
copy variants, each with a test asserting the sharing contract),
in-place vs allocating filters, and rune-safe truncation, including
the test that proves rune-level truncation splits ZWJ emoji families.

```bash
go test ./02-go-language/... -v
```

Cross-references: slice memory behavior pairs with
[09-memory-runtime](../09-memory-runtime/README.md); interface design
philosophy (consumer-side, small interfaces, DI) is in
[04-functions-methods-interfaces](../04-functions-methods-interfaces/README.md).
