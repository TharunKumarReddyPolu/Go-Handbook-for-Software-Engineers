// Package errorslib is the handbook's worked example of production error
// design: a sentinel, a typed domain error with a classification code,
// and a retryability predicate. Companion to 05-errors chapters 1 and 2.
package errorslib

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrNotFound is returned when a lookup finds nothing. Sentinel errors
// are an API contract: once published, they are forever.
var ErrNotFound = errors.New("not found")

// NotFound wraps ErrNotFound with lookup context.
func NotFound(what string) error {
	return fmt.Errorf("%s: %w", what, ErrNotFound)
}

// Load demonstrates the inspect-at-caller pattern with a failing lookup.
func Load(id string) (string, error) {
	return "", NotFound("user " + id)
}

// IsNotFound inspects the wrapped chain.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// Code classifies errors for callers that switch on behavior.
type Code int

const (
	CodeUnknown Code = iota
	CodeNotFound
	CodeInvalid
	CodeConflict
	CodeUnavailable // transient: safe to retry
)

// DomainError carries structured fields callers can act on. It joins
// error chains via Unwrap, so errors.Is/As see through it.
type DomainError struct {
	Op      string // failing operation, e.g. "withdraw"
	Reason  string // human-readable cause
	Code    Code   // machine-readable classification
	wrapped error  // optional cause
}

func (e *DomainError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Reason, e.wrapped)
	}
	return e.Op + ": " + e.Reason
}

// Unwrap makes the DomainError transparent to errors.Is/As.
func (e *DomainError) Unwrap() error { return e.wrapped }

// Wrap builds a DomainError over an optional cause.
func Wrap(op string, code Code, reason string, cause error) *DomainError {
	return &DomainError{Op: op, Reason: reason, Code: code, wrapped: cause}
}

// Retryable classifies whether retrying may plausibly succeed. Wrong
// classification is expensive: retrying a 400 doubles load; not
// retrying a timeout turns blips into outages.
//
// Context errors are deliberately NOT classified as retryable: the
// caller knows why the deadline fired (own budget vs downstream stall)
// and whether the operation is safe to repeat. See the decision table
// in 05-errors/02-error-design.md.
func Retryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false // caller decides; retrying blind doubles load
	}
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == CodeUnavailable
	}
	var nerr net.Error
	if errors.As(err, &nerr) {
		return nerr.Timeout() // network timeouts: retry with backoff
	}
	return false
}
