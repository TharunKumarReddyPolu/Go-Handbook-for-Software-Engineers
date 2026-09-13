package lease

import (
	"errors"
	"testing"
)

// The zombie-leader scenario, the chapter in one test: A holds a
// lease and a token; A pauses (the clock advances past the TTL
// without renewal); B acquires and writes; A wakes still believing
// and attempts a write. The fence must reject A.
func TestZombieLeaderGetsFenced(t *testing.T) {
	clockMs := int64(0)
	coordinator := NewCoordinator(func() int64 { return clockMs }, 15)
	storage := NewFenced()

	// A holds the lease with token 1.
	tokenA, ok := coordinator.Acquire("leader-a")
	if !ok || tokenA.Number != 1 {
		t.Fatalf("A acquire: token=%+v ok=%v", tokenA, ok)
	}

	// A pauses: the clock advances past the TTL with no renewal.
	clockMs += 30

	// B acquires cleanly: the coordinator evicted expired A.
	tokenB, ok := coordinator.Acquire("leader-b")
	if !ok || tokenB.Number != 2 {
		t.Fatalf("B acquire: token=%+v ok=%v (A must have been evicted)", tokenB, ok)
	}

	// B writes with its token: accepted.
	err := storage.Write("jobs", tokenB, func() {})
	if err != nil {
		t.Fatalf("B write: %v", err)
	}

	// A wakes, still believing it is leader, and attempts a write
	// with its old token 1. The fence rejects it.
	err = storage.Write("jobs", tokenA, func() {})
	if !errors.Is(err, ErrFencedOut) {
		t.Fatalf("A's zombie write err = %v, want ErrFencedOut", err)
	}

	// A's stale write must not have been applied: B's state stands.
	if storage.Newest("jobs") != tokenB.Number {
		t.Errorf("newest = %d, want %d (A's write must not apply)",
			storage.Newest("jobs"), tokenB.Number)
	}
}

func TestRenew_AfterExpiryFails(t *testing.T) {
	clockMs := int64(0)
	c := NewCoordinator(func() int64 { return clockMs }, 15)

	if _, ok := c.Acquire("a"); !ok {
		t.Fatal("acquire A")
	}
	clockMs += 30 // past TTL without renewal
	if c.Renew("a") {
		t.Fatal("an expired holder cannot renew: it must re-acquire")
	}

	// Re-acquire gets a NEW token: fencing stays monotonic.
	tok, ok := c.Acquire("a")
	if !ok || tok.Number != 2 {
		t.Fatalf("re-acquire: token=%+v ok=%v", tok, ok)
	}
}

func TestAcquire_SingleHolder(t *testing.T) {
	clockMs := int64(0)
	c := NewCoordinator(func() int64 { return clockMs }, 15)

	if _, ok := c.Acquire("a"); !ok {
		t.Fatal("first holder")
	}
	if _, ok := c.Acquire("b"); ok {
		t.Fatal("second acquire while a valid holder exists must fail")
	}

	// Renewal keeps A alive against B's attempts.
	clockMs += 10
	if !c.Renew("a") {
		t.Fatal("renewal inside TTL")
	}
	if _, ok := c.Acquire("b"); ok {
		t.Fatal("a renewed lease must still block B")
	}
}

func TestFenced_RejectsOutOfOrderTokens(t *testing.T) {
	f := NewFenced()
	tok2 := Token{Number: 2, Holder: "b"}
	tok1 := Token{Number: 1, Holder: "a"}

	if err := f.Write("r", tok2, func() {}); err != nil {
		t.Fatalf("first write (2): %v", err)
	}
	if err := f.Write("r", tok1, func() {}); !errors.Is(err, ErrFencedOut) {
		t.Fatalf("stale write (1): err = %v, want ErrFencedOut", err)
	}
	tok3 := Token{Number: 3, Holder: "c"}
	if err := f.Write("r", tok3, func() {}); err != nil {
		t.Fatalf("newer write (3): %v", err)
	}

	// Different resources fence independently.
	if err := f.Write("other", tok1, func() {}); err != nil {
		t.Fatalf("other resource starts its own watermark: %v", err)
	}
}
