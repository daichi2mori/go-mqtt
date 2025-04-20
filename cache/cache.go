package cache

import "sync"

type CacheInterface[T any] interface {
	Get(key string) (T, bool)
	Set(key string, val T)
	Delete(key string)
}

type Cache[T any] struct {
	val map[string]T
	mu  sync.Mutex
}

func New[T any]() *Cache[T] {
	return &Cache[T]{
		val: make(map[string]T),
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	val, ok := c.val[key]
	return val, ok
}

func (c *Cache[T]) Set(key string, val T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val[key] = val
}

func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.val, key)
}
