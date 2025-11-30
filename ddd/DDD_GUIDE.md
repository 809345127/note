# DDD实践指南

本文档是DDD示例项目的实践指南，详细讲解项目中各个DDD模式的正确实现方式。

---

## 项目架构

### 标准DDD分层架构

```
ddd/
├── domain/                 # 领域层（核心层，不依赖任何其他层）
│   ├── user.go             # 用户聚合根
│   ├── order.go            # 订单聚合根
│   ├── value_objects.go    # 值对象
│   ├── services.go         # 领域服务
│   ├── events.go           # 领域事件
│   ├── repositories.go     # 仓储接口
│   ├── aggregate.go        # 聚合标记接口
│   ├── event_publisher.go  # 事件发布接口
│   ├── unit_of_work.go     # 工作单元接口
│   └── tx_unit_of_work.go  # 事务工作单元
├── service/                # 应用层（依赖domain层）
├── api/                    # 表示层（依赖service层）
├── infrastructure/         # 基础设施层（实现domain层定义的接口）
│   └── persistence/
│       ├── mocks/          # Mock仓储实现
│       └── mysql/          # MySQL仓储实现
└── cmd/                    # 应用入口
```

### 依赖方向

```
┌─────────────────────────────────┐
│   表示层 (api)                  │
│   处理HTTP请求/响应             │
└────────────┬────────────────────┘
             │ 依赖
┌────────────▼────────────────────┐
│   应用层 (service)              │
│   编排业务流程、使用UnitOfWork  │
└────────────┬────────────────────┘
             │ 依赖
┌────────────▼────────────────────┐
│   领域层 (domain)               │  ◄─ 核心层（纯净，不依赖任何层）
│   业务逻辑、实体、聚合根        │
└────────────┬────────────────────┘
             │ 依赖倒置（通过接口）
┌────────────▼────────────────────┐
│   基础设施层 (infrastructure)   │
│   技术实现（仓储、事件发布）    │
└─────────────────────────────────┘
```

**核心原则**：领域层是项目的核心，它不依赖任何框架或技术实现。基础设施层通过实现领域层定义的接口来提供具体实现。

---

## 聚合根（Aggregate Root）

聚合根是DDD的核心概念，它定义了一组相关对象的一致性边界。

### Order聚合根示例

```go
// domain/order.go
type Order struct {
    id          string
    userID      string
    items       []OrderItem           // 聚合内部实体（私有）
    totalAmount Money
    status      OrderStatus
    version     int                   // 乐观锁版本号
    createdAt   time.Time
    updatedAt   time.Time
    events      []DomainEvent         // 领域事件列表
}

// 聚合内部实体（非聚合根，只能通过Order访问）
type OrderItem struct {
    id          string  // 仅在聚合内唯一
    productID   string
    productName string
    quantity    int
    unitPrice   Money
    subtotal    Money
}
```

### 聚合边界保护

所有对聚合内部实体的修改必须通过聚合根进行：

```go
// 通过聚合根方法添加订单项
func (o *Order) AddItem(productID, productName string, quantity int, unitPrice Money) error {
    // 1. 验证聚合不变量
    if o.status != OrderStatusPending {
        return errors.New("can only add items to pending orders")
    }

    // 2. 创建聚合内部实体
    item := OrderItem{
        id:          uuid.New().String(),
        productID:   productID,
        productName: productName,
        quantity:    quantity,
        unitPrice:   unitPrice,
        subtotal:    *NewMoney(unitPrice.Amount() * int64(quantity), unitPrice.Currency()),
    }

    o.items = append(o.items, item)

    // 3. 维护聚合一致性
    o.recalculateTotalAmount()
    o.updatedAt = time.Now()

    return nil
}

// Items()返回副本，防止外部直接修改
func (o *Order) Items() []OrderItem {
    items := make([]OrderItem, len(o.items))
    copy(items, o.items)
    return items
}
```

### 聚合标记接口

```go
// domain/aggregate.go
type AggregateRoot interface {
    ID() string
    Version() int
    PullEvents() []DomainEvent  // 获取并清空事件
}

// 编译时验证
var _ = IsAggregateRoot(&User{})
var _ = IsAggregateRoot(&Order{})
```

