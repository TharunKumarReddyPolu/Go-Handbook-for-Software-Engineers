# 02 · The Go Language

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The topic map below is complete and
sequenced; the flagship chapters cross-reference it where they already
cover material.

## Planned chapters

1. **Arrays & slices**: header semantics, growth, the backing-array
   sharing model; full treatment previewed in
   [01-fundamentals ch. 4](../01-go-fundamentals/04-variables-types-constants.md)
2. **Maps**: invariants, iteration, the two niche sync.Map shapes
3. **Strings, runes & bytes**: UTF-8 reality, conversion costs
4. **Structs**: tags, comparison, zero-value design
5. **Pointers**: value vs pointer semantics; decision rules in
   [FAQ](../FAQ.md)
6. **Methods & method sets**: receiver rules, interface satisfaction
7. **Interfaces & embedding**: internals of the (type, value) pair;
   the nil traps from
   [08-faq-notes](../08-concurrency/08-faq-notes.md)
8. **Type assertions & switches**: safe extraction patterns
9. **Function types, closures & variadics**: capture semantics
10. **Multiple & named returns**: the shadowing traps

## Coverage notes

Every chapter answers the standard contract (see
[CONTRIBUTING](../CONTRIBUTING.md)): why it exists, how it works,
common mistakes, performance/concurrency/security notes, testing,
interview questions, exercises, further reading. Memory/reference
semantics chapters include diagrams; the string vs []byte and
value-receiver vs pointer-receiver comparisons land as tables.
