// 答案版本：自定义 String 类完整实现
//
// 设计目标：
//   - 不可变（immutable）：所有方法返回新实例，内部字节不可被外部修改
//   - UTF-8 感知：区分字节操作和 rune 操作
//   - 高效拼接：Builder 用 []byte 避免 += 的 O(n²) 问题
//   - KMP 搜索：Index/Contains 时间复杂度 O(n+m)
//
// 运行测试：
//   go test ./... -race
package mystring

import "unicode/utf8"

// ── MyString ──────────────────────────────────────────────────────────────────

// MyString 是不可变字符串类型，内部以 []byte 存储。
// 所有返回字符串的方法均创建新实例，原值不变。
type MyString struct {
	b []byte
}

// New 从 Go string 创建 MyString。
func New(s string) MyString {
	return MyString{b: []byte(s)}
}

// String 实现 fmt.Stringer，返回 Go string。
func (s MyString) String() string { return string(s.b) }

// Len 返回字节长度（非 Unicode 字符数）。
func (s MyString) Len() int { return len(s.b) }

// RuneLen 返回 Unicode 字符数（UTF-8 感知）。
//
// 注意：对于多字节字符（如中文），RuneLen != Len。
func (s MyString) RuneLen() int { return utf8.RuneCount(s.b) }

// Empty 返回是否为空字符串。
func (s MyString) Empty() bool { return len(s.b) == 0 }

// Bytes 返回内部字节的防御性拷贝。
//
// 关键：直接返回 s.b 会破坏不可变性——调用方修改切片会影响内部状态。
func (s MyString) Bytes() []byte {
	cp := make([]byte, len(s.b))
	copy(cp, s.b)
	return cp
}

// Equal 按字节比较两个 MyString 是否相等。
func (s MyString) Equal(other MyString) bool {
	if len(s.b) != len(other.b) {
		return false
	}
	for i := range s.b {
		if s.b[i] != other.b[i] {
			return false
		}
	}
	return true
}

// Concat 拼接，返回新 MyString。
//
// 实现要点：预分配 len(s)+len(other) 大小，一次 copy，避免二次分配。
func (s MyString) Concat(other MyString) MyString {
	buf := make([]byte, len(s.b)+len(other.b))
	copy(buf, s.b)
	copy(buf[len(s.b):], other.b)
	return MyString{b: buf}
}

// Slice 返回 [start, end) 的子串（字节索引）。
//
// 做拷贝而非共享底层数组，保证不可变性。
func (s MyString) Slice(start, end int) MyString {
	if start < 0 {
		start = 0
	}
	if end > len(s.b) {
		end = len(s.b)
	}
	if start >= end {
		return MyString{}
	}
	cp := make([]byte, end-start)
	copy(cp, s.b[start:end])
	return MyString{b: cp}
}

// ── KMP 搜索 ──────────────────────────────────────────────────────────────────

// Index 使用 KMP 算法返回 sub 在 s 中首次出现的字节偏移，未找到返回 -1。
//
// KMP 核心思想（相比暴力 O(n*m) 的改进）：
//   预处理 pattern 得到 failure function，失配时不回退文本指针 i，
//   只将 pattern 指针 j 回退到 f[j-1]，跳过已确认不匹配的位置。
//
// 时间复杂度：O(n+m)，空间复杂度：O(m)
func (s MyString) Index(sub MyString) int {
	n, m := len(s.b), len(sub.b)
	if m == 0 {
		return 0
	}
	if m > n {
		return -1
	}

	f := buildFailure(sub.b)
	j := 0 // pattern 中的匹配位置
	for i := 0; i < n; i++ {
		// 失配：利用 failure function 回退 j（不回退 i）
		for j > 0 && s.b[i] != sub.b[j] {
			j = f[j-1]
		}
		if s.b[i] == sub.b[j] {
			j++
		}
		if j == m {
			return i - m + 1
		}
	}
	return -1
}

// buildFailure 构建 KMP failure function（部分匹配表）。
//
// f[i] = pattern[0..i] 中，最长公共真前缀与真后缀的长度。
// 例：pattern="ababc"，f=[0,0,1,2,0]
func buildFailure(p []byte) []int {
	m := len(p)
	f := make([]int, m)
	k := 0
	for i := 1; i < m; i++ {
		for k > 0 && p[i] != p[k] {
			k = f[k-1]
		}
		if p[i] == p[k] {
			k++
		}
		f[i] = k
	}
	return f
}

// Contains 报告 sub 是否是 s 的子串（复用 Index）。
func (s MyString) Contains(sub MyString) bool {
	return s.Index(sub) >= 0
}

// ── 变换操作 ──────────────────────────────────────────────────────────────────

