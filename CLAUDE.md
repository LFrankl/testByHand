# forinterview — 面试练习题库

Go 语言面试题练习项目。每个子目录是一个独立题目，含骨架和完整答案。

## 目录结构

```
<topic>/
├── skeleton/   ← 限时练习骨架（TODO 待填）
│   ├── *.go
│   └── *_test.go
└── answer/     ← 完整实现答案
    ├── *.go
    └── *_test.go
```

每个 skeleton/answer 目录均为独立 Go module（有各自的 go.mod）。

## 已有题目

| 目录 | 题目 |
|------|------|
| `tcp-protocol` | 在 TCP 之上设计自定义应用层协议 |
| `lru` | 线程安全 LRU 缓存 |
| `seckill` | 简单秒杀服务（含 HTTP 层 + 压测脚本） |
| `webframe` | 基于 `net/http` 的 Web 框架（Trie 路由 + 中间件） |
| `workerpool` | 基于 channel 的工作池（含 Future / FanIn / FanOut） |
| `connpool` | 模拟连接池（懒加载 + 空闲超时 + 优雅关闭） |
| `ratelimiter` | 令牌桶 + 滑动窗口限流器 |
| `mystring` | 自定义 String 类（不可变 + KMP + Builder）|
| `syncmap` | sync.Map（两张表 + miss 提升 + expunged 哨兵）|
| `puzzles` | 智力题集（逻辑推理题，单文件维护）|

## 新增题目规范

1. **目录命名**：小写中划线，体现题目内容（如 `connpool`、`ratelimiter`）
2. **骨架文件**
   - 提供完整的数据结构和常量，核心方法体替换为 `panic("TODO")`
   - 注释说明：算法思路、参数含义、实现要点、常见陷阱
   - 提供工具函数（已实现，无需修改）
3. **答案文件**
   - 与骨架结构完全对应
   - 注释重点说明设计决策和陷阱解法，而非逐行解释
4. **测试文件**
   - 骨架和答案共用同一份测试（从 skeleton 复制到 answer）
   - 测试必须覆盖：基本功能、边界条件、并发安全（`-race`）
   - 答案的 `go test ./... -race` 必须全绿才可提交
5. **go.mod**：每个 skeleton/answer 目录独立 `go mod init <topic>`
6. **文档同步**：新增题目后，将 answer/ 中所有非测试 Go 文件的代码和讲解
   同步追加到 `docs/index.html`，并在左侧导航中加入对应条目。

## 智力题规范

智力题与 Go 编程题**不同**，无需 skeleton/answer/go.mod 目录结构：

1. **唯一文档**：所有智力题维护在 `puzzles/puzzles.md`，直接追加，格式：
   - `## N. 题目名称`
   - **题目**（描述）
   - **关键思路**（推理过程，可含表格/列表）
   - **答案**（加粗）
2. **文档同步**：追加新题后，同步在 `docs/index.html` 的 `#puzzles` section 中添加对应
   `.puzzle-card` HTML 块，并将题目内容嵌入（无需 JS codes 对象，直接写 HTML）。
3. **导航**：智力题使用独立导航分区（紫色主题），单个导航入口 `#puzzles`，
   页面内以可折叠卡片展示所有题目。

## 系统设计题规范

系统设计题与 Go 编程题不同，无需 skeleton/answer/go.mod 目录结构：

1. **唯一文档**：所有系统设计题维护在 `sysdesign/sysdesign.md`，直接追加，格式：
   - `## N. 题目名称`
   - **题目**（描述背景和要求）
   - **设计要点**（分点列出核心设计考量）
   - **推荐方案**（架构图文字描述 + 技术选型表格）
2. **文档同步**：追加新题后，同步在 `docs/index.html` 的 `#sysdesign` section 中添加对应
   `.sd-card` HTML 块（参考已有卡片结构，使用 `toggleSD(this)` 和 `.sd-*` 类名）。
3. **导航**：系统设计题使用独立导航分区（蓝色主题 `#2563EB`），单个导航入口 `#sysdesign`，
   页面内以可折叠卡片展示所有题目。

## 代码风格

- 纯标准库，无第三方依赖
- 错误变量统一用 `var Err... = errors.New(...)`
- 并发安全优先：能用 atomic 的不加锁，能用 channel 的不用条件变量
- 骨架的 `panic("TODO")` 下方可保留 `var _ = ...` 确保编译通过

## Git 规范

- 提交者：LFrankl <fuquanliang0@gmail.com>
- 不在 commit message 中注明 AI 参与
- `.gitignore` 排除 `.DS_Store`、编译产物（`*.exe`、`*.out`、二进制）
