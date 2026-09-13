package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Retry ---

func TestRetry_SuccessAfterTransientFailures(t *testing.T) {
	attempts := 0
	err := Retry(t.Context(),
		func(context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("connection reset") // transient
			}
			return nil
		},
		func(err error) bool { return true }, // everything transient here
		time.Millisecond, 5,
	)
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_DeterministicFailureNotRetried(t *testing.T) {
	attempts := 0
	err := Retry(t.Context(),
		func(context.Context) error { attempts++; return errors.New("400 bad request") },
		func(err error) bool { return false }, // classification: never retry
		time.Millisecond, 5,
	)
	if err == nil || attempts != 1 {
		t.Fatalf("attempts = %d err = %v: a 400 must not retry", attempts, err)
	}
}

func TestRetry_BudgetSpentBeforeBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Millisecond)
	defer cancel()

	attempts := 0
	err := Retry(ctx,
		func(context.Context) error { attempts++; return errors.New("timeout") },
		func(error) bool { return true },
		100*time.Millisecond, // backoff exceeds the whole budget
		5,
	)
	if err == nil || attempts != 1 {
		t.Fatalf("attempts = %d err = %v: no attempt should start when the backoff cannot fit the budget", attempts, err)
	}
}

// --- Breaker: fully deterministic via the injected clock ---

func newTestBreaker() (*Breaker, *time.Time) {
	b := NewBreaker(3, time.Minute)
	current := time.Unix(0, 0)
	b.Now(func() time.Time { return current })
	return b, &current
}

func TestBreaker_OpensAtThresholdAndClosesAfterProbe(t *testing.T) {
	b, clock := newTestBreaker()

	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatal("closed breaker must admit")
		}
		b.Record(false)
	}

	if b.Allow() {
		t.Fatal("breaker must be open at threshold")
	}

	*clock = clock.Add(2 * time.Minute) // past cooldown
	if !b.Allow() {
		t.Fatal("after cooldown, first call is the probe")
	}
	if b.Allow() {
		t.Fatal("half-open admits exactly ONE probe")
	}

	b.Record(true)
	if !b.Allow() {
		t.Fatal("successful probe closes the breaker")
	}
}

func TestBreaker_FailedProbeReopens(t *testing.T) {
	b, clock := newTestBreaker()
	for i := 0; i < 3; i++ {
		b.Allow()
		b.Record(false)
	}

	*clock = clock.Add(2 * time.Minute)
	b.Allow() // the probe
	b.Record(false)

	if b.Allow() {
		t.Fatal("failed probe must reopen the breaker")
	}
}

func TestBreaker_SuccessResetsCount(t *testing.T) {
	b, _ := newTestBreaker()
	for i := 0; i < 2; i++ { // below threshold
		b.Allow()
		b.Record(false)
	}
	b.Allow()
	b.Record(true) // reset

	for i := 0; i < 2; i++ { // two more failures: still closed
		b.Allow()
		b.Record(false)
	}
	if !b.Allow() {
		t.Fatal("success must reset the failure count")
	}
}

// --- Bulkhead ---

func TestBulkhead_RejectsBeyondCapacity(t *testing.T) {
	b := NewBulkhead(2)
	release := make(chan struct{})
	hold := func(context.Context) error { <-release; return nil }

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ { // fill every slot
		wg.Add(1)
		go func() { defer wg.Done(); _ = b.Do(t.Context(), hold) }()
	}

	// Third call while both slots are parked: rejected, not queued.
	time.Sleep(10 * time.Millisecond)
	err := b.Do(t.Context(), func(context.Context) error { return nil })
	if !errors.Is(err, ErrBulkheadFull) {
		t.Fatalf("err = %v, want ErrBulkheadFull", err)
	}

	close(release)
	wg.Wait()
	if err := b.Do(t.Context(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("slots must free: %v", err)
	}
}

// --- The storm test: the quartet's arithmetic, composed ---

// fakeDownstream counts hits and fails until told otherwise.
type fakeDownstream struct {
	hits    atomic.Int64
	failAll atomic.Bool
}

func (f *fakeDownstream) call(context.Context) error {
	f.hits.Add(1)
	if f.failAll.Load() {
		return errors.New("500 unavailable")
	}
	return nil
}

func TestStorm_BreakerCapsDownstreamHits(t *testing.T) {
	const (
		callers   = 50
		threshold = 5
	)
	downstream := &fakeDownstream{failAll: atomic.Bool{}}
	downstream.failAll.Store(true)
	breaker := NewBreaker(threshold, time.Hour) // stays open for the storm
	current := time.Unix(0, 0)
	breaker.Now(func() time.Time { return current })
	bulkhead := NewBulkhead(10)

	call := func(ctx context.Context) error {
		if !breaker.Allow() {
			return ErrCircuitOpen
		}
		err := bulkhead.Do(ctx, func(ctx context.Context) error {
			return downstream.call(ctx)
		})
		if !errors.Is(err, ErrBulkheadFull) { // saturation is not a systemic failure
			breaker.Record(err == nil)
		}
		return err
	}

	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := call(t.Context()); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	hits := downstream.hits.Load()
	// Without the breaker: 50 callers x 1 attempt each = up to 50 hits.
	// With it: the breaker opens after `threshold` failures; every
	// later call fast-fails without touching the dependency. The
	// exact hit count depends on scheduling, but it must be far below
	// the storm size: that gap IS the breaker's value.
	if hits >= callers {
		t.Errorf("downstream hits = %d of %d callers: the breaker provided no protection", hits, callers)
	}
	if hits > threshold+10 {
		t.Errorf("downstream hits = %d, expected ~%d (threshold) plus scheduling slack", hits, threshold)
	}
	t.Logf("storm: %d callers produced %d downstream hits (breaker limit %d)", callers, hits, threshold)
}

func TestStorm_RetryAmplification(t *testing.T) {
	// The same storm with retries and no breaker: the arithmetic the
	// chapter warns about. Every caller retries 3x against a failing
	// dependency: ~150 downstream hits instead of 50.
	downstream := &fakeDownstream{failAll: atomic.Bool{}}
	downstream.failAll.Store(true)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = Retry(t.Context(), downstream.call, func(error) bool { return true },
				time.Microsecond, 3)
		}()
	}
	wg.Wait()

	hits := downstream.hits.Load()
	if hits < 100 {
		t.Errorf("hits = %d: expected retry amplification (~150)", hits)
	}
	t.Logf("amplification: 50 callers x 3 retries = %d hits", hits)
	fmt.Println() // keep test output tidy across the two logs
}
