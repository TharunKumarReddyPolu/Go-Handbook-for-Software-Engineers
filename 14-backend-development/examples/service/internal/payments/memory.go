package payments

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStore is the in-memory PaymentStore wired when STORE=memory
// (the example's zero-setup default) and used directly by unit tests.
// Behavioral contract matches the Postgres implementation: Charge is
// atomic per call, cancel transitions states, lists are bounded and
// newest-first. Not durable across restarts: that is the point of the
// config switch, not a hidden fallback.
type MemoryStore struct {
	mu       sync.Mutex
	seq      int
	payments map[string]Payment
	order    []string
}

var _ PaymentStore = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{payments: map[string]Payment{}}
}

func (m *MemoryStore) Charge(_ context.Context, p Payment) (Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	p.ID = fmt.Sprintf("pay_%04d", m.seq)
	m.payments[p.ID] = p
	m.order = append(m.order, p.ID)
	return p, nil
}

func (m *MemoryStore) ByID(_ context.Context, id string) (Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[id]
	if !ok {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

func (m *MemoryStore) Cancel(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[id]
	if !ok {
		return ErrNotFound
	}
	p.Status = StatusCanceled
	m.payments[id] = p
	return nil
}

func (m *MemoryStore) ByCustomer(_ context.Context, customerID string, limit int) ([]Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Payment
	for i := len(m.order) - 1; i >= 0 && len(out) < limit; i-- {
		if p := m.payments[m.order[i]]; p.CustomerID == customerID {
			out = append(out, p)
		}
	}
	return out, nil
}
