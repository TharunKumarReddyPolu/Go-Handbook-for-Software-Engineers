package ring

import "testing"

func TestRing_FIFO_Basic(t *testing.T) {
	r := NewRing[int](4)
	for i := 0; i < 4; i++ {
		if _, evicted := r.Push(i); evicted {
			t.Fatalf("unexpected eviction on push %d", i)
		}
	}
	for want := 0; want < 4; want++ {
		got, ok := r.Pop()
		if !ok || got != want {
			t.Fatalf("Pop() = %d, %v; want %d, true", got, ok, want)
		}
	}
	if _, ok := r.Pop(); ok {
		t.Fatal("Pop on drained ring should miss")
	}
}

func TestRing_DropOldest_AcrossSeam(t *testing.T) {
	r := NewRing[int](3)
	// Fill, overflow by one, drain: exercises the wrap seam.
	for i := 0; i < 4; i++ {
		r.Push(i)
	}
	if r.Len() != 3 {
		t.Fatalf("Len = %d, want 3", r.Len())
	}
	for want := 1; want <= 3; want++ {
		got, ok := r.Pop()
		if !ok || got != want {
			t.Fatalf("after overflow, Pop() = %d, %v; want %d (oldest evicted)", got, ok, want)
		}
	}
}

func TestRing_DeepWrap(t *testing.T) {
	// Push/pop far past capacity, in and out of phase; FIFO order must
	// hold for everything that survives.
	r := NewRing[int](5)
	var want []int
	next := 0
	for phase := 0; phase < 100; phase++ {
		for i := 0; i < 3; i++ {
			if ev, _ := r.Push(next); ev != 0 {
				t.Fatalf("unexpected eviction while under capacity: %+v", ev)
			}
			want = append(want, next)
			if len(want) > 5 {
				want = want[len(want)-5:]
			}
			next++
		}
		for range want {
			got, ok := r.Pop()
			if !ok || got != want[0] {
				t.Fatalf("phase %d: Pop() = %d, %v; want %d", phase, got, ok, want[0])
			}
			want = want[1:]
		}
	}
}

func TestRing_DegenerateCapacities(t *testing.T) {
	for _, cap := range []int{-1, 0, 1, 2} {
		r := NewRing[int](cap)
		if r.Cap() < 1 {
			t.Fatalf("capacity %d: Cap() = %d, want >= 1", cap, r.Cap())
		}
		r.Push(1)
		r.Push(2)
		if got, ok := r.Peek(); !ok || got == 0 && r.Len() != 1 {
			t.Fatalf("capacity %d: Peek = %d, %v, len %d", cap, got, ok, r.Len())
		}
	}
}

// TestRecentKeys_Agreement is the bookkeeping test: the map must
// mirror the ring's live contents after every op. Hits do NOT
// refresh the window (the ring is the recency record), so a repeated
// key still ages out from its original push position.
//
// Assertions use Contains, the pure query: Seen would mutate the
// window it is checking (the observation-changes-the-state trap this
// chapter warns about).
func TestRecentKeys_Agreement(t *testing.T) {
	r := NewRecentKeys(3)

	if r.Seen("a") || r.Seen("b") || r.Seen("c") {
		t.Fatal("fresh keys must not be reported seen")
	}
	if !r.Contains("a") {
		t.Fatal("a should still be in the window")
	}

	r.Seen("d") // evicts a: its slot is the oldest
	if r.Contains("a") {
		t.Fatal("a was evicted; hits do not refresh the ring position")
	}

	r.Seen("e") // evicts b
	r.Seen("f") // evicts c: window is now {d, e, f}
	if r.Contains("b") || r.Contains("c") {
		t.Fatal("b and c should have been evicted")
	}
	if !r.Contains("d") || !r.Contains("e") || !r.Contains("f") {
		t.Fatal("d, e, f should be in the window")
	}

	// Recording a key that is already present is a hit: no eviction,
	// no position change, and Seen reports true.
	if !r.Seen("d") {
		t.Fatal("d is in the window: Seen must report true")
	}
	if !r.Contains("d") || !r.Contains("e") || !r.Contains("f") {
		t.Fatal("a hit must not change the window")
	}

	// Pure queries never mutate: Contains repeats are free.
	for i := 0; i < 3; i++ {
		if !r.Contains("d") || !r.Contains("e") || !r.Contains("f") {
			t.Fatal("window changed during pure queries")
		}
	}
}
