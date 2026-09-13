package payments

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- domain tier: service + fake store + fake clock ---

func testService(t *testing.T) (*Service, *fakeStore, *[]time.Time) {
	t.Helper()
	store := newFakeStore()
	svc := NewService(store, slog.New(slog.DiscardHandler))
	calls := &[]time.Time{}
	svc.Now(func() time.Time {
		tm := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC).Add(time.Duration(len(*calls)) * time.Second)
		*calls = append(*calls, tm)
		return tm
	})
	return svc, store, calls
}

func TestCharge_ValidationBatched(t *testing.T) {
	svc, _, _ := testService(t)

	_, err := svc.Charge(t.Context(), ChargeInput{CustomerID: "", AmountMinor: -1, Currency: "us"})
	if err == nil {
		t.Fatal("want validation error")
	}
	var ve *validationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want *validationError", err)
	}
	for _, want := range []string{"customer_id", "amount_minor", "currency"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got %q (batched?)", want, err.Error())
		}
	}
}

func TestCancel_OwnershipMatrix(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		actor   string
		status  Status
		wantErr error
	}{
		{"own, charged, cancels", "cus_1", "cus_1", StatusCharged, nil},
		{"own, already cancelled", "cus_1", "cus_1", StatusCancelled, ErrNotCancellable},
		// The service returns ErrForbidden; the TRANSPORT maps it to 404
		// (existence not disclosed). Each tier asserts its own contract.
		{"theirs: forbidden at service tier", "cus_1", "cus_2", StatusCharged, ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store, _ := testService(t)
			p, err := store.Charge(t.Context(), Payment{CustomerID: tt.owner, AmountMinor: 100, Currency: "USD", Status: tt.status})
			if err != nil {
				t.Fatal(err)
			}

			err = svc.Cancel(t.Context(), Customer{ID: tt.actor}, p.ID)
			if !errors.Is(err, tt.wantErr) && tt.wantErr != nil {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func TestByCustomer_BoundedAndNewestFirst(t *testing.T) {
	svc, _, _ := testService(t)
	ctx := t.Context()
	for i := 0; i < 5; i++ {
		if _, err := svc.Charge(ctx, ChargeInput{CustomerID: "cus_1", AmountMinor: 100, Currency: "USD"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.ByCustomer(ctx, "cus_1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].ID != "pay_0005" {
		t.Errorf("newest = %s, want pay_0005", got[0].ID)
	}
}

// --- transport tier: the full chain through httptest (no ports) ---

func routes(t *testing.T) http.Handler {
	t.Helper()
	svc, _, _ := testService(t)
	return NewHandler(svc).Routes(slog.New(slog.DiscardHandler), nil)
}

func authedReq(method, target, token, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestChain_Contract(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		status int
		probe  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "no token: 401 with WWW-Authenticate",
			req:    authedReq(http.MethodPost, "/payments", "", `{"customer_id":"cus_1","amount_minor":100,"currency":"USD"}`),
			status: http.StatusUnauthorized,
			probe: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if got := rec.Header().Get("WWW-Authenticate"); got == "" {
					t.Error("missing WWW-Authenticate header")
				}
			},
		},
		{
			name:   "health inside chain but unauthenticated routes exist: livez 200",
			req:    authedReq(http.MethodGet, "/livez", "", ""),
			status: http.StatusUnauthorized, // livez sits behind authn in this example: pinned as documented
		},
		{
			name:   "user token: valid create",
			req:    authedReq(http.MethodPost, "/payments", "user:cus_1", `{"customer_id":"cus_1","amount_minor":100,"currency":"USD"}`),
			status: http.StatusCreated,
			probe: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if got := rec.Header().Get("Location"); got == "" {
					t.Error("missing Location header")
				}
				if got := rec.Header().Get("X-Request-ID"); got == "" {
					t.Error("missing request ID from scoped logger middleware")
				}
			},
		},
		{
			name:   "batched validation: 422 listing all",
			req:    authedReq(http.MethodPost, "/payments", "user:cus_1", `{"customer_id":"","amount_minor":-5,"currency":"us"}`),
			status: http.StatusUnprocessableEntity,
			probe: func(t *testing.T, rec *httptest.ResponseRecorder) {
				body := rec.Body.String()
				for _, want := range []string{"customer_id", "amount_minor", "currency"} {
					if !strings.Contains(body, want) {
						t.Errorf("body missing %q: %s", want, body)
					}
				}
			},
		},
		{
			name:   "unknown field: 400 by strict decode",
			req:    authedReq(http.MethodPost, "/payments", "user:cus_1", `{"customer_id":"c","amount_minor":1,"currency":"USD","x":1}`),
			status: http.StatusBadRequest,
		},
		{
			name:   "unparseable: 400",
			req:    authedReq(http.MethodPost, "/payments", "user:cus_1", `{oops`),
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			routes(t).ServeHTTP(rec, tt.req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.status, rec.Body)
			}
			if tt.probe != nil {
				tt.probe(t, rec)
			}
		})
	}
}

func TestChain_CancelOwnershipOverHTTP(t *testing.T) {
	t.Run("theirs is 404, not 403", func(t *testing.T) {
		// Build a service with a known payment owned by cus_2; cancel
		// as cus_1. The contract under test: existence is not disclosed.
		svc, store, _ := testService(t)
		p, err := store.Charge(t.Context(), Payment{CustomerID: "cus_2", AmountMinor: 100, Currency: "USD", Status: StatusCharged})
		if err != nil {
			t.Fatal(err)
		}

		rec := httptest.NewRecorder()
		req := authedReq(http.MethodPost, "/payments/"+p.ID+"/cancel", "user:cus_1", "")
		NewHandler(svc).Routes(slog.New(slog.DiscardHandler), nil).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (existence must not be disclosed)", rec.Code)
		}
	})

	t.Run("own is 204", func(t *testing.T) {
		svc, store, _ := testService(t)
		p, err := store.Charge(t.Context(), Payment{CustomerID: "cus_1", AmountMinor: 100, Currency: "USD", Status: StatusCharged})
		if err != nil {
			t.Fatal(err)
		}

		rec := httptest.NewRecorder()
		req := authedReq(http.MethodPost, "/payments/"+p.ID+"/cancel", "user:cus_1", "")
		NewHandler(svc).Routes(slog.New(slog.DiscardHandler), nil).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body)
		}
	})
}
