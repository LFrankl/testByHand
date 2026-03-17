package mystring

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	s := New("hello")
	if s.String() != "hello" {
		t.Errorf("want hello, got %s", s.String())
	}
	if s.Len() != 5 {
		t.Errorf("want 5, got %d", s.Len())
	}
	if New("").Empty() == false {
		t.Error("empty string should be empty")
	}
}

func TestRuneLen(t *testing.T) {
	s := New("你好世界")
	if s.RuneLen() != 4 {
		t.Errorf("want 4 runes, got %d", s.RuneLen())
	}
	if s.Len() != 12 { // 每个汉字 3 字节 UTF-8
		t.Errorf("want 12 bytes, got %d", s.Len())
	}
}

func TestImmutability(t *testing.T) {
	s := New("hello")
	b := s.Bytes()
	b[0] = 'X'
	if s.String() != "hello" {
		t.Error("Bytes() should return a defensive copy, not affect original")
	}
}

func TestEqual(t *testing.T) {
	if !New("hello").Equal(New("hello")) {
		t.Error("equal strings should be equal")
	}
	if New("hello").Equal(New("world")) {
		t.Error("different strings should not be equal")
	}
	if !New("").Equal(New("")) {
		t.Error("empty strings should be equal")
	}
}

func TestConcat(t *testing.T) {
	a := New("foo")
	b := New("bar")
	c := a.Concat(b)
	if c.String() != "foobar" {
		t.Errorf("want foobar, got %s", c.String())
	}
	// 不可变性：原值不变
	if a.String() != "foo" {
		t.Error("a should not be modified after Concat")
	}
}

func TestSlice(t *testing.T) {
	s := New("hello world")
	if got := s.Slice(0, 5).String(); got != "hello" {
		t.Errorf("Slice(0,5): want hello, got %s", got)
	}
	if got := s.Slice(6, 11).String(); got != "world" {
		t.Errorf("Slice(6,11): want world, got %s", got)
	}
	if got := s.Slice(0, 0).String(); got != "" {
		t.Errorf("Slice(0,0): want empty, got %s", got)
	}
	// 越界自动截断
	if got := s.Slice(0, 100).String(); got != "hello world" {
		t.Errorf("Slice beyond length failed: %s", got)
	}
}

func TestIndex_KMP(t *testing.T) {
	cases := []struct {
		text, pattern string
		want          int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "xyz", -1},
		{"aabaab", "aab", 0},
		{"aabaab", "baab", 2},
		{"abc", "", 0},
		{"", "a", -1},
		{"abababab", "ababab", 0},
		{"aaaaaa", "aaa", 0},
		{"mississippi", "issip", 4},
		{"a", "a", 0},
		{"a", "b", -1},
	}
	for _, c := range cases {
		got := New(c.text).Index(New(c.pattern))
		if got != c.want {
			t.Errorf("Index(%q, %q) = %d, want %d", c.text, c.pattern, got, c.want)
		}
	}
}

func TestContains(t *testing.T) {
	s := New("hello world")
	if !s.Contains(New("world")) {
		t.Error("should contain 'world'")
	}
	if s.Contains(New("xyz")) {
		t.Error("should not contain 'xyz'")
	}
	if !s.Contains(New("")) {
		t.Error("should contain empty string")
	}
}

func TestReverse(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "olleh"},
		{"", ""},
		{"a", "a"},
		{"ab", "ba"},
	}
	for _, c := range cases {
		if got := New(c.in).Reverse().String(); got != c.want {
			t.Errorf("Reverse(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestReverseRunes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"你好", "好你"},
		{"hello", "olleh"},
		{"", ""},
		{"中文abc", "cba文中"},
	}
	for _, c := range cases {
		if got := New(c.in).ReverseRunes().String(); got != c.want {
			t.Errorf("ReverseRunes(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToUpperLower(t *testing.T) {
	if got := New("Hello World 123").ToUpper().String(); got != "HELLO WORLD 123" {
		t.Errorf("ToUpper: got %s", got)
	}
	if got := New("Hello World 123").ToLower().String(); got != "hello world 123" {
		t.Errorf("ToLower: got %s", got)
	}
}

func TestTrimSpace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  hello  ", "hello"},
		{"  ", ""},
		{"hello", "hello"},
		{"\t\nhello\r\n", "hello"},
		{"", ""},
	}
	for _, c := range cases {
		if got := New(c.in).TrimSpace().String(); got != c.want {
			t.Errorf("TrimSpace(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplit(t *testing.T) {
	parts := New("a,b,c").Split(New(","))
	if len(parts) != 3 {
		t.Fatalf("want 3 parts, got %d", len(parts))
	}
	for i, want := range []string{"a", "b", "c"} {
		if parts[i].String() != want {
			t.Errorf("parts[%d] = %q, want %q", i, parts[i].String(), want)
		}
	}

	// sep not found → single element
	parts2 := New("abc").Split(New(","))
	if len(parts2) != 1 || parts2[0].String() != "abc" {
		t.Errorf("Split no match: want [abc], got %v", parts2)
	}

	// trailing sep
	parts3 := New("a,b,").Split(New(","))
	if len(parts3) != 3 || parts3[2].String() != "" {
		t.Errorf("Split trailing sep failed: %v", parts3)
	}
}

func TestReplace(t *testing.T) {
	s := New("foo bar foo baz foo")
	if got := s.Replace(New("foo"), New("qux"), -1).String(); got != "qux bar qux baz qux" {
		t.Errorf("Replace all: %s", got)
	}
	if got := s.Replace(New("foo"), New("qux"), 1).String(); got != "qux bar foo baz foo" {
		t.Errorf("Replace n=1: %s", got)
	}
	if got := s.Replace(New("foo"), New("qux"), 0).String(); got != "foo bar foo baz foo" {
		t.Errorf("Replace n=0 should not replace: %s", got)
	}
}

func TestBuilder(t *testing.T) {
	var b Builder
	result := b.Write(New("hello")).WriteString(" ").Write(New("world")).Build()
	if result.String() != "hello world" {
		t.Errorf("Builder chain: got %s", result.String())
	}

	b.Reset()
	if b.Len() != 0 {
		t.Error("Reset should clear builder")
	}

	// Build doesn't clear
	b.WriteString("abc")
	b.Build()
	if b.Len() != 3 {
		t.Error("Build should not clear builder")
	}
}

// TestConcurrent 确保多个 goroutine 并发读取同一个 MyString 是安全的。
func TestConcurrent(t *testing.T) {
	s := New("concurrent test string")
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			_ = s.Contains(New("test"))
			_ = s.ToUpper()
			_ = s.Reverse()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}

// BenchmarkIndex_KMP vs 标准库
func BenchmarkIndexKMP(b *testing.B) {
	s := New(strings.Repeat("ab", 1000) + "abc")
	sub := New("abc")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Index(sub)
	}
}

func BenchmarkBuilderVsConcat(b *testing.B) {
	parts := make([]string, 100)
	for i := range parts {
		parts[i] = "hello"
	}

	b.Run("Builder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var builder Builder
			for _, p := range parts {
				builder.WriteString(p)
			}
			_ = builder.Build()
		}
	})

	b.Run("NaiveConcat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := New("")
			for _, p := range parts {
				s = s.Concat(New(p))
			}
			_ = s
		}
	})
}
