// Counter structures compared: protect-one-word shows up as atomic,
// protect-an-invariant as mutex, transfer-ownership as channel. The
// benchmark proves the fit — see 19-performance/03-concurrency-performance.md.
package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

// mutexCounter guards a generic invariant slot.
type mutexCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *mutexCounter) bump() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

// channelCounter routes bumps through an owner goroutine: the ownership
// model. Setup cost is amortized by the benchmark harness.
type channelCounter struct {
	ops   chan struct{}
	done  chan struct{}
	value *atomic.Int64 // owner-side storage; channel is the transport
}

func newChannelCounter() *channelCounter {
	c := &channelCounter{
		ops:   make(chan struct{}),
		done:  make(chan struct{}),
		value: &atomic.Int64{},
	}
	go func() { // the owner
		defer close(c.done)
		for range c.ops {
			c.value.Add(1)
		}
	}()
	return c
}

func (c *channelCounter) bump() { c.ops <- struct{}{} }
func (c *channelCounter) close() {
	close(c.ops)
	<-c.done
}

func BenchmarkCounter(b *testing.B) {
	b.Run("Atomic", func(b *testing.B) {
		var n atomic.Int64
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				n.Add(1)
			}
		})
	})

	b.Run("Mutex", func(b *testing.B) {
		var c mutexCounter
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.bump()
			}
		})
	})

	b.Run("Channel", func(b *testing.B) {
		c := newChannelCounter()
		defer c.close()
		// Single producer per measurement: the ownership model serializes
		// by design; RunParallel would measure the handoff queue, which is
		// exactly the point of the chapter's argument.
		for b.Loop() {
			c.bump()
		}
	})
}
