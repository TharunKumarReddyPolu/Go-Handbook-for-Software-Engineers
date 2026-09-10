# Resources

A curated list. Every link is authoritative or high quality; nothing is here
to pad the count. Official sources come first in each category.

## Official Go

- [Go documentation hub](https://go.dev/doc/) — start here for everything official
- [A Tour of Go](https://go.dev/tour/) — interactive syntax introduction
- [Effective Go](https://go.dev/doc/effective_go) — style foundations (read critically; some advice predates modules and generics)
- [Go FAQ](https://go.dev/doc/faq) — the language designers answering why
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) — the community's collected review guidance
- [Go Style Guide (Google)](https://google.github.io/styleguide/go/) — deep, opinionated, widely respected
- [Go release history](https://go.dev/doc/devel/release) — the source of truth for version-stamped claims
- [Go modules reference](https://go.dev/ref/mod)
- [Go memory model](https://go.dev/ref/mem)
- [Go project repository](https://github.com/golang/go) — issues and proposal discussions are engineering education

## Standard library highlights

- [net/http](https://pkg.go.dev/net/http) — read the package docs fully once; they answer half of all HTTP questions
- [database/sql](https://pkg.go.dev/database/sql)
- [context](https://pkg.go.dev/context)
- [testing](https://pkg.go.dev/testing) — including benchmarks and fuzzing
- [runtime/pprof](https://pkg.go.dev/runtime/pprof) and [net/http/pprof](https://pkg.go.dev/net/http/pprof)

## Tools

- [govulncheck](https://go.dev/blog/vuln) — vulnerability detection that understands call graphs
- [staticcheck](https://staticcheck.dev/) — the static analysis standard
- [Delve](https://github.com/go-delve/delve) — the Go debugger
- [gopls](https://github.com/golang/tools/tree/master/gopls) — the language server behind every good editor setup
- [pprof](https://github.com/google/pprof) — profile visualization

## Books

- *The Go Programming Language* — Donovan & Kernighan. Still the best written introduction; predates modules and generics.
- *Concurrency in Go* — Kennedy. The strongest available treatment of channels, context, and patterns.
- *100 Go Mistakes and How to Avoid Them* — Teboul. Practical, production-flavored, worth reading twice.
- *Go in Action* — Kennedy, Ketelsen, St. Martin. Good second book.

## Blogs and talks

- [The Go Blog](https://go.dev/blog/) — posts on internals and features by the team
- Dave Cheney — [dave.cheney.net](https://dave.cheney.net). *Practical Go* and error-handling essays are canonical
- [Aphyr's posts on distributed systems](https://aphyr.com/) — not Go, essential for Path 4
- "Go Concurrency Patterns" — Rob Pike, Google I/O 2012; and "Advanced Go Concurrency Patterns" — Sameer Ajmani, Google I/O 2013. Both on YouTube; watch after section 08
- Jon Gjengset's Rust/Systems content for deep-dive methodology (transferable, not Go-specific)

## Kafka

- [Apache Kafka documentation](https://kafka.apache.org/documentation/) — the semantics chapter is mandatory reading
- [Kafka: The Definitive Guide, 2nd ed.](https://www.confluent.io/resources/kafka-the-definitive-guide/) — free from Confluent with registration
- [franz-go](https://github.com/twmb/franz-go) — modern, feature-complete Go client
- [IBM/sarama](https://github.com/IBM/sarama) — the established classic Go client
- [segmentio/kafka-go](https://github.com/segmentio/kafka-go) — simple, readable client
- [Confluent's Go client](https://github.com/confluentinc/confluent-kafka-go) — librdkafka wrapper; compare before choosing

## Databases

- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [database/sql tutorial](http://go-database-sql.org/) — community, accurate, focused
- [pgx](https://github.com/jackc/pgx) — the Postgres driver most production Go uses
- [sqlc](https://github.com/sqlc-dev/sqlc) — generate type-safe Go from SQL
- [Redis documentation](https://redis.io/docs/latest/)

## Distributed systems

- [Designing Data-Intensive Applications](https://dataintensive.net/) — Kleppmann. The single most recommended book in this field
- [Jepsen analyses](https://jepsen.io/analyses) — how real distributed databases fail
- [The Google File System](https://research.google/pubs/the-google-file-system/), [MapReduce](https://research.google/pubs/mapreduce-simplified-data-processing-on-large-clusters/), [Bigtable](https://research.google/pubs/bigtable-a-distributed-storage-system-for-structured-data/) — foundational papers, still readable
- [Raft](https://raft.github.io/) — consensus explained to be understood

## Observability

- [OpenTelemetry documentation](https://opentelemetry.io/docs/)
- [Google's SRE books](https://sre.google/books/) — free; the chapters on SLOs and alerting underpin section 20
- [Prometheus documentation](https://prometheus.io/docs/)

## Security

- [OWASP Top Ten](https://owasp.org/www-project-top-ten/)
- [Go vulnerability database](https://vuln.go.dev/)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) — password hashing, correct primitives
- [RFC 9110](https://httpwg.org/specs/rfc9110.html) — HTTP semantics, the standard behind net/http

## FinTech engineering

- [Stripe's API idempotency documentation](https://docs.stripe.com/api/idempotent_requests) — the industry reference design
- [Martín Kleppmann on exactly-once delivery](https://www.confluent.io/blog/exactly-once-semantics-are-possible-heres-how-apache-kafka-does-it/) — precise about what "exactly-once" can and cannot mean
- [The canonical paper on event sourcing and CQRS](https://martinfowler.com/eaaDev/EventSourcing.html) — Fowler's write-up is the stable reference
- [PCI DSS overview](https://www.pcisecuritystandards.org/) — regulatory context, not engineering advice

## Interview preparation

- Section 26 of this handbook — start there; it is organized by difficulty
- [Tech Interview Handbook](https://www.techinterviewhandbook.org/) — process and system-design preparation
- [System Design Primer](https://github.com/donnemartin/system-design-primer) — breadth of design topics

## Open source

- [Go contribution guide](https://go.dev/doc/contribute) — how to contribute to Go itself
- [Go proposal process](https://go.dev/s/proposal-process) — how language changes actually happen
- [First contributions](https://github.com/firstcontributions/first-contributions) — mechanics of the PR workflow
- [Kubernetes contributor guide](https://www.kubernetes.dev/docs/guide/) — the largest Go project's conventions
