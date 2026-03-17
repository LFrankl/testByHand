// 答案版本：sync.Map 完整实现
//
// 核心设计——两张表 + miss 计数：
//
//   read  (atomic.Value)
//     ├─ 无锁读（fast path），存放绝大多数 key
//     └─ 内容可能落后于 dirty
//
//   dirty (map + Mutex)
//     ├─ 有锁写（slow path），包含 read 中没有的新 key
//     └─ 当 misses >= len(dirty) 时整体提升为 read
//
//   entry.p 的三种状态：
//     正常指针  — 条目存在
//     nil       — 已逻辑删除（在 dirty 中被 delete，但 read 中残留）
//     expunged  — 已从 read 删除且 dirty 中不存在（由 dirtyLocked 标记）
//
// 运行测试：
//   go test ./... -race
package syncmap

import (
	"sync"
	"sync/atomic"
)

// expunged 是一个哨兵值，标记"已从 read 删除且不存在于 dirty"的条目。
// 与 nil 区别：nil 表示"软删除，dirty 中也不保留"；expunged 仅在 dirtyLocked 中生成。
var expunged = new(any)

// ── entry ────────────────────────────────────────────────────────────────────

// entry 用原子指针封装 map 的一个值，支持无锁并发读写。
type entry struct {
	// p 的合法状态（见上方注释）
	p atomic.Pointer[any]
}

func newEntry(v any) *entry {
	e := &entry{}
	e.p.Store(&v)
	return e
}

// load 读取值；p==nil 或 p==expunged 时返回 (nil, false)。
func (e *entry) load() (any, bool) {
	p := e.p.Load()
	if p == nil || p == expunged {
		return nil, false
	}
	return *p, true
}

// tryStore 用 CAS 写入新值；若 p==expunged（该 key 不在 dirty 中）则拒绝，返回 false。
// 失败时调用方须走加锁路径（unexpungeLocked + dirtyLocked）。
func (e *entry) tryStore(v *any) bool {
	for {
		p := e.p.Load()
		if p == expunged {
			return false
		}
		if e.p.CompareAndSwap(p, v) {
			return true
		}
	}
}

// unexpungeLocked 尝试 CAS expunged → nil，表示"该 key 需要重新加入 dirty"。
// 必须在持 mu 时调用。返回 true 说明之前是 expunged 状态。
func (e *entry) unexpungeLocked() bool {
	return e.p.CompareAndSwap(expunged, nil)
}

// tryExpungeLocked 尝试 CAS nil → expunged（在 dirtyLocked 中标记软删除条目）。
// 必须在持 mu 时调用。
func (e *entry) tryExpungeLocked() (isExpunged bool) {
	p := e.p.Load()
	for p == nil {
		if e.p.CompareAndSwap(nil, expunged) {
			return true
		}
		p = e.p.Load()
	}
	return p == expunged
}

// delete 逻辑删除：CAS 当前值 → nil。
// 若已是 nil/expunged（已删除），返回 (nil, false)。
func (e *entry) delete() (value any, ok bool) {
	for {
		p := e.p.Load()
		if p == nil || p == expunged {
			return nil, false
		}
		if e.p.CompareAndSwap(p, nil) {
			return *p, true
		}
	}
}

// ── readOnly ─────────────────────────────────────────────────────────────────

// readOnly 是 read map 的快照，存储在 atomic.Value 中。
type readOnly struct {
	m       map[any]*entry
	amended bool // true 表示 dirty 中存在 read.m 里没有的 key
}

// ── Map ──────────────────────────────────────────────────────────────────────

// Map 是并发安全的键值映射，针对"读多写少"和"不同 goroutine 操作不同 key"场景优化。
//
// 关键设计决策：
//   - read 用 atomic.Value，Load 无需加锁（绝大多数读直接命中）
//   - dirty 持有全量数据，写操作加 mu
//   - misses 记录 read 未命中次数；达到 len(dirty) 时将 dirty 整体提升为 read，
//     代价 O(1)（仅交换指针），之后读又走 fast path
type Map struct {
	mu     sync.Mutex
	read   atomic.Value // 存储 readOnly
	dirty  map[any]*entry
	misses int
}

func (m *Map) loadRead() readOnly {
	if v := m.read.Load(); v != nil {
		return v.(readOnly)
	}
	return readOnly{}
}

// Load 返回 key 对应的值。
//
// fast path：read map 命中 → 无锁返回（绝大多数情况）
// slow path：read 未命中且 amended=true → 加锁查 dirty，并计 miss
func (m *Map) Load(key any) (value any, ok bool) {
	read := m.loadRead()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		// double-check：加锁期间 dirty 可能已提升为新 read
		read = m.loadRead()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if !ok {
		return nil, false
	}
	return e.load()
}