---

## 领域事件（Domain Event）

领域事件记录业务系统中发生的重要事件，用于解耦和异步处理。

### 核心原则

1. **聚合根产生事件**：在状态变更时记录事件到内存
2. **UoW 统一保存事件**：在事务提交前，将事件与业务数据一起落库（Outbox Pattern）
3. **事务提交后发布**：后台进程异步读取 outbox 表发布到消息队列

```
┌─────────────────────────────────────────────────────────┐
│  UnitOfWork.Execute()                                   │
│  ┌───────────────────────────────────────────────────┐  │
│  │ BEGIN TRANSACTION                                 │  │
│  │   repo.Save(order)    → 保存聚合根数据            │  │
│  │   outbox.Save(events) → 保存事件到outbox表        │  │
│  │ COMMIT                                            │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  后台进程轮询 outbox 表 → 发布到消息队列 → 删除已发送记录 │
└─────────────────────────────────────────────────────────┘
```

### 聚合根产生事件

```go
// domain/order.go
func NewOrder(userID string, requests []OrderItemRequest) (*Order, error) {
    order := &Order{
        id:        uuid.New().String(),
        userID:    userID,
        status:    OrderStatusPending,
        events:    make([]DomainEvent, 0),
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }

    // ... 添加订单项

    // 记录领域事件
    order.events = append(order.events, NewOrderPlacedEvent(order.id, userID, order.totalAmount))

    return order, nil
}

// 状态变更时记录事件
func (o *Order) Confirm() error {
    if o.status != OrderStatusPending {
        return errors.New("only pending orders can be confirmed")
    }
    o.status = OrderStatusConfirmed
    o.updatedAt = time.Now()
    o.version++

    // 记录事件
    o.events = append(o.events, NewOrderConfirmedEvent(o.id))
    return nil
}

// 获取并清空事件（仅供仓储调用）
func (o *Order) PullEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}
```

### 为什么不在仓储中直接发布事件？

**错误示例（不要这样做）：**

```go
func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
    _, err := r.db.ExecContext(ctx, `INSERT INTO users ...`)
    if err != nil {
        return err
    }
    // ❌ 错误：事务可能还未提交，或后续操作失败导致回滚
    // 但事件已经发出去了，造成数据不一致！
    r.eventPublisher.Publish(user.PullEvents())
    return nil
}
```

**问题：** 事务是在应用服务层（通过 UoW）管理的，仓储只是事务中的一环。如果在 repo.Save() 中直接发布事件，可能出现：
1. 事务还没提交，事件就发出去了
2. 后续操作失败导致事务回滚，但事件已经被消费

**正确做法：** 由 UoW 在事务提交前统一保存事件到 outbox 表，事务提交后由后台进程异步发布。

---

## 工作单元（Unit of Work）

工作单元模式用于管理事务边界，确保多个聚合操作的一致性，并统一处理领域事件的持久化。

### 接口定义

```go
// domain/unit_of_work.go
type UnitOfWork interface {
    Execute(fn func() error) error  // 自动管理事务
    RegisterNew(aggregate AggregateRoot)
    RegisterDirty(aggregate AggregateRoot)
    RegisterRemoved(aggregate AggregateRoot)
}
```

### UoW 实现（含 Outbox Pattern）

```go
// infrastructure/unit_of_work.go
type UnitOfWorkImpl struct {
    db         *sql.DB
    tx         *sql.Tx
    aggregates []domain.AggregateRoot
}

func (uow *UnitOfWorkImpl) Execute(fn func() error) error {
    // 1. 开启事务
    tx, err := uow.db.Begin()
    if err != nil {
        return err
    }
    uow.tx = tx
    uow.aggregates = nil

    // 2. 执行业务逻辑
    if err := fn(); err != nil {
        tx.Rollback()
        return err
    }

    // 3. 收集所有聚合根的事件，保存到 outbox 表（同一事务）
    for _, agg := range uow.aggregates {
        events := agg.PullEvents()
        for _, event := range events {
            payload, _ := json.Marshal(event)
            _, err := tx.Exec(
                "INSERT INTO outbox (event_type, aggregate_id, payload, created_at) VALUES (?, ?, ?, NOW())",
                event.EventName(), agg.ID(), payload,
            )
            if err != nil {
                tx.Rollback()
                return err
            }
        }
    }

    // 4. 提交事务（业务数据 + 事件一起提交，保证原子性）
    return tx.Commit()
}

func (uow *UnitOfWorkImpl) RegisterNew(agg domain.AggregateRoot) {
    uow.aggregates = append(uow.aggregates, agg)
}
```

