# Stack & heap

## Why Does This Matter?

Every value your program creates lives in one of two places, and the
choice drives allocation cost, GC pressure, cache behavior, and
therefore most of your service's performance envelope. Go is unusual
among garbage-collected languages in making the decision at compile
time and making it *visible*: you can ask the compiler where any
value went, and you can design types so the answer is usually
"stack". This chapter is the foundation the rest of the section
builds on ([23 §3](../23-go-internals/03-runtime-architecture.md)'s
runtime architecture, [23 §2](../23-go-internals/02-ssa-and-optimizations.md)'s
escape mechanics, and [19 §2](../19-performance/02-memory-and-allocations.md)'s
applied allocation work).

## Mental Model

```text
Stack (per goroutine)          Heap (shared, GC-managed)
  grows/shrinks automatically    allocated in size classes
  ~2KB initial                   freed only by the GC
  free at "allocation"           costs ~25ns + eventual GC work
  dies when frame returns        lives until unreachable
```

The lifetime rule that decides everything: **a value can live on the
stack exactly when the compiler can prove it does not outlive the
frame.** Everything else goes to the heap. That proof is escape
analysis (chapter 2); here are the two places values live and what
each costs.

**Goroutine stacks**: contiguous and growable. Each goroutine starts
with a small stack (~2KB); on overflow the runtime allocates a
bigger one and *copies* the old stack over, adjusting pointers. The
copy is why Go abandoned segmented stacks: no "hot split" penalty,
amortized O(1) growth. A million goroutines with shallow stacks is
~a few GB, not the ~1MB-per-thread of OS threads.

**The heap**: the runtime allocator ([23
§3](../23-go-internals/03-runtime-architecture.md)) serves small
objects from 67 size classes via per-P caches (lock-free on the hot
path), then central lists, then the page heap. Every heap allocation
is a loan from the GC: the object stays live until the collector
proves nobody can reach it.

## How It Works

Pointer vs value semantics is the engineer-facing half of the
stack/heap decision ([02
§5](../02-go-language/05-pointers-and-receivers.md) covers the
design rules; here is the memory view):

| Choice | Memory consequence | When right |
|---|---|---|
| Value semantics (copy) | Values stay on the stack; zero GC pressure | Small structs, hot paths, data ownership transfer |
| Pointer semantics (share) | Value escapes to the heap; one allocation + GC tracking | Mutation, large structs, shared state |

The counterintuitive part for engineers from C++ or Java: in Go,
copying a small struct is *cheaper* than sharing it, because the
copy is registers-and-stack while the share is heap + GC. The
threshold is small: a few machine words. Measure at the boundary
with benchmarks ([19 §1](../19-performance/01-measure-first.md)).

## Syntax / API

Asking the compiler where a value went:

```bash
go build -gcflags="-m" ./pkg/ 2>&1 | grep escapes
# ./x.go:12:9: &v escapes to heap
# ./x.go:30:5: v does not escape
```

And measuring allocation volume (the section's example pins these
with `testing.AllocsPerRun`, [23
§2](../23-go-internals/02-ssa-and-optimizations.md)):

```go
avg := testing.AllocsPerRun(100, func() { StackLocal(42) }) // 0
```

## Basic Example

The same computation, two memory stories:

```go
type point struct{ x, y float64 }

func distValue(p point) float64 {          // p is a copy: stack
    return math.Hypot(p.x, p.y)
}

func distPtr(p *point) float64 {           // p escapes if returned/stored
    return math.Hypot(p.x, p.y)
}
```

`distValue` forces the caller's `point` into registers or the stack
frame. `distPtr` is equally fast when the caller's point already
lives on the heap or stack, but a call chain that creates a point
and passes its address to something non-inlinable will heap it.

## Real-World Example

The JSON handler that allocates per request: decode allocates the
request struct (heap: it escapes into the decoder), business logic
allocates intermediate maps, encode allocates the response buffer.
None of this is wrong: it is the cost of doing business. The
optimization order that works ([19 §2](../19-performance/02-memory-and-allocations.md)'s
playbook): buffer reuse first (`sync.Pool` for the scratch), then
preallocation (`make(map, n)`), then value semantics for the small
hot structs, then measure again. The point of this chapter: each
fix works *because* of the stack/heap model; the buffer you reuse
never re-enters the allocator.

## Production Example

