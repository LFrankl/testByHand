package ratelimiter

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════
//  令牌桶测试
// ══════════════════════════════════════════════════════

func TestTokenBucketAllow(t *testing.T) {
	// burst=5，初始满桶，前 5 次 Allow 应成功
	tb := NewTokenBucket(1, 5)

	for i := 0; i < 5; i++ {
		if !tb.Allow(1) {
			t.Fatalf("Allow[%d] should succeed (burst=5)", i)
		}
	}
	// 第 6 次应失败（桶空）
	if tb.Allow(1) {
		t.Fatal("Allow should fail when bucket is empty")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	// rate=100/s，burst=5，掏空后等待补充
	tb := NewTokenBucket(100, 5)
	for i := 0; i < 5; i++ {
		tb.Allow(1)
	}

	// 等 60ms，应补充约 6 个令牌（超出 burst 截断为 5）
	time.Sleep(60 * time.Millisecond)
	if !tb.Allow(5) {
		t.Fatal("after refill, Allow(5) should succeed")
	}
}

func TestTokenBucketWait(t *testing.T) {
	// rate=100/s，burst=2，先掏空，Wait 应能等到令牌
	tb := NewTokenBucket(100, 2)
	tb.Allow(2)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := tb.Wait(ctx, 1); err != nil {
		t.Fatalf("Wait should succeed, got %v", err)
	}
}

func TestTokenBucketWaitTimeout(t *testing.T) {
	// rate=1/s（极慢），掏空后 Wait 应超时
	tb := NewTokenBucket(1, 1)
	tb.Allow(1)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := tb.Wait(ctx, 1); err == nil {
		t.Fatal("Wait should timeout")
	}
}

func TestTokenBucketConcurrent(t *testing.T) {
	const (
		rate      = 1000.0
		burst     = 100.0
		goroutines = 50
	)
	tb := NewTokenBucket(rate, burst)

	var passed int64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow(1) {
				atomic.AddInt64(&passed, 1)
			}
		}()
	}
	wg.Wait()

	// 初始满桶 burst=100，goroutines=50，全部应通过
	if passed != goroutines {
		t.Fatalf("want %d passed, got %d", goroutines, passed)
	}
}

// ══════════════════════════════════════════════════════
//  滑动窗口测试
// ══════════════════════════════════════════════════════

func TestSlidingWindowAllow(t *testing.T) {
	// limit=3，窗口=100ms
	sw := NewSlidingWindow(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		if !sw.Allow() {
			t.Fatalf("Allow[%d] should succeed (limit=3)", i)
		}
	}
	if sw.Allow() {
		t.Fatal("4th Allow should be rejected")
	}
}

func TestSlidingWindowSlide(t *testing.T) {
	sw := NewSlidingWindow(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		sw.Allow()
	}

	// 等窗口滑过
	time.Sleep(110 * time.Millisecond)

	// 旧请求已过期，窗口内应为 0
	if sw.Count() != 0 {
		t.Fatalf("after window slide, Count should be 0, got %d", sw.Count())
	}
	if !sw.Allow() {
		t.Fatal("Allow after slide should succeed")
	}
}

func TestSlidingWindowConcurrent(t *testing.T) {
	const (
		limit      = 10
		window     = 100 * time.Millisecond
		goroutines = 100
	)
	sw := NewSlidingWindow(limit, window)

	var passed int64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sw.Allow() {
				atomic.AddInt64(&passed, 1)
			}
		}()
	}
	wg.Wait()

	if passed > int64(limit) {
		t.Fatalf("passed=%d exceeded limit=%d (race condition!)", passed, limit)
	}
	if passed != int64(limit) {
		t.Fatalf("want exactly %d passed, got %d", limit, passed)
	}
}