// Store 存储 key-value。
//
// fast path：key 已在 read 中且未 expunged → CAS 原子更新，无需加锁
// slow path：加锁后处理以下三种情况：
//   1. key 在 read 中（可能 expunged）：unexpunge → 写 entry
//   2. key 在 dirty 中：直接写 entry
//   3. key 完全不存在：若 dirty 未初始化先调 dirtyLocked，再插入新 entry
func (m *Map) Store(key, value any) {
	read := m.loadRead()
	if e, ok := read.m[key]; ok && e.tryStore(&value) {
		return // fast path
	}

	m.mu.Lock()
	read = m.loadRead()
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			// 之前是 expunged，需重新加入 dirty
			m.dirty[key] = e
		}
		e.p.Store(&value)
	} else if e, ok := m.dirty[key]; ok {
		e.p.Store(&value)
	} else {
		if !read.amended {
			// dirty 首次创建：拷贝 read 中存活的 entry，并把软删除 entry 标为 expunged
			m.dirtyLocked()
			m.read.Store(readOnly{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry(value)
	}
	m.mu.Unlock()
}

// LoadOrStore 若 key 存在则返回已有值（loaded=true），否则存储 value 并返回（loaded=false）。
func (m *Map) LoadOrStore(key, value any) (actual any, loaded bool) {
	// fast path：read 命中且值存在
	read := m.loadRead()
	if e, ok := read.m[key]; ok {
		if v, ok := e.load(); ok {
			return v, true
		}
	}

	m.mu.Lock()
	read = m.loadRead()
	if e, ok := read.m[key]; ok {
		if e.unexpungeLocked() {
			m.dirty[key] = e
		}
		if v, ok := e.load(); ok {
			m.mu.Unlock()
			return v, true
		}
		e.p.Store(&value)
	} else if e, ok := m.dirty[key]; ok {
		if v, ok := e.load(); ok {
			m.missLocked()
			m.mu.Unlock()
			return v, true
		}
		e.p.Store(&value)
		m.missLocked()
	} else {
		if !read.amended {
			m.dirtyLocked()
			m.read.Store(readOnly{m: read.m, amended: true})
		}
		m.dirty[key] = newEntry(value)
	}
	m.mu.Unlock()
	return value, false
}

// Delete 删除 key 对应的条目。
func (m *Map) Delete(key any) {
	m.LoadAndDelete(key)
}

// LoadAndDelete 删除 key 并返回其值（若存在）。
func (m *Map) LoadAndDelete(key any) (value any, loaded bool) {
	read := m.loadRead()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadRead()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			delete(m.dirty, key) // 直接从 dirty 中移除
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if ok {
		return e.delete() // 原子地将 entry.p 置 nil
	}
	return nil, false
}

// Range 对每个键值对调用 f，f 返回 false 时停止。
//
// 遍历策略：
//   若 dirty 有 read 中没有的 key（amended=true），先将 dirty 提升为 read，
//   然后遍历 read 快照。遍历期间不持锁，f 内可安全调用 Load/Store/Delete。
func (m *Map) Range(f func(key, value any) bool) {
	read := m.loadRead()
	if read.amended {
		// dirty 提升为 read（O(1)）
		m.mu.Lock()
		read = m.loadRead()
		if read.amended {
			read = readOnly{m: m.dirty}
			m.read.Store(read)
			m.dirty = nil
			m.misses = 0
		}
		m.mu.Unlock()
	}
	for k, e := range read.m {
		v, ok := e.load()
		if !ok {
			continue // 跳过已删除的条目
		}
		if !f(k, v) {
			break
		}
	}
}

// ── 内部辅助 ─────────────────────────────────────────────────────────────────

// missLocked 累计 read 未命中次数。
// 当 misses >= len(dirty) 时，将 dirty 整体提升为 read（代价 O(1)）。
// 必须在持 mu 时调用。
func (m *Map) missLocked() {
	m.misses++
	if m.misses < len(m.dirty) {
		return
	}
	// 提升：dirty 成为新 read，amended=false（dirty 为 nil 后无额外 key）
	m.read.Store(readOnly{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

// dirtyLocked 初始化 dirty map，将 read 中存活的 entry 拷贝进去，
// 并把软删除（p==nil）的 entry 标记为 expunged（不拷入 dirty）。
//
// 这样做的好处：Store 新 key 时只需操作 dirty；
// 若之后对 expunged key 再次 Store，通过 unexpungeLocked 重新加入 dirty。
//
// 必须在持 mu 时调用，且仅在 dirty==nil 时有效。
func (m *Map) dirtyLocked() {
	if m.dirty != nil {
		return
	}
	read := m.loadRead()
	m.dirty = make(map[any]*entry, len(read.m))
	for k, e := range read.m {
		if !e.tryExpungeLocked() {
			m.dirty[k] = e // 只拷贝未删除的 entry
		}
	}
}
