package errorslib

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// asError defeats compile-time folding so the typed-nil test below
// verifies the runtime behavior rather than restating a statically
// known fact.
func asError(e *DomainError) error { return e }

func TestNotFound_WrapsSentinel(t *testing.T) {
	err := NotFound("user 42")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("NotFound() should wrap ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "user 42") {
		t.Errorf("error text should mention the id, got %q", err.Error())
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("42")
	if !IsNotFound(err) {
		t.Fatalf("Load should return not found, got %v", err)
	}
}

func TestWrap_Inspection(t *testing.T) {
	cause := context.DeadlineExceeded
	err := Wrap("authorize", CodeUnavailable, "authorizer did not respond", cause)

	var de *DomainError
	if !errors.As(err, &de) {
		t.Fatal("errors.As should find *DomainError through the chain")
	}
	if de.Code != CodeUnavailable {
		t.Errorf("Code = %d, want %d", de.Code, CodeUnavailable)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("chain should still answer errors.Is for the cause")
	}
	if got := err.Error(); got != "authorize: authorizer did not respond: context deadline exceeded" {
		t.Errorf("Error() = %q", got)
	}
}

func TestRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unavailable", Wrap("op", CodeUnavailable, "down", nil), true},
		{"invalid", Wrap("op", CodeInvalid, "bad input", nil), false},
		{"conflict", Wrap("op", CodeConflict, "stale", nil), false},
		{"not found", NotFound("user 1"), false},
		{"deadline", context.DeadlineExceeded, false}, // caller decides on context errors
		{"canceled", context.Canceled, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Retryable(tt.err); got != tt.want {
				t.Errorf("Retryable(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestTypedNilTrap(t *testing.T) {
	// Documents the typed-nil trap with its full shape:
	//
	// 1. A nil *DomainError converted to error is NOT == nil (the
	//    interface holds a type), so callers checking err != nil will
	//    treat it as a failure. Return literal nil from functions.
	// 2. errors.As still "finds" the type, but the target ends up nil,
	//    so callers must re-check for nil after As.
	err := asError(nil)
	if err == nil {
		t.Fatal("typed nil must not equal nil interface")
	}
	var got *DomainError
	if !errors.As(err, &got) {
		t.Fatal("errors.As matches the type even for a typed nil")
	}
	if got != nil {
		t.Error("errors.As target should be nil for a typed-nil error")
	}
}
