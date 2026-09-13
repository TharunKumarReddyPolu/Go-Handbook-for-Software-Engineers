// Package messaging is the handbook's worked example for
// 17-messaging: a minimal in-memory queue broker implementing the
// queue model's exact semantics: ack, redeliver with attempt counts
// and backoff, dead-letter with triage metadata, plus the envelope
// versioning rule from chapter 4. The test suite in broker_test.go is
// the portability suite chapter 3 defines: the same assertions must
// pass against any real broker adapter.
package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Envelope is the versioned wire shape from chapter 4: version and
// type are checked at decode time; unknown versions are terminal.
type Envelope struct {
	Version int             `json:"version"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// DecodeEnvelope enforces the version contract. A missing or
// unsupported version is terminal: never decode with defaults.
func DecodeEnvelope(value []byte, supported map[int]func(json.RawMessage) error) error {
	var env Envelope
	if err := json.Unmarshal(value, &env); err != nil {
		return Terminal(fmt.Errorf("bad envelope: %w", err))
	}
	decode, ok := supported[env.Version]
	if !ok {
		return Terminal(fmt.Errorf("unsupported envelope version %d", env.Version))
	}
	return decode(env.Payload)
}

// --- error classification: 05 §2's table, 18 §3's marker ---

type TerminalError struct{ Cause error }

func (e *TerminalError) Error() string { return "terminal: " + e.Cause.Error() }
func (e *TerminalError) Unwrap() error { return e.Cause }

// Terminal marks an error non-retryable: the message is broken in a
// way no future attempt can fix, so the consumer skips the retry
// ladder and dead-letters immediately.
func Terminal(err error) error { return &TerminalError{Cause: err} }

// IsTerminal reports whether err was marked terminal.
func IsTerminal(err error) bool {
	var te *TerminalError
	return errors.As(err, &te)
}

// --- the broker ---

// Message is the transport-neutral shape every adapter translates to
// (18 §3's Message, generalized): the idempotency key travels with
// the message, Attempts counts deliveries.
type Message struct {
	ID             string
	Queue          string
	IdempotencyKey string
	Attempts       int
	Value          []byte

	// DeadLetterReason and FirstFailedAt are the triage metadata
	// chapter 3 promises; they are set when a failure first occurs
	// and survive redeliveries.
	DeadLetterReason string
	FirstFailedAt    time.Time
}

// Handler is the transport-free contract (18 §3's pattern): domain
// code implements this; brokers and tests call it.
type Handler interface {
	Process(ctx context.Context, msg Message) error
}

// HandlerFunc adapts functions to Handler.
type HandlerFunc func(ctx context.Context, msg Message) error

func (f HandlerFunc) Process(ctx context.Context, msg Message) error { return f(ctx, msg) }

// Option configures a Broker at construction.
type Option func(*Broker)

// WithMaxAttempts caps deliveries per message (default 3).
func WithMaxAttempts(n int) Option { return func(b *Broker) { b.maxAttempts = n } }

// WithBackoff sets the base redelivery delay; the delay doubles per
// attempt (default 1ms).
func WithBackoff(d time.Duration) Option { return func(b *Broker) { b.backoff = d } }

// WithClock injects the time source. Tests advance the injected clock
// to elapse backoff deterministically instead of sleeping.
func WithClock(now func() time.Time) Option { return func(b *Broker) { b.now = now } }

// Broker is the in-memory queue model: each message is delivered to
// ONE consumer, redelivered with backoff on retryable failure, and
// dead-lettered on terminal failure or attempt exhaustion.
//
// Redelivery never blocks the consumer and never spawns timers: a
// failed message parks in inflight with a visibleAt deadline, and the
// next poll after that deadline returns it to the ready queue. This
// is the scheduling model of a visibility timeout (SQS, JetStream
// ack wait) in miniature.
type Broker struct {
	mu          sync.Mutex
	ready       map[string][]*pending // visible (or future-visible) messages, FIFO per queue
	inflight    map[string]*pending   // parked for redelivery until visibleAt
	dlq         map[string][]Message  // queue -> dead-lettered messages
	maxAttempts int
	backoff     time.Duration
	now         func() time.Time
}

type pending struct {
	msg       Message
	visibleAt time.Time
}

// NewBroker returns a broker with defaults: 3 attempts, 1ms base
// backoff, wall clock.
func NewBroker(opts ...Option) *Broker {
	b := &Broker{
		ready:       map[string][]*pending{},
		inflight:    map[string]*pending{},
		dlq:         map[string][]Message{},
		maxAttempts: 3,
		backoff:     time.Millisecond,
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Publish enqueues one message. Attempts 0 means "first delivery";
// the counter tracks which delivery this is, so the first is 1.
func (b *Broker) Publish(queue string, msg Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg.Queue = queue
	if msg.Attempts == 0 {
		msg.Attempts = 1
	}
	b.ready[queue] = append(b.ready[queue], &pending{msg: msg})
}

// next returns the next visible message for the queue, first
// promoting any parked redeliveries whose backoff has elapsed.
func (b *Broker) next(queue string) (Message, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	for id, p := range b.inflight {
		if !now.Before(p.visibleAt) {
			delete(b.inflight, id)
			b.ready[p.msg.Queue] = append(b.ready[p.msg.Queue], p)
		}
	}
	q := b.ready[queue]
	for i, p := range q {
		if !now.Before(p.visibleAt) {
			b.ready[queue] = append(q[:i], q[i+1:]...)
			return p.msg, true
		}
	}
	return Message{}, false
}

// dead moves a message to the DLQ with full triage metadata.
func (b *Broker) dead(queue string, msg Message, cause error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg.DeadLetterReason = cause.Error()
	if msg.FirstFailedAt.IsZero() {
		msg.FirstFailedAt = b.now()
	}
	b.dlq[queue] = append(b.dlq[queue], msg)
}

// Dead returns the DLQ contents for a queue (the triage surface).
func (b *Broker) Dead(queue string) []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Message, len(b.dlq[queue]))
	copy(out, b.dlq[queue])
	return out
}

// Inflight reports how many messages are parked awaiting redelivery.
func (b *Broker) Inflight() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.inflight)
}

// Consume drains a queue through the handler with the full policy:
// ack on success, dead-letter terminal errors, redeliver retryable
// ones until MaxAttempts. It is the consumer loop from chapter 3.
// Redeliveries are parked, not blocked: the caller re-invokes Consume
// (or advances the clock in tests) once the backoff has elapsed.
func (b *Broker) Consume(ctx context.Context, queue string, h Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, ok := b.next(queue)
		if !ok {
			return nil // nothing visible; this minimal broker does not block
		}
		err := b.handle(ctx, h, msg)
		switch {
		case err == nil:
			// Ack: done forever; the queue model keeps no record.
		case IsTerminal(err):
			b.dead(queue, msg, err)
		default:
			if msg.Attempts >= b.maxAttempts {
				b.dead(queue, msg, fmt.Errorf("max attempts (%d) reached: %w", b.maxAttempts, err))
				continue
			}
			b.redeliver(queue, msg, err)
		}
	}
}

// handle runs the handler with panic recovery: a panicking handler
// is the most terminal failure there is (the process may be dying),
// so the message is marked terminal and dead-lettered with the
// panic attached rather than retried blindly.
func (b *Broker) handle(ctx context.Context, h Handler, msg Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Terminal(fmt.Errorf("panic in handler: %v", r))
		}
	}()
	return h.Process(ctx, msg)
}

// redeliver parks the message for one backoff interval, doubling
// per attempt, and stamps FirstFailedAt on the first failure. The
// message leaves the ready queue and returns through it after the
// delay: scheduling, not blocking. The handler error is carried on
// the parked message, so a later DLQ trip preserves the first
// failure's cause and timestamp for triage.
func (b *Broker) redeliver(queue string, msg Message, cause error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if msg.FirstFailedAt.IsZero() {
		msg.FirstFailedAt = b.now()
	}
	msg.Queue = queue
	msg.Attempts++
	msg.DeadLetterReason = cause.Error()     // provisional until DLQ or success
	delay := b.backoff << (msg.Attempts - 2) // attempt 2: 1x, 3: 2x, 4: 4x
	b.inflight[msg.ID] = &pending{msg: msg, visibleAt: b.now().Add(delay)}
}
