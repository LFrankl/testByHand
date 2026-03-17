// pool.go — 连接池骨架
//
// 功能要求：
//   - Get        从池中取出一个健康连接；若池为空且未达上限则新建；否则阻塞等待
//   - Put        归还连接；不健康的连接直接丢弃（不放回池）
//   - Close      关闭连接池：释放所有空闲连接，等待借出连接全部归还
//   - 空闲超时   连接空闲超过 idleTimeout 时，下次 Put 或 Get 时惰性淘汰
//   - 统计       提供 Stats()：池容量/当前空闲/当前借出 数量
//
// 设计方案（channel-based，无需条件变量）：
//
//	                  ┌──────────────────────────────────┐
//	  Get()  ←───────│  idle chan *Conn  (缓冲=maxSize)  │───→  Put()
//	  (借出)          └──────────────────────────────────┘       (归还)
//	                         ↑ newConn()
//	               当 idle 为空 且 total < maxSize 时新建
//
//	  Pool 关闭后 Get 返回 ErrPoolClosed，Put 直接丢弃。
//
// 运行测试：
//
//	go test ./... -race
package connpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPoolClosed = errors.New("connection pool is closed")
	ErrGetTimeout = errors.New("get connection timeout")
)

// Stats 连接池统计快照。
type Stats struct {
	MaxSize  int // 最大连接数
	Idle     int // 当前空闲连接数
	InUse    int // 当前借出连接数
	Total    int // 当前总连接数（Idle + InUse）
}

// Pool 是并发安全的连接池。
type Pool struct {
	idle        chan *Conn    // 空闲连接队列（带缓冲 channel）
	maxSize     int           // 最大连接数上限
	idleTimeout time.Duration // 连接空闲超时阈值
	total       int64         // 当前总连接数（atomic）
	closed      int32         // 1 = 已关闭（atomic）
	closeOnce   sync.Once
	closeCh     chan struct{}  // 关闭广播
	inUseDone   sync.WaitGroup // 追踪借出的连接（Put 时 Done）
}

// NewPool 创建连接池，初始不新建连接（懒加载）。
// TODO: 实现该函数
//
// 参数：
//   maxSize     最大连接数
//   idleTimeout 空闲超时（0 表示不超时）
func NewPool(maxSize int, idleTimeout time.Duration) *Pool {
	panic("TODO")
}

// Get 从池中获取一个连接，支持 context 超时/取消。
//
// 执行顺序：
//  1. 检查 pool 是否已关闭
//  2. 尝试从 idle 取一个现成连接（非阻塞 select + default）
//     → 取到后做健康检查和空闲超时检查；不健康/超时则丢弃，继续循环
//  3. idle 为空且 total < maxSize → 新建连接
//  4. idle 为空且 total >= maxSize → 阻塞等待，直到有连接归还或 ctx 超时
//
// TODO: 实现该方法
func (p *Pool) Get(ctx context.Context) (*Conn, error) {
	panic("TODO")
}

// Put 归还连接到池。
//
// 归还规则：
//   - pool 已关闭 → 丢弃
//   - 连接不健康  → 丢弃（total-1）
//   - 空闲超时    → 丢弃（total-1）
//   - 否则更新 lastUsed，放回 idle
//
// TODO: 实现该方法
func (p *Pool) Put(c *Conn) {
	panic("TODO")
}

// Close 关闭连接池：
//  1. 广播关闭信号（阻塞的 Get 立即返回 ErrPoolClosed）
//  2. 等待所有借出的连接归还（inUseDone.Wait）
//  3. 排空 idle channel，统计关闭的连接数
//
// TODO: 实现该方法
func (p *Pool) Close() {
	panic("TODO")
}

// Stats 返回当前连接池统计快照（并发安全）。
// TODO: 实现该方法
func (p *Pool) Stats() Stats {
	panic("TODO")
}

// isClosed 返回 pool 是否已关闭。
// TODO: 实现该方法（atomic.LoadInt32）
func (p *Pool) isClosed() bool {
	panic("TODO")
}

// isIdleTimeout 判断连接是否空闲超时。
// TODO: 实现该方法
func (p *Pool) isIdleTimeout(c *Conn) bool {
	panic("TODO")
}

// ─────────────────────────────────────────────
// 确保编译通过
// ─────────────────────────────────────────────
var _ = context.Background
var _ = sync.Once{}
var _ = atomic.LoadInt32
