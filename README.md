<div align="center">
  <img width="1608" height="483" alt="go-handbook" src="https://github.com/user-attachments/assets/ba12a0dd-6253-469c-ab63-950a73347b46" />
</div>

# Go Handbook for Software Engineers

**A practical, engineering-focused Go handbook: from the fundamentals to production-grade, distributed, and financial systems.** Every runnable example compiles and is verified by CI on every push.

[![CI](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/actions/workflows/ci.yml/badge.svg)](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/go1.27)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)](CONTRIBUTING.md)

## Start here

```bash
git clone https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers.git
cd Go-Handbook-for-Software-Engineers
go test ./08-concurrency/... -race   # see the examples run
```

Read the Markdown in any editor. Sections marked **[In depth]** are fully
written; the rest have outlines tracked in [ROADMAP.md](ROADMAP.md).

## The sections

| # | Section | Status |
|---|---|---|
| 01 | [Go Fundamentals](01-go-fundamentals/) | ✅ In depth |
| 02 | [The Go Language](02-go-language/) | ✅ In depth |
| 03 | [Data Structures](03-data-structures/) | ✅ In depth |
| 04 | [Functions, Methods & Interfaces](04-functions-methods-interfaces/) | ✅ In depth |
| 05 | [Error Handling](05-errors/) | ✅ In depth |
| 06 | [Packages & Modules](06-packages-modules/) | ✅ In depth |
| 07 | [Generics](07-generics/) | ✅ In depth |
| 08 | [Concurrency](08-concurrency/) | ✅ In depth |
| 09 | [Memory & Runtime](09-memory-runtime/) | 📝 Outline |
| 10 | [Testing](10-testing/) | ✅ In depth |
| 11 | [Tooling](11-tooling/) | 📝 Outline |
| 12 | [HTTP & Networking](12-http-networking/) | ✅ In depth |
| 13 | [Databases](13-databases/) | ✅ In depth |
| 14 | [Backend Development](14-backend-development/) | ✅ In depth |
| 15 | [Microservices](15-microservices/) | ✅ In depth |
| 16 | [Distributed Systems](16-distributed-systems/) | ✅ In depth |
| 17 | [Messaging](17-messaging/) | ✅ In depth |
| 18 | [Kafka with Go](18-kafka-with-go/) | ✅ In depth |
| 19 | [Performance](19-performance/) | ✅ In depth |
| 20 | [Observability](20-observability/) | ✅ In depth |
| 21 | [Security](21-security/) | ✅ In depth |
| 22 | [Production Go](22-production-go/) | ✅ In depth |
| 23 | [Go Internals](23-go-internals/) | 📝 Outline |
| 24 | [System Design with Go](24-system-design/) | 📝 Outline |
| 25 | [FinTech with Go](25-fintech-with-go/) | ✅ In depth |
| 26 | [Go Interview Preparation](26-go-interview-preparation/) | ✅ In depth |
| 27 | [Open Source](27-open-source/) | 📝 Outline |
| 28 | [Projects](28-projects/) | 📝 Outline |

Progress and the writing order live in [ROADMAP.md](ROADMAP.md).

## How to read it

Pick the row that matches you, read the sections in order:

| You are... | Read in this order |
|---|---|
| New to Go | 01 → 02 → 03 → 04 → 05 → 10 → 08 |
| Coming from Java/Python/C++/JS | [01 §8](01-go-fundamentals/08-coming-from-other-languages.md) → 04 → 05 → 02 → 07 → 08 |
| A backend engineer | 12 → 13 → 05 → 14 → 10 → 20 → 22 → 21 |
| Building distributed systems | 08 → 15 → 16 → 17 → 18 → 19 → 24 |
| Building fintech systems | 25 → 16 → 18 → 21 → 28 |
| Preparing for interviews | 26 → 08 → 19 → 09 → 23 → 24 |
| Contributing to open source | 27 → [CONTRIBUTING.md](CONTRIBUTING.md) → 11 → 06 |

## What makes this handbook different

- **Depth before breadth.** A topic is not covered until it answers: what is
  it, why does it exist, how does it work, when should I *not* use it, what
  goes wrong in production, and how would I debug it?
- **Few, deep chapters.** Sections ship 4-7 chapters, not a chapter per
  topic: related topics fold into one chapter, and a topic another section
  already owns is a cross-reference, never a duplicate. One worked example
  per section, with tests that pin its claims.
- **Everything compiles.** Every example package passes `go build`, `go vet`,
  `go test -race`, and staticcheck in CI on every push. Tests skip cleanly
  when optional infrastructure (Kafka, Postgres) is absent.
- **No copying.** Original explanations; official documentation is linked
  under *Further Reading* at the end of each chapter, never reproduced.

Go changes every six months. Version-dependent claims are stamped
("Introduced in Go X"); the convention is explained in
[meta/versioning.md](meta/versioning.md).

## Contributing

Contributions are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md) for the
chapter template and the verification commands every PR must pass. By
participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

---

## ⭐ Support the Project

If this handbook helps you on your path to mastering Go, please consider:

<div align="center">

[![Star this repo](https://img.shields.io/badge/⭐%20Star%20this%20repo-important?style=for-the-badge)](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/stargazers)
[![Watch this repo](https://img.shields.io/badge/👁%20Watch%20this%20repo-informational?style=for-the-badge)](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/subscription)
[![Fork this repo](https://img.shields.io/badge/🍴%20Fork%20this%20repo-success?style=for-the-badge)](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/fork)

</div>

---

<div align="center">

# **Learn Go smarter, not harder!** 🧠

</div>

> **Inspired by practical production engineering** with key insights from official sources like go.dev, Effective Go, the Go Blog, and real-world systems experience. **Happy Coding! 🚀**
