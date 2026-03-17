package connpool

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ─── Conn ─────────────────────────────────────────────

func TestConnDo(t *testing.T) {
	c := newConn()
	if !c.IsHealthy() {
		t.Fatal("new conn should be healthy")
	}
	res, err := c.Do("SELECT 1")
	if err != nil {
		t.Fatalf("Do on healthy conn: %v", err)
	}
	if res == "" {
		t.Fatal("Do should return non-empty result")
	}

	c.MarkUnhealthy()
	if c.IsHealthy() {
		t.Fatal("conn should be unhealthy after MarkUnhealthy")
	}
	_, err = c.Do("SELECT 1")
	if err == nil {
		t.Fatal("Do on unhealthy conn should return error")
	}
}

// ─── Pool 基本 ────────────────────────────────────────

func TestGetPut(t *testing.T) {
	p := NewPool(3, 0)
	defer p.Close()

	c, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c == nil {
		t.Fatal("Get returned nil conn")
	}
	p.Put(c)

	stats := p.Stats()
	if stats.InUse != 0 {
		t.Fatalf("after Put, InUse should be 0, got %d", stats.InUse)
	}
	if stats.Idle != 1 {
		t.Fatalf("after Put, Idle should be 1, got %d", stats.Idle)
	}
}

func TestMaxSize(t *testing.T) {
	const maxSize = 3
	p := NewPool(maxSize, 0)
	defer p.Close()

	conns := make([]*Conn, maxSize)
	for i := 0; i < maxSize; i++ {
		c, err := p.Get(context.Background())
		if err != nil {
			t.Fatalf("Get[%d]: %v", i, err)
		}
		conns[i] = c
	}

	// 超出上限：应超时
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := p.Get(ctx)
	if err != ErrGetTimeout && err != context.DeadlineExceeded {
		t.Fatalf("want timeout error, got %v", err)
	}

	// 归还后可再取
	p.Put(conns[0])
	c, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get after Put: %v", err)
	}
	p.Put(c)

	for _, c := range conns[1:] {
		p.Put(c)
	}
}

// ─── 并发安全 ─────────────────────────────────────────

func TestConcurrent(t *testing.T) {
	const (
		maxSize    = 5
		goroutines = 50
		rounds     = 20
	)
	p := NewPool(maxSize, 0)
	defer p.Close()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				c, err := p.Get(context.Background())
				if err != nil {
					return
				}
				c.Do("query") //nolint
				time.Sleep(time.Millisecond)
				p.Put(c)
			}
		}()
	}
	wg.Wait()

	stats := p.Stats()
	if stats.InUse != 0 {
		t.Fatalf("after all Put, InUse should be 0, got %d", stats.InUse)
	}
	if stats.Total > maxSize {
		t.Fatalf("total=%d exceeded maxSize=%d", stats.Total, maxSize)
	}
}

// ─── 不健康连接丢弃 ───────────────────────────────────

func TestUnhealthyConnDiscarded(t *testing.T) {
	p := NewPool(2, 0)
	defer p.Close()

	c, _ := p.Get(context.Background())
	id := c.id
	c.MarkUnhealthy()
	p.Put(c) // 不健康，应丢弃

	// total 应减少
	stats := p.Stats()
	if stats.Idle != 0 {
		t.Fatalf("unhealthy conn should not be in idle, got idle=%d", stats.Idle)
	}

	// 再 Get 应能新建连接（不复用被丢弃的）
	c2, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get after discard: %v", err)
	}
	if c2.id == id {
		t.Fatal("should not reuse unhealthy conn")
	}
	p.Put(c2)
}

// ─── 空闲超时 ─────────────────────────────────────────

func TestIdleTimeout(t *testing.T) {
	p := NewPool(2, 50*time.Millisecond)
	defer p.Close()

	c, _ := p.Get(context.Background())
	p.Put(c)

	// 等超时
	time.Sleep(80 * time.Millisecond)

	// Get 应丢弃超时连接并新建
	c2, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get after idle timeout: %v", err)
	}
	if c2.id == c.id {
		t.Fatal("should not reuse idle-timeout conn")
	}
	p.Put(c2)
}

// ─── Close ────────────────────────────────────────────

func TestClose(t *testing.T) {
	p := NewPool(3, 0)

	c, _ := p.Get(context.Background())

	// 另一个 goroutine 持有连接，稍后归还
	go func() {
		time.Sleep(30 * time.Millisecond)
		p.Put(c)
	}()

	p.Close() // 应等待 c 归还后返回

	// Close 后 Get 应报错
	_, err := p.Get(context.Background())
	if err != ErrPoolClosed {
		t.Fatalf("after Close, Get should return ErrPoolClosed, got %v", err)
	}

	// 幂等
	p.Close()
}
