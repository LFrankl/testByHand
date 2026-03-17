package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── WorkerPool ───────────────────────────────────────

func TestWorkerPoolBasic(t *testing.T) {
	p := NewWorkerPool(4, 16)
	defer p.Shutdown()

	var count int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		if err := p.Submit(func() {
			defer wg.Done()
			atomic.AddInt64(&count, 1)
		}); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	wg.Wait()

	if count != 100 {
		t.Fatalf("want 100 tasks done, got %d", count)
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	// 验证同时执行的 goroutine 数量不超过 workers
	const workers = 3
	p := NewWorkerPool(workers, 32)
	defer p.Shutdown()

	var active int64
	var maxActive int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			defer wg.Done()
			cur := atomic.AddInt64(&active, 1)
			mu.Lock()
			if cur > maxActive {
				maxActive = cur
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&active, -1)
		})
	}
	wg.Wait()

	if maxActive > workers {
		t.Fatalf("max concurrent=%d exceeded workers=%d", maxActive, workers)
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	p := NewWorkerPool(2, 8)

	var done int64
	for i := 0; i < 5; i++ {
		_ = p.Submit(func() {
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt64(&done, 1)
		})
	}

	p.Shutdown() // 应等待正在执行的任务完成

	// 验证：Shutdown 返回后，正在执行的任务必须已完成
	// （队列里尚未被 worker 取走的任务不保证执行，只保证已取走的完成）
	if done == 0 {
		t.Fatal("Shutdown returned but no task completed")
	}
}

func TestWorkerPoolClosedError(t *testing.T) {
	p := NewWorkerPool(2, 4)
	p.Shutdown()

	err := p.Submit(func() {})
	if err != ErrPoolClosed {
		t.Fatalf("want ErrPoolClosed, got %v", err)
	}
}

func TestTrySubmit(t *testing.T) {
	// 队列容量为 1，先塞满
	p := NewWorkerPool(1, 1)
	defer p.Shutdown()

	// 先投一个耗时任务占住 worker
	blocker := make(chan struct{})
	_ = p.Submit(func() { <-blocker })

	// 再投一个塞满队列
	time.Sleep(5 * time.Millisecond) // 等 worker 取走第一个
	_ = p.TrySubmit(func() {})       // 这个进队列

	// 此时队列满，TrySubmit 应返回 false
	ok := p.TrySubmit(func() {})
	close(blocker)
	if ok {
		t.Fatal("TrySubmit should return false when queue is full")
	}
}

// ─── Future ───────────────────────────────────────────

func TestFuture(t *testing.T) {
	p := NewWorkerPool(2, 8)
	defer p.Shutdown()

	f, err := SubmitFuture(p, func() (any, error) {
		time.Sleep(10 * time.Millisecond)
		return 42, nil
	})
	if err != nil {
		t.Fatalf("SubmitFuture: %v", err)
	}

	res, err := f.Get(context.Background())
	if err != nil {
		t.Fatalf("Future.Get: %v", err)
	}
	if res.Value.(int) != 42 {
		t.Fatalf("want 42, got %v", res.Value)
	}
}

func TestFutureTimeout(t *testing.T) {
	p := NewWorkerPool(1, 4)
	defer p.Shutdown()

	f, _ := SubmitFuture(p, func() (any, error) {
		time.Sleep(500 * time.Millisecond)
		return "done", nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Get(ctx)
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
}

// ─── FanIn / FanOut ───────────────────────────────────

func TestFanIn(t *testing.T) {
	ctx := context.Background()

	gen := func(vals ...int) <-chan int {
		ch := make(chan int, len(vals))
		for _, v := range vals {
			ch <- v
		}
		close(ch)
		return ch
	}

	out := FanIn(ctx, gen(1, 2), gen(3, 4), gen(5, 6))

	var got []int
	for v := range out {
		got = append(got, v)
	}

	if len(got) != 6 {
		t.Fatalf("want 6 values, got %d: %v", len(got), got)
	}
}

func TestFanOut(t *testing.T) {
	ctx := context.Background()

	in := make(chan int, 20)
	for i := 1; i <= 10; i++ {
		in <- i
	}
	close(in)

	out := FanOut(ctx, in, 4, func(v int) int { return v * 2 })

	var sum int64
	for v := range out {
		atomic.AddInt64(&sum, int64(v))
	}

	// 1+2+…+10 = 55，每个乘2 = 110
	if sum != 110 {
		t.Fatalf("want sum=110, got %d", sum)
	}
}

func TestFanInContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	slow := make(chan int) // 永远不关闭
	out := FanIn(ctx, slow)

	cancel() // 触发取消

	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("expected channel to close after context cancel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("FanIn did not respect context cancellation")
	}
}
