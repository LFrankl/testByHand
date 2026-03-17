// 秒杀服务压测脚本
//
// 使用方法（先在另一个终端启动服务）：
//   go run ../answer/seckill.go ../answer/handler.go  ← 需要加 main，见下方说明
//
// 直接内嵌服务启动，无需单独跑服务端：
//   go run main.go
//   go run main.go -stock 100 -users 5000 -concurrency 200
//
// 输出示例：
//   === 压测参数 ===
//   库存: 100 | 并发用户: 5000 | 并发度: 200
//
//   === 压测结果 ===
//   总耗时:        312.45ms
//   总请求:        5000
//   成功(下单):    100
//   库存不足:      4843
//   重复购买:      57
//   其他错误:      0
//   QPS:           16001
//   延迟 p50:      1.23ms
//   延迟 p95:      3.45ms
//   延迟 p99:      5.67ms
//
//   === 正确性校验 ===
//   ✓ 无超卖（成功订单 100 == 库存 100）
//   ✓ 无重复用户订单
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// ========== 内嵌秒杀服务（与 answer 逻辑一致）==========

var (
	errOutOfStock    = errors.New("out of stock")
	errAlreadyBought = errors.New("already bought")
)

type order struct {
	ID     string
	UserID int64
}

type seckillService struct {
	mu     sync.Mutex
	stock  int
	sold   map[int64]bool
	orders []order
}

func newService(stock int) *seckillService {
	return &seckillService{
		stock:  stock,
		sold:   make(map[int64]bool, stock),
		orders: make([]order, 0, stock),
	}
}

var orderSeq uint64

func (s *seckillService) seckill(userID int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sold[userID] {
		return "", errAlreadyBought
	}
	if s.stock <= 0 {
		return "", errOutOfStock
	}
	s.stock--
	s.sold[userID] = true
	id := fmt.Sprintf("ORD-%d-%d", time.Now().UnixNano(), atomic.AddUint64(&orderSeq, 1))
	s.orders = append(s.orders, order{ID: id, UserID: userID})
	return id, nil
}

func (s *seckillService) snapshot() (stock int, orders []order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]order, len(s.orders))
	copy(cp, s.orders)
	return s.stock, cp
}

// ========== HTTP 处理器 ==========

type handler struct{ svc *seckillService }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/seckill":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		raw := r.URL.Query().Get("user_id")
		uid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		orderID, err := h.svc.seckill(uid)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"order_id": orderID})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// ========== 压测核心 ==========

type result struct {
	success     int64
	outOfStock  int64
	alreadyBought int64
	otherErr    int64
	latencies   []time.Duration // 受 mu 保护
	mu          sync.Mutex
}

func (r *result) record(latency time.Duration, status int, body string) {
	r.mu.Lock()
	r.latencies = append(r.latencies, latency)
	r.mu.Unlock()

	switch {
	case status == 200:
		atomic.AddInt64(&r.success, 1)
	case status == 409 && containsStr(body, "out of stock"):
		atomic.AddInt64(&r.outOfStock, 1)
	case status == 409 && containsStr(body, "already bought"):
		atomic.AddInt64(&r.alreadyBought, 1)
	default:
		atomic.AddInt64(&r.otherErr, 1)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findStr(s, sub))
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// ========== main ==========

func main() {
	stock       := flag.Int("stock", 100, "初始库存")
	numUsers    := flag.Int("users", 5000, "模拟用户总数（每个 user_id 唯一）")
	concurrency := flag.Int("concurrency", 200, "并发 goroutine 数")
	flag.Parse()

	// 1. 启动内嵌 HTTP 服务（随机端口）
	svc := newService(*stock)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	addr := ln.Addr().String()
	srv := &http.Server{Handler: &handler{svc: svc}}
	go srv.Serve(ln) //nolint
	defer srv.Close()

	baseURL := "http://" + addr

	// 预热：确认服务已就绪
	for i := 0; i < 10; i++ {
		resp, err := http.Get(baseURL + "/seckill")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	fmt.Printf("=== 压测参数 ===\n")
	fmt.Printf("库存: %d | 并发用户: %d | 并发度: %d\n\n", *stock, *numUsers, *concurrency)

	// 2. 构造任务队列
	tasks := make(chan int64, *numUsers)
	for i := 0; i < *numUsers; i++ {
		tasks <- int64(i + 1)
	}
	close(tasks)

	// 3. 并发压测
	res := &result{latencies: make([]time.Duration, 0, *numUsers)}
	client := &http.Client{Timeout: 5 * time.Second}

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range tasks {
				url := fmt.Sprintf("%s/seckill?user_id=%d", baseURL, userID)

				t0 := time.Now()
				resp, err := client.Post(url, "", nil)
				latency := time.Since(t0)

				if err != nil {
					res.record(latency, 0, "")
					continue
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				res.record(latency, resp.StatusCode, string(bodyBytes))
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 4. 统计
	sort.Slice(res.latencies, func(i, j int) bool {
		return res.latencies[i] < res.latencies[j]
	})

	total := int64(*numUsers)
	qps := float64(total) / elapsed.Seconds()

	fmt.Printf("=== 压测结果 ===\n")
	fmt.Printf("总耗时:        %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("总请求:        %d\n", total)
	fmt.Printf("成功(下单):    %d\n", res.success)
	fmt.Printf("库存不足:      %d\n", res.outOfStock)
	fmt.Printf("重复购买:      %d\n", res.alreadyBought)
	fmt.Printf("其他错误:      %d\n", res.otherErr)
	fmt.Printf("QPS:           %.0f\n", qps)
	fmt.Printf("延迟 p50:      %v\n", percentile(res.latencies, 0.50).Round(time.Microsecond))
	fmt.Printf("延迟 p95:      %v\n", percentile(res.latencies, 0.95).Round(time.Microsecond))
	fmt.Printf("延迟 p99:      %v\n", percentile(res.latencies, 0.99).Round(time.Microsecond))

	// 5. 正确性校验
	_, orders := svc.snapshot()
	fmt.Printf("\n=== 正确性校验 ===\n")

	oversold := int(res.success) > *stock
	if oversold {
		fmt.Printf("✗ 超卖！成功订单 %d > 库存 %d\n", res.success, *stock)
	} else {
		fmt.Printf("✓ 无超卖（成功订单 %d == 库存 %d）\n", res.success, *stock)
	}

	seen := make(map[int64]bool, len(orders))
	dupUser := false
	for _, o := range orders {
		if seen[o.UserID] {
			dupUser = true
			fmt.Printf("✗ 重复用户订单: userID=%d\n", o.UserID)
		}
		seen[o.UserID] = true
	}
	if !dupUser {
		fmt.Printf("✓ 无重复用户订单\n")
	}

	if int(res.success) != len(orders) {
		fmt.Printf("✗ 计数不一致：HTTP 成功=%d，订单记录=%d\n", res.success, len(orders))
	} else {
		fmt.Printf("✓ HTTP 成功数与订单记录一致（%d）\n", len(orders))
	}
}
