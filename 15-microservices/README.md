# 15 · Microservices

**Status: in depth: 5 chapters + the resilience quartet example.**
Phase 6 begins here. The decision question is summarized from
[26 §4](../26-go-interview-preparation/04-senior-scenarios.md); the
event backbone is [18-kafka-with-go](../18-kafka-with-go/); the
financial-grade saga/outbox builds are
[25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md).

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Monolith to microservices](01-monolith-to-microservices.md) | the honest progression, split-ready layout, the signals that actually justify splitting |
| 2 | [Boundaries & contracts](02-boundaries-and-contracts.md) | data ownership, REST vs gRPC table, versioning and deprecation discipline |
| 3 | [The resilience quartet](03-resilience-patterns.md) | timeouts, retries with jitter, circuit breakers, bulkheads; retry-storm arithmetic |
| 4 | [Idempotency, sagas & the outbox](04-idempotency-sagas-outbox.md) | the three-promise key contract, orchestration vs choreography, atomic events |
| 5 | [Discovery, events & when NOT to](05-discovery-events-and-when-not.md) | gRPC vs VIP balancing, the call-vs-event rule, the refusal checklist |

## The example

`examples/resilience/` implements chapter 3's quartet as small,
composable pieces with deterministic tests:

- `Retry`: exponential backoff with jitter, budget-aware (no attempt
  starts when the backoff cannot fit the remaining deadline),
  classification-gated
- `Breaker`: the closed/open/half-open state machine with an injected
  clock; half-open admits exactly one probe; a failed probe reopens
- `Bulkhead`: per-dependency semaphore with immediate rejection
  (`ErrBulkheadFull`)

The centerpiece is the storm test: 50 goroutines against a failing
dependency produce roughly the breaker threshold's downstream hits
(not 50), while the same storm *with retries and no breaker*
amplifies to ~150. The gap between those two numbers is what the
quartet is for, and the test prints both.

```bash
go test ./15-microservices/... -v   # watch the storm arithmetic
```

## Progress checklist

- [x] The progression: monolith, modular monolith, microservices
- [x] Split-ready layout and the dependency-direction test
- [x] Data ownership and the two rules of a real boundary
- [x] REST vs gRPC as a decision table, not a war
- [x] Versioning, compatibility discipline, deprecation process
- [x] Timeout budgets propagated per hop
- [x] Retries: classification, backoff, jitter, budget awareness
- [x] Circuit breakers: state machine, half-open probing
- [x] Bulkheads and the retry-storm arithmetic
- [x] Idempotency keys: the three-promise contract
- [x] Sagas: orchestration vs choreography, compensations
- [x] The outbox and consumer idempotency (mechanics in 25 §3)
- [x] Discovery and load balancing (incl. the gRPC/VIP trap)
- [x] The call-vs-event decision rule
- [x] The when-NOT-to anti-chapter with a refusal checklist
- [ ] Service mesh deep dive (22-production-go)
- [ ] Container/Kubernetes deployment mechanics (22-production-go)
