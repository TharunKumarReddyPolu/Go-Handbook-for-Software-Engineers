# 16 · Distributed Systems

**Status: in depth: 5 chapters + the fencing-lease example.** The
section generalizes what [18-kafka-with-go](../18-kafka-with-go/),
[25 Section 3](../25-fintech-with-go/03-integrity-and-exactly-once.md), and
[15](../15-microservices/) built for specific transports into the
systems fundamentals, with Go implementations for the parts you
actually own.

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [The failure model: CAP & PACELC](01-failure-model-cap-pacelc.md) | partial failure, the unknown-response state, the per-operation consistency matrix |
| 2 | [Replication & consistency models](02-replication-and-consistency.md) | the ladder from eventual to linearizable, read-your-writes routing, conflict resolution |
| 3 | [Consensus, leases & fencing](03-leader-election-leases-fencing.md) | Raft intuition, lease TTLs from measured pauses, the fencing token that stops zombie leaders |
| 4 | [Quorums, sharding & ordering](04-quorums-sharding.md) | W+R>N and its three caveats, shard keys as promises, hot keys, resharding |
| 5 | [Delivery, backpressure & load shedding](05-delivery-backpressure-shedding.md) | at-least-once + two-tier dedup, bounded queues, block-vs-shed policies |

## The example

`examples/lease/` implements the fencing-token pattern end to end,
deterministically:

- `Coordinator`: allocates leases with monotonic fencing tokens over
  an injected clock; expired holders are evicted on the TTL
- `Fenced`: the storage-side guard, the analog of the conditional
  `UPDATE ... WHERE fencing_no < $1`

The centerpiece test is the zombie-leader scenario, the chapter in
miniature: A holds the lease, pauses past the TTL, B acquires and
writes, A wakes still believing and attempts a write with its stale
token. The fence rejects it (`ErrFencedOut`) and A's write never
lands: the exact property that separates a lease from a hope.

```bash
go test ./16-distributed-systems/... -v
```

Chapter 5's `Dedupe` (two-tier deduplication) lives in its chapter as
the teaching implementation; its backstop is always a storage
constraint ([15 Section 4](../15-microservices/04-idempotency-sagas-outbox.md)'s
claim table).


