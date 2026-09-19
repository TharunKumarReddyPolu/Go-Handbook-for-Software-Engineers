# 25 · FinTech with Go

Money is the least forgiving domain software touches: every bug is
accounting, every duplicate is a loss, every lost message is a
regulatory finding. This section teaches the engineering patterns that
make financial systems trustworthy: double-entry ledgers, idempotency,
reconciliation: with Go implementations you can run and test.

> **Honesty labels.** Examples here are educational. They demonstrate
> *patterns*, not audited, production-ready systems. Real financial
> infrastructure adds: ACID databases with defined isolation, regulatory
> compliance (PCI, SOX, jurisdictional rules), formal review of money
> flows, and far more. Chapters say so at the point of simplification.

## Objectives

By the end of this section you can:

- Model money in Go correctly (integer minor units, currencies, no floats)
- Design and implement a double-entry ledger with integrity invariants
- Build idempotent payment processing (the claim pattern from 18, durable)
- Reason about retries and distributed transactions without lying to yourself
- Design reconciliation and audit trails, and know why they exist
- Separate the exactly-once myths from what is actually achievable

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Money & payments in Go](01-money-and-payments.md) | Minor units, idempotency, retry semantics |
| 2 | [Double-entry ledgers](02-double-entry-ledger.md) | The ledger model, invariants, a tested implementation |
| 3 | [Integrity, consistency & exactly-once myths](03-integrity-and-exactly-once.md) | Distributed transactions, outbox, reconciliation |
| 4 | [Risk, market data & compliance](04-risk-and-compliance.md) | Fraud pipelines, event-driven finance, regulatory notes |

## Examples

`examples/ledger/`: an educational in-memory double-entry ledger:

- `money.go`: Money type: integer minor units + currency, no floats
- `ledger.go`: accounts, balanced journal entries, query API
- `money_test.go`, `ledger_test.go`: the invariant tests that make the
  design trustworthy (including concurrency tests under `-race`)

```bash
go test ./25-fintech-with-go/... -race
```


