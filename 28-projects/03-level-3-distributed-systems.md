# Level 3: Distributed systems

## Why This Level Exists

Level 3 removes the comforting lie of Level 2: that state lives in
your process. Here, messages arrive twice (or not at all), nodes die
mid-operation, and two services must agree about one fact. Every
project pairs a handbook section's concepts with a running system,
and every project's hardest milestone is *not* the happy path.

## The level rules

- **Infrastructure is allowed now, with stated reasons.** Kafka via
  franz-go ([18 Section 2](../18-kafka-with-go/02-go-clients.md)), Postgres,
  Redis: each dependency gets a paragraph in the README: what it
  buys, what it costs, what happens when it dies.
- **Delivery semantics are explicit.** Each project states at-least-
  once, at-most-once, or effectively-once (via dedup), and the tests
  pin it ([18 Section 1](../18-kafka-with-go/01-kafka-concepts.md),
  [16 Section 5](../16-distributed-systems/05-delivery-backpressure-shedding.md)).
- **Tests run without infrastructure when possible**, and skip
  cleanly when not (the
  [18 example convention](../18-kafka-with-go/README.md)):
  deterministic suites for logic, containerized integration for
  wiring.

## Shared additions to the gates

- Every consumer is idempotent: a test delivers the same message
  twice and asserts one effect ([15 Section 4](../15-microservices/04-idempotency-sagas-outbox.md)).
- Every cross-service write has a story for "crash between step 2
  and 3": outbox, saga, or reconciliation, named and tested.
- Load shape is documented: sustained rate, burst rate, partition
  count, and the backpressure behavior at 10x
  ([16 Section 5](../16-distributed-systems/05-delivery-backpressure-shedding.md)).

## Project 1: Event-driven order service

**Goal:** an order API whose state changes flow through events
(create → paid → shipped), with the outbox pattern bridging the
database and the broker.

| Milestone | Done when |
|---|---|
| M1: order core | API + Postgres with the order state machine enforced in the domain layer ([25 Section 1](../25-fintech-with-go/01-money-and-payments.md)'s transition discipline) |
| M2: outbox | state change + outbox row in one transaction; a relay publishes and marks sent ([15 Section 4](../15-microservices/04-idempotency-sagas-outbox.md)) |
| M3: consumers | at least two consumers (shipping, notifications), each idempotent with dedup keys |
| M4: failure day | broker down: orders still commit; broker returns: backlog drains without duplicates or loss (tested) |
| M5: proof | an integration test that kills the relay mid-drain and asserts convergence |

**The lesson:** the outbox is the cheapest distributed-systems
correctness you will ever buy. The M5 test is the one that makes
interviewers sit up (see [26 Section 4](../26-go-interview-preparation/04-senior-scenarios.md)).

## Project 2: Kafka-based processing system

**Goal:** a production-shaped pipeline: producer with durability
guarantees, consumer group with offset discipline, retries, and a
DLQ. This is [18]'s full stack as a standalone project.

| Milestone | Done when |
|---|---|
| M1: producer | acks=all, idempotence on, compression chosen with a reason ([18 Section 3](../18-kafka-with-go/03-producer-consumer.md)) |
| M2: consumer group | partition-assigned processing, offsets committed after side effects, rebalance survival |
| M3: failure routing | retriable failures retried with backoff; poison messages dead-lettered with triage metadata ([17 Section 3](../17-messaging/03-portable-patterns.md)) |
| M4: ordering | per-key ordering guaranteed and *tested* under rebalance |
| M5: proof | chaos test: kill consumers randomly during load; assert zero loss, bounded duplicate rate, per-key order intact |

**The lesson:** exactly-once is a marketing term; effectively-once is
an engineering achievement. This project is where the distinction
becomes your test suite ([25 Section 3](../25-fintech-with-go/03-integrity-and-exactly-once.md)).

## Project 3: Distributed job scheduler

**Goal:** many workers, periodic and one-shot jobs, exactly-one
execution despite any number of greedy workers: leases, fencing,
and fairness.

| Milestone | Done when |
|---|---|
| M1: lease claiming | `SKIP LOCKED` claim with lease expiry ([16 Section 3](../16-distributed-systems/03-leader-election-leases-fencing.md)) |
| M2: fencing | a worker holding an expired lease cannot corrupt a successor: fencing tokens enforced at the write path |
| M3: scheduling | cron-like periodic jobs and one-shot jobs share the queue; jitter prevents thundering herds ([24 Section 5](../24-system-design/05-job-scheduler-and-distributed-cache.md)) |
| M4: fairness | many worker contention test: no job starves, no double-run within lease windows |
| M5: proof | the zombie-leader test: pause a worker past its lease, assert its writes are rejected |

**The lesson:** this project is [16 Section 3]'s lease theory compiled to
code, and the M5 assertion is the fingerprint of someone who has
actually operated distributed systems.

## Project 4: Notification platform

**Goal:** multi-channel notifications (email/push/webhook) with
priority lanes, dedup, honest 202s, and vendor-rate-limit respect.

| Milestone | Done when |
|---|---|
| M1: lanes | critical/transactional/marketing lanes with different delivery guarantees and shed order ([24 Section 4](../24-system-design/04-notification-system.md)) |
| M2: dedup | idempotency keys at intake *and* channel level; a test proves a double-submit is one email |
| M3: vendor discipline | per-vendor rate limiters, retry-with-backoff on 429s, webhook receiver with signature verification ([21 Section 1](../21-security/01-threat-model-and-validation.md)) |
| M4: honest API | 202 says "queued", never "sent"; status endpoint reflects reality |
| M5: proof | vendor-outage simulation: critical lane degrades last, marketing lane sheds first, by test |

**The lesson:** fan-out systems are shedding systems. The design
lesson is [24 Section 4]; the operational lesson is that your most important
config is the shed order.

## Project 5: Metrics platform

**Goal:** ingest metrics from your own Level 2-3 services, aggregate
into windows, expose query and alerting, and profile your own
throughput ceiling.

| Milestone | Done when |
|---|---|
| M1: intake | a fast ingestion path (batched writes, bounded queues) with drop-policy on overload |
| M2: windows | tumbling-window aggregation from keyed streams ([24 Section 6](../24-system-design/06-event-processing-and-analytics.md)) |
| M3: query + alert | windowed queries and simple threshold alerts; the SLO vocabulary from [20 Section 4](../20-observability/04-slos-and-alerting.md) in the alert definitions |
| M4: self-observability | the platform exposes its own RED metrics ([20 Section 1](../20-observability/01-structured-logging.md)); ingestion benchmarks find the ceiling ([19 Section 1](../19-performance/01-measure-first.md)) |
| M5: proof | backpressure test: 10x overload degrades to dropping low-priority series, never corrupting aggregates |

**The lesson:** observability tools must observe themselves. M4 is
the milestone that turns a demo into a platform.

## When this level is done

Every project's README contains the sentence "what happens when X
dies" for every X, and each claim has a test. You are ready for
[Level 4](04-level-4-fintech-integrity.md) when a duplicate message
feels like a normal Tuesday and your reflex is to reach for the dedup
key, not the retry.

## Further Reading

- [18-kafka-with-go](../18-kafka-with-go/README.md) and [16-distributed-systems](../16-distributed-systems/README.md): the backbone sections
- [17 Section 4](../17-messaging/04-schemas-evolution-and-testing.md): schema evolution discipline every M2+ consumer here needs
