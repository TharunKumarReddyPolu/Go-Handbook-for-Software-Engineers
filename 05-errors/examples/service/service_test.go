package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandler() http.Handler {
	return newHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
}

func postPayment(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPayment_Created(t *testing.T) {
	rec := postPayment(t, testHandler(),
		`{"idempotency_key":"k1","amount_minor":1000,"currency":"USD"}`)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}
}

func TestPayment_ValidationIs400(t *testing.T) {
	rec := postPayment(t, testHandler(),
		`{"idempotency_key":"","amount_minor":-5,"currency":"EU"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	// Joined validation errors should mention each violation.
	body := rec.Body.String()
	for _, want := range []string{"idempotency_key", "amount_minor", "currency"} {
		if !bytes.Contains([]byte(body), []byte(want)) {
			t.Errorf("body should mention %q; body: %s", want, body)
		}
	}
}

func TestPayment_PolicyCapIs400Not500(t *testing.T) {
	rec := postPayment(t, testHandler(),
		`{"idempotency_key":"k2","amount_minor":100000000,"currency":"USD"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (classified domain error)", rec.Code)
	}
}

func TestPayment_CancelledContextIs503(t *testing.T) {
	h := testHandler()
	req := httptest.NewRequest(http.MethodPost, "/payments",
		bytes.NewBufferString(`{"idempotency_key":"k3","amount_minor":500,"currency":"USD"}`))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 for retryable unavailability", rec.Code)
	}
	if ra := rec.Header().Get("Retry-After"); ra == "" {
		t.Error("503 should include Retry-After")
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("context")) {
		t.Error("response body must not leak internal error detail")
	}
}

func TestPayment_MalformedJSON(t *testing.T) {
	rec := postPayment(t, testHandler(), `{not json`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
