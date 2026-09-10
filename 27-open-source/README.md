# 27 · Open Source

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** This repository's own
[CONTRIBUTING.md](../CONTRIBUTING.md) doubles as a worked example of
the standards below.

## Planned chapters

1. **Finding the right repository**: evaluating activity, governance,
   maintainer responsiveness (the criteria used in
   [18 §2](../18-kafka-with-go/02-go-clients.md))
2. **Understanding a Go codebase fast**: package layout reading,
   `go doc`, tests as documentation, starting from failing behavior
3. **Finding beginner-friendly issues**: labels, issue quality signals,
   asking before building
4. **Contribution guidelines & CLAs**: what they actually govern
5. **Writing good issues**: reproductions, versions, evidence
6. **Writing good PRs**: small scope, tests included, description
   that explains *why* (the checklist in
   [pull_request_template.md](../.github/pull_request_template.md))
7. **Git workflow**: branches, focused commits, rebasing etiquette
8. **Code review**: giving and receiving; the review loop from this
   repo's [CONTRIBUTING](../CONTRIBUTING.md)
9. **Maintainer communication**: negotiating scope, handling "no",
   long-term contributor habits

## Special sections

- **Contributing to Go**: the Go project's own process: proposals,
  Gerrit (not GitHub PRs), CLA, the compatibility promise as review
  context; linking the official contribution guide and proposal process.
- **Contributing to Kafka with Go**: the Go-Kafka ecosystem's OSS
  surface: franz-go/sarama/segmentio clients, kafka-python-style
  ecosystem tools, testcontainers modules, and how the
  [18-kafka-with-go](../18-kafka-with-go/README.md) examples model the
  PRs those projects actually merge (docs, features, KIP tracking).
- **Contributing to Go-based infrastructure**: Kubernetes, etcd,
  Prometheus: their SIGs, their review cultures, and what transfers
  from smaller projects.
