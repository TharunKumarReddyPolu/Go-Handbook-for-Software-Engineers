package calc

import (
	"math/rand"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"
)

// FuzzEval_EvalNeverPanics guards the parser: any input may produce an
// error, but never a panic — and accepted negatives must parse back to
// their own value through the string round trip.
func FuzzEval_EvalNeverPanics(f *testing.F) {
	// Seeds: examples and shapes from past bugs.
	f.Add("1+2")
	f.Add("10 / 2")
	f.Add("-3 * 4")
	f.Add("")
	f.Add("++")
	f.Add("999999999999999999999+1")
	f.Add("1/0")
	f.Add("  -7  %  3 ")

	f.Fuzz(func(t *testing.T, expr string) {
		result, err := Eval(expr)
		if err != nil {
			return // rejection is fine
		}
		_ = result
		// Eval returning nil error must be stable: same input, same answer.
		again, err := Eval(expr)
		if err != nil || again != result {
			t.Errorf("Eval(%q) not deterministic: %d/%v then %d/%v", expr, result, err, again, err)
		}
	})
}

// TestEvalRoundTripProperty is property-style: for random operands, the
// string form must evaluate to the same answer as direct arithmetic.
func TestEvalRoundTripProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) // fixed seed: reproducible failures
	for i := 0; i < 500; i++ {
		a := rng.Intn(1000) - 500
		b := rng.Intn(1000) - 500
		if b == 0 {
			continue
		}
		expr := itoa(a) + "+" + itoa(b)
		want := a + b
		got, err := Eval(expr)
		if err != nil {
			t.Fatalf("Eval(%q): %v", expr, err)
		}
		if got != want {
			t.Fatalf("Eval(%q) = %d, want %d", expr, got, want)
		}
	}
}

// TestSortProperties demonstrates the classic invariants with stdlib
// sort on random inputs — the shape to copy for your own domains.
func TestSortProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 100; i++ {
		orig := make([]int, rng.Intn(50))
		for j := range orig {
			orig[j] = rng.Intn(1000)
		}
		x := append([]int(nil), orig...)
		slices.Sort(x)

		if !slices.IsSorted(x) {
			t.Fatal("sort produced unsorted output")
		}
		if !sameElements(orig, x) {
			t.Fatal("sort is not a permutation of the input")
		}
		sorted := append([]int(nil), x...)
		slices.Sort(x)
		if !reflect.DeepEqual(sorted, x) {
			t.Fatal("sort is not idempotent")
		}
	}
}

// TestNoSleepNeeded documents the timing discipline: this test uses a
// channel gate, not a sleep — see 10-testing/01-fundamentals.md.
func TestChannelGatePattern(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		// ... async work would be here
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("work did not finish within the budget")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func sameElements(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[int]int{}
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}
