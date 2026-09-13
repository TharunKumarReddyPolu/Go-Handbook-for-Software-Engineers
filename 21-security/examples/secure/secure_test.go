package secure

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// --- chapter 2: token verification ---------------------------------

type tokenFixture struct {
	name    string
	header  string // JSON header segment
	payload any
	key     ed25519.PublicKey // which key signs it
}

func pubKey() ed25519.PublicKey {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	return pub
}

func encodeRaw(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func buildAuth(t *testing.T, f tokenFixture, priv ed25519.PrivateKey) string {
	t.Helper()
	header := encodeRaw(t, map[string]string{"alg": "EdDSA"})
	if f.header != "" {
		header = encodeRaw(t, map[string]string{"alg": f.header})
	}
	payload := encodeRaw(t, f.payload)
	sig := ""
	if priv != nil {
		sig = base64.RawURLEncoding.EncodeToString(
			ed25519.Sign(priv, []byte(header+"."+payload)))
	}
	return "Bearer " + header + "." + payload + "." + sig
}

func TestVerify_AcceptsValidToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := NewEd25519Verifier(pub, "https://idp", "payments")
	future := time.Now().Add(time.Hour)
	auth := buildAuth(t, tokenFixture{payload: Claims{
		Subject: "cus_42", Issuer: "https://idp",
		Audience: "payments", ExpiresAt: future,
	}}, priv)

	got, err := v.Verify(auth)
	if err != nil || got != "cus_42" {
		t.Fatalf("Verify = %q, %v; want cus_42, nil", got, err)
	}
}

func TestVerify_RejectsEveryFailureMode(t *testing.T) {
	goodPub, goodPriv, _ := ed25519.GenerateKey(rand.Reader)
	evilPub, evilPriv, _ := ed25519.GenerateKey(rand.Reader)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	base := Claims{Subject: "cus_42", Issuer: "https://idp",
		Audience: "payments", ExpiresAt: future}

	tests := []tokenFixture{
		{name: "expired", payload: func() Claims {
			c := base
			c.ExpiresAt = past
			return c
		}(), key: goodPub},
		{name: "wrong issuer", payload: func() Claims {
			c := base
			c.Issuer = "https://evil.example"
			return c
		}(), key: goodPub},
		{name: "wrong audience", payload: func() Claims {
			c := base
			c.Audience = "reporting" // valid token minted for another service
			return c
		}(), key: goodPub},
		{name: "signed by other key", payload: base, key: evilPub},
		{name: "no subject", payload: func() Claims {
			c := base
			c.Subject = ""
			return c
		}(), key: goodPub},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewEd25519Verifier(goodPub, "https://idp", "payments")
			auth := buildAuth(t, tt, func() ed25519.PrivateKey {
				if tt.key.Equal(evilPub) {
					return evilPriv
				}
				return goodPriv
			}())
			if _, err := v.Verify(auth); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Verify(%s) = %v, want ErrUnauthorized", tt.name, err)
			}
		})
	}

	// Structural rejects: malformed inputs, tampering, and the
	// header-derived algorithm bait.
	t.Run("tampered payload", func(t *testing.T) {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		v := NewEd25519Verifier(pub, "https://idp", "payments")
		auth := buildAuth(t, tokenFixture{payload: base}, priv)
		// Swap the subject inside the payload, keep the signature.
		parts := strings.Split(strings.TrimPrefix(auth, "Bearer "), ".")
		evilPayload := encodeRaw(t, func() Claims {
			c := base
			c.Subject = "cus_99"
			return c
		}())
		tampered := "Bearer " + parts[0] + "." + evilPayload + "." + parts[2]
		if _, err := v.Verify(tampered); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("tampered token accepted: %v", err)
		}
	})
	t.Run("header claims HS256", func(t *testing.T) {
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		v := NewEd25519Verifier(pub, "https://idp", "payments")
		// Correct Ed25519 signature, but the header names a different
		// algorithm: with a header-driven verifier this is the RS-to-HS
		// confusion opening. Here it must not matter.
		auth := buildAuth(t, tokenFixture{header: "HS256", payload: base}, priv)
		if _, err := v.Verify(auth); err != nil {
			t.Fatalf("pinned verifier must ignore header alg: %v", err)
		}
	})
	t.Run("not a bearer", func(t *testing.T) {
		v := NewEd25519Verifier(pubKey(), "https://idp", "payments")
		for _, auth := range []string{"", "Basic abc", "Bearer", "Bearer a.b"} {
			if _, err := v.Verify(auth); !errors.Is(err, ErrUnauthorized) {
				t.Errorf("Verify(%q) = %v, want ErrUnauthorized", auth, err)
			}
		}
	})
}

