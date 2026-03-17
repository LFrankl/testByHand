// context.go — 请求上下文完整实现
package webframe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// HandlerFunc 是框架统一的处理函数类型。
type HandlerFunc func(*Context)

// Context 封装单次 HTTP 请求的所有状态。
//
// 设计要点：
//   - handlers 切片按顺序存放中间件 + 最终 handler
//   - index 从 -1 开始，Next() 每次递增后执行对应 handler
//   - Abort() 将 index 直接跳到末尾，后续 handler 不再执行
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	Params map[string]string

	handlers []HandlerFunc
	index    int

	mu   sync.RWMutex
	keys map[string]any
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer:  w,
		Request: r,
		Params:  make(map[string]string),
		index:   -1,
	}
}

// ---------- 中间件控制 ----------

// Next 推进到下一个 handler 并执行。
// 中间件模板：
//
//	func MW() HandlerFunc {
//	    return func(c *Context) {
//	        // 前置逻辑
//	        c.Next()
//	        // 后置逻辑（此时下游已执行完毕）
//	    }
//	}
func (c *Context) Next() {
	c.index++
	for ; c.index < len(c.handlers); c.index++ {
		c.handlers[c.index](c)
	}
}

// Abort 终止中间件链。
// 已执行的中间件中 Next() 之后的代码仍会继续（defer 安全）。
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

// ---------- 请求读取 ----------

func (c *Context) Param(key string) string {
	return c.Params[key]
}

func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *Context) PostForm(key string) string {
	return c.Request.FormValue(key)
}

// ---------- KV 存储 ----------

func (c *Context) Set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys == nil {
		c.keys = make(map[string]any)
	}
	c.keys[key] = val
}

func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.keys[key]
	return v, ok
}

// ---------- 响应写入 ----------

func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key, val string) {
	c.Writer.Header().Set(key, val)
}

func (c *Context) String(code int, format string, values ...any) {
	c.SetHeader("Content-Type", "text/plain; charset=utf-8")
	c.Status(code)
	fmt.Fprintf(c.Writer, format, values...)
}

func (c *Context) JSON(code int, obj any) {
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Status(code)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

func (c *Context) Data(code int, contentType string, data []byte) {
	c.SetHeader("Content-Type", contentType)
	c.Status(code)
	_, _ = c.Writer.Write(data)
}

func (c *Context) Fail(code int, msg string) {
	c.JSON(code, map[string]string{"error": msg})
}
