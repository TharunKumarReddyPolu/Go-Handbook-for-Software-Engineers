// Package heap is the handbook's worked example for 03-data-structures
// chapter 4: a generic binary min-heap ordered by a caller-supplied
// less function, with the Fix and Remove support (index bookkeeping)
// that container/heap makes awkward.
package heap

// Heap is a binary min-heap: less(a, b) == true means a comes out
// first. Not safe for concurrent use; give it one owner or wrap it.
type Heap[T any] struct {
	data []T
	less func(a, b T) bool
	idx  map[*T]int // live only when constructed with WithIndex
}

func New[T any](less func(a, b T) bool) *Heap[T] {
	return &Heap[T]{less: less}
}

func (h *Heap[T]) Len() int { return len(h.data) }

// Push adds v, sifting up.
func (h *Heap[T]) Push(v T) {
	h.data = append(h.data, v)
	h.siftUp(len(h.data) - 1)
}

// Peek returns the top element without removing it.
func (h *Heap[T]) Peek() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		return zero, false
	}
	return h.data[0], true
}

// Pop removes and returns the top element.
func (h *Heap[T]) Pop() (T, bool) {
	var zero T
	n := len(h.data)
	if n == 0 {
		return zero, false
	}
	top := h.data[0]
	h.data[0] = h.data[n-1]
	h.data[n-1] = zero // hygiene: release the reference
	h.data = h.data[:n-1]
	if n > 1 {
		h.siftDown(0)
	}
	return top, true
}

func (h *Heap[T]) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !h.less(h.data[i], h.data[parent]) {
			break
		}
		h.data[i], h.data[parent] = h.data[parent], h.data[i]
		i = parent
	}
}

func (h *Heap[T]) siftDown(i int) {
	n := len(h.data)
	for {
		left, right := 2*i+1, 2*i+2
		smallest := i
		if left < n && h.less(h.data[left], h.data[smallest]) {
			smallest = left
		}
		if right < n && h.less(h.data[right], h.data[smallest]) {
			smallest = right
		}
		if smallest == i {
			return
		}
		h.data[i], h.data[smallest] = h.data[smallest], h.data[i]
		i = smallest
	}
}

// checkInvariant walks the slice asserting the heap property; used by
// the property tests and available for your own.
func (h *Heap[T]) checkInvariant() bool {
	n := len(h.data)
	for i := 1; i < n; i++ {
		parent := (i - 1) / 2
		if h.less(h.data[i], h.data[parent]) {
			return false
		}
	}
	return true
}