### 应用层使用示例

```go
// service/order_service.go
func (s *OrderApplicationService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
    var order *domain.Order

    // 使用工作单元管理事务
    err := s.uow.Execute(func() error {
        // 1. 验证用户是否可以下单
        user, err := s.userRepo.FindByID(req.UserID)
        if err != nil {
            return err
        }

        if !user.CanMakePurchase() {
            return errors.New("user cannot make purchases")
        }

        // 2. 创建订单聚合根（聚合根内部记录事件）
        order, err = domain.NewOrder(req.UserID, req.Items)
        if err != nil {
            return err
        }

        // 3. 保存聚合根
        if err := s.orderRepo.Save(order); err != nil {
            return err
        }

        // 4. 注册到工作单元（UoW 会在提交前收集事件）
        s.uow.RegisterNew(order)

        return nil
    })

    // Execute 自动处理：
    // - 开始事务（Begin）
    // - 执行业务操作
    // - 收集聚合根事件，保存到 outbox 表
    // - 提交事务（业务数据 + 事件原子提交）
    // - 失败则回滚事务

    if err != nil {
        return nil, err
    }

    return s.convertToResponse(order), nil
}
```

### Outbox 表结构

```sql
CREATE TABLE outbox (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(36) NOT NULL,
    payload JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    published_at TIMESTAMP NULL,
    INDEX idx_unpublished (published_at, created_at)
);
```

### 事件发布：Message Relay（独立后台服务）

**重要：Application Service 不负责 Publish Event，只负责 Save Event 到 outbox 表。**

实际发布由独立的后台服务（Message Relay）完成，常见实现方式：

| 方式 | 延迟 | 复杂度 | 说明 |
|-----|------|-------|------|
| Polling（轮询） | 秒级 | 低 | 简单但有延迟 |
| CDC（Change Data Capture） | 毫秒级 | 高 | 推荐，如 Debezium |
| JIT Polling | 可控 | 中 | 混合方式，兼顾延迟和简单性 |

#### JIT Polling 实现（推荐）

```go
// infrastructure/outbox_processor.go
type OutboxProcessor struct {
    db           *sql.DB
    messageQueue MessageQueue
    triggerCh    chan struct{}  // 用于接收立即处理通知
}

func NewOutboxProcessor(db *sql.DB, mq MessageQueue) *OutboxProcessor {
    return &OutboxProcessor{
        db:           db,
        messageQueue: mq,
        triggerCh:    make(chan struct{}, 1),  // 带缓冲，避免阻塞
    }
}

// NotifyNewMessages 通知处理器有新消息，可立即处理（非阻塞）
func (p *OutboxProcessor) NotifyNewMessages() {
    select {
    case p.triggerCh <- struct{}{}:
    default:  // 已有通知在等待，无需重复
    }
}

// Run 启动后台处理循环
func (p *OutboxProcessor) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)  // 定时兜底
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-p.triggerCh:    // 立即触发
            p.processOutbox()
        case <-ticker.C:       // 定时兜底（防止通知丢失）
            p.processOutbox()
        }
    }
}

func (p *OutboxProcessor) processOutbox() {
    rows, err := p.db.Query(
        "SELECT id, event_type, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at LIMIT 100",
    )
    if err != nil {
        log.Printf("Failed to query outbox: %v", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var eventType string
        var payload []byte
        if err := rows.Scan(&id, &eventType, &payload); err != nil {
            continue
        }

        // 发布到消息队列
        if err := p.messageQueue.Publish(eventType, payload); err != nil {
            log.Printf("Failed to publish event %d: %v", id, err)
            continue
        }

        // 标记为已发布
        p.db.Exec("UPDATE outbox SET published_at = NOW() WHERE id = ?", id)
    }
}
```

