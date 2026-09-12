# Version notes

## Why Does This Matter?

Generics are the fastest-evolving corner of Go: introduced in 1.18,
refined in nearly every release since, and structurally extended in
1.27. Code that compiles on one toolchain silently fails on older
ones, and advice written for 1.18 (much of the internet's generics
content) is now outdated in specific, checkable ways. This chapter is
the timeline and the compatibility rules, in the handbook's
versioning convention ([meta/versioning.md](../meta/versioning.md)).

## The timeline

| Version | Change | Practical effect |
|---|---|---|
| **1.18** (Mar 2022) | Type parameters, constraints, `any` alias | The feature ships; `comparable`, unions, inference v1 |
| **1.19** | Minor constraint fixes; `atomic` types (not generics, but same era) | Smoother constraints |
| **1.20** | Comparable satisfied by ordinary interface types | Some previously-invalid constraint code compiles |
| **1.21** | `slices`, `maps`, `cmp` packages; `min`/`max`/`clear` builtins | The stdlib generics era begins; stop hand-rolling |
| **1.22** | `slices.Delete` zeroes removed elements; loop variables per-iteration | Closures/captures pre-1.22 differ (see below) |
| **1.23** | `iter.Seq`/`iter.Seq2`; range-over-func; `maps.Keys/Values` return iterators | Iterator-style generic APIs standardize |
| **1.25** | `sync.WaitGroup.Go` helper | One-liner goroutine+WaitGroup (concurrency, but changes generic pool examples) |
| **1.27** | **Generic methods**: methods may declare type parameters | The last structural gap closes; interface methods still may not |

## Go 1.27: generic methods, precisely

Before 1.27, a type could be generic but its methods could not add
type parameters. The workarounds were famous (the "view type" dance):

```go
// Pre-1.27 workaround: free function or wrapper type.
func Map[K comparable, V, R any](m map[K]V, f func(K, V) R) map[K]R

// Go 1.27: methods may declare their own type parameters.
func (c *Cache[K]) Map[V2 any](f func(K, V) V2) map[K]V2 {
    out := make(map[K]V2, len(c.items))
    for k, v := range c.items {
        out[k] = f(k, v)
    }
    return out
}
```

The rules that remain after 1.27:

- **Methods declare their own type parameters** in addition to the
  receiver type's: `func (c *Cache[K]) Map[V2 any](...)`. The
  receiver's `K` is in scope; the method adds `V2`.
- **Interface methods still cannot be generic**: an interface method
  with type parameters is not legal (runtime dispatch of arbitrary
  instantiations is the stated reason). Behavior polymorphism stays
  with concrete-type methods or free functions.
- **No virtual dispatch through generic methods**: they compile to
  direct calls; you cannot put them behind an interface the way a
  non-generic method works. If you need interface dispatch, the
  method cannot be generic.

The migration order for pre-1.27 workaround code: free functions
become methods (mechanical), wrapper "view types" dissolve, and
interface-shaped workarounds stay (they were never wrong, just
necessary).

## Loop variables and 1.22: the generics-adjacent trap

Not generics, but every generic-container example written before 1.22
(and much copied since) carries the closure-capture bug or its
fix-shim:

```go
// Pre-1.22: one loop variable; all closures see the last value.
for _, job := range jobs {
    go func() { process(job) }()      // BUG pre-1.22
}
// Go 1.22+: job is per-iteration; the code above is correct.
```

Version-stamped generic example code often shows the
`job := job` shadow-fix "for safety"; on 1.22+ it is noise (harmless,
but noise). Know which world a snippet came from before copying it
into production: this is exactly the class of version-specific claim
the handbook's convention flags
([meta/versioning.md](../meta/versioning.md)).

## Reading old generics content: the dated tells

The internet's 2022-era generics advice has specific, recognizable
staleness:

| Dated advice | Current truth |
|---|---|
| "Write your own Map/Filter/Reduce" | `slices`/`maps` + iterators cover most of it (1.21/1.23) |
| "Constraints can't express X, use reflection" | Often expressible now; reflection stays last resort |
| "No generic methods, use view types" | 1.27 closed it (methods yes, interface methods no) |
| "comparable is very limited" | 1.20 relaxed it; check what your case actually needs |
| "Generics are slow because dictionaries" | Shape stenciling is nanoseconds; measure (03 of this section) |

The habit that ages well: **stamp claims with versions** and prefer
sources that do; the handbook's baseline is Go 1.27 and older-claim
audits are part of its CI discipline.

## Toolchain policy for sections using generics

- The go directive floor for this handbook's examples is `go 1.27`
  ([go.mod](../go.mod)): iterators, generic methods, and the
  zeroing-Delete semantics are load-bearing.
- Readers on older toolchains: the *concepts* hold from 1.21
  onward; 1.27-specific features (generic methods) will not compile
  and are marked inline where used.
- CI pins the toolchain ([06-packages-modules/05](../06-packages-modules/05-reproducible-builds.md)):
  examples do not silently start or stop compiling with ambient Go
  versions.

## Common Mistakes

- **Copying pre-1.21 generics tutorials** and rebuilding stdlib
  functions that now exist: check `slices`/`maps` before writing any
  generic helper ([04](04-generic-apis.md)).
- **Expecting generic methods behind interfaces**: 1.27 gives methods
  type parameters, not virtual dispatch through them; the interface
  boundary remains non-generic by design.
- **Assuming all colleagues share the toolchain**: a mixed-version
  team compiles different subsets of the same repository; the go
  directive plus a pinned CI toolchain is the shared truth
  ([06-packages-modules/02](../06-packages-modules/02-modules-in-production.md)).

## Idiomatic Go

The versioning habit that keeps this section honest: **every
generics-specific claim in this handbook is stamped** ("introduced in
Go X", "since Go Y"), and examples that need new toolchains say so
inline. The same discipline for your own codebases: a `// Go 1.23:`
comment on an iterator API costs nothing and saves the next reader an
archaeology session.

## Interview Questions

1. *When did generics land, and what were the headline packages that
   followed?*: 1.18; slices/maps/cmp plus min/max/clear in 1.21;
   iterators 1.23; generic methods 1.27.
2. *What did Go 1.27 add, and what is still off the table?*: Generic
  methods on concrete types; interface methods remain non-generic
  (no virtual dispatch of instantiations).
3. *Why do interface methods stay non-generic?*: Interface dispatch
  requires a fixed method signature per runtime itab; arbitrary
  instantiations would break the two-word model.
4. *How do you date-check a generics snippet from a blog post?*: Look
  for the tells (hand-rolled slices helpers, view-type workarounds,
  pre-1.22 closure shims) and re-verify against the current release
  notes.
5. *Your module's go directive is 1.21 and a teammate uses generic
  methods: what happens?*: Compilation fails on their files (the
  language floor rejects the syntax); fix by raising the directive
  deliberately ([06-packages-modules/06](../06-packages-modules/06-semantic-versioning.md)).

## Practice Exercises

1. Write the same container transformation three ways: pre-1.18
   (interface{}), 1.18-era (free functions), 1.27 (generic methods);
   note where each reads best.
2. Take a 2022-era generics blog post; list its dated tells; verify
   each against current release notes.
3. Add a version-stamp comment discipline to a real codebase: every
   1.22+-dependent closure and 1.27 method marked; review the diff's
   information value.

## Further Reading

- [Go 1.18 release notes: generics](https://go.dev/doc/go1.18#generics)
- [Go 1.23 release notes: iterators](https://go.dev/blog/range-functions)
- [meta/versioning.md](../meta/versioning.md) (this handbook's
  versioning convention)
