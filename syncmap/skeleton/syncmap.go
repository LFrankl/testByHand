// 骨架版本：sync.Map
//
// 实现一个并发安全的 Map，核心思路与标准库 sync.Map 一致：
//
//   两张表：
//     read  (atomic.Value) — 无锁读，fast path
//     dirty (map + Mutex)  — 有锁写，slow path
//
//   entry.p 的三种状态：
//     正常指针  — 条目存在
//     nil       — 已逻辑删除（dirty 中不再保留）
//     expunged  — 已从 read 删除且 dirty 中不存在（由 dirtyLocked 标记）
//
//   提升机制：
//     每次 Load 走 dirty 路径计 miss；
//     当 misses >= len(dirty) 时，将 dirty 整体替换 read（O(1)）。
//
// 运行测试：
//   go test ./... -race
package syncmap

import (
	"sync"
	"sync/atomic"
)

// expunged 是哨兵值，标记"已从 read 删除且不在 dirty"的条目。
var expunged = new(any)

// ── entry ─────────────────────────────────────────────────────────────────────

// entry 用 atomic.Pointer 封装 map 中的一个值，支持无锁并发读。
type entry struct {
	p atomic.Pointer[any]
}

func newEntry(v any) *entry {
	e := &entry{}
	e.p.Store(&v)
	return e
}

// load 读取值。p==nil 或 p==expunged 时返回 (nil, false)。
// （已实现，供参考）
func (e *entry) load() (any, bool) {
	p := e.p.Load()
	if p == nil || p == expunged {
		return nil, false
	}
	return *p, true
}

// tryStore 尝试 CAS 写入新值；若 p==expunged 则拒绝，返回 false。
//
// 实现提示：
//   for 循环 + CompareAndSwap：
//     - p==expunged → return false（该 key 不在 dirty，无法直接更新）
//     - CAS 成功 → return true
//     - CAS 失败（被其他 goroutine 抢先）→ 重试
func (e *entry) tryStore(v *any) bool {
	panic("TODO")
}

// unexpungeLocked 尝试 CAS expunged → nil（表示该 key 需要重新加入 dirty）。
// 必须在持 mu 时调用。返回 true 表示之前是 expunged 状态。
func (e *entry) unexpungeLocked() bool {
	panic("TODO")
}

// tryExpungeLocked 尝试 CAS nil → expunged（在 dirtyLocked 中标记软删除条目）。
// 必须在持 mu 时调用。
func (e *entry) tryExpungeLocked() (isExpunged bool) {
	panic("TODO")
}

// delete 逻辑删除：CAS 当前值 → nil，不可分割。
// 已是 nil/expunged 则返回 (nil, false)。
func (e *entry) delete() (value any, ok bool) {
	panic("TODO")
}

// ── readOnly ──────────────────────────────────────────────────────────────────

// readOnly 是 read map 的快照，以 atomic.Value 存储。
type readOnly struct {
	m       map[any]*entry
	amended bool // true 表示 dirty 中有 read.m 里没有的 key
}

// ── Map ───────────────────────────────────────────────────────────────────────

// Map 是并发安全的键值映射，针对"读多写少"场景优化。
type Map struct {
	mu     sync.Mutex
	read   atomic.Value // 存储 readOnly
	dirty  map[any]*entry
	misses int
}

// loadRead 加载 read 快照（已实现）。
func (m *Map) loadRead() readOnly {
	if v := m.read.Load(); v != nil {
		return v.(readOnly)
	}
	return readOnly{}
}

// Load 返回 key 对应的值。
//
// 实现步骤：
//  1. loadRead()，在 read.m 中查 key
//  2. 命中 → 调 e.load() 返回
//  3. 未命中且 read.amended=true → 加锁，double-check，查 dirty，调 missLocked()
func (m *Map) Load(key any) (value any, ok bool) {
	panic("TODO")
}

// Store 存储 key-value。
//
// fast path：key 在 read 中且未 expunged → e.tryStore(&value) 直接 CAS，无需加锁。
//
// slow path（加锁后三分支）：
//   1. key 在 read.m 中（可能 expunged）：
//      - unexpungeLocked() 若成功 → 将 e 重新加入 dirty
//      - e.p.Store(&value)
//   2. key 在 dirty 中：e.p.Store(&value)
//   3. 全新 key：若 dirty==nil 先 dirtyLocked()，再 dirty[key]=newEntry(value)
func (m *Map) Store(key, value any) {
	panic("TODO")
}

// LoadOrStore 若 key 存在返回已有值（loaded=true），否则存储并返回 value（loaded=false）。
//
// 实现提示：
//   fast path 先查 read；slow path 加锁后同样需要 double-check，
//   三分支逻辑与 Store 类似，但在存储前先尝试 e.load()，若成功则返回已有值。
func (m *Map) LoadOrStore(key, value any) (actual any, loaded bool) {
	panic("TODO")
}

// Delete 删除 key。
func (m *Map) Delete(key any) {
	m.LoadAndDelete(key)
}

// LoadAndDelete 删除 key 并返回其值（若存在）。
//
// fast path：在 read.m 中找到 → e.delete() 原子置 nil。
// slow path：在 dirty 中找到 → delete(m.dirty, key)，再 e.delete()，调 missLocked()。
func (m *Map) LoadAndDelete(key any) (value any, loaded bool) {
	panic("TODO")
}

// Range 对每个键值对调用 f，f 返回 false 时停止。
//
// 实现提示：
//   若 read.amended=true，先加锁将 dirty 提升为 read（m.read.Store, dirty=nil, misses=0）。
//   然后遍历 read.m（无锁），跳过 e.load() 返回 ok=false 的条目。
func (m *Map) Range(f func(key, value any) bool) {
	panic("TODO")
}

// ── 内部辅助 ──────────────────────────────────────────────────────────────────

// missLocked 累计 miss，达到 len(dirty) 时将 dirty 提升为 read。
//
// 提升：m.read.Store(readOnly{m: m.dirty})，然后 dirty=nil, misses=0。
// 必须在持 mu 时调用。
func (m *Map) missLocked() {
	panic("TODO")
}

// dirtyLocked 初始化 dirty map（仅当 dirty==nil 时执行）：
//   遍历 read.m，对 entry 调 tryExpungeLocked()：
//     - 返回 true（已标记 expunged）→ 不拷入 dirty
//     - 返回 false（存活 entry）→ dirty[k] = e
//
// 必须在持 mu 时调用。
func (m *Map) dirtyLocked() {
	panic("TODO")
}