#### 应用服务配合 JIT Polling

```go
// service/order_service.go
type OrderApplicationService struct {
    uow             domain.UnitOfWork
    orderRepo       domain.OrderRepository
    outboxProcessor *infrastructure.OutboxProcessor  // 持有处理器引用
}

func (s *OrderApplicationService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
    var order *domain.Order

    err := s.uow.Execute(func() error {
        // ... 业务逻辑（创建订单、保存聚合根）
        // UoW 会在事务中保存事件到 outbox 表
        return nil
    })

    if err != nil {
        return nil, err
    }

    // 事务成功后，通知处理器有新消息（非阻塞，可选）
    // 注意：这里只是"通知"，不是直接发布
    s.outboxProcessor.NotifyNewMessages()

    return s.convertToResponse(order), nil
}
```

### 职责总结

```
┌─────────────────────────────────────────────────────────────────────┐
│  角色                      │  职责                                  │
├─────────────────────────────────────────────────────────────────────┤
│  聚合根                    │  产生事件，暂存在内存                   │
│  UoW                       │  事务管理，Save Event 到 outbox 表     │
│  Application Service       │  业务编排，可通知处理器（不直接发布）   │
│  Message Relay（独立进程） │  从 outbox 读取，Publish 到消息队列    │
└─────────────────────────────────────────────────────────────────────┘
```

**核心原则：Application Service 不直接 Publish Event。**

---

## 仓储（Repository）

仓储提供聚合根的持久化抽象。

### 接口设计原则

```go
// domain/repositories.go
type OrderRepository interface {
    // ID生成
    NextIdentity() string

    // 基本操作（聚合根级别）
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
    Remove(ctx context.Context, id string) error  // 逻辑删除

    // 受控查询（限定范围）
    FindByUserID(ctx context.Context, userID string) ([]*Order, error)
}
```

### 仓储职责

**应该做**：
- 只持久化聚合根（整个聚合一起保存）
- 提供领域语义的查询接口
- 保证聚合的原子性操作

**不应该做**：
- 暴露底层存储细节（SQL语句等）
- 允许绕过聚合根修改内部实体
- 提供批量操作（如FindAll）
- 包含业务逻辑
- 直接发布领域事件（这是 UoW 的职责）

### 逻辑删除

DDD推荐使用逻辑删除，保留业务历史：

```go
// infrastructure/persistence/mysql/order_repository.go
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
    // 逻辑删除：标记为已取消
    _, err := r.db.ExecContext(ctx, `
        UPDATE orders SET status = ?, updated_at = NOW() WHERE id = ?
    `, string(domain.OrderStatusCancelled), id)
    return err
}
```

---

## 领域服务 vs 应用服务

### 领域服务

处理跨实体的业务规则，**不负责持久化和事件发布**：

```go
// domain/services.go
type OrderDomainService struct {
    userRepository  UserRepository
    orderRepository OrderRepository
}

// 只做业务规则验证，返回验证结果
func (s *OrderDomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
    order, err := s.orderRepository.FindByID(ctx, orderID)
    if err != nil {
        return nil, err
    }

    user, err := s.userRepository.FindByID(ctx, order.UserID())
    if err != nil {
        return nil, err
    }

    // 跨实体业务规则验证
    if !user.IsActive() {
        return nil, ErrUserNotActive
    }
    if order.Status() != OrderStatusPending {
        return nil, errors.New("only pending orders can be processed")
    }

    return order, nil
}
```

### 应用服务

编排业务流程，**负责调用Save**：

```go
// service/order_service.go
type OrderApplicationService struct {
    orderRepo          domain.OrderRepository
    orderDomainService *domain.OrderDomainService
}

func (s *OrderApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
    // 1. 通过领域服务验证
    order, err := s.orderDomainService.CanProcessOrder(ctx, orderID)
    if err != nil {
        return err
    }

    // 2. 修改聚合根状态
    if err := order.Confirm(); err != nil {
        return err
    }

    // 3. 持久化（仓储只负责持久化，事件由 UoW 保存到 outbox 表）
    return s.orderRepo.Save(ctx, order)
}
```

