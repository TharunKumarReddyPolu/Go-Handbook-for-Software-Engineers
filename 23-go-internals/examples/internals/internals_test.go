package internals

import (
	"errors"
	"testing"
)

// Chapter 1/2: escape analysis, pinned with AllocsPerRun. The
// compiler's decision is observable as an allocation count.

func TestEscaped_Allocates(t *testing.T) {
	avg := testing.AllocsPerRun(100, func() {
		if p := Escaped(42); *p != 42 {
			t.Fatal("wrong value")
		}
	})
	if avg != 1 {
		t.Fatalf("Escaped allocated %v times per run; heap escape did not happen", avg)
	}
}

func TestStackLocal_DoesNotAllocate(t *testing.T) {
	avg := testing.AllocsPerRun(100, func() {
		if got := StackLocal(42); got != 43 {
			t.Fatal("wrong value")
		}
	})
	if avg != 0 {
		t.Fatalf("StackLocal allocated %v times per run; wanted stack-only", avg)
	}
}

// Chapter 5: the typed-nil trap and the honest fix.

func TestTypedNil_TrapAndFix(t *testing.T) {
	err := TypedNil()
	// The trap: the interface is non-nil even though the pointer is nil.
	if err == nil {
		t.Fatal("expected the typed-nil trap: interface must be non-nil")
	}
	// errors.Is sees no match: nothing equals the typed nil.
	if errors.Is(err, &NilError{Code: "x"}) {
		t.Fatal("unexpected match")
	}
	if Normalize(err) != nil {
		t.Fatal("Normalize must collapse the typed nil to true nil")
	}
	if Normalize(nil) != nil {
		t.Fatal("Normalize(nil) must stay nil")
	}
	real := errors.New("real")
	if Normalize(real) != real {
		t.Fatal("Normalize must pass real errors through")
	}
	if HonestNil() != nil {
		t.Fatal("HonestNil must return true nil")
	}
}

// Chapter 4: map iteration order is randomized; SortedKeys is the
// discipline. Failing order-sensitivity this loudly would break the
// 100-run window for at least one fixed map.
func TestSortedKeys_Deterministic(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1, "c": 3}
	want := []string{"a", "b", "c"}
	for i := 0; i < 100; i++ {
		got := SortedKeys(m)
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("run %d: order %v, want %v", i, got, want)
			}
		}
	}
}

// Chapter 1/2: Sum is correct and (per the transcript in chapter 2)
// compiles without a bounds check in the loop body.
func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3, 4}); got != 10 {
		t.Fatalf("Sum = %d, want 10", got)
	}
	if got := Sum(nil); got != 0 {
		t.Fatalf("Sum(nil) = %d, want 0", got)
	}
}
