package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Handler tests: NewRequest + NewRecorder, no ports (see 10-testing/02).
// ---------------------------------------------------------------------------

func TestCreatePayment(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
		loc    string
	}{
		{"valid", `{"customer_id":"cus_1","amount_minor":4200,"currency":"USD"}`, http.StatusCreated, "/payments/pay_0001"},
		{"missing customer", `{"amount_minor":4200,"currency":"USD"}`, http.StatusUnprocessableEntity, ""},
		{"zero amount", `{"customer_id":"cus_1","amount_minor":0,"currency":"USD"}`, http.StatusUnprocessableEntity, ""},
		{"bad currency", `{"customer_id":"cus_1","amount_minor":1,"currency":"usd"}`, http.StatusUnprocessableEntity, ""},
		{"unparseable body", `{oops`, http.StatusBadRequest, ""},
		{"unknown field", `{"customer_id":"c","amount_minor":1,"currency":"USD","x":1}`, http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(NewMemoryStore())
			req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			s.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.status, rec.Body)
			}
			if tt.loc != "" {
				if got := rec.Header().Get("Location"); got != tt.loc {
					t.Errorf("Location = %q, want %q", got, tt.loc)
				}
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") && tt.status == http.StatusCreated {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestGetPayment_NotFound(t *testing.T) {
	s := NewServer(NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/payments/pay_9999", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "payment not found") == false {
		t.Errorf("body should carry the safe classification, got %q", rec.Body)
	}
}

func TestGetPayment_Found(t *testing.T) {
	s := NewServer(NewMemoryStore())
	created, err := s.store.Create(t.Context(), Payment{CustomerID: "c", AmountMinor: 5, Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/payments/"+created.ID, nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != created.ID || got.AmountMinor != 5 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestMethodMismatch_Gets405(t *testing.T) {
	s := NewServer(NewMemoryStore())
	req := httptest.NewRequest(http.MethodDelete, "/payments/pay_1", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Errorf("Allow header = %q, want it to list GET", allow)
	}
}

// ---------------------------------------------------------------------------
// Middleware behavior.
// ---------------------------------------------------------------------------

func TestRecover_PanicBecomes500(t *testing.T) {
	boom := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	h := Chain(boom, Recover)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req) // must not panic the test process

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRecover_No500AfterWrite(t *testing.T) {
	// A handler that writes 200 then panics: the client must get the
	// 200 it was promised, not a dropped second write.
	latePanic := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "partial")
		panic("boom")
	})
	h := Chain(latePanic, Recover)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no rewrite after commit)", rec.Code)
	}
}

func TestRequestID_Echoed(t *testing.T) {
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}), RequestID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID should be set on every response")
	}
}

// ---------------------------------------------------------------------------
// Readiness + graceful shutdown, in-process (chapter 05).
// ---------------------------------------------------------------------------

func TestReadiness_Draining(t *testing.T) {
	s := NewServer(NewMemoryStore())
	s.ready.Store(false)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 while draining", rec.Code)
	}
}

func TestGracefulShutdown_CompletesInFlight(t *testing.T) {
	// The shutdown contract, verified in-process: after Shutdown
	// starts, the listener refuses new connections but an in-flight
	// request still completes with its full response.
	var started atomic.Bool
	block := make(chan struct{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		started.Store(true)
		<-block // hold the request open until the test releases it
		fmt.Fprint(w, "done")
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go func() { _ = srv.Serve(ln) }() // Serve returns ErrServerClosed on Shutdown

	client := &http.Client{Timeout: 5 * time.Second}
	done := make(chan string, 1)
	go func() {
		resp, err := client.Get("http://" + ln.Addr().String() + "/slow")
		if err != nil {
			done <- "ERR: " + err.Error()
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 16)
		n, _ := resp.Body.Read(buf)
		done <- string(buf[:n])
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !started.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !started.Load() {
		t.Fatal("slow handler never started")
	}

	// Begin shutdown with a generous grace period.
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- srv.Shutdown(ctx)
	}()

	time.Sleep(50 * time.Millisecond) // let Shutdown close the listener
	close(block)                      // release the in-flight request

	select {
	case body := <-done:
		if body != "done" {
			t.Fatalf("in-flight request got %q, want \"done\"", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete during shutdown")
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return after in-flight completed")
	}
}

// ---------------------------------------------------------------------------
// The store: concurrent safety under the race detector.
// ---------------------------------------------------------------------------

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p, err := s.Create(t.Context(), Payment{CustomerID: fmt.Sprint(n), AmountMinor: 1, Currency: "USD"})
			if err != nil {
				errs <- err
				return
			}
			if _, err := s.Get(t.Context(), p.ID); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
