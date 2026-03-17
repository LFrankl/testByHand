// 限时练习骨架：令牌桶 + 滑动窗口限流器
//
// ══════════════════════════════════════════════════════
//  一、令牌桶（Token Bucket）
// ══════════════════════════════════════════════════════
//
//  原理：
//    桶容量 = burst，以固定速率 rate（个/秒）向桶中补充令牌。
//    请求到来时消耗 n 个令牌；桶中令牌不足时阻塞（Wait）或拒绝（Allow）。
//
//  关键公式（无需后台 goroutine，惰性计算）：
//    elapsed   = now - lastRefill
//    newTokens = elapsed * rate
//    tokens    = min(tokens + newTokens, burst)
//
//  接口：
//    Allow(n)       非阻塞：令牌够则消耗并返回 true，否则 false
//    Wait(ctx, n)   阻塞：等到令牌够或 ctx 超时
//
// ══════════════════════════════════════════════════════
//  二、滑动窗口（Sliding Window）
// ══════════════════════════════════════════════════════
//
//  原理：
//    维护一个时间窗口（size）内的请求时间戳队列。
//    每次请求到来时：
//      1. 淘汰窗口外的旧时间戳
//      2. 若窗口内请求数 < limit，记录本次时间戳并放行
//      3. 否则拒绝
//
//  接口：
//    Allow()        非阻塞：放行返回 true，否则 false
//
// 运行测试：
//   go test ./... -race
package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════
//  令牌桶
// ══════════════════════════════════════════════════════

// TokenBucket 是基于惰性计算的令牌桶限流器。
type TokenBucket struct {
	rate       float64   // 每秒补充令牌数
	burst      float64   // 桶容量（最大令牌数）
	tokens     float64   // 当前令牌数
	lastRefill time.Time // 上次补充时间
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶，初始令牌满桶。
// TODO: 实现该函数
func NewTokenBucket(rate, burst float64) *TokenBucket {
	panic("TODO")
}

// refill 惰性补充令牌（调用方须持锁）。
// 公式：elapsed = now - lastRefill; tokens = min(tokens + elapsed*rate, burst)
// TODO: 实现该方法
func (tb *TokenBucket) refill() {
	panic("TODO")
}

// Allow 非阻塞消耗 n 个令牌。令牌不足时返回 false。
// TODO: 实现该方法
func (tb *TokenBucket) Allow(n float64) bool {
	panic("TODO")
}

// Wait 阻塞等待直到可消耗 n 个令牌或 ctx 超时/取消。
// 等待时间 = (n - tokens) / rate（令牌不足时需等多久才能补满）。
// TODO: 实现该方法
//
// 提示：
//   1. 计算需要等待的时长
//   2. 若等待时长为 0 → 直接扣令牌返回
//   3. 否则 select 同时监听 time.After(waitDur) 和 ctx.Done()
func (tb *TokenBucket) Wait(ctx context.Context, n float64) error {
	panic("TODO")
}

// ══════════════════════════════════════════════════════
//  滑动窗口
// ══════════════════════════════════════════════════════

// SlidingWindow 是基于时间戳队列的滑动窗口限流器。
type SlidingWindow struct {
	limit      int           // 窗口内最大请求数
	windowSize time.Duration // 窗口时长
	timestamps []time.Time   // 窗口内的请求时间戳队列（升序）
	mu         sync.Mutex
}

// NewSlidingWindow 创建滑动窗口限流器。
// TODO: 实现该函数
func NewSlidingWindow(limit int, windowSize time.Duration) *SlidingWindow {
	panic("TODO")
}

// Allow 非阻塞检查是否放行当前请求。
//
// 步骤：
//  1. 淘汰 now-windowSize 之前的旧时间戳
//  2. 若 len(timestamps) < limit → 追加 now，返回 true
//  3. 否则返回 false
//
// TODO: 实现该方法
func (sw *SlidingWindow) Allow() bool {
	panic("TODO")
}

// Count 返回当前窗口内的请求数（用于观测）。
// TODO: 实现该方法
func (sw *SlidingWindow) Count() int {
	panic("TODO")
}

// ─────────────────────────────────────────────
// 确保编译通过
// ─────────────────────────────────────────────
var _ = context.Background
var _ = sync.Mutex{}
var _ = time.Now
