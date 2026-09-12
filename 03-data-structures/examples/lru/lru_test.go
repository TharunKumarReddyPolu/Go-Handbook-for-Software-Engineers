package lru

import (
	"reflect"
	"sync"
	"testing"
)

func TestLRU_BasicContract(t *testing.T) {
	c := New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = %d, %v", v, ok)
	}
	c.Put("c", 3) // evicts b (least recently used)

	if _, ok := c.Peek("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d, want 2", c.Len())
	}
}

func TestLRU_Promotion(t *testing.T) {
	c := New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")    // promote a: b is now LRU
	c.Put("c", 3) // evicts b

	if _, ok := c.Peek("a"); !ok {
		t.Fatal("a should survive: it was promoted")
	}
	if _, ok := c.Peek("b"); ok {
		t.Fatal("b should have been evicted")
	}
}

func TestLRU_PeekDoesNotPromote(t *testing.T) {
	c := New[string, int](2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Peek("a")   // observe only: a stays LRU
	c.Put("c", 3) // evicts a

	if _, ok := c.Peek("a"); ok {
		t.Fatal("Peek must not promote: a should have been evicted")
	}
}

func TestLRU_EvictionCallback(t *testing.T) {
	type evict struct {
		k string
		v int
	}
	var got []evict
	c := New[string, int](2)
	c.OnEvict(func(k string, v int) { got = append(got, evict{k, v}) })

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)
	c.Put("d", 4)

	want := []evict{{"a", 1}, {"b", 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("evictions = %v, want %v", got, want)
	}
	_, _, evictions := c.Metrics()
	if evictions != 2 {
		t.Fatalf("eviction metric = %d, want 2", evictions)
	}
}

func TestLRU_Metrics(t *testing.T) {
	c := New[string, int](1)
	c.Put("a", 1)
	c.Get("a")
	c.Get("missing")
	c.Put("b", 2) // evicts a

	hits, misses, evictions := c.Metrics()
	if hits != 1 || misses != 1 || evictions != 1 {
		t.Fatalf("metrics = (%d, %d, %d), want (1, 1, 1)", hits, misses, evictions)
	}
}

// TestLRU_Differential replays random op sequences against a
// reference implementation (map + explicit order) for small n.
func TestLRU_Differential(t *testing.T) {
	const (
		capacity = 3
		ops      = 500
	)
	c := New[int, int](capacity)
	refKeys := []int{} // back = LRU order, index 0 = most recent
	refVals := map[int]int{}

	next := 0
	for op := 0; op < ops; op++ {
		switch op % 3 {
		case 0, 1: // Put new key
			c.Put(next, next*10)
			refVals[next] = next * 10
			refKeys = append([]int{next}, refKeys...)
			if len(refKeys) > capacity {
				evicted := refKeys[len(refKeys)-1]
				refKeys = refKeys[:len(refKeys)-1]
				delete(refVals, evicted)
			}
			next++
		case 2: // Get a random live key (promotes in both)
			if len(refKeys) == 0 {
				continue
			}
			i := (op / 7) % len(refKeys)
			key := refKeys[i]
			if v, ok := c.Get(key); !ok || v != refVals[key] {
				t.Fatalf("op %d: Get(%d) = %d, %v; ref %d", op, key, v, ok, refVals[key])
			}
			refKeys = append([]int{key}, append(refKeys[:i:i], refKeys[i+1:]...)...)
		}
		if c.Len() != len(refKeys) {
			t.Fatalf("op %d: Len = %d, ref %d", op, c.Len(), len(refKeys))
		}
		for _, k := range refKeys {
			if _, ok := c.Peek(k); !ok {
				t.Fatalf("op %d: ref key %d missing from cache", op, k)
			}
		}
	}
}

func TestLRU_ConcurrentStorm(t *testing.T) {
	c := New[int, int](64)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				k := (g*1000 + i) % 128
				c.Put(k, i)
				c.Get(k)
				c.Peek(k)
			}
		}(g)
	}
	wg.Wait()
	if c.Len() > 64 {
		t.Fatalf("capacity violated: Len = %d", c.Len())
	}
}