// Reverse 按字节反转（适合纯 ASCII）。
//
// 陷阱：对多字节 UTF-8 字符直接字节反转会产生非法 UTF-8 序列。
// UTF-8 安全的反转请用 ReverseRunes。
func (s MyString) Reverse() MyString {
	n := len(s.b)
	buf := make([]byte, n)
	for i, c := range s.b {
		buf[n-1-i] = c
	}
	return MyString{b: buf}
}

// ReverseRunes 按 Unicode rune 反转（UTF-8 安全）。
//
// 先转 []rune（解码 UTF-8），双指针交换，再转回 []byte（重新编码）。
func (s MyString) ReverseRunes() MyString {
	runes := []rune(string(s.b))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return MyString{b: []byte(string(runes))}
}

// ToUpper 将所有 ASCII 小写字母转为大写。
//
// ASCII 大小写差值为 32：'a'(0x61) - 'A'(0x41) = 32。
func (s MyString) ToUpper() MyString {
	buf := make([]byte, len(s.b))
	for i, c := range s.b {
		if c >= 'a' && c <= 'z' {
			buf[i] = c - 32
		} else {
			buf[i] = c
		}
	}
	return MyString{b: buf}
}

// ToLower 将所有 ASCII 大写字母转为小写。
func (s MyString) ToLower() MyString {
	buf := make([]byte, len(s.b))
	for i, c := range s.b {
		if c >= 'A' && c <= 'Z' {
			buf[i] = c + 32
		} else {
			buf[i] = c
		}
	}
	return MyString{b: buf}
}

// TrimSpace 去除首尾空白字符（' '、'\t'、'\n'、'\r'）。
//
// 双指针从两端向中间收缩，找到第一个非空白位置。
func (s MyString) TrimSpace() MyString {
	l, r := 0, len(s.b)-1
	for l <= r && isSpace(s.b[l]) {
		l++
	}
	for r >= l && isSpace(s.b[r]) {
		r--
	}
	return s.Slice(l, r+1)
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Split 按 sep 分割，返回子串切片。
//
// 行为与 strings.Split 一致：
//   - sep 为空：按字节逐个分割
//   - 否则：循环用 Index 定位下一个 sep，收集两侧片段
func (s MyString) Split(sep MyString) []MyString {
	if sep.Len() == 0 {
		result := make([]MyString, len(s.b))
		for i, c := range s.b {
			result[i] = MyString{b: []byte{c}}
		}
		return result
	}

	var result []MyString
	start := 0
	for {
		idx := MyString{b: s.b[start:]}.Index(sep)
		if idx < 0 {
			break
		}
		result = append(result, s.Slice(start, start+idx))
		start += idx + sep.Len()
	}
	result = append(result, s.Slice(start, len(s.b)))
	return result
}

// Replace 将前 n 个 old 替换为 newStr（n<0 表示全部替换）。
//
// 用 Builder 累积结果，避免逐步拼接的 O(n²) 问题。
func (s MyString) Replace(old, newStr MyString, n int) MyString {
	if n == 0 || old.Equal(newStr) {
		return s
	}
	var b Builder
	src := s.b
	replaced := 0
	for {
		if n >= 0 && replaced >= n {
			break
		}
		idx := MyString{b: src}.Index(old)
		if idx < 0 {
			break
		}
		b.writeRaw(src[:idx])
		b.writeRaw(newStr.b)
		src = src[idx+old.Len():]
		replaced++
	}
	b.writeRaw(src)
	return b.Build()
}

// ── Builder ───────────────────────────────────────────────────────────────────

// Builder 使用 []byte 高效拼接字符串，支持链式调用。
//
// 为何不用 string +=：
//   每次 += 都分配新字节数组并复制，n 次拼接的总开销是 O(n²)。
//   Builder 用 append 摊销分配，总开销 O(n)。
type Builder struct {
	buf []byte
}

// Write 追加一个 MyString，返回自身（支持链式调用）。
func (b *Builder) Write(s MyString) *Builder {
	b.buf = append(b.buf, s.b...)
	return b
}

// WriteString 追加一个 Go string。
func (b *Builder) WriteString(s string) *Builder {
	b.buf = append(b.buf, s...)
	return b
}

// WriteByte 追加一个字节。
func (b *Builder) WriteByte(c byte) *Builder {
	b.buf = append(b.buf, c)
	return b
}

// writeRaw 内部使用，直接追加字节切片。
func (b *Builder) writeRaw(p []byte) {
	b.buf = append(b.buf, p...)
}

// Len 返回当前已累积的字节数。
func (b *Builder) Len() int { return len(b.buf) }

// Reset 清空内容，复用底层数组（避免重新分配）。
func (b *Builder) Reset() { b.buf = b.buf[:0] }

// Build 返回当前内容的 MyString（做防御性拷贝），不清空 Builder。
func (b *Builder) Build() MyString {
	cp := make([]byte, len(b.buf))
	copy(cp, b.buf)
	return MyString{b: cp}
}
