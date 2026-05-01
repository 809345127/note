# ID 选型与分页设计完整总结

## 一、双 ID 架构（最佳实践）

一般系统推荐"内部主键 + 外部业务 ID"的双 ID 方案，两者职责分离。

### 主键 ID：`BIGINT UNSIGNED AUTO_INCREMENT`

- **InnoDB 聚簇索引**按主键物理排列，自增意味着写入几乎都是顺序追加，页分裂少
- 8 字节，JOIN 和索引开销远小于 UUID 的 16 字节（或字符串存储的 36 字节）
- 仅用于内部关联（外键、JOIN），不对外暴露

### 业务 ID：`prefix + UUIDv7`

- 对外暴露不泄露行数、增长速率
- 应用层预生成，不依赖数据库 round-trip
- 分布式环境天然无冲突
- 前缀带来可读性：`ord_<编码后的 UUIDv7>` 一眼知道是订单对象
- 业界参考：Stripe 在公开文档和 SDK 中使用 `prefix_<随机串>` 形式（如 `pi_xxx`、`ch_xxx`），具体编码、ID 长度演进、内部是否做主键映射等细节官方未完全公开，本文不展开

### 推荐 Struct

```go
type Order struct {
    ID         int64     // BIGINT AUTO_INCREMENT，内部主键（如需 UNSIGNED，Go 侧用 uint64）
    BizID      string    // prefix + UUIDv7，对外业务 ID
    OccurredAt time.Time // 业务层生成，代表真实业务发生时间
    CreatedAt  time.Time // DB 自动生成，代表大致入库时间
}
```

> SQL 端用 `BIGINT` 还是 `BIGINT UNSIGNED` 取决于团队规范；Go 端类型需对齐（`int64` ↔ `BIGINT`、`uint64` ↔ `BIGINT UNSIGNED`），避免溢出/隐式转换坑。

---

## 二、有序性问题深入分析

### AUTO_INCREMENT 的有序性

| 阶段 | 是否有序 | 说明 |
|------|---------|------|
| 分配顺序 | ⚠️ 取决于 lock mode | MySQL 8.0 默认 `innodb_autoinc_lock_mode=2`：保证唯一+单调，但**并发 INSERT 之间可交错**，bulk insert 内部也可能不连续。`lock_mode=0/1` 在单语句内连续，但事务回滚、`INSERT IGNORE` 失败、`REPLACE` 等仍会留洞——任何模式下都不该假设"无洞且严格连续" |
| 提交顺序 | ❌ 不保证 | 事务 A 先拿到 id=100，事务 B 拿到 id=101，B 可能先 commit |
| 可见性顺序 | ❌ 不保证 | 存在"空洞"——已分配但尚未提交的 ID |

```
事务A: BEGIN → 拿到id=100 → 慢操作...... → COMMIT (T5)
事务B: BEGIN → 拿到id=101 → COMMIT (T2)
事务C: BEGIN → 拿到id=102 → COMMIT (T3)

id 序列上 T3 时刻的状态: [_, 101, 102]  ← 100 是空洞
```

### UUIDv7 的有序性

| 维度 | 是否有序 | 说明 |
|------|---------|------|
| 跨毫秒 | ✅ 大致有序 | 前 48 bit 是 Unix 毫秒级时间戳（RFC 9562 §5.7） |
| 同毫秒内（单进程） | ⚠️ 取决于实现 | RFC 9562 §6.2 给出多种 monotonic 方法；如 `github.com/google/uuid` v1.6+ 通过计数器实现 **per-process** 单调 |
| 同毫秒内（多实例） | ❌ 不保证 | 不同实例的随机部分互相独立 |
| 与 commit 顺序的关系 | ❌ 不保证 | 时间戳是生成时刻，不是提交时刻 |

### CreatedAt 时间戳的有序性

| 维度 | 是否有序 | 说明 |
|------|---------|------|
| 大致有序 | ✅ | `NOW()` 取**语句开始时间**；默认到秒级，需 `NOW(6)` + `DATETIME(6)`/`TIMESTAMP(6)` 才到微秒 |
| 严格有序 | ❌ | 高并发下即使微秒精度也可能重合；`NOW()` 取**语句开始时间**而非真实墙钟，跨语句不严格单调；也不反映 commit 顺序 |

### 核心结论

> **没有一个常规字段能精确反映事务提交顺序。** AUTO_INCREMENT 反映的是分配顺序，时间戳反映的是语句执行时间，UUIDv7 反映的是应用层生成时间。

---

## 三、需要严格 Commit 顺序怎么办

### 方案：序列表（Commit Sequence）

在同一事务的**最后一步**去序列表抢一个递增号，利用行锁保证全局串行。

```sql
CREATE TABLE commit_sequences (
    seq_name  VARCHAR(64)      NOT NULL PRIMARY KEY,
    seq_val   BIGINT UNSIGNED  NOT NULL DEFAULT 0
) ENGINE=InnoDB;
```

