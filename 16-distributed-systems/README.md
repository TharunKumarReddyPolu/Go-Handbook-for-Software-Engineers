# 16 · Distributed Systems

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Delivery semantics and idempotency
already exist at production depth in
[18 §1](../18-kafka-with-go/01-kafka-concepts.md) and
[25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md); this
section broadens to systems fundamentals with Go implementations.

## Planned chapters

1. **CAP, consistency & availability** — what the theorem does and
   doesn't say; PACELC framing
2. **Replication** — leader/follower, sync vs async, read-your-writes
   in practice
3. **Consensus & leader election** — Raft intuition, leases, how Go
   libraries (etcd) expose it
4. **Quorums** — the W+R>N math and its caveats
5. **Sharding & partitioning** — keys, hot spots, resharding pain
6. **Distributed locks** — leases vs locks, why fencing tokens exist
7. **Delivery semantics** — at-most/at-least/effectively-once; the
   [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md)
   teardown, generalized
8. **Retries & exponential backoff** — jitter, budgets, retry
   amplification ([21-security](../21-security/) DoS framing)
9. **Idempotency & deduplication** — the key namespace threading
10. **Ordering** — partial order reality, sequence numbering
11. **Backpressure** — end-to-end flow control; extends
    [08 §5](../08-concurrency/05-patterns.md)
12. **Eventual consistency in practice** — convergence, conflict
    resolution, user-facing staleness
