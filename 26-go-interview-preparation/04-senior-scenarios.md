# Senior track & scenarios

Senior interviews are scenario-driven: given an ambiguous system and a
symptom, what do you do first, second, third? The graded skill is
process under constraints — clarify, measure, decide, and say the
tradeoffs out loud.

## Architecture

**Q: How do you structure a production Go backend?**

Strong answer: domain package with business types and rules (no I/O);
repository interfaces declared at the *consumer* (service) side;
transport layer (HTTP/gRPC) converting wire↔domain; main does wiring.
Anti-structure to name: `utils/`, `models/`, framework-vocabulary
packages. The senior addendum: modular monolith first; split into
services only on proven boundaries. — [14-backend-development](../14-backend-development/) (when it ships)

**Q: When should a team NOT use microservices?**

Strong answer: unknown boundaries, small teams, transactional
coupling that would become sagas-for-nothing, and ops maturity that
can't handle distributed debugging. The counterpoint that shows
judgment: the monolith/microservice axis is a team-communication
decision (Conway), not a technical purity contest. —
[15-microservices](../15-microservices/) (when it ships)

**Q: You inherit a Go monolith that deploys hourly and breaks weekly.
What do you do in the first month?**

Strong answer: measurement before surgery — CI (build/vet/test/race),
observability (RED metrics, tracing), deployment rollback capability,
then the highest-value refactor inside the monolith (package
boundaries, error handling). Microservices appear only after
boundaries are proven and the pain is team-scaling, not code-quality.

## Reliability & scalability

**Q: Design a payment service that must not double-charge.**

Strong answer: idempotency keys end-to-end (API → outbox → Kafka →
ledger claim), claim+result in one DB transaction, PSP timeouts
treated as unknown with reconciliation queries, replay-returns-
original. The grade: the *workflow* (pending claims, reconciliation),
not the happy path. — [25-fintech](../25-fintech-with-go/01-money-and-payments.md)

**Q: A service handles 50k RPS with p99 40ms. Traffic doubles; p99
goes to 900ms. Diagnose.**

Strong answer: capacity shape first (queueing — utilization over ~70%
turns nonlinear), then profiling: CPU, allocation rate, lock
contention, GC, upstream latency. The senior move: instrument the
*queue depth* and identify the serialization point before optimizing
code. Fixed concurrency points (one mutex, one DB) are the usual
culprits. — [19-performance](../19-performance/03-concurrency-performance.md)

**Q: What's your timeout/retry/circuit-breaker policy for service-to-
service calls?**

Strong answer: budgets propagate (context deadlines, split per hop);
retries only for retryable classes with backoff+jitter, capped, and
idempotency-keyed; circuit breakers per dependency to fail fast and
let recovery; bulkheads isolate critical from batch traffic. The
senior note: retries multiply load — a 3x retry on a struggling
dependency is a DDoS on yourself.

**Q: Kafka consumer lag is growing in one partition. Walk me through
it.**

Strong answer: time-to-drain math, producer-rate history (traffic or
hot key?), handler p95 (degraded? profile it), rebalance history, DLQ
rate. Fix ordering: scale consumers → optimize handler → re-partition
(with its "forever" caveat). — [04-observability-tuning](../18-kafka-with-go/04-observability-tuning.md)

## Observability & production debugging

**Q: Memory climbs 10MB/hour at steady traffic. Debug it.**

Strong answer: goroutine count first (leaks are usually goroutines —
pprof profile shows identical parked stacks), then heap live-set
growth (unbounded maps/caches), then GC pacing. The graded instinct:
*goroutine profile before heap profile*. — [07-pitfalls](../08-concurrency/07-pitfalls.md)

**Q: What's on your minimal dashboard for a Go service?**

Strong answer: RED (rate, errors, duration/p99), saturation (CPU,
mem, goroutines, GC pause), dependency health (upstream latency/error
rates, DB pool saturation, consumer lag if Kafka), and panic rate.
The senior framing: dashboards answer your runbook's questions — start
from the questions. — [20-observability](../20-observability/) (when it ships)

**Q: You get paged: 5xx at 20%. First 15 minutes?**

Strong answer: stabilize before diagnosing — check deploys (rollback
if correlated), check scope (all instances? one region? one
endpoint?), check dependencies (a downstream often explains "our"
errors). Mitigate (shed, fail over, scale) then root-cause with
profiles/logs/traces captured *during* the incident. Write the
timeline as you go; the postmortem is already being written.

## Team & technical leadership

**Q: How do you get a team to stop writing non-idiomatic Go?**

Strong answer: standards that live in tooling (gofmt, vet,
staticcheck, CI gates, review templates), a written style doc with
*reasons*, exemplar code in the repo, and design review for new
packages. Coercion fails; tooling and examples scale.

**Q: Your team's test suite takes 40 minutes. Plan?**

Strong answer: tier it — unit (fast, parallel, every commit) vs
integration (containers, PR-gated) vs e2e (nightly); replace sleeps
with synchronization; per-test state isolation; measure the suite like
a service. The senior note: flaky tests are deleted or fixed, never
retried into background noise. — [03-integration-and-e2e](../10-testing/03-integration-and-e2e.md)

## Scenario rapid-fire

| Scenario | First move |
|---|---|
| p99 doubles after a deploy | rollback check, then profile diff |
| goroutine count grows without traffic | goroutine profile: identical stacks |
| DB pool exhausted | leak in conn Close; timeout missing on ctx |
| Kafka rebalance storm | slow member hunt; poll interval vs handler time |
| GC CPU > 25% | allocation-rate profile; batch/pool the top churner |
| One tenant slows everyone | per-tenant rate limits + isolation (bulkheads) |
| Graceful shutdown drops requests | drain ordering: stop intake → finish in-flight → exit |
| Race found in auth path | P0: treat as security bug, not tech debt |

## The senior checklist (what to hit in every answer)

1. **Clarify** — restate constraints; ask one sharp question.
2. **Measure** — name the metric/profile that decides.
3. **Decide with tradeoffs** — state what you're choosing *against*.
4. **Failure modes** — what breaks at 10x, and who gets paged.
5. **Verification** — the test/benchmark that proves the fix.

Answer with those five and the level signals take care of themselves.
