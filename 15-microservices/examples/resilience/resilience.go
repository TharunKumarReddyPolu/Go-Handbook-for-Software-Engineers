// Package resilience is the handbook's worked example for
// 15-microservices chapter 3: the resilience quartet (timeout, retry,
// circuit breaker, bulkhead) as small composable pieces, each with a
// deterministic test. The injected clock makes backoff and cooldowns
// testable without sleeps.
package resilience

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2" // v2: Int64N and a per-process source, goroutine-safe
	"sync"
	"time"
)

// Sentinels per layer: the error taxonomy from chapter 3. Callers
// switch on these to know which bound fired.
var (
	ErrCircuitOpen  = errors.New("circuit open")
	ErrBulkheadFull = errors.New("bulkhead full")
)

// Retryable mirrors 05 Section 2's classification seam: the caller decides
// what is transient. Breakers never see 4xx-shaped results.
type Retryable func(error) bool

// Retry calls op until success, cap, or budget exhaustion. Backoff is
// exponential (base << attempt) with jitter. Budget-aware: no attempt
// starts if the remaining deadline cannot fit its own backoff.
func Retry(ctx context.Context, op func(context.Context) error, isRetryable Retryable, base time.Duration, maxAttempts int) error {
	deadline, hasDeadline := ctx.Deadline()
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := op(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if isRetryable != nil && !isRetryable(err) {
			return lastErr // deterministic failure: retrying lies
		}
		backoff := base << attempt
		if hasDeadline && time.Until(deadline) <= backoff {
			return fmt.Errorf("retry budget spent before backoff %s: %w", backoff, lastErr)
		}
		jitter := time.Duration(rand.Int64N(int64(backoff) / 2))
		select {
		case <-time.After(backoff + jitter):
		case <-ctx.Done():
			return ctx.Err() // budget died mid-backoff
		}
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

// Breaker is the chapter 3 state machine: closed (normal), open
// (fast-fail until cooldown), half-open (one probe admits).
type Breaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	cooldown  time.Duration
	openUntil time.Time
	halfOpen  bool // exactly one probe admitted
	now       func() time.Time
}

func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now}
}

// Now overrides the clock in tests.
func (b *Breaker) Now(now func() time.Time) { b.now = now }

// Allow reports whether a call may proceed. In half-open state,
// exactly one probe is admitted; everyone else fast-fails.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if now.Before(b.openUntil) {
		return false // open
	}
	if b.halfOpen {
		return false // a probe is already out
	}
	if !b.openUntil.IsZero() && now.After(b.openUntil) {
		b.halfOpen = true // first arrival after cooldown becomes the probe
	}
	return true
}

// Record reports an outcome. Systemic failures (5xx, timeouts) only;
// counting 4xx trips breakers on client bugs.
func (b *Breaker) Record(ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.halfOpen {
		b.halfOpen = false
		b.openUntil = time.Time{} // zero: closed
		if ok {
			b.failures = 0
		} else {
			b.openUntil = b.now().Add(b.cooldown) // probe failed: reopen
		}
		return
	}
	if ok {
		b.failures = 0
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = b.now().Add(b.cooldown)
	}
}

// Bulkhead bounds in-flight calls to one dependency (08 Section 5's
// semaphore). Full may either block on ctx or reject immediately;
// this version rejects: fast-fail composes with the breaker.
type Bulkhead struct {
	sem chan struct{}
}

func NewBulkhead(maxInFlight int) *Bulkhead {
	return &Bulkhead{sem: make(chan struct{}, maxInFlight)}
}

// Do runs fn under the bulkhead; ErrBulkheadFull when all slots are
// taken, so callers and metrics can distinguish saturation from
// failure.
func (b *Bulkhead) Do(ctx context.Context, fn func(context.Context) error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn(ctx)
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBulkheadFull
	}
}
