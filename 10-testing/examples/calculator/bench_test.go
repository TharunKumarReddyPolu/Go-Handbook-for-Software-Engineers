package calc

import (
	"strconv"
	"strings"
	"testing"
)

// JoinBad is O(n^2): each += copies the accumulated string.
func JoinBad(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}

// JoinGood is O(n): strings.Builder amortizes the buffer growth.
func JoinGood(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

var sink string // package-level sink defeats dead-code elimination

func BenchmarkStringJoin(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		parts := make([]string, size)
		for i := range parts {
			parts[i] = strconv.Itoa(i)
		}
		b.Run("bad", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = JoinBad(parts)
			}
		})
		b.Run("good", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = JoinGood(parts)
			}
		})
	}
}

func BenchmarkEval(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Eval("1234 + 5678"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvalParallel exposes contention shapes if Eval were ever made
// stateful; today it is pure, so this mainly demonstrates the form.
func BenchmarkEvalParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := Eval("10 / 2"); err != nil {
				b.Fatal(err)
			}
		}
	})
}
