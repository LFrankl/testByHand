// conn.go — 模拟连接完整实现
package connpool

import (
	"fmt"
	"sync/atomic"
	"time"
)

var connIDGen uint64

// Conn 是一个模拟连接。
type Conn struct {
	id        uint64
	healthy   bool
	createdAt time.Time
	lastUsed  time.Time
}

func newConn() *Conn {
	return &Conn{
		id:        atomic.AddUint64(&connIDGen, 1),
		healthy:   true,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}
}

func (c *Conn) IsHealthy() bool { return c.healthy }

func (c *Conn) MarkUnhealthy() { c.healthy = false }

// Do 模拟在连接上执行操作，不健康时返回 error。
func (c *Conn) Do(query string) (string, error) {
	if !c.healthy {
		return "", fmt.Errorf("conn#%d is unhealthy", c.id)
	}
	c.lastUsed = time.Now()
	return fmt.Sprintf("conn#%d: ok(%s)", c.id, query), nil
}

func (c *Conn) String() string {
	return fmt.Sprintf("Conn#%d(healthy=%v)", c.id, c.healthy)
}
