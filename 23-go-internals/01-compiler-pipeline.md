# The compiler pipeline

## Why Does This Matter?

Go's observable behavior: fast compiles, strict errors, dead code
refused, unused imports rejected: all fall out of the compiler's
design. You do not need to hack on the compiler to benefit from
knowing its shape; you need it to answer the engineer's questions:
why is this build fast, why did the compiler reject this, what did
it do with my code? Every optimization discussed in [19
§4](../19-performance/04-compiler-and-pgo.md) is a *pass* in this
pipeline; this chapter is where the passes live.

## Mental Model

```mermaid
flowchart LR
    S[source.go] --> L[Lexer]
    L --> P[Parser]
    P --> AST[AST]
    AST --> T[Type check]
    T --> IR[IR / SSA]
    IR --> OPT[Optimization passes]
    OPT --> MC[Machine code]
```

Two properties explain most of what Go feels like:

- **Compiles are fast because the unit is the package and the work
  is linear.** No header files, no template instantiation explosions,
  no link-time code generation by default. A cold build of a large
  project takes seconds; incremental builds take none.
- **The compiler is strict where other languages are runtime-
  lenient.** Unused variables, unused imports, unreachable code:
  all compile errors. The design bet: errors caught at compile time
  cost seconds; errors caught in production cost on-call.

## How It Works

**Lexer/parser**: Go's grammar is deliberately simple (no
operator-precedence surprises beyond a short table; semicolon
insertion is a lexical rule, not a parser hack). This is why
`gofmt` can parse and re-print any file losslessly: the AST is
faithful to the source, comments included. It is also why every Go
tooling tool (gopls, `go vet`, gofix) shares the same `go/ast`
world.

**Type checking**: one pass over the AST building the type graph.
Generics (Go 1.18+) type-check via constraint satisfaction here;
instantiation happens at compile time, which is why generic code
has no runtime dispatch cost the way interface calls can.

**IR to SSA**: the compiler lowers the typed AST to an
SSA form (every variable assigned once, phi nodes at control-flow
joins). Almost every optimization you have heard of is an SSA
pass:

| Pass | What it does | Observable flag |
|---|---|---|
| Escape analysis | stack vs heap per allocation | `-gcflags=-m` |
| Inlining | body substitution under a cost budget | `-gcflags=-m` (`can inline`) |
| Bounds-check elimination | removes redundant range checks | `-d=ssa/check_bce/debug=1` |
| Dead store/load elimination | removes work it can prove unused | (internal) |
| Devirtualization | concrete-type interface calls become direct calls | `-gcflags=-m` (`devirtualizing`) |

**Machine code**: SSA lowers to architecture-specific assembly
(`GOOS`/`GOARCH`), then the linker produces the static binary. The
`go tool compile -S` output is the ground truth for "what did the
compiler emit" (chapter 7 reads it).

## Syntax / API

The three commands every engineer should run occasionally:

```bash
go build -gcflags="-m" ./pkg/        # escape + inline decisions
go build -d=ssa/check_bce/debug=1 ./pkg/   # remaining bounds checks
go tool compile -S x.go 2>&1 | head  # actual assembly
```

The full pass list is discoverable in the source; the diagnostic
flags above are the stable, documented surface.

## Basic Example

The pipeline is visible from one function. `Sum` from this
section's example package (`23-go-internals/examples/internals/`):

```go
func Sum(s []int) int {
    total := 0
    for i := 0; i < len(s); i++ {
        total += s[i]
    }
    return total
}
```

The compiler reports `can inline Sum` (cost under budget) and, via
the BCE debug flag, finds *no* bounds check for `s[i]`: the loop
condition `i < len(s)` is exactly the proof the pass needs.

## Real-World Example

Why does Go refuse unused imports? The parser + type checker know
the import set before codegen; enforcing there turns "build with
dead references" from a link-time mystery into an instant edit-time
error. The same philosophy places `go vet`'s checks before you ever
commit: every stage of the pipeline is a debugging surface, and the
earliest one is the cheapest.

