# 21 · Security

**Status: in depth.** Five chapters consolidating the handbook's
defensive engineering, absorbing the topics sections
[12](../12-http-networking/) and [14](../14-backend-development/)
deferred here: token verification mechanics, TLS configuration, and
rate limiting. A dependency-free example package
([examples/secure](examples/secure/)) pins the chapter 2, 4, and 5
mechanics with tests.

## Chapters

1. **[Threat modeling & validation](01-threat-model-and-validation.md)**:
   the one-page model, trust boundaries, injection and traversal
   defenses, parse-don't-validate
2. **[Authentication & authorization](02-authentication-and-authorization.md)**:
   pinned-algorithm token verification, sessions vs JWTs, the two
   authz gates, the 401/403/404 disclosure discipline
3. **[TLS & certificates](03-tls-certificates.md)**: modern Go
   defaults, autocert, internal mTLS, rotation and expiry monitoring
4. **[Limits & hardening](04-limits-and-hardening.md)**: the
   two-layer rate limiter, body caps, cheap-checks-first ordering,
   the 429 contract
5. **[Secrets, hashing & the supply chain](05-secrets-and-supply-chain.md)**:
   redaction types, PBKDF2/argon2id, SSRF allowlisted fetching, CORS,
   govulncheck and fuzzing

## The example

`examples/secure` implements three controls from the chapters:

- `Ed25519Verifier`: the algorithm pinned by the key type itself;
  the test suite rejects expired, wrong-issuer, wrong-audience,
  cross-key, tampered, and header-baited tokens
- `Limiter`: global + per-identity token buckets with TTL eviction,
  fully deterministic on an injected clock
- `Fetcher`: HTTPS-only, host-allowlisted, private-IP-rejecting
  outbound client with redirects disabled

```bash
go test ./21-security/... -race -v
```

## Placement map

Where each control sits in the request path of the Section 14
service:

```text
Instrument (metrics: 429s and 401s are traffic)
  -> Recover -> ScopedLogger
  -> Authenticate      (ch 2: token verification)
  -> RateLimit         (ch 4: identity-keyed)
  -> handler           (ch 1: decode + validate boundary)
  -> service           (ch 2: ownership authz)
```
