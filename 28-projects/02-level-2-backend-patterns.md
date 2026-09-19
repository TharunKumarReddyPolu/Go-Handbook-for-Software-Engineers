# Level 2: Backend patterns

## Why This Level Exists

Level 2 adds the two things every real backend has: resources that
must be shared safely under load (limits, caches, queues) and
identity (who is calling). The projects here are the ones production
teams actually glue together, and each one isolates a pattern you
will reuse in Levels 3-5.

## The level rules

- Level 1's gates carry forward (tests, error taxonomy, owned
  goroutines, README). A Level 2 project without Level 1 discipline
  is a Level 1 project with more files.
- **Bounded everything.** Queues, caches, connection pools, rate
  buckets: state each bound, and state what happens at the bound.
- Degrade honestly: every project must answer "what do you do when
  the dependency is down?" with a behavior, not a shrug
  ([22 §4](../22-production-go/04-dependency-failures.md)).

## Shared additions to the gates

```bash
go test ./... -race -shuffle=on     # as before
go test -bench . -benchmem          # any claim about limits or throughput
```

- Every concurrency bound has a benchmark or test that proves it holds
  under contention ([19 §3](../19-performance/03-concurrency-performance.md)).
- Every timeout has a value, a source (config), and a test that fires
  it ([08 §2](../08-concurrency/02-select-and-timeouts.md)).

## Project 1: Task queue (bounded, retries, DLQ shape)

**Goal:** an in-process queue that accepts jobs, runs them with a
worker pool, retries failures with backoff, and dead-letters what
never succeeds.

| Milestone | Done when |
|---|---|
| M1: queue core | bounded channel + worker pool with graceful drain: stage-5 pattern as a service ([08 §5](../08-concurrency/05-patterns.md)) |
| M2: retries | per-job attempt count, exponential backoff with jitter, context-aware cancellation mid-retry |
| M3: DLQ shape | exhausted jobs land in a inspectable dead-letter list with triage metadata (the shape from [17 §3](../17-messaging/03-portable-patterns.md)) |
| M4: durability boundary | document exactly what is lost on process death; that honesty is the deliverable, the fix is Level 3's outbox |
| M5: proof | a test that floods the queue and asserts bounds hold; backoff schedule unit-tested with an injected clock |

**The lesson:** this project is [17 §1](../17-messaging/01-queues-logs-pubsub.md)'s
queue model, implemented by you, with the durability gap visible.
Level 3 closes that gap with Kafka; feel the gap first.

## Project 2: URL crawler (politeness, rate limits)

**Goal:** fetch many URLs concurrently without being a bad citizen:
per-host limits, a global cap, dedup, and clean cancellation.

| Milestone | Done when |
|---|---|
| M1: fetch loop | `http.Client` with real timeouts and transport tuning ([12 §4](../12-http-networking/04-clients-and-timeouts.md)) |
| M2: politeness | per-host concurrency = 1..2, global semaphore, robots awareness as a stated policy |
| M3: dedup + frontier | visited set ([03 §2](../03-data-structures/02-sets.md)) and a bounded frontier; behavior when the frontier hits its cap is documented |
| M4: shutdown | SIGINT drains in-flight fetches, writes partial results, exits nonzero on error budget breach |
| M5: proof | `-race` clean at 500 URLs; goroutine count flat over time (leak check per [09 §5](../09-memory-runtime/05-memory-leaks.md)) |

**The lesson:** rate limiting is relationship management, not
throughput tuning. The limiter you build here is the client-side
twin of the service-side one next.

## Project 3: Rate limiter service

**Goal:** a standalone limit service (or library) enforcing global
and per-identity budgets, with the two-layer design and the 429
contract.

| Milestone | Done when |
|---|---|
| M1: two-layer limiter | global + identity-keyed token buckets; the exact design and its tests exist in [21 §4](../21-security/04-limits-and-hardening.md): build yours to the same contract |
| M2: HTTP enforcement | middleware that answers 429 with `Retry-After`, and load-sheds before it collapses ([16 §5](../16-distributed-systems/05-delivery-backpressure-shedding.md)) |
| M3: degraded mode | behavior when backing store is down: fail-closed for anonymous, documented policy for authenticated ([21 §4](../21-security/04-limits-and-hardening.md)) |
| M4: proof | burst-then-sustained load test: first N burst allowed, refill rate measured within tolerance; contention benchmark |

**The lesson:** a limiter is a product decision (who gets shed first)
encoded in code. The design walkthrough is [24 §2](../24-system-design/02-rate-limiter-and-gateway.md);
your README should read like its failure-modes table.

## Project 4: Redis-backed API (cache-aside, singleflight)

**Goal:** any read-heavy API from Level 1, now with Redis cache-aside
and stampede protection.

| Milestone | Done when |
|---|---|
| M1: cache-aside | read path caches with TTLs; write path invalidates ([13 §5](../13-databases/05-caching-with-redis.md)) |
| M2: singleflight | concurrent misses collapse into one backend fetch ([19 §3](../19-performance/03-concurrency-performance.md)) |
| M3: degradation | Redis down → serve from source (or serve stale per policy), never 500 the read path |
| M4: proof | a test that fires 100 concurrent misses and asserts exactly 1 backend fetch; hit-ratio benchmark |

**The lesson:** caching correctness is invalidation discipline, and
the singleflight test (M4) is the difference between a cache and a
load amplifier. The tradeoff table you write belongs in the README
next to the numbers.

## Project 5: Authentication service (sessions, hashing)

**Goal:** register/login/refresh with secure password storage and
session tokens, as a library-plus-HTTP-service.

| Milestone | Done when |
|---|---|
| M1: password storage | PBKDF2/argon2 hashing with per-user salt and versioned parameters ([21 §5](../21-security/05-secrets-and-supply-chain.md)) |
| M2: sessions vs tokens | pick one model, justify in the README using [21 §2](../21-security/02-authentication-and-authorization.md)'s table; implement issuance, expiry, revocation |
| M3: transport discipline | constant-time comparisons, no tokens in URLs or logs, authz gate separate from authn ([14 §4](../14-backend-development/04-authn-authz-and-validation.md)) |
| M4: abuse resistance | login rate limiting and lockout policy wired to the Level-2 limiter |
| M5: proof | fuzz the token parser; a test that rejects every tampered-credential shape you can construct |

**The lesson:** auth is where defensive engineering pays visibly:
every M5 rejection test corresponds to a real attack. Threat-model
first ([21 §1](../21-security/01-threat-model-and-validation.md)),
then build.

## When this level is done

You have five services that each degrade honestly, bound explicitly,
and prove it with tests. The jump to [Level 3](03-level-3-distributed-systems.md)
is one word: *network*. Everything that was in-process (the queue,
the store, the limiter's state) now has failure modes you cannot
`go vet` away.

## Further Reading

- [08-concurrency](../08-concurrency/README.md) stages 2-5 and [13-databases](../13-databases/README.md): the two sections this level leans on hardest
- [19 §3](../19-performance/03-concurrency-performance.md): the contention benchmarks that back your limit claims
