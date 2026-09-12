// Package httpjson is the platform JSON boundary from 12 §3: write,
// decode with limits, and the single error-mapping point. Platform
// code knows no domain.
package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// MaxBodyBytes bounds every decode; the request body is untrusted.
const MaxBodyBytes = 1 << 20

// Write encodes v as JSON with the given status. Encode errors are
// logged (headers are already committed; the client sees a truncated
// body).
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// Decode reads a bounded, strict JSON body into T.
func Decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("decode body: %w", err)
	}
	return v, nil
}

// ErrNoRows-shaped sentinel translation is the caller's job; this
// file stays domain-free. WriteError takes a mapper instead.
type ErrorMapper func(err error) (msg string, status int, ok bool)

// WriteError maps err through mapper; the fallback branch (unknown
// error) is the 500 with a generic body, logged here exactly once.
func WriteError(w http.ResponseWriter, err error, mapper ErrorMapper) {
	if mapper == nil {
		slog.Error("internal error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	msg, status, ok := mapper(err)
	if !ok {
		// Unknown = our fault. Detail in the log; generic body out.
		slog.Error("internal error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if status >= 500 {
		slog.Error("request failed", "err", err)
	}
	http.Error(w, msg, status)
}

// ErrBodyTooLarge is returned by Decode when the body exceeds the
// bound (transport wraps it as 413 if desired).
var ErrBodyTooLarge = errors.New("request body too large")
