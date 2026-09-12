# Generic APIs

## Why Does This Matter?

A generic function's signature is a contract written in three
languages at once: the compiler's (constraints), the reader's (what
types work and why), and the maintainer's (what can change without
breaking). The difference between a generic API that ages well and
one that accretes workarounds is signature design, and the stdlib's
`slices`/`maps` packages are the canonical study: small, ruthless,
and worth reading as documentation of the intended style.

## Mental Model

The signature is the whole API; the body is an implementation detail:

```go
func Contains[S ~[]E, E comparable](s S, v E) bool
//         ^~~~~~~^  ^~~~~~~~~~^
//         container  element: exactly what Contains needs (== only)
```

Read it as a specification: "any slice-typed thing (including named
slice types), whose elements support equality, probed for one value."
Nothing more is promised; nothing less is licensed. The stdlib's
style rules fall out of that discipline:

1. **Constrain the container by underlying type** (`~[]E`): callers
   with `type Row []string` work unmodified.
2. **Constrain elements minimally**: `comparable` for equality work,
   `cmp.Ordered` for ordering, `any` when nothing is needed.
3. **Name type parameters by role**: `S` slice, `E` element, `K` key,
   `V` value, `T` when there is only one. Single letters, consistent
   across the package.
4. **Order type parameters data-first**: the container, then the
   element, then extras; call sites read left to right the same way.

## The catalog: what the stdlib solved so you do not have to

| Need | stdlib answer | Note |
|---|---|---|
| Membership | `slices.Contains(s, v)` | E comparable |
| Index of | `slices.Index(s, v)` | -1 when absent |
| Sort | `slices.Sort(s)` / `slices.SortFunc(s, cmp)` | Ordered / any+comparator |
| Sort stable | `slices.SortStableFunc` | preserves equal order |
| Search sorted | `slices.BinarySearch(s, v)` / `SortFunc` variant | returns (i, found) |
| Compare | `slices.Compare` / `CompareFunc` | lexicographic |
| Equal | `slices.Equal` / `EqualFunc` | element-wise |
| Clone / Copy | `slices.Clone(s)` / `slices.Copy` | reallocation vs overwrite |
| Delete | `slices.Delete(s, i, j)` / `DeleteFunc` | zeroed since Go 1.22 |
| Insert | `slices.Insert(s, i, vs...)` | grows |
| Compact | `slices.Compact` / `CompactFunc` | dedupe adjacent |
| Grow / Clip | `slices.Grow(s, n)` / `Clip` | capacity control |
| Reverse | `slices.Reverse(s)` | in place |
| Keys / Values | `maps.Keys(m)` / `maps.Values(m)` | iter.Seq, Go 1.23+ |
| Clone map | `maps.Clone(m)` | nil-safe |
| Equal map | `maps.Equal(m1, m2)` | same rules |
| min/max | `min(a, b)` / `max(a, b)` builtins | Go 1.21, no import |
| Clear | `clear(m)` / `clear(s)` builtin | Go 1.21 |

Two habits this table should install: **before writing any generic
helper, check `slices`, `maps`, `cmp`, and the builtins**; and **the
helper you were about to write probably reveals a naming or design
gap, not a missing function**.

## Designing your own: the pattern that reads like documentation

The stdlib covers primitives; domain APIs follow the same style with
domain constraints:

```go
// The signature is the docs: "any container of IDs, deduped, order
// preserved".
func DedupIDs[S ~[]E, E ~string](ids S) S {
    seen := make(map[E]struct{}, len(ids))
    out := make(S, 0, len(ids))
    for _, id := range ids {
        if _, ok := seen[id]; !ok {
            seen[id] = struct{}{}
            out = append(out, id)
        }
    }
    return out
}
```

Design rules earned from real API reviews:

- **The narrowest constraint that the body needs.** Every extra
  capability the constraint grants is a promise the body might use
  later, which is a compatibility lock
  ([06-packages-modules/06](../06-packages-modules/06-semantic-versioning.md)):
  loosening a constraint later is fine; tightening it breaks callers.
