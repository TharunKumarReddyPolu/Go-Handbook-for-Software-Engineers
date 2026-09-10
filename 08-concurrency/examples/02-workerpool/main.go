// Command workerpool is stage 2 of the concurrency progression: a
// bounded worker pool with a results channel. The pool owns the results
// channel and closes it after all workers finish. See
// 08-concurrency/05-patterns.md and 01-goroutines-and-channels.md.
package main

import (
	"fmt"
	"sync"
)

// Job is a unit of work; Result carries its outcome.
type Job struct {
	ID   int
	Name string
}

type Result struct {
	JobID  int
	Output string
	Err    error
}

// Run processes jobs with n workers and returns results in a closed
// channel. Results may arrive in any order (pools do not preserve
// ordering).
func Run(jobs []Job, n int) <-chan Result {
	in := make(chan Job)
	out := make(chan Result, len(jobs)) // buffered: workers never block

	go func() {
		defer close(in) // owner of in: sends then closes
		for _, j := range jobs {
			in <- j
		}
	}()

	var workers sync.WaitGroup
	for i := 0; i < n; i++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			for j := range in { // exits when in closes and drains
				out <- Result{JobID: j.ID, Output: fmt.Sprintf("worker %d did %s", id, j.Name)}
			}
		}(i)
	}

	go func() {
		workers.Wait() // all sends on out are done...
		close(out)     // ...so close is safe
	}()

	return out
}

func main() {
	jobs := []Job{{1, "alpha"}, {2, "beta"}, {3, "gamma"}}
	for r := range Run(jobs, 2) {
		fmt.Printf("%+v\n", r)
	}
}
