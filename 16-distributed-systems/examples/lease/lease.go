// Package lease is the handbook's worked example for
// 16-distributed-systems chapter 3: a lease with fencing tokens.
// The coordinator hands out a monotonically increasing token with
// each lease; the storage side refuses writes carrying a token older
// than the newest it has seen. A paused-but-alive ex-leader ("zombie
// leader") cannot corrupt the new leader's work, even though it
// never learned it lost.
//
// The two sides mirror production: Lease is the coordinator (etcd
// leases in real fleets); Fenced is the storage guard (the
// conditional UPDATE in ch. 3's SQL).
package lease

import (
	"errors"
	"sync"
)

// ErrFencedOut is returned when a write carries a stale token: the
// holder must stop immediately and re-acquire.
var ErrFencedOut = errors.New("fenced out by a newer lease holder")

// Coordinator allocates leases. In production this is etcd's lease +
// election machinery; here it is a mutex-protected counter with an
// injected clock so tests control expiry.
type Coordinator struct {
	mu     sync.Mutex
	next   int64
	now    func() int64 // monotonic test clock, milliseconds
	ttlMs  int64
	active map[string]int64 // holder -> expiry
	tokens map[string]int64 // holder -> fencing token
}

func NewCoordinator(now func() int64, ttlMs int64) *Coordinator {
	return &Coordinator{
		now:    now,
		ttlMs:  ttlMs,
		active: map[string]int64{},
		tokens: map[string]int64{},
	}
}

// Acquire grants a lease to holder with a fresh fencing token. An
// expired previous holder is evicted here (that is the TTL working).
func (c *Coordinator) Acquire(holder string) (Token, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for h, expiry := range c.active {
		if expiry <= now { // expired: the zombie is evicted by the coordinator
			delete(c.active, h)
			delete(c.tokens, h)
		}
	}
	if len(c.active) > 0 {
		return Token{}, false // a valid holder exists
	}
	c.next++
	c.active[holder] = now + c.ttlMs
	c.tokens[holder] = c.next
	return Token{Number: c.next, Holder: holder}, true
}

// Renew extends the holder's lease. A zombie that let its lease
// expire cannot renew: it must re-acquire like everyone else.
func (c *Coordinator) Renew(holder string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	expiry, ok := c.active[holder]
	if !ok || expiry <= c.now() {
		delete(c.active, holder)
		delete(c.tokens, holder)
		return false
	}
	c.active[holder] = c.now() + c.ttlMs
	return true
}

// Token is the fencing token: proof of a valid lease, comparable
// monotonically. Storage checks it on every durable write.
type Token struct {
	Number int64
	Holder string
}

// Fenced is the storage guard: a table of per-resource tokens that
// rejects stale writers. The SQL analog is the conditional UPDATE
// with `fencing_no < $1`.
type Fenced struct {
	mu     sync.Mutex
	newest map[string]int64 // resource -> newest accepted token
}

func NewFenced() *Fenced { return &Fenced{newest: map[string]int64{}} }

// Write attempts a guarded write: it succeeds only when the token is
// newer than anything seen for the resource. A zombie leader's write
// (old token) lands here and is rejected: ErrFencedOut.
func (f *Fenced) Write(resource string, t Token, apply func()) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.Number <= f.newest[resource] {
		return ErrFencedOut
	}
	apply()
	f.newest[resource] = t.Number
	return nil
}

// Newest exposes the accepted watermark (tests and diagnostics).
func (f *Fenced) Newest(resource string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.newest[resource]
}
