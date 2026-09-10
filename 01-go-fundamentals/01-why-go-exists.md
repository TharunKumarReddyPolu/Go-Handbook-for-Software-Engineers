# Why Go exists

## Why Does This Matter?

You will spend years with this language. Knowing what it optimizes for,
and what it deliberately gives up: explains almost every design decision
you will otherwise find odd: no inheritance, no exceptions, unused
variables are compile errors, no REPL-driven culture. Go is not a pile of
features; it is a set of trade-offs aimed at a specific kind of work.

## Mental Model

Go was designed at Google around 2007 by engineers maintaining very large
C++ and Java systems. Their pain was not "we lack features." It was:

- builds that took tens of minutes;
- onboarding engineers onto codebases whose abstractions required an
  interpreter to explain;
- languages with one concurrency model (threads) whose costs forced
  event-loop workarounds;
- deployments that needed a carefully staged runtime.

Go's answer to each: fast compiles, one obvious way to write most things,
goroutines and channels in the language, static single binaries. When you
evaluate a Go feature, ask what problem from that list it solves, and what
it gave up to solve it. The features that seem missing are usually the
price paid for the ones that are there.

## What Go optimizes for

| Goal | Mechanism | What it costs you |
|---|---|---|
| Fast builds | Strict import graph, no cyclic deps, unused import/variable = error | No "just quickly" sloppy imports |
| Readable at scale | One formatting (gofmt), no operator overloading, minimal keywords | Less expressiveness for clever code |
| Easy concurrency | Goroutines + channels in the runtime | You must still design for races |
| Simple deployment | Static binaries, cross-compilation | Larger binaries than a JVM shared runtime model |
| Low friction onboarding | Small spec (~a tram read), few ways to do things | Fewer tools for DSL-like domains |

## The philosophy in practice

Three sentences you will see reflected in every chapter of this handbook:

1. **Clear is better than clever.** If a reviewer needs to think about your
   type gymnastics, the code lost.
2. **The zero value is useful.** Types should be usable without a
   constructor ceremony (see the [zero values chapter](05-zero-values.md)).
3. **Errors are values.** Failures are part of every function's signature,
   not a side channel (see [05-errors](../05-errors/)).

None of these are aesthetic preferences. Each exists because a team of
thousands writing services for a decade needs code that is *predictable*
more than it needs to be *concise*.

## Where Go is the wrong choice

Be honest about this in design docs and interviews:

- **GUIs and games**: ecosystems live elsewhere.
- **Heavy numeric computing**: no SIMD ergonomics, no BLAS culture;
  Python + native libs or Rust win.
- **Rich domain modeling with deep hierarchies**: you *can* do DDD in Go,
  but modeling tools that rely on inheritance or macros will fight you.
- **Sub-microsecond latency budgets**: possible in Go, but C++/Rust give
  more control (this handbook still covers real-time-adjacent Go in
  [19-performance](../19-performance/) and fintech use in
  [25-fintech-with-go](../25-fintech-with-go/)).

## Interview Questions

1. *Why did Google build Go instead of improving C++?*: Answer in terms of
   build times, onboarding, and dependency hygiene, not syntax taste.
2. *What does Go deliberately lack, and why is that a feature?*: Inheritance,
   exceptions, ternary, operator overloading; each answer should cite the
   maintainability cost it avoids.
3. *Name two runtime features Go gives you that C++ does not.*: Goroutines
   with a work-stealing scheduler, and a garbage collector with sub-ms
   pauses.

## Practice Exercises

1. Write a paragraph (no code) explaining why unused imports being a
   compile error *accelerates* large teams.
2. List the last three services you worked on. For each: would Go's
   trade-offs have helped or hurt? Be specific.

## Further Reading

- [Go at Google: Language Design in the Service of Software Engineering](https://go.dev/talks/2012/splash.article): Rob Pike's own account
- [Go FAQ: Design](https://go.dev/doc/faq#design)
- [The Go Memory Model](https://go.dev/ref/mem): for later, after section 08
