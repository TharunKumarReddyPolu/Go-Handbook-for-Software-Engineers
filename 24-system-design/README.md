# 24 · System Design with Go

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Each design follows the template:
Requirements → APIs → Data model → Architecture → Scaling →
Reliability → Failure modes → Observability → Security → **Go
implementation considerations** (the differentiator: which of this
handbook's patterns slot in where).

## Planned designs

1. **URL shortener**: hashing/ID generation, read-heavy caching
2. **Rate limiter**: token bucket at scale; extends
   [08 §5](../08-concurrency/05-patterns.md)
3. **API gateway**: routing, auth, backpressure at the edge
4. **Payment service**: the idempotency/ledger stack from
   [25 §1-3](../25-fintech-with-go/01-money-and-payments.md)
5. **Notification system**: fan-out, per-channel rate limits,
   delivery guarantees
6. **Job scheduler**: the stage-5 processor
   ([08-concurrency](../08-concurrency/examples/05-jobprocessor/))
   grown to a distributed fleet
7. **Distributed cache**: sharding, singleflight, invalidation
   ([19 §3](../19-performance/03-concurrency-performance.md) pairs)
8. **Event processing system**: the Kafka stack from
   [18](../18-kafka-with-go/01-kafka-concepts.md) end to end
9. **Transaction ledger**: the ledger from
   [25 §2](../25-fintech-with-go/02-double-entry-ledger.md) at scale
10. **Fraud detection pipeline**: the risk stack from
    [25 §4](../25-fintech-with-go/04-risk-and-compliance.md)
11. **Order processing system**: sagas + outbox
    ([25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md))
12. **Real-time analytics system**: windowed aggregation on streams
