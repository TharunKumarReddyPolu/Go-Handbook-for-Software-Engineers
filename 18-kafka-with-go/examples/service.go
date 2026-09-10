// Package main holds the handbook's Kafka example: an order-events
// processor. This file is the transport-free core: the handler
// contract, idempotency claims, and domain logic that unit tests
// exercise without any broker. producer.go and consumer.go adapt
// franz-go around it. See 18-kafka-with-go/03-producer-consumer.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Message is the transport-neutral envelope. Handlers depend on this,
// never on client types: the seam that keeps the client swappable.
type Message struct {
	Key     string
	Value   []byte
	Headers map[string]string
}

// TerminalError marks failures that retries cannot fix (malformed
// input, unsupported schema versions): the consumer routes them to the
// dead-letter queue immediately.
type TerminalError struct{ Cause error }

func (e *TerminalError) Error() string { return "terminal: " + e.Cause.Error() }
func (e *TerminalError) Unwrap() error { return e.Cause }

// Terminal wraps err as non-retryable.
func Terminal(err error) error { return &TerminalError{Cause: err} }

// Malformed marks structurally invalid messages.
var ErrMalformed = errors.New("malformed message")

// Envelope is the wire format: version first, payload lazy.
type Envelope struct {
	Version   int             `json:"v"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"ts"`
	Payload   json.RawMessage `json:"payload"`
}

// OrderPayload is the v1 order schema.
type OrderPayload struct {
	OrderID     string `json:"order_id"`
	AccountID   string `json:"account_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

// ClaimStore is the idempotency surface. Production implementations
// back this with a database (INSERT ... ON CONFLICT DO NOTHING); the
// in-memory version below mirrors the contract for tests.
type ClaimStore interface {
	// Claim marks id as being processed. It returns true when this call
	// won the claim (first delivery), false when someone else holds it.
	Claim(ctx context.Context, id string) (bool, error)
	// Release frees a claim after a failed apply, so retries can run.
	// Successful applies never release.
	Release(ctx context.Context, id string) error
}

// OrderApplier is the side effect: persist, charge, notify: whatever
// "applying an order" means. Failures are retryable by contract.
type OrderApplier interface {
	Apply(ctx context.Context, order OrderPayload) error
}

// Handler is the transport-free processing contract consumers wire in.
type Handler interface {
	Process(ctx context.Context, msg Message) error
}

// OrderService wires idempotency around an applier.
type OrderService struct {
	claims  ClaimStore
	applier OrderApplier
}

func NewOrderService(claims ClaimStore, applier OrderApplier) *OrderService {
	return &OrderService{claims: claims, applier: applier}
}

// Process is the handler: idempotent by construction, and the exact
// place unit tests verify duplicate collapse and claim release.
func (s *OrderService) Process(ctx context.Context, msg Message) error {
	id := msg.Headers["idempotency_key"]
	if id == "" {
		return Terminal(fmt.Errorf("%w: missing idempotency_key", ErrMalformed))
	}

	var env Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		return Terminal(fmt.Errorf("%w: bad envelope: %v", ErrMalformed, err))
	}

	switch env.Version {
	case 1:
		return s.processV1(ctx, id, env)
	default:
		// Unknown future versions: DLQ with a clear reason instead of
		// guessing. Schema evolution rule: never silently accept.
		return Terminal(fmt.Errorf("%w: unsupported envelope version %d", ErrMalformed, env.Version))
	}
}

func (s *OrderService) processV1(ctx context.Context, id string, env Envelope) error {
	var order OrderPayload
	if err := json.Unmarshal(env.Payload, &order); err != nil {
		return Terminal(fmt.Errorf("%w: bad v1 payload: %v", ErrMalformed, err))
	}

	// Claim BEFORE applying: concurrent duplicates collapse here.
	inserted, err := s.claims.Claim(ctx, id)
	if err != nil {
		return fmt.Errorf("claim %s: %w", id, err) // store trouble: retryable
	}
	if !inserted {
		return nil // duplicate delivery: already handled; success is the correct answer
	}

	if err := s.applier.Apply(ctx, order); err != nil {
		if relErr := s.claims.Release(ctx, id); relErr != nil {
			// Losing the release means the retry sees a false duplicate.
			// Surface it loudly; this is an integrity-relevant state.
			return fmt.Errorf("apply failed (%v) AND release failed (%v)", err, relErr)
		}
		return fmt.Errorf("apply %s: %w", order.OrderID, err)
	}
	return nil // success: the claim stands; future duplicates no-op above
}

// MemoryClaims is the in-memory ClaimStore: contract-faithful, safe for
// concurrent tests, and obviously not durable: production uses a DB.
type MemoryClaims struct {
	mu     sync.Mutex
	active map[string]bool
}

func NewMemoryClaims() *MemoryClaims {
	return &MemoryClaims{active: make(map[string]bool)}
}

func (m *MemoryClaims) Claim(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active[id] {
		return false, nil
	}
	m.active[id] = true
	return true, nil
}

func (m *MemoryClaims) Release(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, id)
	return nil
}
