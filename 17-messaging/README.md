# 17 · Messaging

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Kafka's model is covered in depth in
[18 §1](../18-kafka-with-go/01-kafka-concepts.md); this section is the
broker-agnostic layer beneath it.

## Planned chapters

1. **Queues vs logs vs pub/sub**: the three models and their
   delivery-semantics DNA (queue = work distribution, log = event
   history, pub/sub = notification)
2. **Producers & consumers**: the vocabulary in Go terms; the
   transport-free handler seam from
   [18 §3](../18-kafka-with-go/03-producer-consumer.md) as the
   portable pattern
3. **RabbitMQ with Go**: AMQP model (exchanges, bindings,
   acknowledgments) and where it beats log-based brokers
4. **NATS with Go**: lightweight subjects, request-reply, JetStream
   for persistence; when "simple and fast" wins
5. **Consumer groups, offsets & retries**: cross-broker comparison of
   the semantics from [18 §1](../18-kafka-with-go/01-kafka-concepts.md)
6. **Dead-letter queues**: the DLQ contract from
   [18 §3](../18-kafka-with-go/03-producer-consumer.md), generalized
7. **Ordering & idempotency**: the same two problems everywhere
8. **Schema evolution**: envelope pattern (18 §3) and registries
