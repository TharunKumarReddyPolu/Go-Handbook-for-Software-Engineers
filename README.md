# Go Handbook for Software Engineers

A comprehensive Go handbook for software engineers covering fundamentals, idiomatic Go, concurrency, generics, testing, APIs, databases, performance, microservices, system design, and production engineering.

[![CI](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/actions/workflows/ci.yml/badge.svg)](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/go1.27)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)](CONTRIBUTING.md)

This is **not** just another Go syntax tutorial. It is a practical,
engineering-focused handbook for learning Go and building reliable,
scalable, production-ready software.

Most Go material online stops at "here is the syntax." This handbook starts
there and keeps going: how Go behaves under load, why the runtime does what
it does, how to structure a service you can page someone about at 3 a.m.,
and how to design distributed and financial systems with the tools Go gives
you.

Every chapter follows the same contract:

- **Why it exists**: the problem, not just the feature
- **Mental model**: how to think about it before memorizing syntax
- **Compiled examples**: everything labeled runnable compiles and runs; `go test ./...` verifies them in CI
- **Common mistakes**: the bugs this handbook's author has seen (or committed)
- **Production notes**: performance, concurrency, and security implications
- **Interview questions and exercises**, so learning turns into retention

## Table of Contents

- [Who this is for](#who-this-is-for)
- [The philosophy](#the-philosophy)
- [Learning paths](#learning-paths)
- [Quick start](#quick-start)
- [The handbook](#the-handbook)
- [How examples are verified](#how-examples-are-verified)
- [Reading version-specific content](#reading-version-specific-content)
- [Contributing and community](#contributing-and-community)

## Who this is for

| You are... | Start with | Why |
|---|---|---|
| A complete beginner | [Path 1](#path-1-complete-beginner--go-engineer) | Sections 01–03 build the vocabulary; skip nothing in 01 |
| A Java, Python, C++, or JavaScript developer | [Path 2](#path-2-java--python--c--javascript-developer--go-engineer) | Section 01.07 maps your old habits to Go's; unlearn inheritance early |
| A backend engineer new to Go | [Path 3](#path-3-backend-engineer--production-go) | Errors → HTTP → databases → backend structure |
| An experienced Go engineer | [Path 4](#path-4-go-engineer--distributed-systems-engineer) | Concurrency, distributed systems, performance, internals |
| Building fintech or payment systems | [Path 5](#path-5-go-engineer--fintech-engineer) | Ledgers, idempotency, reconciliation, exactly-once myths |
| Preparing for interviews | [Path 7](#path-7-go-engineer--interview-ready) | Section 26, then drills in 08 and 19 |
| Wanting to contribute to open source | [Path 6](#path-6-go-engineer--open-source-contributor) | Section 27 plus the CONTRIBUTING guide of this repo as a worked example |

## The philosophy

The handbook is organized around a deliberate ladder. Each rung depends on
the ones below it:

```text
Learn Go
  ↓
Understand Go
  ↓
Write Idiomatic Go
  ↓
Build Reliable Software
  ↓
Build Production Systems
  ↓
Understand Go Internals
  ↓
Build Distributed Systems
  ↓
Use Go in Real-World Domains
  ↓
Contribute to Open Source
```

Two rules follow from this and are enforced throughout:

1. **Depth before breadth.** A topic is not "covered" until it answers:
   what is it, why does it exist, how does it work, when should I *not*
   use it, what goes wrong in production, and how would I debug it?
2. **No copying.** Explanations are original. Official documentation is
   linked under *Further Reading* at the end of each chapter, never
   reproduced.

## Learning paths

Sections are numbered; paths pick a route through them.

### Path 1: Complete Beginner → Go Engineer

1. [01 Go Fundamentals](01-go-fundamentals/): everything, especially zero values and defer
2. [02 The Go Language](02-go-language/): slices, maps, structs, pointers
3. [03 Data Structures](03-data-structures/): idiomatic implementations
4. [04 Functions, Methods & Interfaces](04-functions-methods-interfaces/)
5. [05 Error Handling](05-errors/): read twice; errors are values here
6. [10 Testing](10-testing/): table-driven tests from day one
7. [08 Concurrency](08-concurrency/): goroutines and channels only
8. [28 Projects](28-projects/): Level 1 projects

### Path 2: Java / Python / C++ / JavaScript Developer → Go Engineer

1. [01 §8 Coming from another language](01-go-fundamentals/08-coming-from-other-languages.md)
2. [04 Functions, Methods & Interfaces](04-functions-methods-interfaces/): unlearn class hierarchies
3. [05 Error Handling](05-errors/): no exceptions, and that changes design
4. [02 The Go Language](02-go-language/): value semantics vs reference semantics
5. [07 Generics](07-generics/), and when *not* to use them
6. [06 Packages & Modules](06-packages-modules/): no more classpath debates
7. [08 Concurrency](08-concurrency/): CSP vs threads-and-locks

### Path 3: Backend Engineer → Production Go

1. [12 HTTP & Networking](12-http-networking/)
2. [13 Databases](13-databases/)
3. [05 Error Handling](05-errors/)
4. [14 Backend Development](14-backend-development/)
5. [10 Testing](10-testing/): httptest and integration tests
6. [20 Observability](20-observability/)
7. [22 Production Go](22-production-go/)
8. [21 Security](21-security/)

### Path 4: Go Engineer → Distributed Systems Engineer

1. [08 Concurrency](08-concurrency/): the patterns chapters
2. [15 Microservices](15-microservices/)
3. [16 Distributed Systems](16-distributed-systems/)
4. [17 Messaging](17-messaging/)
5. [18 Kafka with Go](18-kafka-with-go/)
6. [19 Performance](19-performance/)
7. [24 System Design with Go](24-system-design/)

### Path 5: Go Engineer → FinTech Engineer

1. [25 FinTech with Go](25-fintech-with-go/): start with the exactly-once myths chapter
2. [16 Distributed Systems](16-distributed-systems/): idempotency and delivery semantics
3. [18 Kafka with Go](18-kafka-with-go/): event-driven payment flows
4. [21 Security](21-security/): defensive engineering
5. [28 Projects](28-projects/): Level 4, then the Level 5 capstone

### Path 6: Go Engineer → Open Source Contributor

1. [27 Open Source](27-open-source/): includes contributing to Go itself and to the Go/Kafka ecosystem
2. This repository's [CONTRIBUTING.md](CONTRIBUTING.md) as a worked example
3. [11 Tooling](11-tooling/): the tools every maintainer assumes you know
4. [06 Packages & Modules](06-packages-modules/)

### Path 7: Go Engineer → Interview Ready

1. [26 Go Interview Preparation](26-go-interview-preparation/): tiered by difficulty, scenario-based
2. [08 Concurrency](08-concurrency/): the most common deep-dive topic
3. [19 Performance](19-performance/): profiling stories beat trivia
4. [09 Memory & Runtime](09-memory-runtime/) and [23 Go Internals](23-go-internals/)
5. [24 System Design with Go](24-system-design/)

## Quick start

Install Go 1.27 or later from <https://go.dev/dl/>, then:

```bash
git clone https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers.git
cd Go-Handbook-for-Software-Engineers

# Every labeled-runnable example compiles:
go build ./...

# Every test suite passes, with race detection:
go test ./... -race

# Run one section's examples, e.g. concurrency:
go test ./08-concurrency/... -race -v
```

Read the Markdown in any editor; every code block marked as an example file
has a real file you can open next to it. Chapters label anything that will
not compile (intentionally incomplete snippets) as such.

## The handbook

Sections marked **[In depth]** are fully written with verified examples.
Sections marked **[Outline]** have a complete topic map and are being
written: see [ROADMAP.md](ROADMAP.md) for status and ordering.

### Foundations

- **[01 Go Fundamentals](01-go-fundamentals/)** [In depth]: toolchain, program structure, variables and types, zero values, control flow, defer/panic/recover, coming from other languages
- **[02 The Go Language](02-go-language/)** [Outline]: arrays, slices, maps, strings and runes, structs, pointers, methods, interfaces, embedding
- **[03 Data Structures](03-data-structures/)** [Outline]: built-ins and idiomatic implementations, complexity, memory behavior
- **[04 Functions, Methods & Interfaces](04-functions-methods-interfaces/)** [Outline]: method sets, small interfaces, composition over inheritance, dependency inversion

### Core Go

- **[05 Error Handling](05-errors/)** [In depth]: errors are values, wrapping, sentinel and domain errors, retryability, HTTP mapping, logging
- **[06 Packages & Modules](06-packages-modules/)** [Outline]: package design, visibility, modules, workspaces, reproducible builds
- **[07 Generics](07-generics/)** [Outline]: type parameters, constraints, when generics help and hurt (includes Go 1.27 generic methods)
- **[08 Concurrency](08-concurrency/)** [In depth]: goroutines, channels, context, sync primitives, patterns, pitfalls, race detection; examples progress from toy to production
- **[09 Memory & Runtime](09-memory-runtime/)** [Outline]: stack vs heap, escape analysis, GC, scheduler

### Quality and tooling

- **[10 Testing](10-testing/)** [In depth]: table-driven tests, doubles, httptest, benchmarks, fuzzing, race detection
- **[11 Tooling](11-tooling/)** [Outline]: the professional workflow: gofmt, vet, pprof, delve, govulncheck, staticcheck, gopls

### Backend and APIs

- **[12 HTTP & Networking](12-http-networking/)** [Outline]: stdlib-first HTTP: handlers, middleware, clients, timeouts, TLS, graceful shutdown
- **[13 Databases](13-databases/)** [Outline]: database/sql, Postgres, transactions, pooling, Redis, caching
- **[14 Backend Development](14-backend-development/)** [Outline]: layering, configuration, DI, validation, a complete production-style service

### Distributed systems

- **[15 Microservices](15-microservices/)** [Outline]: boundaries, contracts, resilience patterns, when NOT to use microservices
- **[16 Distributed Systems](16-distributed-systems/)** [Outline]: CAP, replication, delivery semantics, idempotency, backpressure
- **[17 Messaging](17-messaging/)** [Outline]: queues and pub/sub concepts, broker comparisons
- **[18 Kafka with Go](18-kafka-with-go/)** [In depth]: architecture, client tradeoffs, producers, consumers, offsets, reliability, observability

### Production engineering

- **[19 Performance](19-performance/)** [In depth]: measure first: benchmarks, pprof, allocations, contention, PGO
- **[20 Observability](20-observability/)** [Outline]: structured logs, metrics, tracing, OpenTelemetry, SLOs, incident debugging
- **[21 Security](21-security/)** [Outline]: authn/authz, TLS, injection, secrets, govulncheck, fuzzing
- **[22 Production Go](22-production-go/)** [Outline]: config, shutdown, deployment, containers, Kubernetes, rollbacks

### Deep Go

- **[23 Go Internals](23-go-internals/)** [Outline]: compiler pipeline, runtime, scheduler, channels and maps under the hood
- **[24 System Design with Go](24-system-design/)** [Outline]: twelve Go-centric design exercises from URL shortener to fraud pipeline

### Domain and career

- **[25 FinTech with Go](25-fintech-with-go/)** [In depth]: ledgers, double-entry, idempotent payments, reconciliation, exactly-once myths, regulatory notes
- **[26 Go Interview Preparation](26-go-interview-preparation/)** [In depth]: beginner → senior tracks, scenario questions, reasoning-first answers
- **[27 Open Source](27-open-source/)** [Outline]: contributing to Go projects, Go itself, and the Kafka-in-Go ecosystem
- **[28 Projects](28-projects/)** [Outline]: five levels, 20+ projects, ending in the Production-Grade Financial Transaction Platform capstone

## How examples are verified

- One Go module at the repository root: `go build ./...` compiles every
  runnable example in every section.
- `go test ./... -race` runs the test suites in CI on every pull request,
  with shuffled test order to catch hidden dependencies.
- CI also runs `gofmt`, `go vet`, staticcheck, a weekly govulncheck, and a
  dependency-free internal-link checker (`tools/linkcheck`).
- Code that needs external infrastructure (a Kafka broker, a database) is
  written so its tests skip cleanly when the service is absent. See
  [18 §14 Testing against a broker](18-kafka-with-go/README.md) when it
  ships.

## Reading version-specific content

Go changes every six months. This handbook stamps every version-dependent
claim ("Introduced in Go X") and explains the convention in
[meta/versioning.md](meta/versioning.md). The baseline for this handbook is
Go 1.27 (August 2026). Watch for:

- **Go 1.22**: loop variables are per-iteration (the classic closure-capture bug is fixed)
- **Go 1.23**: range-over-function iterators; timer channels made synchronous
- **Go 1.25**: `sync.WaitGroup.Go` helper
- **Go 1.27**: generic methods

If you are on an older toolchain, the language basics still apply exactly;
version-stamped features will not compile.

## Contributing and community

Contributions are welcome and structured: see
[CONTRIBUTING.md](CONTRIBUTING.md) for chapter templates, code standards,
and the verification commands every PR must pass. By participating you
agree to the [Code of Conduct](CODE_OF_CONDUCT.md). Good first
contributions: fixing an inaccuracy (file an issue with the quote), adding
an exercise, improving a diagram.
