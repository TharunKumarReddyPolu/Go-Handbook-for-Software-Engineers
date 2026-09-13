package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// clock is a deterministic test clock: Consume loops poll against
// it, and tests advance it to elapse backoff without sleeping.
type clock struct {
	now time.Time
}

func newClock() *clock                   { return &clock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} }
func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// newTestBroker wires a broker to a test clock so tests control
// backoff elapsing exactly.
func newTestBroker(t *testing.T) (*Broker, *clock) {
	t.Helper()
	c := newClock()
	b := NewBroker(WithMaxAttempts(3), WithBackoff(time.Second), WithClock(c.Now))
	return b, c
}

func TestConsume_AcksSuccess(t *testing.T) {
	b, _ := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1", Value: []byte("x")})

	processed := 0
	err := b.Consume(context.Background(), "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error {
			processed++
			return nil
		}))
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("handler ran %d times, want 1", processed)
	}
	// Acked: gone forever. A second consume finds nothing.
	processed = 0
	_ = b.Consume(context.Background(), "orders", HandlerFunc(func(ctx context.Context, msg Message) error {
		processed++
		return nil
	}))
	if processed != 0 {
		t.Fatalf("acked message was redelivered: ran %d times", processed)
	}
}

func TestConsume_TerminalSkipsRetryLadder(t *testing.T) {
	b, _ := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1", Value: []byte("garbage")})

	calls := 0
	_ = b.Consume(context.Background(), "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error {
			calls++
			return Terminal(errors.New("unparseable payload"))
		}))
	if calls != 1 {
		t.Fatalf("terminal error retried: handler ran %d times, want 1", calls)
	}
	dead := b.Dead("orders")
	if len(dead) != 1 {
		t.Fatalf("terminal message not dead-lettered, DLQ has %d", len(dead))
	}
	if !IsTerminal(Terminal(errors.New("x"))) {
		t.Fatal("IsTerminal(Terminal(err)) = false")
	}
}

func TestConsume_RedeliversWithBackoffUntilMaxAttempts(t *testing.T) {
	b, c := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1", Value: []byte("flaky")})

	// Attempts 1, 2, 3 fail; each parks with doubling backoff (1s,
	// 2s). Advance an hour per cycle: every parked message becomes
	// visible again. The first failure stamps FirstFailedAt; later
	// failures leave it alone.
	attempts := 0
	goErr := errors.New("dependency timeout")
	for {
		err := b.Consume(context.Background(), "orders", HandlerFunc(
			func(ctx context.Context, msg Message) error {
				attempts++
				return goErr
			}))
		if err != nil {
			t.Fatalf("Consume: %v", err)
		}
		if b.Inflight() == 0 {
			break
		}
		c.Advance(time.Hour)
	}
	if attempts != 3 {
		t.Fatalf("handler ran %d times, want 3 (max attempts)", attempts)
	}
	dead := b.Dead("orders")
	if len(dead) != 1 {
		t.Fatalf("want 1 dead-lettered message, got %d", len(dead))
	}
	m := dead[0]
	if m.Attempts != 3 {
		t.Errorf("dead message Attempts = %d, want 3", m.Attempts)
	}
	if !errors.Is(fmt.Errorf("%s", m.DeadLetterReason), goErr) && m.DeadLetterReason == "" {
		t.Errorf("DLQ message missing reason: %q", m.DeadLetterReason)
	}
	if m.FirstFailedAt.IsZero() {
		t.Error("DLQ message missing FirstFailedAt")
	}
	if !m.FirstFailedAt.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("FirstFailedAt = %v, want the first failure at t0", m.FirstFailedAt)
	}
}

func TestConsume_ExponentialBackoffDoubles(t *testing.T) {
	b, c := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1"})

	fail := func(ctx context.Context, msg Message) error { return errors.New("boom") }
	_ = b.Consume(context.Background(), "orders", HandlerFunc(fail)) // attempt 1 -> park 1s
	if b.Inflight() != 1 {
		t.Fatal("message not parked for redelivery")
	}
	c.Advance(time.Second + time.Millisecond)                        // elapse attempt 2's window
	_ = b.Consume(context.Background(), "orders", HandlerFunc(fail)) // attempt 2 -> park 2s
	// 1s of the new 2s window has elapsed; the message stays parked.
	c.Advance(time.Second)
	_ = b.Consume(context.Background(), "orders", HandlerFunc(fail))
	if len(b.Dead("orders")) != 0 {
		t.Fatal("message dead-lettered before its backoff elapsed")
	}
	if b.Inflight() != 1 {
		t.Fatal("message returned before doubled backoff elapsed")
	}
	c.Advance(time.Second + time.Millisecond)
	// Attempt 3: it fails and hits max attempts -> DLQ.
	_ = b.Consume(context.Background(), "orders", HandlerFunc(fail))
	if len(b.Dead("orders")) != 1 {
		t.Fatal("message did not reach DLQ at max attempts")
	}
}

