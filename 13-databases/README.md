# 13 · Databases

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The transactional patterns this section
relies on are used in anger by the fintech ledger:
[25 §2-3](../25-fintech-with-go/02-double-entry-ledger.md).

## Planned chapters

1. **database/sql** — drivers, pool configuration, context-driven
   queries, scanning patterns
2. **PostgreSQL with Go** — pgx vs database/sql tradeoffs, parameterized
   queries, isolation levels in practice (the ledger's posting
   transaction as the worked example)
3. **Transactions & isolation** — read phenomena, when SERIALIZABLE
   earns its cost, optimistic vs pessimistic patterns
4. **Connection pooling** — sizing math, saturation symptoms,
   [19 §3](../19-performance/03-concurrency-performance.md) contention
   framing
5. **Query performance** — EXPLAIN-driven work, indexes for engineers
6. **Migrations** — versioned schema evolution, the integration-test
   contract from [10 §3](../10-testing/03-integration-and-e2e.md)
7. **Repository pattern** — interfaces at the consumer; sqlc's
   generated type-safety as the modern default
8. **Redis & caching** — cache-aside, invalidation, singleflight
   (extends [19 §3](../19-performance/03-concurrency-performance.md))
9. **Distributed caching** — consistency tradeoffs, stampede control
