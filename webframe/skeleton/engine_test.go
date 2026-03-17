package webframe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- 工具 ----------

func doRequest(e *Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w
}

// ---------- 测试用例 ----------

func TestStaticRoute(t *testing.T) {
	e := New()
	e.GET("/ping", func(c *Context) { c.String(200, "pong") })

	w := doRequest(e, "GET", "/ping")
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "pong" {
		t.Fatalf("want pong, got %q", w.Body.String())
	}
}

func TestDynamicParam(t *testing.T) {
	e := New()
	e.GET("/users/:id", func(c *Context) {
		c.String(200, "user=%s", c.Param("id"))
	})

	w := doRequest(e, "GET", "/users/42")
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "user=42" {
		t.Fatalf("want user=42, got %q", w.Body.String())
	}
}

func TestWildcard(t *testing.T) {
	e := New()
	e.GET("/static/*filepath", func(c *Context) {
		c.String(200, "file=%s", c.Param("filepath"))
	})

	w := doRequest(e, "GET", "/static/css/style.css")
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "css/style.css") {
		t.Fatalf("want filepath, got %q", w.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	e := New()
	e.GET("/exists", func(c *Context) { c.String(200, "ok") })

	w := doRequest(e, "GET", "/not-exists")
	if w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestJSONResponse(t *testing.T) {
	e := New()
	e.GET("/info", func(c *Context) {
		c.JSON(200, map[string]string{"name": "webframe"})
	})

	w := doRequest(e, "GET", "/info")
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if result["name"] != "webframe" {
		t.Fatalf("want webframe, got %s", result["name"])
	}
}

func TestMiddlewareOrder(t *testing.T) {
	e := New()
	var order []string

	e.Use(func(c *Context) {
		order = append(order, "mw1-before")
		c.Next()
		order = append(order, "mw1-after")
	})
	e.Use(func(c *Context) {
		order = append(order, "mw2-before")
		c.Next()
		order = append(order, "mw2-after")
	})
	e.GET("/order", func(c *Context) {
		order = append(order, "handler")
		c.String(200, "ok")
	})

	doRequest(e, "GET", "/order")

	want := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("middleware order wrong\ngot:  %v\nwant: %v", order, want)
	}
}

func TestGroupMiddleware(t *testing.T) {
	e := New()
	var log []string

	e.Use(func(c *Context) {
		log = append(log, "global")
		c.Next()
	})

	api := e.Group("/api")
	api.Use(func(c *Context) {
		log = append(log, "api-mw")
		c.Next()
	})
	api.GET("/hello", func(c *Context) {
		log = append(log, "handler")
		c.String(200, "hi")
	})

	// 全局路由：只触发 global 中间件
	e.GET("/ping", func(c *Context) {
		log = append(log, "ping")
		c.String(200, "pong")
	})

	log = nil
	doRequest(e, "GET", "/api/hello")
	if fmt.Sprint(log) != fmt.Sprint([]string{"global", "api-mw", "handler"}) {
		t.Fatalf("group middleware order wrong: %v", log)
	}

	log = nil
	doRequest(e, "GET", "/ping")
	if fmt.Sprint(log) != fmt.Sprint([]string{"global", "ping"}) {
		t.Fatalf("global-only middleware wrong: %v", log)
	}
}

func TestAbort(t *testing.T) {
	e := New()
	e.Use(func(c *Context) {
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		c.Abort() // 不再执行后续 handler
	})
	e.GET("/secret", func(c *Context) {
		c.String(200, "secret data") // 不应被执行
	})

	w := doRequest(e, "GET", "/secret")
	if w.Code != 401 {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestContextKV(t *testing.T) {
	e := New()
	e.Use(func(c *Context) {
		c.Set("user", "alice")
		c.Next()
	})
	e.GET("/me", func(c *Context) {
		user, ok := c.Get("user")
		if !ok {
			c.String(500, "no user")
			return
		}
		c.String(200, "%s", user.(string))
	})

	w := doRequest(e, "GET", "/me")
	if strings.TrimSpace(w.Body.String()) != "alice" {
		t.Fatalf("want alice, got %q", w.Body.String())
	}
}

func TestRecovery(t *testing.T) {
	e := New()
	e.Use(Recovery())
	e.GET("/panic", func(c *Context) {
		panic("something went wrong")
	})

	w := doRequest(e, "GET", "/panic")
	if w.Code != 500 {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
