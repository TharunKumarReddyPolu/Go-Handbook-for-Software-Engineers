# 22 · Production Go

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The shutdown pattern exists in
[08 §3](../08-concurrency/03-context.md); the "works → reliable"
framing drives this section.

## Planned chapters

1. **Configuration & secrets in production** — env precedence,
   validation at boot, secret managers
2. **The server lifecycle** — start probes, readiness gates, graceful
   drain, the two-context shutdown from
   [08 §3](../08-concurrency/03-context.md) completed
3. **Resource limits** — GOMEMLIMIT/GOMAXPROCS in containers
   ([19 §2](../19-performance/02-memory-and-allocations.md) pairs),
   file descriptors, connection bounds
4. **Timeouts & retries as policy** — budget propagation across the
   fleet
5. **Dependency failure handling** — circuit breakers, bulkheads,
   fallback behavior decisions with owners
6. **Deployment** — static binaries, distroless containers, build
   provenance; PGO in the release pipeline
   ([19 §4](../19-performance/04-compiler-and-pgo.md))
7. **Kubernetes for Go services** — probes, resource requests/limits
   vs GOMAXPROCS, rolling updates, HPA on Go metrics
8. **CI/CD** — this repo's [ci.yml](../.github/workflows/ci.yml) grown
   up: build, test, race, scan, release
9. **Reliability engineering** — SLOs driving deploy velocity, error
   budgets, game days
10. **Incidents & rollbacks** — the mitigation-first flow from
    [26 §4](../26-go-interview-preparation/04-senior-scenarios.md),
    blameless postmortems, backward compatibility as a deploy
    prerequisite
