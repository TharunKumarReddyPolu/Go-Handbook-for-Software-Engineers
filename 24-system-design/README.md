# 24 · System Design with Go

**Status: in depth.** Seven design walkthroughs, each following the
same template: Requirements → APIs → Data model → Architecture →
Scaling → Reliability → Failure modes → Observability → Security →
**Go implementation considerations**. The differentiator is that
last section: every design cites the handbook's own tested examples
and explains how they compose at design scale.

## The designs

| # | Design | The lesson |
|---|---|---|
| 1 | [URL shortener](01-url-shortener.md) | the warm-up: ID generation, read-heavy caching, the open-redirector trap |
| 2 | [Rate limiter & API gateway](02-rate-limiter-and-gateway.md) | the edge: shared buckets, auth-then-limit-then-route, shed-before-collapse |
| 3 | [Payment service & ledger](03-payment-service-and-ledger.md) | the flagship: idempotency, double-entry, outbox, the unknown-PSP-response discipline |
| 4 | [Notification system](04-notification-system.md) | fan-out under vendor limits: priority lanes, dedup layers, honest 202s |
| 5 | [Job scheduler & distributed cache](05-job-scheduler-and-distributed-cache.md) | ownership: SQL leases with SKIP LOCKED, fencing, consistent hashing, singleflight |
| 6 | [Event processing & analytics](06-event-processing-and-analytics.md) | streams: per-key ordering, windowed aggregation, offsets-after-upsert |
| 7 | [Fraud & order processing](07-fraud-and-order-processing.md) | the commerce flow: sagas plus an async risk gate on one event backbone |

## Why paired designs

Designs pair when their tradeoffs only make sense side by side:
the limiter without a gateway has nothing to protect; analytics
without an event backbone has nothing to consume; fraud without
orders has nothing to gate. The URL shortener stays solo as the
teachable warm-up; the payment service stays solo at the center of
the section because it is the handbook's spine ([25](../25-fintech-with-go/README.md)).

## How to use this section

- **Interview prep**: each design is a rehearsal grid: the tables
  are the answer skeleton, the Go sections are the differentiator
  ([26 §4](../26-go-interview-preparation/04-senior-scenarios.md)'s
  scenario drills use them).
- **Building**: the Go implementation considerations are the
  build order: start from the cited, tested example; add the
  production extensions the design names.
- **Reading order**: 1 → 2 → 3 builds the vocabulary; 4-7 reuse
  it. Cross-references do the heavy lifting: nothing here
  re-explains what [25 §3](../25-fintech-with-go/03-integrity-and-exactly-once.md)
  or [16 §3](../16-distributed-systems/03-leader-election-leases-fencing.md)
  already proved with tested code.