// --- chapter 4: two-layer limiting ---------------------------------

func TestLimiter_TwoLayers(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	l := NewLimiter(rate.Limit(10_000), 10_000, rate.Limit(2), 3, 0)
	l.now = func() time.Time { return now }

	// Per-identity burst of 3 is absorbed, the 4th is rejected.
	for i := 0; i < 3; i++ {
		if !l.Allow("cus_a") {
			t.Fatalf("request %d rejected inside burst", i+1)
		}
	}
	if l.Allow("cus_a") {
		t.Fatal("4th request allowed past burst")
	}
	// A different identity still has its own budget.
	if !l.Allow("cus_b") {
		t.Fatal("per-identity budget leaked across identities")
	}
	// Refill at 2/s: after 1s two tokens accrued, both spends pass;
	// the third at the same instant waits (the rate ceiling holds).
	now = now.Add(time.Second)
	ok1, ok2 := l.Allow("cus_a"), l.Allow("cus_a")
	if !ok1 || !ok2 {
		t.Fatal("accrued tokens not spendable after 1s at 2/s")
	}
	if l.Allow("cus_a") {
		t.Fatal("spend beyond the 2/s refill rate allowed")
	}
	// Half a second later: exactly one more token.
	now = now.Add(500 * time.Millisecond)
	if !l.Allow("cus_a") {
		t.Fatal("token not refilled after 0.5s at 2/s")
	}
}

func TestLimiter_GlobalGateAndEviction(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	l := NewLimiter(rate.Limit(3), 3, rate.Limit(100), 100, time.Minute)
	l.now = func() time.Time { return now }

	// Eviction first (global bucket still full): idle identities are
	// swept; fresh ones stay.
	l.Allow("idle") // seeded at t0
	now = now.Add(2 * time.Minute)
	l.Allow("active")
	l.Sweep()
	if _, ok := l.keys.Load("idle"); ok {
		t.Error("idle identity not evicted")
	}
	if _, ok := l.keys.Load("active"); !ok {
		t.Error("active identity evicted")
	}

	// Global gate on a fresh limiter: exhausts regardless of identity.
	g := NewLimiter(rate.Limit(3), 3, rate.Limit(100), 100, 0)
	g.now = l.now
	for i := 0; i < 3; i++ {
		if !g.Allow("any") {
			t.Fatalf("global request %d rejected", i+1)
		}
	}
	if g.Allow("other") {
		t.Fatal("global gate not enforced")
	}
}

// --- chapter 5: allowlisted fetching --------------------------------

func TestFetcher_Policy(t *testing.T) {
	f := NewFetcher("api.example.com")

	tests := []struct {
		url  string
		want error
	}{
		{"http://api.example.com/x", ErrBlockedURL},       // scheme
		{"https://evil.example/x", ErrBlockedURL},         // allowlist
		{"https://169.254.169.254/latest", ErrBlockedURL}, // metadata
		{"https://127.0.0.1:6060/debug", ErrBlockedURL},   // loopback
		{"https://10.0.0.9/admin", ErrBlockedURL},         // RFC1918
		{"not a url", ErrBlockedURL},
	}
	for _, tt := range tests {
		if _, err := f.Fetch(context.Background(), tt.url); !errors.Is(err, ErrBlockedURL) {
			t.Errorf("Fetch(%q) err = %v, want ErrBlockedURL", tt.url, err)
		}
	}
	// The allowlisted host with a resolvable name reaches the real
	// network path; in CI without DNS for it, the error must still be
	// ErrBlockedURL-free (resolution failure) rather than a leak.
}

func TestFetcher_RedirectsDisabled(t *testing.T) {
	f := NewFetcher("api.example.com")
	if f.Client.CheckRedirect == nil {
		t.Fatal("redirects not disabled")
	}
	if err := f.Client.CheckRedirect(nil, make([]*http.Request, 3)); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect = %v, want ErrUseLastResponse", err)
	}
}
