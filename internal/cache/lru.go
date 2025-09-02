// L0_project/internal/cache/lru.go
package cache

import (
	"container/list"
	"sync"
)

type Cache interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
}

type lruCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	queue    *list.List
}

type cacheItem struct {
	key   string
	value interface{}
}

func NewLRUCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		queue:    list.New(),
	}
}

func (c *lruCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*cacheItem).value = value
		return
	}

	if c.queue.Len() == c.capacity {
		c.removeOldest()
	}

	item := &cacheItem{key: key, value: value}
	element := c.queue.PushFront(item)
	c.items[key] = element
}

func (c *lruCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if element, exists := c.items[key]; exists {
		c.mu.RUnlock() // Release read lock before acquiring write lock
		c.mu.Lock()
		c.queue.MoveToFront(element)
		c.mu.Unlock()
		c.mu.RLock() // Re-acquire read lock
		return element.Value.(*cacheItem).value, true
	}

	return nil, false
}

func (c *lruCache) removeOldest() {
	element := c.queue.Back()
	if element != nil {
		item := c.queue.Remove(element).(*cacheItem)
		delete(c.items, item.key)
	}
}
