// pool.go — 连接池完整实现
//
// 设计要点：
//
//  1. idle channel（带缓冲=maxSize）既是队列又是信号量：
//     len(idle) == 空闲数，cap(idle) == maxSize。
//     但 total 与 idle 是两个概念——total 含借出中的连接。
//
//  2. total 用 atomic int64，避免为读一个数字加整把锁。
//
//  3. Get 逻辑（无锁 CAS 控制新建）：
//       loop:
//         非阻塞取 idle → 健康且未超时 → 返回
//                       → 否则 total-- 丢弃，继续 loop
//         idle 为空 → CAS total++ → 若成功则 newConn
//                   → CAS 失败（已达上限）→ 阻塞 select 等 idle/closeCh/ctx
//
//  4. Put 直接写 idle channel（非阻塞 select + default 防止极端情况写满阻塞）。
//     若 idle 已满（极罕见：Close 后仍有连接归还），丢弃。
//
//  5. Close 步骤：
//       ① atomic 标记关闭 → ② close(closeCh) 唤醒所有阻塞的 Get
//       ③ inUseDone.Wait() 等借出连接全部归还 → ④ 排空 idle
//
// 运行测试：go test ./... -race
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

type Stats struct {
	MaxSize int
	Idle    int
	InUse   int
	Total   int
}

type Pool struct {
	idle        chan *Conn
	maxSize     int
	idleTimeout time.Duration
	total       int64        // atomic：当前总连接数（空闲 + 借出）
	closed      int32        // atomic：0=open 1=closed
	closeOnce   sync.Once
	closeCh     chan struct{}
	inUseDone   sync.WaitGroup // 每次 Get 成功时 Add(1)，Put 时 Done()
}

// NewPool 创建连接池，懒加载（初始不建连接）。
func NewPool(maxSize int, idleTimeout time.Duration) *Pool {
	return &Pool{
		idle:        make(chan *Conn, maxSize),
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		closeCh:     make(chan struct{}),
	}
}

// Get 从池中获取一个连接，支持 context 超时/取消。
func (p *Pool) Get(ctx context.Context) (*Conn, error) {
	for {
		if p.isClosed() {
			return nil, ErrPoolClosed
		}

		// 1. 先尝试从 idle 非阻塞取
		select {
		case c := <-p.idle:
			// 健康检查 + 空闲超时检查（惰性淘汰）
			if !c.IsHealthy() || p.isIdleTimeout(c) {
				atomic.AddInt64(&p.total, -1)
				continue // 丢弃，再试
			}
			p.inUseDone.Add(1)
			return c, nil
		default:
		}

		// 2. idle 为空：尝试 CAS 新建
		cur := atomic.LoadInt64(&p.total)
		if cur < int64(p.maxSize) {
			if atomic.CompareAndSwapInt64(&p.total, cur, cur+1) {
				c := newConn()
				p.inUseDone.Add(1)
				return c, nil
			}
			// CAS 被其他 goroutine 抢先，重试整个循环
			continue
		}

		// 3. 已达上限，阻塞等待
		select {
		case c := <-p.idle:
			if !c.IsHealthy() || p.isIdleTimeout(c) {
				atomic.AddInt64(&p.total, -1)
				continue
			}
			p.inUseDone.Add(1)
			return c, nil
		case <-p.closeCh:
			return nil, ErrPoolClosed
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, ErrGetTimeout
			}
			return nil, ctx.Err()
		}
	}
}

// Put 归还连接。不健康或超时的连接直接丢弃。
func (p *Pool) Put(c *Conn) {
	defer p.inUseDone.Done()

	if p.isClosed() || !c.IsHealthy() || p.isIdleTimeout(c) {
		atomic.AddInt64(&p.total, -1)
		return
	}

	c.lastUsed = time.Now()

	// 非阻塞写回 idle（防止 idle 已满时阻塞）
	select {
	case p.idle <- c:
	default:
		// idle 已满（不应发生，防御性丢弃）
		atomic.AddInt64(&p.total, -1)
	}
}

// Close 关闭连接池，等待所有借出连接归还后清空 idle。
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		atomic.StoreInt32(&p.closed, 1)
		close(p.closeCh)     // 唤醒所有阻塞的 Get
		p.inUseDone.Wait()   // 等待借出连接全部 Put 归还

		// 排空 idle channel
		for {
			select {
			case <-p.idle:
			default:
				return
			}
		}
	})
}

// Stats 返回当前统计快照。
func (p *Pool) Stats() Stats {
	idle := len(p.idle)
	total := int(atomic.LoadInt64(&p.total))
	return Stats{
		MaxSize: p.maxSize,
		Idle:    idle,
		InUse:   total - idle,
		Total:   total,
	}
}

func (p *Pool) isClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *Pool) isIdleTimeout(c *Conn) bool {
	if p.idleTimeout == 0 {
		return false
	}
	return time.Since(c.lastUsed) > p.idleTimeout
}
