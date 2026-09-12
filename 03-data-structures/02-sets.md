# Sets via maps

## Why Does This Matter?

Go has no built-in set type, and does not need one: the map-plus-empty-
struct idiom covers 95% of uses with zero imports. The remaining 5%
(order-preservation, multiset semantics, set algebra at scale) is where
judgment lives. Because the idiom is universal in Go codebases, reading
and writing it fluently is table stakes; implementing it *well* (with
generics, in a small reusable shape) is the interview differentiator.

## Mental Model

A set is a map whose value type carries no information:

```go
present := map[string]struct{}{}
present["k"] = struct{}{}     // zero-size value
_, ok := present["k"]         // the bool IS the answer
```

`struct{}` is a zero-byte type, so the bucket stores only the key: a
`map[string]struct{}` with a million members costs essentially the
memory of the keys plus bucket overhead, nothing more. `bool` values
work but cost a byte per slot and invite "false means what?" ambiguity;
`struct{}` forces the comma-ok idiom to be the only read path.

## The reusable generic Set

```go
type Set[T comparable] map[T]struct{}

func New[T comparable](items ...T) Set[T] {
    s := make(Set[T], len(items))
    for _, it := range items {
        s[it] = struct{}{}
    }
    return s
}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Remove(v T)   { delete(s, v) }
func (s Set[T]) Len() int     { return len(s) }
```

Set algebra composes from the same shape:

```go
func Union[T comparable](a, b Set[T]) Set[T] {
    out := make(Set[T], a.Len()+b.Len())
    for v := range a { out[v] = struct{}{} }
    for v := range b { out[v] = struct{}{} }
    return out
}

func Intersect[T comparable](a, b Set[T]) Set[T] {
    small, large := a, b
    if a.Len() > b.Len() { small, large = b, a }  // iterate the smaller
    out := make(Set[T], small.Len())
    for v := range small {
        if large.Has(v) { out[v] = struct{}{} }
    }
    return out
}

func Difference[T comparable](a, b Set[T]) Set[T] {
    out := make(Set[T], a.Len())
    for v := range a {
        if !b.Has(v) { out[v] = struct{}{} }
    }
    return out
}
```

Note the micro-decision in `Intersect` (iterate the smaller set): for
mixed sizes it is a real constant-factor win; it also shows the
template for all set operations, iterate + probe.

## When the plain map is enough

Most production sets are one-off membership checks inside a function.
The type ceremony of a generic Set is worth it when: the set crosses
function boundaries, the operations read like algebra (diff/intersect),
or it appears in three places. Inside one function, prefer the raw
idiom: fewer indirections, zero API to learn.

## The variants worth knowing

| Need | Shape | Cost note |
|---|---|---|
| Membership only | `map[T]struct{}` | optimal memory |
| Counting membership (multiset) | `map[T]int` | the bag; negative counts are a bug class |
| Ordered output | sort keys on demand | `slices.Sorted(maps.Keys(s))`, Go 1.21+ |
| Order-preserving set | map + slice of keys | 2x memory; see below |
| Huge universes, sparse | bitset (`[]uint64`) | 1 bit per element; see below |

The **ordered set** pattern (map for membership, slice for sequence):

```go
type OrderedSet[T comparable] struct {
    seen  map[T]struct{}
    order []T
}
func (o *OrderedSet[T]) Add(v T) bool {
    if _, ok := o.seen[v]; ok { return false }
    o.seen[v] = struct{}{}
    o.order = append(o.order, v)
    return true
}
```

This is the dedup-while-preserving-order workhorse for stream
processing; `seen` without `order` is the classic mistake (output order
becomes randomized, which [02/02-maps](../02-go-language/02-maps.md)
explains is guaranteed chaos, not just unlucky).

The **bitset** for dense small-int universes:

```go
type Bitset []uint64
func (b Bitset) Has(i int) bool    { return b[i/64]&(1<<(i%64)) != 0 }
func (b Bitset) Add(i int)         { b[i/64] |= 1 << (i % 64) }
```

64x denser than `map[int]struct{}`; right for page-id sets, feature
flags, bloom-filter internals. Wrong when the universe is sparse or
unbounded: indices must be validated against the backing length.

## Common Mistakes

- **`map[string]bool` sets** where the zero-value semantics of a bool
  invite `if set[k] {` (which works!) but also `set[k] = false`
  (which "removes" in one place, "does not remove" in another). Pick
  `struct{}` and `delete`.
