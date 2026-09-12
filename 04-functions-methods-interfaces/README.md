# 04 · Functions, Methods & Interfaces

**Status: complete.** Seven chapters plus the runnable `di` example
package. Every chapter follows the standard contract (see
[CONTRIBUTING](../CONTRIBUTING.md)).

Go's approach to abstraction: signatures as contracts, methods with
honest receiver rules, tiny consumer-side interfaces, and composition
replacing inheritance. The section's centerpiece is the design clinic
(chapter 4), which builds the same feature three ways from
class-habit Go to idiomatic Go.

## Chapters

1. **[Functions: signatures as contracts](01-functions-as-contracts.md)**:
   what a signature promises, the three dependency styles, closures at
   seams
2. **[Methods & method sets](02-methods-and-method-sets.md)**: the
   per-type receiver rule, promotion, method values vs expressions
3. **[Interfaces: the philosophy](03-interfaces-philosophy.md)**:
   consumer-side definition, small interfaces, when generics replace
   them
4. **[Design clinic](04-design-clinic.md)**: the same feature built
   BAD, BETTER, and IDIOMATIC, with the generalizable checklist
5. **[Dependency inversion](05-dependency-inversion.md)**: constructor
   injection, functional options, wiring packages, no DI container
6. **[Constructors & invariants](06-constructors-and-invariants.md)**:
   zero-value-first design, the NewX gate, Parse/New/Must
7. **[Composition over inheritance](07-composition-over-inheritance.md)**:
   why Go dropped class hierarchies, the four replacement patterns

## Runnable example

[examples/di](examples/di/): the clinic's UserService with inline fakes
(needing no mock library), a validation-table domain type, injected
clock, and tests proving storage failures propagate while notification
failures stay best-effort.

```bash
go test ./04-functions-methods-interfaces/... -v
```

Standing cross-references: the errors section demonstrates
interface-first design against transport layers
([05-errors/02](../05-errors/02-error-design.md)); the Kafka client
seam is dependency inversion applied
([18/02](../18-kafka-with-go/02-go-clients.md)).
