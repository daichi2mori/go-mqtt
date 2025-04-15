package cache

import (
	"sync"
	"testing"
)

func TestCacheSetGet(t *testing.T) {
	cache := GetInstance()
	
	// 基本的なセットとゲット
	cache.Set("key1", "value1")
	value, exists := cache.Get("key1")
	
	if !exists {
		t.Errorf("Expected key1 to exist in cache")
	}
	
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
	
	// 存在しないキー
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Errorf("Expected nonexistent key to not exist")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := GetInstance()
	
	// 同時アクセスのテスト用の数
	concurrentAccess := 1000
	var wg sync.WaitGroup
	
	// 複数のgoroutineから同時に書き込みと読み取り
	for i := 0; i < concurrentAccess; i++ {
		wg.Add(1)
		
		go func(index int) {
			defer wg.Done()
			key := "concurrent_key"
			// 同じキーに対して複数のgoroutineが書き込みと読み取りを行う
			cache.Set(key, index)
			_, _ = cache.Get(key)
		}(i)
	}
	
	wg.Wait()
	
	// キャッシュに値が存在することを確認
	_, exists := cache.Get("concurrent_key")
	if !exists {
		t.Errorf("Expected concurrent_key to exist after concurrent operations")
	}
}
