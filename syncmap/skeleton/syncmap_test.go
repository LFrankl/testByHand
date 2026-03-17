package syncmap

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// ── 基础功能 ─────────────────────────────────────────────────────────────────

func TestStoreLoad(t *testing.T) {
	var m Map

	m.Store("k1", 1)
	m.Store("k2", "hello")

	if v, ok := m.Load("k1"); !ok || v != 1 {
		t.Errorf("Load k1: want (1, true), got (%v, %v)", v, ok)
	}
	if v, ok := m.Load("k2"); !ok || v != "hello" {
		t.Errorf("Load k2: want (hello, true), got (%v, %v)", v, ok)
	}
	if _, ok := m.Load("missing"); ok {
		t.Error("Load missing key should return ok=false")
	}
}

func TestOverwrite(t *testing.T) {
	var m Map
	m.Store("k", 1)
	m.Store("k", 2)
	if v, _ := m.Load("k"); v != 2 {
		t.Errorf("want 2, got %v", v)
	}
}

func TestDelete(t *testing.T) {
	var m Map
	m.Store("k", 1)
	m.Delete("k")
	if _, ok := m.Load("k"); ok {
		t.Error("key should be deleted")
	}
	// 删除不存在的 key 不应 panic
	m.Delete("nonexistent")
}

func TestLoadAndDelete(t *testing.T) {
	var m Map
	m.Store("k", 42)

	v, loaded := m.LoadAndDelete("k")
	if !loaded || v != 42 {
		t.Errorf("want (42, true), got (%v, %v)", v, loaded)
	}
	_, loaded = m.LoadAndDelete("k")
	if loaded {
		t.Error("second LoadAndDelete should return loaded=false")
	}
}

func TestLoadOrStore(t *testing.T) {
	var m Map

	// key 不存在：存储并返回 value
	v, loaded := m.LoadOrStore("k", 1)
	if loaded || v != 1 {
		t.Errorf("new key: want (1, false), got (%v, %v)", v, loaded)
	}

	// key 已存在：返回已有值
	v, loaded = m.LoadOrStore("k", 99)
	if !loaded || v != 1 {
		t.Errorf("existing key: want (1, true), got (%v, %v)", v, loaded)
	}
}

func TestRange(t *testing.T) {
	var m Map
	keys := []string{"a", "b", "c", "d"}
	for i, k := range keys {
		m.Store(k, i)
	}

	seen := make(map[string]int)
	m.Range(func(key, value any) bool {
		seen[key.(string)] = value.(int)
		return true
	})

	if len(seen) != len(keys) {
		t.Errorf("want %d entries, got %d", len(keys), len(seen))
	}
	for i, k := range keys {
		if seen[k] != i {
			t.Errorf("key %s: want %d, got %d", k, i, seen[k])
		}
	}
}

func TestRangeEarlyStop(t *testing.T) {
	var m Map
	for i := 0; i < 10; i++ {
		m.Store(i, i)
	}
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return count < 3 // 只遍历 3 个
	})
	if count != 3 {
		t.Errorf("want 3 iterations, got %d", count)
	}
}

// ── 删除后重新写入 ─────────────────────────────────────────────────────────────

// TestStoreAfterDelete 覆盖"expunged 路径"：先写、miss 提升 dirty、再删、再写。
func TestStoreAfterDelete(t *testing.T) {
	var m Map

	// 写入 key，触发多次 miss 让 dirty 提升为 read
	m.Store("k", 1)
	for i := 0; i < 20; i++ {
		m.Store(fmt.Sprintf("tmp%d", i), i)
		m.Load(fmt.Sprintf("tmp%d", i))
	}

	// 删除 key（此时 key 在 read 中，entry.p 变 nil）
	m.Delete("k")
	if _, ok := m.Load("k"); ok {
		t.Error("key should be deleted")
	}

	// 再次写入 key（走 unexpungeLocked 路径）
	m.Store("k", 999)
	v, ok := m.Load("k")
	if !ok || v != 999 {
		t.Errorf("after re-store: want (999, true), got (%v, %v)", v, ok)
	}
}

// ── 并发安全 ─────────────────────────────────────────────────────────────────

func TestConcurrentStoreLoad(t *testing.T) {
	var m Map
	const goroutines = 100
	const ops = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 并发写
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				m.Store(i*ops+j, j)
			}
		}()
	}
	// 并发读（可能读到 0 值，只要不 panic 即可）
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				m.Load(i*ops + j)
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentLoadOrStore(t *testing.T) {
	var m Map
	const goroutines = 200

	var storedCount atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, loaded := m.LoadOrStore("shared", 42)
			if !loaded {
				storedCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// 只有一个 goroutine 应该成功 store
	if storedCount.Load() != 1 {
		t.Errorf("want exactly 1 store, got %d", storedCount.Load())
	}
	if v, _ := m.Load("shared"); v != 42 {
		t.Errorf("want 42, got %v", v)
	}
}

func TestConcurrentDeleteStore(t *testing.T) {
	var m Map
	var wg sync.WaitGroup
	const goroutines = 50

	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.Store("k", j)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.Delete("k")
			}
		}()
	}
	wg.Wait()
	// 不检查最终值，只要没有 race 即通过
}

func TestConcurrentRange(t *testing.T) {
	var m Map
	for i := 0; i < 100; i++ {
		m.Store(i, i)
	}

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			m.Range(func(key, value any) bool {
				return true
			})
		}()
	}
	wg.Wait()
}

// ── miss 提升路径 ─────────────────────────────────────────────────────────────

// TestMissPromotion 验证在 dirty 中写入多个新 key 后，
// 重复查找这些 key 会触发 dirty→read 的提升。
func TestMissPromotion(t *testing.T) {
	var m Map

	// 写入足够多的 key（全部走 dirty 路径）
	for i := 0; i < 100; i++ {
		m.Store(i, i*10)
	}

	// 重复 Load，每次 read miss 都累积 misses；
	// 当 misses >= len(dirty) 时 dirty 被提升为 read
	for i := 0; i < 100; i++ {
		if v, ok := m.Load(i); !ok || v != i*10 {
			t.Errorf("Load(%d): want (%d, true), got (%v, %v)", i, i*10, v, ok)
		}
	}

	// 提升后再次读取，应走 fast path（read 命中）
	for i := 0; i < 100; i++ {
		if v, ok := m.Load(i); !ok || v != i*10 {
			t.Errorf("post-promotion Load(%d): want (%d, true)", i, i*10)
		}
	}
}

// BenchmarkLoad 对比 sync.Map 和本实现的 Load 性能
func BenchmarkLoad(b *testing.B) {
	var m Map
	m.Store("key", "value")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Load("key")
		}
	})
}

func BenchmarkStore(b *testing.B) {
	var m Map
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Store(i, i)
			i++
		}
	})
}
