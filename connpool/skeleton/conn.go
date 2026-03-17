// conn.go — 模拟连接骨架
//
// Conn 模拟一个"外部连接"（如数据库连接），具备：
//   - 唯一 ID
//   - 是否健康（模拟连接断开）
//   - 创建时间（用于空闲超时淘汰）
//   - 最后使用时间
package connpool

import (
	"fmt"
	"sync/atomic"
	"time"
)

var connIDGen uint64 // 全局连接 ID 生成器

// Conn 是一个模拟连接。
type Conn struct {
	id        uint64
	healthy   bool
	createdAt time.Time
	lastUsed  time.Time
}

// newConn 创建一个新的模拟连接（始终健康）。
// TODO: 实现该函数
func newConn() *Conn {
	panic("TODO")
}

// IsHealthy 返回连接是否健康。
// TODO: 实现该方法
func (c *Conn) IsHealthy() bool {
	panic("TODO")
}

// MarkUnhealthy 将连接标记为不健康（模拟网络断开）。
// TODO: 实现该方法
func (c *Conn) MarkUnhealthy() {
	panic("TODO")
}

// Do 模拟在连接上执行一个操作。
// 不健康的连接执行任何操作均返回 error。
// TODO: 实现该方法
func (c *Conn) Do(query string) (string, error) {
	panic("TODO")
}

// String 方便打印调试。
func (c *Conn) String() string {
	return fmt.Sprintf("Conn#%d(healthy=%v)", c.id, c.healthy)
}

// ─────────────────────────────────────────────
// 确保编译通过
// ─────────────────────────────────────────────
var _ = atomic.AddUint64
var _ = time.Now
