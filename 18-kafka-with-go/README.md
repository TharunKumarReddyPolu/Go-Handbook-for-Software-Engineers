# 18 · Kafka with Go

Kafka is the backbone of event-driven Go services, and the place where
concurrency, delivery semantics, and operational discipline all meet.
This section assumes no Kafka knowledge and builds to production
consumers with retries, dead-letter queues, and idempotency.

## Objectives

By the end of this section you can:

- Explain partitions, offsets, and consumer groups, and what ordering
  guarantees actually exist
- Choose a Go client with stated, defensible reasons
- Write producers with batching, idempotence, and backpressure handling
- Write consumer-group consumers with offset management that survives
  crashes
- Design retry + DLQ flows and idempotent consumers
- Observe and tune a Kafka service under load

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Kafka concepts](01-kafka-concepts.md) | Architecture, partitions, offsets, delivery semantics |
| 2 | [Choosing a Go client](02-go-clients.md) | franz-go vs Sarama vs segmentio vs confluent, with tradeoffs |
| 3 | [Producers & consumers in Go](03-producer-consumer.md) | Working code, offset management, retries, DLQ |
| 4 | [Observability & tuning](04-observability-tuning.md) | Metrics, lag, throughput knobs |

## Examples

`examples/` is a complete order-events processor:

- `service.go`: the *transport-free* processing logic (idempotency,
  ordering, retries): unit-tested without Kafka
- `producer.go` / `consumer.go`: franz-go wiring around that logic
- `service_test.go`: unit tests (run in CI)
- `broker_test.go`: broker tests; skip cleanly without Kafka (set
  `KAFKA_BROKERS` to run them against a local broker)

```bash
go test ./18-kafka-with-go/...                      # unit tier, always green
KAFKA_BROKERS=localhost:9092 go test ./18-kafka-with-go/... -tags=broker
```

## Progress checklist

- [x] Kafka architecture and concepts
- [x] Topics, partitions, offsets, consumer groups
- [x] Delivery semantics (at-least-once / at-most-once / exactly-once myths)
- [x] Go client comparison with tradeoffs
- [x] Producer implementation (batching, idempotence)
- [x] Consumer implementation (groups, offset management)
- [x] Partitioning and ordering
- [x] Serialization and schema evolution
- [x] Retries and dead-letter handling
- [x] Idempotent consumers
- [x] Observability
- [x] Performance tuning