```go
// import "context"; "database/sql"; "fmt"
// 事务最后一步执行，持锁窗口最短
func NextCommitSeq(ctx context.Context, tx *sql.Tx, seqName string) (int64, error) {
    res, err := tx.ExecContext(ctx,
        `UPDATE commit_sequences
            SET seq_val = LAST_INSERT_ID(seq_val + 1)
          WHERE seq_name = ?`, seqName)
    if err != nil {
        return 0, fmt.Errorf("update commit_sequences: %w", err)
    }
    affected, err := res.RowsAffected()
    if err != nil {
        return 0, err
    }
    if affected == 0 {
        // 序列行不存在；不能直接读 LAST_INSERT_ID()，否则会拿到该连接残留值
        return 0, fmt.Errorf("commit_sequences row not found: %q", seqName)
    }

    var seq int64
    if err := tx.QueryRowContext(ctx, `SELECT LAST_INSERT_ID()`).Scan(&seq); err != nil {
        return 0, fmt.Errorf("read last_insert_id: %w", err)
    }
    return seq, nil
}
```

**关键点：**
- `LAST_INSERT_ID(expr)` 的返回值是 **session-scoped**（per connection），必须在**同一连接/同一 `tx`** 内紧接着读取
- `UPDATE` 命中 0 行时不能直接读 `LAST_INSERT_ID()`——它会返回该连接上一次的残留值，造成静默错号
- 行锁持续到事务 commit/rollback。**抢到锁的事务先结束**（成功或回滚），后继事务在锁上排队；最终**成功提交事务的序号顺序 = 提交顺序**
- 回滚事务对应的 `+1` 也会回滚，下一个事务拿到的仍是当前值 +1，序号无空洞
- 放在事务最后一步，最小化持锁窗口
- `seq_val` 声明为 `BIGINT UNSIGNED`，但 Go 端用 `int64` 接收——业务层 TPS 远达不到 `2^63` 上限，可不处理；若担心溢出，改用 `uint64` 或把列改成 `BIGINT`

**代价：**
- 序列表该行成为全局热点，所有写事务排队
- 经验值上限大约几千 TPS
- 绝大多数系统不需要这个级别的保证

### 什么时候需要

| 场景 | 是否需要 |
|------|---------|
| 用户翻页看列表 | ❌ |
| 后台管理面板 | ❌ |
| 金融对账 | ✅ |
| 事件溯源（Event Sourcing） | ✅ |
| 审计日志严格排序 | ✅ |

其他可选方案：消费 binlog（Debezium 等）获取 commit 顺序——前提是**单 MySQL 实例 + `binlog_order_commits=ON`（默认）**。一旦涉及多源、分库分表、Kafka 重分区或读副本，就不再是全局精确提交序，需要在下游重新做归并。

---

## 四、分页查询中的"空洞"问题

### 问题

并发写入时，小 ID 可能在大 ID 之后才提交（空洞），导致基于 cursor 的分页漏数据：

```
T4: 客户端 WHERE id > 99 LIMIT 2 → [101, 102], cursor=102
T5: 事务A commit → id=100 落库
T6: 客户端 WHERE id > 102 → id=100 永远丢失
```

### 解决方案对比

| 方案 | 做法 | 代价 | 适用场景 |
|------|------|------|---------|
| **延迟水位线** | `WHERE created_at < NOW() - INTERVAL 5 SECOND` | 几秒延迟；**等待时长须大于业务允许的最长事务时间**，否则仍会漏 | 大多数系统 |
| **接受最终一致** | 直接用 id 分页，不做特殊处理 | 漏掉所有"在 cursor 前分配、cursor 推进后才提交"的记录 | 用户翻页、后台列表 |
| **双阶段查询** | 先锁定上界，等空洞填满再拉取闭合区间（见下文） | 同步有延迟 | 数据同步、对账 |
| **单队列写入** | 业务 → MQ → 单消费者顺序写 DB | 吞吐受限，写入异步 | 金融级严格有序 |
| **消费 binlog** | Debezium 等 CDC 工具 | 架构复杂度；多源/重分区下不再是全局序（见上节） | 单实例下接近精确 commit 顺序 |

### 双阶段查询详解

适用场景：定时数据同步、对账等批量拉取任务，需要保证**不丢任何一条已提交记录**。

**核心思路：先画一条线，等尘埃落定，再收割线以内的全部数据。**

```
════════ 阶段一：打标记 ════════

T1: 记录当前最大 id（锁定上界）
    SELECT MAX(id) FROM orders            -- 独立短事务/autocommit
    → high_water_mark = 1003

════════ 等待安全窗口 ════════

T2: sleep(W)   -- W 必须 > 业务允许的最长事务时间（含锁等待、慢查询）

    这段窗口内：
    - id=1002 的慢事务提交了 ✅（空洞被填上）
    - id=1004 新事务也提交了（不管，下次再拉）

════════ 阶段二：拉取闭合区间 ════════

T3: SELECT * FROM orders
    WHERE id > 1000            -- 上次同步的 cursor
      AND id <= 1003           -- 阶段一打的标记
    ORDER BY id

    结果: [1001, 1002, 1003]

T4: 更新 cursor = 1003，下次从 1003 继续
```

