// Package di is the handbook's worked example for 04-functions-methods-
// interfaces: the design clinic's UserService (chapter 4) built with
// consumer-side interfaces (chapter 3), constructor injection
// (chapter 5), and an invariant-enforcing domain type (chapter 6).
package di

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrValidation is the sentinel wrapping every validation failure; the
// domain never returns bare fmt.Errorf for rule violations (05-errors).
var ErrValidation = errors.New("validation failed")

// User is the domain type. Fields are unexported so every invariant
// (non-empty ID, valid email, OpenedAt set) holds by construction.
type User struct {
	id       string
	email    string
	openedAt time.Time
}

// NewUser is the gate: an invalid user is unrepresentable past it.
func NewUser(email string, now time.Time) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@") {
		return nil, fmt.Errorf("%w: email %q", ErrValidation, email)
	}
	return &User{id: "u_" + email, email: email, openedAt: now}, nil
}

func (u *User) ID() string          { return u.id }
func (u *User) Email() string       { return u.email }
func (u *User) OpenedAt() time.Time { return u.openedAt }