### 职责对比

| 特征       | 领域服务             | 应用服务                       |
|-----------|---------------------|-------------------------------|
| **职责**   | 跨实体业务规则验证    | 编排业务流程                   |
| **持久化** | 不调用 Save          | 调用 Save                      |
| **事件处理** | 不负责             | 通过 UoW 保存事件到 outbox 表   |
| **调用方** | 应用服务             | 表示层（Controller）           |

---

## 值对象（Value Object）

值对象是不可变的，通过值而非身份来标识。

### Money值对象

```go
// domain/value_objects.go
type Money struct {
    amount   int64   // 以分为单位
    currency string
}

func NewMoney(amount int64, currency string) *Money {
    return &Money{amount: amount, currency: currency}
}

func (m Money) Amount() int64     { return m.amount }
func (m Money) Currency() string  { return m.currency }

// 值对象不可变，操作返回新实例
func (m Money) Add(other Money) (*Money, error) {
    if m.currency != other.currency {
        return nil, errors.New("currency mismatch")
    }
    return NewMoney(m.amount + other.amount, m.currency), nil
}
```

### Email值对象

```go
type Email struct {
    value string
}

func NewEmail(value string) (*Email, error) {
    // 验证邮箱格式
    if !isValidEmail(value) {
        return nil, errors.New("invalid email format")
    }
    return &Email{value: value}, nil
}

func (e Email) Value() string { return e.value }
```

---

## 最佳实践总结

### 分层职责

| 层 | 职责 | 依赖 |
|---|-----|-----|
| 表示层 | HTTP请求处理、参数验证 | 应用层 |
| 应用层 | 业务流程编排、调用Save | 领域层 |
| 领域层 | 业务逻辑、聚合根、领域服务 | 无依赖 |
| 基础设施层 | 技术实现、仓储、事件发布 | 领域层接口 |

### 事件流转（Outbox Pattern）

```
聚合根状态变更 → 记录事件到内存
                    ↓
              UoW.Execute()
                    ↓
    ┌───────────────────────────────┐
    │ BEGIN TRANSACTION             │
    │   repo.Save(聚合根)           │
    │   outbox.Save(事件)           │
    │ COMMIT                        │
    └───────────────────────────────┘
                    ↓
        后台进程轮询 outbox 表
                    ↓
        发布到消息队列 → 事件处理器
```

**关键点：**
- 聚合根产生事件，UoW 统一保存
- 事件与业务数据同事务落库，保证原子性
- 后台进程异步发布，保证最终一致性

### 代码质量检查

1. **领域类不依赖框架**
2. **无法绕过聚合根修改内部实体**
3. **业务规则在实体内部**
4. **仓储接口精炼**

---

## 进阶主题

### CQRS模式

分离命令（写）和查询（读）模型：

```go
// 命令模型：通过聚合根
orderRepo.Save(order)

// 查询模型：专门的查询服务
type OrderQueryService interface {
    SearchOrders(criteria OrderSearchCriteria) ([]OrderDTO, error)
}
```

### 事件溯源

通过事件序列重建聚合状态：

```go
type EventStore interface {
    SaveEvents(aggregateID string, events []DomainEvent, version int) error
    LoadEvents(aggregateID string) ([]DomainEvent, error)
}
```

---

## 参考资料

### 推荐书籍

1. **《领域驱动设计》** - Eric Evans（DDD开山之作）
2. **《实现领域驱动设计》** - Vaughn Vernon（实践指南，强烈推荐）
3. **《领域驱动设计精粹》** - Vaughn Vernon（快速入门）

### 设计模式

- 聚合模式 - 维护一致性边界
- 仓储模式 - 持久化抽象
- 工作单元模式 - 事务管理 + 事件收集
- 领域事件模式 - 解耦和异步处理
- **Outbox Pattern** - 事件与数据原子性落库，后台异步发布
