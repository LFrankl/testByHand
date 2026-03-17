package seckill

import (
	"bytes"
	"fmt"
	"net/http"
)

// fakeResponseWriter 是测试用的轻量 ResponseWriter，避免引入 net/http/httptest。
type fakeResponseWriter struct {
	Code    int
	Body    bytes.Buffer
	headers http.Header
}

func (f *fakeResponseWriter) Header() http.Header {
	if f.headers == nil {
		f.headers = make(http.Header)
	}
	return f.headers
}
func (f *fakeResponseWriter) WriteHeader(code int) { f.Code = code }
func (f *fakeResponseWriter) Write(b []byte) (int, error) {
	if f.Code == 0 {
		f.Code = 200
	}
	return f.Body.Write(b)
}

func newTestRequest(method, path string) (*fakeResponseWriter, *http.Request) {
	r, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(fmt.Sprintf("newTestRequest: %v", err))
	}
	return &fakeResponseWriter{}, r
}
