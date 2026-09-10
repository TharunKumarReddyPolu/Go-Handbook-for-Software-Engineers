# 12 · HTTP & Networking

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Boundary patterns already demonstrated
in [05 §2](../05-errors/02-error-design.md) and
[10 §2 httptest](../10-testing/02-doubles-and-httptest.md); shutdown in
[08 §3 context](../08-concurrency/03-context.md).

## Planned chapters: standard library first, frameworks only where
useful

1. **Handlers & the request lifecycle**: ServeMux patterns (incl.
   method/wildcard routing), handler composition
2. **Middleware**: the wrapping pattern, ordering, context
   propagation; a full middleware chain worked example
3. **JSON & REST APIs**: encoding hygiene, validation at the boundary,
   error mapping from [05 §2](../05-errors/02-error-design.md)
4. **HTTP clients**: transport pooling, timeouts (every level),
   retries with idempotency, connection lifetime
5. **TLS**: servers, clients, internal mTLS
6. **Cookies, authn & authz**: session patterns, OAuth2/JWT pitfalls
   (pairs with [21-security](../21-security/))
7. **CORS**: what the browser actually enforces
8. **Rate limiting**: per-route/per-source; extends
   [08 §5](../08-concurrency/05-patterns.md)
9. **Health, readiness & liveness**: Kubernetes probes done right
10. **Graceful shutdown**: full server lifecycle; extends
    [08 §3](../08-concurrency/03-context.md)
