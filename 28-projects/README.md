# 28 · Projects

**Status: in depth.** Five level guides, 18 projects, one capstone.
Guides are written one per level: each walks its projects' common
rules, build order, and gates, with the individual projects as
milestones inside the guide. Every project exercises named handbook
sections: the patterns are already explained and tested there; the
guides organize them into builds.

## The levels

| Level | Guide | Projects | The skill it builds |
|---|---|---|---|
| 1 | [Foundations](01-level-1-foundations.md) | CLI tool, URL shortener, REST API, file processor | fundamentals under real constraints: stdlib only |
| 2 | [Backend patterns](02-level-2-backend-patterns.md) | task queue, crawler, rate limiter, Redis-backed API, auth service | bounded resources, degradation, identity |
| 3 | [Distributed systems](03-level-3-distributed-systems.md) | event-driven orders, Kafka pipeline, job scheduler, notifications, metrics platform | delivery semantics, leases, honest 202s |
| 4 | [FinTech-grade integrity](04-level-4-fintech-integrity.md) | payment processing, financial ledger, fraud pipeline, distributed transactions | money never moves twice |
| 5 | [Capstone](05-capstone-financial-platform.md) | production-grade financial transaction platform | everything, assembled under real tensions |

## The two rules that hold across all levels

1. **Gates before glory.** Every level's guide defines shared gates
   (`gofmt`, `go vet`, `-race` tests, property tests where money is
   involved) that outrank feature completeness. A project that
   "works" without its gates is not done.
2. **Honesty is a deliverable.** Every README states what is lost on
   crash, what degrades when dependencies die, and what is *not*
   handled. Hidden gaps fail review here and in production alike.

Levels build on each other; each guide's "when this level is done"
section is the exit test. The capstone's acceptance criteria are
designed to be verifiable by an outside engineer: that is the
standard the whole ladder aims at.

## Where to start

- New to Go: begin at [Level 1](01-level-1-foundations.md) after
  sections [01](../01-go-fundamentals/README.md)-[05](../05-errors/README.md).
- Backend engineer new to Go: [Level 2](02-level-2-backend-patterns.md)
  after [12](../12-http-networking/README.md)-[14](../14-backend-development/README.md).
- Practiced Go engineer: skim Levels 1-2, start
  [Level 3](03-level-3-distributed-systems.md), and treat Levels 4-5
  as the main event.
