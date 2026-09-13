# 22 · Production Go

**Status: in depth.** Six chapters that take a service from "it
works" to "it can run reliably in production," composing the pieces
built earlier: config and lifecycle from
[14](../14-backend-development/) and [12](../12-http-networking/),
limits and degradation from [19](../19-performance/) and
[15](../15-microservices/), signals from [20](../20-observability/),
and controls from [21](../21-security/).

## Chapters

1. **[Configuration & secrets](01-configuration-and-secrets.md)**:
   the three tiers, boot-time batched validation, atomic runtime
   swaps, rotation windows
2. **[The server lifecycle](02-server-lifecycle.md)**: the
   boot/shutdown state machine, readiness vs liveness contracts,
   bounded worker joins, the grace-period arithmetic
3. **[Resource limits in containers](03-resource-limits.md)**:
   GOMAXPROCS/GOMEMLIMIT vs cgroup limits, OOM-kill and throttling
   forensics, the bounds the runtime does not manage
4. **[Dependency failures & degradation](04-dependency-failures.md)**:
   the critical/degrading/best-effort classification, the honest
   fallback menu, fail-closed security controls
5. **[Deploying: containers & Kubernetes](05-deploying-kubernetes.md)**:
   distroless images, the five manifest decisions, probes and grace
   periods, PGO in the pipeline
6. **[Releases & rollbacks](06-releases-and-rollbacks.md)**: the
   blast-radius ladder, N/N-1 compatibility, pre-delegated rollback
   criteria

## The through-line

The same contract appears in every chapter at different scale:
**make failure fast, visible, and reversible.**

| Chapter | Fast | Visible | Reversible |
|---|---|---|---|
| 01 | boot validation | config summary logs | hot swap |
| 02 | honest readiness | probe metrics | redeploy |
| 03 | GC over OOM | runtime metrics | resize |
| 04 | breaker opens | degraded flags | flag flip |
| 05 | SHA-tagged | version labels | rollout undo |
| 06 | canary burn | SLO by version | every layer |
