# 系统设计题集

互联网大厂开发岗系统设计面试题，按题目编号顺序维护。

格式：`## N. 题目名称` + **题目** + **设计要点** + **推荐方案**

---

## 1. 设计推特（Twitter Feed）

**题目**

设计一个类似 Twitter 的信息流系统。用户可以发布推文、关注其他用户，并查看关注者的最新时间线（Home Timeline）。要求支持海量用户（亿级）、高并发读（每秒百万级）。

**设计要点**

- **规模估算**：日活 1 亿，每用户平均关注 200 人，发推 QPS ~5000，读 Timeline QPS ~50 万
- **核心难点**：写扩散 vs 读扩散。大 V（百万粉）发一条推文，若写扩散需写入百万用户的 Timeline 缓存
- **Fanout 策略**：普通用户（<1 万粉）写时扇出到粉丝 Feed 缓存；大 V 走读时合并（避免写放大）
- **存储**：推文表用 MySQL/TiDB（按 user_id 分片）；Timeline 缓存用 Redis Sorted Set（score=时间戳）
- **缓存预热**：用户登录时从数据库拉取最近 N 条 Feed，写入 Redis；TTL 7 天
- **消息队列**：发推事件投入 Kafka，消费者异步写扩散，削峰解耦
- **CDN**：媒体文件（图片/视频）通过 CDN 分发，减少源站压力

**推荐方案**

```
用户发推
  → 写 MySQL（推文持久化）
  → 投 Kafka（topic: new_tweet）
  → 消费者判断发送者粉丝数
      ≤ 1万  → 遍历粉丝，写 Redis ZADD timeline:{uid} score=ts tweet_id
      > 1万  → 只写推文，不扇出（大V）

用户读 Timeline
  → Redis ZREVRANGE timeline:{uid} 0 99
  → 大V 发的推文：单独查 MySQL，合并排序
  → 返回推文 ID 列表，批量查推文详情（多级缓存）
```

| 层次 | 技术选型 | 说明 |
|------|---------|------|
| 接入层 | Nginx + API Gateway | 限流、鉴权、路由 |
| 推文存储 | MySQL（按 user_id 分片） | 持久化，支持复杂查询 |
| Timeline 缓存 | Redis Sorted Set | 高速读，score=时间戳 |
| 消息队列 | Kafka | 异步扇出，削峰 |
| 媒体存储 | 对象存储 + CDN | 图片/视频分发 |
| 搜索 | Elasticsearch | 推文全文检索 |

---

## 2. 设计 URL 短链服务

**题目**

设计一个 URL 缩短服务（类似 bit.ly）。将长 URL 映射为 6~8 位短码，访问短链时 302 重定向到原始 URL。要求：写 QPS ~1000，读 QPS ~10 万，短链有效期可配置。

**设计要点**

- **短码生成**：62 进制（a-z A-Z 0-9）编码，6 位 = 56 亿种组合，满足需求
- **生成算法**：
  - 哈希截断：MD5/MurmurHash 取前 6 字节转 62 进制，冲突时自增重试
  - 发号器：数据库自增 ID 转 62 进制（推荐，无冲突，可预分配 ID 段）
- **存储**：短码→长 URL 用 MySQL（短码做主键）；热点缓存 Redis（TTL 与短链一致）
- **重定向**：302（临时重定向，不缓存，统计准确）vs 301（永久重定向，客户端缓存，减少服务器压力）
- **自定义短码**：允许用户指定短码，需加唯一索引防冲突
- **过期清理**：TTL 字段 + 定时任务（Cron）扫描删除；Redis Key 设置相同 TTL 自动失效

**推荐方案**

```
写请求（长 URL → 短码）
  → 检查 Redis：该长URL是否已有短码（去重）
  → 无缓存：向发号器服务申请自增 ID
  → ID 转 62 进制得到短码
  → MySQL INSERT (short_code, long_url, expire_at)
  → Redis SET short:{code} long_url EX ttl
  → 返回短链

读请求（短码 → 重定向）
  → Redis GET short:{code}
  → 未命中 → MySQL 查询 → 写回 Redis
  → 未找到 → 404
  → 找到 → HTTP 302 Location: long_url
```

| 组件 | 技术 | 说明 |
|------|------|------|
| 发号器 | MySQL auto_increment + 号段模式 | 每次领取 1000 个 ID，减少 DB 访问 |
| 持久化 | MySQL（short_code 主键） | 唯一约束防冲突 |
| 缓存 | Redis String | 热点短链，减少 DB 读压力 |
| 统计 | Kafka + ClickHouse | 异步统计点击量、来源 |

---

## 3. 设计微信群聊消息系统

**题目**

