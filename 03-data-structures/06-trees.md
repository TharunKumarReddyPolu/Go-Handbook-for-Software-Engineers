# Trees

## Why Does This Matter?

Trees model hierarchy: file systems, org charts, tries for prefix
matching, decision trees, JSON. They also have a gravitational pull on
engineers who learned them academically: production Go reaches for a
tree far less often than a map-plus-sort, and knowing which problems
genuinely need a tree (ordered iteration, prefix queries, recursive
domains) versus which just feel tree-shaped is the judgment this
chapter builds.

## Mental Model

A tree is nodes + parent/child discipline. The engineering questions:

- **Balanced or not?** An unbalanced BST degrades to a linked list
  (O(n) lookups); Go ships no balanced-tree stdlib, so if balance
  matters you either pick a library or reconsider.
- **What does iteration order mean?** In-order traversal = sorted for
  BSTs; that is their whole selling point over hash maps.
- **Recursive or iterative?** Go stacks grow dynamically, but deeply
  recursive walks on untrusted input are an exhaustions vector;
  iterative with an explicit stack is the production habit.

```mermaid
flowchart TD
    R["8"] --- L["3"] --- LL["1"] --- LR["6"]
    L --- RL2[""] --- RR2[""]
    R --- RR["10"] --- RRL["14"]
```

## The decision table: tree vs map+sort

| Need | Tree answer | map+sort answer | Verdict |
|---|---|---|---|
| Sorted iteration after bulk load | BST in-order | sort keys once | map+sort wins: simpler, faster |
| Ordered structure with interleaved insert/delete | balanced BST | re-sort each time | tree wins (O(log n) vs O(n log n) per op) |
| Prefix/range queries ("all keys from a-f") | BST range walk or trie | sort + binary search | both fine; tree if updates are frequent |
| Exact lookups only | (no tree needed) | map | map, always |
| Hierarchical domain data | tree/domain model | adjacency + parent map | tree, but often just a struct graph |
| Top-k / running median | heap (Ch. 4) | sort | heap wins |

The stdlib reality: no balanced BST (`red-black`, `avl` are
third-party), but `sort` + `slices.BinarySearch` + maps cover most
"ordered collection" needs. Google's `btree` package is the common
library answer when interleaved ordered updates are genuinely hot.

## Binary search tree: the honest baseline

```go
type Node[T cmp.Ordered] struct {
    key   T
    val   any
    left, right *Node[T]
}

func Insert[T cmp.Ordered](n *Node[T], key T, val any) *Node[T] {
    if n == nil { return &Node[T]{key: key, val: val} }
    switch {
    case key < n.key:  n.left = Insert(n.left, key, val)
    case key > n.key:  n.right = Insert(n.right, key, val)
    default:           n.val = val          // update in place
    }
    return n
}
```

`cmp.Ordered` (Go 1.21+) is the constraint that makes the key type
real. This tree is *unbalanced*: fine for random-ish keys, fatal for
sorted insertion (degenerates to a list). The production framing: use
it when keys arrive unpredictably and n is modest; reach for a library
btree when adversarial ordering or large n is in play.

In-order traversal, iterative (the production shape for untrusted
depth):

```go
func InOrder[T cmp.Ordered](root *Node[T]) []T {
    var out []T
    stack := []*Node[T]{}
    cur := root
    for cur != nil || len(stack) > 0 {
        for cur != nil { stack = append(stack, cur); cur = cur.left }
        cur = stack[len(stack)-1]
        stack = stack[:len(stack)-1]
        out = append(out, cur.key)
        cur = cur.right
    }
    return out
}
```

## Tries: prefix power

```go
type Trie struct {
    children map[byte]*Trie
    terminal bool
}

func (t *Trie) Insert(s string) {
    node := t
    for i := 0; i < len(s); i++ {
        c := s[i]
        child := node.children[c]
        if child == nil {
            child = &Trie{children: map[byte]*Trie{}}
            node.children[c] = child
        }
        node = child
    }
    node.terminal = true
}

func (t *Trie) HasPrefix(p string) bool {  // any word starts with p?
    node := t
    for i := 0; i < len(p); i++ {
        if node = node.children[p[i]]; node == nil { return false }
    }
    return true
}
```

Tries buy O(len) prefix queries regardless of dictionary size: routers
(longest-prefix match), autocompleters, route-table matching. Costs:
memory (nodes per character) and an allocation per branch; for
byte-ASCII keys a `[26]*Trie`-style array children beats a map, for
UTF-8 keys handle runes explicitly (the byte-vs-rune rules are
[02/03](../02-go-language/03-strings-runes-bytes.md)).

The production alternative worth knowing: sorted slice +
`slices.BinarySearch`-style prefix scan (`sort.SearchStrings` for the
first >= prefix, walk while `strings.HasPrefix`). For mostly-static
dictionaries it has far better cache behavior and no per-node
allocations.

## Hierarchies: the domain-tree pattern

Most real trees are domain objects with a parent pointer or adjacency
map, not algorithmic BSTs:

