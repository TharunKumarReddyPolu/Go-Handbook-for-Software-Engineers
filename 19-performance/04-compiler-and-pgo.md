# Compiler & PGO

## Why Does This Matter?

Before you optimize by hand, know what the compiler is already doing —
and how to teach it where your hot paths are. Inlining, bounds-check
elimination, and profile-guided optimization routinely deliver 5-30% on
real services without touching a line of logic. This chapter covers the
free wins and their diagnostics.

## Mental Model

Your code compiles through:

```text
source → AST → SSA → optimizations → machine code
                ↑         ↑
        escape analysis   inlining, bounds-check
        (ch. 02)          elimination, PGO
```

Three levers matter to application engineers: **inlining** (function
call overhead removed, further optimizations enabled), **bounds-check
elimination** (slice indexing without runtime checks), and **PGO**
(build-time profile feedback). Everything else is the compiler's job —
your job is not to defeat it.

## Inlining — the invisible enabler

```bash
go build -gcflags='-m' ./...
# ./calc.go:12:6: can inline Add with cost 12 as: func(int, int) int
# ./big.go:40:6: cannot inline BigFunc: function too complex: cost 132 exceeds budget 80
```

- Small functions (cost ≤ ~80) inline automatically. The budget is per-
  function complexity, not line count.
- Inlining isn't just call-overhead removal: it *enables* escape
  analysis and bounds-check elimination across the call boundary.
- Defeating it: `//go:noinline` (for debugging), huge functions,
  anything the cost model fears (defers used to block inlining; that
  changed — verify on your version).
- Mid-stack inlining (functions calling inlined functions) makes small
  wrapper layers nearly free — the reason idiomatic Go can afford lots
  of tiny functions.

## Bounds-check elimination (BCE)

Every slice index has a runtime check — unless the compiler proves it
redundant:

```go
func sumBad(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i] // provably in-bounds: check eliminated
	}
	return total
}

func sumWorse(s []int) int {
	total := 0
	for i := 0; i <= len(s)-1; i++ { // hmm — the compiler usually still gets this
		total += s[i]
	}
	return total
}

func sumDefeated(s []int) int {
	total := 0
	for _, i := range order(s) { // indices from a function call: opaque
		total += s[i]           // bounds check retained
	}
	return total
}
```

The proof tool: `go build -gcflags='-d=ssa/check_bce'` prints which
lines keep checks. The practical rules: derive indices from `len` in
simple patterns, hoist it into `for i := range s`, and avoid opaque
index sources on hot paths. This is a micro-optimization — the last 10%,
not the first; allocation rate still dominates (ch. 02).

## PGO — profile-guided optimization

PGO (Go 1.21+) feeds a *production CPU profile* into the build; the
compiler then inlines hot functions more aggressively, devirtualizes
interface calls where the profile shows a dominant implementation, and
lays out code by frequency.

```text
1. Collect: pprof CPU profile from production (30s+ of representative load)
2. Name:     default.pgo at the main package directory
3. Build:    go build -pgo=auto ./...   (auto = use default.pgo if present)
```

```text
service/
├── cmd/api/
│   ├── main.go
│   └── default.pgo   ← the profile lives with the binary's main package
```

Rules that keep PGO honest:

- **Representative load** — profile the traffic shape you ship for;
  profiling the wrong workload mis-optimizes.
- **Rebuild with fresh profiles** on a cadence; stale profiles calcify
  yesterday's hot paths.
- **Measure**: PGO gains are typically 2-7% CPU on real services, up to
  ~14% on inlining-heavy workloads — real but not transformative; it's
  the cheapest 5% you'll ever add.
- **Update-triggered PGO** (Go 1.23+ lets profiles be consumed
  cross-version) — the toolchain handles most of this automatically.

## Zero-cost interfaces? — the devirtualization story

Interface calls cost a pointer indirection + dynamic dispatch — until
PGO sees one concrete implementation dominating and devirtualizes the
call. Design consequence: keeping implementations narrow and stable in
hot paths lets PGO work; sprawling fan-out at the same call site caps
it.