设计微信群聊消息系统。支持万人群，消息实时推送，离线消息存储，消息序号保证有序，支持已读/未读状态。

**设计要点**

- **消息序号（Message ID）**：每个群维护单调递增序号（Snowflake 或群级别 Redis INCR），保证有序
- **存储**：消息表按 group_id + msg_id 分片；用户收件箱（Inbox）维护游标（cursor = 已读到哪条 msg_id）
- **推送模型**：在线用户通过 WebSocket/长连接实时推送；离线用户拉取（pull）
- **万人群写扩散问题**：不在 DB 层为每个成员写一条收件记录（写放大 1 万倍），改用群消息 + 成员游标
- **离线消息**：用户上线后，查询各群 cursor 之后的消息，批量拉取
- **已读回执**：更新 cursor 即视为已读，服务端汇总计算未读数

**推荐方案**

```
发消息
  → 分配 msg_id（Redis INCR group:{gid}:seq）
  → 写 MySQL msg 表（gid, msg_id, sender, content, ts）
  → 投 Kafka（topic: group_msg）
  → 消费者：对在线成员 WebSocket 推送 msg_id
  → 离线成员：等上线后 pull

用户上线
  → 查 cursor 表：获取各群最后已读 msg_id
  → 对每个群：SELECT * FROM msg WHERE gid=? AND msg_id > cursor LIMIT 100
  → 推送离线消息，更新 cursor

读最新消息
  → 优先 Redis（热群消息缓冲最近 1000 条）
  → 历史消息走 MySQL
```

| 问题 | 方案 |
|------|------|
| 消息有序 | 群级别自增 msg_id（Redis INCR + Lua 保证原子） |
| 万人群写放大 | 不写扩散，成员维护 cursor，上线 pull |
| 实时推送 | WebSocket 长连接，接入层 push-server（无状态 + 一致性哈希路由） |
| 消息持久化 | 按 gid 分片 MySQL，冷数据归档对象存储 |
| 未读数 | cursor 与群最新 msg_id 之差，Redis 缓存 |

---

## 4. 设计秒杀系统

**题目**

设计一个双十一秒杀系统。1000 件商品，100 万用户同时抢购，要求不超卖、不少卖、高可用，系统 QPS 峰值可达 10 万。

**设计要点**

- **核心问题**：库存扣减的原子性（避免超卖）
- **流量漏斗**：请求在到达数据库之前应被大量过滤，只有极少数能到扣库存环节
- **限流策略**：Nginx 层限流（令牌桶）→ 网关层用户级限频（同一用户 1 秒只处理 1 次）→ MQ 削峰
- **库存预热**：活动开始前将库存写入 Redis（`SETNX`/`SET NX EX`），利用 Redis 原子性扣减
- **Redis 扣库存**：`DECRBY`（原子操作）或 Lua 脚本（判断 + 扣减原子化），避免 ABA 问题
- **异步下单**：扣库存成功后将订单信息投入 Kafka，消费者异步写 DB；用户轮询订单状态
- **幂等性**：同一用户同一商品只能扣一次（Redis SET 用户+商品唯一键，NX 保证）
- **DB 兜底**：消费者写 DB 时用乐观锁（`WHERE stock > 0 AND version = ?`）再次保证不超卖

**推荐方案**

```
请求链路：
  用户 → CDN（静态资源）
       → Nginx 限流（令牌桶 10w QPS → 放行 1w）
       → API Gateway（用户级限频、鉴权）
       → 秒杀服务
           1. Redis: SET seckill:uid:gid 1 NX EX 3  （去重，防重复抢）
           2. Redis: DECR seckill:stock:{gid}        （扣库存，原子）
              → 返回 < 0 → 库存不足，INCR 回滚
           3. 投 Kafka（topic: seckill_order）
           4. 返回"排队中"给用户
  消费者：
       → 从 Kafka 消费订单消息
       → MySQL INSERT order + UPDATE stock（乐观锁）
       → 写订单状态到 Redis，用户轮询
```

| 问题 | 方案 | 原因 |
|------|------|------|
| 超卖 | Redis DECR + 乐观锁兜底 | 原子操作 + DB 最终校验 |
| 重复购买 | Redis NX 去重键 | 原子 SET，TTL 自动清理 |
| 流量削峰 | Kafka 缓冲 + MQ 消费 | 解耦，保护 DB |
| 高可用 | Redis Cluster + MySQL 主从 | 避免单点 |

---

## 5. 设计消息队列（简化版 Kafka）

**题目**

设计一个简化版的消息队列系统，支持：Producer 发布消息、Consumer 订阅消费、消息持久化、至少一次投递（at-least-once）、消费者组（Consumer Group）负载均衡。

**设计要点**