```go
type Org struct {
    ID       string
    ParentID string            // or *Org for in-memory
    Children []*Org
}
```

The subtleties are practical: cycles (an org chart with a loop is a
bug; enforce at insert), orphan cleanup (delete a subtree: walk or
mark), and depth (breadth of org charts is fine, but recursive
serialization of deep structures needs an explicit stack or
`encoding/json` will do it for you via recursion anyway).

## Common Mistakes

- **Unbalanced BST with sorted input**: degenerates to O(n); anyone
  can trigger it by inserting in order. Use a library btree or hash
  map.
- **Recursive traversal on user-shaped input**: deep nesting =
  goroutine stack growth and latency cliffs; iterate.
- **Trie with byte indexing on UTF-8**: `s[i]` splits multi-byte runes
  (the trap from [02/03](../02-go-language/03-strings-runes-bytes.md));
  range over runes or normalize first.
- **map+sort in a hot loop** (sort per request): cache the sorted
  view, or accept the tree; O(n log n) per op at high QPS is a real
  cost.
- **Parent pointers without cleanup discipline**: deleting a node with
  children and no policy leaks the subtree silently.
- **Mutating a tree during iteration**: same discipline as maps;
  collect-then-mutate.

## Idiomatic Go

```go
// Sorted view on demand, the workhorse.
keys := slices.Sorted(maps.Keys(m))            // Go 1.21+
i, found := slices.BinarySearch(keys, target)

// Range query on a sorted slice: cheaper than a tree for read-mostly.
start, _ := slices.BinarySearch(keys, lo)
end, _ := slices.BinarySearch(keys, hi)
window := keys[start:end]
```

## Performance Considerations

- Lookup: map O(1) ~15-25ns vs balanced tree O(log n) with pointer
  chasing ~50-200ns at 1M keys; the tree's win is ordering, not speed.
- Iteration: sorted-slice scans beat tree walks by 5-10x (locality);
  keep a sorted view for read-heavy ordered iteration.
- Tries: O(len) lookups, but memory scales with total characters; a
  million 10-char keys is ~10M+ nodes. Compact tries (radix) or the
  sorted-slice approach trade memory for locality.
- Bulk-load then query: sort once (O(n log n)), then binary-search
  forever; almost always the right shape for read-mostly data.

## Concurrency Considerations

Read-only trees are safe to share; anything else needs the ownership
discipline (one goroutine) or a lock. Copy-on-write shines for
hierarchical config: rebuild the immutable tree, swap an atomic
pointer, readers never lock (the pattern is in
[08-concurrency/04](../08-concurrency/04-sync-primitives.md)).

## Security Considerations

- Adversarial input shapes trees: sorted-insertion DoS on unbalanced
  BSTs, deep-nesting exhaustion on recursive parsers, trie blowup via
  unique prefixes. Bound depth and node counts at ingress.
- Tree walks on path-like input (file paths, route patterns) must
  normalize first (`path.Clean`, no `..` escape) before structure
  decisions; the traversal rules are in
  [21-security](../21-security/README.md).

## Testing Strategy

Trees have the richest property-test surface: after random
insert/delete sequences, in-order traversal must be sorted and contain
exactly the live keys; balanced-tree invariants (where applicable) are
checkable by walk. The four structural cases (empty, leaf, one-child,
two-child delete) are the unit-test table; the differential pattern
(replay ops against map+sort reference) catches subtle pointer bugs.

## Interview Questions

1. *BST vs hash map: what does the tree buy?*: Ordered iteration and
   range queries at O(log n) per update; the map buys raw lookup speed.
2. *Validate a BST?*: In-order traversal must be strictly increasing
   (or min/max bounds recursion); the edge is duplicate keys.
3. *Implement a trie and discuss memory*: O(len) prefix ops; node
   counts scale with total characters; compact/radix variants trade
   code complexity for density.
4. *Lowest common ancestor, two approaches?*: Recursion with path
   tracking (O(n)) vs parent-pointer ascent with a visited set; discuss
   memory vs time.
5. *When do you just use map+sort instead?*: Read-mostly ordered data,
   bulk-load-then-query, small n; say the cache-behavior reason out
   loud.

## Practice Exercises

1. Add `Delete` (with the two-child case) to the generic BST; write the
   four-case table test and an in-order-sorted property test.
2. Build a rune-based trie with `WordsWithPrefix`; compare memory
   against sorted-slice + prefix binary search for a 100k-word
   dictionary.
3. Serialize and deserialize a binary tree (preorder + markers);
   test round-trip on degenerate (linked-list-shaped) trees.

## Further Reading

- [google/btree](https://github.com/google/btree) (the production
  balanced-tree answer, used by etcd)
- [Spec: cmp.Ordered](https://pkg.go.dev/cmp) (Go 1.21+ ordering
  constraint)
- [Trie, radix and Patricia trees](https://en.wikipedia.org/wiki/Radix_tree)
  (the compact-trie family)