- **Return concrete shapes, take abstract ones**: `make(S, ...)`
  returning `S` (the caller's own named type) preserves identity;
  returning `[]E` silently downgrades `Row` to a plain slice.
- **Anchoring**: at least one regular parameter should carry each
  type parameter so inference works
  ([02](02-type-parameters-and-constraints.md)); a function whose
  type parameters appear only in results forces explicit brackets at
  every call.
- **Comparator parameters over comparison constraints** when the
  order is domain logic: `func TopK[T any](...) less func(a, b T)
  bool` (the heap case study in
  [03](03-generic-data-structures.md)); `cmp.Ordered` only when
  builtin ordering is genuinely the contract.
- **Do not generic-ify one call site**: a bracket that appears exactly
  once in the codebase is a constraint with no constituency; write
  the concrete function (the section's standing rule).

## Error handling in generic APIs

Generics do not change the error contract
([05-errors](../05-errors/README.md)); the trap to avoid is
*inventing new failure shapes* per instantiation:

```go
// Good: the same sentinel surfaces for every instantiation.
var ErrEmpty = errors.New("empty")

func Max[T cmp.Ordered](xs []T) (T, error) {
    if len(xs) == 0 {
        var zero T
        return zero, ErrEmpty
    }
    ...
}
```

The failure mode to avoid: panics inside generic helpers "because
the constraint guarantees it". Constraints guarantee types, not
values; an empty slice satisfies `[]T` for any T. The zero+error
pattern above is the shape.

## Common Mistakes

- **Bracket sprawl**: `func F[T1, T2, T3 any](a T1, b T2, c T3)`
  where two parameters could be concrete: each type parameter must
  earn its place by enabling multiple instantiations *that exist*.
- **Constraints wider than the body**: `any` with a runtime type
  switch inside is erasure wearing generics ([01](01-why-generics-exist.md));
  the switch's cases are the real constraint, write them down.
- **Returning `any`** from a generic function: throws away the type
  information the brackets just established; return `T` or `R`.
- **Generic methods on non-generic types pre-1.27 workarounds**:
  wrapper structs ("view types") carrying one method; Go 1.27
  retires most of these ([06](06-version-notes.md)).
- **Package-level state in generic functions**: a generic function
  with a cache keyed by instantiation looks clever and behaves
  surprisingly; state belongs in instantiated types, not in generic
  code paths.
- **Exported generic helpers without examples in godoc**: the
  constraint syntax is dense; an Example function per tricky
  signature is the cheapest documentation you will ever write.

## Idiomatic Go

```go
// A small, real generic API: the retry helper every service has.
type Backoff interface {
    Next(attempt int) time.Duration
}

func Retry[T any](
    ctx context.Context,
    op func(context.Context) (T, error),
    b Backoff,
) (T, error) {
    var zero T
    for attempt := 0; ; attempt++ {
        v, err := op(ctx)
        if err == nil {
            return v, nil
        }
        if ctx.Err() != nil {
            return zero, ctx.Err()  // context dead: its error wins
        }
        d := b.Next(attempt)
        if d <= 0 {
            return zero, err        // out of patience: the op's error
        }
        select {
        case <-time.After(d):
        case <-ctx.Done():
            return zero, ctx.Err()
        }
    }
}
```

Note the two failure exits are deliberately distinct: a dead context
returns `ctx.Err()` (the caller must see cancellation, not a stale
op error: this test-found distinction is pinned in
[examples/genlib](examples/genlib/)), while exhausted backoff returns
the operation's error.

Note what the brackets buy here: the operation's *result type* flows
through unchanged (pre-generics this exact helper required `any` and
an assertion at every call, or per-type copies). `T any` is the right
constraint because the function treats the result as opaque: the
narrowest honest contract.

## Performance Considerations

- Generic API boundaries compile to the same calls concrete ones do;
  the design cost is zero, the readability win is the point.
- Avoid forcing conversions at call sites: if callers must cast into
  your constraint shapes, the constraint is wrong-shaped
  (`~`-types usually fix it).
- Hot-loop generic helpers over scalars: the stenciling dict adds
  nanoseconds; if profiles show it, specialize that one path
  ([03](03-generic-data-structures.md),
  [19-performance/01](../19-performance/01-measure-first.md)).

## Concurrency Considerations

Generic helpers that take callbacks or spawn goroutines inherit every
rule from their concrete cousins: context propagation, WaitGroup
discipline, race-detector runs. The `Retry` above is concurrency-safe
because it holds no state and honors ctx: the same review checklist
as any helper ([08-concurrency](../08-concurrency/README.md)).

## Security Considerations

Constraints are compile-time narrowing: a `~[]byte`-constrained API
refuses the struct someone "just wants to pass through". The security
relevance is boundary honesty: generics let you type the boundary
precisely so validation happens at ingress with real types, not with
assertions after the fact ([21-security](../21-security/README.md)).

## Testing Strategy

Test the API through its instantiations, and test the *constraint*
by trying to violate it (a deliberately wrong instantiation in a
compile-check test or a commented failure case). The
[examples/genlib](examples/genlib/) package demonstrates
instantiation tables plus the satisfaction checks for named types
under `~` constraints.

## Interview Questions

1. *Walk through the design decisions in `slices.Contains`'s
   signature.*: `~[]E` (named slices qualify), `E comparable` (the
   minimum `==` needs), S-first ordering, inference-friendly.
2. *Why is loosening a constraint later safe but tightening it
   breaking?*: Callers are licensed by what the constraint allows
   today; removing a capability breaks bodies; adding one breaks
   callers who passed types without it.
3. *When do you take a `less func` instead of `cmp.Ordered`?*: When
   ordering is domain logic; constraints express operator needs,
   functions parameterize policies.
4. *What is anchoring and why does it matter?*: A type parameter
   must appear in a regular parameter position so inference can fix
   it from call sites; unanchored generics force brackets everywhere.
5. *A teammate proposes a generic helper used at exactly one call
   site; your review?*: Concrete function until the second use; the
   bracket is a maintenance tax with no constituency yet.

## Practice Exercises

1. Reimplement three `slices` functions from memory; diff against the
   stdlib source; note every signature decision you got wrong.
2. Design `ChunkBy[S ~[]E, E any](s S, size int) [][]S` and `Flatten`;
   decide and document their empty-input contracts; instantiate with
   named slice types.
3. Write the godoc example for `Retry`; make the example itself a
   test (the Example pattern from
   [10-testing/01](../10-testing/01-fundamentals.md)).

## Further Reading

- [slices package](https://pkg.go.dev/slices)
- [maps package](https://pkg.go.dev/maps)
- [When to use generics](https://go.dev/blog/when-generics) (the
  official decision framework)
