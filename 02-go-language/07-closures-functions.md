# Function types, closures & variadics

## Why Does This Matter?

Functions are values in Go: they get types, they cross APIs, they
close over variables. This is the machinery behind middleware chains,
`http.HandlerFunc`, context cancellation, sort comparators, and every
"pass a callback" API you will use or write. Capture semantics are also
where subtle concurrency bugs hide: a closure that captures a loop
variable or a shared counter is a race waiting for production.

## Mental Model

A function type specifies parameters and results only; the name is not
part of the type. Two functions with the same signature are
interchangeable:

```go
type Handler func(ctx context.Context, req *Request) (*Response, error)
```

A **closure** is a function value plus the variables it captures.
Captured variables are captured *by reference*, not copied: the
closure sees later mutations, and mutations through the closure are
visible outside. This is the part to internalize:

```go
func counter() func() int {
    n := 0
    return func() int {
        n++             // mutates the captured n
        return n
    }
}
c := counter()
c(); c(); c()           // 1, 2, 3: n outlives the function call
```

Each call to `counter` creates a **fresh** `n`: closures instantiate
their captured environment. This is why the pattern works for
per-request state.

## First-class functions in real APIs

```go
// Middleware is just a function returning a function.
func WithLogging(next Handler) Handler {
    return func(ctx context.Context, req *Request) (*Response, error) {
        log.Println("start")
        resp, err := next(ctx, req)
        log.Println("done")
        return resp, err
    }
}

// http.HandlerFunc: adapter from func to interface.
mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}))
```

The stdlib is full of this shape: `http.HandlerFunc`,
`context.WithCancelCause`, `sort.Slice`/`slices.SortFunc` comparators,
`time.AfterFunc`, `sync.OnceFunc` (Go 1.21+).

## Variadic functions: the rules that matter

```go
func Sum(nums ...int) int { /* nums is []int */ }

Sum(1, 2, 3)        // slice built for you: [3]int{...}
Sum(vals...)        // spread: pass an existing slice (no copy)
Sum()               // empty slice, not nil (len 0)
```

Three rules worth memorizing:

1. `nums` is a real slice; you can slice, range, and append to it.
2. Spreading `s...` passes the slice *itself*: mutations inside the
   function are visible to the caller when they write through to the
   backing array. A variadic literal call (`Sum(1,2,3)`) creates a
   fresh array, invisible outside.
3. Variadic must be the last parameter; there is no keyword-argument
   machinery in Go.

The functional-options pattern (built from variadics plus closures)
is the standard library-adjacent answer to constructors with many
optional settings; worked example in
[04-functions-methods-interfaces](../04-functions-methods-interfaces/README.md).

## Deferred functions, captured

`defer` captures the function value and its arguments *at defer time*
(you will see this fully in 01-fundamentals); closures inside loops
capture the loop variable itself, which is why the pre-Go 1.22 footgun
existed. [Go 1.22](https://go.dev/doc/go1.22) made loop variables
per-iteration, closing the classic bug:

```go
for _, job := range jobs {
    go func() { process(job) }()   // safe in Go 1.22+: job is per-iteration
}
```

On any toolchain you do not control, the explicit copy costs nothing:

```go
job := job // shadow with the iteration's value
go func() { process(job) }()
```

## Common Mistakes

- **Capturing the loop variable** on pre-1.22 toolchains: all
  goroutines see the last value. Test with `-race` and shuffled order.
- **Assuming variadic spread copies**: `f(s...)` aliases `s`; a
  variadic literal call does not. When the function writes through to
  the slice, that difference is a bug.
- **Capturing large structures** in long-lived closures: the closure
  pins the captured variables in memory for its lifetime (escape
  analysis will heap-allocate them).
- **Capturing `t *testing.T` in parallel subtests** pre-1.22 semantics;
  since Go 1.22 `t.Run` shadowing works as expected, but explicit
  passing remains clearer.
- **Returning functions that capture locks**: a closure holding a
  mutex-guarded pointer can serialize unrelated code paths; know what
  your closures pin.

## Idiomatic Go

```go
// Closures for one-off glue; named functions for reuse.
onDone := func(err error) { close(done) }

// Options pattern: variadic + closure.
srv := NewServer(":8080",
    WithTimeout(5*time.Second),
    WithLogger(slog.Default()),
)
```

## Performance Considerations

- Closures that capture variables force those variables to the heap if
  the closure outlives the frame; loops that allocate a closure per
  iteration allocate per iteration. Hoist invariants out of hot loops.
- Function values are two-word headers (pointer + closure env);
  calling through them is an indirect call: same cost class as
  interface dispatch.
- Variadic calls allocate the slice unless the compiler can prove it
  can stack-allocate; spreading an existing slice (`f(s...)`) avoids
  the new allocation.

## Concurrency Considerations

A closure capturing a mutable variable is a shared mutable variable
with extra steps. If two goroutines call closures sharing one captured
variable, that is a data race (see [08-concurrency/07-pitfalls](../08-concurrency/07-pitfalls.md)).
Capture-by-value deliberately by copying into a local first, and let
the race detector prove it.

## Security Considerations

- Callbacks are an execution path into your code: validate inputs
  inside the closure exactly as you would in a public method.
- Long-lived closures holding credentials, tokens, or DB handles
  extend the lifetime of sensitive state; capture identifiers and fetch
  at point of use ([21-security](../21-security/README.md)).

## Testing Strategy

Closures make excellent inline fakes (a counter closure in a test
needs no mock library). Assert on behavior across multiple calls to
verify statefulness, and run with `-race` when the closure spans
goroutines. The `examples/seqops` package demonstrates a closure-based
predicate API with table-driven tests.

## Interview Questions

1. *What does a closure capture, and by what mechanism?*: The
   referenced variables, by reference; mutations are visible both ways,
   and the variables outlive the frame.
2. *Why did loop-variable capture break before Go 1.22?*: One variable
   per loop, shared by all closures; 1.22 made it per-iteration.
3. *Difference between `f(s...)` and `f(s[0], s[1])`?*: Spread passes
   the existing slice (aliasing possible); the literal form builds a
   fresh array.
4. *What is the cost of a variadic call?*: A slice allocation (usually),
   unless spread from an existing slice or optimized by the compiler.
5. *How would you implement middleware with just function types?*:
   `func(Handler) Handler`: wrap and return closures; show the chain.

## Practice Exercises

1. Implement `Retry(n int, fn func() error) error` as a closure-based
   API; add exponential backoff and make it testable without real
   sleeping (inject a clock).
2. Write `Map`, `Filter`, `Reduce` over slices using generics plus
   function values; benchmark against plain loops.
3. Write a middleware chain `A(B(C(handler)))` and test that order of
   execution is what you think; document with a sequence diagram.

## Further Reading

- [Spec: Function types](https://go.dev/ref/spec#Function_types)
- [First-class functions in Go](https://go.dev/doc/codewalk/functions/)
- [Functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
  (Dave Cheney)
