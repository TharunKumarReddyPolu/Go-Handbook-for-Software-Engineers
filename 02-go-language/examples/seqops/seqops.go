// Package seqops is the handbook's worked example for section 02:
// slice aliasing, map semantics, and pointer-vs-value receiver rules,
// each with a deterministic test next to it.
package seqops

// Chunk returns non-overlapping chunks of size n sharing s's backing
// array. The last chunk is shorter when len(s) is not a multiple of n.
// n <= 0 returns nil. Aliasing is deliberate: document it if you wrap.
func Chunk[T any](s []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	out := make([][]T, 0, (len(s)+n-1)/n)
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

// CopyChunks is Chunk with an independent backing array per chunk.
// Costlier, but the caller can mutate freely without touching s.
func CopyChunks[T any](s []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	out := make([][]T, 0, (len(s)+n-1)/n)
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		chunk := make([]T, end-i)
		copy(chunk, s[i:end])
		out = append(out, chunk)
	}
	return out
}

// FilterAlloc returns the elements pred keeps, pre-sized when the
// caller can guess a capacity. The input is never mutated.
func FilterAlloc[T any](s []T, pred func(T) bool) []T {
	out := make([]T, 0, len(s)) // upper bound; cap may over-allocate
	for _, v := range s {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// FilterInPlace compacts s into its own backing array and returns the
// shortened view. The input's tail is clobbered: value semantics end
// where sharing begins.
func FilterInPlace[T any](s []T, pred func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// Truncate cuts s to maxRunes without splitting a UTF-8 rune. It
// walks byte offsets with range so multi-byte runes stay intact.
func Truncate(s string, maxRunes int) string {
	if maxRunes < 0 {
		return ""
	}
	n := 0
	for i := range s { // i is the byte offset of each rune
		if n == maxRunes {
			return s[:i]
		}
		n++
	}
	return s // shorter than the limit
}