- **核心抽象**：Topic → Partition（分区）→ Segment（分段日志文件）
- **顺序写磁盘**：消息追加写到 Partition 日志文件（Append-Only），顺序 I/O 接近内存速度
- **Offset**：消费者维护消费位点，重启后从上次 offset 继续，实现 at-least-once
- **Consumer Group**：同一 Group 内，一个 Partition 只被一个 Consumer 消费（分区独占），保证组内不重复消费；不同 Group 独立消费
- **Partition 分配**：Group Coordinator 负责 Rebalance（Consumer 加入/退出时重新分配 Partition）
- **副本（Replication）**：每个 Partition 有 N 个副本，Leader 接收读写，Follower 同步；Leader 宕机自动选主（ISR 机制）
- **索引文件**：稀疏索引（每隔 4KB 记录一个 offset→文件位置），加速随机读
- **消息保留**：基于时间（7 天）或大小（10GB）滚动删除旧 Segment

**推荐方案**

```
存储结构：
  /data/topic-name/
    partition-0/
      00000000.log    ← 消息文件（顺序追加）
      00000000.index  ← 稀疏 offset 索引
    partition-1/
      ...

写入流程：
  Producer → 按 key hash 选 Partition（无 key 则轮询）
           → 发送到 Partition Leader
           → Leader 追加写 .log
           → Follower 同步（ISR 列表内的副本确认后 ack）
           → 返回 offset 给 Producer

消费流程：
  Consumer → 向 Coordinator 注册，加入 ConsumerGroup
           → Coordinator 分配 Partition（一对一）
           → 从上次 committed offset 开始 Fetch
           → 处理成功 → Commit offset（写 __consumer_offsets topic）
           → 失败 → 不 commit，下次重新投递（at-least-once）
```

| 特性 | 实现 |
|------|------|
| 高吞吐 | 顺序写磁盘 + 零拷贝（sendfile） |
| 持久化 | Append-Only 日志，多副本 |
| 有序 | 同一 Partition 内严格有序 |
| 负载均衡 | Partition 分配给 Consumer Group 成员 |
| 容灾 | ISR 副本 + Leader 自动选举 |

---

## 6. 设计分布式 ID 生成器

**题目**

设计一个分布式 ID 生成服务，满足：全局唯一、趋势递增（有利于 B+ 树索引写入）、高性能（单机 >10 万 QPS）、高可用（无单点）、ID 不暴露业务信息（不可被枚举）。

**设计要点**

各方案对比：

| 方案 | 唯一性 | 趋势递增 | 性能 | 主要缺点 |
|------|--------|---------|------|---------|
| UUID v4 | ✅ | ❌ 完全随机 | ✅ 本地极快 | 128 bit；随机写导致 B+ 树页分裂；不含时间 |
| DB 自增 ID | ✅（单库） | ✅ 严格递增 | ❌ DB 瓶颈 | 单点；多库步长扩容麻烦；暴露业务规模 |
| Redis INCR | ✅ | ✅ | ✅ ~10w QPS | AOF 持久化有丢失风险；引入额外依赖 |
| Snowflake | ✅ | ✅ 趋势递增 | ✅ 本地极快 | 依赖系统时钟；时钟回拨导致 ID 重复 |
| 美团 Leaf | ✅ | ✅ | ✅ 号段缓存 | 号段模式 DB 宕机有缓存兜底；ZK 分配 workerId |
| 百度 UidGenerator | ✅ | ✅ | ✅ RingBuffer | DB 注册 workerId；RingBuffer 预生成消除毛刺 |

**Snowflake 结构（64 bit）**

```
符号位(1) | 时间戳ms(41) | workerId(10) | 序列号(12)
```
- 41 bit 时间戳：相对自定义 epoch，可用约 69 年
- 10 bit workerId：5 bit datacenter + 5 bit worker，共 1024 节点
- 12 bit 序列号：同毫秒内自增，上限 4096；单机理论 QPS ~409 万

**时钟回拨解决方案**

- **拒绝服务**：小回拨（<2ms）等待追上，大回拨报警拒绝发号
- **借位方案**：将 workerId 高位几 bit 作回拨次数计数器，每次回拨自增（美团 Leaf 思路）
- **逻辑时钟**：不依赖系统时钟，用单调递增逻辑计数，彻底消除回拨风险

**美团 Leaf 双模式**

- **号段模式**：DB 维护 bizTag 的 max_id + step，批量取号（一次取 1000）存内存依次发放；剩余低于 10% 时异步预取下一段（双 buffer），消除取号延迟毛刺
- **Snowflake 模式**：ZK 持久节点存储 workerId，启动时注册获取；定期上报心跳时间戳用于检测时钟回拨