- **Nil set as read-only is fine; nil set writes panic** (the nil-map
  rule from [02/02](../02-go-language/02-maps.md)); a `Set` method
  should document whether it tolerates a nil receiver.
- **Mutating a set while ranging** and expecting deterministic
  results: same rules as map iteration; collect-then-mutate if needed.
- **Using a set where a slice + `slices.Contains` is fine**: at small
  n (roughly under 20-50), building a map costs more than the scans it
  saves.
- **Copy semantics**: `b := a` shares the set; "copy" means rebuild or
  a documented `Clone` (maps.Clone, Go 1.21+).

## Idiomatic Go

```go
// Dedup preserving order: the one-liner every codebase has.
seen := make(map[string]struct{}, len(in))
out := in[:0]  // careful: in-place; copy first if in is shared
for _, v := range in {
    if _, ok := seen[v]; !ok {
        seen[v] = struct{}{}
        out = append(out, v)
    }
}

// Difference for a permission check.
missing := Difference(required, granted)
if missing.Len() > 0 { /* deny with details */ }
```

## Performance Considerations

- Set ops are O(len) of what you iterate; always iterate the smaller
  side of an intersection.
- Pre-size with `make(Set[T], n)`: same rehash-avoidance win as maps.
- For set-heavy algebra on large membership, consider a sorted-slice
  representation: `slices.BinarySearch`-based ops are O(n+m) with
  perfect cache behavior and 5-10x less memory than a map when
  membership is mostly static. This is the classic "sorted vector set"
  tradeoff; benchmark both if set memory shows in the heap profile
  (see [19-performance/02](../19-performance/02-memory-and-allocations.md)).

## Concurrency Considerations

A set is a map: concurrent mutation is a fatal runtime error. The
generic Set above is not concurrent-safe, and should not pretend to be;
wrap with `sync.RWMutex` at the owning layer or shard. Copy-on-write
(rebuild the whole set, swap the pointer) is an excellent pattern for
read-mostly permission sets: readers need no lock, writers pay O(n).

## Security Considerations

- Permission and deny-list sets must be built from validated input at
  boot, not mutated by request handlers: an unvalidated `Add` is a
  privilege-escalation primitive.
- String-keyed sets of user-controlled values grow unboundedly: cap or
  evict (LRU, Chapter 9).

## Testing Strategy

Set algebra has property-test-friendly invariants: `Union(a,b) ==
Union(b,a)`, `Intersect ⊆ a`, `Difference(a,a) == ∅`, `Has` after `Add`
is always true. Fuzz or property-test those relations with small
generators; the technique is from
[10-testing/04](../10-testing/04-benchmarks-coverage-fuzzing.md). The
`examples/dsbench` package includes a contract test suite for the
generic Set.

## Interview Questions

1. *Why `struct{}` values instead of `bool`?*: Zero memory and the
   comma-ok read path is the only read path; no true/false ambiguity.
2. *Implement `Intersect` efficiently: what do you iterate first?*:
   The smaller set; the other is probed. O(min) work after build.
3. *How do you preserve insertion order in a set?*: Map for membership
   plus slice for sequence; or sort keys on demand when order is only
   needed at output.
4. *When is a sorted slice a better set than a map?*: Mostly-static
   membership, memory-sensitive, binary-search-shaped queries; wins on
   cache behavior and density.
5. *How do you make a read-mostly permission set safe for concurrent
   readers?*: Copy-on-write with atomic pointer swap, or RWMutex;
   `sync.Map` only for its niche shapes.

## Practice Exercises

1. Add `Equal`, `Subset`, and `Clone` to the generic Set; property-test
   the algebra invariants with a small generator.
2. Implement the sorted-slice set and benchmark it against the map set
   for 1M string members with a 10% mutation rate; report the memory
   delta.
3. Build the ordered-set dedup and prove (test) that output order
   equals first-occurrence order, including duplicates separated by
   other values.

## Further Reading

- [Go maps in action](https://go.dev/blog/maps) (the set idiom's
  foundation)
- [`maps` package](https://pkg.go.dev/maps) (Clone, Keys, Go 1.21+)
- [Bloom filters explained](https://pages.cs.wisc.edu/~cao/papers/summary-cache/node8.html)
  (when even a bitset is too big)
