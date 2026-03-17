// engine.go — 框架入口完整实现
package webframe

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"runtime/debug"
	"strings"
	"time"
)

// RouterGroup 代表一个路由分组，支持嵌套和中间件。
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine
}

// Engine 是框架核心，实现 http.Handler。
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

// New 创建 Engine，内嵌一个根 RouterGroup。
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// ---------- 分组 ----------

// Group 创建子分组，继承父前缀。
//
// 关键点：子分组和父分组共享同一个 engine，
// 从而共享 router 和 groups 列表。
func (g *RouterGroup) Group(prefix string) *RouterGroup {
	child := &RouterGroup{
		prefix: g.prefix + prefix,
		engine: g.engine,
	}
	g.engine.groups = append(g.engine.groups, child)
	return child
}

// Use 为当前分组追加中间件。
func (g *RouterGroup) Use(middlewares ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// ---------- 路由注册 ----------

// addRoute 合并前缀，收集适用中间件，注册到 router。
//
// 中间件归属规则：遍历所有分组，若当前路径以某分组的 prefix 开头，
// 则该分组的中间件适用于本路由（前缀越长越精确，顺序由 groups 注册顺序保证）。
func (g *RouterGroup) addRoute(method, comp string, handler HandlerFunc) {
	fullPath := path.Join(g.prefix, comp)

	// 收集所有适用中间件（按 groups 注册顺序）
	var handlers []HandlerFunc
	for _, group := range g.engine.groups {
		if strings.HasPrefix(fullPath, group.prefix) {
			handlers = append(handlers, group.middlewares...)
		}
	}
	handlers = append(handlers, handler)

	g.engine.router.addRoute(method, fullPath, handlers)
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

// ---------- http.Handler ----------

// ServeHTTP 是框架入口：每个请求创建独立 Context，委托给 router.handle。
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, r)
	e.router.handle(c)
}

// Run 启动 HTTP 服务器。
func (e *Engine) Run(addr string) error {
	log.Printf("webframe listening on %s\n", addr)
	return http.ListenAndServe(addr, e)
}

// ---------- 内置中间件 ----------

// Logger 记录每次请求的方法、路径和耗时（洋葱模型后置统计）。
func Logger() HandlerFunc {
	return func(c *Context) {
		start := time.Now()
		c.Next()
		log.Printf("[%d] %s %s  %v",
			// 注意：WriteHeader 后 Code 已写入，这里用 ResponseWriter 无法直接读 Code
			// 生产框架会用自定义 ResponseWriter 记录 Code；此处简化
			200, c.Request.Method, c.Request.URL.Path, time.Since(start))
	}
}

// Recovery 捕获下游 panic，返回 500，防止进程崩溃。
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Recovery] panic: %v\n%s", err, debug.Stack())
				c.Fail(http.StatusInternalServerError, fmt.Sprintf("%v", err))
			}
		}()
		c.Next()
	}
}