## Production Example

Build caching makes the pipeline's cost model practical: the build
cache keys on file content, flags, and toolchain version, so CI
builds hit warm caches after the first run
([06 §5](../06-packages-modules/05-reproducible-builds.md)'s
reproducible flags matter here: different flags = different cache
keys). `-trimpath` and pinned toolchains make the cache portable
across machines. When someone proposes "we should precompile" or
"we should split packages for build speed", the answer is measured
against this pipeline: package granularity affects parallelism;
flags affect cache keys; both are testable
([19 §1](../19-performance/01-measure-first.md)).

## Common Mistakes

| Mistake | Reality | Do instead |
|---|---|---|
| Trusting intuition about inlining | The budget is mechanical: cost ≤ 80, and inlining enables other passes | Ask with `-gcflags=-m` |
| Adding interfaces "for later" | Devirtualization only fires when the concrete type is provable | Keep hot paths concrete ([19 §4](../19-performance/04-compiler-and-pgo.md)) |
| Believing `unsafe` tricks beat the compiler | The SSA passes usually win | Benchmark before believing |
| Ignoring vet | Vet is the pipeline's cheap early stage | Run it in CI (this repo does) |
| Assuming generics slow builds | Type-check cost, not codegen bloat, and modest | Measure build times before refactoring around it |

## Idiomatic Go

- Do not write for the compiler: write clearly; the passes reward
  clarity (small functions inline; simple ranges prove bounds).
- Keep hot functions small enough to inline: the budget is the
  guidance.
- Treat compiler diagnostics as documentation: they are current,
  versioned, and true for *your* flags.

## Performance Considerations

Compilers optimize hot code well and cold code adequately; your
leverage is structure, not micro-idioms. The pass that matters
most for services is escape analysis ([19
§2](../19-performance/02-memory-and-allocations.md)): allocation
volume drives GC work, which drives tail latency. Reading `-m`
output on your hot path is the cheapest performance review there is.

## Concurrency Considerations

The memory model's guarantees (chapter 6) are what make compiler
reordering safe to reason about: within the guarantees, the
compiler may reorder, and it will. Data-race-free code does not
observe reordering; racy code observes it in ways that look like
hardware bugs. This is why "it works without the mutex" is not
evidence.

## Security Considerations

The toolchain ships mitigations at the codegen stage: stack guard
checks on functions with locals, position-independent executables
by default on most platforms, and `GOEXPERIMENT` hardening flags as
they land. `govulncheck` covers dependencies, but the binary's own
surface is the compiler's; rebuild with new toolchains when CVEs
name the runtime ([21 §5](../21-security/05-secrets-and-supply-chain.md)).

## Testing Strategy

- Pin behaviors with tests, not transcripts: this section's
  `internals` package asserts escape/no-escape with
  `testing.AllocsPerRun`, so a toolchain upgrade that changes the
  decision fails CI loudly rather than silently regressing.
- Snapshot compiler-flag output in docs sparingly; prefer naming
  the command and the expected shape.

## Interview Questions

1. Name the stages from source to binary and one optimization per
   stage.
2. Why are Go builds fast? (Package granularity, simple grammar,
   linear type check, no header/template expansion, cacheable.)
3. What is SSA and why does every optimization live there?
4. Where does escape analysis sit in the pipeline, and what does
   its decision control?

## Practice Exercises

1. Run `-gcflags=-m` on the internals package; find every
   "escapes to heap" line and explain each from the source.
2. Make `Sum`'s bounds check reappear (index from `len(s)-1-i`)
   and confirm with the BCE debug flag.
3. Time a cold and warm `go build ./...` in this repo; explain the
   difference with the cache model.

## Further Reading

- [go tool compile documentation](https://pkg.go.dev/cmd/compile)
- [Compiler and runtime design docs](https://github.com/golang/go/tree/master/src/cmd/compile)
- [Go assembly reference (ground truth for codegen)](https://go.dev/doc/asm)
