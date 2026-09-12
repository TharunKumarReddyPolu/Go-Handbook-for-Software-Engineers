package seqops

import (
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestChunk_SharesBackingArray(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	got := Chunk(s, 2)

	want := [][]int{{1, 2}, {3, 4}, {5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Chunk(%v, 2) = %v, want %v", s, got, want)
	}

	// The aliasing contract: writing through a chunk mutates s.
	got[0][0] = 99
	if s[0] != 99 {
		t.Fatalf("Chunk should share the backing array; s[0] = %d, want 99", s[0])
	}
}

func TestCopyChunks_Independent(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	got := CopyChunks(s, 2)

	got[0][0] = 99
	if s[0] != 1 {
		t.Fatalf("CopyChunks should not share storage; s[0] = %d, want 1", s[0])
	}
}

func TestChunk_BadSize(t *testing.T) {
	if got := Chunk([]int{1}, 0); got != nil {
		t.Fatalf("Chunk with n=0 = %v, want nil", got)
	}
}

func TestFilterAlloc_DoesNotMutateInput(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6}
	got := FilterAlloc(s, func(v int) bool { return v%2 == 0 })

	want := []int{2, 4, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterAlloc = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(s, []int{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("input mutated: %v", s)
	}
}

func TestFilterInPlace_KeepsOrder_ClobbersTail(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6}
	got := FilterInPlace(s, func(v int) bool { return v%2 == 0 })

	if !reflect.DeepEqual(got, []int{2, 4, 6}) {
		t.Fatalf("FilterInPlace = %v, want [2 4 6]", got)
	}
	// The backing array tail still holds old values: len hides them,
	// but reslicing past got's len would resurrect them. That is the
	// documented contract.
	if !reflect.DeepEqual(s[:3], []int{2, 4, 6}) {
		t.Fatalf("prefix should be compacted, got %v", s[:3])
	}
}

func TestTruncate_RuneSafe(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		maxRunes int
		want     string
	}{
		{"ascii", "hello", 3, "hel"},
		{"accent", "héllo", 2, "hé"},        // é is 2 bytes; never split
		{"cjk", "田中さん", 2, "田中"},            // each rune is 3 bytes
		{"zwj-family", "👨‍👩‍👧x", 1, "👨"},    // family is 5 runes: rune truncation splits it
		{"zwj-first", "👨‍👩‍👧x", 5, "👨‍👩‍👧"}, // 5 runes keeps the family, drops x
		{"no-op", "hi", 5, "hi"},
		{"zero", "hi", 0, ""},
		{"negative", "hi", -1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.in, tt.maxRunes)
			if got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.in, tt.maxRunes, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("result is not valid UTF-8: %q", got)
			}
		})
	}
}
