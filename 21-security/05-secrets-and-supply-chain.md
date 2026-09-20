# Secrets, hashing & the supply chain

## Why Does This Matter?

Three last controls round out the section, and they fail in boring
ways: the API key that ends up in a log line, the password hashed
with SHA-256, the internal endpoint that fetches whatever URL a
user pastes, and the transitive dependency with a vulnerability you
never chose. Each has a stdlib-first fix.

## Mental Model

| Control | The failure it prevents | The Go tool |
|---|---|---|
| Secrets management | Credentials in logs/repos/images | `config.Secret` type + env/mounts |
| Password hashing | Stolen password DB = stolen logins | `crypto/pbkdf2` (or argon2id) |
| SSRF defense | Server fetches attacker URLs | Allowlisted fetcher |
| CORS | Browser-based token theft | Deliberate origin allowlist |
| govulncheck | Known-vulnerable dependencies | CI gate |
| Fuzzing | Parser bugs at trust boundaries | `go test -fuzz` |

## How It Works

**Secrets** ([14 Section 2](../14-backend-development/02-configuration-and-secrets.md)
established the `Secret` type with `LogValue` redaction; the rules
here complete it): secrets come from the environment or mounted
secret files, never from code, never in images, never in logs.
Rotation is a config restart or a re-read, designed in from day
one: if rotating a key requires a migration, you built it wrong.

**Password hashing** must be deliberately slow. SHA-256 is wrong
because GPUs compute it billions of times per second. PBKDF2 ships
in the stdlib (Go 1.24+):

```go
import "crypto/pbkdf2"
import "crypto/rand"

func hashPassword(pw string) (hash, salt []byte, err error) {
    salt = make([]byte, 16)
    _, _ = rand.Read(salt)
    hash, err = pbkdf2.Key(sha256.New, pw, salt, 210_000, 32)
    return hash, salt, err
}

func checkPassword(hash, salt []byte, pw string) bool {
    h, err := pbkdf2.Key(sha256.New, pw, salt, 210_000, 32)
    if err != nil { return false }
    return subtle.ConstantTimeCompare(h, hash) == 1
}
```

210,000 iterations is OWASP's current floor for PBKDF2-SHA256;
store `salt` with the hash and make the iteration count a parameter
you can raise (verify with stored count, rehash on login to bump).
For new code where `golang.org/x/crypto/argon2` is available,
prefer argon2id; the pattern (salt, slow KDF, constant-time
compare) is identical.

**SSRF**: the server-side request forgery problem is your server
fetching an attacker-chosen URL, reaching cloud metadata
(`169.254.169.254`), localhost admin ports, or internal RFC1918
space. Defense is an allowlisted client:

```go
var safeHost = map[string]bool{"api.stripe.com": true, "hooks.slack.com": true}

func safeFetch(ctx context.Context, client *http.Client, raw string) (*http.Response, error) {
    u, err := url.Parse(raw)
    if err != nil || u.Scheme != "https" || !safeHost[u.Hostname()] {
        return nil, ErrInvalidURL
    }
    ip, err := net.LookupIP(u.Hostname())[0], error(nil)
    if err == nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
        return nil, ErrInvalidURL
    }
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
    return client.Do(req) // client has redirects OFF or re-validated
}
```

The redirect clause matters: validation of the original URL is
defeated by a redirect to `169.254.169.254` unless redirects are
disabled or re-validated per hop.

**CORS** is a browser rule, not an API rule: your API without
CORS headers is callable by any server and any curl; CORS only
constrains what a *browser* will let a *web page* do with the
response. Configure it only when browsers are first-class clients,
then allowlist origins explicitly (`cors` middleware with an exact
origin map), never reflect `Origin` blindly, and combine with
`SameSite` cookies (chapter 2) so CSRF and CORS defense don't
depend on each other.

## Syntax / API

The dependency gate that runs in CI already
([.github/workflows/security.yml](../.github/workflows/security.yml)):

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

`govulncheck` is call-graph aware: it reports only vulnerabilities
in functions your code actually reaches, which keeps the noise low
enough to act on every finding. Fuzzing completes the supply chain
of trust in your own parsers:

```go
func FuzzResolveUnder(f *testing.F) {
    f.Add("../etc/passwd")
    f.Fuzz(func(t *testing.T, name string) {
        p, err := resolveUnder("/srv/files", name)
        if err == nil && !strings.HasPrefix(p, "/srv/files/") {
            t.Fatalf("escaped root: %q -> %q", name, p)
        }
    })
}
```

## Basic Example

