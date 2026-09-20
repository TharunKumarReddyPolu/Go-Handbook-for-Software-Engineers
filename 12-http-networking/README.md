# 12 · HTTP & Networking

**Status: in depth: 5 chapters + a runnable stdlib-only API example.**
Standard library first; frameworks discussed only where they earn their
place. Error mapping builds on [05 Section 2](../05-errors/02-error-design.md);
httptest mechanics live in [10 Section 2](../10-testing/02-doubles-and-httptest.md);
context rules in [08 Section 3](../08-concurrency/03-context.md).

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Handlers & routing](01-handlers-and-routing.md) | Go 1.22 ServeMux patterns, handler discipline, what still needs a framework |
| 2 | [Middleware](02-middleware.md) | the wrapping pattern, Chain, ordering rules, stdlib TimeoutHandler |
| 3 | [JSON & REST APIs](03-json-and-rest-apis.md) | wire types vs domain types, decode/encode helpers, status-code decisions |
| 4 | [HTTP clients & timeouts](04-clients-and-timeouts.md) | the knob map, pooling judgment, idempotent retries |
| 5 | [Graceful shutdown](05-graceful-shutdown.md) | the full lifecycle, readiness, dependency close order |

## The example

`examples/api/` is a payments API composed entirely from the standard
library: mux routing with Go 1.22 patterns, a middleware chain
(request ID, logging, recover), bounded handlers via
`http.TimeoutHandler`, the JSON helpers from chapter 3, error mapping
from [05-errors](../05-errors/), and the graceful-shutdown lifecycle
from chapter 5.

Its tests cover the layers independently: handler table tests
(`httptest.NewRequest` + `httptest.NewRecorder`, no ports), middleware
behavior (panic → 500, no rewrite after commit), method-mismatch 405s,
readiness draining, and an in-process shutdown test proving an
in-flight request completes after `Shutdown` begins. The store test is
race-detector bait on purpose.

Deliberately out of scope here: authentication details, rate-limiting
implementations, and TLS configuration live in
[21-security](../21-security/) (outline until written); observability
wiring in [20-observability](../20-observability/); the full
production service layout in [14-backend-development](../14-backend-development/).


