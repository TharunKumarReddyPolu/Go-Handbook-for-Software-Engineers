package di

import (
	"context"
	"errors"
	"time"
)

// The fakes below are the entire testing apparatus: no mock library.
// Each implements exactly the consumer-side interface it fakes, and
// the compile-time checks keep them honest as the interfaces evolve.

var (
	_ Storer   = (*MemStore)(nil)
	_ Notifier = (*RecordingNotifier)(nil)
	_ Notifier = (*FailingNotifier)(nil)
)

// MemStore is the in-memory Storer: the "planned second backend" from
// the design clinic, doubling as a test double.
type MemStore struct {
	users map[string]*User
}

func NewMemStore() *MemStore {
	return &MemStore{users: make(map[string]*User)}
}

func (m *MemStore) Save(_ context.Context, u *User) error {
	m.users[u.ID()] = u
	return nil
}

func (m *MemStore) Get(_ context.Context, id string) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *MemStore) Len() int { return len(m.users) }

// RecordingNotifier records calls: state without a framework.
type RecordingNotifier struct {
	welcomed []*User
}

func (r *RecordingNotifier) Welcome(_ context.Context, u *User) error {
	r.welcomed = append(r.welcomed, u)
	return nil
}

// FailingNotifier simulates an outage: the best-effort edge's test.
type FailingNotifier struct{ err error }

func (f *FailingNotifier) Welcome(_ context.Context, _ *User) error {
	return f.err
}

// FixedClock is the injected time dependency.
func FixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}