**Why p99 climbs with allocation rate**: the GC runs more often as
garbage volume rises ([20 §2](../20-observability/02-metrics.md)'s
`/gc/pauses` chart), and marking costs CPU proportional to live
pointer density. A service that allocates 100x its live set per
request window spends real CPU in `mallocgc` and GC assist: visible
in profiles as `runtime.mallocgc` and `runtime.gcAssistAlloc`. The
remedy is allocation discipline, not GC tuning ([19
§2](../19-performance/02-memory-and-allocations.md)'s tuning order:
GOMEMLIMIT first, GOGC rarely, allocations always).

**The stack-growth corner**: deep recursion (parsing nested untrusted
input!) grows stacks per goroutine; a million-goroutine service each
holding a 64KB stack after one deep call is 64GB. Bound recursion
depth at trust boundaries ([21 §1](../21-security/01-threat-model-and-validation.md)'s
input rules); goroutine stacks do not shrink aggressively.

## Common Mistakes

| Mistake | Reality | Do instead |
|---|---|---|
| Pointers "to avoid copying" | Small copies are cheaper than heap + GC | Values for small hot structs; pointers for mutation/sharing |
| Assuming all goroutine stacks are 2KB | They grow on demand and stay grown | Bound recursion; watch `NumGoroutine` x stack depth in capacity math |
| Fighting escape analysis with micro-idioms | The proof is structural: return/store/capture escapes | Keep hot values frame-local; check with `-m` |
| Tuning GC before fixing allocation volume | Allocation rate dominates GC cost | Reduce allocations first ([19 §2](../19-performance/02-memory-and-allocations.md)) |
| Copying huge structs on hot paths | Cache-line traffic is real | Pointers past a few machine words; benchmark the boundary |

## Idiomatic Go

- Default to values; introduce pointers when mutation, sharing, or
  size demands them.
- Accept slices/strings freely (headers are cheap); be deliberate
  about structs and arrays.
- Keep constructors returning values for small types; the escape is
  the caller's choice, not the constructor's.

## Performance Considerations

The cost ladder per allocation: per-P cache hit (~25ns, no lock) →
central list (lock) → page heap (bigger lock, more work) → GC work
amortized across everything. Size classes mean requesting 65 bytes
consumes 80: capacity math for cache-heavy services counts classes,
not bytes ([19 §2](../19-performance/02-memory-and-allocations.md)).
The stack path costs ~nothing, which is why the whole game is
keeping hot values off the heap.

## Concurrency Considerations

Per-goroutine stacks need no synchronization: values that never
escape are race-free by construction. Sharing (pointers) is what
creates races and what makes the memory model (chapter 6 in [23](../23-go-internals/),
chapter [23 §6](../23-go-internals/06-memory-model.md)) matter. The
design rule: transfer ownership by value where possible; when you
must share, share through the sync primitives' edges.

## Security Considerations

Stack growth on untrusted input depth is a memory-exhaustion vector
(bound recursion); heap growth on untrusted input *size* is the
same ([21 §4](../21-security/04-limits-and-hardening.md)'s body
caps). Secrets on the heap outlive their use (GC decides when);
wiping is best-effort in a GC language ([23
§5](../23-go-internals/05-interfaces-slices-strings.md)'s string
notes): minimize secret copies instead.

## Testing Strategy

- `testing.AllocsPerRun` pins the no-allocation contracts of hot
  helpers (the pattern used in [23
  §2](../23-go-internals/02-ssa-and-optimizations.md)'s example).
- Benchmarks with `-benchmem` track B/op and allocs/op over time;
  benchstat for before/after honesty.
- Capacity tests assert structural facts (like this section's
  `TestBig_PinVersusCopy`) without flirting with GC internals.

## Interview Questions

1. When does a value go to the heap even if it never leaves the
   function? (Interface boxing, reflection, closures that escape,
   non-inlinable calls taking addresses.)
2. Why is copying a small struct often faster than passing a
   pointer?
3. How do growable stacks work and what problem did segmented
   stacks have?
4. Where does allocation cost actually come from (cache vs central
   vs GC), and what does that imply for buffer reuse design?

## Practice Exercises

1. Write one function where `&v` escapes and a twin where it does
   not; prove both with `-m` output and `AllocsPerRun` tests.
2. Benchmark `distValue` vs `distPtr` at point-counts 1, 8, 64;
   find the crossover and explain it.
3. Force stack growth with recursion in a goroutine; measure
   goroutine memory before/after with `runtime/metrics`.

## Further Reading

- [Go Memory & Memory Allocation (GC guide)](https://go.dev/doc/gc-guide)
- [Stack growth design doc](https://docs.google.com/document/d/1wAaf1rYoM4S4gncZ0hr9GnI1tkZ8TBonmPFBwiZwnNQ)
- [Escape analysis diagnostics](https://pkg.go.dev/cmd/compile)
