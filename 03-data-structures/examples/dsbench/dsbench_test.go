// Package dsbench holds the benchmarks behind the cost tables in
// 03-data-structures. Numbers here are for ratio comparison on one
// machine, not absolute truth: rerun on your hardware.
//
//	go test ./03-data-structures/examples/dsbench -bench . -benchmem
package dsbench

import "testing"

var sinkInt int
var sinkFound bool

// --- Scan vs map: the crossover -----------------------------------------

func makeSlice(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func makeMap(n int) map[int]struct{} {
	m := make(map[int]struct{}, n)
	for i := 0; i < n; i++ {
		m[i] = struct{}{}
	}
	return m
}

// BenchmarkScanVsMap measures the crossover the built-ins chapter
// claims: contiguous scans beat hash lookups until n is surprisingly
// large.
func BenchmarkScanVsMap(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 10000} {
		s := makeSlice(n)
		m := makeMap(n)
		needle := n / 2

		b.Run("scan_found", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				found := false
				for _, v := range s {
					if v == needle {
						found = true
						break
					}
				}
				sinkFound = found
			}
		})
		b.Run("map_found", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, ok := m[needle]
				sinkFound = ok
			}
		})
		b.Run("map_miss", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, ok := m[-1]
				sinkFound = ok
			}
		})
	}
}

// --- Append: size hint vs blind growth -----------------------------------

func BenchmarkAppendHinted(b *testing.B) {
	src := makeSlice(100000)
	for i := 0; i < b.N; i++ {
		out := make([]int, 0, len(src))
		out = append(out, src...)
		sinkInt = len(out)
	}
}

func BenchmarkAppendBlind(b *testing.B) {
	src := makeSlice(100000)
	for i := 0; i < b.N; i++ {
		var out []int
		out = append(out, src...)
		sinkInt = len(out)
	}
}

// --- Map insert: size hint -----------------------------------------------

func BenchmarkMapInsertHinted(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := make(map[int]int, 100000)
		for j := 0; j < 100000; j++ {
			m[j] = j
		}
		sinkInt = len(m)
	}
}

func BenchmarkMapInsertBlind(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := make(map[int]int)
		for j := 0; j < 100000; j++ {
			m[j] = j
		}
		sinkInt = len(m)
	}
}

// --- Stack and queue shapes ----------------------------------------------

type sliceStack struct{ items []int }

func (s *sliceStack) Push(v int) { s.items = append(s.items, v) }
func (s *sliceStack) Pop() (int, bool) {
	n := len(s.items)
	if n == 0 {
		return 0, false
	}
	v := s.items[n-1]
	s.items = s.items[:n-1]
	return v, true
}

func BenchmarkStackOps(b *testing.B) {
	var s sliceStack
	for i := 0; i < b.N; i++ {
		s.Push(i)
		s.Push(i)
		s.Pop()
	}
	sinkInt = len(s.items)
}

// resliceQueue is the leaky queue from chapter 3: dequeue walks the
// window forward, so the backing array grows with total ops.
func BenchmarkQueueReslice(b *testing.B) {
	q := make([]int, 0, 1024)
	for i := 0; i < b.N; i++ {
		q = append(q, i)
		if i%2 == 1 {
			q = q[1:]
		}
	}
	sinkInt = len(q)
}

// headQueue is the bounded-memory shape: head index + rewind.
type headQueue struct {
	items []int
	head  int
}

func (q *headQueue) Enqueue(v int) { q.items = append(q.items, v) }
func (q *headQueue) Dequeue() (int, bool) {
	if q.head >= len(q.items) {
		q.items = q.items[:0]
		q.head = 0
		return 0, false
	}
	v := q.items[q.head]
	q.head++
	return v, true
}

func BenchmarkQueueHeadIndex(b *testing.B) {
	var q headQueue
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
		if i%2 == 1 {
			q.Dequeue()
		}
	}
	sinkInt = q.head
}

// --- List vs slice: locality ---------------------------------------------

type node struct {
	val  int
	next *node
}

func buildList(n int) *node {
	var head, tail *node
	for i := 0; i < n; i++ {
		nd := &node{val: i}
		if head == nil {
			head, tail = nd, nd
			continue
		}
		tail.next = nd
		tail = nd
	}
	return head
}

func BenchmarkIterateList(b *testing.B) {
	const n = 1000000
	head := buildList(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := 0
		for cur := head; cur != nil; cur = cur.next {
			sum += cur.val
		}
		sinkInt = sum
	}
}

func BenchmarkIterateSlice(b *testing.B) {
	s := makeSlice(1000000)
	for i := 0; i < b.N; i++ {
		sum := 0
		for _, v := range s {
			sum += v
		}
		sinkInt = sum
	}
}
