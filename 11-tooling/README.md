# 11 · Tooling

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The daily workflow is already covered
in [01 §2 Toolchain & workflow](../01-go-fundamentals/02-toolchain-and-workflow.md).

## Planned chapters

1. **The professional workflow**: gofmt, goimports, go vet, staticcheck
   wired into editor + CI (this repo's
   [ci.yml](../.github/workflows/ci.yml) as the worked example)
2. **pprof & trace in practice**: extends
   [19 §1](../19-performance/01-measure-first.md) with capture
   automation and continuous profiling
3. **Delve**: debugging real services: breakpoints, goroutine
   inspection, core dumps
4. **gopls & editor setup**: what the language server gives you and
   why it matters for reviews
5. **govulncheck & supply chain**: call-graph-aware vulnerability
   detection, SBOM basics ([21-security](../21-security/) pairs)
6. **go fix & modernizers**: keeping code current across Go versions
   (new modernizers landed in Go 1.27: see
   [meta/versioning.md](../meta/versioning.md))
