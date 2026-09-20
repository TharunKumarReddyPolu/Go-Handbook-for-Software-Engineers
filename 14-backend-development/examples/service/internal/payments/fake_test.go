package payments

import (
	"context"
	"fmt"
	"sync"
)

// fakeStore is the pure-Go PaymentStore for unit tests (13 Section 4's
// pattern). It mirrors the behavioral contract: atomic Charge with an
// overdraft guard, cancel transitions, bounded newest-first lists.
// Lives in the test files so it ships with the tests that justify it.
type fakeStore struct {
	mu       sync.Mutex
	seq      int
	payments map[string]Payment
	order    []string // insertion order for newest-first lists
}

func newFakeStore() *fakeStore {
	return &fakeStore{payments: map[string]Payment{}}
}

// compile-time check: the fake must always satisfy the interface.
var _ PaymentStore = (*fakeStore)(nil)

func (f *fakeStore) Charge(_ context.Context, p Payment) (Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	p.ID = fmt.Sprintf("pay_%04d", f.seq)
	f.payments[p.ID] = p
	f.order = append(f.order, p.ID)
	return p, nil
}

func (f *fakeStore) ByID(_ context.Context, id string) (Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.payments[id]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) Cancel(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.payments[id]
	if !ok {
		return ErrNotFound
	}
	p.Status = StatusCancelled
	f.payments[id] = p
	return nil
}

func (f *fakeStore) ByCustomer(_ context.Context, customerID string, limit int) ([]Payment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Payment
	for i := len(f.order) - 1; i >= 0 && len(out) < limit; i-- {
		if p := f.payments[f.order[i]]; p.CustomerID == customerID {
			out = append(out, p)
		}
	}
	return out, nil
}
