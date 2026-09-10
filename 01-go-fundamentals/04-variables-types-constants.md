# Variables, types & constants

## Why Does This Matter?

Declaration, inference, and conversion look like trivia until they interact
with API design: every signature you write forces a choice between `var`,
`:=`, literals, and named types. This chapter covers the mechanics *and*
the judgment.

## Mental Model

Go is statically typed with a strict, explicit conversion rule:

> A value of type T is only assignable to U if they are identical, or a
> rule says so (interface satisfaction, unnamed-type identity, constants).

There is no implicit numeric conversion anywhere — not even int→int64.
That friction is deliberate: it makes every narrowing explicit and
reviewable.

## Declaration: the three forms

```go
// 1. var — full form: declare with type and/or initializer
var port int          // zero value 0
var host = "localhost" // type inferred
var timeout time.Duration = 5 * time.Second // explicit when inference loses clarity

// 2. short declaration — inside functions only
name := "handbook"
i, j := 0, 1

// 3. grouped — package-level constants and vars
const (
	StatusOK = 200
	StatusCreated = 201
)
```

Rules of thumb:

- **Package level: `var`/`const`.** `:=` is illegal there.
- **Inside functions: `:=`** unless you need the zero value first
   (`var buf bytes.Buffer`) or a specific type (`var r io.Reader = f`).
- **Group related declarations.** Parenthesized blocks read as a table.

## Basic types

| Category | Types | Zero value |
|---|---|---|
| Booleans | `bool` | `false` |
| Integers (signed) | `int, int8..64` | `0` |
| Integers (unsigned) | `uint, uint8..64, uintptr` | `0` |
| Floating point | `float32, float64` | `0.0` |
| Complex | `complex64, complex128` | `(0+0i)` |
| String | `string` | `""` |
| Runes | `rune` (= int32) | `0` |
| Bytes | `byte` (= uint8) | `0` |
| Error | `error` | `nil` |

Defaults matter: use `int` unless a format demands otherwise; `float64`
unless memory-bound. `int` is 64-bit on all platforms Go supports today
but treat it as "natural word size."

## Type inference

`:=` infers from the right side, and inference is *exact*, not "whatever
fits":

```go
x := 10        // int, not int32, not float64
y := x + 1     // int
z := 1.5       // float64
d := time.Second // time.Duration
```

Untyped constants are the exception — they carry a default type only when
forced:

```go
const big = 1 << 40    // untyped integer constant; arbitrary precision
var f float64 = big    // fine: constant adapts
var i int = big        // also fine
```

## Conversions

Always explicit, always visible:

```go
seconds := 90
mins := seconds / 60        // integer division: 1 — classic bug
ratio := float64(seconds) / 60.0 // 1.5

var u uint8 = 255
u++        // wraps to 0 — unsigned overflow, no panic, no warning
```

Conversions between named types with the same underlying type are legal
and common:

```go
type UserID int64
type OrderID int64

func (id UserID) String() string { return fmt.Sprintf("user-%d", int64(id)) }

var uid UserID = 42
// oid := uid            // compile error — different named types
oid := OrderID(uid)      // legal, but smells: IDs should not be interchangeable
```

That last line compiles — the compiler cannot stop you from confusing a
user ID with an order ID. Named types create vocabulary, not safety;
conversion discipline is yours.

## Constants: compile-time values

```go
type ByteSize int64

const (
	KB ByteSize = 1 << (10 * (iota + 1))
	MB
	GB
)

const (
	Read = 1 << iota  // 1
	Write             // 2
	Exec              // 4
)
```

`iota` is the classic Go enumerator: it counts const blocks, and each
`const` line repeats the previous expression. Constants may be typed or
untyped, and untyped constants have arbitrary precision until they meet a
type — that is why `math.MaxInt64` works in float expressions and why
bit flags above int32 compile on 32-bit platforms.

When to use a constant vs a var: if the value is known at compile time
and must never change, constant. Config that varies per environment is a
var (or better, injected — see [14-backend-development](../14-backend-development/)).

