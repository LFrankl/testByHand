// context.go — 请求上下文骨架
//
// Context 贯穿整个请求生命周期：
//   - 封装 ResponseWriter / Request，提供便捷读写方法
//   - 持有路由参数（:id）和中间件链
//   - 通过 Next() / Abort() 控制中间件执行流
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
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	// 路由动态参数，如 /users/:id 中的 "id"
	Params map[string]string

	// 中间件链：所有 handler（中间件 + 最终处理函数）顺序存储
	handlers []HandlerFunc
	index    int // 当前执行到第几个 handler

	// 通用 KV 存储（在中间件间传递数据）
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

// Next 执行链中下一个 handler。
// 中间件在调用 Next() 前的代码为"前置逻辑"，之后为"后置逻辑"。
// TODO: 实现该方法（index 递增后循环调用 handlers）
func (c *Context) Next() {
	panic("TODO")
}

// Abort 终止中间件链，后续 handler 不再执行。
// TODO: 实现该方法（将 index 跳到末尾）
func (c *Context) Abort() {
	panic("TODO")
}

// ---------- 请求读取 ----------

// Param 返回路由动态参数，如 /users/:id → c.Param("id")。
// TODO: 实现该方法
func (c *Context) Param(key string) string {
	panic("TODO")
}

// Query 返回 URL 查询参数，如 ?page=2 → c.Query("page")。
// TODO: 实现该方法
func (c *Context) Query(key string) string {
	panic("TODO")
}

// PostForm 返回表单字段值。
// TODO: 实现该方法
func (c *Context) PostForm(key string) string {
	panic("TODO")
}

// ---------- KV 存储 ----------

// Set 在 Context 中存储键值对（线程安全）。
// TODO: 实现该方法
func (c *Context) Set(key string, val any) {
	panic("TODO")
}

// Get 从 Context 中读取键值对（线程安全）。
// TODO: 实现该方法
func (c *Context) Get(key string) (any, bool) {
	panic("TODO")
}

// ---------- 响应写入 ----------

// Status 写入 HTTP 状态码。
// TODO: 实现该方法
func (c *Context) Status(code int) {
	panic("TODO")
}

// SetHeader 设置响应头。
// TODO: 实现该方法
func (c *Context) SetHeader(key, val string) {
	panic("TODO")
}

// String 以纯文本格式响应。
// TODO: 实现该方法
func (c *Context) String(code int, format string, values ...any) {
	panic("TODO")
}

// JSON 以 JSON 格式响应。
// TODO: 实现该方法
func (c *Context) JSON(code int, obj any) {
	panic("TODO")
}

// Data 响应原始字节。
// TODO: 实现该方法
func (c *Context) Data(code int, contentType string, data []byte) {
	panic("TODO")
}

// Fail 响应错误信息（JSON）。
func (c *Context) Fail(code int, msg string) {
	c.JSON(code, map[string]string{"error": msg})
}

// -------- 确保编译通过（供骨架使用）--------
var _ = fmt.Sprintf
var _ = json.Marshal
var _ = sync.RWMutex{}
