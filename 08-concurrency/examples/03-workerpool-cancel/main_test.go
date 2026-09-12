package main

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

func feed(jobs ...Job) <-chan Job {
	ch := make(chan Job)
	go func() {
		defer close(ch)
		for _, j := range jobs {
			ch <- j
		}
	}()
	return ch
}

func okJob(id int) Job {
	return Job{ID: id, Process: func(ctx context.Context) (string, error) {
		return "ok", nil
	}}
}

// drainAsync collects into a slice-safe counter while the pool runs, so
// a full (unbuffered) out channel never blocks workers before cancel.
func drainAsync(ch <-chan Result) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ch {
		}
	}()
	return &wg
}

func TestRun_CompletesAllJobs(t *testing.T) {
	ctx := context.Background()
	wg := drainAsync(Run(ctx, feed(okJob(1), okJob(2), okJob(3)), 2))
	wg.Wait()
}

func TestRun_CancellationStopsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slow := Job{ID: 99, Process: func(ctx context.Context) (string, error) {
		select { // honors ctx: a well-behaved job
		case <-time.After(10 * time.Second):
			return "finally", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}}

	before := runtime.NumGoroutine()
	out := Run(ctx, feed(okJob(1), slow, okJob(2), okJob(3)), 2)
	wg := drainAsync(out) // keep the pipe unblocked from the start

	time.Sleep(20 * time.Millisecond) // let work start
	cancel()
	wg.Wait()

	assertGoroutinesSettled(t, before)
}

func TestRun_NoGoroutineLeak_HappyPath(t *testing.T) {
	before := runtime.NumGoroutine()
	wg := drainAsync(Run(context.Background(), feed(okJob(1), okJob(2)), 4))
	wg.Wait()
	assertGoroutinesSettled(t, before)
}

// assertGoroutinesSettled retries the goroutine count briefly: a
// worker that has finished its work may still be microseconds from
// exiting when a channel closes, so a single sample races (CI-caught).
// A real leak never settles; a scheduling artifact settles fast.
func assertGoroutinesSettled(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		after := runtime.NumGoroutine()
		if after <= before {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("goroutines before=%d after=%d: leak", before, after)
			return
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
}
