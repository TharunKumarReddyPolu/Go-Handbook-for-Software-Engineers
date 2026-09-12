# Roadmap

The handbook grows in phases. This file tracks what is done, what is next,
and in what order remaining sections will be written. Progress checklists
also live inside each section's README; this file is the top-level view.

Last updated: 2026-09-12

## Phase 1: Fundamentals

- [x] 01-go-fundamentals: complete (7 chapters, compiled examples, transition guide)
- [x] 02-go-language: complete (8 chapters, seqops example package with tests)
- [x] 03-data-structures: complete (9 chapters, dsbench + ring + lru examples with tests)
- [x] 04-functions-methods-interfaces: complete (7 chapters, design clinic + di example)

## Phase 2: Core Go

- [x] 05-errors: complete (2 chapters, examples with tests)
- [ ] 06-packages-modules: outline ready
- [ ] 07-generics: outline ready (Go 1.27 generic methods noted)

## Phase 3: Concurrency

- [x] 08-concurrency: complete (8 chapters, examples 1→5 with tests)

## Phase 4: Testing & Tooling

- [x] 10-testing: complete (4 chapters, compiled test suites)
- [ ] 11-tooling: outline ready

## Phase 5: Backend

- [ ] 12-http-networking: outline ready
- [ ] 13-databases: outline ready
- [ ] 14-backend-development: outline ready

## Phase 6: Distributed Systems

- [ ] 15-microservices: outline ready
- [ ] 16-distributed-systems: outline ready
- [ ] 17-messaging: outline ready
- [x] 18-kafka-with-go: complete (3 chapters + client comparison + examples with skip-if-no-broker tests)

## Phase 7: Performance

- [x] 19-performance: complete (4 chapters with before/after benchmarks)

## Phase 8: Security

- [ ] 21-security: outline ready
- [ ] 20-observability: outline ready (planned alongside 21)

## Phase 9: Production

- [ ] 22-production-go: outline ready
- [ ] 23-go-internals: outline ready
- [ ] 24-system-design: outline ready

## Phase 10: FinTech

- [x] 25-fintech-with-go: complete (4 chapters + tested ledger example)

## Phase 11: Interview, Open Source, Capstone

- [x] 26-go-interview-preparation: complete (tiered tracks + scenarios)
- [ ] 27-open-source: outline ready
- [ ] 28-projects: outline with five levels; project guides come last so they can reference finished sections
- [ ] 09-memory-runtime: outline ready (fits best after 23-go-internals drafting starts)

## Planned order for remaining sections

1. 02-go-language (done), 03-data-structures (done), 04-functions-methods-interfaces (done: Phase 1 finished)
2. 06-packages-modules, 07-generics
3. 12-http-networking, 13-databases, 14-backend-development
4. 09-memory-runtime (pairs with 19-performance)
5. 15-microservices, 16-distributed-systems, 17-messaging
6. 20-observability, 21-security, 22-production-go
7. 11-tooling, 23-go-internals, 24-system-design
8. 27-open-source, 28-projects (project guides reference finished sections)

## Standing work

- [ ] Keep version stamps current with each Go release (see meta/versioning.md)
- [ ] Re-run govulncheck weekly (CI scheduled)
- [ ] Periodic link and terminology audit (GLOSSARY.md is the source of truth)
- [ ] Reader feedback: recurring confusion points become new "Common Mistakes" entries