## Operators worth knowing

- `/` on integers truncates; use float conversion when you mean it.
- `%` follows the sign of the dividend (`-7 % 3 == -1`) — different from
  Python. For "always positive modulo": `((x % m) + m) % m`.
- `&^` (AND NOT) is Go's bit-clear: `flags &^ Write`.
- Strings compare lexicographically by bytes; for Unicode-aware
  comparison use collation packages.
- No ternary. Write the if. This is a style decision, not an omission.

## Common Mistakes

```go
// MISTAKE: integer division losing the fraction
avg := total / count // if both int, truncates

// MISTAKE: mixing types
var a int32 = 1
// b := a + 1        // fine; b is int32
// c := a + int64(1) // fine — but stop and ask why you are mixing widths

// MISTAKE: shadowing in a new scope
err := doThing()
if ok {
	err := doOther() // NEW err — outer one never checked
	_ = err
}
// check outer err here... but it was silently fine above
```

Shadowing (`:=` inside a block redeclaring an outer name) is the single
most common beginner bug in real PRs. `go vet -shadow`-style linters and
reviewers catch it; your eyes should too.

## Idiomatic Go

```go
func parseInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", s, err)
	}
	return n, nil
}
```

- Infer with `:=` in functions; be explicit when the type is the point
  (`var r io.Reader`).
- Return zero value + error, never a pointer + nil error on failure.
- Named result parameters only when they clarify docs, not to skip returns.

## Performance Considerations

- Small values in variables stay in registers; pointers force heap
  questions. Prefer values for small structs (see
  [09-memory-runtime](../09-memory-runtime/)).
- Constants are free: no memory, folded at compile time. Prefer them for
  magic numbers even once.
- Conversions between numeric types cost nothing meaningful; conversions
  between string/[]byte copy — that one matters (see section 02).

## Concurrency Considerations

Variables are per-goroutine until shared. Sharing without synchronization
is a data race; the rules and fixes are the whole of
[08-concurrency](../08-concurrency/). Preview: a `bool` flag shared across
goroutines is a race, not a pattern.

## Security Considerations

- Unsigned wraparound is silent; validate inputs *before* arithmetic on
  sizes (an attacker-controlled `uint` count can wrap to 0 and pass a
  bounds check).
- Floats for money is a defect, not a style issue —
  [25-fintech-with-go](../25-fintech-with-go/) uses integer minor units.

## Testing Strategy

Table-driven tests exercise conversions at boundaries: 0, 1, max, -1,
overflow. You will meet the pattern properly in [10-testing](../10-testing/);
here is a preview with a real package:

```go
func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"negative", "-3", -3, false},
		{"empty", "", 0, true},
		{"overflow", "99999999999999999999", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseInt(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
```

## Interview Questions

1. *What is the zero value and why does it matter for API design?* — See
   next chapter; the answer should mention usable-by-default types.
2. *Why no implicit conversions?* — Explicit narrowing; reviewable loss of
   precision; fewer surprise bugs at scale.
3. *What is `iota`?* — Counter within a const block; repeated expressions;
   the idiom for bit flags and sized enums.
4. *int vs int64 — when do you care?* — Serialization formats, wire
   protocols, FFI; `int` maps to platform word size.

## Practice Exercises

1. Predict, then verify: `var x float32 = 0.1; x == 0.1` with a float64
   comparison. Explain the result.
2. Write `Even(n int) bool` without `%` (hint: `&1`). Benchmark both — is
   the difference measurable?
3. Find one shadowing bug in your own old code with
   `go vet -vettool`-style tooling or by eye; fix it with `:=` → `=`.

## Further Reading

- [The Go spec: constants](https://go.dev/ref/spec#Constants) — precise and readable
- [Go FAQ: unused variables](https://go.dev/doc/faq#unused_variables)
- [Go Slices usage and internals](https://go.dev/blog/slices-intro) — for when types become collections (section 02)