func TestIdempotentConsumer_DedupesRedelivery(t *testing.T) {
	b, c := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1", IdempotencyKey: "order:48123:charge"})

	// The claim store from 15 §4, as a fake: one effect per key.
	claims := map[string]bool{}
	handler := HandlerFunc(func(ctx context.Context, msg Message) error {
		if msg.IdempotencyKey == "" {
			return Terminal(errors.New("missing idempotency key"))
		}
		if claims[msg.IdempotencyKey] {
			return nil // already processed: ack the redelivery
		}
		claims[msg.IdempotencyKey] = true
		return errors.New("effect applied, ack lost") // simulate crash-after-effect
	})

	// First consume: effect applied, ack lost, message parked.
	_ = b.Consume(context.Background(), "orders", handler)
	c.Advance(time.Hour)
	// Redelivery: the claim store collapses it to a no-op success.
	_ = b.Consume(context.Background(), "orders", handler)
	if len(b.Dead("orders")) != 0 {
		t.Fatalf("idempotent redelivery should ack, not dead-letter: %v", b.Dead("orders"))
	}
}

func TestConsume_MissingIdempotencyKeyIsTerminal(t *testing.T) {
	b, _ := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1"})

	_ = b.Consume(context.Background(), "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error {
			if msg.IdempotencyKey == "" {
				return Terminal(errors.New("missing idempotency key"))
			}
			return nil
		}))
	if len(b.Dead("orders")) != 1 {
		t.Fatal("unkeyed message should dead-letter, not retry")
	}
}

func TestConsume_PerKeyOrdering(t *testing.T) {
	b, _ := newTestBroker(t)
	// One key's messages publish in order; a single consumer must see
	// them in publish order (the queue model's FIFO guarantee).
	for i := 0; i < 5; i++ {
		b.Publish("orders", Message{ID: fmt.Sprintf("m%d", i), Value: []byte(fmt.Sprint(i))})
	}
	var seen []int
	_ = b.Consume(context.Background(), "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error {
			seen = append(seen, int(msg.Value[0]-'0'))
			return nil
		}))
	for i, v := range seen {
		if v != i {
			t.Fatalf("order violated: saw %v, want [0 1 2 3 4]", seen)
		}
	}
}

func TestConsume_PanicDeadLettersImmediately(t *testing.T) {
	b, _ := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1"})

	// Consume recovers handler panics and treats them as terminal:
	// a panicking handler is the most terminal failure there is.
	_ = b.Consume(context.Background(), "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error {
			panic("handler bug")
		}))
	dead := b.Dead("orders")
	if len(dead) != 1 {
		t.Fatalf("panicked message should be dead-lettered, DLQ has %d", len(dead))
	}
	if !strings.Contains(dead[0].DeadLetterReason, "panic") {
		t.Errorf("DLQ reason should mention the panic, got %q", dead[0].DeadLetterReason)
	}
}

func TestConsume_ContextCancelReturns(t *testing.T) {
	b, _ := newTestBroker(t)
	b.Publish("orders", Message{ID: "m1"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.Consume(ctx, "orders", HandlerFunc(
		func(ctx context.Context, msg Message) error { return nil })); !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume with canceled ctx = %v, want context.Canceled", err)
	}
}

func TestEnvelope_VersionContract(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{"order": "48123"})
	good, _ := json.Marshal(Envelope{Version: 2, Type: "order.created", Payload: payload})

	var gotOrder string
	supported := map[int]func(json.RawMessage) error{
		2: func(raw json.RawMessage) error {
			var m map[string]string
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			gotOrder = m["order"]
			return nil
		},
	}
	if err := DecodeEnvelope(good, supported); err != nil {
		t.Fatalf("DecodeEnvelope(v2): %v", err)
	}
	if gotOrder != "48123" {
		t.Fatalf("payload decode = %q, want 48123", gotOrder)
	}

	// Unknown version: terminal, never decoded with defaults.
	old, _ := json.Marshal(Envelope{Version: 9, Type: "order.created", Payload: payload})
	err := DecodeEnvelope(old, supported)
	if !IsTerminal(err) {
		t.Fatalf("unknown version should be terminal, got %v", err)
	}

	// Malformed envelope: terminal.
	err = DecodeEnvelope([]byte("{not json"), supported)
	if !IsTerminal(err) {
		t.Fatalf("malformed envelope should be terminal, got %v", err)
	}
}
