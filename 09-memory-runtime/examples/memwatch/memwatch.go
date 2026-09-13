// Package memwatch is the handbook's worked example for 09-memory-runtime:
// the leak taxonomy from chapter 5 as runnable code. Each function
// exhibits one leak shape; each Fix demonstrates the ownership rule that
// prevents it; the tests assert observable memory behavior with
// runtime metrics rather than folklore.
package memwatch

import (
	"sync"
)

// SubslicePin demonstrates leak shape 2: a tiny window pins a huge
// backing array. Big returns a 16-element view of a 1M-element array.
// The whole megabyte stays live as long as anyone holds the view.
func Big() []int {
	base := make([]int, 1_000_000)
	for i := range base {
		base[i] = i
	}
	return base[:16] // the window that pins the world
}

// BigFix returns a copy: the backing array is exactly as large as the
// data anyone can observe. The rule: copy at the ownership boundary.
func BigFix() []int {
	base := make([]int, 1_000_000)
	for i := range base {
		base[i] = i
	}
	out := make([]int, 16)
	copy(out, base[:16])
	return out
}

// LenBytes reports the observable size of a returned view, used by the
// test to distinguish view-from-copy.
func LenBytes(b []byte) int { return len(b) }

// Tracker demonstrates leak shape 1: goroutines that never stop. Start
// launches a goroutine per subscription with no stop mechanism: every
// call is a permanent allocation of stack + closure.
type Tracker struct {
	mu   sync.Mutex
	stop map[int]chan struct{}
	next int
}

func NewTracker() *Tracker {
	return &Tracker{stop: make(map[int]chan struct{})}
}

// StartNoOwner leaks: nothing can ever stop the goroutine.
func (t *Tracker) StartNoOwner(name string) {
	go func() {
		for {
			_ = name // "process events" forever
		}
	}()
}

// Start returns a stop function: the caller owns the goroutine's
// lifetime. This is the ownership rule: whoever starts it, stops it,
// via an explicit handle.
func (t *Tracker) Start(name string) (stop func()) {
	quit := make(chan struct{})
	id := t.nextID()
	t.mu.Lock()
	t.stop[id] = quit
	t.mu.Unlock()
	go func() {
		for {
			select {
			case <-quit:
				t.mu.Lock()
				delete(t.stop, id)
				t.mu.Unlock()
				return
			default:
				_ = name // process events
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(quit) }) }
}

func (t *Tracker) nextID() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.next++
	return t.next
}

// LiveWorkers reports how many started goroutines have not stopped:
// the metric that turns leak shape 1 from folklore into an alert.
func (t *Tracker) LiveWorkers() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.stop)
}

// MapGrowth demonstrates leak shape 3: unbounded collections keyed by
// unbounded input. Sessions never expire; every unique key is forever.
type MapGrowth struct {
	mu       sync.Mutex
	sessions map[string][]byte
}

func NewMapGrowth() *MapGrowth {
	return &MapGrowth{sessions: make(map[string][]byte)}
}

func (m *MapGrowth) Add(key string, payload []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[key] = payload
}

// AddWithTTL is the fix shape: entries carry expiry, and a sweeper (or
// this lazy check on insert) bounds the collection. TTLs convert an
// unbounded map into a bounded one.
func (m *MapGrowth) AddWithTTL(key string, payload []byte, maxEntries int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) >= maxEntries {
		m.sessions = make(map[string][]byte) // crude bound; real code expires per-entry
	}
	m.sessions[key] = payload
}

// Size exposes the cardinality for tests and metrics.
func (m *MapGrowth) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}
