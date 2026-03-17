// 骨架版本：自定义 String 类
//
// 实现一个不可变的 MyString 类型，要求：
//   1. 内部以 []byte 存储，所有方法返回新实例（不可变）
//   2. 区分字节操作（Len/Slice/Reverse）和 rune 操作（RuneLen/ReverseRunes）
//   3. Index/Contains 用 KMP 算法实现（O(n+m)）
//   4. Builder 支持 O(n) 高效拼接（禁止内部用 += 或 Concat 循环）
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
// 提示：utf8.RuneCount(b []byte) int
func (s MyString) RuneLen() int {
	_ = utf8.RuneCount // 消除未使用 import 编译错误
	panic("TODO")
}

// Empty 返回是否为空字符串。
func (s MyString) Empty() bool {
	panic("TODO")
}

// Bytes 返回内部字节的防御性拷贝。
//
// 关键：直接返回 s.b 会破坏不可变性——调用方修改切片会影响内部状态。
// 提示：make + copy
func (s MyString) Bytes() []byte {
	panic("TODO")
}

// Equal 按字节比较两个 MyString 是否相等。
func (s MyString) Equal(other MyString) bool {
	panic("TODO")
}

// Concat 拼接，返回新 MyString。
//
// 实现要点：预分配 len(s)+len(other) 大小，一次写入，避免二次分配。
func (s MyString) Concat(other MyString) MyString {
	panic("TODO")
}

// Slice 返回 [start, end) 的子串（字节索引）。
//
// 实现要点：
//   - 边界 clamp（start<0 → 0，end>len → len）
//   - 做拷贝而非共享底层数组，保证不可变性
func (s MyString) Slice(start, end int) MyString {
	panic("TODO")
}

// ── KMP 搜索 ──────────────────────────────────────────────────────────────────

// Index 使用 KMP 算法返回 sub 在 s 中首次出现的字节偏移，未找到返回 -1。
//
// KMP 核心思想（相比暴力 O(n*m) 的改进）：
//   预处理 pattern，构建 failure function（部分匹配表）：
//     f[i] = pattern[0..i] 的最长公共真前缀与真后缀的长度
//   搜索时，失配则将 pattern 指针 j 回退到 f[j-1]，不回退文本指针 i。
//
// 时间复杂度：O(n+m)，空间复杂度：O(m)
//
// 实现步骤：
//   1. 特判：sub 为空返回 0，sub 比 s 长返回 -1
//   2. 调用 buildFailure(sub.b) 得到 f
//   3. 双指针遍历：i 走文本，j 走 pattern
//      - s.b[i] != sub.b[j] 且 j>0：j = f[j-1]（回退）
//      - s.b[i] == sub.b[j]：j++
//      - j == m：返回 i-m+1
func (s MyString) Index(sub MyString) int {
	panic("TODO")
}

// buildFailure 构建 KMP failure function（部分匹配表）。
//
// 算法（双指针 k, i）：
//   k=0，遍历 i=1..m-1：
//     while k>0 && p[i]!=p[k]: k = f[k-1]  // 利用已知结果回退
//     if p[i]==p[k]: k++
//     f[i] = k
//
// 示例：p="ababc" → f=[0,0,1,2,0]
func buildFailure(p []byte) []int {
	panic("TODO")
}

// Contains 报告 sub 是否是 s 的子串（复用 Index）。
func (s MyString) Contains(sub MyString) bool {
	panic("TODO")
}

// ── 变换操作 ──────────────────────────────────────────────────────────────────

// Reverse 按字节反转（适合纯 ASCII）。
//
// 陷阱：对多字节 UTF-8 字符直接字节反转会产生非法序列。
// 提示：创建等长 buf，倒序赋值。
func (s MyString) Reverse() MyString {
	panic("TODO")
}

// ReverseRunes 按 Unicode rune 反转（UTF-8 安全）。
//
// 提示：
//   1. []rune(string(s.b)) 解码 UTF-8
//   2. 双指针交换
//   3. []byte(string(runes)) 重新编码
func (s MyString) ReverseRunes() MyString {
	panic("TODO")
}

// ToUpper 将所有 ASCII 小写字母转为大写。
//
// 提示：ASCII 大小写差值为 32（'a' - 'A' == 32），减 32 即转大写。
func (s MyString) ToUpper() MyString {
	panic("TODO")
}

// ToLower 将所有 ASCII 大写字母转为小写。
func (s MyString) ToLower() MyString {
	panic("TODO")
}

// TrimSpace 去除首尾空白字符（' '、'\t'、'\n'、'\r'）。
//
// 提示：双指针 l, r 从两端向中间收缩，找到第一个非空白位置后调用 Slice。
func (s MyString) TrimSpace() MyString {
	panic("TODO")
}

// isSpace 判断字节是否为空白字符（已实现，供 TrimSpace 使用）。
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Split 按 sep 分割，返回子串切片。
//
// 行为与 strings.Split 一致：
//   - sep 为空：按字节逐个分割
//   - 否则：循环调用 Index 找下一个 sep，收集左侧部分
//
// 陷阱：循环结束后，剩余部分 src[start:] 也要追加到结果中。
func (s MyString) Split(sep MyString) []MyString {
	panic("TODO")
}

// Replace 将前 n 个 old 替换为 newStr（n<0 表示全部替换）。
//
// 实现要点：用 Builder 累积结果，循环用 Index 定位下一个 old。
// n==0 或 old==newStr 时直接返回原值。
func (s MyString) Replace(old, newStr MyString, n int) MyString {
	panic("TODO")
}

// ── Builder ───────────────────────────────────────────────────────────────────

// Builder 使用 []byte 高效拼接字符串，支持链式调用。
//
// 为何不用 string +=：
//   每次 += 都分配新字节数组并复制，n 次拼接总开销是 O(n²)。
//   Builder 用 append 摊销分配，总开销 O(n)。
type Builder struct {
	buf []byte
}

// Write 追加一个 MyString，返回自身（链式调用）。
func (b *Builder) Write(s MyString) *Builder {
	panic("TODO")
}

// WriteString 追加一个 Go string。
func (b *Builder) WriteString(s string) *Builder {
	panic("TODO")
}

// WriteByte 追加一个字节。
func (b *Builder) WriteByte(c byte) *Builder {
	panic("TODO")
}

// writeRaw 内部使用（已实现），供 Replace 等方法直接追加字节切片。
func (b *Builder) writeRaw(p []byte) {
	b.buf = append(b.buf, p...)
}

// Len 返回当前已累积的字节数。
func (b *Builder) Len() int { return len(b.buf) }

// Reset 清空内容，复用底层数组（避免重新分配）。
func (b *Builder) Reset() { b.buf = b.buf[:0] }

// Build 返回当前内容的 MyString（做防御性拷贝），不清空 Builder。
//
// 关键：必须拷贝，否则 Builder 后续 Reset/Write 会影响已返回的 MyString。
func (b *Builder) Build() MyString {
	panic("TODO")
}
