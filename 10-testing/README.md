# 10 · Testing

Go made testing a first-class language feature: one command, one
convention, zero configuration. This section covers the discipline on
top of the machinery — what to test, how to structure tests that survive
refactors, and when each layer (unit, integration, e2e) earns its cost.

## Objectives

By the end of this section you can:

- Write table-driven tests that reviewers trust
- Choose the right test double (fake, stub, mock) for each dependency
- Test HTTP handlers with `httptest` without running servers
- Structure integration tests that skip cleanly without infrastructure
- Benchmark, measure coverage honestly, fuzz inputs, and run property-style tests

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Fundamentals](01-fundamentals.md) | Table-driven tests, subtests, helpers, t.Cleanup |
| 2 | [Test doubles & httptest](02-doubles-and-httptest.md) | Fakes, stubs, mocks, HTTP handler testing |
| 3 | [Integration & e2e](03-integration-and-e2e.md) | Tiers, build tags, skip patterns |
| 4 | [Benchmarks, coverage, fuzzing](04-benchmarks-coverage-fuzzing.md) | The measurement toolkit |

## Examples

Every chapter's code lives in `examples/calculator/` — one package with
a full, progressive test suite:

- `calc.go` — the code under test (a tiny expression evaluator)
- `calc_test.go` — table-driven unit tests, subtests, helpers
- `api.go` / `api_test.go` — an HTTP layer tested with httptest
- `bench_test.go` — benchmarks with allocations
- `fuzz_test.go` — fuzz and property-style tests
- `integration_test.go` — integration tier behind a build tag

Run it all:

```bash
go test ./10-testing/...          # unit tier
go test ./10-testing/... -bench=. -benchmem
go test ./10-testing/... -tags=integration
```

## Progress checklist

- [x] Unit testing
- [x] Table-driven tests
- [x] Subtests
- [x] Test helpers and fixtures
- [x] Test doubles: fakes, stubs, mocks
- [x] httptest
- [x] Integration testing
- [x] End-to-end testing
- [x] Benchmarking
- [x] Coverage
- [x] Race detection
- [x] Fuzzing
- [x] Property-style testing
