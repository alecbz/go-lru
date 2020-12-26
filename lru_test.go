package lru

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
)

func TestCache_BasicSetGet(t *testing.T) {
	c := New(5)
	c.Set("hello", "world")
	val, ok := c.Get("hello")
	if !ok {
		t.Fatalf("cache %v returned !ok for key %q", c, "hello")
	}
	if val != "world" {
		t.Errorf("got value %q for key %q, wanted %q", val, "hello", "world")
	}
}

func TestCache_MissingValue(t *testing.T) {
	c := New(5)
	assertNotHas(t, c, "foo")
}

func TestCache_ReachesCapacity(t *testing.T) {
	c := New(5)
	set(t, c, "a")
	set(t, c, "b")
	set(t, c, "c")
	set(t, c, "d")
	set(t, c, "e")

	assertHas(t, c, "a")
	assertHas(t, c, "b")
	assertHas(t, c, "c")
	assertHas(t, c, "d")
	assertHas(t, c, "e")
}
func TestCache_ExceedsCapacity(t *testing.T) {
	c := New(3)
	set(t, c, "a")
	set(t, c, "b")
	set(t, c, "c")
	set(t, c, "d")

	// 'a' was popped
	assertNotHas(t, c, "a")

	// but the rest remain
	assertHas(t, c, "b")
	assertHas(t, c, "c")
	assertHas(t, c, "d")
}

func TestCache_LRU(t *testing.T) {
	c := New(3)

	set(t, c, "a")
	set(t, c, "b")
	set(t, c, "c")

	// access 'a', so that b is now the oldest
	c.Get("a")

	// setting d causes b to be popped
	set(t, c, "d")
	assertNotHas(t, c, "b")

	assertHas(t, c, "a")
	assertHas(t, c, "c")
	assertHas(t, c, "d")
}

func TestCache_UpdateValue(t *testing.T) {
	c := New(2)

	set(t, c, "a")
	set(t, c, "b")
	set(t, c, "a")

	if c.Size() != 2 {
		t.Fatalf("expecting cache to only have two elements; got %v\n%v", c.Size(), c)
	}

	c.Set("a", "boop")
	val, ok := c.Get("a")
	if !ok {
		t.Fatalf("updated key %q not found in cache\n%v", "a", c)
	}
	if val != "boop" {
		t.Fatalf("updated key %q had value %v; wanted %v", "a", val, "boop")
	}
}

func TestCache_UpdatedValueLRU(t *testing.T) {
	c := New(2)

	set(t, c, "a")
	set(t, c, "b")
	set(t, c, "a")

	set(t, c, "c") // should kick out b

	assertHas(t, c, "a")
	assertNotHas(t, c, "b")
	assertHas(t, c, "c")
}

func TestCache_MultipleGets(t *testing.T) {
	c := New(2)

	set(t, c, "a")

	assertHas(t, c, "a")
	assertHas(t, c, "a")
}

func TestCache_MultipleSets(t *testing.T) {
	c := New(2)

	set(t, c, "a")
	set(t, c, "a")
}

func set(t *testing.T, c *Cache, k string) {
	t.Helper()
	c.Set(k, k)
	assertValid(t, c)
}

func assertHas(t *testing.T, c *Cache, k string) {
	t.Helper()

	if !c.Has(k) {
		t.Errorf("cache does not contain key %q\n%v", k, c)
		return
	}

	val, ok := c.Get(k)
	if !ok {
		t.Errorf("Get(%q) returned !ok after .Has returned true", k)
		return
	}
	if val != k {
		t.Errorf("got value %q for key %q; expecting %q", val, k, k)
	}
	assertValid(t, c)
}

func assertNotHas(t *testing.T, c *Cache, k string) {
	t.Helper()

	if c.Has(k) {
		t.Errorf("cache has key %q; should not\n%v", k, c)
		return
	}

	val, ok := c.Get(k)
	if val != nil {
		t.Errorf("got non-nil value from non-existent key %q", k)
	}
	if ok {
		t.Errorf("Get(%q) returned ok after .Has returned false", k)
	}
	assertValid(t, c)
}

func assertValid(t *testing.T, c *Cache) {
	t.Helper()
	if c.head == nil {
		return
	}
	seen := make(map[*entry]struct{})
	var a, b *entry
	for a, b = c.head, c.head.next; b != nil; a, b = b, b.next {
		if _, ok := seen[a]; ok {
			t.Fatalf("cycle in linked list at node %v", a)
		}
		seen[a] = struct{}{}
		if b.prev != a {
			t.Fatalf("%v does not point back to %v", b, a)
		}
	}
	if a != c.tail {
		t.Fatalf("last seen element (%v) is not the tail (%v)", a, c.tail)
	}
	if _, ok := seen[a]; ok {
		t.Fatalf("cycle in linked list at node %v", a)
	}
}

func BenchmarkCache_Set(b *testing.B) {
	for _, capacity := range []int{10, 100, 1000, 1e6, 1e9} {
		for _, keyFactor := range []float64{0.1, 0.5, 1, 2, 10} {
			keySpace := int(math.Floor(float64(capacity) * keyFactor))
			if keySpace == 0 {
				keySpace = 1
			}
			b.Run(fmt.Sprintf("cap=%d,keyspace=%d", capacity, keySpace), func(b *testing.B) {
				var keys []string
				for i := 0; i < b.N; i++ {
					keys = append(keys, strconv.Itoa(rand.Intn(keySpace)))
				}
				c := New(capacity)
				b.ResetTimer()

				for i, key := range keys {
					c.Set(key, i)
				}
			})
		}
	}
}

func BenchmarkCache_Get(b *testing.B) {
	for _, capacity := range []int{10, 100, 1000, 1e6, 1e9} {
		for _, keyFactor := range []float64{0.1, 0.5, 1, 2, 10} {
			keySpace := int(math.Floor(float64(capacity) * keyFactor))
			if keySpace == 0 {
				keySpace = 1
			}
			b.Run(fmt.Sprintf("cap=%d,keyspace=%d", capacity, keySpace), func(b *testing.B) {
				c := New(capacity)
				for i := 0; i < b.N; i++ {
					key :=  strconv.Itoa(rand.Intn(keySpace))
					c.Set(key, i)
				}
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					key :=  strconv.Itoa(rand.Intn(keySpace))
					c.Get(key)
				}
			})
		}
	}
}