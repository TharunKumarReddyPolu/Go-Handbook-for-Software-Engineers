# Coming from other languages

## Why Does This Matter?

Most engineers meet Go while fluent elsewhere. The fastest route to
idiomatic Go is not learning new syntax — it is noticing which habits
transfer, which must be unlearned, and why Go made the opposite choice.
This chapter is the map.

## Java → Go

| Java habit | Go replacement | Why |
|---|---|---|
| Classes + inheritance | structs + interfaces + embedding | Inheritance couples hierarchies; Go composes |
| `implements` clauses | implicit interface satisfaction | Decouples consumers from providers |
| Exceptions (checked/unchecked) | error values in signatures | Failures are part of the API contract |
| Constructors + factories | plain funcs (`NewX`) + zero values | Zero value is usable; less ceremony |
| Getters/setters everywhere | direct fields, or methods when needed | JavaBean ceremony buys nothing here |
| `public/private` keywords | name casing (`Foo`/`foo`) | Visibility at a glance, per file |
| Generics on classes | generics on funcs/types (Go 1.18+) | Similar power, simpler constraint model |
| Streams API | `for` + `slices`/`maps` stdlib packages | Explicit loops; allocation-light |
| Spring DI containers | manual wiring in main, functional options | Explicit beats reflective magic |
| mvn/gradle | `go mod` | One toolchain, no build dialects |

The deepest mindset shift: **no exceptions**. In Java, a method's
signature lies about its failure modes. In Go, `error` is a return value;
*every* call site decides. Expect code to look more verbose and be far
more predictable. Related: `null` practically disappears — pointers can be
nil, but the zero values and `ok` idioms push nil checks to boundaries.

## Python → Go

| Python habit | Go replacement | Why |
|---|---|---|
| Duck typing | interfaces (implicit, structural) | Same philosophy, checked at compile time |
| List/dict comprehensions | `for` loops + `slices`/`maps` pkgs | No syntax for it; perf is predictable |
| GIL-thwarting via multiprocessing | goroutines, no GIL | Real parallelism, cheap |
| Decorators | higher-order funcs, middleware patterns | Explicit wrapping; no @ syntax |
| Dynamic dicts for everything | named struct types | Types document and check themselves |
| `pip` + venv | modules (`go mod`) | No environment activation dance |
| REPL-driven | tests + `go run` scratch files | Compile-fast enough to explore |
| Exceptions | error values | Same as Java, doubly strange coming from try/except |

Python's "we'll check at runtime" becomes Go's "the compiler checks it."
The friction you feel in week one (declaring types, handling every error)
is the static typing paying off in month three (refactoring without fear).

## C++ → Go

| C++ habit | Go replacement | Why |
|---|---|---|
| Manual memory management | GC (concurrent, low-pause) | You keep escape analysis knowledge, lose the bugs |
| Templates | generics (much tamer) | Constraints, no metaprogramming games |
| RAII | defer | Resource lifetime tied to scope, simplified |
| Operator overloading | none | Code reads as written |
| Multiple inheritance | embedding (no virtual dispatch) | Composition without the diamond |
| Move semantics | value semantics + GC | Copying is safe and often fast |
| Undefined behavior on races | data race = undefined, but `-race` detects it | Same danger, real tooling |
| Header/impl split | none — single file per package unit | Builds are fast; duplication is fine |

C++ engineers over-apply escape-analysis intuitions ("pass by pointer to
avoid the copy!") — Go's compiler often allocates less with clean value
code than with defensive pointers. Measure before copying your old
instincts (see [19-performance](../19-performance/)).

## JavaScript/TypeScript → Go

| JS/TS habit | Go replacement | Why |
|---|---|---|
| async/await | goroutines + channels | Concurrency is not syntax sugar; it's a runtime feature |
| Promise chains | sequential code + goroutines | Blocking style, no callback coloring |
| npm + node_modules | modules (no node_modules analog in your repo) | `go.sum` instead of lockfile drama |
| undefined vs null | zero values + `ok` pattern | One answer, not two |
| Spread/rest, optional chaining | explicit checks, variadics | More lines, fewer surprises |
| Dynamic objects with extra fields | structs + `map[string]any` at boundaries | Types where you can, maps where you must |
| tsconfig/tslint churn | gofmt/vet/staticcheck, config-free | One way |

The "function coloring" problem (async infects callers) is the sharpest
contrast: in Go, *every* function is synchronous-looking; blocking calls
just work because goroutines are cheap. Concurrency is explicit at the
call site (`go f()`), not in the signature.

## What transfers beautifully

- **Systems thinking**: ownership, lifetimes, invariants — all transfer.
- **Testing discipline**: table-driven tests will feel familiar if you
  used pytest/Jest parameterization.
- **API design**: small surfaces, clear contracts — same goals.
- **Debugging**: your process (hypothesize, bisect, verify) transfers;
  the tools change ([11-tooling](../11-tooling/)).

## What to unlearn deliberately

1. **Reaching for class hierarchies.** When you feel "I need a base
   class," the Go answer is usually an interface at the *consumer* + a
   struct with embedded shared behavior, or plain functions.
2. **Defensive null checks everywhere.** Zero values and boundary
   validation replace it; internal invariants are documented, not
   re-checked at every line.
3. **Premature abstraction.** Go's cultural norm: write it twice before
   extracting; an interface needs a second implementation to exist.
4. **Framework-first thinking.** A backend in Go is stdlib + a router at
   most. Reach for [12-http-networking](../12-http-networking/) before
   reaching for a web framework.

## Interview Questions

1. *"You're a Python team lead; your team moves to Go. What are the first
   three friction points?"* — Error handling verbosity, type declarations,
   missing comprehensions; pair each with the payoff that follows.
2. *"How do you handle 'checked exceptions' style flows without
   exceptions?"* — errors as values, wrapping with `%w`, sentinel/domain
   errors ([05-errors](../05-errors/)).
3. *"A Java engineer writes a 5-level type hierarchy for a plugin system.
   What do you suggest?"* — Small interface at the consumer, registered
   implementations, embedding for shared code; show the refactor.

## Practice Exercises

1. Port a 50-line script from your old language to Go. Note every place
   you reached for an old idiom that does not exist, and find the Go way.
2. Write the same "fetch three URLs concurrently, return first success"
   in your old language and in Go with goroutines; compare the code's
   shape and failure modes.
3. Pair review: give your old-language version of a small service to a Go
   engineer and ask "what looks non-Go?" Write the list; it doubles as
   your unlearning checklist.

## Further Reading

- [Effective Go](https://go.dev/doc/effective_go) — read once early, again after a month of Go
- [Go for JavaScript Developers](https://github.com/breo/go-for-javascript-developers) — community reference comparison
- [Go wiki: ComeFrom sections](https://go.dev/wiki) — community notes on migrating from specific languages
