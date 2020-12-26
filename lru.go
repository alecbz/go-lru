package lru

import (
	"fmt"
	"strings"
)

type Cache struct {
	head, tail *entry
	entries    map[string]*entry
	cap        int
}

func New(cap int) *Cache {
	return &Cache{
		entries: make(map[string]*entry),
		cap:     cap,
	}
}

type entry struct {
	key string
	value interface{}
	next, prev *entry
}

func (e *entry) String() string {
	return fmt.Sprintf("entry[%q: %v]", e.key, e.value)
}

func (c *Cache) Set(key string, value interface{}) {
	e, ok := c.entries[key]
	if !ok {
		e = &entry{key: key, value: value}
		c.entries[key] = e
	} else {
		e.value = value
	}

	c.moveFront(e)

	if len(c.entries) > c.cap {
		c.pop()
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.moveFront(e)
	return e.value, true
}

func (c *Cache) Has(key string) bool {
	_, ok := c.entries[key]
	return ok
}


func (c *Cache) Size() int {
	return len(c.entries)
}

// Used to test internal validity of the cache's linked list. Useful in debugging implementation changes.

func (c *Cache) String() string {
	if c.head == nil {
		return fmt.Sprint(c.head)
	}
	var parts []string
	seen := make(map[*entry]struct{})
	for e := c.head; e != nil; e = e.next {
		if _, ok := seen[e]; ok {
			panic("cycle in cache linked list")
		}
		seen[e] = struct{}{}
		parts = append(parts, e.String())
	}
	return strings.Join(parts, " -> ")
}

func (c *Cache) moveFront(e *entry) {
	if e == c.head {
		return
	}

	if e.next != nil {
		e.next.prev = e.prev
	}
	if e.prev != nil {
		e.prev.next = e.next
	}

	if e == c.tail {
		// Update the tail if we're moving the tail to the front
		c.tail = e.prev
	}

	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e

	// Initialize the tail if we don't have one:
	if c.tail == nil {
		c.tail = e
	}
}

func (c *Cache) pop() {
	if c.tail == nil {
		panic("pop called with no tail")
	}
	if c.tail.prev != nil {
		c.tail.prev.next = nil
	}
	delete(c.entries, c.tail.key)
	c.tail = c.tail.prev
}