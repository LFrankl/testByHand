// router.go — Trie 路由树骨架
//
// 数据结构：前缀树（Trie），每个节点对应路径的一段（以 "/" 分隔）。
//
// 支持：
//   - 静态路径：/users/list
//   - 动态参数：/users/:id         （以 ":" 开头）
//   - 通配符：  /static/*filepath  （以 "*" 开头，必须在末尾）
//
// 匹配优先级：静态 > 动态参数 > 通配符
//
// 示例：
//   router.addRoute("GET", "/users/:id", handler)
//   router.getRoute("GET", "/users/42")
//   → node.path="/users/:id", params={"id":"42"}
package webframe

import "strings"

// node 是 Trie 的一个节点，对应路径中的一段。
type node struct {
	path     string  // 完整路由模式（仅叶子节点有值），如 "/users/:id"
	part     string  // 本节点的段，如 "users"、":id"、"*filepath"
	children []*node // 子节点列表
	isWild   bool    // 是否为模糊匹配节点（part 以 ":" 或 "*" 开头）
}

// matchChild 找到第一个能匹配 part 的子节点（用于插入）。
// 精确匹配优先，其次 isWild 节点。
// TODO: 实现该方法
func (n *node) matchChild(part string) *node {
	panic("TODO")
}

// matchChildren 找到所有能匹配 part 的子节点（用于查找，可能多个）。
// TODO: 实现该方法
func (n *node) matchChildren(part string) []*node {
	panic("TODO")
}

// insert 将路由模式递归插入 Trie。
// pattern 为完整路径，parts 为按 "/" 分割的各段，height 为当前深度。
// TODO: 实现该方法
func (n *node) insert(pattern string, parts []string, height int) {
	panic("TODO")
}

// search 在 Trie 中递归查找匹配的叶子节点。
// parts 为请求路径各段，height 为当前深度。
// TODO: 实现该方法
func (n *node) search(parts []string, height int) *node {
	panic("TODO")
}

// ---------- Router ----------

// router 按 HTTP Method 分别维护一棵 Trie。
type router struct {
	roots    map[string]*node        // method → 根节点
	handlers map[string][]HandlerFunc // "METHOD-pattern" → handlers
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string][]HandlerFunc),
	}
}

// parsePath 将路径按 "/" 分割为各段，忽略空串。
// "/users/:id/posts" → ["users", ":id", "posts"]
func parsePath(path string) []string {
	vs := strings.Split(path, "/")
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		if v != "" {
			parts = append(parts, v)
			if v[0] == '*' { // 通配符必须在末尾
				break
			}
		}
	}
	return parts
}

// addRoute 注册路由。
// TODO: 实现该方法（在对应 method 的 Trie 中插入，并存储 handlers）
func (r *router) addRoute(method, path string, handlers []HandlerFunc) {
	panic("TODO")
}

// getRoute 查找路由，返回匹配的节点和解析出的动态参数。
// 未找到时返回 nil, nil。
// TODO: 实现该方法
func (r *router) getRoute(method, path string) (*node, map[string]string) {
	panic("TODO")
}

// handle 根据请求查找路由并执行 handler 链。
// TODO: 实现该方法（找到路由 → 注入 params → 执行 c.Next()）
func (r *router) handle(c *Context) {
	panic("TODO")
}

// -------- 确保编译通过 --------
var _ = strings.Split
