// engine.go — 框架入口骨架
//
// 层次结构：
//
//   Engine（顶层）
//   └── RouterGroup（路由分组，Engine 内嵌一个根 Group）
//       ├── 中间件列表
//       ├── 路由注册方法（GET/POST/…）
//       └── Group() 创建子分组
//
// 中间件执行顺序（洋葱模型）：
//
//   请求 → [MW1.前] → [MW2.前] → Handler → [MW2.后] → [MW1.后] → 响应
//
// 使用示例：
//
//   e := webframe.New()
//   e.Use(Logger(), Recovery())
//
//   api := e.Group("/api")
//   api.Use(Auth())
//   api.GET("/users/:id", getUser)
//
//   e.Run(":8080")
package webframe

import (
	"log"
	"net/http"
	"path"
)

// RouterGroup 代表一个路由分组，支持嵌套和中间件。
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine // 所有分组共享同一个 Engine
}

// Engine 是框架的核心，实现了 http.Handler 接口。
type Engine struct {
	*RouterGroup        // 内嵌根分组
	router  *router
	groups  []*RouterGroup // 所有已注册的分组
}

// New 创建一个新的 Engine。
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// ---------- 分组 ----------

// Group 基于当前分组创建子分组，子分组继承父分组的前缀。
// TODO: 实现该方法
func (g *RouterGroup) Group(prefix string) *RouterGroup {
	panic("TODO")
}

// Use 为当前分组注册中间件。
// TODO: 实现该方法
func (g *RouterGroup) Use(middlewares ...HandlerFunc) {
	panic("TODO")
}

// ---------- 路由注册 ----------

// addRoute 内部路由注册：合并分组前缀，收集所有适用中间件，存入 router。
// TODO: 实现该方法
//
// 提示：中间件的适用条件——当前请求路径以某个分组的 prefix 开头，
// 则该分组的 middlewares 都要加入 handler 链。
func (g *RouterGroup) addRoute(method, comp string, handler HandlerFunc) {
	panic("TODO")
}

func (g *RouterGroup) GET(path string, handler HandlerFunc) {
	g.addRoute("GET", path, handler)
}

func (g *RouterGroup) POST(path string, handler HandlerFunc) {
	g.addRoute("POST", path, handler)
}

func (g *RouterGroup) PUT(path string, handler HandlerFunc) {
	g.addRoute("PUT", path, handler)
}

func (g *RouterGroup) DELETE(path string, handler HandlerFunc) {
	g.addRoute("DELETE", path, handler)
}

// ---------- http.Handler 实现 ----------

// ServeHTTP 是框架的入口，每次请求都会调用此方法。
// TODO: 实现该方法（创建 Context → 委托给 router.handle）
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}

// Run 启动 HTTP 服务器。
func (e *Engine) Run(addr string) error {
	log.Printf("webframe listening on %s\n", addr)
	return http.ListenAndServe(addr, e)
}

// ---------- 内置中间件 ----------

// Logger 记录每次请求的方法、路径和耗时。
func Logger() HandlerFunc {
	return func(c *Context) {
		// TODO: 记录开始时间，调用 c.Next()，再计算耗时并打印
		panic("TODO")
	}
}

// Recovery 捕获 handler 中的 panic，返回 500。
func Recovery() HandlerFunc {
	return func(c *Context) {
		// TODO: 用 defer+recover 包裹 c.Next()
		panic("TODO")
	}
}

// -------- 确保编译通过 --------
var _ = path.Join
var _ = log.Println
var _ = http.ListenAndServe
