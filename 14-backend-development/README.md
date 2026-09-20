# 14 · Backend Development

**Status: in depth: 5 chapters + a complete layered service example.**
This section composes the API layer of [12](../12-http-networking/)
and the store discipline of [13](../13-databases/) into the production
layout, and completes the backend phase of the ROADMAP. Error
boundaries come from [05 Section 2](../05-errors/02-error-design.md); the
shutdown lifecycle from [12 Section 5](../12-http-networking/05-graceful-shutdown.md).

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Service layout & layering](01-service-layout.md) | domain-shaped packages, dependency direction, the request's path through layers |
| 2 | [Configuration & secrets](02-configuration-and-secrets.md) | env-first loading, boot-time validation, the redaction seam |
| 3 | [Wiring & dependency injection](03-wiring-and-dependency-injection.md) | the explicit graph in main, constructor escalation, init() demotion, the extracted run loop |
| 4 | [Validation, authn & authz](04-authn-authz-and-validation.md) | the three gates, coarse vs fine authz, the 403-vs-404 decision |
| 5 | [Logging, health & flags](05-observability-health-flags.md) | request-scoped slog, liveness vs readiness, flags as the config boundary |

## The example

`examples/service/` is the layout in miniature, zero setup to run:

```
main.go                       # wiring only: config, constructors, run
internal/
├── config/                   # ch 2: typed, validated, redacting loader
│   └── config.go (+ tests)
├── platform/                 # ch 1's rule: knows no domain
│   ├── httpjson/             # 12 Section 3's boundary helpers
│   └── httpmw/               # 12 Section 2's chain + scoped logger (ch 5)
└── payments/                 # the domain
    ├── service.go            # decisions; owns PaymentStore interface
    ├── memory.go             # STORE=memory implementation
    ├── http.go               # transport: wire types, policy table, error map
    └── service_test.go       # both tiers: domain + full-chain contract
```

The test suite (~15 cases) pins the contracts each chapter promised:
batched validation errors (ch 2), the ownership matrix with the
service-tier `ErrForbidden` → transport 404 mapping (ch 4), the
status-code contract across the whole chain (401/400/422/201), and
the scoped-logger request ID reaching the response header (ch 5).

```bash
go test ./14-backend-development/...   # runs the example's tests
go run ./14-backend-development/examples/service   # then curl :8080/livez
```

The Postgres store is deliberately not wired here: 13-databases's
`bank` package carries the full SQL implementation with its two-tier
tests, and wiring it is the capstone's exercise
([28-projects](../28-projects/)). The config switch
(`STORE=postgres`) documents exactly where it lands.


