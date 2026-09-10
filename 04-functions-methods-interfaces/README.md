# 04 · Functions, Methods & Interfaces

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Interview-level summaries already exist
in [26 §2](../26-go-interview-preparation/02-intermediate.md).

## Planned chapters

1. **Functions** — signatures as contracts, first-class function types,
   closures and capture semantics
2. **Methods & method sets** — value vs pointer receivers (decision
   rules), promotion through embedding
3. **Interfaces** — implicit satisfaction, the (type, value) internals,
   small-interface philosophy, composition (`io.ReaderWriter` shapes)
4. **BAD → BETTER → IDIOMATIC design clinic** — the same feature built
   three ways: class-hierarchy Go, pragmatic Go, idiomatic Go
5. **Dependency inversion in Go** — consumer-side interfaces vs
   Java-style DI containers; functional options
6. **Constructors & invariants** — `NewX` conventions, zero-value-first
   design from [05-zero-values](../01-go-fundamentals/05-zero-values.md)
7. **Composition over inheritance** — why Go dropped class hierarchies,
   and the embedding patterns that replace them

## Standing cross-references

- The errors section demonstrates interface-first design against
  transport layers: [05 §2](../05-errors/02-error-design.md)
- The Kafka client seam is dependency inversion applied:
  [18 §2](../18-kafka-with-go/02-go-clients.md)
