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

1. **聚合根生成事件**：在状态变更时记录事件
2. **仓储发布事件**：在持久化成功后发布

### 聚合根生成事件

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

### 仓储发布事件

```go
// infrastructure/persistence/mysql/user_repository.go
type UserRepository struct {
    db             *sql.DB
    eventPublisher domain.DomainEventPublisher  // 仓储持有事件发布器
}

func NewUserRepository(db *sql.DB, eventPublisher domain.DomainEventPublisher) *UserRepository {
    return &UserRepository{
        db:             db,
        eventPublisher: eventPublisher,
    }
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
    // 1. 保存到数据库
    _, err := r.db.ExecContext(ctx, `INSERT INTO users ...`)
    if err != nil {
        return err
    }

    // 2. 获取并发布事件（DDD核心原则）
    r.publishEvents(user.PullEvents())

    return nil
}

// publishEvents 发布领域事件（失败不影响主流程，保证最终一致性）
func (r *UserRepository) publishEvents(events []domain.DomainEvent) {
    if r.eventPublisher == nil {
        return
    }
    for _, event := range events {
        if err := r.eventPublisher.Publish(event); err != nil {
            log.Printf("[WARN] Failed to publish event %s: %v", event.EventName(), err)
        }
    }
}
```

---

## 工作单元（Unit of Work）

工作单元模式用于管理事务边界，确保多个聚合操作的一致性。

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

        // 2. 创建订单聚合根
        order, err = domain.NewOrder(req.UserID, req.Items)
        if err != nil {
            return err
        }

        // 3. 注册到工作单元（自动保存）
        s.uow.RegisterNew(order)

        return nil
    })

    // Execute自动处理：
    // - 开始事务（Begin）
    // - 执行操作
    // - 成功：保存聚合、发布事件、提交事务
    // - 失败：回滚事务

    if err != nil {
        return nil, err
    }

    return s.convertToResponse(order), nil
}
```

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
- 在保存后发布聚合根的事件

**不应该做**：
- 暴露底层存储细节（SQL语句等）
- 允许绕过聚合根修改内部实体
- 提供批量操作（如FindAll）
- 包含业务逻辑

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

    // 3. 持久化（仓储会自动发布事件）
    return s.orderRepo.Save(ctx, order)
}
```

### 职责对比

| 特征 | 领域服务 | 应用服务 |
|-----|---------|---------|
| **职责** | 跨实体业务规则验证 | 编排业务流程 |
| **持久化** | 不调用Save | 调用Save |
| **事件发布** | 不负责 | 由仓储自动发布 |
| **调用方** | 应用服务 | 表示层（Controller） |

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

### 事件流转

```
聚合根状态变更 → 记录事件 → 仓储Save → 发布事件 → 事件处理器
```

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
- 工作单元模式 - 事务管理
- 领域事件模式 - 解耦和异步处理
