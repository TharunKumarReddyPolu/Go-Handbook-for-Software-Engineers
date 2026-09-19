# 27 · Open Source

**Status: in depth.** Four chapters on the contribution craft, plus
three special sections on the destinations that matter most to Go
engineers. This repository's own [CONTRIBUTING.md](../CONTRIBUTING.md)
and [PR template](../.github/pull_request_template.md) are the worked
example the chapters refer to.

## Chapters

1. **[Finding the right repository & first issue](01-finding-repo-and-first-issue.md)**:
   the health-signal table, choosing the first issue, ask-before-build
2. **[Understanding a Go codebase fast](02-understanding-codebases.md)**:
   behavior-first reading, tests as spec, tooling-assisted blast radius
3. **[Contributing well: the issue → PR → review loop](03-contributing-well.md)**:
   good issues, PRs that merge, review in both directions, CLA/DCO
4. **[Maintainer communication & long-term contribution](04-maintainer-communication.md)**:
   negotiating scope, handling no, the reliability habits that compound

## Special sections

- **[Contributing to Go](special-contributing-to-go.md)**: the official
  process: CLA, Gerrit, `git-codereview`, the commit format, proposals,
  and the compatibility promise as review context
- **[Contributing to the Kafka-in-Go ecosystem](special-kafka-ecosystem.md)**:
  the three contribution layers, KIP tracking, testcontainers and
  harness work, building on [18-kafka-with-go](../18-kafka-with-go/README.md)
- **[Contributing to Go-based infrastructure](special-infrastructure.md)**:
  Kubernetes SIGs and KEPs, etcd's review depth, `client_golang` as
  the approachable high-impact target

## The through-line

Open-source skill is not generosity; it is engineering with strangers:
same rigor, added communication. The chapters teach the loop once
([§3](03-contributing-well.md)); the special sections show how the
loop flexes when the destination is Gerrit, a KIP, or a SIG.

| If you want to... | Read |
|---|---|
| Land your first merged PR anywhere | 01 → 02 → 03 |
| Contribute to the Go project itself | 03, then the Go special section |
| Contribute to Kafka clients or tooling | 03, then the Kafka special section |
| Grow into a trusted, long-term contributor | 04, then the infrastructure special section |
