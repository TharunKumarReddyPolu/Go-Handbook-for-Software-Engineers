# 27 · Open Source

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** This repository's own
[CONTRIBUTING.md](../CONTRIBUTING.md) doubles as a worked example of
the standards below.## Planned chapters

1. **Finding the right repository & first issue**: evaluating
   activity, governance, and maintainer responsiveness (the criteria
   used in [18 §2](../18-kafka-with-go/02-go-clients.md)); issue
   quality signals, labels, asking before building
2. **Understanding a Go codebase fast**: package layout reading,
   `go doc`, tests as documentation, starting from failing behavior
3. **Contributing well**: good issues, good PRs (small scope, tests
   included, a description that explains *why*: the checklist in
   [pull_request_template.md](../.github/pull_request_template.md)),
   git workflow, code review as giver and receiver, CLAs and what
   they govern
4. **Maintainer communication**: negotiating scope, handling "no",
   long-term contributor habits

## Special sections

- **Contributing to Go**: the Go project's own process: proposals,
  Gerrit (not GitHub PRs), CLA, the compatibility promise as review
  context; linking the official contribution guide and proposal
  process.
- **Contributing to Kafka with Go**: the Go-Kafka ecosystem's OSS
  surface: franz-go/sarama/segmentio clients, ecosystem tools,
  testcontainers modules, and how the
  [18-kafka-with-go](../18-kafka-with-go/README.md) examples model
  the PRs those projects actually merge (docs, features, KIP
  tracking).
- **Contributing to Go-based infrastructure**: Kubernetes, etcd,
  Prometheus: their SIGs, their review cultures, and what transfers
  from smaller projects.

(Consolidated from 9 planned chapters into 4: the issue/PR/review/git
mechanics are one contribution loop, taught once.)
