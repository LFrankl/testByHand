// 答案版本：基于 channel 的并发工作池完整实现
//
// 设计要点（两个常见陷阱及其解法）：
//
//	陷阱1 — 关闭 taskQueue 引发 panic：
//	  若用 close(taskQueue) 通知 worker 退出，而 Submit 的 select 同时监听
//	  taskQueue 和 closed，两路同时就绪时 Go 随机选择；若选中 taskQueue<-task
//	  但 taskQueue 已被关闭，则 panic。
//	  解法：taskQueue 永不关闭，用 nil sentinel 通知 worker 退出。
//
//	陷阱2 — wg.Add 在 worker 内部导致 Shutdown 提前返回：
//	  若 worker 取到任务后才 wg.Add(1)，Shutdown 调用 wg.Wait() 时
//	  可能队列里还有任务但 wg 计数已为 0，Wait 立即返回。
//	  解法：wg.Add(1) 在 Submit 时做，wg.Done() 在 worker 执行完任务后做。
//
// 运行测试：
//
//	go test ./... -race
package workerpool

import (
	"context"
	"errors"
	"sync"
)

var ErrPoolClosed = errors.New("worker pool is closed")

type Task func()

// ─────────────────────────────────────────
// WorkerPool
// ─────────────────────────────────────────

// WorkerPool 是固定大小的 goroutine 池。
//
//   taskQueue   带缓冲 channel，Submit 写入，worker 消费
//   mu+shutdown 原子保护"是否已关闭"，避免 select 竞态
//   wg          计数已提交但未完成的任务（Add 在 Submit，Done 在 worker）
//   numWorkers  Shutdown 时发送对应数量的 nil sentinel
type WorkerPool struct {
	taskQueue  chan Task
	numWorkers int
	mu         sync.Mutex
	shutdown   bool
	wg         sync.WaitGroup
	once       sync.Once
}

// NewWorkerPool 创建池并启动 workers 个常驻 goroutine。
//
// worker 循环消费 taskQueue，收到 nil sentinel 时退出。
// wg.Done() 放在任务执行完之后，确保 Shutdown 的 wg.Wait() 等到真正完成。
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	p := &WorkerPool{
		taskQueue:  make(chan Task, queueSize),
		numWorkers: workers,
	}
	for i := 0; i < workers; i++ {
		go func() {
			for task := range p.taskQueue {
				if task == nil { // sentinel：退出
					return
				}
				task()
				p.wg.Done()
			}
		}()
	}
	return p
}

// Submit 阻塞投递任务；池已关闭时返回 ErrPoolClosed。
//
// mu 保证 shutdown 检查 与 wg.Add 原子执行：
// 若两步之间 Shutdown 插入，要么 Add 在 Wait 前完成（正常等待），
// 要么 shutdown=true 被检测到（返回错误），不存在中间态。
func (p *WorkerPool) Submit(task Task) error {
	p.mu.Lock()
	if p.shutdown {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.wg.Add(1) // 持锁期间 Add，保证 Shutdown 的 wg.Wait 能感知
	p.mu.Unlock()

	p.taskQueue <- task // 阻塞直到有空位（worker 消费后释放）
	return nil
}

// TrySubmit 非阻塞投递；池关闭或队列满时返回 false。
func (p *WorkerPool) TrySubmit(task Task) bool {
	p.mu.Lock()
	if p.shutdown {
		p.mu.Unlock()
		return false
	}
	p.wg.Add(1)
	p.mu.Unlock()

	select {
	case p.taskQueue <- task:
		return true
	default:
		p.wg.Done() // 撤销 Add
		return false
	}
}

// Shutdown 优雅关闭，步骤严格有序：
//
//  1. shutdown=true   — 后续 Submit 直接返回 ErrPoolClosed
//  2. wg.Wait()       — 等待所有已提交（含队列中）的任务执行完毕
//  3. 发送 nil sentinel — 逐个唤醒并终止 worker
//
// 为何先 Wait 再发 sentinel？
// wg.Wait 返回时所有任务已完成（taskQueue 已空），
// 此时再发 nil，worker 读到 nil 后队列必然为空，干净退出。
func (p *WorkerPool) Shutdown() {
	p.once.Do(func() {
		p.mu.Lock()
		p.shutdown = true
		p.mu.Unlock()

		p.wg.Wait() // 等待所有已提交任务完成

		for i := 0; i < p.numWorkers; i++ {
			p.taskQueue <- nil // sentinel：让每个 worker 退出
		}
	})
}

// ─────────────────────────────────────────
// Future：带返回值的任务
// ─────────────────────────────────────────

type Result struct {
	Value any
	Err   error
}

// Future 代表一个尚未完成的计算。
// ch 缓冲为 1：即使调用方从未调用 Get，worker 也不会阻塞在写入上。
type Future struct {
	ch chan Result
}

// Get 阻塞等待结果，支持 context 超时/取消。
func (f *Future) Get(ctx context.Context) (Result, error) {
	select {
	case res := <-f.ch:
		return res, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// SubmitFuture 将带返回值的函数投递到池，立即返回 *Future。
func SubmitFuture(p *WorkerPool, fn func() (any, error)) (*Future, error) {
	f := &Future{ch: make(chan Result, 1)}
	err := p.Submit(func() {
		val, err := fn()
		f.ch <- Result{Value: val, Err: err}
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ─────────────────────────────────────────
// Pipeline 工具：FanIn / FanOut
// ─────────────────────────────────────────

// FanIn 将多个输入 channel 合并为一个输出 channel。
//
// 每个 in 启动一个转发 goroutine，用 WaitGroup 追踪，
// 全部退出后关闭 out。ctx 取消时，转发 goroutine 感知后提前退出。
func FanIn(ctx context.Context, ins ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	forward := func(in <-chan int) {
		defer wg.Done()
		for {
			select {
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(ins))
	for _, in := range ins {
		go forward(in)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// FanOut 从一个输入 channel 读取，n 个 worker 并行处理后汇入输出 channel。
//
// Go channel 天然支持多 goroutine 并发读取（无需额外分发逻辑），
// 这正是 fan-out 模式的精髓。
func FanOut(ctx context.Context, in <-chan int, n int, fn func(int) int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case v, ok := <-in:
					if !ok {
						return
					}
					select {
					case out <- fn(v):
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
