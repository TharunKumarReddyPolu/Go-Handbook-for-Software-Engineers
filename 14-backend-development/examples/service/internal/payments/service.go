// Package payments is the domain core from chapter 1: the service
// owns business decisions and defines the store interface it needs.
// It imports no transport packages and constructs no infrastructure.
package payments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Domain errors: sentinels the transport maps (05 §2's pipeline).
// ErrForbidden deliberately maps to 404 at the transport: existence
// of someone else's payment is not disclosed (chapter 4's decision).
var (
	ErrNotFound          = errors.New("payment not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrForbidden         = errors.New("forbidden")
	ErrNotCancellable    = errors.New("payment not cancellable")
)

// Status is a payment's lifecycle state.
type Status string

const (
	StatusCharged   Status = "charged"
	StatusCancelled Status = "cancelled"
)

// Payment is the domain record.
type Payment struct {
	ID          string
	CustomerID  string
	AmountMinor int64 // integer minor units (25 §1): never float money
	Currency    string
	Status      Status
	CreatedAt   time.Time
}

// ChargeInput is the service's input contract (not a wire type).
type ChargeInput struct {
	CustomerID  string
	AmountMinor int64
	Currency    string
}

// Validate covers invariants that need no I/O; shape validation
// (JSON well-formedness) happened in transport, per chapter 4's
// split. Pure function: trivially testable.
func (in ChargeInput) Validate() error {
	var errs []error
	if strings.TrimSpace(in.CustomerID) == "" {
		errs = append(errs, errors.New("customer_id is required"))
	}
	if in.AmountMinor <= 0 {
		errs = append(errs, errors.New("amount_minor must be positive"))
	}
	if len(in.Currency) != 3 || strings.ToUpper(in.Currency) != in.Currency {
		errs = append(errs, errors.New("currency must be a 3-letter uppercase code"))
	}
	return errors.Join(errs...) // batched (chapter 2's rule)
}

// PaymentStore is the consumer-side interface (04 §3, 13 §4). The
// Postgres implementation lives behind it; tests use a fake.
type PaymentStore interface {
	Charge(ctx context.Context, p Payment) (Payment, error)
	ByID(ctx context.Context, id string) (Payment, error)
	Cancel(ctx context.Context, id string) error
	ByCustomer(ctx context.Context, customerID string, limit int) ([]Payment, error)
}

// Service makes decisions; the store persists.
type Service struct {
	store PaymentStore
	log   *slog.Logger
	now   func() time.Time // injected clock (chapter 3)
}

func NewService(store PaymentStore, log *slog.Logger) *Service {
	return &Service{store: store, log: log, now: time.Now}
}

// Now overrides the clock in tests.
func (s *Service) Now(now func() time.Time) { s.now = now }

func (s *Service) Charge(ctx context.Context, in ChargeInput) (Payment, error) {
	if err := in.Validate(); err != nil {
		return Payment{}, &validationError{joined: err}
	}
	p, err := s.store.Charge(ctx, Payment{
		CustomerID:  in.CustomerID,
		AmountMinor: in.AmountMinor,
		Currency:    in.Currency,
		Status:      StatusCharged,
		CreatedAt:   s.now(),
	})
	if err != nil {
		return Payment{}, err // classification happened at the store
	}
	s.log.InfoContext(ctx, "payment charged", "payment_id", p.ID, "customer_id", p.CustomerID)
	return p, nil
}

// Cancel enforces the fine-grained authz rule from chapter 4: the
// actor must own the payment. Coarse authz (route policy) already
// ran in middleware; this check cannot silently disappear.
func (s *Service) Cancel(ctx context.Context, actor Customer, paymentID string) error {
	p, err := s.store.ByID(ctx, paymentID)
	if err != nil {
		return err // 404 path: unknown or hidden
	}
	if p.CustomerID != actor.ID {
		return fmt.Errorf("cancel %s: %w", paymentID, ErrForbidden)
	}
	if p.Status != StatusCharged {
		return fmt.Errorf("%w: status %q", ErrNotCancellable, p.Status)
	}
	if err := s.store.Cancel(ctx, paymentID); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "payment cancelled", "payment_id", paymentID, "actor", actor.ID)
	return nil
}

func (s *Service) ByCustomer(ctx context.Context, customerID string, limit int) ([]Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 100 // list methods bound themselves (13 §4)
	}
	return s.store.ByCustomer(ctx, customerID, limit)
}

// Customer is the actor identity the service consumes (transport
// maps its authn Identity to this; the domain stays transport-free).
type Customer struct{ ID string }
