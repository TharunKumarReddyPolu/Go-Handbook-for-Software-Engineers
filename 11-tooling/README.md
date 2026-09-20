# 11 · Tooling

**Status: in depth.** Four chapters on the professional Go
workflow: the tools every maintainer assumes you know, wired into
editor and CI so quality is mechanical. [01 Section 2](../01-go-fundamentals/02-toolchain-and-workflow.md)
covers the daily verbs; [19 Section 1](../19-performance/01-measure-first.md)
teaches the profiles' mechanics; this section is the operational
layer above both.

## Chapters

1. **[The professional workflow](01-professional-workflow.md)**:
   the tier map (editor → save → CI), gofmt/goimports/vet/
   staticcheck/gopls, and this repo's CI as the worked example
2. **[pprof & trace in practice](02-pprof-and-trace-practice.md)**:
   safe mounting, capture automation, continuous profiling, the
   readiness checklist
3. **[Delve](03-delve-debugging.md)**: breakpoints, goroutine
   inspection, core dumps, and the debug-build discipline
4. **[Supply chain & modernizers](04-supply-chain-and-modernizers.md)**:
   govulncheck triage, SBOMs, `go fix`, and the upgrade rhythm

## The tool-to-tier map

| Tier | Tools | Latency | Authority |
|---|---|---|---|
| Editor | gopls (+staticcheck), goimports | ms | suggests |
| Save | gofmt, goimports | ms | formats |
| Pre-push | go vet, go test | seconds | personal gate |
| CI | vet, staticcheck, -race, govulncheck, linkcheck | minutes | blocks |

The design rule: every check exists at the cheapest tier that can
carry it, and CI repeats them all as the blocking gate ([01
Section 1](01-professional-workflow.md)'s feedback-tier example).

## The commands worth memorizing

```bash
gofmt -l . && go vet ./... && staticcheck ./...   # the hygiene trio
go test ./... -race -shuffle=on                    # the correctness gate
govulncheck ./...                                  # the supply-chain scan
dlv test ./pkg -run TestX                          # the debugging entry
go tool pprof -seconds=30 http://localhost:6060/debug/pprof/profile
```
