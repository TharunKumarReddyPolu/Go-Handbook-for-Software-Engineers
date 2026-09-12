package heap

import (
	"math/rand"
	"sort"
	"testing"
)

func TestHeap_BasicOrder(t *testing.T) {
	h := New[int](func(a, b int) bool { return a < b })
	for _, v := range []int{5, 3, 8, 1, 9, 2} {
		h.Push(v)
	}
	want := []int{1, 2, 3, 5, 8, 9}
	for _, w := range want {
		got, ok := h.Pop()
		if !ok || got != w {
			t.Fatalf("Pop() = %d, %v; want %d", got, ok, w)
		}
	}
	if _, ok := h.Pop(); ok {
		t.Fatal("Pop on empty heap should miss")
	}
}

func TestHeap_Peek(t *testing.T) {
	h := New[string](func(a, b string) bool { return a < b })
	h.Push("m")
	h.Push("a")
	h.Push("z")
	got, ok := h.Peek()
	if !ok || got != "a" {
		t.Fatalf("Peek() = %q, %v; want a", got, ok)
	}
	if h.Len() != 3 {
		t.Fatalf("Peek must not remove: Len = %d", h.Len())
	}
}

// TestHeap_InvariantProperty pushes and pops a random sequence,
// asserting the heap invariant holds after every operation.
func TestHeap_InvariantProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	h := New[int](func(a, b int) bool { return a < b })
	live := 0

	for i := 0; i < 2000; i++ {
		if rng.Intn(3) == 0 && live > 0 { // pop
			if _, ok := h.Pop(); !ok {
				t.Fatal("Pop missed on non-empty heap")
			}
			live--
		} else { // push
			h.Push(rng.Intn(1000))
			live++
		}
		if !h.checkInvariant() {
			t.Fatalf("invariant violated after op %d (len %d)", i, h.Len())
		}
		if h.Len() != live {
			t.Fatalf("Len = %d, want %d after op %d", h.Len(), live, i)
		}
	}
}

// TestHeap_Differential replays ops against a sort-based reference.
func TestHeap_Differential(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	var ref []int
	h := New[int](func(a, b int) bool { return a < b })

	for i := 0; i < 1000; i++ {
		v := rng.Intn(500)
		h.Push(v)
		ref = append(ref, v)

		if i%5 == 0 && len(ref) > 0 { // pop both, compare
			got, _ := h.Pop()
			sort.Ints(ref)
			want := ref[0]
			ref = ref[1:]
			if got != want {
				t.Fatalf("op %d: Pop() = %d, want %d", i, got, want)
			}
		}
	}
}

func TestTopK(t *testing.T) {
	stream := []int{4, 1, 7, 3, 9, 2, 8, 5, 6}
	got := TopK(stream, 3, func(a, b int) bool { return a > b })
	want := []int{9, 8, 7}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("TopK[%d] = %d, want %d (full: %v)", i, got[i], w, got)
		}
	}
}

// TopK returns the k largest elements of in, largest first, using a
// bounded heap: O(n log k) time, O(k) memory.
func TopK(in []int, k int, less func(a, b int) bool) []int {
	// min-heap of the current top-k, ordered so the WEAKEST keeper
	// sits at the top: less(b, a) inverts the caller's ordering.
	// (Negating with !less(a, b) is wrong: it turns == into true in
	// both directions, which is not a strict order and corrupts the
	// sift. Swap the arguments, never the polarity.)
	h := New[int](func(a, b int) bool { return less(b, a) })
	for _, v := range in {
		if h.Len() < k {
			h.Push(v)
			continue
		}
		if top, _ := h.Peek(); less(v, top) {
			h.Pop()
			h.Push(v)
		}
	}
	out := make([]int, 0, h.Len())
	for h.Len() > 0 {
		v, _ := h.Pop()
		out = append(out, v)
	}
	// Popping a bounded min-heap yields WORST-first (the root is the
	// weakest keeper); reverse so the caller gets best-first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
