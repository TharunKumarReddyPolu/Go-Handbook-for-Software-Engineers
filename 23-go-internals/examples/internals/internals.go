// Package internals is the handbook's worked example for 23-go-internals:
// each function demonstrates one internal behavior with a test that pins
// it deterministically: escape analysis (via testing.AllocsPerRun), the
// typed-nil interface trap and its honest fix, and map iteration order
// discipline. Compiler-flag transcripts captured against this package
// are embedded in the chapters.
package internals

import "reflect"

// Sum adds a slice of ints. The compiler eliminates the bounds check
// inside the loop (see chapter 2's transcript: "Found IsInBounds"), so
// the loop body compiles to the raw load without a branch.
func Sum(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}

// sink escapes everything assigned to it: the compiler must assume it
// lives forever, so values stored here allocate on the heap.
var sink *int

// Sink returns the last escaped value: the observation point that
// makes the escape decision testable (and satisfies the linter that
// a write-only global would otherwise annoy).
func Sink() *int { return sink }

// Escaped returns a pointer that flows to the global sink: escape
// analysis must move the allocation to the heap.
func Escaped(v int) *int {
	p := &v // escapes: assigned to sink
	sink = p
	return p
}

// StackLocal returns a value derived from a local: nothing outlives the
// call, so v stays on the stack and the call allocates nothing.
func StackLocal(v int) int {
	p := &v // does not escape: local use only
	return *p + 1
}

// NilError is a concrete pointer type used as an error.
type NilError struct{ Code string }

func (*NilError) Error() string { return "nil error" }

// TypedNil demonstrates the trap from chapter 5: the returned value is
// a nil *NilError stored inside a non-nil error interface. Callers
// comparing `err != nil` see an error; errors.Is(err, target) correctly
// reports nothing matches.
func TypedNil() error {
	var e *NilError // nil pointer
	return e        // the interface: (type=*NilError, value=nil)
}

// HonestNil returns a true nil error: (type=nil, value=nil).
func HonestNil() error { return nil }

// Normalize returns nil for both true nil and typed-nil-with-nil-value
// errors: the production fix for APIs that may hand you the trap.
// Non-nil errors pass through untouched.
func Normalize(err error) error {
	if err == nil {
		return nil
	}
	if reflect.ValueOf(err).IsNil() {
		return nil
	}
	return err
}

// SortedKeys returns the map's keys in sorted order. Iteration order of
// a Go map is randomized by the runtime (chapter 4); any code that needs
// determinism sorts explicitly, exactly like this.
func SortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
