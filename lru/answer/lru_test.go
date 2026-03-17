package lru

import (
	"sync"
	"testing"
)

func TestBasic(t *testing.T) {
	c := NewLRUCache(2)

	c.Put(1, 1)
	c.Put(2, 2)

	if got := c.Get(1); got != 1 {
		t.Fatalf("Get(1) = %d, want 1", got)
	}

	c.Put(3, 3) // 淘汰 key=2

	if got := c.Get(2); got != -1 {
		t.Fatalf("Get(2) = %d, want -1 (evicted)", got)
	}

	// Put(3) 后链表顺序：H<->3<->1<->T（3 最新，1 最旧）
	// Put(4) 淘汰尾部 key=1
	c.Put(4, 4)

	if got := c.Get(1); got != -1 {
		t.Fatalf("Get(1) = %d, want -1 (evicted)", got)
	}
	if got := c.Get(3); got != 3 {
		t.Fatalf("Get(3) = %d, want 3", got)
	}
	if got := c.Get(4); got != 4 {
		t.Fatalf("Get(4) = %d, want 4", got)
	}
}

func TestUpdateExisting(t *testing.T) {
	c := NewLRUCache(2)
	c.Put(1, 1)
	c.Put(2, 2)
	c.Put(1, 10) // 更新 key=1

	if got := c.Get(1); got != 10 {
		t.Fatalf("Get(1) = %d, want 10", got)
	}

	c.Put(3, 3) // 淘汰 key=2（key=1 刚刚更新，是最近使用）

	if got := c.Get(2); got != -1 {
		t.Fatalf("Get(2) = %d, want -1 (evicted)", got)
	}
}

func TestConcurrent(t *testing.T) {
	c := NewLRUCache(100)
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Put(i%50, i)
			c.Get(i % 50)
		}(i)
	}
	wg.Wait()
}
