# Choosing a Go client

## Why Does This Matter?

There is no "standard" Kafka client for Go — four production-viable
options with genuinely different tradeoffs. This chapter is a decision
document: it compares them honestly, states the recommendation this
handbook's examples use, and — more importantly — gives you the
evaluation framework to re-decide for your context.

## The candidates

| Client | Binding model | Maturity | Highlights |
|---|---|---|---|
| [franz-go](https://github.com/twmb/franz-go) (`twmb/franz-go`) | pure Go | active, modern | full protocol coverage, KIP-level features fast (KIP-848, EOS, rack awareness), high throughput, client-side balancing |
| [IBM/sarama](https://github.com/IBM/sarama) | pure Go | mature, widely deployed | the historical default; large ecosystem knowledge; some advanced KIPs lag |
| [segmentio/kafka-go](https://github.com/segmentio/kafka-go) | pure Go | stable | small, readable API; great for simple consumers/producers; fewer advanced features |
| [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go) | cgo over librdkafka | very mature | battle-tested C core, excellent performance/finishing; cgo build/deploy cost |

## The real tradeoffs

### 1. cgo or not (confluent vs the rest)

librdkafka (C) is extremely fast and hardened. The costs are Go-specific:

- Cross-compilation stops being the free party trick
  (`CGO_ENABLED=0` static binaries are off the table);
- Alpine/distroless images need C toolchains or a compat layer;
- Crashes in C are process-killers that Go cannot recover or profile.

Pure-Go clients deploy as any Go binary. For most teams, deployment
simplicity outweighs librdkafka's throughput edge — Go clients are fast
enough for the vast majority of workloads (hundreds of MB/s per
producer is reachable).

### 2. Feature velocity (franz-go's case)

franz-go tracks Kafka protocol KIPs aggressively (new group protocols,
fetch-on-replica, EOS improvements) and supports modern rebalance
protocols that cut rebalance pause times. If your consumer fleet is
large or rebalance pauses hurt, this matters.

### 3. API shape and your code

- sarama's API is older (partition-consumer loops, manual rebalance
  plumbing); the community has real experience but also real friction
  stories.
- segmentio is the easiest to *read* — worth considering when your
  producer/consumer needs are simple and your team is new to Kafka.
- franz-go's API is comprehensive — more surface to learn, less to work
  around later.

### 4. Operations and debugging

Pure-Go clients debug with normal Go tooling (pprof, race detector).
librdkafka issues need librdkafka logs/stats — a genuinely different
debugging experience your on-call rotation must learn.

## The decision framework

Answer these in order; stop when one decides:

1. **Do you need specific KIP-level features today?** (new group
   protocol, transactions in a specific shape) → check franz-go
   coverage first, librdkafka second.
2. **Does your platform require static binaries/distroless?** → pure
   Go.
3. **Is your team's Kafka experience thin and needs simple?** →
   segmentio for simple flows, franz-go when you'll grow.
4. **Are you operating at librdkafka-class throughput and have cgo
   expertise?** → confluent-kafka-go.

## What this handbook uses, and why

The examples use **franz-go**, for stated reasons (not fashion):

- pure Go: static binaries, normal Go debugging — matches this
  handbook's deployment chapters;
- rebalance protocol support and broker compatibility are current;
- the consumer-group + offset-management story maps cleanly onto the
  job-processor patterns from [08-concurrency](../08-concurrency/);
- active maintenance with responsive issue triage — the open-source
  criteria from [27-open-source](../27-open-source/) applied.

The code is written so the client sits behind a thin transport layer:
the domain logic (`service.go`) consumes `Message` structs and knows
nothing about franz-go. Switching clients touches two files. That
seam — not the client choice — is the durable part of this chapter.

## Common Mistakes

- **Choosing by GitHub stars** — popularity ≠ fit; sarama's stars
  reflect history, not feature currency.
- **No transport/domain seam** — client types (`sarama.ConsumerMessage`)
  leak into business logic; the client becomes unchangeable.
- **Benchmarking clients on localhost** — loopback hides the network
  batching behavior that dominates real throughput.
- **Ignoring client TLS/auth configuration parity** — each client
  configures SASL/TLS differently; audit it like any dependency (see
  [21-security](../21-security/)).

## Interview Questions

1. *Your team must pick a Kafka client for a new Go service. What do
   you evaluate?* — The decision framework; grade on deployment,
   feature, and operational criteria — not preference.
2. *What changes in your build if you pick the cgo client?* — Static
   binaries, images, cross-compilation, crash semantics, debugging.
3. *How do you keep a client swappable?* — The transport/domain seam;
   show the interface you'd define.

## Practice Exercises

1. Read franz-go's and sarama's READMEs and list three concrete
   feature/rebalance differences you'd care about at 50 consumers.
2. Wrap this section's `service.go` handler with a *second* transport
   (segmentio) and count the lines of new transport code — that number
   is the cost of the seam, and it should be small.
3. Write the checklist your team would apply before adopting any new
   infrastructure client; compare it with this chapter's framework.

## Further Reading

- [franz-go](https://github.com/twmb/franz-go) — README + docs dir
- [sarama](https://github.com/IBM/sarama) — MAINTAINERS.md and the issues around KIP tracking
- [kafka-go](https://github.com/segmentio/kafka-go) — the readable-API reference point
- [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go) — build notes for cgo deployments
