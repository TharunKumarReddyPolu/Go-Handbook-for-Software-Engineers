// Package secure is the handbook's worked example for 21-security:
// a pinned-algorithm bearer-token verifier (chapter 2), a two-layer
// token-bucket limiter with TTL eviction (chapter 4), and an
// allowlisted outbound fetcher (chapter 5). Small, dependency-free,
// and tested against the failure modes each chapter names.
package secure

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// ErrUnauthorized covers every rejected token: the caller gets one
// signal, never a reason an attacker could iterate on. Log the
// specific cause server-side instead.
var ErrUnauthorized = errors.New("unauthorized")

// Claims is the minimum claim set this service trusts. Anything not
// here is ignored: an attacker-controlled header never influences
// verification (chapter 2's pinned-algorithm rule).
type Claims struct {
	Subject   string    `json:"sub"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
}

// Ed25519Verifier validates Bearer JWTs signed with Ed25519. The
// algorithm is pinned by the key type itself: ed25519.Verify cannot
// be tricked into HMAC mode, and there is no "alg: none" path.
type Ed25519Verifier struct {
	key      ed25519.PublicKey
	issuer   string
	audience string
	now      func() time.Time
}

// NewEd25519Verifier returns a verifier with the production clock.
func NewEd25519Verifier(key ed25519.PublicKey, issuer, audience string) *Ed25519Verifier {
	return &Ed25519Verifier{key: key, issuer: issuer, audience: audience, now: time.Now}
}

// WithNow replaces the clock for deterministic tests.
func (v *Ed25519Verifier) WithNow(now func() time.Time) *Ed25519Verifier {
	v.now = now
	return v
}

// Verify checks signature, expiry, issuer, and audience, in that
// order, and returns only the caller identity on success.
func (v *Ed25519Verifier) Verify(authHeader string) (string, error) {
	tok, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return "", ErrUnauthorized
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return "", ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrUnauthorized
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ErrUnauthorized
	}
	if !claims.ExpiresAt.After(v.now()) {
		return "", ErrUnauthorized
	}
	if claims.Issuer != v.issuer || claims.Audience != v.audience {
		return "", ErrUnauthorized
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", ErrUnauthorized
	}
	// The signed content is "header.payload" verbatim, reassembled
	// from the raw segments; re-encoding either part would be the
	// classic flaw.
	signed := parts[0] + "." + parts[1]
	if !ed25519.Verify(v.key, []byte(signed), sig) {
		return "", ErrUnauthorized
	}
	if claims.Subject == "" {
		return "", ErrUnauthorized
	}
	// The header is deliberately untrusted: the algorithm is pinned
	// by the key type (ed25519.Verify has no other mode).
	return claims.Subject, nil
}

// Limiter is chapter 4's two-layer gate: a global token bucket plus
// a per-identity bucket, with TTL eviction so identities that stop
// calling do not leak memory. Identifiers are caller subjects where
// authn exists and IP literals otherwise.
type Limiter struct {
	global *rate.Limiter
	keys   sync.Map // string -> *rate.Limiter
	limit  rate.Limit
	burst  int
	ttl    time.Duration
	now    func() time.Time
}

// NewLimiter builds the two-layer limiter. perID is the sustained
// per-identity rate; burst is its bucket depth. ttl is the eviction
// window for idle identities (0 disables eviction).
func NewLimiter(global rate.Limit, globalBurst int, perID rate.Limit, burst int, ttl time.Duration) *Limiter {
	return &Limiter{
		global: rate.NewLimiter(global, globalBurst),
		limit:  perID,
		burst:  burst,
		ttl:    ttl,
		now:    time.Now,
	}
}

// Allow reports whether the identity may proceed now. Global
// exhaustion rejects before per-identity work: cheap failure first.
// Both buckets run on the injected clock (AllowN), which is what
// makes the limiter deterministically testable.
func (l *Limiter) Allow(id string) bool {
	if !l.global.AllowN(l.now(), 1) {
		return false
	}
	v, _ := l.keys.LoadOrStore(id, newKeyedLimiter(l))
	kl := v.(*keyedLimiter)
	kl.mark(l.now())
	return kl.l.AllowN(l.now(), 1)
}

// Sweep evicts limiters idle longer than the TTL. Call it
// periodically from one goroutine (a ticker in main); Allow itself
// stays lock-cheap.
func (l *Limiter) Sweep() {
	if l.ttl <= 0 {
		return
	}
	cutoff := l.now().Add(-l.ttl)
	l.keys.Range(func(key, v any) bool {
		kl := v.(*keyedLimiter)
		if kl.seenNano() < cutoff.UnixNano() {
			l.keys.CompareAndDelete(key, v)
		}
		return true
	})
}

// keyedLimiter pairs a limiter with its last-seen time. The
// timestamp is atomic so Sweep never takes a per-key lock.
type keyedLimiter struct {
	l    *rate.Limiter
	last atomic.Int64 // UnixNano
}

func newKeyedLimiter(l *Limiter) *keyedLimiter {
	return &keyedLimiter{l: rate.NewLimiter(l.limit, l.burst)}
}

func (k *keyedLimiter) mark(now time.Time) { k.last.Store(now.UnixNano()) }
func (k *keyedLimiter) seenNano() int64    { return k.last.Load() }

// Fetcher is chapter 5's allowlisted outbound client: HTTPS only,
// hostname allowlist, private/link-local IP rejection, redirects
// disabled so per-hop validation cannot be bypassed.
type Fetcher struct {
	Client *http.Client
	Allow  map[string]bool
}

// NewFetcher builds a fetcher for the given hosts. The client has
// redirects disabled by CheckRedirect: every hop must re-validate.
func NewFetcher(allowedHosts ...string) *Fetcher {
	set := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		set[strings.ToLower(h)] = true
	}
	return &Fetcher{
		Client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Allow: set,
	}
}

// ErrBlockedURL is returned for any URL outside the allowlist or
// targeting private infrastructure.
var ErrBlockedURL = errors.New("url blocked by policy")

// Fetch issues a GET to rawURL after full validation. The IP check
// runs on a fresh lookup per call: DNS rebinding between calls is
// out of scope here, but per-call checks defeat the static bypass.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme must be https", ErrBlockedURL)
	}
	host := strings.ToLower(u.Hostname())
	if !f.Allow[host] {
		return nil, fmt.Errorf("%w: host %q not allowed", ErrBlockedURL, host)
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("%w: resolve failed", ErrBlockedURL)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return nil, fmt.Errorf("%w: private address", ErrBlockedURL)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: bad request", ErrBlockedURL)
	}
	return f.Client.Do(req)
}
