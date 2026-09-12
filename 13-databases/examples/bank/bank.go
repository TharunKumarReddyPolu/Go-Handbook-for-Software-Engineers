// Package bank is the handbook's worked example for 13-databases: a
// payments store with the repository boundary from chapter 4. The
// interface, fake, and transaction helper are pure Go and tested in
// CI; the Postgres implementation (postgres.go) carries the real SQL
// and is tested under the `db` build tag.
//
// Domain rule under test everywhere: an overdraft is impossible, and
// that guarantee lives in SQL (balance >= $1), not in application
// check-then-act logic.
package bank

import (
	"context"
	"errors"
)

// ErrInsufficientFunds is returned when an account cannot cover a
// charge. Domain sentinel: driver errors never escape the repo.
var ErrInsufficientFunds = errors.New("insufficient funds")

// ErrNotFound is returned for unknown accounts or payments.
var ErrNotFound = errors.New("not found")

// Payment is one charge against one customer account.
type Payment struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"` // integer minor units, per 25 §1
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

// Account is a customer's payable balance.
type Account struct {
	CustomerID   string `json:"customer_id"`
	BalanceMinor int64  `json:"balance_minor"`
	Currency     string `json:"currency"`
}

// PaymentStore is the consumer-side interface from 13 §4: domain-
// shaped methods, stated in the service's language. The Postgres
// implementation and the test fake both satisfy it.
type PaymentStore interface {
	// Charge debits the account and records the payment atomically.
	// Insufficient balance returns ErrInsufficientFunds and changes
	// nothing.
	Charge(ctx context.Context, p Payment) (Payment, error)

	// ByCustomer returns at most limit payments, newest first.
	ByCustomer(ctx context.Context, customerID string, limit int) ([]Payment, error)

	// Account returns the account for a customer.
	Account(ctx context.Context, customerID string) (Account, error)
}