The webhook receiver that got SSRF right: allowlist of two hosts,
HTTPS only, private-IP check, redirects off, response size capped.
The same code path with `http.Get(userURL)` is an incident waiting
for a schedule.

## Real-World Example

The payments service's secret story, end to end: `config.Secret`
redacts in logs; `Summary()` prints config with only shapes, never
values; the store DSN is assembled from parts so the password
never appears in an error message; and CI's govulncheck plus the
weekly schedule (already wired in this repo) catches dependency
CVEs before deploy.

## Production Example

**Secret rotation without downtime**: two readers (old key verify,
new key sign/verify) overlapping for the TTL of your longest
token; flip sign-over after deploy; remove the old key after the
TTL. The same overlap window works for DB credentials (chapter 3's
short-lived mTLS certs follow the same shape).

**The confused deputy**: your service holds credentials for many
customers (OAuth tokens per tenant). An attacker tricks the service
into using tenant A's credentials against tenant B's data. Defense
is the ownership gate from chapter 2 with the tenant ID resolved
per request, never cached across requests: the classic
implementation bug is a tenant-keyed cache consulted before the
ownership check.

## Common Mistakes

| Mistake | Consequence | Do instead |
|---|---|---|
| Secrets in struct fields with `%+v` nearby | Log leak | `Secret` type with redaction |
| SHA-256/MD5 for passwords | Instant brute force | PBKDF2/argon2id, salted, parameterized |
| `subtle.ConstantTimeCompare` forgotten | Timing attacks on token checks | Constant-time for all secret comparisons |
| Fetch-then-validate URLs | SSRF via redirects | No redirects or re-validate per hop |
| CORS `*` with credentials | Browser token theft | Exact origin allowlist |
| govulncheck on schedule only | Vulnerability ships with the deploy | Gate the PR, schedule as backstop |
| One shared internal credential | No revocation, blast radius | Per-service identity (mTLS or tokens) |

## Idiomatic Go

- `crypto/rand`, never `math/rand`, for anything secret-shaped.
- `subtle.ConstantTimeCompare` for every comparison involving a
  secret.
- Secrets managers (AWS/GCP/Vault) are integration points, not Go
  libraries to hand-roll; the Go-side discipline (types, redaction,
  rotation windows) is portable across all of them.

## Performance Considerations

Deliberately slow hashing is the point: budget it. At 210k
iterations, PBKDF2-SHA256 costs tens of milliseconds: fine for
login, wrong for per-request verification (that is what tokens are
for). SSRF's DNS check adds a lookup per fetch: cache with a short
TTL, and revalidate per redirect instead of trusting the cache.

## Concurrency Considerations

Key rotation races: verify with the old key while the new key is
already signing. Solve by holding both keys during the overlap
window and trying verifies in order. Password hashing is
CPU-bound: a login endpoint without a semaphore is a DoS
amplifier; bound concurrent hashes (chapter 4's limits).

## Security Considerations

The whole chapter is security; the cross-cutting rule is defense in
depth: redaction types *and* log review, allowlists *and* egress
firewalls, govulncheck *and* least privilege for what a dependency
can touch when it is nonetheless compromised.

## Testing Strategy

- Hash round-trip tests plus a wrong-password and wrong-iteration
  test.
- SSRF table tests: each private/loopback/metadata IP class
  rejected; allowlisted hosts accepted.
- CORS: preflight and actual-request tests per configured origin,
  and one proving the unconfigured origin gets no headers.
- `go test -fuzz` the boundary decoders in CI (short duration) and
  overnight in a scheduled job.

## Interview Questions

1. Why is constant-time comparison required for MAC checks but
   not for table lookups of non-secret keys?
2. Walk through an SSRF fix on a webhook feature: what do you
   validate, when, and what defeats each check?
3. Your dependency has a critical CVE; govulncheck says your code
   does not call it. What do you do and why? (Track; fix on
   schedule; the call-graph result is the severity input, not
   license to ignore.)
4. Design key rotation for a signing key with zero downtime.

## Practice Exercises

1. Add `FuzzSafeFetch` that asserts no fetch to private IPs ever
   succeeds, including via redirects; fix what it finds.
2. Wire a rehash-on-login bump (stored iterations 100k -> 210k)
   behind a flag.
3. Add a CORS preflight test suite for the service's configured
   origins.

## Further Reading

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [crypto/pbkdf2 documentation](https://pkg.go.dev/crypto/pbkdf2)
- [govulncheck documentation](https://go.dev/blog/vuln)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Go fuzzing](https://go.dev/doc/fuzz/)
