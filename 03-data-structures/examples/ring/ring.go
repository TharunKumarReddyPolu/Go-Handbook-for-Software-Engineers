// Package ring is the handbook's worked example for 03-data-structures
// chapter 8: a fixed-capacity FIFO with drop-oldest eviction, plus the
// bounded RecentKeys dedup shape from that chapter's idioms section.
package ring

// Ring is a fixed-capacity FIFO with drop-oldest policy. Not safe for
// concurrent use; guard with a mutex or give it a single owner.
type Ring[T any] struct {
	items []T
	head  int // next to pop
	count int // live elements
}

// NewRing builds a ring of the given capacity; capacity < 1 becomes 1.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring[T]{items: make([]T, capacity)}
}

func (r *Ring[T]) Len() int { return r.count }
func (r *Ring[T]) Cap() int { return len(r.items) }

// Push appends v. If the ring is full, the oldest element is evicted
// and returned: drop-oldest policy, made observable.
func (r *Ring[T]) Push(v T) (evicted T, didEvict bool) {
	// Evict BEFORE writing: when full, tail == head, so writing first
	// would clobber the oldest element and return it as the new value.
	var zero T
	if r.count == len(r.items) {
		evicted = r.items[r.head] // read the oldest first
		r.items[r.head] = zero    // hygiene: release the reference
		r.head = (r.head + 1) % len(r.items)
		r.count--
		didEvict = true
	}
	tail := (r.head + r.count) % len(r.items)
	r.items[tail] = v
	r.count++
	return evicted, didEvict
}

// Pop removes and returns the oldest element.
func (r *Ring[T]) Pop() (T, bool) {
	var zero T
	if r.count == 0 {
		return zero, false
	}
	v := r.items[r.head]
	r.items[r.head] = zero // hygiene
	r.head = (r.head + 1) % len(r.items)
	r.count--
	return v, true
}

// Map applies f to every element in FIFO order and returns a ring of
// the same capacity holding the results. The method declares its own
// type parameter U: generic methods, introduced in Go 1.27
// (07-generics/06-version-notes.md). CI compiling and testing this
// method is the handbook's empirical pin on that version claim.
func (r *Ring[T]) Map[U any](f func(T) U) *Ring[U] {
	out := NewRing[U](len(r.items))
	for i := 0; i < r.count; i++ {
		idx := (r.head + i) % len(r.items)
		out.Push(f(r.items[idx]))
	}
	return out
}

// Peek returns the oldest element without removing it.
func (r *Ring[T]) Peek() (T, bool) {
	var zero T
	if r.count == 0 {
		return zero, false
	}
	return r.items[r.head], true
}

// RecentKeys is the bounded-memory dedup shape from chapter 8: a ring
// bounds it, a map indexes it, eviction cleans the map.
//
// Contract: Seen reports whether k was PUSHED within the last Size
// pushes. Hits do not refresh the window: the ring is the recency
// record, and the map is membership only. (Refreshing on hit would
// require O(1) middle-removal, which a ring cannot do; state that
// tradeoff when you design one of these.)
type RecentKeys struct {
	ring *Ring[string]
	seen map[string]struct{}
}

func NewRecentKeys(size int) *RecentKeys {
	return &RecentKeys{ring: NewRing[string](size), seen: make(map[string]struct{}, size)}
}

// Seen reports whether k was pushed within the last Size pushes.
// Note the contract: Seen RECORDS k when it was not present (and that
// push may evict the oldest entry). Use Contains for a pure query.
func (r *RecentKeys) Seen(k string) bool {
	if _, ok := r.seen[k]; ok {
		return true
	}
	if old, evicted := r.ring.Push(k); evicted {
		delete(r.seen, old) // the map and ring must stay in lockstep
	}
	r.seen[k] = struct{}{}
	return false
}

// Contains observes the window without recording: a pure query.
func (r *RecentKeys) Contains(k string) bool {
	_, ok := r.seen[k]
	return ok
}

func (r *RecentKeys) Size() int { return r.ring.Cap() }
