// Command backpressure is stage 4 of the concurrency progression: the
// pool with a bounded intake queue and explicit load shedding. When the
// queue is full, work is rejected and counted instead of queued without
// limit: the difference between degrading and dying. See
// 08-concurrency/05-patterns.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ErrOverloaded is returned to callers when the system sheds load.
// Callers should treat it as retryable-with-delay, not as failure.
var ErrOverloaded = errors.New("overloaded: queue at capacity")

type Processor struct {
	queue   chan Job
	workers int
	// Metrics: exported via Snapshot for dashboards.
	accepted  atomic.Int64
	rejected  atomic.Int64
	processed atomic.Int64
}

func New(workers, queueDepth int) *Processor {
	p := &Processor{
		queue:   make(chan Job, queueDepth), // the bound: memory is now finite
		workers: workers,
	}
	return p
}

// Start launches the workers. Call Stop to drain.
func (p *Processor) Start(ctx context.Context) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case j, ok := <-p.queue:
					if !ok {
						return // intake closed and drained: Stop's contract met
					}
					j.Process(ctx)
					p.processed.Add(1)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return &wg
}

// Submit tries to enqueue a job without blocking. Full queue means shed,
// not wait: the caller learns immediately and can retry, defer, or
// degrade. This is the backpressure decision surface.
func (p *Processor) Submit(j Job) error {
	select {
	case p.queue <- j:
		p.accepted.Add(1)
		return nil
	default:
		p.rejected.Add(1)
		return ErrOverloaded
	}
}

// Stop closes the intake and waits for workers to drain the queue.
func (p *Processor) Stop(wg *sync.WaitGroup) {
	close(p.queue)
	wg.Wait()
}

// Snapshot returns the counters for metrics exposure.
func (p *Processor) Snapshot() (accepted, rejected, processed int64) {
	return p.accepted.Load(), p.rejected.Load(), p.processed.Load()
}

type Job struct {
	Name    string
	Process func(ctx context.Context)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := New(2, 4) // 2 workers, 4-slot queue: hard memory bound
	wg := p.Start(ctx)

	for i := 0; i < 20; i++ {
		j := Job{Name: fmt.Sprintf("job-%d", i), Process: func(ctx context.Context) {
			// work happens here
		}}
		if err := p.Submit(j); err != nil {
			fmt.Printf("%s: %v\n", j.Name, err)
		}
	}
	p.Stop(wg)

	a, r, pr := p.Snapshot()
	fmt.Printf("accepted=%d rejected=%d processed=%d\n", a, r, pr)
}
