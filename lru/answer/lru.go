// 答案版本：线程安全的 LRU（Least Recently Used）缓存完整实现
//
// 数据结构：双向链表 + 哈希表
//   - 链表头部 = 最近使用；链表尾部 = 最久未用
//   - 哨兵头尾节点简化边界处理，避免空指针判断
//   - sync.Mutex 保证并发安全
//
// 所有操作均为 O(1)。
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
//
// 关键点：head <-> tail 构成空链表的初始状态。
// 后续所有真实节点插入在 head.next 和 tail.prev 之间。
func NewLRUCache(cap int) *LRUCache {
	head := &node{}
	tail := &node{}
	head.next = tail
	tail.prev = head

	return &LRUCache{
		cap:   cap,
		cache: make(map[int]*node, cap),
		head:  head,
		tail:  tail,
	}
}

// Get 返回 key 对应的值；若不存在返回 -1。
// 访问后将该节点移到链表头部（标记为最近使用）。
func (c *LRUCache) Get(key int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.cache[key]
	if !ok {
		return -1
	}
	c.moveToFront(n)
	return n.val
}

// Put 插入或更新 key-value。
//   - key 已存在：更新值并移到头部。
//   - key 不存在：新建节点插到头部；若超出容量则摘除并删除尾部节点。
func (c *LRUCache) Put(key, val int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if n, ok := c.cache[key]; ok {
		// 已存在：更新值，移到头部
		n.val = val
		c.moveToFront(n)
		return
	}

	// 不存在：新建节点
	n := &node{key: key, val: val}
	c.cache[key] = n
	c.addToFront(n)

	// 超出容量：淘汰最久未用（链表尾部）
	if len(c.cache) > c.cap {
		evicted := c.removeLast()
		delete(c.cache, evicted.key)
	}
}

// ---------- 链表操作（私有辅助方法）----------

// addToFront 将节点插入到哨兵头节点之后。
//
//	head <-> [n] <-> (原 head.next) <-> ... <-> tail
func (c *LRUCache) addToFront(n *node) {
	n.prev = c.head
	n.next = c.head.next
	c.head.next.prev = n
	c.head.next = n
}

// remove 将节点从链表中摘除（不删除节点本身，可复用）。
//
//	... <-> n.prev <-> [n] <-> n.next <-> ...
//	变为
//	... <-> n.prev <-> n.next <-> ...
func (c *LRUCache) remove(n *node) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

// removeLast 摘除并返回尾哨兵前的节点（最久未用）。
// 调用前须确保链表非空（即 head.next != tail）。
func (c *LRUCache) removeLast() *node {
	last := c.tail.prev
	c.remove(last)
	return last
}

// moveToFront 将已在链表中的节点移到逻辑头部。
func (c *LRUCache) moveToFront(n *node) {
	c.remove(n)
	c.addToFront(n)
}
