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
