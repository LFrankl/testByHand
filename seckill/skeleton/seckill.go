// 限时练习骨架：简单秒杀服务
//
// 核心挑战：
//   1. 超卖（Overselling）— 并发扣库存时售出数量超过总库存
//   2. 重复购买（Dup-buy）— 同一用户抢购多次
//   3. 并发安全 — 高并发下数据一致性
//
// 设计方案（纯内存，无外部依赖）：
//   - sync.Mutex 保护临界区：查库存 + 扣库存 + 记录订单 必须原子执行
//   - sold map 实现用户级去重
//   - Order 记录每笔成功订单
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
	ErrOutOfStock   = errors.New("out of stock")
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
type SeckillService struct {
	mu    sync.Mutex
	stock int           // 剩余库存
	sold  map[int64]bool // userID → 是否已购买
	orders []Order
}

// -------- 构造 --------

// NewSeckillService 创建初始库存为 stock 的秒杀服务。
// TODO: 实现该函数
func NewSeckillService(stock int) *SeckillService {
	panic("TODO")
}

// -------- 核心逻辑 --------

// Seckill 为 userID 执行一次秒杀。
//
// 成功：返回订单 ID，error 为 nil。
// 失败：返回 "", ErrOutOfStock 或 ErrAlreadyBought。
//
// 注意事项：
//   - 查库存、扣库存、写订单必须在同一个临界区内完成，否则会产生超卖。
//   - 先判断重复购买，再判断库存（短路原则）。
//
// TODO: 实现该方法
func (s *SeckillService) Seckill(userID int64) (string, error) {
	panic("TODO")
}

// Stock 返回当前剩余库存（用于观测，无需加锁也可，这里加锁保证一致读）。
// TODO: 实现该方法
func (s *SeckillService) Stock() int {
	panic("TODO")
}

// Orders 返回所有成功订单的快照。
// TODO: 实现该方法
func (s *SeckillService) Orders() []Order {
	panic("TODO")
}

// -------- 工具函数（已提供）--------

// 全局序列号，用于生成唯一订单 ID。
var orderSeq uint64

// newOrderID 生成唯一订单 ID（格式：ORD-时间戳-序号）。
func newOrderID() string {
	seq := atomic.AddUint64(&orderSeq, 1)
	return fmt.Sprintf("ORD-%d-%d", time.Now().UnixNano(), seq)
}
