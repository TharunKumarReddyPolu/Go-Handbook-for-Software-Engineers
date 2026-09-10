package main

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

func noop(name string) Job {
	return Job{Name: name, Process: func(ctx context.Context) {}}
}

// TestSubmit_RejectsWhenFull pins the shedding contract deterministically:
// the worker is parked inside its Process while we fill worker + queue,
// so queue state cannot race ahead of the test.
func TestSubmit_RejectsWhenFull(t *testing.T) {
	p := New(1, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entered := make(chan struct{}) // worker signals it is inside Process
	release := make(chan struct{}) // test lets the worker finish
	blocked := Job{Name: "blocker", Process: func(ctx context.Context) {
		close(entered)
		<-release
	}}
	wg := p.Start(ctx)

	if err := p.Submit(blocked); err != nil {
		t.Fatalf("first submit should be accepted, got %v", err)
	}
	<-entered // worker is now parked in Process; queue is untouched

	for i := 0; i < 2; i++ {
		if err := p.Submit(noop("fill")); err != nil {
			t.Fatalf("fill submit %d should be accepted, got %v", i, err)
		}
	}
	if err := p.Submit(noop("overflow")); !errors.Is(err, ErrOverloaded) {
		t.Errorf("overflow submit err = %v, want ErrOverloaded", err)
	}
	if err := p.Submit(noop("overflow2")); !errors.Is(err, ErrOverloaded) {
		t.Errorf("second overflow submit err = %v, want ErrOverloaded", err)
	}

	close(release)
	p.Stop(wg)

	if _, rejected, _ := p.Snapshot(); rejected != 2 {
		t.Errorf("rejected = %d, want 2", rejected)
	}
}

func TestStop_DrainsQueue(t *testing.T) {
	p := New(2, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := p.Start(ctx)

	processed := make(chan struct{}, 8)
	for i := 0; i < 6; i++ {
		if err := p.Submit(Job{Name: "drain", Process: func(ctx context.Context) {
			processed <- struct{}{}
		}}); err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}
	p.Stop(wg) // must not return until queued jobs are processed

	if a, r, pr := p.Snapshot(); a != 6 || r != 0 || pr != 6 {
		t.Errorf("accepted=%d rejected=%d processed=%d, want 6/0/6", a, r, pr)
	}
}

func TestNoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	p := New(2, 2)
	ctx, cancel := context.WithCancel(context.Background())
	wg := p.Start(ctx)
	_ = p.Submit(noop("x"))
	p.Stop(wg)
	cancel()
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines before=%d after=%d: leak", before, after)
	}
}
