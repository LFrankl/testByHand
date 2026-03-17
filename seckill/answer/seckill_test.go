package seckill

import (
	"fmt"
	"sync"
	"testing"
)

// TestNoOversell 验证高并发下不会超卖。
func TestNoOversell(t *testing.T) {
	const (
		initStock  = 10
		numBuyers  = 200 // 模拟 200 个用户同时抢购
	)

	svc := NewSeckillService(initStock)

	var (
		wg         sync.WaitGroup
		successCnt int64
		mu         sync.Mutex
	)

	for i := 0; i < numBuyers; i++ {
		wg.Add(1)
		go func(userID int64) {
			defer wg.Done()
			_, err := svc.Seckill(userID)
			if err == nil {
				mu.Lock()
				successCnt++
				mu.Unlock()
			}
		}(int64(i))
	}
	wg.Wait()

	if successCnt > initStock {
		t.Fatalf("oversell! sold=%d > stock=%d", successCnt, initStock)
	}
	if successCnt != int64(initStock) {
		t.Fatalf("underbuy? sold=%d, expected=%d", successCnt, initStock)
	}
	if svc.Stock() != 0 {
		t.Fatalf("remaining stock should be 0, got %d", svc.Stock())
	}
}

// TestNoDupBuy 验证同一用户只能购买一次。
func TestNoDupBuy(t *testing.T) {
	svc := NewSeckillService(100)

	const userID = int64(42)
	const attempts = 50

	var wg sync.WaitGroup
	results := make([]error, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := svc.Seckill(userID)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	successCnt := 0
	for _, err := range results {
		if err == nil {
			successCnt++
		}
	}
	if successCnt != 1 {
		t.Fatalf("user %d bought %d times, want exactly 1", userID, successCnt)
	}
}

// TestOrdersConsistency 验证订单列表与成功次数一致，且无重复用户。
func TestOrdersConsistency(t *testing.T) {
	const (
		initStock = 5
		numBuyers = 50
	)
	svc := NewSeckillService(initStock)

	var wg sync.WaitGroup
	for i := 0; i < numBuyers; i++ {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			svc.Seckill(uid) //nolint
		}(int64(i))
	}
	wg.Wait()

	orders := svc.Orders()
	if len(orders) != initStock {
		t.Fatalf("order count=%d, want %d", len(orders), initStock)
	}

	seen := make(map[int64]bool)
	for _, o := range orders {
		if seen[o.UserID] {
			t.Fatalf("duplicate order for user %d", o.UserID)
		}
		seen[o.UserID] = true
		if o.ID == "" {
			t.Fatal("order ID should not be empty")
		}
	}
}

// TestHTTPSeckill 验证 HTTP 层端到端行为（不启动真实服务器）。
func TestHTTPSeckill(t *testing.T) {
	svc := NewSeckillService(1)
	h := NewHandler(svc)

	// 第一次购买 → 成功
	w1, r1 := newTestRequest("POST", "/seckill?user_id=1")
	h.ServeHTTP(w1, r1)
	if w1.Code != 200 {
		t.Fatalf("first buy: want 200, got %d body=%s", w1.Code, w1.Body.String())
	}

	// 第二次购买（库存耗尽）→ 409
	w2, r2 := newTestRequest("POST", "/seckill?user_id=2")
	h.ServeHTTP(w2, r2)
	if w2.Code != 409 {
		t.Fatalf("out of stock: want 409, got %d", w2.Code)
	}

	// 重复购买 → 409
	w3, r3 := newTestRequest("POST", "/seckill?user_id=1")
	h.ServeHTTP(w3, r3)
	if w3.Code != 409 {
		t.Fatalf("dup buy: want 409, got %d", w3.Code)
	}

	// 查库存 → 0
	w4, r4 := newTestRequest("GET", "/stock")
	h.ServeHTTP(w4, r4)
	if w4.Code != 200 {
		t.Fatalf("stock: want 200, got %d", w4.Code)
	}
	fmt.Printf("stock response: %s\n", w4.Body.String())
}
