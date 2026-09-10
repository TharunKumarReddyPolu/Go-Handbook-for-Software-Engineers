# 28 · Projects

**Status: outline with the full project ladder defined — project guides
are written last so they can reference finished sections (see
[ROADMAP.md](../ROADMAP.md)).**

Levels build on each other: each project uses the patterns from the
sections it lists. Every project ships with tests, CI-appropriate
checks, and a README — the handbook's own
[CONTRIBUTING standards](../CONTRIBUTING.md#code-standards) apply.

## Level 1 — Foundations

| Project | Sections it exercises |
|---|---|
| CLI tool (flags, io, testing) | [01](../01-go-fundamentals/README.md), [10](../10-testing/README.md) |
| URL shortener (stdlib HTTP) | [01](../01-go-fundamentals/README.md), [05](../05-errors/README.md), [12](../12-http-networking/) |
| REST API (stdlib, in-memory store) | [05](../05-errors/README.md), [10](../10-testing/README.md) |
| File processor (streaming, worker pool) | [08](../08-concurrency/README.md) stages 2-3 |

## Level 2 — Backend patterns

| Project | Sections it exercises |
|---|---|
| Task queue (bounded, retries, DLQ shape) | [08](../08-concurrency/README.md) stage 5 |
| URL crawler (politeness, rate limits) | [08 §5](../08-concurrency/05-patterns.md) |
| Rate limiter service | [08 §5](../08-concurrency/05-patterns.md), [19 §3](../19-performance/03-concurrency-performance.md) |
| Redis-backed API (cache-aside, singleflight) | [13](../13-databases/), [19 §3](../19-performance/03-concurrency-performance.md) |
| Authentication service (sessions, hashing) | [12](../12-http-networking/), [21](../21-security/) |

## Level 3 — Distributed systems

| Project | Sections it exercises |
|---|---|
| Event-driven order service | [18](../18-kafka-with-go/README.md), [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md) |
| Kafka-based processing system | [18](../18-kafka-with-go/README.md) end to end |
| Distributed job scheduler | [16](../16-distributed-systems/), leases + the stage-5 pattern |
| Notification platform | [17](../17-messaging/), fan-out at scale |
| Metrics platform | [20](../20-observability/), [19](../19-performance/README.md) |

## Level 4 — FinTech-grade integrity

| Project | Sections it exercises |
|---|---|
| Payment processing system | [25 §1](../25-fintech-with-go/01-money-and-payments.md), [18](../18-kafka-with-go/README.md) |
| Financial ledger | [25 §2](../25-fintech-with-go/02-double-entry-ledger.md) (the example, productionized) |
| Fraud detection pipeline | [25 §4](../25-fintech-with-go/04-risk-and-compliance.md) |
| Distributed transaction processor | [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md), [16](../16-distributed-systems/) |

## Level 5 — Capstone

**Production-Grade Financial Transaction Platform**

Everything the handbook teaches in one system:

- **Go** — service layout per [14](../14-backend-development/)
- **PostgreSQL** — the ledger per [25 §2](../25-fintech-with-go/02-double-entry-ledger.md)
- **Kafka** — event backbone per [18](../18-kafka-with-go/README.md)
- **Redis** — caching per [13](../13-databases/)
- **REST/gRPC** — transport per [12](../12-http-networking/)
- **OpenTelemetry** — per [20](../20-observability/)
- **Docker + Kubernetes** — per [22](../22-production-go/)
- **CI/CD** — per [22](../22-production-go/)

Success criteria (the real ones): idempotent end to end, reconciled
nightly, SLOs defined, chaos-tested consumer, blameless-postmortem
ready. The capstone guide walks the build order and gates.
