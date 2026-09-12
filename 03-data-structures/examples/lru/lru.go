// Package lru is the handbook's worked example for 03-data-structures
// chapter 9: the map-plus-list LRU with O(1) operations, an eviction
// hook, and hit/miss/eviction counters. Guarded by a mutex; the
// chapter discusses the sharding ladder for higher contention.
package lru

import (
	"container/list"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V
}

// LRU is a bounded cache evicting the least recently used entry.
// Get and Put both count metrics; Peek observes without promoting.
type LRU[K comparable, V any] struct {
	cap     int
	ll      *list.List // front = most recent
	item    map[K]*list.Element
	mu      sync.Mutex
	onEvict func(K, V)

	hits      uint64
	misses    uint64
	evictions uint64
}

func New[K comparable, V any](cap int) *LRU[K, V] {
	if cap < 1 {
		cap = 1
	}
	return &LRU[K, V]{cap: cap, ll: list.New(), item: make(map[K]*list.Element, cap)}
}

// OnEvict registers a callback invoked with each evicted pair, while
// the cache lock is held: keep it fast and non-blocking.
func (c *LRU[K, V]) OnEvict(fn func(K, V)) { c.onEvict = fn }

func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.item[key]; ok {
		c.ll.MoveToFront(e)
		c.hits++
		return e.Value.(*entry[K, V]).value, true
	}
	c.misses++
	var zero V
	return zero, false
}

// Peek observes without promoting: monitoring reads must not change
// eviction order.
func (c *LRU[K, V]) Peek(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.item[key]; ok {
		return e.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

func (c *LRU[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.item[key]; ok {
		c.ll.MoveToFront(e)
		e.Value.(*entry[K, V]).value = value
		return
	}
	e := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.item[key] = e
	if c.ll.Len() > c.cap {
		oldest := c.ll.Back()
		if oldest != nil {
			ent := oldest.Value.(*entry[K, V])
			c.ll.Remove(oldest)
			delete(c.item, ent.key)
			c.evictions++
			if c.onEvict != nil {
				c.onEvict(ent.key, ent.value)
			}
		}
	}
}

// Remove deletes a key explicitly.
func (c *LRU[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.item[key]; ok {
		ent := e.Value.(*entry[K, V])
		c.ll.Remove(e)
		delete(c.item, ent.key)
		return true
	}
	return false
}

func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Metrics returns hit, miss, and eviction counts.
func (c *LRU[K, V]) Metrics() (hits, misses, evictions uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses, c.evictions
}
