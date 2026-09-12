// Package main is the handbook's runnable example for section 12: a
// payments API built entirely from the standard library, composing the
// five chapters: routing (01), middleware (02), the JSON boundary (03),
// error mapping from 05-errors, and graceful shutdown (05).
//
// One file for teaching; in a real service these pieces live in
// separate packages (see 14-backend-development when it ships).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------
// Domain: a tiny in-memory store so the example runs with zero setup.
// The interface seam is what tests fake and what a database would
// implement in a real service.
// ---------------------------------------------------------------------------

// ErrNotFound is the sentinel the store returns for missing payments.
var ErrNotFound = errors.New("payment not found")

// ValidationError marks request bodies that parsed but are invalid.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Payment is the domain record.
type Payment struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"` // integer minor units; never float money
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

// Store is the persistence seam.
type Store interface {
	Create(ctx context.Context, p Payment) (Payment, error)
	Get(ctx context.Context, id string) (Payment, error)
}

// MemoryStore is a store safe for concurrent handlers.
type MemoryStore struct {
	mu   sync.RWMutex
	byID map[string]Payment
	seq  int
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{byID: map[string]Payment{}} }

func (s *MemoryStore) Create(_ context.Context, p Payment) (Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	p.ID = fmt.Sprintf("pay_%04d", s.seq)
	p.Status = "created"
	s.byID[p.ID] = p
	return p, nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// The JSON boundary helpers from chapter 03.
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are committed; the client sees a truncated body. Log
		// only: there is no response left to write.
		slog.Error("encode response", "err", err)
	}
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("decode body: %w", err)
	}
	return v, nil
}

// writeError is the single translation point from 05-errors/02: domain
// classes in, status codes out. Clients see classification, not text.
func writeError(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		http.Error(w, ve.Msg, http.StatusUnprocessableEntity)
	case errors.Is(err, ErrNotFound):
		http.Error(w, "payment not found", http.StatusNotFound)
	default:
		// Unknown = our fault. Generic body; detail belongs in logs.
		slog.Error("internal error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// ---------------------------------------------------------------------------
// Middleware from chapter 02. Each does one thing; Chain composes them
// so the first listed is outermost.
// ---------------------------------------------------------------------------

type middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- { // reverse: first listed = outermost
		h = mws[i](h)
	}
	return h
}

// statusWriter records whether and what was written. Recover and
// Logging both need it: the first to avoid writing a 500 over a
// started response, the second to log a real status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (c *statusWriter) WriteHeader(code int) {
	if c.status == 0 {
		c.status = code
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *statusWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK // implicit 200 on bare Write
	}
	return c.ResponseWriter.Write(b)
}

// Recover catches handler panics. It only writes a 500 when the
// response has not started; otherwise the write would be dropped.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic in handler", "rec", rec, "path", r.URL.Path)
				if sw.status == 0 {
					http.Error(sw, "internal error", http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

// RequestID assigns an ID per request and echoes it back; real tracing
// would use OpenTelemetry (20-observability when it ships).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req_%d", time.Now().UnixNano())
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// Logging emits one line per request with status and duration.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		slog.Info("http", "method", r.Method, "path", r.URL.Path,
			"status", sw.status, "dur", time.Since(start).String())
	})
}

// ---------------------------------------------------------------------------
// Server: dependencies as fields, handlers as methods (chapter 01).
// ---------------------------------------------------------------------------

type Server struct {
	store Store
	ready atomic.Bool
}

func NewServer(store Store) *Server {
	s := &Server{store: store}
	s.ready.Store(true)
	return s
}

type createPaymentRequest struct {
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[createPaymentRequest](w, r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// Validation at the boundary (chapter 03): batched, pure, testable.
	var verrs []string
	if req.CustomerID == "" {
		verrs = append(verrs, "customer_id is required")
	}
	if req.AmountMinor <= 0 {
		verrs = append(verrs, "amount_minor must be positive")
	}
	if len(req.Currency) != 3 || strings.ToUpper(req.Currency) != req.Currency {
		verrs = append(verrs, "currency must be a 3-letter uppercase code")
	}
	if len(verrs) > 0 {
		writeError(w, &ValidationError{Msg: strings.Join(verrs, "; ")})
		return
	}

	p, err := s.store.Create(r.Context(), Payment{
		CustomerID: req.CustomerID, AmountMinor: req.AmountMinor,
		Currency: req.Currency,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/payments/"+p.ID)
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetPayment(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ok")
}

// Handler assembles the full chain. Kept as a method so tests (and
// main) construct the exact production stack.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("POST /payments", s.handleCreatePayment)
	mux.HandleFunc("GET /payments/{id}", s.handleGetPayment)

	// Bound every handler (chapter 02): stdlib TimeoutHandler, 503 on
	// overrun, no hand-rolled select loop.
	bounded := http.TimeoutHandler(mux, 10*time.Second, "timeout\n")

	return Chain(bounded, Logging, RequestID, Recover)
}

// ---------------------------------------------------------------------------
// main: the graceful-shutdown lifecycle from chapter 05.
// ---------------------------------------------------------------------------

func main() {
	s := NewServer(NewMemoryStore())
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		// Server failed to start (port taken, ...): a real failure.
		slog.Error("server failed", "err", err)
		os.Exit(1)
	case <-ctx.Done(): // SIGTERM / SIGINT received
	}

	// Graceful path in the order from chapter 05:
	// 1. flip readiness so the LB stops routing,
	// 2. drain in-flight requests within the grace period,
	// 3. exit. (A real service would also flush async producers
	// before closing; see 18-kafka-with-go.)
	s.ready.Store(false)
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced close after grace period", "err", err)
		_ = srv.Close()
	}
	slog.Info("bye")
}
