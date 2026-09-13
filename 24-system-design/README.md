# 24 · System Design with Go

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Each design follows the template:
Requirements → APIs → Data model → Architecture → Scaling →
Reliability → Failure modes → Observability → Security → **Go
implementation considerations** (the differentiator: which of this
handbook's patterns slot in where).

## Planned designs

Each design is a worked walkthrough of the template, not a new
system to build: the Go implementation considerations cite the
handbook's own tested examples and explain how they compose at
design scale.

1. **URL shortener**: hashing/ID generation, read-heavy caching:
   the warm-up design
2. **Rate limiter & API gateway**: edge concerns in one design:
   token buckets at scale, auth, backpressure at the door (extends
   [21 §4](../21-security/04-limits-and-hardening.md))
3. **Payment service & transaction ledger**: the
   idempotency/ledger/outbox stack from
   [25 §1-3](../25-fintech-with-go/01-money-and-payments.md) as one
   coherent system
4. **Notification system**: fan-out, per-channel rate limits,
   delivery guarantees
5. **Job scheduler & distributed cache**: the stage-5 processor
   ([08-concurrency](../08-concurrency/examples/05-jobprocessor/))
   grown to a fleet, plus sharding, singleflight, invalidation
   ([19 §3](../19-performance/03-concurrency-performance.md) pairs)
6. **Event processing & real-time analytics**: the Kafka stack from
   [18](../18-kafka-with-go/01-kafka-concepts.md) end to end, with
   windowed aggregation on streams
7. **Fraud detection & order processing**: the risk stack from
   [25 §4](../25-fintech-with-go/04-risk-and-compliance.md) and the
   saga-driven order flow from
   [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md)

(Consolidated from 12 planned designs into 7: related systems are
designed together because their tradeoffs only make sense side by
side; the URL shortener stays solo as the teachable warm-up.)
