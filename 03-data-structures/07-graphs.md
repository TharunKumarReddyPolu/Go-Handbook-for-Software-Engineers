# Graphs

## Why Does This Matter?

Graphs are the structure of dependencies: service topologies, build
ordering, payment flows, social feeds, network routing. In backend Go
the most common graph problems are concrete: "does this dependency
graph have a cycle?", "what order do I apply migrations?", "how does
this request fan out?", "which node is now unreachable?". The chapter
focuses on those production shapes, with the adjacency map as the
default representation and BFS/DFS/Dijkstra as the three algorithms
you should be able to write cold.

## Mental Model

Two representations, one decision:

| | Adjacency map | Adjacency matrix | Edge list |
|---|---|---|---|
| Shape | `map[K][]K` | `[][]bool` | `[]Edge` |
| Edge lookup | O(deg) | O(1) | O(n) |
| Iterate neighbors | O(deg) | O(V) | O(n) |
| Memory | O(V+E) | O(V²) | O(E) |
| Use when | sparse, arbitrary keys | dense tiny V | streaming, sorting edges |

Sparse is the norm in services (each service talks to a handful of
others), so **adjacency map is the default**. Node keys are whatever
the domain uses: strings, structs (the comparable-struct-key rule from
[02/02](../02-go-language/02-maps.md)).

```go
type Graph[K comparable] struct {
    adj map[K][]K
}
func New[K comparable]() *Graph[K] {
    return &Graph[K]{adj: make(map[K][]K)}
}
func (g *Graph[K]) AddEdge(from, to K) { g.adj[from] = append(g.adj[from], to) }
func (g *Graph[K]) Neighbors(k K) []K  { return g.adj[k] }   // nil for isolated nodes: fine
```

Note what Go's zero values buy: `Neighbors` on an unknown key returns
nil and ranges as empty. No "vertex not found" ceremony.

## BFS: shortest path in unweighted graphs

```go
func (g *Graph[string]) ShortestPath(from, to string) []string {
    if from == to { return []string{from} }
    prev := map[string]string{from: ""}   // visited + path reconstruction
    queue := []string{from}
    for head := 0; head < len(queue); head++ {
        cur := queue[head]
        for _, next := range g.Neighbors(cur) {
            if _, seen := prev[next]; seen { continue }
            prev[next] = cur
            if next == to {
                return reconstruct(prev, to)
            }
            queue = append(queue, next)
        }
    }
    return nil
}

func reconstruct(prev map[string]string, to string) []string {
    path := []string{}
    for cur := to; ; {
        path = append(path, cur)
        if p, ok := prev[cur]; ok && p != "" {
            cur = p
        } else { break }
    }
    slices.Reverse(path)   // Go 1.21+
    return path
}
```

The index-based queue walk (Chapter 3) avoids the reslice-queue trap.
`prev` doubles as the visited set: one map, two jobs, which is the
idiomatic move.

## DFS: cycles, reachability, topological order

Iterative DFS with an explicit stack (the production habit from
[06-trees](06-trees.md); recursion on adversarial graphs is an
exhaustion vector):

```go
func (g *Graph[string]) Reachable(from string) map[string]bool {
    seen := map[string]bool{from: true}
    stack := []string{from}
    for len(stack) > 0 {
        cur := stack[len(stack)-1]
        stack = stack[:len(stack)-1]
        for _, next := range g.Neighbors(cur) {
            if !seen[next] {
                seen[next] = true
                stack = append(stack, next)
            }
        }
    }
    return seen
}
```

**Cycle detection** (the dependency-graph question) is DFS with
three-color marking:

```go
// white: unvisited, gray: in progress, black: done
func (g *Graph[string]) HasCycle() bool {
    const (white, gray, black = 0, 1, 2)
    color := map[string]int{}
    var visit func(k string) bool
    visit = func(k string) bool {
        color[k] = gray
        for _, next := range g.Neighbors(k) {
            switch color[next] {
            case gray:  return true      // back edge: cycle
            case white:
                if visit(next) { return true }
            }
        }
        color[k] = black
        return false
    }
    for k := range g.adj {
        if color[k] == white && visit(k) { return true }
    }
    return false
}
```

**Topological sort** (build order, migration order) is the same DFS
with a finish-time stack: append to `order` after all children are
black, then reverse. Kahn's algorithm (repeatedly take zero-in-degree
nodes) is the queue-based alternative and produces the same order with
friendlier cycle reporting ("the remaining nodes ARE the cycle").

## Dijkstra: weighted shortest paths

```go
func (g *Graph[string]) Dijkstra(from string) map[string]int {
    dist := map[string]int{from: 0}
    type item struct {
        node string
        d    int
    }
    h := heap.New(func(a, b item) bool { return a.d < b.d }) // min-heap, Ch. 4
    h.Push(item{from, 0})
    for h.Len() > 0 {
        cur, _ := h.Pop()
        d, stale := dist[cur.node]
        if stale && d < cur.d {
            continue                  // lazy deletion: skip outdated entries
        }
        for _, next := range g.Neighbors(cur.node) {
            nd := cur.d + g.weight(cur.node, next)
            if old, seen := dist[next]; !seen || nd < old {
                dist[next] = nd
                h.Push(item{next, nd})
            }
        }
    }
    return dist
}
```

