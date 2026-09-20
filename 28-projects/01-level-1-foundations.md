# Level 1: Foundations

## Why This Level Exists

Level 1 projects have one job: make the fundamentals muscle memory
under real constraints (requirements, gates, a README that documents
decisions). They use almost no infrastructure, so every failure is
your code, and every fix teaches the language rather than a framework.
Each project exercises named handbook sections; the patterns are
already explained, tested, and benchmarked there.

## The level rules

- **Standard library only.** No web frameworks, no config libraries,
  no DI containers. If you feel a third-party dependency is needed,
  that instinct is the project's first lesson (see
  [06 Section 4](../06-packages-modules/04-dependency-hygiene.md)).
- **Every project ships a README** with: what it does, the API or CLI
  surface, the error taxonomy, and the limits you know about.
- **Every project passes the shared gates** below before you call it
  done. "It works" is not done.

## Shared gates

```bash
gofmt -l .                 # empty
go vet ./... && go test ./... -race -shuffle=on
```

- Table-driven tests for every behavior branch
  ([10 Section 1](../10-testing/01-fundamentals.md)).
- Errors are values with a taxonomy: sentinel or typed, wrapped with
  `%w`, classified retryable vs not ([05 Section 1](../05-errors/01-errors-are-values.md),
  [05 Section 2](../05-errors/02-error-design.md)).
- No goroutine you start can outlive its owner: every spawn has a
  stop ([08 Section 1](../08-concurrency/01-goroutines-and-channels.md),
  [09 Section 5](../09-memory-runtime/05-memory-leaks.md)).
- No `_ = err` anywhere. An ignored error is a design decision you
  must defend in the README.

## Project 1: CLI tool

**Goal:** a file-oriented tool (choose: log summarizer, CSV splitter,
timestamp normalizer) driven by flags and stdin/stdout.

| Milestone | Done when |
|---|---|
| M1: skeleton | `flag` parsing, `--help` text, exit codes: 0 ok, 1 usage, 2 runtime ([01 Section 2](../01-go-fundamentals/02-toolchain-and-workflow.md)) |
| M2: core logic | pure functions handle the transformation; unit-tested as a package, independent of `main` |
| M3: streaming | files processed with `bufio.Scanner`, constant memory on a 1 GB input ([02 Section 1](../02-go-language/01-arrays-and-slices.md): no read-all) |
| M4: polish | structured errors to stderr, `defer` hygiene ([01 Section 7](../01-go-fundamentals/07-defer-panic-recover.md)), fuzz the parser for panics ([10 Section 4](../10-testing/04-benchmarks-coverage-fuzzing.md)) |

**The lesson to extract:** command-line behavior is an API contract;
exit codes and stderr format are load-bearing. Document them like any
API.

## Project 2: URL shortener (stdlib HTTP)

**Goal:** create short codes, redirect lookups, in-memory store.
The requirements-and-architecture version lives in
[24 Section 1](../24-system-design/01-url-shortener.md); here you build the
simplest honest version of it.

| Milestone | Done when |
|---|---|
| M1: routing | `net/http` ServeMux patterns, health endpoint ([12 Section 1](../12-http-networking/01-handlers-and-routing.md)) |
| M2: store | an interface `Store` with an in-memory impl; collision handling with retry |
| M3: JSON boundary | create returns 201 + location; validation errors are 4xx with a stable body shape ([12 Section 3](../12-http-networking/03-json-and-rest-apis.md), [05 Section 2](../05-errors/02-error-design.md)) |
| M4: redirects | 302 until you decide permanence, open-redirector trap documented ([24 Section 1](../24-system-design/01-url-shortener.md)) |
| M5: tests | `httptest` end-to-end tests of the API surface ([10 Section 2](../10-testing/02-doubles-and-httptest.md)) |

**The lesson:** the store interface is dependency inversion in
miniature ([04 Section 5](../04-functions-methods-interfaces/05-dependency-inversion.md)):
the handler never learns whether the store is a map or a database.

## Project 3: REST API (stdlib, in-memory)

**Goal:** a small domain (pick one: bookmarks, expenses, inventory)
with CRUD, pagination, and consistent error mapping.

| Milestone | Done when |
|---|---|
| M1: domain first | types and invariants before HTTP exists ([04 Section 6](../04-functions-methods-interfaces/06-constructors-and-invariants.md)); zero values chosen deliberately ([01 Section 5](../01-go-fundamentals/05-zero-values.md)) |
| M2: service layer | business rules in a service, transport as a thin translator |
| M3: error mapping | domain errors → status codes in one table; 401/403/404 discipline if you add identity ([21 Section 2](../21-security/02-authentication-and-authorization.md)) |
| M4: middleware | request ID + logging + panic recovery as composable wrappers ([12 Section 2](../12-http-networking/02-middleware.md)) |
| M5: full suite | table tests per endpoint, concurrent-access test with `-race` |

**The lesson:** the layering from [14 Section 1](../14-backend-development/01-service-layout.md)
in embryo. If your handlers contain business rules, the refactor that
removes them is the real exercise.

## Project 4: File processor (streaming + worker pool)

**Goal:** process many large files (or one huge one) with a bounded
worker pool: parse, transform, emit results.

| Milestone | Done when |
|---|---|
| M1: streaming core | line/chunk pipeline with backpressure: the producer blocks, memory stays flat ([08 Section 2](../08-concurrency/02-select-and-timeouts.md)) |
| M2: fan-out pool | stage-2/3 worker pool with a results channel and ordered or unordered output, your choice documented ([08 Section 5](../08-concurrency/05-patterns.md)) |
| M3: cancellation | context plumbed end to end; Ctrl-C drains cleanly ([08 Section 3](../08-concurrency/03-context.md)) |
| M4: failure semantics | per-item errors collected, not fatal; poison items dead-lettered to a reject file |
| M5: proof | `-race` clean; a benchmark before/after pool sizing ([10 Section 4](../10-testing/04-benchmarks-coverage-fuzzing.md)) |

**The lesson:** backpressure is not an optimization, it is the design.
The unbounded-buffer version that "works" on ten files is the bug the
pool version prevents ([08 Section 7](../08-concurrency/07-pitfalls.md)).

## When this level is done

You can build all four from memory in an evening each, with tests
first, and every gate green without looking anything up. That is the
signal to move to [Level 2](02-level-2-backend-patterns.md).

## Further Reading

- [12-http-networking](../12-http-networking/README.md) and [10-testing](../10-testing/README.md): the two sections this level leans on hardest
- This repository's [CI workflow](../.github/workflows/ci.yml): copy its checks into your project's own CI
