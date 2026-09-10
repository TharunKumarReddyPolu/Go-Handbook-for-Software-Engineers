// Command jobprocessor is stage 5 of the concurrency progression: a
// production-shaped job processor. It combines every earlier stage,
// bounded intake (backpressure), worker pool, context cancellation: and
// adds the production concerns: bounded retries with backoff, graceful
// shutdown that drains in-flight work, and snapshot-able metrics. The
// same skeleton fronts Kafka consumers and webhook processors; see
// 18-kafka-with-go for the transport that feeds it.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

// Handler processes one job. Returning a retryable error triggers the
// retry policy; returning anything else fails the job immediately.
type Handler func(ctx context.Context, job Job) error

// RetryPolicy bounds retry attempts and spacing.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

func (p RetryPolicy) WithDefaults() RetryPolicy {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.Backoff <= 0 {
		p.Backoff = 100 * time.Millisecond
	}
	return p
}

// PermanentError marks a failure that retries cannot fix. Wrap the
// cause with it at the point you know; the processor unwraps.
type PermanentError struct{ Cause error }

func (e *PermanentError) Error() string { return "permanent: " + e.Cause.Error() }
func (e *PermanentError) Unwrap() error { return e.Cause }

// Permanent wraps err as non-retryable.
func Permanent(err error) error { return &PermanentError{Cause: err} }

type Job struct {
	ID  string
	Run Handler
}

type Metrics struct {
	Accepted  atomic.Int64
	Rejected  atomic.Int64
	Succeeded atomic.Int64
	Failed    atomic.Int64
	Retried   atomic.Int64
	InFlight  atomic.Int64
}

func (m *Metrics) Snapshot() Metrics {
	return Metrics{
		Accepted:  atomic.Int64{},
		Rejected:  atomic.Int64{},
		Succeeded: atomic.Int64{},
		Failed:    atomic.Int64{},
		Retried:   atomic.Int64{},
		InFlight:  atomic.Int64{},
	}
}

// Processor is a bounded, cancellable, retrying job processor.
type Processor struct {
	queue   chan Job
	workers int
	policy  RetryPolicy
	log     func(format string, args ...any)
	Metrics Metrics
}

func NewProcessor(workers, queueDepth int, policy RetryPolicy, log func(string, ...any)) *Processor {
	return &Processor{
		queue:   make(chan Job, queueDepth),
		workers: workers,
		policy:  policy.WithDefaults(),
		log:     log,
	}
}

// Start runs workers until ctx is cancelled or Stop is called.
func (p *Processor) Start(ctx context.Context) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer p.Metrics.InFlight.Add(-1)
			p.Metrics.InFlight.Add(1)
			for {
				select {
				case job, ok := <-p.queue:
					if !ok {
						return // intake closed and drained: Stop's contract met
					}
					p.runWithRetries(ctx, job)
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	return &wg
}

// Submit enqueues without blocking; a full queue sheds load.
func (p *Processor) Submit(job Job) error {
	select {
	case p.queue <- job:
		p.Metrics.Accepted.Add(1)
		return nil
	default:
		p.Metrics.Rejected.Add(1)
		return ErrOverloaded
	}
}

// runWithRetries applies the retry policy around one job.
func (p *Processor) runWithRetries(ctx context.Context, job Job) {
	policy := p.policy
	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		err := safeRun(ctx, job)
		if err == nil {
			p.Metrics.Succeeded.Add(1)
			return
		}
		var perm *PermanentError
		if errors.As(err, &perm) || errors.Is(err, context.Canceled) {
			// Retrying a permanent error or an intentional cancel is waste.
			break
		}
		lastErr = err
		if attempt < policy.MaxAttempts {
			p.Metrics.Retried.Add(1)
			p.log("job %s attempt %d failed: %v; retrying in %s", job.ID, attempt, err, policy.Backoff)
			select {
			case <-time.After(policy.Backoff):
			case <-ctx.Done():
				p.fail(job, ctx.Err())
				return
			}
		}
	}
	p.fail(job, lastErr)
}

func (p *Processor) fail(job Job, err error) {
	p.Metrics.Failed.Add(1)
	p.log("job %s permanently failed: %v", job.ID, err)
}

// safeRun runs a handler, converting panics into errors so a buggy job
// cannot kill a worker (or the process).
func safeRun(ctx context.Context, job Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job %s panicked: %v", job.ID, r)
		}
	}()
	return job.Run(ctx, job)
}

// Stop closes intake and drains in-flight work before returning.
func (p *Processor) Stop(wg *sync.WaitGroup) {
	close(p.queue) // workers finish remaining queued jobs, then see closed channel
	wg.Wait()
}

var ErrOverloaded = errors.New("overloaded: queue at capacity")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	p := NewProcessor(4, 16, RetryPolicy{MaxAttempts: 3, Backoff: 20 * time.Millisecond}, log)
	wg := p.Start(ctx)

	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("job-%d", i)
		_ = p.Submit(Job{ID: id, Run: func(ctx context.Context, j Job) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}})
	}

	<-ctx.Done() // SIGINT
	log("shutting down: draining in-flight jobs")
	p.Stop(wg)
	log("stopped: succeeded=%d failed=%d retried=%d",
		p.Metrics.Succeeded.Load(), p.Metrics.Failed.Load(), p.Metrics.Retried.Load())
}
