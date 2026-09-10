// Command toy is stage 1 of the concurrency progression: a producer and
// a consumer sharing one channel with single ownership (producer closes).
// See 08-concurrency/01-goroutines-and-channels.md.
package main

import "fmt"

// Produce returns a receive-only channel of [1..n], closed when done.
// The producing goroutine is the owner: it sends and closes.
func Produce(n int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := 1; i <= n; i++ {
			ch <- i
		}
	}()
	return ch
}

// Collect ranges the channel until it closes — the receive-only contract
// means the consumer cannot close or send on it, even by accident.
func Collect(in <-chan int) []int {
	var out []int
	for v := range in {
		out = append(out, v)
	}
	return out
}

func main() {
	for _, v := range Collect(Produce(3)) {
		fmt.Println("processed", v)
	}
}