## What the compiler can't fix

- O(n²) algorithms — no flag for that.
- Network round-trips — batching is yours (ch. 03).
- Allocation churn from your data model — ch. 02.
- Lock contention — ch. 03.

The hierarchy: algorithm > architecture > allocation > compiler. PGO
and BCE polish the top of a pyramid you must build right first.

## Basic Example — the free-win checklist

```bash
# 1. Is it inlining?
go build -gcflags='-m' ./... | grep 'can inline'

# 2. Are bounds checks eliminated on the hot loop?
go build -gcflags='-d=ssa/check_bce' ./... 2>&1 | grep hotfile.go

# 3. What does the profile say BEFORE and AFTER enabling PGO?
go test -bench=. -count=10 > nopgo.txt
go build -pgo=auto -o api ./cmd/api && load-test && capture profile → default.pgo
go test -bench=. -count=10 > pgo.txt
benchstat nopgo.txt pgo.txt
```

## Common Mistakes

- **Committing a stale `default.pgo`** from a demo load — the build then
  optimizes for traffic that no longer exists; PGO with wrong profiles
  can *regress*.
- **Micro-optimizing BCE on cold paths** — the check is nanoseconds;
  the readability cost is forever.
- **Assuming `unsafe` beats the compiler** — `unsafe` defeats
  optimizations as often as it enables them; measure, and read ch. 02's
  safety tax.
- **Believing -O folklore from C** — Go has no -O flags; tuning comes
  from profiles, not build flags.

## Idiomatic Go

- Small functions everywhere — the inliner makes them free and the code
  readable; do not pre-merge "for performance" without proof.
- Ship `default.pgo` from the previous release's production profile as
  part of the release pipeline ([22-production-go](../22-production-go/)
  covers the CI wiring).

## Performance Considerations

The self-referential loop closes: PGO profiles come from pprof (ch. 01),
validate with benchstat, and interact with GC/scheduler behavior (ch.
02/03). One release's PGO profile + one benchstat table = the whole
story.

## Concurrency Considerations

PGO optimizes code layout, not synchronization — contention taxes
(ch. 03) are untouched by any compiler flag. The runtime's scheduler
also uses PGO-adjacent hints nothing you control; never tune around
scheduler internals that may change per release.

## Security Considerations

- Profiles are reconnaissance data: function names, hot paths, code
  layout. Treat `default.pgo` and captured profiles as internal
  artifacts; don't ship them in public images ([21-security](../21-security/)).
- Devirtualization can change which code paths exist in a binary —
  security scanning (govulncheck) runs on the *built* configuration, so
  keep CI building exactly what you deploy (same flags, same PGO).

## Testing Strategy

- Benchmarks run with and without PGO in the nightly job; benchstat the
  delta and record it in the release notes.
- `check_bce` greps as opt-in tests on named hot files (ch. 02's escape
  regression tests, sibling technique).

## Interview Questions

1. *What does PGO actually change in the binary?* — Aggressive inlining
   on hot paths, block layout by frequency, devirtualization; measured
   gains and how to verify.
2. *Why are small functions idiomatic in Go if calls cost time?* —
   Mid-stack inlining makes them free and enables cross-call
   optimizations; readability wins because the compiler removes the
   cost.
3. *Where does the compiler NOT help?* — Algorithms, allocation models,
   contention, I/O shapes — the hierarchy answer.

## Practice Exercises

1. Find one function in a dependency-heavy package that fails to inline;
   shrink it until it inlines; benchstat the hot path that calls it.
2. Run `check_bce` on a slice-heavy hot loop; restructure until the
   checks vanish; report whether the benchmark moved.
3. Ship a PGO build of one service to staging; measure p50/p95/p99
   under identical load for 24h with and without — the honest PGO
   experiment.

## Further Reading

- [Profile-guided optimization user guide](https://go.dev/doc/pgo) — official
- [PGO announcement](https://go.dev/blog/pgo) — design rationale
- [Compiler optimization wiki](https://github.com/golang/go/wiki/CompilerOptimizations) — inlining and BCE details
