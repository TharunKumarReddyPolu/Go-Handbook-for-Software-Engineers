# Zero values

## Why Does This Matter?

Every variable in Go has a value the moment it exists. Not "undefined,"
not garbage in safe code, not null-by-default: a *specific* zero value the
language guarantees. This one decision ripples through the entire
ecosystem — it is why `var b bytes.Buffer` just works, why you do not
write constructors for every struct, and why "nil pointer" is the most
common panic in production Go. Engineers who internalize zero values
design better APIs; engineers who do not write initialization ceremony
that Go never needed.

## Mental Model

When memory is allocated for a variable and no explicit value is given,
Go sets every field/byte to zero:

| Type | Zero value |
|---|---|
| `int, int8..64, uint...` | `0` |
| `float32, float64` | `0.0` |
| `bool` | `false` |
| `string` | `""` (len 0, valid) |
| `complex64/128` | `(0+0i)` |
| pointer, func, interface, map, slice, channel | `nil` |
| array | each element's zero value |
| struct | each field's zero value |

The interesting cases are the last three rows. `nil` means "no backing
storage yet." For reference-like types, the zero value is *usable* in some
operations and panics in others:

| Zero type | `len()` | read | write | iterate |
|---|---|---|---|---|
| `nil` slice | ✓ (0) | ✓ | ✗ panic | ✓ (0 iters) |
| `nil` map | ✓ (0) | ✓ (zero value returned) | ✗ panic | ✓ (0 iters) |
| `nil` channel | blocks forever | blocks forever | blocks forever | blocks forever |
| `nil` pointer | n/a | deref panics | deref panics | n/a |
| `nil` interface | n/a | method call panics | n/a | n/a |

Memorize this table. It is the difference between "works" and "3 a.m.
panic."

## How It Works

Zeroing is not a constructor convention; it is a memory-model guarantee.
Two consequences:

1. **Allocation is safe by default.** `new(T)`, `&T{}`, `make`, a fresh
   goroutine's stack frame — all zero memory first. C++ classes teach you
   to fear uninitialized members; Go eliminates the fear by paying a
   zeroing cost.
2. **Types should be designed to be *useful* at zero.** The standard
   library is the proof:
   - `bytes.Buffer{}` — ready to write, no constructor.
   - `sync.Mutex{}` — ready to lock.
   - `sync.WaitGroup{}` — ready to Add.
   - `time.Time{}` — the specific instant January 1, year 1 — a defined,
     comparable value.

## Basic Example

```go
// examples/zero/main.go
package main

import "fmt"

type Config struct {
	Retries int           // 0 means "no retries" — or does it?
	Timeout time.Duration // 0 means "no timeout" — a footgun!
	TLS     bool
	Host    string
}

func main() {
	var c Config
	fmt.Printf("%+v\n", c)
	// {Retries:0 Timeout:0s TLS:false Host:}
	//
	// A zero Config is a structurally valid program input that is
	// semantically dangerous: zero timeout often means "wait forever"
	// in client libraries.
}
```

## Real-World Example

A zero-value-friendly API design:

```go
// Retry policy usable at zero: zero means "no retries", explicitly.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// WithDefaults returns the policy to use when the caller left it zero.
// Making the zero→default transition explicit and visible beats
// silently patching it inside Send().
func (p RetryPolicy) WithDefaults() RetryPolicy {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.Backoff <= 0 {
		p.Backoff = 100 * time.Millisecond
	}
	return p
}
```

Contrast with the footgun pattern (seen in many libraries):

```go
// MISTAKE: zero Timeout silently meaning "no deadline".
client := &http.Client{} // Timeout 0 = never times out in net/http!
```

`http.Client{}` is safe *structurally* but dangerous *semantically*: a
zero `Timeout` disables the deadline. Production code always sets a
timeout or uses context deadlines. This is the most important zero-value
footgun in the standard library.

## Production Example

Configuration systems live or die by zero-value semantics:

```go
type ServerConfig struct {
	Port            int           // zero port is invalid → validate
	ReadTimeout     time.Duration // zero = infinite → validate or default
	MaxHeaderBytes  int           // zero = use library default → fine
	GracefulPeriod  time.Duration
}

func (c ServerConfig) Validate() error {
	var errs []error
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d out of range", c.Port))
	}
	if c.ReadTimeout <= 0 {
		errs = append(errs, errors.New("read_timeout must be set; zero would disable it"))
	}
	return errors.Join(errs...)
}
```

