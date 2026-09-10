# 15 · Microservices

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The decision question is answered in
[26 §4](../26-go-interview-preparation/04-senior-scenarios.md); the
event backbone in [17-18](../18-kafka-with-go/01-kafka-concepts.md).

## Planned chapters

1. **Monolith → modular monolith → microservices**: the honest
   progression; boundaries as team contracts
2. **Service boundaries**: DDD-in-Go pragmatics, data ownership,
   the "one database per service" reality
3. **API contracts**: REST vs gRPC (a table, not a war), versioning,
   backward compatibility discipline
4. **Service discovery & load balancing**: client-side vs server-side,
   Go's defaults
5. **Timeouts, retries, circuit breakers, bulkheads**: the resilience
   quartet with Go implementations; extends the retry tables in
   [05 §2](../05-errors/02-error-design.md)
6. **Idempotency across services**: the payment patterns from
   [25 §1](../25-fintech-with-go/01-money-and-payments.md) generalized
7. **Distributed transactions**: sagas and outboxes in depth; extends
   [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md)
8. **Event-driven architecture**: when events replace calls
9. **When NOT to use microservices**: the anti-chapter, treated with
   equal depth
