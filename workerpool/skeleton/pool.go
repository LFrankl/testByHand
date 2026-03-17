// 限时练习骨架：基于 channel 的并发工作池
//
// 实现目标：
//
//	WorkerPool — 有界 goroutine 池
//	  · workers 个 goroutine 持续消费 taskQueue channel 中的任务
//	  · Submit    阻塞投递（池关闭后返回 ErrPoolClosed）
//	  · TrySubmit 非阻塞投递（队列满或池关闭返回 false）
//	  · Shutdown  优雅关闭：不再接受新任务，等待所有已提交任务完成
//
//	Future — 带返回值的任务封装
//	  · Submit 后立即获得 *Future，调用 Get 阻塞等待结果
//
//	Pipeline 工具函数
//	  · FanIn  将多个 <-chan int 合并为一个
//	  · FanOut 将一个 <-chan int 分发给 n 个 worker 并行处理
//
// ⚠️  两个常见陷阱（骨架注释里有提示）：
//   1. close(taskQueue) 会让 Submit 的 select 写入已关闭 channel 导致 panic
//      → 解法：taskQueue 永不关闭，用 nil sentinel 通知 worker 退出
//   2. wg.Add 在 worker 内部会让 Shutdown 的 wg.Wait 提前返回
//      → 解法：wg.Add 在 Submit 时做，wg.Done 在 worker 执行完后做
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
type WorkerPool struct {
	taskQueue  chan Task
	numWorkers int
	mu         sync.Mutex // 保护 shutdown 字段
	shutdown   bool
	wg         sync.WaitGroup // 计数已提交但未完成的任务
	once       sync.Once
}

// NewWorkerPool 创建并启动 workers 个 goroutine，任务队列容量为 queueSize。
// TODO: 实现该函数
//
// worker 循环要点：
//   - 从 taskQueue 读取任务并执行
//   - 收到 nil 时退出（nil 是 Shutdown 发送的 sentinel）
//   - wg.Done() 放在任务执行完之后
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	panic("TODO")
}

// Submit 阻塞投递任务，池关闭后返回 ErrPoolClosed。
// TODO: 实现该方法
//
// 关键点：mu.Lock 期间同时完成 shutdown 检查 和 wg.Add，
// 保证两步原子，避免 Shutdown 在两步之间插入导致 wg 漏计。
func (p *WorkerPool) Submit(task Task) error {
	panic("TODO")
}

// TrySubmit 非阻塞投递，池关闭或队列满时立即返回 false。
// TODO: 实现该方法（用 select + default 分支）
func (p *WorkerPool) TrySubmit(task Task) bool {
	panic("TODO")
}

// Shutdown 优雅关闭，步骤严格有序：
//  1. shutdown=true   → 后续 Submit 直接报错
//  2. wg.Wait()       → 等待所有已提交（含队列中）的任务完成
//  3. 发送 nil sentinel → 每个 worker 一个，让其退出
//
// TODO: 实现该方法（用 sync.Once 保证幂等）
func (p *WorkerPool) Shutdown() {
	panic("TODO")
}

// ─────────────────────────────────────────
// Future：带返回值的任务
// ─────────────────────────────────────────

type Result struct {
	Value any
	Err   error
}

// Future 代表一个尚未完成的计算。
type Future struct {
	ch chan Result // 缓冲为 1，防止 worker 在无人 Get 时永久阻塞
}

// Get 阻塞等待结果，支持 context 超时/取消。
// TODO: 实现该方法（select 监听 ch 和 ctx.Done）
func (f *Future) Get(ctx context.Context) (Result, error) {
	panic("TODO")
}

// SubmitFuture 将带返回值的函数投递到池，立即返回 *Future。
// TODO: 实现该函数
//
// 提示：
//  1. make(chan Result, 1)  ← 缓冲为 1
//  2. 包装成 Task：执行 fn 并将结果写入 ch
//  3. 调用 p.Submit(task)
func SubmitFuture(p *WorkerPool, fn func() (any, error)) (*Future, error) {
	panic("TODO")
}

// ─────────────────────────────────────────
// Pipeline 工具：FanIn / FanOut
// ─────────────────────────────────────────

// FanIn 将多个输入 channel 合并为一个输出 channel。
// 所有输入 channel 关闭后，输出 channel 自动关闭。
// ctx 取消时提前退出。
// TODO: 实现该函数
//
// 提示：每个 in 对应一个转发 goroutine + sync.WaitGroup 追踪，
// 全部退出后 close(out)。
func FanIn(ctx context.Context, ins ...<-chan int) <-chan int {
	panic("TODO")
}

// FanOut 从 in 读取数据，n 个 worker 并行处理后汇入输出 channel。
// ctx 取消时提前退出。
// TODO: 实现该函数
//
// 提示：Go channel 天然支持多 goroutine 并发读（无需额外分发逻辑）。
func FanOut(ctx context.Context, in <-chan int, n int, fn func(int) int) <-chan int {
	panic("TODO")
}

// ─────────────────────────────────────────
// 确保编译通过
// ─────────────────────────────────────────
var _ = context.Background
var _ = sync.WaitGroup{}
var _ = errors.New
