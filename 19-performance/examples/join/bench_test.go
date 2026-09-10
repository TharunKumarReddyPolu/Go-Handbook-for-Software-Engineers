// String building compared: += copies the accumulated string every
// iteration (O(n^2)); strings.Builder amortizes (O(n)). Companion to
// 19-performance/01-measure-first.md. Run with:
//
//	go test ./19-performance/examples/join -bench=. -benchmem
package main

import (
	"strconv"
	"strings"
	"testing"
)

func joinPlus(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p
	}
	return s
}

func joinBuilder(parts []string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

var sink string

func BenchmarkJoin(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		parts := make([]string, size)
		total := 0
		for i := range parts {
			parts[i] = strconv.Itoa(i)
			total += len(parts[i])
		}
		b.Run("plus", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = joinPlus(parts)
			}
		})
		b.Run("builder", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sink = joinBuilder(parts)
			}
		})
		_ = total
	}
}
