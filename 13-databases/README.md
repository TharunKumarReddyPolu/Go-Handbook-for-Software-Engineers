# 13 · Databases

**Status: in depth: 5 chapters + a two-tier bank example.** Standard
library first (`database/sql` + pgx as the driver); ORMs covered as an
honest tradeoff, not a default. The transactional contract this
section explains is used in anger by the fintech ledger:
[25 §2-3](../25-fintech-with-go/02-double-entry-ledger.md).

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [database/sql in depth](01-database-sql.md) | the pool-handle mental model, the 90% queries, scanning traps, drivers |
| 2 | [Transactions & isolation](02-transactions-and-isolation.md) | the transaction function, read phenomena, optimistic vs pessimistic, retry discipline |
| 3 | [Pooling, drivers & migrations](03-pooling-drivers-migrations.md) | sizing math, symptoms, pgx vs database/sql, expand/contract deploys |
| 4 | [Repositories, ORMs & testing](04-repositories-and-testing.md) | consumer-side interfaces, the ORM ledger, sqlc, the two-tier test strategy |
| 5 | [Caching with Redis](05-caching-with-redis.md) | cache-aside, invalidation rules, stampede control, the degrade-not-die hierarchy |

## The example

`examples/bank/` is a payments store built on the repository pattern
from chapter 4:

- `bank.go`: the consumer-side `PaymentStore` interface and domain types
- `fake.go`: the pure-Go implementation (atomic behavior mirrored, no
  database needed)
- `postgres.go`: the `database/sql` implementation, with the overdraft
  guard in SQL (`balance >= $1`) and the panic-safe `withinTx` helper
- `bank_test.go`: unit tier, runs always: the parity harness runs the
  same cases against every registered store, plus a lease-release test
  for the transaction function (commit/rollback/panic paths) and race
  bait for the fake
- `postgres_test.go`: database tier under the `db` build tag, skipped
  without `TEST_DATABASE_URL` (the same skip-cleanly pattern as
  [18-kafka-with-go](../18-kafka-with-go/))

```bash
go test ./13-databases/...                     # unit tier: always green
TEST_DATABASE_URL=postgres://postgres:pw@localhost:5432/postgres?sslmode=disable \
  go test ./13-databases/examples/bank -tags=db -v   # real Postgres tier
```

The concurrency tests are the point: N goroutines charging one
account must conserve the balance exactly, which is only provable
because the overdraft guard lives in SQL, not in check-then-act Go
code (chapter 2's lesson, enforced by a test).

Deliberately out of scope here: injection and secret handling
([21-security](../21-security/)), metrics around pool stats
([20-observability](../20-observability/)), the service-layer wiring
([14-backend-development](../14-backend-development/)).

## Progress checklist

- [x] database/sql: pool model, context discipline, scanning, drivers
- [x] Transactions: the transaction function, isolation levels, retries
- [x] Optimistic vs pessimistic concurrency
- [x] Pool sizing math, saturation symptoms, fleet budgets
- [x] pgx vs database/sql, PgBouncer protocol caveats
- [x] Migrations without frameworks, expand/contract deploy rules
- [x] Repository pattern: consumer-side interfaces, domain shapes
- [x] ORM tradeoffs and sqlc, decided honestly
- [x] Two-tier testing with a parity harness
- [x] Cache-aside with Redis, invalidation, singleflight stampede control
- [ ] Redis cluster topology and distributed locks (16-distributed-systems)
- [ ] Full EXPLAIN walkthrough (19-performance follow-up)
