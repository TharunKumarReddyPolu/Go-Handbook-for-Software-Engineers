package di

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	testTime = time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	testCtx  = context.Background()
)

func TestRegister_HappyPath(t *testing.T) {
	store := NewMemStore()
	notifier := &RecordingNotifier{}
	svc := NewService(store, notifier, FixedClock(testTime))

	u, err := svc.Register(testCtx, "Ada@example.com ")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.Email() != "ada@example.com" {
		t.Fatalf("email not normalized: %q", u.Email())
	}
	if u.OpenedAt() != testTime {
		t.Fatalf("injected clock not used: %v", u.OpenedAt())
	}
	if store.Len() != 1 {
		t.Fatalf("user not stored: %d", store.Len())
	}
	if len(notifier.welcomed) != 1 {
		t.Fatalf("welcome not sent: %d", len(notifier.welcomed))
	}
}

func TestNewUser_ValidationTable(t *testing.T) {
	tests := []struct {
		email string
		want  bool // want success
	}{
		{"ada@example.com", true},
		{"  bob@example.com  ", true}, // trimmed
		{"", false},
		{"no-at-sign", false},
		{"@leading", false},
		{"trailing@", false},
	}
	for _, tt := range tests {
		u, err := NewUser(tt.email, testTime)
		if tt.want && err != nil {
			t.Errorf("NewUser(%q): %v, want success", tt.email, err)
		}
		if !tt.want {
			if err == nil {
				t.Errorf("NewUser(%q): success, want validation error", tt.email)
			}
			if u != nil {
				t.Errorf("NewUser(%q): returned user with error; must be nil", tt.email)
			}
		}
	}
}

func TestRegister_StorageFailurePropagates(t *testing.T) {
	// A store that fails: the seam makes failure injectable without
	// any framework. The fake returns the shared sentinel so the
	// errors.Is chain check is meaningful.
	store := failingStore{err: errDbDown}
	svc := NewService(store, &RecordingNotifier{}, FixedClock(testTime))

	_, err := svc.Register(testCtx, "ada@example.com")
	if err == nil {
		t.Fatal("storage failure must propagate")
	}
	if !errors.Is(err, errDbDown) {
		t.Fatalf("error chain lost sentinel: %v", err)
	}
}

var errDbDown = errors.New("db down")

type failingStore struct{ err error }

func (f failingStore) Save(_ context.Context, _ *User) error { return f.err }
func (f failingStore) Get(_ context.Context, _ string) (*User, error) {
	return nil, f.err
}

func TestRegister_NotificationFailureDoesNotAbort(t *testing.T) {
	// The business rule: welcome failure is logged, registration
	// succeeds. The FailingNotifier proves the edge is best-effort.
	store := NewMemStore()
	svc := NewService(store, &FailingNotifier{err: errors.New("smtp down")}, FixedClock(testTime))

	u, err := svc.Register(testCtx, "ada@example.com")
	if err != nil {
		t.Fatalf("notification failure must not abort registration: %v", err)
	}
	if store.Len() != 1 {
		t.Fatal("user should still be stored")
	}
	_ = u
}

func TestGet_NotFound(t *testing.T) {
	svc := NewService(NewMemStore(), &RecordingNotifier{}, FixedClock(testTime))

	_, err := svc.Get(testCtx, "u_missing")
	if err == nil {
		t.Fatal("missing user must error")
	}
}

// TestInterfaceSatisfaction pins the method sets: if User ever gains a
// pointer-receiver method that breaks its value form, these fail at
// compile time.
var (
	_ = func() { var _ Storer = (*MemStore)(nil) }
)
