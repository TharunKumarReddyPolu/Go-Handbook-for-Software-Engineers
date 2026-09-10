# 14 · Backend Development

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** The structure question is answered at
interview depth in
[26 §4](../26-go-interview-preparation/04-senior-scenarios.md); the
error-boundary and httptest layers already exist in
[05 §2](../05-errors/02-error-design.md) and
[10 §2](../10-testing/02-doubles-and-httptest.md).

## Planned chapters

1. **API architecture**: layering (domain / service / transport),
   dependency direction, package layout
   (cmd/internal shape from [01 §3](../01-go-fundamentals/03-program-structure.md))
2. **Configuration**: env-first loading, validation, the zero-value
   rules from [01 §5](../01-go-fundamentals/05-zero-values.md)
3. **Dependency injection without frameworks**: constructor wiring,
   functional options, consumer-side interfaces
4. **Validation & authn/authz**: boundary validation patterns;
   middleware composition from [12](../12-http-networking/) (planned)
5. **Logging, metrics & tracing in the service**: the request-scoped
   plumbing ([20-observability](../20-observability/) pairs)
6. **Graceful shutdown & health**: full lifecycle
7. **Secrets & feature flags**: boundary handling
8. **Complete production-style example**: a small service assembling
   every chapter: one codebase, fully tested, the capstone's skeleton
   ([28-projects](../28-projects/))
