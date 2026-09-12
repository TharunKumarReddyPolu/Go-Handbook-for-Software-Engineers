package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Storer is consumer-side: the service declares what it needs, and
// implementations (Postgres, in-memory fake) satisfy it implicitly
// without importing this package.
type Storer interface {
	Save(ctx context.Context, u *User) error
	Get(ctx context.Context, id string) (*User, error)
}

// Notifier is a single-method capability seam.
type Notifier interface {
	Welcome(ctx context.Context, u *User) error
}

// Service holds dependencies, never per-request state. Safe for
// concurrent use exactly as far as its dependencies are.
type Service struct {
	store    Storer
	notifier Notifier
	now      func() time.Time
}

// NewService takes required dependencies positionally; the injected
// clock keeps every timestamp test deterministic.
func NewService(store Storer, notifier Notifier, now func() time.Time) *Service {
	return &Service{store: store, notifier: notifier, now: now}
}

// Register shows the happy path as a story: validate, save,
// notify-best-effort. Notification failure is logged, never returned:
// a business rule, visible in one glance.
func (s *Service) Register(ctx context.Context, email string) (*User, error) {
	u, err := NewUser(email, s.now())
	if err != nil {
		return nil, err // domain error: validation failed
	}
	if err := s.store.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("register %s: %w", u.ID(), err)
	}
	if err := s.notifier.Welcome(ctx, u); err != nil {
		slog.Error("welcome notification failed", "user", u.ID(), "err", err)
	}
	return u, nil
}

// Get delegates to storage; the error is wrapped with context.
func (s *Service) Get(ctx context.Context, id string) (*User, error) {
	u, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", id, err)
	}
	return u, nil
}
