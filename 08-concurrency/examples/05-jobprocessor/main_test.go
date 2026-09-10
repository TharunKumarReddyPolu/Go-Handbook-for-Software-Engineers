package main

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestProcessor(t *testing.T, workers, depth int, policy RetryPolicy) (*Processor, *sync.WaitGroup, context.CancelFunc) {
	t.Helper()
	p := NewProcessor(workers, depth, policy, func(string, ...any) {})
	ctx, cancel := context.WithCancel(context.Background())
	wg := p.Start(ctx)
	t.Cleanup(func() {
		// Tests that call p.Stop(wg) themselves have already drained; a
		// second Stop would panic on double close, so only stop via
		// cancel for tests that don't.
		cancel()
	})
	return p, wg, cancel
}

func TestSucceedsOnFirstAttempt(t *testing.T) {
	p, _, cancel := newTestProcessor(t, 2, 4, RetryPolicy{MaxAttempts: 3, Backoff: time.Millisecond})
	defer cancel()

	calls := make(chan struct{}, 1)
	if err := p.Submit(Job{ID: "j1", Run: func(ctx context.Context, j Job) error {
		calls <- struct{}{}
		return nil
	}}); err != nil {
		t.Fatalf("submit: %v", err)
	}

	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("job never ran")
	}
}

func TestRetriesUntilSuccess(t *testing.T) {
	p, wg, cancel := newTestProcessor(t, 1, 4, RetryPolicy{MaxAttempts: 3, Backoff: time.Millisecond})
	defer cancel()

	var attempts atomic.Int32 // test-local; p serializes work to one worker
	done := make(chan struct{})
	_ = p.Submit(Job{ID: "j2", Run: func(ctx context.Context, j Job) error {
		if attempts.Add(1) < 3 {
			return errors.New("transient")
		}
		close(done)
		return nil
	}})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job never succeeded after retries")
	}
	p.Stop(wg)
	if p.Metrics.Retried.Load() != 2 {
		t.Errorf("retried = %d, want 2", p.Metrics.Retried.Load())
	}
}

func TestPermanentErrorDoesNotRetry(t *testing.T) {
	p, wg, cancel := newTestProcessor(t, 1, 4, RetryPolicy{MaxAttempts: 5, Backoff: time.Millisecond})
	defer cancel()

	var calls int
	done := make(chan struct{})
	_ = p.Submit(Job{ID: "j3", Run: func(ctx context.Context, j Job) error {
		calls++
		close(done)
		return Permanent(errors.New("bad request"))
	}})

	<-done
	p.Stop(wg)
	if calls != 1 {
		t.Errorf("handler called %d times, want 1 (no retries)", calls)
	}
	if p.Metrics.Failed.Load() != 1 {
		t.Errorf("failed = %d, want 1", p.Metrics.Failed.Load())
	}
}

func TestSubmitShedsWhenFull(t *testing.T) {
	p := NewProcessor(1, 1, RetryPolicy{MaxAttempts: 1}, func(string, ...any) {})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Gate the worker deterministically: park it inside Process while we
	// fill worker(1) + queue(1), so the shedding assertions cannot race.
	entered := make(chan struct{})
	release := make(chan struct{})
	wg := p.Start(ctx)

	_ = p.Submit(Job{ID: "block", Run: func(ctx context.Context, j Job) error {
		close(entered)
		<-release
		return nil
	}})
	<-entered

	_ = p.Submit(Job{ID: "fill", Run: func(ctx context.Context, j Job) error { return nil }})

	if err := p.Submit(Job{ID: "shed", Run: func(ctx context.Context, j Job) error { return nil }}); !errors.Is(err, ErrOverloaded) {
		t.Errorf("err = %v, want ErrOverloaded", err)
	}
	if err := p.Submit(Job{ID: "shed2", Run: func(ctx context.Context, j Job) error { return nil }}); !errors.Is(err, ErrOverloaded) {
		t.Errorf("second shed err = %v, want ErrOverloaded", err)
	}

	close(release)
	p.Stop(wg)
	if p.Metrics.Rejected.Load() != 2 {
		t.Errorf("rejected = %d, want 2", p.Metrics.Rejected.Load())
	}
}

func TestPanicBecomesError(t *testing.T) {
	p, wg, cancel := newTestProcessor(t, 1, 1, RetryPolicy{MaxAttempts: 1})
	defer cancel()

	done := make(chan struct{})
	_ = p.Submit(Job{ID: "boom", Run: func(ctx context.Context, j Job) error {
		defer close(done)
		panic("worker should survive")
	}})

	<-done
	p.Stop(wg) // would hang if the worker died
	if p.Metrics.Failed.Load() != 1 {
		t.Errorf("failed = %d, want 1 (panic recorded as failure)", p.Metrics.Failed.Load())
	}
}

func TestNoGoroutineLeakAfterStop(t *testing.T) {
	before := runtime.NumGoroutine()
	p, wg, _ := newTestProcessor(t, 3, 8, RetryPolicy{MaxAttempts: 1})
	_ = p.Submit(Job{ID: "x", Run: func(ctx context.Context, j Job) error { return nil }})
	p.Stop(wg) // no cancel: Stop alone must end all workers
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines before=%d after=%d: leak", before, after)
	}
}
