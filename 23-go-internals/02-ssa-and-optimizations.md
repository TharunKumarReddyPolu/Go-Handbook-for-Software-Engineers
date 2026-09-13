# SSA & optimizations

## Why Does This Matter?

The optimizations that decide whether your hot loop is fast are
mechanical: they fire when the compiler can *prove* things, and
each has a diagnostic flag that tells you exactly what it proved
about your code. This chapter walks the three passes with the
biggest latency lever: escape analysis, inlining, and bounds-check
elimination, using this section's example package and its real
compiler output. The applied view (how to act on these) is [19
§4](../19-performance/04-compiler-and-pgo.md); here is the
mechanical view: what the pass sees and why it decides.

## Mental Model

SSA (static single assignment) means every variable is assigned
exactly once; control-flow joins merge versions with phi nodes.
That single constraint is what makes the optimizations cheap to
reason about:

```text
x := 1
if c { x = 2 }
y := x + 1
```
becomes: `x1 = 1; x2 = 2; x3 = phi(x1, x2); y = x3 + 1`

Now "which x does y use" is dataflow, not archaeology. Each pass
is a proof attempt over this graph; the diagnostics show you the
proofs.

## How It Works

**Escape analysis** decides, per allocation: stack or heap?

- Stack: the value dies when the function returns. Free
  allocation, free "GC".
- Heap: the value may outlive the frame. Costs an allocation, adds
  GC work.

A value escapes when its address *could* flow somewhere the frame
does not cover: returned, stored in a global, captured by a
goroutine that outlives the call, or passed to something the
analysis cannot see through (an interface, reflection, `fmt`'s
variadic `...any`).

**Inlining** substitutes small function bodies at call sites. The
budget is a cost number (~80): bodies under it inline. The real
prize is not the removed call: an inlined body is *visible to the
other passes*, so escape analysis and dead-code elimination get
better inputs. This is the "inlining enables escape analysis"
effect [19 §4](../19-performance/04-compiler-and-pgo.md) shows in
benchmarks.

**Bounds-check elimination (BCE)** removes the `if i >= len(s) {
panic }` checks the language contract requires. The pass proves
safety from the loop conditions and comparisons you already wrote;
`for i := 0; i < len(s); i++` is a complete proof for `s[i]`.

## Syntax / API

The diagnostics, run against this section's
`examples/internals`:

```text
$ go build -gcflags="-m" ./...
./internals.go:14:6: can inline Sum
./internals.go:28:6: can inline Escaped
./internals.go:36:6: can inline StackLocal
./internals.go:14:10: s does not escape
./internals.go:28:14: moved to heap: v        <- Escaped's v
./internals.go:75:14: make([]string, ...) escapes to heap

$ go build -d=ssa/check_bce/debug=1 ./...
./internals.go:80:28: Found IsInBounds        <- insertion sort
./internals.go:81:29: Found IsInBounds
./internals.go:81:40: Found IsInBounds
```

Read the two outputs together:

- `Escaped`'s local `v` is `moved to heap`: it is assigned to the
  global `sink`, so the analysis must assume it lives forever.
- `StackLocal`'s identical-looking `&v` produces no such line:
  nothing outlives the call. The tests
  (`TestEscaped_Allocates`, `TestStackLocal_DoesNotAllocate`) pin
  both decisions with `testing.AllocsPerRun`.
- `Sum`'s `s[i]` (line 17) appears nowhere in the BCE output: the
  check was *eliminated*. The insertion sort's `keys[j]` (lines
  80-81) remains: the swap's jumbled indexing defeats the proof.

## Basic Example

The heap-decision cost, measurable:

```go
var sink *int

func Escaped(v int) *int { sink = &v; return &v }  // 1 alloc/run
func StackLocal(v int) int { p := &v; return *p+1 } // 0 allocs/run
```

`testing.AllocsPerRun` turns the compiler's decision into a
failing test when it changes. That is the whole trick the example
package uses to make internals regression-testable.

## Real-World Example

Why the insertion sort still has bounds checks, and how to read
that: the pass proves ranges, not cleverness. Rewriting so each
index is provably within `len(keys)` (hoisting `j-1 >= 0` into a
variable comparison) removes checks; restructuring the loop
conditions removes more. But BCE optimizations only matter on hot
loops: the sort in a config loader is cold, and the readable
version should win. Measure first ([19 §1](../19-performance/01-measure-first.md)).

## Production Example

The two escape traps every service hits:

1. **`fmt.Printf("%v", x)` forces x to heap** when x is address-
   taken (the variadic `...any` boxes and escapes). Debug logging
   in a hot path is an allocation story ([20
   §1](../20-observability/01-structured-logging.md)'s lazy
   `LogValuer` fields exist partly for this).
2. **Method values and closures capture by reference**: `go
   func() { process(item) }()` inside a loop allocates per
   iteration (the closure escapes to the new goroutine). That is
   usually fine; in a per-packet path it is the profile's top
   allocator ([19 §2](../19-performance/02-memory-and-allocations.md)).

The audit loop: `pprof` finds the allocation-heavy function; `-m`
explains why; the fix is usually making a value not escape (pass
by value, avoid the interface in the hot path) rather than pooling.

## Common Mistakes

| Mistake | Reality | Do instead |
|---|---|---|
| "Pointers are always faster" | Heap-allocated pointer = GC pressure | Pass values; let small structs stay on the stack ([19 §2](../19-performance/02-memory-and-allocations.md)) |
| Trusting `-m` output across versions | Decisions evolve with the toolchain | Pin behaviors with AllocsPerRun tests, like the example package |
| Fighting the inliner with micro-splitting | Cost budget is mechanical; splitting can *disable* outer inlining | Keep functions small and single-purpose; verify with `-m` |
| Believing BCE means "no checks ever" | New code shape can reintroduce them | Re-check hot loops after refactors (the flag is one command) |
| Manual `unsafe` to dodge checks | Undefined behavior beats optimization | Prove bounds structurally; use the diagnostics |

## Idiomatic Go

- Write loops the pass can prove: `for i := range s` or
  `i < len(s)` conditions; avoid re-deriving lengths mid-loop.
- Prefer value semantics for small structs in hot paths; reach for
  pointers when mutation or sharing is real ([02
  §5](../02-go-language/05-pointers-and-receivers.md)).
- Treat `-gcflags=-m` like `gofmt`: cheap to run, occasionally
  illuminating, part of review for hot-path PRs.

## Performance Considerations

Escape analysis is a *hint generator* for the GC story: fewer
escapes, fewer allocations, less GC, lower p99. The compounding
effects (inlining feeding escape analysis feeding BCE) mean one
structural change can move a whole loop's profile. The example
package's allocation-count tests are the regression harness for
exactly this class of change.

## Concurrency Considerations

Goroutine starts escape their closures by definition (the
goroutine outlives the frame); the cost is one allocation per
`go` statement plus captures. Channels, mutexes, and atomics do
not change escape behavior; what escapes is decided by the data
flow, not the synchronization.

## Security Considerations

Timing side channels survive compiler optimization: the compiler
does not constant-time-protect secret comparisons (`subtle`
exists because of this, [21 §5](../21-security/05-secrets-and-supply-chain.md)).
Conversely, dead-code elimination does not remove security checks
"by accident": if a check is removable, it was already provably
unused. Vet and the compiler are allies here; review still matters.

## Testing Strategy

- `testing.AllocsPerRun` pins escape decisions (used by the
  example package and by `sync.Pool` users everywhere).
- Benchmarks with `-benchmem` catch allocation regressions
  ([10 §4](../10-testing/04-benchmarks-coverage-fuzzing.md)).
- The BCE flag in CI as a *diff* (fail if new checks appear on a
  designated hot file) is a niche but powerful guard.

## Interview Questions

1. Explain escape analysis to a Java colleague. What is the
   difference in philosophy? (Java JIT-escapes at runtime; Go
   decides at compile time, deterministically.)
2. Why does inlining enable other optimizations?
3. Write a function where a local escapes; rewrite it so it does
   not; show both diagnostics.
4. What is a phi node and what property of SSA makes proofs
   easier?

## Practice Exercises

1. Add a function that escapes via interface boxing (`var i any =
   &v`) and one that escapes via goroutine capture; verify with
   `-m` and pin with `AllocsPerRun`.
2. Remove the insertion sort's bounds checks via loop
   restructuring; benchmark before/after and report whether it
   was worth the readability ([19 §1](../19-performance/01-measure-first.md)'s
   rule).
3. Find one `fmt.Sprintf` in a hot path in a codebase you own;
   check `-m` for the escape; replace with a typed field.

## Further Reading

- [Compiler optimizations wiki (Go source)](https://github.com/golang/go/wiki/CompilerOptimizations)
- [Go compiler internal docs: SSA](https://github.com/golang/go/blob/master/src/cmd/compile/internal/ssa/README.md)
- [Escape analysis diagnostics reference](https://pkg.go.dev/cmd/compile)
