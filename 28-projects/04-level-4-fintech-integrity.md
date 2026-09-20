# Level 4: FinTech-grade integrity

## Why This Level Exists

Level 4 raises the stakes one notch: in Levels 1-3, a duplicate
message is an annoyance; here it is money. These four projects share
one theme, stated once and enforced everywhere: **money is never
created, destroyed, or moved twice.** Everything else (latency,
throughput, elegance) negotiates with that sentence and loses.

## The level rules

- **Educational, clearly labeled.** These are production-*shaped*
  systems for learning, not certified financial infrastructure. Your
  READMEs carry the same disclaimer this handbook's fintech section
  uses ([25](../25-fintech-with-go/README.md)). Real payment work
  needs PCI-DSS scope review, licensed partners, and compliance sign-
  off that no side project provides.
- **Integer minor units only.** No floats touch money, ever
  ([25 Section 1](../25-fintech-with-go/01-money-and-payments.md)).
- **Every mutation is auditable:** who did what, when, with which
  request ID, reconstructed from the ledger alone.
- **The F-word discipline:** every write path answers "what if the
  process dies right here?" with a recovery, not a hope
  ([25 Section 3](../25-fintech-with-go/03-integrity-and-exactly-once.md)).

## Shared additions to the gates

- A property test that asserts ledger conservation: sum of debits ==
  sum of credits, after every scenario including failures and
  retries ([25 Section 2](../25-fintech-with-go/02-double-entry-ledger.md)).
- Idempotency tests are exhaustive: same key twice, different key
  same payload, concurrent same-key submissions.
- A reconciliation check runs in CI: synthetic day of activity,
  night job balances it to zero difference.

## Project 1: Payment processing system

**Goal:** accept a payment request, execute it against a simulated
PSP, record it atomically, emit events: with idempotency end to end.

| Milestone | Done when |
|---|---|
| M1: idempotency layer | the four-step commit ordering from [24 Section 3](../24-system-design/03-payment-service-and-ledger.md): claim key → transactional commit (+outbox) → PSP call with its own idempotency → finalize |
| M2: unknown-response discipline | PSP timeout after debit: the response is "processing", the reconciliation job decides; no double charges, by test |
| M3: retries | payment retries are safe by construction (keyed), PSP-side and ledger-side both |
| M4: events | authorized/captured/failed events on the Level-3 Kafka backbone; consumers are downstreamLedger-safe |
| M5: proof | the unknown-response test suite: every ambiguous PSP outcome maps to an eventual, correct, single state |

**The lesson:** the M2 milestone is the whole section: distributed
payments are reconciliation systems that occasionally take payments.
Internalize it and [25 Section 3](../25-fintech-with-go/03-integrity-and-exactly-once.md)
stops being a chapter and becomes a reflex.

## Project 2: Financial ledger

**Goal:** a double-entry ledger service: accounts, postings,
transfers, statements: the [25 Section 2] example productionized.

| Milestone | Done when |
|---|---|
| M1: the ledger core | immutable postings, balanced transfers in one transaction, integer minor units ([25 Section 2](../25-fintech-with-go/02-double-entry-ledger.md)) |
| M2: concurrent transfers | contention test: N workers transfer between overlapping accounts; conservation holds with `-race` |
| M3: statements | point-in-time balances and statement reads that do not block writes (isolation level chosen per read, with a reason: [13 Section 2](../13-databases/02-transactions-and-isolation.md)) |
| M4: audit trail | every posting carries actor, request ID, and correlation ID; a query reconstructs any account's history |
| M5: proof | the conservation property test runs against randomized transfer graphs including injected failures; a daily reconciliation closes to zero |

**The lesson:** a ledger is boring on purpose. The engineering value
is in what it refuses: unbalanced transfers, mutating postings,
floats, and silent corrections (corrections are new postings).

## Project 3: Fraud detection pipeline

**Goal:** score transactions for risk in real time with synchronous
hard rules and asynchronous model scoring, feeding explainable
decisions.

| Milestone | Done when |
|---|---|
| M1: hard rules | synchronous velocity/countersign rules at the payment path ([24 Section 7](../24-system-design/07-fraud-and-order-processing.md)) |
| M2: async scoring | model-simulating scorer on the event backbone with its own feature cache ([25 Section 4](../25-fintech-with-go/04-risk-and-compliance.md)) |
| M3: decisions + appeals | verdicts recorded with the contributing signals; a decision is auditable and reversible |
| M4: feedback loop | confirmed fraud feeds features back; the loop's staleness is measured, not assumed |
| M5: proof | an end-to-end test: a fraud pattern injected upstream is caught by rule or model within the SLO window |

**The lesson:** fraud systems are precision-recall products with an
audit requirement. Every M3 verdict you cannot explain is a support
ticket and a regulator's question; explainability is a feature test,
not a hope.

## Project 4: Distributed transaction processor

**Goal:** multi-service workflows (reserve inventory → charge →
reserve shipping) that always converge: sagas with compensation,
driven by events.

| Milestone | Done when |
|---|---|
| M1: saga definition | the order flow as forward actions + compensations, each idempotent, each with a timeout ([15 Section 4](../15-microservices/04-idempotency-sagas-outbox.md)) |
| M2: the orchestrator | state persisted per step; crash at any step resumes correctly (test each step boundary) |
| M3: compensation | failed charge triggers inventory release; compensation failures alert and retry, never silently drop |
| M4: isolation honesty | document what a customer can observe mid-saga (reservations visible); pick and justify the tradeoff ([16 Section 2](../16-distributed-systems/02-replication-and-consistency.md)) |
| M5: proof | kill the orchestrator at every step boundary in a loop; final states are always consistent (charged ⇔ ordered, or fully compensated) |

**The lesson:** sagas trade atomicity for availability, and the M5
step-boundary kill loop is the honesty test: it converts "we handle
failures" into a table of 6 crashes and 6 correct outcomes.

## When this level is done

When you can answer *"how do you know this payment wasn't charged
twice?"* with a test name rather than a sentence. That question, and
its evidence-first answer, is the backbone of fintech interviews
([26 Section 4](../26-go-interview-preparation/04-senior-scenarios.md)) and
of real production reviews alike. The
[capstone](05-capstone-financial-platform.md) assembles all four into
one system.

## Further Reading

- [25-fintech-with-go](../25-fintech-with-go/README.md): the domain section every project here leans on
- [24 Section 3](../24-system-design/03-payment-service-and-ledger.md) and [24 Section 7](../24-system-design/07-fraud-and-order-processing.md): the design walkthroughs these projects implement
