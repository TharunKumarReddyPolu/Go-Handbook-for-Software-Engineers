// Package ledger is the handbook's educational double-entry ledger:
// integer minor units, balanced immutable entries, derived balances.
// Educational -- not an audited production system. See
// 25-fintech-with-go chapters 1-3 for the design reasoning.
package ledger

import (
	"errors"
	"fmt"
)

// ErrCurrencyMismatch is returned when operations combine currencies.
var ErrCurrencyMismatch = errors.New("currency mismatch")

// ErrInvalidAmount is returned for malformed amounts or currencies.
var ErrInvalidAmount = errors.New("invalid amount")

// Money is an amount in minor units with its currency. The zero value
// is deliberately invalid: New is the only constructor, because a
// silently-zero Money value would be a loss. See the zero-values
// chapter in 01-go-fundamentals for the tradeoff discussion.
type Money struct {
	amount   int64
	currency string
}

// New validates and constructs. Currency is an ISO-4217 code; amount
// is minor units (cents for USD).
func New(amount int64, currency string) (Money, error) {
	if len(currency) != 3 {
		return Money{}, fmt.Errorf("%w: currency %q", ErrInvalidAmount, currency)
	}
	return Money{amount: amount, currency: currency}, nil
}

// MustNew is for tests and package-level constants where validation is
// statically known to pass.
func MustNew(amount int64, currency string) Money {
	m, err := New(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return m.currency }

// Add combines two monies; operands not built via New (the zero value)
// and mismatched currencies are errors, never a silent sum.
func (m Money) Add(o Money) (Money, error) {
	if len(m.currency) != 3 || len(o.currency) != 3 {
		return Money{}, fmt.Errorf("%w: operand not built via New", ErrInvalidAmount)
	}
	if m.currency != o.currency {
		return Money{}, fmt.Errorf("%w: %s + %s", ErrCurrencyMismatch, m.currency, o.currency)
	}
	return Money{amount: m.amount + o.amount, currency: m.currency}, nil
}

// Negate returns -m: the credit side of a debit/credit pair.
func (m Money) Negate() Money { return Money{amount: -m.amount, currency: m.currency} }

func (m Money) IsZero() bool     { return m.amount == 0 }
func (m Money) IsPositive() bool { return m.amount > 0 }

func (m Money) String() string {
	// Educational formatting: minor units shown raw. Production
	// formatting is per-currency (see ISO-4217 minor units).
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
