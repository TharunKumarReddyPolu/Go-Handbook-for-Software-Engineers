// Package main is a small but complete demonstration of error mapping at
// an HTTP boundary: a payment service that returns classified domain
// errors, and a handler that translates them to responses exactly once.
// Companion to 05-errors/02-error-design.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/05-errors/examples/errorslib"
)

// PaymentRequest is the wire input. AmountMinor is integer minor units
// (cents): never floats for money. See 25-fintech-with-go.
type PaymentRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
}

// validate returns a joined validation error listing every violation.
func (r PaymentRequest) validate() error {
	var errs []error
	if r.IdempotencyKey == "" {
		errs = append(errs, errors.New("idempotency_key is required"))
	}
	if r.AmountMinor <= 0 {
		errs = append(errs, errors.New("amount_minor must be positive"))
	}
	if len(r.Currency) != 3 {
		errs = append(errs, errors.New("currency must be a 3-letter code"))
	}
	return errors.Join(errs...)
}

// PaymentService is the domain layer: it returns classified errors and
// knows nothing about HTTP.
type PaymentService struct {
	// In production: repositories, authorizer clients. Here: stateless.
}

// Process authorizes a payment. The simulated paths show how a service
// layer converts infrastructure failures into classified domain errors.
//
// Note the cancellation handling: a raw context error is NOT retryable
// (errorslib.Retryable returns false for it — the caller must decide why
// the deadline fired), so the service translates it into CodeUnavailable
// when the right client-facing answer is "try again later".
func (s *PaymentService) Process(ctx context.Context, req PaymentRequest) error {
	if err := ctx.Err(); err != nil {
		return errorslib.Wrap("authorize", errorslib.CodeUnavailable,
			"request cancelled", err)
	}
	if req.AmountMinor > 100_000_00 { // policy: per-payment cap
		return errorslib.Wrap("authorize", errorslib.CodeInvalid,
			"amount exceeds per-payment limit", nil)
	}
	return nil
}

// writeError is the single translation point from errors to responses.
// Domain classification is authoritative; Retryable is the fallback for
// unclassified infrastructure errors (e.g. a raw net.Error timeout).
func writeError(w http.ResponseWriter, err error) {
	var de *errorslib.DomainError
	switch {
	case errors.As(err, &de) && de.Code == errorslib.CodeNotFound:
		http.Error(w, "resource not found", http.StatusNotFound)
	case errors.As(err, &de) && de.Code == errorslib.CodeInvalid:
		http.Error(w, de.Reason, http.StatusBadRequest)
	case errors.As(err, &de) && de.Code == errorslib.CodeConflict:
		http.Error(w, "conflicting state", http.StatusConflict)
	case errors.As(err, &de) && de.Code == errorslib.CodeUnavailable:
		w.Header().Set("Retry-After", "1")
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	case errorslib.Retryable(err):
		w.Header().Set("Retry-After", "1")
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// newHandler wires the service into HTTP. Separated from main so tests
// can exercise it without opening ports.
func newHandler(logger *slog.Logger) http.Handler {
	svc := &PaymentService{}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /payments", func(w http.ResponseWriter, r *http.Request) {
		var req PaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.WarnContext(r.Context(), "undecodable payment request", "err", err)
			http.Error(w, "malformed JSON body", http.StatusBadRequest)
			return
		}
		if err := req.validate(); err != nil {
			logger.WarnContext(r.Context(), "invalid payment request", "err", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := svc.Process(r.Context(), req); err != nil {
			logger.ErrorContext(r.Context(), "payment failed",
				"err", err, "idempotency_key", req.IdempotencyKey)
			writeError(w, err) // log once here, not inside the service
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	return mux
}

func main() {
	logger := slog.New(slog.NewJSONHandler(nil, nil))
	_ = http.ListenAndServe(":8080", newHandler(logger))
}
