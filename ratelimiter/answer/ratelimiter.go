// 答案版本：令牌桶 + 滑动窗口限流器完整实现
//
// 运行测试：go test ./... -race
package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════
//  令牌桶
// ══════════════════════════════════════════════════════

// TokenBucket 是基于惰性计算的令牌桶。
//
// 设计要点：
//   不使用后台 goroutine 定时补充，而是在每次 Allow/Wait 时
//   根据"距上次补充经过的时间"惰性计算当前令牌数。
//   这样实现更简单，且无需关闭操作。
type TokenBucket struct {
	rate       float64 // 每秒补充令牌数
	burst      float64 // 桶容量（上限）
	tokens     float64 // 当前令牌数
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶，初始令牌满桶。
func NewTokenBucket(rate, burst float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     burst, // 初始满桶
		lastRefill: time.Now(),
	}
}

// refill 惰性补充令牌（调用方须持锁）。
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastRefill = now
}

// Allow 非阻塞消耗 n 个令牌。
func (tb *TokenBucket) Allow(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens < n {
		return false
	}
	tb.tokens -= n
	return true
}

// Wait 阻塞等待直到可消耗 n 个令牌或 ctx 超时/取消。
//
// 等待时长计算：
//   need     = n - tokens          （还差多少令牌）
//   waitDur  = need / rate * second （以当前速率需要等多久）
//
// 注意：Wait 返回前再次 refill + 扣令牌，保证原子性。
func (tb *TokenBucket) Wait(ctx context.Context, n float64) error {
	tb.mu.Lock()
	tb.refill()
	need := n - tb.tokens
	var waitDur time.Duration
	if need > 0 {
		waitDur = time.Duration(need/tb.rate*1e9) * time.Nanosecond
	}
	tb.mu.Unlock()

	if waitDur <= 0 {
		tb.mu.Lock()
		tb.refill()
		tb.tokens -= n
		tb.mu.Unlock()
		return nil
	}

	select {
	case <-time.After(waitDur):
		tb.mu.Lock()
		tb.refill()
		tb.tokens -= n
		if tb.tokens < 0 { // 极小概率时钟精度问题，兜底
			tb.tokens = 0
		}
		tb.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ══════════════════════════════════════════════════════
//  滑动窗口
// ══════════════════════════════════════════════════════

// SlidingWindow 是基于时间戳队列的滑动窗口限流器。
//
// 设计要点：
//   维护一个升序时间戳切片，每次 Allow 时：
//     1. 用二分（或线性扫描）淘汰 now-windowSize 之前的记录
//     2. 判断剩余数量是否 < limit
//   全程持锁，保证并发安全。
//
//   时间复杂度：O(k)，k 为窗口内请求数。
//   空间复杂度：O(limit)，超出 limit 的请求被拒绝，不会追加。
type SlidingWindow struct {
	limit      int
	windowSize time.Duration
	timestamps []time.Time
	mu         sync.Mutex
}

// NewSlidingWindow 创建滑动窗口限流器。
func NewSlidingWindow(limit int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:      limit,
		windowSize: windowSize,
		timestamps: make([]time.Time, 0, limit),
	}
}

// Allow 检查并记录当前请求，返回是否放行。
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	// 淘汰窗口外的旧时间戳（timestamps 有序，找到第一个 >= cutoff 的位置）
	i := 0
	for i < len(sw.timestamps) && sw.timestamps[i].Before(cutoff) {
		i++
	}
	sw.timestamps = sw.timestamps[i:]

	if len(sw.timestamps) >= sw.limit {
		return false
	}
	sw.timestamps = append(sw.timestamps, now)
	return true
}

// Count 返回当前窗口内的有效请求数（同时做一次淘汰）。
func (sw *SlidingWindow) Count() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)
	i := 0
	for i < len(sw.timestamps) && sw.timestamps[i].Before(cutoff) {
		i++
	}
	sw.timestamps = sw.timestamps[i:]
	return len(sw.timestamps)
}
