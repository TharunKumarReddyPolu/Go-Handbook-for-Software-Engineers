# Fundamentals

## Why Does This Matter?

Testing in Go is not a framework skill; it is a design skill. The
conventions are fixed (a `Test` prefix, the `*testing.T`, files ending
`_test.go`), so the craft is everything else: what you assert, how tests
fail, and how they age. Good Go tests read like specifications; bad ones
read like archaeology after the first refactor.

## Mental Model

```text
func TestXxx(t *testing.T)   // unit: fast, isolated, deterministic
func BenchmarkXxx(b *testing.B)  // performance: run with -bench
func FuzzXxx(f *testing.F)       // robustness: run with -fuzz
func ExampleXxx()                // documentation that is compiled and verified
```

One rule runs the whole section: **a test's value is the precision of
its failure message.** `t.Errorf("got %v, want %v", got, want)` is a
specification; `t.Error("wrong")` is noise.

## Table-driven tests: the workhorse

```go
func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"positives", 2, 3, 5},
		{"negatives", -2, -3, -5},
		{"zero identity", 0, 7, 7},
		{"overflow", math.MaxInt, 1, math.MinInt}, // documents intent!
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Add(tt.a, tt.b); got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
```

Why this shape won the ecosystem:

- Adding a case is one line; the test grows with the spec.
- `t.Run` names each case, so failures name themselves:
  `TestAdd/overflow`.
- Run a single case: `go test -run 'TestAdd/overflow' ./...`: debugging
  without re-running the world.
- The table *documents* behavior, including edge cases you'd otherwise
  bury in a comment.

When NOT to table-drive: one or two cases with different setup flows,
forcing heterogeneous setups into one table makes the test harder to
read, not easier.

## Subtests: structure and scoping

`t.Run` does more than name cases:

```go
func TestStore(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t) // helper with t.Cleanup inside

	t.Run("save and load", func(t *testing.T) { ... })
	t.Run("missing key", func(t *testing.T) { ... })
	t.Run("concurrent access", func(t *testing.T) {
		t.Parallel() // this subtest runs concurrently with siblings
		...
	})
}
```

- `t.Parallel()` marks tests safe to run concurrently; use it after
  tests are genuinely independent, and let `-race` verify the claim.
- Subtests inherit the parent's scope: resources created before
  `t.Run` are shared, which is exactly what you want for per-suite
  fixtures.
- Parallel subtests capture loop variables: pass them in
  `t.Run(name, func(t *testing.T) { ... })` closures: safe since Go
  1.22 loop-var changes, but explicit capture is still the clearest
  form.

## Helpers, and the t.Helper() contract

```go
// mustUser builds a valid User or fails the test. The t.Helper()
// call makes failure reports point at the caller's line, not here.
func mustUser(t *testing.T, name string) User {
	t.Helper()
	u, err := NewUser(name)
	if err != nil {
		t.Fatalf("NewUser(%q): %v", name, err)
	}
	return u
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st := NewStore()
	t.Cleanup(func() { st.Close() }) // runs at test end, LIFO
	return st
}
```

Three contracts every helper should honor:

1. `t.Helper()`: failures report the real call site.
2. First parameter `*testing.T` (or `*testing.B`): enables the above
   and keeps helpers discoverable.
3. Register cleanup with `t.Cleanup`, not manual deferred calls in every
   test: the helper owns its resources' lifecycle.

`t.Cleanup` beats `defer` for helper-created resources: it runs when the
*test* ends (including after parallel subtests finish), not when the
helper returns.

## Assertions: what stdlib gives you and what it doesn't

Go's stdlib has no assertion library by design; you write `if` + `t.*`:

| Need | Form |
|---|---|
| Fail now (can't continue) | `t.Fatalf` |
| Fail, continue (collect more failures) | `t.Errorf` |
| Skip with reason | `t.Skipf` |
| Log without failing | `t.Logf` |

Compare whole structures with `reflect.DeepEqual` or (better, for
results) `diff`-style output; compare with tolerance for floats using
`math.Abs(got-want) < eps`. If the team wants richer diffs, the common
choice is `google/go-cmp`: one dependency, used only in tests.

## Test organization

- Same package (`calc_test.go` in `package calc`): white-box, tests can
  touch internals. Default.
- External package (`package calc_test`): black-box, forces you through
  the public API. Use for API packages and examples.
- Both in one package is normal: `calc_test.go` (internal) +
  `calc_public_test.go` (external).

## Basic Example: the full pattern

```go
// calc.go
package calc

import "errors"

var ErrDivideByZero = errors.New("divide by zero")

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }

// Divide returns a/b, or ErrDivideByZero.
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrDivideByZero
	}
	return a / b, nil
}
```

```go
// calc_test.go
package calc

import (
	"errors"
	"math"
	"testing"
)

func TestDivide(t *testing.T) {
	tests := []struct {
		name    string
		a, b    int
		want    int
		wantErr error
	}{
		{"exact", 10, 2, 5, nil},
		{"truncates toward zero", -7, 2, -3, nil},
		{"by zero", 1, 0, 0, ErrDivideByZero},
		{"min int by -1", math.MinInt, -1, math.MinInt, nil}, // Go wraps
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Divide(tt.a, tt.b)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Divide(%d, %d) error = %v, want %v", tt.a, tt.b, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Divide(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
```

Note `errors.Is` for error comparison: sentinel errors may be wrapped;
`==` breaks under wrapping (see [05-errors](../05-errors/)).

## Real-World Example: golden files for structured output

When output is large and structured (rendered reports, JSON), golden
files pin it:

```go
func TestReport(t *testing.T) {
	got := renderReport(sampleData())
	golden := filepath.Join("testdata", "report.golden")
	if *update {
		os.WriteFile(golden, []byte(got), 0o644) // go test -update
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("report mismatch:\n got: %s\nwant: %s", got, want)
	}
}
```

The `-update` flag pattern makes intentional changes cheap while
defaulting to strict verification. Keep goldens small; a 10k-line golden
file is a diff nobody reads.

## Common Mistakes

- **Asserting on unrelated implementation details** (internal field
  values): tests should verify observable behavior, so refactors don't
  mass-fail.
- **One giant test function** with sequential steps: first failure
  hides the rest; use subtests.
- **`t.Fatal` in goroutines**: undefined behavior; only the test
  goroutine may call Fatal/Skip. Signal via channel and fail from the
  test goroutine.
- **Reusing table entries across test functions**: tables belong to
  one test; sharing couples unrelated specs.
- **Ignoring the error return in test setup**: `mustUser(t, ...)`
  patterns exist precisely so setup failures fail loudly.
- **Testing time by sleeping**: inject clocks or shrink durations;
  sleeps make suites slow and flaky (see the concurrency section's
  gated-channel pattern).

## Idiomatic Go

- Name tests by behavior: `TestDivide_TruncatesTowardZero`, not
  `Test1`.
- Want-error columns carry the sentinel, not a bool: the test then
  verifies *which* error.
- Helpers return values, never mutate global state; globals make
  parallel tests lie.

## Performance Considerations

- Unit tests should be fast enough to run on save; keep heavy cases in
  the integration tier.
- `testing.Short()` gates long-running tests: `if testing.Short() {
  t.Skip("slow") }`: pair with `go test -short` in fast loops.
- Suite-wide: cached results (`go test` caches passing packages) keep
  the edit-test loop instant; tests that read the clock/network/paths
  defeat the cache: mark them with `t.Setenv` (auto-disables caching
  for that test) or keep them behind tags.

## Concurrency Considerations

- `t.Parallel()` on independent tests; `-race` in CI always. The
  handbook's concurrency section shows the full discipline.
- Helpers that spawn goroutines must register cleanup that joins them,
  leaked test goroutines poison the next test's goroutine count.

## Security Considerations

- Never commit real credentials/tokens as fixtures; use obviously-fake
  values (`test-api-key-do-not-use`), and structure code so tests
  cannot accidentally run against production endpoints.
- Golden files and fixtures can leak PII from recorded sessions; scrub
  at capture time.

## Testing Strategy

Tests themselves follow the same rules: deterministic, isolated,
failure-precise. CI (this repo's
[.github/workflows/ci.yml](../.github/workflows/ci.yml)) runs
`go test ./... -race -shuffle=on`: shuffled order catches hidden
inter-test dependencies, race detection catches the rest.

## Interview Questions

1. *Why does Go have no assertion library?*: Control flow stays in the
   language (`if`, `return`); no macro magic; failures are plain
   function calls. Teams add go-cmp for diffs when needed.
2. *Design tests for a function that parses cron expressions.*: Table
   with valid/invalid/edge rows; want-value columns; fuzz for the
   long tail (see ch. 04).
3. *Your test suite takes 8 minutes; how do you cut it?*: Profile the
   suite: parallelize independents, move infra tests to the integration
   tier, replace sleeps with synchronization, cache-friendly hygiene.

## Practice Exercises

1. Extend the calculator's table with the four integer-division edge
   cases (min-int, by -1, by zero, truncation signs): then break the
   implementation and verify each case fails with a precise message.
2. Write `mustParse(t, expr string) Node` and refactor three tests to
   use it; observe the failure-line improvement.
3. Add `t.Parallel()` to a suite, run with `-race`, and fix what falls
   out. The fallout *is* the lesson.

## Further Reading

- [testing package docs](https://pkg.go.dev/testing): the contract
- [Table driven tests](https://go.dev/wiki/TableDrivenTests): official wiki
- [t.Cleanup proposal](https://go.dev/issue/37700): rationale for cleanup ordering
