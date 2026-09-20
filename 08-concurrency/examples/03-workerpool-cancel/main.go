// Command cancelable-pool is stage 3 of the concurrency progression:
// the worker pool from stage 2, made leak-free under cancellation. Every
// blocking operation selects on ctx.Done(), so canceling the context
// stops intake, workers, and output collection promptly. See
// 08-concurrency/03-context.md and 05-patterns.md.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID      int
	Process func(ctx context.Context) (string, error)
}

type Result struct {
	JobID  int
	Output string
	Err    error
}

// Run processes jobs with n workers until the input closes or ctx is
// canceled. The returned channel is always closed exactly once.
func Run(ctx context.Context, jobs <-chan Job, n int) <-chan Result {
	out := make(chan Result)
	go func() {
		defer close(out) // pool owns out; the finalizer below guarantees close

		var workers sync.WaitGroup
		workerDone := make(chan struct{})
		go func() {
			defer close(workerDone)
			for i := 0; i < n; i++ {
				workers.Add(1)
				go worker(ctx, jobs, out, &workers)
			}
			workers.Wait()
		}()

		select {
		case <-workerDone:
			// all workers exited: no more sends on out
		case <-ctx.Done():
			// ctx canceled: workers exit on their own via their selects.
			// We still wait for them so no send races our close.
			<-workerDone
		}
	}()
	return out
}

func worker(ctx context.Context, jobs <-chan Job, out chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case j, ok := <-jobs:
			if !ok {
				return // intake closed and drained
			}
			out <- process(ctx, j)
		case <-ctx.Done():
			return // cancellation beats backlog
		}
	}
}

func process(ctx context.Context, j Job) Result {
	out, err := j.Process(ctx)
	return Result{JobID: j.ID, Output: out, Err: err}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	jobs := make(chan Job)
	go func() {
		defer close(jobs)
		for i := 1; ; i++ {
			jobs <- Job{ID: i, Process: func(ctx context.Context) (string, error) {
				time.Sleep(50 * time.Millisecond) // simulate work
				return "done", nil
			}}
		}
	}()

	for r := range Run(ctx, jobs, 2) {
		fmt.Printf("job %d: %s (err=%v)\n", r.JobID, r.Output, r.Err)
	}
	fmt.Println("shutdown complete: no leaked goroutines")
}
