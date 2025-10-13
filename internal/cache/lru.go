package cache

import (
	"L0_project/internal/model"
	"container/list"
	"sync"
)

type lruCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	queue    *list.List
}

type cacheItem struct {
	key   string
	value *model.Order
}

// NewLRUCache создает новый экземпляр LRU-кэша.
func NewLRUCache(capacity int) OrderCache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		queue:    list.New(),
	}
}

// Add добавляет заказ в кэш.
func (c *lruCache) Add(key string, order *model.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*cacheItem).value = order
		return
	}

	if c.queue.Len() == c.capacity {
		c.removeOldest()
	}

	item := &cacheItem{key: key, value: order}
	element := c.queue.PushFront(item)
	c.items[key] = element
}

// Get извлекает заказ из кэша.
func (c *lruCache) Get(key string) (*model.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if element, exists := c.items[key]; exists {
		c.queue.MoveToFront(element)
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
