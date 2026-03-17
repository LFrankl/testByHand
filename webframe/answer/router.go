// router.go — Trie 路由树完整实现
package webframe

import (
	"fmt"
	"strings"
)

// node 是 Trie 的一个节点，对应路径中的一段。
type node struct {
	path     string  // 完整路由模式（仅叶子节点有值），如 "/users/:id"
	part     string  // 本节点的段，如 "users"、":id"、"*filepath"
	children []*node
	isWild   bool // part 以 ":" 或 "*" 开头时为 true
}

func (n *node) String() string {
	return fmt.Sprintf("node{path=%s, part=%s, isWild=%v}", n.path, n.part, n.isWild)
}

// matchChild 找到第一个能匹配 part 的子节点（插入时使用）。
// 精确匹配优先，其次 isWild 节点（保证静态路由不被参数路由覆盖）。
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// matchChildren 找到所有能匹配 part 的子节点（查找时使用，可能多条路径）。
func (n *node) matchChildren(part string) []*node {
	var nodes []*node
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// insert 将路由模式递归插入 Trie。
//
// 关键点：
//   - height == len(parts) 时到达叶子，记录 path
//   - 通配符节点（"*"）直接终止向下插入
func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.path = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{
			part:   part,
			isWild: part[0] == ':' || part[0] == '*',
		}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

// search 在 Trie 中递归查找匹配的叶子节点。
//
// 关键点：
//   - 通配符节点（part[0]=='*'）直接匹配剩余路径，终止递归
//   - 到达 parts 末尾时，只有 path 非空的节点才算命中
func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.path == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)
	for _, child := range children {
		if result := child.search(parts, height+1); result != nil {
			return result
		}
	}
	return nil
}

// ---------- Router ----------

type router struct {
	roots    map[string]*node
	handlers map[string][]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string][]HandlerFunc),
	}
}

// parsePath 将路径按 "/" 分割为各段，遇到通配符截止。
func parsePath(path string) []string {
	vs := strings.Split(path, "/")
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		if v != "" {
			parts = append(parts, v)
			if v[0] == '*' {
				break
			}
		}
	}
	return parts
}

// addRoute 在对应 method 的 Trie 中注册路由。
func (r *router) addRoute(method, path string, handlers []HandlerFunc) {
	parts := parsePath(path)
	key := method + "-" + path

	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(path, parts, 0)
	r.handlers[key] = handlers
}

// getRoute 查找路由，返回匹配节点和动态参数。
//
// 参数提取逻辑：
//   - ":id"      → params["id"] = 对应段的值
//   - "*filepath"→ params["filepath"] = 剩余路径拼接（含 "/"）
func (r *router) getRoute(method, path string) (*node, map[string]string) {
	searchParts := parsePath(path)
	params := make(map[string]string)

	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)
	if n == nil {
		return nil, nil
	}

	// 将模式路径各段与实际路径各段对齐，提取参数
	parts := parsePath(n.path)
	for idx, part := range parts {
		if part[0] == ':' {
			params[part[1:]] = searchParts[idx]
		}
		if part[0] == '*' && len(part) > 1 {
			// 通配符捕获剩余所有段
			params[part[1:]] = strings.Join(searchParts[idx:], "/")
			break
		}
	}
	return n, params
}

// handle 根据请求查找路由，注入参数，执行 handler 链。
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Request.Method, c.Request.URL.Path)
	if n == nil {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(404, "404 Not Found: %s\n", c.Request.URL.Path)
		})
	} else {
		c.Params = params
		key := c.Request.Method + "-" + n.path
		c.handlers = append(c.handlers, r.handlers[key]...)
	}
	c.Next()
}
