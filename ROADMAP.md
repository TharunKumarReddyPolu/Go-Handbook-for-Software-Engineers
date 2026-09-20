# Roadmap

The handbook grows in phases. This file tracks what is done, what is next,
and in what order remaining sections will be written; this file is the
top-level view.

Last updated: 2026-09-19

## Repository at a glance

All 28 sections are fully written; this table is the single-glance
index. Chapter counts are Markdown chapters (the section README not
counted); example packages are the compiled, CI-tested Go packages
that back each section.

| # | Section | Chapters | Example packages |
|---|---|---|---|
| 01 | [Go Fundamentals](01-go-fundamentals/) | 8 | `examples/hello`, `examples/zero` |
| 02 | [The Go Language](02-go-language/) | 8 | `examples/seqops` |
| 03 | [Data Structures](03-data-structures/) | 9 | `examples/dsbench`, `examples/heap`, `examples/lru`, `examples/ring` |
| 04 | [Functions, Methods & Interfaces](04-functions-methods-interfaces/) | 7 | `examples/di` |
| 05 | [Error Handling](05-errors/) | 2 | `examples/errorslib`, `examples/service` |
| 06 | [Packages & Modules](06-packages-modules/) | 6 | - |
| 07 | [Generics](07-generics/) | 6 | `examples/genlib` |
| 08 | [Concurrency](08-concurrency/) | 8 | `examples/01-toy` through `examples/05-jobprocessor` (5-stage progression) |
| 09 | [Memory & Runtime](09-memory-runtime/) | 5 | `examples/memwatch` |
| 10 | [Testing](10-testing/) | 4 | `examples/calculator` |
| 11 | [Tooling](11-tooling/) | 4 | - |
| 12 | [HTTP & Networking](12-http-networking/) | 5 | `examples/api` |
| 13 | [Databases](13-databases/) | 5 | `examples/bank` |
| 14 | [Backend Development](14-backend-development/) | 5 | `examples/service` (7 internal packages, incl. observability wiring) |
| 15 | [Microservices](15-microservices/) | 5 | `examples/resilience` |
| 16 | [Distributed Systems](16-distributed-systems/) | 5 | `examples/lease` |
| 17 | [Messaging](17-messaging/) | 4 | `examples/messaging` |
| 18 | [Kafka with Go](18-kafka-with-go/) | 4 | `examples` (franz-go service; skips cleanly without a broker) |
| 19 | [Performance](19-performance/) | 4 | `examples/channels-vs-mutex`, `examples/join`, `examples/profiling` |
| 20 | [Observability](20-observability/) | 5 | lives inside `14/examples/service` (`obshttp`, `otelwiring`) |
| 21 | [Security](21-security/) | 5 | `examples/secure` |
| 22 | [Production Go](22-production-go/) | 6 | - |
| 23 | [Go Internals](23-go-internals/) | 7 | `examples/internals` |
| 24 | [System Design with Go](24-system-design/) | 7 | - (design walkthroughs; cites other sections' examples) |
| 25 | [FinTech with Go](25-fintech-with-go/) | 4 | `examples/ledger` |
| 26 | [Go Interview Preparation](26-go-interview-preparation/) | 4 | - |
| 27 | [Open Source](27-open-source/) | 7 | - (4 chapters + 3 special sections) |
| 28 | [Projects](28-projects/) | 5 | - (5 level guides; the capstone assembles the examples above) |

Totals: **154 chapters across 28 sections, 20 with tested example
packages of their own**, one root Go module: `go build ./...` and
`go test ./... -race` cover every example in every section.

## How sections are written

Sections are planned lean and written leaner: **few, deep chapters**, not
a chapter per topic. The working rules, applied since section 13 and now
stated in every outline:

1. Related topics fold into one chapter (transactions into pooling,
   channels with maps, rate limiter with the gateway) when they only make
   sense side by side.
2. A topic another section already owns is a cross-reference, never a
   duplicate (outbox mechanics live in 25 Section 3; pprof in 19 Section 1; delivery
   semantics in 18 Section 1).
3. One worked example per section, not per chapter, with tests that pin
   the chapter's claims.
4. Chapter counts stay in the 4-7 range; an outline that plans more gets
   consolidated before writing starts.

Sections written before this rule was made explicit (01-12) stand as
written; they are revisited opportunistically, not rewritten wholesale.

## Phase 1: Fundamentals

- [x] 01-go-fundamentals: complete (7 chapters, compiled examples, transition guide)
- [x] 02-go-language: complete (8 chapters, seqops example package with tests)
- [x] 03-data-structures: complete (9 chapters, dsbench + ring + lru examples with tests)
- [x] 04-functions-methods-interfaces: complete (7 chapters, design clinic + di example)
- [x] 09-memory-runtime: complete (5 chapters: stack & heap, escape analysis, GC, scheduler, leak taxonomy + memwatch example)

## Phase 2: Core Go

- [x] 05-errors: complete (2 chapters, examples with tests)
- [x] 06-packages-modules: complete (6 chapters: design, modules, multi-repo, hygiene, reproducible builds, semver)
- [x] 07-generics: complete (6 chapters, genlib example with instantiation-table tests, Go 1.27 baseline)

## Phase 3: Concurrency

- [x] 08-concurrency: complete (8 chapters, examples 1→5 with tests)

## Phase 4: Testing & Tooling

- [x] 10-testing: complete (4 chapters, compiled test suites)
- [x] 11-tooling: complete (4 chapters: professional workflow, pprof & trace in practice, Delve, supply chain & modernizers)

## Phase 5: Backend

- [x] 12-http-networking: complete (5 chapters: routing, middleware, JSON, clients, shutdown + runnable stdlib-only API example)
- [x] 13-databases: complete (5 chapters: database/sql, transactions, pooling, repositories, caching + two-tier bank example)
- [x] 14-backend-development: complete (5 chapters: layout, config, wiring, authz, observability seams + layered service example; backend phase finished)

## Phase 6: Distributed Systems

- [x] 15-microservices: complete (5 chapters: progression, contracts, resilience quartet, idempotency/sagas/outbox, when-NOT + storm-tested resilience example)
- [x] 16-distributed-systems: complete (5 chapters: failure model/CAP, replication & consistency, leases & fencing, quorums & sharding, delivery & backpressure + zombie-leader lease example)
- [x] 17-messaging: complete (4 chapters: queue/log/pubsub models, broker comparison, portable patterns, schemas & testing + deterministic in-memory broker example)
- [x] 18-kafka-with-go: complete (3 chapters + client comparison + examples with skip-if-no-broker tests)

## Phase 7: Performance

- [x] 19-performance: complete (4 chapters with before/after benchmarks)

## Phase 8: Security

- [x] 21-security: complete (5 chapters: threat modeling & validation, authn/authz mechanics, TLS & certificates, limits & hardening, secrets & supply chain + secure example with deterministic tests)
- [x] 20-observability: complete (5 chapters: logging, metrics, tracing/OTel, SLOs & alerting, incident debugging; instrumented inside the 14 service example)

## Phase 9: Production

- [x] 22-production-go: complete (6 chapters: config & secrets, server lifecycle, resource limits, dependency degradation, containers/Kubernetes, releases & rollbacks)
- [x] 23-go-internals: complete (7 chapters: compiler pipeline, SSA & optimizations with real flag transcripts, runtime architecture, channels & maps, interfaces/slices/strings, memory model, reflection & assembly + AllocsPerRun-tested example)
- [x] 24-system-design: complete (7 paired walkthroughs: URL shortener; rate limiter & gateway; payment & ledger; notifications; scheduler & cache; events & analytics; fraud & orders)

## Phase 10: FinTech

- [x] 25-fintech-with-go: complete (4 chapters + tested ledger example)

## Phase 11: Interview, Open Source, Capstone

- [x] 26-go-interview-preparation: complete (tiered tracks + scenarios)
- [x] 27-open-source: complete (4 chapters: repo & first-issue selection, codebase reading, the contribution loop, maintainer communication + 3 special sections: contributing to Go (Gerrit/CLA/proposals), the Kafka-in-Go ecosystem (KIPs, tooling layers), Go-based infrastructure (Kubernetes/etcd/Prometheus))
- [x] 28-projects: complete (5 level guides, 18 projects as milestones: foundations, backend patterns, distributed systems, fintech integrity + the Level 5 capstone: production-grade financial transaction platform with six milestones and outside-verifiable acceptance criteria)

**The roadmap is complete: all 28 sections are written.** Remaining work is
standing maintenance (below) and opportunistic improvement.

## Standing work

- [ ] Keep version stamps current with each Go release (see meta/versioning.md)
- [ ] Re-run govulncheck weekly (CI scheduled)
- [ ] Periodic link and terminology audit (GLOSSARY.md is the source of truth)
- [ ] Reader feedback: recurring confusion points become new "Common Mistakes" entries
