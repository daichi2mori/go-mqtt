package cache

import (
	"sync"
)

// Cache はgoroutine安全なシンプルなインメモリキャッシュです
type Cache struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// キャッシュのシングルトンインスタンス
var (
	instance *Cache
	once     sync.Once
)

// GetInstance はキャッシュのシングルトンインスタンスを返します
func GetInstance() *Cache {
	once.Do(func() {
		instance = &Cache{
			data: make(map[string]interface{}),
		}
	})
	return instance
}

// Set はキャッシュにデータを保存します
// 継続的に実行されるgoroutine間でのデータ共有に使用します
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = value
}

// Get はキャッシュからデータを取得します
// 存在しない場合は、第2戻り値がfalseになります
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, exists := c.data[key]
	return value, exists
}
