// 限时练习骨架：实现一个线程安全的 LRU（Least Recently Used）缓存
//
// 要求：
//   - Get(key)   O(1)
//   - Put(key, value) O(1)
//   - 容量满时淘汰最久未使用的条目
//   - 并发安全
//
// 数据结构选择：
//   - 双向链表：维护访问顺序（头部=最近使用，尾部=最久未用）
//   - 哈希表：O(1) 定位链表节点
//
// 运行测试：
//   go test ./...
package lru

import "sync"

// node 是双向链表的节点。
type node struct {
	key, val   int
	prev, next *node
}

// LRUCache 是线程安全的 LRU 缓存。
type LRUCache struct {
	cap        int
	mu         sync.Mutex
	cache      map[int]*node
	head, tail *node // 哨兵节点，不存储数据
}

// NewLRUCache 创建容量为 cap 的 LRU 缓存。
// TODO: 初始化哨兵头尾节点，并将它们相互连接。
func NewLRUCache(cap int) *LRUCache {
	panic("TODO")
}

// Get 返回 key 对应的值；若不存在返回 -1。
// 访问后该条目变为最近使用。
// TODO: 实现该方法
func (c *LRUCache) Get(key int) int {
	panic("TODO")
}

// Put 插入或更新 key-value。
// 若 key 已存在：更新值并移到头部。
// 若 key 不存在：插入到头部；若超出容量则删除尾部节点。
// TODO: 实现该方法
func (c *LRUCache) Put(key, val int) {
	panic("TODO")
}

// ---------- 链表操作（私有辅助方法）----------

// addToFront 将节点插入到哨兵头节点之后（即链表逻辑头部）。
// TODO: 实现该方法
func (c *LRUCache) addToFront(n *node) {
	panic("TODO")
}

// remove 将节点从链表中摘除（不释放内存）。
// TODO: 实现该方法
func (c *LRUCache) remove(n *node) {
	panic("TODO")
}

// removeLast 摘除并返回尾哨兵前的节点（最久未用）。
// TODO: 实现该方法
func (c *LRUCache) removeLast() *node {
	panic("TODO")
}

// moveToFront 将已在链表中的节点移到逻辑头部。
// 提示：先 remove 再 addToFront。
// TODO: 实现该方法
func (c *LRUCache) moveToFront(n *node) {
	panic("TODO")
}
