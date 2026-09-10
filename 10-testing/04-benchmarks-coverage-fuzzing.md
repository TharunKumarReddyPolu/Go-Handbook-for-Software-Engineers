# Benchmarks, coverage, fuzzing

## Why Does This Matter?

Three tools answer three different questions: benchmarks answer "how
fast," coverage answers "what did the tests touch," and fuzzing answers
"what input breaks this." Confusing them produces vanity metrics —
coverage numbers that prove nothing and benchmarks that measure the
wrong thing. This chapter is the measurement toolkit with the judgment.

## Mental Model

```text
go test -bench=.      → testing.B loop, ns/op + allocations
go test -cover        → % of statements executed by tests
go test -fuzz=FuzzXxx → randomized inputs hunting for panics/hangs
```

They compose with the ordinary flags: `-benchmem`, `-race`, `-cpuprofile`.

## Benchmarks

```go
func BenchmarkFib(b *testing.B) {
	for b.Loop() { // Go 1.24+: per-iteration state without b.N gymnastics
		Fib(20)
	}
}

func BenchmarkParse(b *testing.B) {
	input := buildLargeInput()
	b.ReportAllocs() // allocations/op: the first suspect in Go perf work
	b.ResetTimer()   // after setup costs
	for i := 0; i < b.N; i++ {
		if _, err := Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}
```

Reading the output:

```text
BenchmarkParse-8   2451231   489.2 ns/op   512 B/op   6 allocs/op
            ^cores  ^iterations ^ns/op       ^bytes     ^allocations per op
```

The number that matters is usually **allocs/op**: in Go, allocation rate
drives GC pressure, and GC pressure drives p99 latency. See
[19-performance](../19-performance/) for the full discipline.

Rules that keep benchmarks honest:

- **Compiler elision**: if the result is unused, the compiler may delete
  the work. Assign to a package-level sink.
- **BenchTime**: `go test -bench=. -benchtime=2s` for noisy functions;
  default 1s gives ±few% noise.
- **benchstat** is mandatory for comparisons: `benchstat old.txt
  new.txt` — never compare two raw runs by eyeball.
- **b.Run for variants**:
  ```go
  func BenchmarkParseSizes(b *testing.B) {
	  for _, size := range []int{10, 100, 1000} {
		  b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) { ... })
	  }
  }
  ```
- **b.RunParallel** models concurrent access (scheduler/contention
  shapes): see the sync chapter's shard benchmarks.

## Coverage — honest use

```bash
go test ./... -cover
go test ./... -coverprofile=cover.out && go tool cover -html=cover.out
go tool cover -func=cover.out | tail -1   # total
```

Coverage tells you what the tests *executed*, not what they *verified*.
The two failure modes:

1. **High coverage, weak assertions** — the code ran, nothing was
   checked. Coverage red, confidence green, bugs shipped.
2. **100% on trivial code, 20% on the parser** — the average lies; read
   `-func` output per critical function.

Sane policy: cover the money paths (auth, payments, state machines)
thoroughly; don't chase a repo-wide number. This handbook's examples
hold near-total coverage on small, deliberate packages — a choice, not a
template.

## Fuzzing — let the machine find the inputs you didn't

```go
func FuzzParseDuration(f *testing.F) {
	// Seed corpus: known-interesting inputs, from examples and past bugs.
	f.Add("10s")
	f.Add("-5m")
	f.Add("")
	f.Add("99999999999999999999s")

	f.Fuzz(func(t *testing.T, s string) {
		d, err := ParseDuration(s)
		if err != nil {
			return // invalid input is fine — we hunt panics and hangs
		}
		if d < 0 {
			t.Errorf("ParseDuration(%q) accepted negative duration %v", s, d)
		}
		if d > 100*365*24*time.Hour {
			t.Errorf("ParseDuration(%q) produced absurd %v", s, d)
		}
	})
}
```

Run: `go test -fuzz=FuzzParseDuration -fuzztime=30s ./...`. Failing
inputs land in `testdata/fuzz/...` and become permanent regression cases
— the corpus grows into a record of every bug found.

What fuzzing is for: parsers, decoders, deserializers, anything at a
trust boundary ([21-security](../21-security/) treats fuzzing as a
security control). What it isn't for: business-logic tests with curated
expectations — fuzz invariants ("never panics, never negative") not
exact outputs.

## Property-style testing — invariants over examples

Fuzzing checks "never bad"; property tests check "always consistent."
The Go style: loop over random-but-seeded inputs in a normal test:

```go
func TestRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) // fixed seed: reproducible
	for i := 0; i < 1000; i++ {
		orig := randomTransaction(rng)
		enc := encode(orig)
		got, err := decode(enc)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !reflect.DeepEqual(orig, got) {
			t.Fatalf("round trip changed the value:\norig: %+v\ngot:  %+v", orig, got)
		}
	}
}
```

Classic properties worth writing per domain: encode∘decode = identity;
serialization is deterministic; sorting is idempotent; idempotency keys
collapse duplicate applies. Frameworks (gopter, rapid) exist; the stdlib
loop above covers most needs and stays grep-able.

## Basic Example — a benchmark clinic

Before: a "slow" function. After: the fix + the benchmark proving it.

```go
// SLOW: concatenation in a loop is O(n²) — each += copies the string.
func JoinBad(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}

// FAST: strings.Builder amortizes to O(n) with a single backing buffer.
func JoinGood(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}
```

The repo's `10-testing/examples/calculator/bench_test.go` contains this
comparison with reported allocations; run it:

```bash
go test ./10-testing/examples/calculator -bench=BenchmarkStringJoin -benchmem
```

## Common Mistakes

- **Benchmarking uncompiled-release-like code** — benchmarks run the
  code you wrote; if the production path differs (middleware, buffers),
  the benchmark lies.
- **Ignoring benchstat** — two 5% "improvements" in a row is noise;
  statistics exist for a reason.
- **Coverage as a KPI** — see failure mode 1; pay for assertions, not
  execution.
- **Fuzzing without seeds** — the corpus starts blind; seeds encode your
  understanding of interesting inputs.
- **Property tests with random seeds not fixed** — a failure you cannot
  reproduce is a failure you cannot fix. Log or fix the seed.

## Idiomatic Go

- `b.Loop()` (Go 1.24+) for new benchmarks — clearer and prevents
  compiler-elision accidents.
- Benchmarks live in `_test.go` beside the code; run selectively —
  `go test -bench=ParseSize ./...` — never the whole suite by accident.
- Fuzz targets are named after the property they guard
  (`FuzzParseNoPanics`, `FuzzRoundTrip`).

## Performance Considerations

This chapter measures performance; the discipline that consumes the
measurements is [19-performance](../19-performance/): profile before
optimizing, optimize the allocation rate first, and re-benchmark after
every change with benchstat in hand.

## Concurrency Considerations

- `b.RunParallel` for contention questions; `-race` + benchmarks is a
  valid (slow) combination for finding races under load.
- Benchmarks that share state across iterations measure their own
  synchronization, not the code — reset state per iteration or per
  b.Run.

## Security Considerations

- Fuzzing at trust boundaries (parsing untrusted input) is defense in
  depth; wire it into CI for parsers ([21-security](../21-security/)).
- Fuzz corpora may capture sensitive inputs from past bugs; treat
  `testdata/fuzz` as potentially sensitive.

## Testing Strategy

The tiers compose: unit tests pin behavior, fuzzers hunt invalid inputs,
properties guard invariants, benchmarks track cost. A parser's complete
test story has all four; a CRUD handler usually needs only the first.

## Interview Questions

1. *Your PR claims 30% faster; prove it.* — benchstat on matched
   environments, allocation deltas, and a reproducible benchmark; the
   answer that cites noise thresholds (±2%) grades highest.
2. *What does 80% coverage mean and not mean?* — Executed vs verified;
   the follow-up: "what would 100% still miss?" (assertions,
   interleavings, real infrastructure).
3. *Design the fuzz target for a JWT parser.* — Never panic; signature
   check rejects tampered payloads; no timing-dependent acceptance;
   seeds from known CVE-ish structures.

## Practice Exercises

1. Benchmark `JoinBad` vs `JoinGood` at sizes 10/100/1000 with
   `b.Run`; explain the crossover (there isn't one — find out why).
2. Write a fuzz target for the calculator's expression parser with
   seeds for malformed input; run 60s and triage anything it finds.
3. Add a property test: `Sort(Sort(x)) == Sort(x)` and `Sort` is a
   permutation — two properties, one function; use a fixed seed.

## Further Reading

- [testing: benchmarks](https://pkg.go.dev/testing#hdr-Benchmarks) and
  [fuzzing](https://go.dev/doc/fuzz) — official
- [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) — the comparison tool
- [Go 1.24: testing.B.Loop](https://go.dev/doc/go1.24) — the new loop form
