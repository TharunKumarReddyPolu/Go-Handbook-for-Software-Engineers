# 07 · Generics

**Status: complete.** Six chapters with the runnable `genlib` example
package. Every chapter follows the standard contract (see
[CONTRIBUTING](../CONTRIBUTING.md)). The baseline is Go 1.27
(generic methods); version-stamped claims follow
[meta/versioning.md](../meta/versioning.md).

## The standing rule

Generics are introduced *after* concrete and interface-based
solutions in every example, because the correct engineering question
is never "can I use generics?" but "did the duplication earn them?"

## Chapters

1. **[Why generics exist](01-why-generics-exist.md)**: the four
   interface{}-era costs (erasure, duplication, generation, boxing)
   and what type parameters fixed
2. **[Type parameters & constraints](02-type-parameters-and-constraints.md)**:
   the built-in constraints, unions with `~`, the operator problem,
   inference and anchoring
3. **[Generic data structures](03-generic-data-structures.md)**:
   containers that earn brackets, GC shape stenciling, when the
   built-ins still win
4. **[Generic APIs](04-generic-apis.md)**: the slices/maps style
   dissected, the narrowest-constraint rule, the Retry case study
5. **[When generics hurt](05-when-generics-hurt.md)**: the two gates,
   the anti-pattern gallery, the readability tiebreakers
6. **[Version notes](06-version-notes.md)**: the 1.18 to 1.27
   timeline, generic methods precisely, dated-tell spotting

## Runnable example

[examples/genlib](examples/genlib/): constraint styles with
instantiation-table tests (int/string/named ~types), the
zero-value-first Stack, the zero-value-usable Cache, and the Retry
helper with its two distinct failure exits (context dead vs backoff
exhausted), a distinction its own tests forced.

```bash
go test ./07-generics/... -v
```
