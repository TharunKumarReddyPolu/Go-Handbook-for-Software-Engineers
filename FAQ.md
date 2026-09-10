# FAQ

Short, direct answers. Each question links to the section with the full
treatment.

## Why Go?

Because it optimizes for the engineering you actually do in backend and
infrastructure work: fast builds, one way to write most things, first-class
concurrency, static binaries that deploy without a runtime, and a standard
library that covers HTTP, TLS, and testing. It trades language cleverness
for team velocity and operational predictability. If you write services,
CLIs, or infrastructure, Go's defaults are excellent. If you want expressive
type systems or language-level metaprogramming, look elsewhere.
→ [01-go-fundamentals](01-go-fundamentals/)

## Why does Go have no traditional inheritance?

Because inheritance couples types through implementation reuse, and that
coupling ages badly: deep hierarchies make change expensive. Go separates
interface (what a type can do) from implementation (how it does it), and
uses composition and embedding for reuse. Behavior is defined at use sites
by small interfaces, not at type-definition sites by base classes. The
result is flatter, more refactorable code.
→ [04-functions-methods-interfaces](04-functions-methods-interfaces/)

## When should I use a pointer?

Use a pointer when the receiver or function must mutate the value, when the
value is large and copying is expensive, or when identity must be shared
(e.g., a mutex must not be copied). Use a value for small immutable data,
it is clearer and often faster because values avoid heap indirection and
are easier for escape analysis. Default to value receivers; reach for
pointer receivers when mutation is needed, and be consistent per type.
→ [02-go-language](02-go-language/) and [09-memory-runtime](09-memory-runtime/)

## When should I use a mutex instead of a channel?

Rule of thumb: channels for transferring ownership of data or signaling
across goroutines; mutexes for protecting shared state that stays in place.
If several goroutines read and write a cache, use `sync.RWMutex`. If work
items flow from producers to consumers, use channels. Choosing wrong shows
up as either deadlocks (channels misused as locks) or races (channels
avoided where shared state exists).
→ [08-concurrency](08-concurrency/)

## When should I use generics?

When the same algorithm applies across types with identical code and
type-safe behavior: containers, utility functions like `slices.Map`, or
constraints like `constraints.Ordered`. Do not use generics to fake
inheritance, to avoid designing an interface, or for a single concrete use;
"generify" only the second or third time you duplicate. Readability is
the constraint that wins.
→ [07-generics](07-generics/)

## Why are interfaces implicit?

So behavior can be attached to types you do not own. A type from another
package satisfies your interface without importing it or being modified,
this is how `io.Writer` works across the entire ecosystem. Explicit
implements-headers would force adapters everywhere and couple providers to
consumers. Implicitness keeps dependencies pointing one way: toward the
consumer.
→ [04-functions-methods-interfaces](04-functions-methods-interfaces/)

## Why does nil behave strangely with interfaces?

An interface value is a pair (type, value). A nil interface has no type and
no value. But a non-nil interface holding a typed nil pointer *has* a type,
so it is not nil, and calling a method on it that dereferences the
pointer panics. `var w io.Writer = (*os.File)(nil); w == nil` is false.
This is a consequence of representation, not a bug; check for nil at the
value level when you construct interfaces.
→ [08-concurrency/08-faq-notes.md](08-concurrency/08-faq-notes.md) and section 02

## Why do goroutines leak?

A goroutine leaks when it blocks forever on a channel send/receive, a lock,
or a context that is never cancelled: typically because the other side
went away without closing the channel or because nobody calls cancel.
Leaks accumulate until they exhaust memory or file descriptors. Prevent by
design: whoever creates a goroutine owns its exit; every blocking operation
needs a cancellation path (usually `ctx.Done()`). Verify with tools like
goleak in tests.
→ [08-concurrency/07-pitfalls.md](08-concurrency/07-pitfalls.md)

## How do I structure a Go backend?

Small domain package holding types and business rules; dependencies
(dependencies in, not dependencies out) implemented around it; transport
layer (HTTP, gRPC) converting wire formats to domain calls; main package
wiring it together. Avoid layer-name packages (`utils`, `models`, `handlers`
holding everything). Start with a modular monolith; split into services
only when boundaries are proven.
→ [14-backend-development](14-backend-development/) and [15-microservices](15-microservices/)

## How do I optimize Go?

Measure first: benchmark with `testing.B`, profile with `pprof` (CPU and
heap), trace with `go tool trace`. The usual culprits in order: unbounded
allocations in hot paths, lock contention, chatty serialization, GC
pressure from garbage structs. Fix algorithms and allocations before
micro-optimizations; consider PGO last. Never optimize without a
reproducible benchmark showing before/after.
→ [19-performance](19-performance/)

## How do I debug memory issues?

Start with `go build -gcflags="-m"` to see escape analysis, then heap
profiles from `pprof` to find who allocates, then `runtime/metrics` or
`GODEBUG=gctrace=1` for GC pressure. Sudden growth is usually a leak:
unbounded maps/slices, goroutines holding references, or missing
`T.SetFinalizer`-free long-lived caches. Section 19 walks through the
actual commands with expected output.
→ [19-performance](19-performance/) and [09-memory-runtime](09-memory-runtime/)

## How do I contribute to Go?

Read the [contribution guide](https://go.dev/doc/contribute). Code changes
start with an issue (proposal for language/stdlib changes), CLA signing,
and Gerrit-based review (not GitHub PRs). Non-code contributions: docs,
bugs with reproductions, proposals with data: are equally welcome. For
contributing to Go-based projects (Kubernetes, Kafka clients), section 27
walks the general workflow.
→ [27-open-source](27-open-source/)

## Still stuck?

Search [GLOSSARY.md](GLOSSARY.md) for terminology, or open an issue with
the question if it belongs in this FAQ.
FAQ.
