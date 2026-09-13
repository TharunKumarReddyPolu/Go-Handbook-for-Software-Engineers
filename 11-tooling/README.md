# 11 · Tooling

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The daily workflow is already covered
in [01 §2 Toolchain & workflow](../01-go-fundamentals/02-toolchain-and-workflow.md).

## Planned chapters

1. **The professional workflow**: gofmt, goimports, go vet,
   staticcheck, gopls and editor setup, wired into editor + CI (this
   repo's [ci.yml](../.github/workflows/ci.yml) as the worked
   example)
2. **pprof & trace in practice**: extends
   [19 §1](../19-performance/01-measure-first.md) with capture
   automation and continuous profiling
3. **Delve**: debugging real services: breakpoints, goroutine
   inspection, core dumps
4. **govulncheck, go fix & modernizers**: call-graph-aware
   vulnerability scanning (SBOM basics,
   [21-security](../21-security/) pairs), keeping code current
   across Go versions (new modernizers landed in Go 1.27: see
   [meta/versioning.md](../meta/versioning.md))

(Consolidated from 6 planned topics: gopls/editor setup folds into
the workflow chapter, go fix/modernizers join the supply-chain
chapter.)
