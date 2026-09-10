# Defer, panic & recover

## Why Does This Matter?

Go splits cleanup from teardown: `defer` schedules work for function exit,
`panic` aborts the normal flow, `recover` catches panics. Most engineers
learn defer and never learn the contract around panic — which is exactly
how panics end up crossing API boundaries or, worse, killing servers
silently. This chapter covers the mechanics and, more importantly, the
production rules.

## Mental Model

```text
defer  = "run this when the function exits, however it exits"
panic  = "this function cannot continue; unwind the stack"
recover = "only meaningful inside a deferred function; stops the unwind"
```

- `defer` is scoped to the *function*, not the block. A defer inside an
  `if` still runs at function exit.
- Deferred calls run LIFO: last registered runs first. Cleanup mirrors
  setup: close what was opened most recently first.
- Arguments are evaluated *at defer time*, not at run time (except
  method receivers evaluate then too — capture matters).
- `panic` runs defers while unwinding; `recover` only works directly in a
  deferred function.

## Defer: the workhorse

```go
func handle(conn net.Conn) error {
	defer conn.Close()          // runs on every exit path

	mu.Lock()
	defer mu.Unlock()           // LIFO with the next defer: unlock before close

	f, err := os.Open(path)
	if err != nil {
		return err               // conn.Close and mu.Unlock both run
	}
	defer f.Close()
	...
}
```

Costs and caveats:

- A defer is cheap since Go 1.14 (open-coded defers) — ~ns when the
  function does not loop. In a tight loop over thousands of iterations,
  move it into a helper function instead.
- Deferred *arguments* are evaluated immediately:

```go
i := 0
defer fmt.Println("i =", i) // prints i = 0 — arg evaluated now
i = 42
```

To defer a *current* value, wrap in a closure: `defer func() {
fmt.Println(i) }()`.

- `defer os.File.Close()` ignoring the error is acceptable in short
  scripts; in production code where the close matters (writes), capture
  it: `defer func() { cerr = f.Close() }()` and merge into the returned
  error.

## panic: the contract

Panics are for **programmer errors and unrecoverable invariant
violations**, not for control flow:

| Use panic | Use errors |
|---|---|
| nil map write you caused via bug | file not found |
| index out of range in internal code | user input invalid |
| impossible enum value reached | downstream dependency failed |
| library API misused (flagrant) | expected failure mode |

The rules that keep production sane:

1. **Library code returns errors; it does not panic.** The caller cannot
   defend against what it cannot see.
2. **Package-level init can panic** (e.g., `regexp.MustCompile`):
   impossible-to-continue startup failures are the accepted exception —
   fail fast at process start.
3. **Never panic across API boundaries**; convert at the edge.

## recover: the one legitimate pattern

```go
func safeHandler(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log the stack, return 500. Do NOT swallow silently.
				log.Printf("panic serving %s: %v\n%s", r.RemoteAddr, rec, debug.Stack())
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	})
}
```

Note `net/http` already does this per-request — the example shows the
pattern for your own goroutine boundaries. The rule: **each goroutine you
start owns its panics.** A panic in a goroutine kills the process; if you
spawn workers that call third-party code, wrap them.

```go
go func() {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("worker panic: %v\n%s", rec, debug.Stack())
		}
	}()
	worker(ctx)
}()
```

What recover cannot do: repair state. If your function panicked halfway
through mutating a shared structure, catching the panic leaves it
corrupted. When in doubt, crash: restart from a known-good state.

## Common Mistakes

- **Using panic for validation errors.** Users send bad input; that is an
  error value, not an abort.
- **recover() outside a deferred call** — always nil.
- **Swallowing panics silently** (`recover()` with no log): hides bugs
  behind mysterious behavior. Log with stack, always.
- **Assuming `os.Exit` runs defers.** It does not — `os.Exit` terminates
  immediately. Buffer flushing, unlock, close: all skipped. The fix:
  return through main instead.
- **Defers in loops** with large N: allocation + growth per iteration;
  hoist into a helper function so the defer scope is small.

## Idiomatic Go

```go
// Lock immediately, defer unlock on the next line — never in between.
mu.Lock()
defer mu.Unlock()

// Open/defer-close pairs adjacent, so the reader sees the pairing.
f, err := os.Open(p)
if err != nil { return err }
defer f.Close()
```

## Performance Considerations

- Open-coded defers (Go 1.14+) made the common case nearly free; a defer
  in a function called per-iteration is still fine.
- In tight loops processing millions of items, measure: extract the loop
  body into a function so defers run per item at near-zero cost instead
  of accumulating.
- Panics are expensive (stack unwinding, formatting); they are a crash
  path, not an error channel.

## Concurrency Considerations

- A panic in any goroutine crashes the process — `recover` does not cross
  goroutine boundaries. Spawned workers need their own recover.
- `defer mu.Unlock()` prevents the classic "forgot unlock on error path"
  deadlock. For conditional unlocks, prefer restructuring over
  `mu.Unlock()` in two branches.
- `recover` + shared mutable state = corruption risk; prefer crashing.

## Security Considerations

- A panic message with sensitive data (tokens in URLs, PII) that reaches
  logs is a leak; sanitize what you log in the recover path.
- Recovering all panics in request handlers prevents DoS-by-panic but
  also hides bugs — the mitigations are stack-trace logging and
  alerting on panic-rate metrics (see [20-observability](../20-observability/)).

## Testing Strategy

```go
func TestParsePanicsOnNil(t *testing.T) {
	defer func() {
		if rec := recover(); rec == nil {
			t.Fatal("expected panic on nil input")
		}
	}()
	Parse(nil) // must panic: programmer error by contract
}
```

Test that panic contracts hold for exported functions where panic is the
documented behavior (e.g., must-style constructors).

## Interview Questions

1. *When do deferred functions run, and in what order?* — Function exit,
   LIFO; args evaluated at defer time.
2. *Does os.Exit run defers?* — No. Consequence for flushing logs and
   buffers; fix by returning through main.
3. *When is panic appropriate in library code?* — Almost never; the
   accepted exceptions are must-style init and flagrant API misuse. Say
   "errors are for expected failures, panics for invariant violations."
4. *A goroutine panics — what happens?* — Process crash; recover must be
   in the same goroutine; wrap worker bodies.

## Practice Exercises

1. Write a function with three defers printing A, B, C; predict output,
   run, explain the LIFO order.
2. Write `func safeDiv(a, b int) (n int, err error)` that recovers a
   divide-by-zero panic *and* explain why returning an error is better
   than panicking in the first place.
3. Spawn a goroutine that panics; observe the process crash. Add a
   recover wrapper; observe the process survive. This one experiment
   teaches the whole chapter.

## Further Reading

- [Defer, Panic, and Recover](https://go.dev/blog/defer-panic-and-recover) — official blog
- [The Go spec: handling panics](https://go.dev/ref/spec#Handling_panics)
- [Defer is getting faster](https://go.dev/blog/go1.14-defers) — Go 1.14 open-coded defers
