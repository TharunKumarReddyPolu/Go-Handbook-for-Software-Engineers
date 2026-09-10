package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func envelope(t *testing.T, v int, order OrderPayload) []byte {
	t.Helper()
	raw, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	env := Envelope{Version: v, Type: "order.created", Timestamp: time.Now(), Payload: raw}
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type countingApplier struct {
	mu     sync.Mutex
	calls  map[string]int
	result error
}

func newCountingApplier() *countingApplier { return &countingApplier{calls: map[string]int{}} }

func (c *countingApplier) Apply(_ context.Context, order OrderPayload) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[order.OrderID]++
	return c.result
}

func TestProcess_AppliesOnce(t *testing.T) {
	applier := newCountingApplier()
	svc := NewOrderService(NewMemoryClaims(), applier)
	msg := Message{
		Key:     "acct-1",
		Value:   envelope(t, 1, OrderPayload{OrderID: "o1", AccountID: "acct-1", AmountMinor: 1000, Currency: "USD"}),
		Headers: map[string]string{"idempotency_key": "k1"},
	}

	if err := svc.Process(context.Background(), msg); err != nil {
		t.Fatalf("first Process: %v", err)
	}
	// Duplicate delivery: must succeed AND not re-apply.
	if err := svc.Process(context.Background(), msg); err != nil {
		t.Fatalf("duplicate Process: %v", err)
	}
	if n := applier.calls["o1"]; n != 1 {
		t.Errorf("applier called %d times, want 1 (duplicate collapsed)", n)
	}
}

func TestProcess_ConcurrentDuplicatesCollapse(t *testing.T) {
	applier := newCountingApplier()
	svc := NewOrderService(NewMemoryClaims(), applier)
	msg := Message{
		Key:     "acct-1",
		Value:   envelope(t, 1, OrderPayload{OrderID: "o2", AccountID: "acct-1", AmountMinor: 2000, Currency: "USD"}),
		Headers: map[string]string{"idempotency_key": "k2"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			_ = svc.Process(context.Background(), msg) // exactly one wins; others no-op
		})
	}
	wg.Wait()
	if n := applier.calls["o2"]; n != 1 {
		t.Errorf("applier called %d times, want 1", n)
	}
}

func TestProcess_FailureReleasesClaim(t *testing.T) {
	applier := newCountingApplier()
	applier.result = errors.New("db down")
	svc := NewOrderService(NewMemoryClaims(), applier)
	msg := Message{
		Key:     "acct-1",
		Value:   envelope(t, 1, OrderPayload{OrderID: "o3", AccountID: "acct-1", AmountMinor: 3000, Currency: "USD"}),
		Headers: map[string]string{"idempotency_key": "k3"},
	}

	if err := svc.Process(context.Background(), msg); err == nil {
		t.Fatal("first Process should fail")
	}
	applier.result = nil // recovery
	if err := svc.Process(context.Background(), msg); err != nil {
		t.Fatalf("retry after recovery: %v", err)
	}
	if n := applier.calls["o3"]; n != 2 {
		t.Errorf("applier called %d times, want 2 (claim released on failure)", n)
	}
}

func TestProcess_TerminalErrors(t *testing.T) {
	svc := NewOrderService(NewMemoryClaims(), newCountingApplier())

	tests := []struct {
		name string
		msg  Message
	}{
		{"missing key", Message{Value: envelope(t, 1, OrderPayload{OrderID: "x"})}},
		{"bad json", Message{Headers: map[string]string{"idempotency_key": "k"}, Value: []byte("not json")}},
		{"future version", Message{Headers: map[string]string{"idempotency_key": "k"}, Value: envelope(t, 99, OrderPayload{OrderID: "x"})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Process(context.Background(), tt.msg)
			var term *TerminalError
			if !asTerminal(err, &term) {
				t.Fatalf("err = %v, want TerminalError", err)
			}
			if !errors.Is(err, ErrMalformed) {
				t.Errorf("err = %v, want wrapped ErrMalformed", err)
			}
		})
	}
}