**推荐方案**

**答案**：中小规模用 Redis INCR + 号段批量取；大规模上美团 Leaf（号段模式主力 + Snowflake 模式补充）；容器环境用百度 UidGenerator（DB 注册 workerId + RingBuffer 异步预生成）。时钟回拨处理：小回拨等待，大回拨降级号段模式。

---

## 7. 设计 Feed 流系统

**题目**

设计一个通用 Feed 流系统（类微博/朋友圈/抖音）。用户关注其他用户后，能在首页看到关注者按时间倒序排列的内容流。要求：日活 1 亿，写 QPS ~5 万，读 QPS ~50 万，支持普通用户和大 V（千万粉），延迟 <200ms。

**设计要点**

三种 Fanout 模型对比：

| 模型 | 原理 | 读延迟 | 写代价 | 适用场景 |
|------|------|--------|--------|---------|
| 推模式（写扩散） | 发布时将 post_id 写入所有粉丝的 Timeline 缓存 | 极低 | 写放大=粉丝数 | 普通用户（粉丝<1万） |
| 拉模式（读扩散） | 读取时 union 所有关注者发布内容，归并排序 | 高 | 无额外写入 | 关注数少 |
| 混合模式（Hybrid） | 普通用户推、大 V 拉，读时合并 | 可控 | 可控 | ✅ 生产主流（微博/Twitter） |

关键细节：
- **大 V 阈值**：粉丝数超阈值（如 5 万）标记为大 V，不做写扩散
- **游标分页**：不用 offset（数据插入导致漂移），用 `ts + post_id` 作游标
- **Timeline 缓存**：Redis Sorted Set，score=ts，只存 post_id，最多保留 800 条，TTL 7 天
- **冷启动**：新用户 Timeline 为空时，从关注列表拉取每人最新 3 条填充首屏
- **热点帖子**：大 V 爆款帖多级缓存（本地 + Redis 多副本）

**推荐方案**

```
发布：写 MySQL → 投 Kafka → 消费者判断粉丝数
  ≤ 阈值 → ZADD timeline:{fan_uid} score=ts post_id（写扩散）
  > 阈值 → 不扇出，仅落 post 表

读取：
  Step 1: Redis ZREVRANGE timeline:{uid} 0 99（普通关注者推来的）
  Step 2: 查大 V 列表，每人 SELECT 最新 20 条
  Step 3: 归并排序，取 Top 20
  Step 4: 批量 mget 帖子详情（Redis + DB 兜底）
```

**答案**：混合 Fanout 是核心答案。普通用户推模式保证低读延迟，大 V 走拉模式避免写放大，读取时归并两路结果。游标分页、Timeline 缓存裁剪、热点多级缓存是加分项。

---

## 8. 设计多设备登录与踢人下线的认证系统

**题目**

设计一个支持多设备登录、踢人下线的登录认证系统。要求：同一账号可在手机/PC/平板同时登录；支持配置「同类型设备互踢」（如只允许一台手机在线）；Token 无状态但支持主动失效；安全可审计。

**设计要点**

Token 方案选型：

| 方案 | 优点 | 缺点 |
|------|------|------|
| Session + Cookie | 主动失效简单 | 有状态，需共享 Session |
| JWT 纯无状态 | 水平扩展友好 | ❌ 无法主动失效 |
| JWT + Redis 黑名单 | 兼顾性能与失效 | 每请求多一次 Redis 查询 |
| Opaque Token（引用 Token） | 精细管理多设备 | 每次必须查 Redis |

推荐：Access Token = 短期 JWT（15min）+ Refresh Token = Opaque Token 存 Redis。

多设备管理：
- Redis Hash `sessions:{uid}`，field = `{device_type}:{device_id}`，value = session JSON
- 新设备登录时检查同类型设备数，超限则踢最旧的

踢人下线方案：
- **同类型互踢**：HDEL 旧设备 session + 旧 token_id 写黑名单（TTL = JWT 剩余时长）
- **管理员强制下线**：DEL sessions:{uid} + 批量写黑名单
- **全部下线**：`INCR token_ver:{uid}`，JWT 中携带版本号，版本不符即 401，无需维护黑名单列表

安全要点：
- Refresh Token Rotation：每次续期颁发新 Refresh Token，旧的立即失效；重用检测视为泄露，全部下线
- HTTPS + HttpOnly Cookie 存 Refresh Token，防 XSS 窃取
- 设备指纹哈希防止伪造 device_id 绕过互踢

**答案**：短期 JWT + Opaque Refresh Token + Redis sessions Hash 管理多设备；踢人 = 删 session + 写黑名单；全部下线用 token_ver 版本号方案，避免黑名单 Key 爆炸。
