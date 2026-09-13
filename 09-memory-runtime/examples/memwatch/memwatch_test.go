package memwatch

import (
	"runtime"
	"testing"
	"time"
)

// Chapter 5, leak shape 1: goroutines with owners stop; the count
// proves it. This is the alerting primitive chapter 5 recommends
// exporting.
func TestTracker_OwnedWorkersStop(t *testing.T) {
	tr := NewTracker()
	stops := make([]func(), 0, 8)
	for i := 0; i < 8; i++ {
		stops = append(stops, tr.Start("worker"))
	}
	if got := tr.LiveWorkers(); got != 8 {
		t.Fatalf("LiveWorkers = %d, want 8", got)
	}
	for _, stop := range stops {
		stop()
	}
	// The stop functions are synchronous in flag terms, but the
	// goroutine's bookkeeping happens on its own goroutine; poll
	// briefly with a deadline.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tr.LiveWorkers() == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("workers never stopped: LiveWorkers = %d", tr.LiveWorkers())
}

// Chapter 5, leak shape 1, the negative control: unowned goroutines
// accumulate. Asserted via the runtime's own counter.
func TestStartNoOwner_Leaks(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 5; i++ {
		tr := NewTracker()
		tr.StartNoOwner("leak")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() >= before+5 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected goroutine count to grow by at least 5; before=%d after=%d",
		before, runtime.NumGoroutine())
}

// Chapter 5, leak shape 3: bounded map stays bounded.
func TestMapGrowth_Bounded(t *testing.T) {
	m := NewMapGrowth()
	for i := 0; i < 1000; i++ {
		m.AddWithTTL(string(rune('a'+i%26))+time.Now().Format("150405"), make([]byte, 16), 64)
	}
	if got := m.Size(); got > 64 {
		t.Fatalf("Size = %d, want <= 64", got)
	}
}

// Chapter 5, leak shape 2: the pin. We cannot assert the GC kept a
// megabyte alive without hooking the allocator, but we can assert the
// structural fact the chapter teaches: the window reports 16 while
// keeping capacity far beyond it, and the fix reports capacity 16.
func TestBig_PinVersusCopy(t *testing.T) {
	view := Big()
	if len(view) != 16 {
		t.Fatalf("len(view) = %d, want 16", len(view))
	}
	if cap(view) <= 16 {
		t.Fatalf("cap(view) = %d; the pin (huge backing array) is the lesson", cap(view))
	}
	fixed := BigFix()
	if len(fixed) != 16 || cap(fixed) != 16 {
		t.Fatalf("BigFix len/cap = %d/%d, want 16/16", len(fixed), cap(fixed))
	}
}
