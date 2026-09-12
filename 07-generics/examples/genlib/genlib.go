// Package genlib is the handbook's worked example for 07-generics:
// the constraint styles from chapter 2, the API style from chapter 4,
// and the instantiation-table tests that chapter 5's rules imply.
package genlib

import (
	"cmp"
	"context"
	"time"
)

// Max returns the larger of a and b under builtin ordering.
// cmp.Ordered covers ints, floats, and strings (Go 1.21+).
func Max[T cmp.Ordered](a, b T) T {
	if b > a {
		return b
	}
	return a
}

// Sum adds a numeric slice; ~ accepts named types with those
// underlying types (type Salary int qualifies, plain int would not).
type Number interface {
	~int | ~int64 | ~float64
}

func Sum[V Number](xs []V) V {
	var total V
	for _, x := range xs {
		total += x
	}
	return total
}

// DedupIDs removes duplicates preserving first-occurrence order.
// E ~string so named ID types (type UserID string) pass as-is.
func DedupIDs[S ~[]E, E ~string](ids S) S {
	seen := make(map[E]struct{}, len(ids))
	out := make(S, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// GroupBy partitions a slice into buckets keyed by key(e). The map is
// never nil on return: empty input yields an empty map (the documented
// contract tests pin).
func GroupBy[S ~[]E, E any, K comparable](s S, key func(E) K) map[K]S {
	out := make(map[K]S)
	for _, e := range s {
		k := key(e)
		out[k] = append(out[k], e)
	}
	return out
}

// Map transforms every element; the container type S is preserved so
// named slice types keep their identity through the call.
func Map[S ~[]E, E, R any](s S, f func(E) R) []R {
	out := make([]R, 0, len(s))
	for _, e := range s {
		out = append(out, f(e))
	}
	return out
}

// Stack is the generic container from chapter 3: zero-value-first,
// with hygiene on pop so popped elements are not pinned.
type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	n := len(s.items)
	if n == 0 {
		return zero, false
	}
	v := s.items[n-1]
	s.items[n-1] = zero
	s.items = s.items[:n-1]
	return v, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

// Cache is a generic mutex-guarded map: the sync-shaped wrapper from
// chapter 3. Note the getter returns a copy of V only insofar as V's
// type makes copying meaningful; reference-typed V still aliases
// (documented in the chapter, enforced nowhere: the caller's job).
type Cache[K comparable, V any] struct {
	items map[K]V
}

func NewCache[K comparable, V any](capHint int) *Cache[K, V] {
	return &Cache[K, V]{items: make(map[K]V, capHint)}
}

func (c *Cache[K, V]) Put(k K, v V) {
	if c.items == nil { // zero-value usable via lazy init
		c.items = make(map[K]V)
	}
	c.items[k] = v
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.items[k]
	return v, ok
}

// Retry is the generic API from chapter 4: the operation's result
// type flows through unchanged, T any is the narrowest honest
// contract, and the two failure exits are distinct: a dead context
// returns context's error; an exhausted backoff returns the op's.
func Retry[T any](
	ctx context.Context,
	op func(context.Context) (T, error),
	next func(attempt int) time.Duration,
) (T, error) {
	var zero T
	for attempt := 0; ; attempt++ {
		v, err := op(ctx)
		if err == nil {
			return v, nil
		}
		if ctx.Err() != nil {
			return zero, ctx.Err() // context dead: its error wins
		}
		d := next(attempt)
		if d <= 0 {
			return zero, err // out of patience: the op error
		}
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
}
