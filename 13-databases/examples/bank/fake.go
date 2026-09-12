package bank

import (
	"context"
	"fmt"
	"sync"
)

// FakeStore is the pure-Go PaymentStore used by unit tests. Same
// behavioral contract as the Postgres implementation: Charge is
// atomic per call and refuses overdrafts, so service tests exercise
// real semantics in microseconds. Safe for concurrent use, matching
// the real store.
type FakeStore struct {
	mu       sync.Mutex
	seq      int
	accounts map[string]Account
	payments []Payment
}

// NewFakeStore seeds one account per customer ID with the given
// balance (minor units) in the given currency.
func NewFakeStore(balances map[string]int64, currency string) *FakeStore {
	fs := &FakeStore{accounts: map[string]Account{}}
	for id, bal := range balances {
		fs.accounts[id] = Account{CustomerID: id, BalanceMinor: bal, Currency: currency}
	}
	return fs
}

func (f *FakeStore) Charge(_ context.Context, p Payment) (Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	acct, ok := f.accounts[p.CustomerID]
	if !ok {
		return Payment{}, fmt.Errorf("account %q: %w", p.CustomerID, ErrNotFound)
	}
	// The Postgres version enforces this in SQL (balance >= $1); the
	// fake mirrors the *behavior* so service tests are honest.
	if acct.BalanceMinor < p.AmountMinor {
		return Payment{}, ErrInsufficientFunds
	}
	f.seq++
	p.ID = fmt.Sprintf("pay_%04d", f.seq)
	p.Status = "charged"
	acct.BalanceMinor -= p.AmountMinor
	f.accounts[p.CustomerID] = acct
	f.payments = append(f.payments, p)
	return p, nil
}

func (f *FakeStore) ByCustomer(_ context.Context, customerID string, limit int) ([]Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []Payment
	// Newest first: the fake stores oldest-first, so walk backwards.
	for i := len(f.payments) - 1; i >= 0 && len(out) < limit; i-- {
		if f.payments[i].CustomerID == customerID {
			out = append(out, f.payments[i])
		}
	}
	return out, nil
}

func (f *FakeStore) Account(_ context.Context, customerID string) (Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	acct, ok := f.accounts[customerID]
	if !ok {
		return Account{}, fmt.Errorf("account %q: %w", customerID, ErrNotFound)
	}
	return acct, nil
}

// Payments returns all recorded payments, oldest first (test aid).
func (f *FakeStore) Payments() []Payment {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Payment, len(f.payments))
	copy(out, f.payments)
	return out
}
