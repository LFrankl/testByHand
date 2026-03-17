// 答案版本：简单秒杀服务完整实现
//
// 核心思路：
//   - 单把 sync.Mutex 保护整个临界区（查重复 + 查库存 + 扣库存 + 写订单）
//   - 临界区内操作全为内存操作，持锁时间极短，足以应对面试场景
//   - sold map 实现 O(1) 用户去重
//
// 运行测试：
//   go test ./... -race
package seckill

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// -------- 错误定义 --------

var (
	ErrOutOfStock    = errors.New("out of stock")
	ErrAlreadyBought = errors.New("already bought")
)

// -------- 数据结构 --------

// Order 代表一笔成功的秒杀订单。
type Order struct {
	ID        string
	UserID    int64
	CreatedAt time.Time
}

// SeckillService 是秒杀服务的核心。
//
// 设计要点：
//   - mu 保护 stock、sold、orders 三者的一致性
//   - 先判重复购买（短路），再判库存，最后才扣减——顺序不能乱
type SeckillService struct {
	mu     sync.Mutex
	stock  int
	sold   map[int64]bool
	orders []Order
}

// -------- 构造 --------

// NewSeckillService 创建初始库存为 stock 的秒杀服务。
func NewSeckillService(stock int) *SeckillService {
	return &SeckillService{
		stock:  stock,
		sold:   make(map[int64]bool, stock),
		orders: make([]Order, 0, stock),
	}
}

// -------- 核心逻辑 --------

// Seckill 为 userID 执行一次秒杀。
//
// 成功：返回订单 ID，error 为 nil。
// 失败：返回 "", ErrOutOfStock 或 ErrAlreadyBought。
//
// 并发安全分析：
//   - 若将"查库存"和"扣库存"拆成两次加锁，两次锁之间库存可能被其他协程耗尽，导致超卖。
//   - 因此必须在同一把锁内完成：check → deduct → record。
func (s *SeckillService) Seckill(userID int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 去重：同一用户只能买一次
	if s.sold[userID] {
		return "", ErrAlreadyBought
	}

	// 2. 检查库存
	if s.stock <= 0 {
		return "", ErrOutOfStock
	}

	// 3. 扣库存 + 记录订单（原子操作，不可分割）
	s.stock--
	s.sold[userID] = true
	order := Order{
		ID:        newOrderID(),
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	s.orders = append(s.orders, order)

	return order.ID, nil
}

// Stock 返回当前剩余库存的一致性快照。
func (s *SeckillService) Stock() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stock
}

// Orders 返回所有成功订单的副本，避免外部持有内部切片引用。
func (s *SeckillService) Orders() []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]Order, len(s.orders))
	copy(cp, s.orders)
	return cp
}

// -------- 工具函数 --------

var orderSeq uint64

func newOrderID() string {
	seq := atomic.AddUint64(&orderSeq, 1)
	return fmt.Sprintf("ORD-%d-%d", time.Now().UnixNano(), seq)
}