The "lazy deletion" line is the production detail: instead of
decrease-key (which `container/heap` makes awkward), push duplicates
and skip stale ones on pop. Memory grows slightly; code complexity
falls dramatically, and it is the standard Go shape.

## Common Mistakes

- **Adjacency matrix by default**: O(V²) memory dies at 100k nodes;
  services are sparse.
- **Recursive DFS on untrusted input**: cycle depth is attacker
  controlled; iterate or bound depth explicitly.
- **Visited tracking only on pop (BFS)**: duplicates flood the queue;
  mark on enqueue.
- **Forgetting reverse edges** in undirected problems: `AddEdge` both
  ways, or the path exists one direction only.
- **Mutating the graph during traversal**: snapshot neighbors or
  collect-then-mutate.
- **Integer weights without overflow checks** on long paths: use int64
  for cumulative distances in big graphs.

## Idiomatic Go

```go
// Dependency ordering, Kahn's algorithm in place.
indeg := map[string]int{}
for _, outs := range g.adj {
    for _, to := range outs { indeg[to]++ }
}
var order []string
ready := []string{}                    // zero in-degree seeds
for k := range g.adj { if indeg[k] == 0 { ready = append(ready, k) } }
for len(ready) > 0 {
    k := ready[len(ready)-1]
    ready = ready[:len(ready)-1]
    order = append(order, k)
    for _, to := range g.Neighbors(k) {
        if indeg[to]--; indeg[to] == 0 { ready = append(ready, to) }
    }
}
if len(order) < len(g.adj) { /* cycle: the leftovers are it */ }
```

## Performance Considerations

- Traversals are O(V+E) with map-indirection constants; for
  performance-critical static graphs, remap string keys to dense ints
  and use `[][]int` adjacency: 5-10x faster (the dense-remap trick
  from [01-builtins-in-depth](01-builtins-in-depth.md)).
- Dijkstra with a binary heap is O((V+E) log V); for the common case
  of few distinct weights, BFS-layered variants (0-1 BFS with a deque)
  beat it.
- Neighborhood scans of huge `[]K` slices: dedupe edges at build; real
  dependency graphs often carry accidental duplicates.

## Concurrency Considerations

Concurrent read-only traversal is safe. Concurrent mutation needs the
ownership discipline: one goroutine owns the graph, requests via
channel ([08-concurrency/01](../08-concurrency/01-goroutines-and-channels.md)),
or an RWMutex for read-mostly service-topology graphs that update on
discovery events. Copy-on-write + atomic pointer swap is excellent for
topology snapshots.

## Security Considerations

- Graphs built from external input (social edges, dependency files)
  get adversarial: star topologies, deep chains, duplicate edges. Bound
  degree, depth, and total edges at ingress.
- Unweighted-shortest-path on user-linked entities can leak
  relationship data; authorize at the query layer, not the traversal
  layer ([21-security](../21-security/README.md)).

## Testing Strategy

Small graphs with known answers are the unit tests (path exists, path
absent, cycle present, disconnected). Property tests: reachability is
symmetric for undirected graphs; topological order never violates an
edge; Dijkstra distances are monotone along shortest paths. The
differential pattern (vs a brute-force reference for n <= 8) catches
heap and bookkeeping bugs cheaply.

## Interview Questions

1. *BFS vs DFS: what does each buy?*: BFS gives shortest unweighted
   paths and level structure; DFS gives cycle detection, topological
   order, and reachability with O(V) memory.
2. *Detect a cycle in a directed graph?*: Three-color DFS; in
   undirected graphs, parent-check DFS or union-find.
3. *Topologically sort a build graph; what if it has a cycle?*: Kahn's
   algorithm; the nodes that never reach zero in-degree are the cycle.
4. *Dijkstra vs BFS: when?*: Weighted, non-negative edges vs
   unweighted; negative weights need Bellman-Ford (and Dijkstra will
   silently give wrong answers).
5. *Design: how do you represent a service dependency graph that
   updates every 30s and is queried constantly?*: Rebuild
   copy-on-write, atomic swap; readers iterate a stable snapshot. Say
   the concurrency reasoning out loud.

## Practice Exercises

1. Implement `AllPaths(from, to string, maxLen int)` with pruning;
   test on a graph with cycles to prove termination.
2. Implement union-find with path compression and rank; use it to
   detect cycles in undirected graphs and compare with the DFS
   approach.
3. Build a migration runner on topological order with partial-failure
   resume; test the "crash mid-order" path deterministically.

## Further Reading

- [Dijkstra's algorithm](https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm)
  (the lazy-deletion variant is standard practice)
- [Kahn's algorithm](https://en.wikipedia.org/wiki/Topological_sorting)
- [google/btree-adjacent: goraph](https://github.com/gyuho/goraph)
  (production-shaped graph library reference)
