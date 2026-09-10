# 21 · Security

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Security notes are embedded throughout
the handbook (every chapter has the section); this section consolidates
the defensive-engineering depth.

## Planned chapters

1. **Threat modeling for engineers**: trust boundaries in Go services;
   the fintech sharpening from
   [25 §4](../25-fintech-with-go/04-risk-and-compliance.md)
2. **Authentication & authorization**: OAuth2/OIDC flows, JWT
   pitfalls (alg none, kid confusion), session patterns, mTLS between
   services
3. **TLS everywhere**: server config, internal mTLS, certificate
   rotation
4. **Input validation & injection**: SQL injection
   ([13-databases](../13-databases/) pairs), command injection,
   path traversal; validation at boundaries
5. **SSRF, XSS, CSRF**: the HTTP-adjacent trio with Go-specific
   mitigations
6. **Secrets management**: managers, rotation, the never-log rules;
   idempotency keys as credentials-adjacent
   ([25 §1](../25-fintech-with-go/01-money-and-payments.md))
7. **Password hashing**: argon2/bcrypt via golang.org/x/crypto,
   parameter choices
8. **Rate limiting as a security control**: per-source bounds, the
   DoS framing from [08 §7](../08-concurrency/07-pitfalls.md)
9. **Supply chain**: govulncheck in CI
   ([.github/workflows/security.yml](../.github/workflows/security.yml)),
   dependency review, SBOM
10. **Fuzzing as defense**: trust-boundary decoders
    ([10 §4](../10-testing/04-benchmarks-coverage-fuzzing.md) pairs)
11. **Profiling endpoints & debug surfaces**: pprof exposure rules
    ([19 §1](../19-performance/01-measure-first.md) pairs)
