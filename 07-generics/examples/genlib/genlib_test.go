package genlib

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// TestMax_Instantiations is the instantiation-table pattern: one
// table, multiple type arguments, constraint mistakes surface as
// compile errors per instantiation.
func TestMax_Instantiations(t *testing.T) {
	if Max(3, 7) != 7 {
		t.Fatal("int")
	}
	if Max(2.5, 2.4) != 2.5 {
		t.Fatal("float64")
	}
	if Max("b", "a") != "b" {
		t.Fatal("string")
	}
	type Duration64 int64 // named type with underlying int64
	var a, b Duration64 = 5, 9
	if Max(a, b) != b {
		t.Fatal("named ~int64 type")
	}
}

// TestSum_TildeConstraint pins the ~ behavior: named numeric types
// work; a non-numeric named type would fail at compile time (kept as
// a comment: it is the compile-check).
func TestSum_TildeConstraint(t *testing.T) {
	type Celsius int
	if got := Sum([]Celsius{1, 2, 3}); got != 6 {
		t.Fatalf("Sum over named ~int type: %d", got)
	}
	if got := Sum([]float64{1.5, 2.5}); got != 4.0 {
		t.Fatalf("Sum over float64: %v", got)
	}
	// The compile-check that the constraint is honest:
	// Sum([]string{"a"}) // does not compile: string not in Number.
}

func TestDedupIDs_NamedTypes(t *testing.T) {
	type UserID string
	in := []UserID{"a", "b", "a", "c", "b"}
	got := DedupIDs(in)
	want := []UserID{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DedupIDs = %v, want %v", got, want)
	}

	// Plain strings: same function, second instantiation.
	got2 := DedupIDs([]string{"x", "x", "y"})
	if !reflect.DeepEqual(got2, []string{"x", "y"}) {
		t.Fatalf("DedupIDs(string) = %v", got2)
	}
}

func TestGroupBy_EmptyAndFilled(t *testing.T) {
	type Task struct {
		Priority int
		Name     string
	}
	tasks := []Task{
		{1, "a"}, {2, "b"}, {1, "c"},
	}
	got := GroupBy(tasks, func(t Task) int { return t.Priority })

	if len(got) != 2 {
		t.Fatalf("buckets = %d, want 2", len(got))
	}
	if len(got[1]) != 2 || len(got[2]) != 1 {
		t.Fatalf("bucket sizes wrong: %+v", got)
	}

	// The documented contract: empty input yields an empty (non-nil)
	// map.
	if empty := GroupBy([]Task(nil), func(t Task) int { return t.Priority }); empty == nil || len(empty) != 0 {
		t.Fatalf("empty input: %v, want empty non-nil map", empty)
	}
}

func TestMap_PreservesLengthAndOrder(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(v int) string {
		return string(rune('a' + v - 1))
	})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Map = %v, want %v", got, want)
	}
}

func TestStack_LIFO_AndHygiene(t *testing.T) {
	var s Stack[string]
	s.Push("a")
	s.Push("b")
	if s.Len() != 2 {
		t.Fatalf("Len = %d", s.Len())
	}
	if v, ok := s.Pop(); !ok || v != "b" {
		t.Fatalf("Pop = %q, %v; want b", v, ok)
	}
	if v, ok := s.Pop(); !ok || v != "a" {
		t.Fatalf("Pop = %q, %v; want a", v, ok)
	}
	if _, ok := s.Pop(); ok {
		t.Fatal("empty Pop must miss")
	}

	// Hygiene: after popping, the internal slot must not keep the
	// value reachable through the slice.
	s.Push("x")
	s.Pop()
	if len(s.items) != 0 || (len(s.items) > 0 && s.items[0] != "") {
		t.Fatalf("popped slot not zeroed: %q", s.items)
	}
}

func TestCache_ZeroValueUsable(t *testing.T) {
	var c Cache[string, int] // zero value: lazy map init in Put
	c.Put("k", 1)
	if v, ok := c.Get("k"); !ok || v != 1 {
		t.Fatalf("zero-value cache failed: %d, %v", v, ok)
	}
	if _, ok := c.Get("missing"); ok {
		t.Fatal("missing key must miss")
	}
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	attempts := 0
	next := func(attempt int) time.Duration { return time.Microsecond } // fast, positive: keep retrying
	v, err := Retry(context.Background(), func(context.Context) (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("flaky")
		}
		return 42, nil
	}, next)
	if err != nil || v != 42 {
		t.Fatalf("Retry = %d, %v", v, err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestRetry_StopsWhenBackoffExhausted(t *testing.T) {
	calls := 0
	next := func(attempt int) time.Duration {
		if attempt >= 2 {
			return 0 // give up signal
		}
		return time.Millisecond
	}
	_, err := Retry(context.Background(), func(context.Context) (string, error) {
		calls++
		return "", errors.New("down")
	}, next)
	if err == nil {
		t.Fatal("exhausted backoff must return the error")
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (initial + 2 retries)", calls)
	}
}

func TestRetry_ContextCancelHonored(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-canceled
	next := func(int) time.Duration { return time.Hour }
	_, err := Retry(ctx, func(context.Context) (bool, error) {
		return false, errors.New("down")
	}, next)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled ctx: %v", err)
	}
}