> ⚠️ **必须使用独立事务执行两个阶段**（或以 `READ COMMITTED` 隔离级别运行）。如果两阶段在同一 `REPEATABLE READ` 事务里，第二阶段读到的仍是 T1 的快照，`sleep` 期间新提交的行根本看不见，方案失效。
>
> ⚠️ **W 的取值是工程权衡**：必须严格大于系统中**最长事务的提交时延**（含 `SET innodb_lock_wait_timeout`、慢查询、长事务监控阈值）。线上一般配合 `information_schema.innodb_trx` 监控最长事务年龄来动态决定，硬编码 5s 不一定够。

与延迟水位线的区别：
- **延迟水位线**：上界是动态模糊的（`NOW() - INTERVAL 5 SECOND`），适合实时分页
- **双阶段查询**：上界是精确确定的（`MAX(id)`），保证闭合区间内不丢不重，适合批量同步

---

## 五、Open API Cursor 分页设计

### Stripe 的做法（基于公开文档）

Stripe 公开文档可证实的部分：

1. List API 默认按对象创建时间**倒序**返回（reverse chronological）
2. 分页 cursor 是**对象 ID** 本身（`starting_after` / `ending_before` 参数）
3. ID 不严格递增，但 cursor 分页可用

**作者推测（非官方）：** 在按 `created` 排序、又允许 ID 非严格递增的前提下，一种合理的内部实现是用 `(created, id)` 复合条件作为 tie-breaker：

```
-- 仅为示意，非 Stripe 公开实现
WHERE (created, id) < (:cursor_created, :cursor_id)
ORDER BY created DESC, id DESC LIMIT 100
```

**Stripe 对严格一致性的态度（公开声明）：** Search API 文档明确说明它最终一致，不要用于读后写场景；普通 list API 也不承诺在并发写入下零丢失。

### 推荐的 Cursor 设计

**对外：opaque token，不暴露内部实现**

```go
// 编码 cursor（可随时更换底层实现）
func EncodeCursor(id int64, createdAt time.Time) string {
    raw := fmt.Sprintf("%d:%d", id, createdAt.UnixMicro())
    return base64.URLEncoding.EncodeToString([]byte(raw))
}
// → "MjM0NTY3OjE3MDk4MjM0NTY3ODkwMDA="

// 解码
func DecodeCursor(cursor string) (id int64, createdAt time.Time) { ... }
```

**对内：根据一致性要求选择策略**

```sql
-- 普通场景：直接用 id（接受微小不一致）
WHERE id < :last_id ORDER BY id DESC LIMIT 20

-- 需要更好一致性：用 created_at 做水位线 + 复合 cursor
-- ⚠️ cutoff 要在「首次发起分页时」算好并随 cursor 带回，不要每页重算 NOW()——
--    每页重算会让"刚跨过 NOW()-5s"的记录在不同请求中"看见/不看见"地切换
WHERE (created_at, id) < (:last_created_at, :last_id)
  AND created_at < :cutoff   -- = first_request_time - INTERVAL 5 SECOND，写入 cursor
ORDER BY created_at DESC, id DESC LIMIT 20

-- 需要按业务时间排序：复合条件
WHERE (occurred_at, id) < (:last_occurred_at, :last_id)
ORDER BY occurred_at DESC, id DESC LIMIT 20
```

---

## 六、一张图总结各字段的定位

```
┌─────────────┬──────────────────┬──────────────────────────────────┐
│   字段       │  谁生成          │  用途                             │
├─────────────┼──────────────────┼──────────────────────────────────┤
│ ID          │ DB AUTO_INC      │ 内部关联、JOIN、分页 cursor 底层   │
│ BizID       │ 应用层 UUIDv7    │ 对外暴露、跨系统引用、API 响应     │
│ OccurredAt  │ 业务层           │ 业务排序（下单时间、支付时间）      │
│ CreatedAt   │ DB DEFAULT       │ 大致入库时间、审计                 │
│ CommitSeq   │ 序列表（可选）    │ 严格 commit 顺序（仅金融级需要）   │
└─────────────┴──────────────────┴──────────────────────────────────┘

对外 API cursor = opaque(ID) 或 opaque(排序字段 + ID)
排序 ≠ cursor，cursor 只是定位指针
```

---

## 七、决策流程

```
Q: 需要对外暴露 ID 吗？
├─ 是 → 用 BizID (prefix + UUIDv7)，不暴露自增主键
└─ 否 → 内部用自增 ID 即可

Q: 分页排序用什么？
├─ 按入库顺序 → 用 ID（最简单，性能最好）
├─ 按业务时间 → 用 (OccurredAt, ID) 复合排序
└─ 按精确提交顺序 → 用 CommitSeq（确认你真的需要）

Q: 能接受几秒延迟吗？
├─ 能 → 加延迟水位线（前提：等待时长 > 业务最长事务时间）
└─ 不能 → 接受可能漏数据，或用序列表/单队列保证强序（binlog CDC 解决的是"精确顺序"，本身也有毫秒~秒级延迟）
```