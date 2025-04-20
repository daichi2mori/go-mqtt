package cache

import (
	"sync"
	"testing"
)

func TestCache_Basic(t *testing.T) {
	// 基本的なキャッシュ操作をテストする
	c := New[string]()

	// 空のキャッシュからの取得
	val, ok := c.Get("nonexistent")
	if ok {
		t.Errorf("Expected Get to return not ok for nonexistent key")
	}
	if val != "" {
		t.Errorf("Expected zero value for nonexistent key, got %v", val)
	}

	// 値のセット
	c.Set("key1", "value1")

	// 値の取得
	val, ok = c.Get("key1")
	if !ok {
		t.Errorf("Expected Get to return ok for existing key")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got '%s'", val)
	}

	// 値の削除
	c.Delete("key1")
	val, ok = c.Get("key1")
	if ok {
		t.Errorf("Expected Get to return not ok after Delete")
	}

	// 値の上書き
	c.Set("key2", "value2-original")
	c.Set("key2", "value2-updated")
	val, ok = c.Get("key2")
	if !ok {
		t.Errorf("Expected Get to return ok for existing key")
	}
	if val != "value2-updated" {
		t.Errorf("Expected 'value2-updated', got '%s'", val)
	}
}

func TestCache_Types(t *testing.T) {
	// 異なる型での動作をテスト
	type testStruct struct {
		Field string
	}

	// 文字列キャッシュ
	cString := New[string]()
	cString.Set("key", "value")
	val, _ := cString.Get("key")
	if val != "value" {
		t.Errorf("String cache: expected 'value', got '%s'", val)
	}

	// 整数キャッシュ
	cInt := New[int]()
	cInt.Set("key", 42)
	num, _ := cInt.Get("key")
	if num != 42 {
		t.Errorf("Int cache: expected 42, got %d", num)
	}

	// 構造体キャッシュ
	cStruct := New[testStruct]()
	cStruct.Set("key", testStruct{Field: "test"})
	s, _ := cStruct.Get("key")
	if s.Field != "test" {
		t.Errorf("Struct cache: expected Field 'test', got '%s'", s.Field)
	}
}

func TestCache_Concurrency(t *testing.T) {
	// 並行処理の安全性をテスト
	c := New[int]()
	done := make(chan bool)

	// 100個のgoroutineで同時に操作
	for i := 0; i < 100; i++ {
		go func(idx int) {
			c.Set(string(rune(idx+'a')), idx)
			done <- true
		}(i)
	}

	// すべてのgoroutineの完了を待つ
	for i := 0; i < 100; i++ {
		<-done
	}

	// すべてのキーが正しく保存されていることを確認
	for i := 0; i < 100; i++ {
		val, ok := c.Get(string(rune(i + 'a')))
		if !ok {
			t.Errorf("Key %c not found", rune(i+'a'))
		}
		if val != i {
			t.Errorf("Expected value %d for key %c, got %d", i, rune(i+'a'), val)
		}
	}
}

func TestCache_Concurrency_ReadWrite(t *testing.T) {
	// 読み取りと書き込みの並行処理をテスト
	c := New[int]()
	const numReaders = 10
	const numWriters = 5
	const numOperations = 100

	// テスト用のデータをセットアップ
	for i := 0; i < 10; i++ {
		c.Set(string(rune(i+'a')), i)
	}

	// 同期用チャネル
	done := make(chan bool)
	var wg sync.WaitGroup

	// 複数のリーダーを作成
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				key := string(rune((i % 10) + 'a'))
				_, _ = c.Get(key)
			}
		}()
	}

	// 複数のライターを作成
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				key := string(rune((i % 10) + 'a'))
				c.Set(key, writerID*1000+i)
			}
		}(w)
	}

	// 削除オペレーションも同時に実行
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			if i%5 == 0 { // 一部のキーのみを削除
				key := string(rune((i % 10) + 'a'))
				c.Delete(key)
			}
		}
	}()

	// すべてのgoroutineが完了するのを待つ
	wg.Wait()
	done <- true

	<-done // テストが終了

	// ここでキャッシュの状態を確認することもできる
	// しかし、並行処理のテストでは最終状態は非決定的かもしれない
}

// モックキャッシュの実装（インターフェースに対するテスト用）
type MockCache[T any] struct {
	GetFunc    func(key string) (T, bool)
	SetFunc    func(key string, val T)
	DeleteFunc func(key string)
}

func (m *MockCache[T]) Get(key string) (T, bool) {
	return m.GetFunc(key)
}

func (m *MockCache[T]) Set(key string, val T) {
	m.SetFunc(key, val)
}

func (m *MockCache[T]) Delete(key string) {
	m.DeleteFunc(key)
}

func TestCacherInterface(t *testing.T) {
	// インターフェースを使用したテスト
	var c CacheInterface[string]

	// 実装をインターフェースに割り当て
	c = New[string]()
	c.Set("key", "value")
	val, ok := c.Get("key")
	if !ok || val != "value" {
		t.Errorf("Interface test failed")
	}

	// モックを使用
	mockCalled := false
	mock := &MockCache[string]{
		GetFunc: func(key string) (string, bool) {
			if key == "test" {
				return "mocked value", true
			}
			return "", false
		},
		SetFunc: func(key string, val string) {
			mockCalled = true
		},
		DeleteFunc: func(key string) {
			// 何もしない
		},
	}

	c = mock
	c.Set("any", "value") // モックを呼び出す

	if !mockCalled {
		t.Errorf("Mock Set was not called")
	}

	val, ok = c.Get("test")
	if !ok || val != "mocked value" {
		t.Errorf("Mock Get returned unexpected value: %v, %v", val, ok)
	}
}