The design rule: **decide what zero means for every field you expose.**
Three legitimate answers — "invalid, validate it," "default it
explicitly," "a meaningful zero (off/false/empty)" — and one illegitimate
one: "we never thought about it."

## Common Mistakes

- **Assuming a zero struct is a valid config.** See `http.Client{}` above;
  validate at the boundary.
- **`var m map[string]int` then `m[k] = v` → panic.** Declare with
  `make` or a literal. Reads on nil maps are fine; writes are not.
- **Checking `== nil` on an interface that holds a typed nil.** Covered in
  depth in section 02; the one-liner: an interface is nil only when both
  its type and value are nil.
- **Relying on zero values for security-sensitive state.** A zero
  `[]byte` key is empty, not random; a zero `time.Time` may pass
  "set?" checks unintentionally.
- **`new(T)` vs `&T{}` confusion.** They are equivalent for structs;
  `&T{field: v}` is idiomatic when initializing.

## Idiomatic Go

```go
buf := new(bytes.Buffer)     // fine, but...
var buf bytes.Buffer         // ...this is the Go-native way: no pointer,
                             // zero value ready, no allocation question.

var mu sync.Mutex            // zero value ready to Lock — no init.

counters := map[string]int{} // explicit empty, non-nil map when you will
                             // write soon but not necessarily now.
```

## Performance Considerations

- Zeroing is a real cost for very large allocations; the runtime may
  return zeroed pages from the OS for free, so often it is not.
- Escape analysis interacts with zero values: values that stay on the
  stack still get zeroed; see [09-memory-runtime](../09-memory-runtime/).
- Do not add constructors that only set fields to defaults — expose a
  `WithDefaults()` method (or document zero semantics) and let callers
  compose.

## Concurrency Considerations

Zero values make it safe to *create* shared state before coordination
exists: a zero mutex protects nothing yet but is ready. The dangerous
zero is the nil channel: sending/receiving on it blocks forever — the
basis of both the "nil channel disables a select case" pattern and many
deadlocks. See [08-concurrency](../08-concurrency/).

## Security Considerations

- Zeroing secrets: wiping a byte slice after use is best-effort in Go
  (GC may have copied it); prefer keeping secrets in memory only as long
  as needed and rely on well-vetted libs (e.g., `crypto/subtle`) rather
  than manual zeroing for security guarantees.
- A zero-valued token/key passed validation in a real incident pattern:
  "empty string" is a *value*, so "value present" checks must be explicit.

## Testing Strategy

Test the zero value explicitly:

```go
func TestRetryPolicy_ZeroIsNoRetry(t *testing.T) {
	got := RetryPolicy{}.WithDefaults()
	if got.MaxAttempts != 1 {
		t.Errorf("zero policy MaxAttempts = %d, want 1", got.MaxAttempts)
	}
}
```

If zero semantics of a type are load-bearing, they are spec — test them
like it.

## Interview Questions

1. *What are the zero values of slice, map, and channel, and what
   operations do they support?* — The table above; this is asked constantly.
2. *Why does Go zero memory instead of leaving it uninitialized?* —
   Determinism, safety, and no UB class of bugs; cost paid at allocation.
3. *Design a config type whose zero value is safe.* — The answer should
   mention: either make zero valid (off/disabled) or require explicit
   defaults via `WithDefaults()`/validation, and cite `http.Client{}`
   Timeout as the cautionary tale.

## Practice Exercises

1. Print zero values for `chan int`, `[]int`, `map[string]int`,
   `func()`, and `interface{}` with `%#v`. Explain each.
2. Write `func (c Config) Sanitize() Config` that turns a zero `Timeout`
   into 5s and zero `Retries` into 3, and a test proving it.
3. Deliberately write to a nil map, catch the panic with recover, and
   print the panic value's type. (Then never write to a nil map again.)

## Further Reading

- [The Go spec: the zero value](https://go.dev/ref/spec#The_zero_value)
- [Effective Go: allocation with new](https://go.dev/doc/effective_go#allocation_new)
- [net/http Client docs](https://pkg.go.dev/net/http#Client) — read the Timeout paragraph
