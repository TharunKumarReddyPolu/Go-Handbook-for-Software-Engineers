# 17 · Messaging

**Status: in depth.** One Go module (`examples/messaging`) implements
the portable patterns with a deterministic test suite; Kafka-specific
depth lives in [18 §1](../18-kafka-with-go/01-kafka-concepts.md) and
[18 §3](../18-kafka-with-go/03-producer-consumer.md). This section is
the broker-agnostic layer beneath it: the models, the tradeoffs, and
the patterns that survive every transport.

## Chapters

1. **[Queues, logs & pub/sub](01-queues-logs-pubsub.md)**: the three
   models and the delivery semantics each one gives you for free
2. **[Broker comparison](02-broker-comparison.md)**: Kafka, RabbitMQ,
   NATS, SQS, Pub/Sub on the axes that actually decide architecture
3. **[Portable patterns](03-portable-patterns.md)**: the consumer
   loop, retries, DLQ, idempotency, and per-key ordering, broker by
   broker
4. **[Schemas, evolution & testing](04-schemas-evolution-and-testing.md)**:
   the versioned envelope, registry tradeoffs, and the testing tiers

## The example

`examples/messaging` is a minimal in-memory broker (~200 lines) that
implements the queue model's exact semantics: at-least-once delivery,
attempt counts, exponential backoff via an injected clock (no sleeps
in tests), a triage-ready DLQ, and panic-to-terminal classification.
Run it:

```bash
go test ./17-messaging/... -v
```

The suite is the portability contract: the same five assertions
(ack, redeliver with attempts, DLQ at max, dedup collapse, per-key
order) should be ported verbatim to any real broker adapter you
write.

## Where to go next

- [18 Kafka with Go](../18-kafka-with-go/README.md): the log-based
  broker in full depth, transport-free handlers, offset management
- [15 §4](../15-microservices/04-idempotency-sagas-outbox.md): the
  claim-store and outbox patterns the consumer side depends on
- [16 §5](../16-distributed-systems/05-delivery-backpressure-shedding.md):
  delivery semantics and backpressure as system properties
