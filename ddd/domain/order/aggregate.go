/*
Package order 订单子域 - DDD架构的核心层

领域层是整个应用的核心，包含：
- 聚合根（Aggregate Root）：维护一致性边界的实体
- 实体（Entity）：具有唯一标识的对象
- 值对象（Value Object）：通过属性值标识的不可变对象
- 领域服务（Domain Service）：跨实体的业务逻辑
- 领域事件（Domain Event）：记录业务系统中发生的重要事件
- 仓储接口（Repository Interface）：聚合根持久化的抽象

DDD核心原则：
1. 领域层不依赖任何其他层（纯净的业务逻辑）
2. 所有字段私有，通过方法暴露行为
3. 业务规则封装在实体和值对象内部
*/
package order

import (
	"errors"
	"time"

	"ddd-example/domain/shared"

	"github.com/google/uuid"
)

// Order 订单聚合根
// Order作为聚合根，维护订单的一致性边界
// 所有对Order和OrderItem的修改都必须通过Order聚合根进行
type Order struct {
	id          string
	userID      string
	items       []OrderItem
	totalAmount shared.Money
	status      Status
	version     int // 乐观锁版本号，用于并发控制
	createdAt   time.Time
	updatedAt   time.Time

	// 领域事件列表，用于记录聚合内发生的领域事件
	events []shared.DomainEvent
}

// OrderItem 订单项 - 聚合内的实体（非聚合根）
// OrderItem是聚合的一部分，没有全局唯一标识，只能通过Order访问
type OrderItem struct {
	id          string // OrderItem在聚合内的唯一标识
	productID   string
	productName string
	quantity    int
	unitPrice   shared.Money
	subtotal    shared.Money
}

// Status 订单状态枚举
type Status string

const (
	StatusPending   Status = "PENDING"   // 待处理
	StatusConfirmed Status = "CONFIRMED" // 已确认
	StatusShipped   Status = "SHIPPED"   // 已发货
	StatusDelivered Status = "DELIVERED" // 已送达
	StatusCancelled Status = "CANCELLED" // 已取消
)

// PostOptions 创建订单的选项
type PostOptions struct {
	UserID string
	Items  []ItemRequest
}

// ItemRequest 创建订单项的请求
type ItemRequest struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
}

// ============================================================================
// 工厂方法 - 创建聚合根
// ============================================================================
//
// DDD原则：使用工厂方法创建聚合根，而非直接使用struct字面量
// 优点：
// 1. 封装创建逻辑和验证规则
// 2. 确保聚合根创建时处于有效状态
// 3. 可以在创建时记录领域事件

// NewOrder 创建新的Order聚合根
// 这是创建Order的唯一入口，确保订单创建时满足所有业务规则
func NewOrder(userID string, requests []ItemRequest) (*Order, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if len(requests) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	// 创建订单项
	items := make([]OrderItem, len(requests))
	for i, req := range requests {
		if req.Quantity <= 0 {
			return nil, errors.New("quantity must be positive")
		}

		items[i] = OrderItem{
			id:          uuid.New().String(),
			productID:   req.ProductID,
			productName: req.ProductName,
			quantity:    req.Quantity,
			unitPrice:   req.UnitPrice,
			subtotal:    *shared.NewMoney(req.UnitPrice.Amount()*int64(req.Quantity), req.UnitPrice.Currency()),
		}
	}

	// 计算总金额
	totalAmount := shared.NewMoney(0, "CNY")
	var err error
	for _, item := range items {
		totalAmount, err = totalAmount.Add(item.subtotal)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	order := &Order{
		id:          uuid.New().String(),
		userID:      userID,
		items:       items,
		totalAmount: *totalAmount,
		status:      StatusPending,
		version:     0,
		createdAt:   now,
		updatedAt:   now,
		events:      make([]shared.DomainEvent, 0),
	}

	// 记录领域事件
	order.events = append(order.events, NewOrderPlacedEvent(order.id, userID, order.totalAmount))

	return order, nil
}

// ============================================================================
// 重建DTO - 仅供仓储层使用
// ============================================================================
//
// DDD原则：聚合根从数据库重建时需要特殊处理
// 由于字段是私有的，仓储层需要一种方式来重建聚合根
// 使用DTO + 工厂方法模式，而非暴露setter或使用反射

// ReconstructionDTO 订单重建数据传输对象
// 仅限于仓储层使用，用于从数据库重建Order聚合根
// 这是一个特殊的设计，保持了领域模型的封装性
// ⚠️ 注意：此DTO仅应在仓储实现中使用，不应在应用层调用
type ReconstructionDTO struct {
	ID          string
	UserID      string
	Items       []OrderItem
	TotalAmount shared.Money
	Status      Status
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RebuildFromDTO 从DTO重建Order聚合根
// 这是一个工厂方法，专门用于仓储层重建聚合根
// ⚠️ 注意：此方法仅应在仓储实现中使用，不应在应用层调用
func RebuildFromDTO(dto ReconstructionDTO) *Order {
	return &Order{
		id:          dto.ID,
		userID:      dto.UserID,
		items:       dto.Items,
		totalAmount: dto.TotalAmount,
		status:      dto.Status,
		version:     dto.Version,
		createdAt:   dto.CreatedAt,
		updatedAt:   dto.UpdatedAt,
		events:      []shared.DomainEvent{},
	}
}

// ItemReconstructionDTO 订单项重建数据传输对象
type ItemReconstructionDTO struct {
	ID          string
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
	Subtotal    shared.Money
}

// RebuildItemFromDTO 从DTO重建OrderItem
func RebuildItemFromDTO(dto ItemReconstructionDTO) OrderItem {
	return OrderItem{
		id:          dto.ID,
		productID:   dto.ProductID,
		productName: dto.ProductName,
		quantity:    dto.Quantity,
		unitPrice:   dto.UnitPrice,
		subtotal:    dto.Subtotal,
	}
}

// ============================================================================
// 聚合根行为方法 - 管理聚合内实体
// ============================================================================
//
// DDD原则：聚合内的实体（OrderItem）只能通过聚合根（Order）操作
// 外部代码无法直接创建或修改OrderItem

// AddItem 通过聚合根添加订单项
// 这是DDD的重要原则：聚合内的实体只能通过聚合根访问和修改
// 参数productID, productName string, quantity int, unitPrice shared.Money
func (o *Order) AddItem(productID, productName string, quantity int, unitPrice shared.Money) error {
	// 验证当前状态是否允许修改
	if o.status != StatusPending {
		return errors.New("can only add items to pending orders")
	}

	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	// 创建新的订单项
	item := OrderItem{
		id:          uuid.New().String(),
		productID:   productID,
		productName: productName,
		quantity:    quantity,
		unitPrice:   unitPrice,
		subtotal:    *shared.NewMoney(unitPrice.Amount()*int64(quantity), unitPrice.Currency()),
	}

	o.items = append(o.items, item)

	// 重新计算总金额
	newTotal := shared.NewMoney(0, "CNY")
	var err error
	for _, it := range o.items {
		newTotal, err = newTotal.Add(it.subtotal)
		if err != nil {
			// 回滚添加操作
			o.items = o.items[:len(o.items)-1]
			return err
		}
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()

	return nil
}

// RemoveItem 通过聚合根删除订单项
func (o *Order) RemoveItem(itemID string) error {
	if o.status != StatusPending {
		return errors.New("can only remove items from pending orders")
	}

	// 查找并删除订单项
	found := false
	for i, item := range o.items {
		if item.id == itemID {
			// 从切片中删除元素
			o.items = append(o.items[:i], o.items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return errors.New("item not found")
	}

	// 重新计算总金额
	newTotal := shared.NewMoney(0, "CNY")
	for _, item := range o.items {
		newTotal, _ = newTotal.Add(item.subtotal)
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()

	return nil
}

// ============================================================================
// 状态变更方法 - 领域行为
// ============================================================================
//
// DDD原则：状态变更必须通过聚合根的方法进行，而非直接修改字段
// 这样可以：
// 1. 封装业务规则（如状态转换限制）
// 2. 自动维护版本号（乐观锁）
// 3. 记录领域事件
// 4. 保证聚合内部一致性

// Confirm 确认订单（状态从PENDING -> CONFIRMED）
// 业务规则：只有待处理的订单才能被确认
func (o *Order) Confirm() error {
	if o.status != StatusPending {
		return errors.New("only pending orders can be confirmed")
	}

	o.status = StatusConfirmed
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Cancel 取消订单
// 业务规则：已送达或已取消的订单不能再取消
func (o *Order) Cancel() error {
	if o.status == StatusDelivered || o.status == StatusCancelled {
		return errors.New("cannot cancel delivered or cancelled orders")
	}

	o.status = StatusCancelled
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Ship 发货（状态从CONFIRMED -> SHIPPED）
// 业务规则：只有已确认的订单才能发货
func (o *Order) Ship() error {
	if o.status != StatusConfirmed {
		return errors.New("only confirmed orders can be shipped")
	}

	o.status = StatusShipped
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Deliver 送达（状态从SHIPPED -> DELIVERED）
// 业务规则：只有已发货的订单才能标记为送达
func (o *Order) Deliver() error {
	if o.status != StatusShipped {
		return errors.New("only shipped orders can be delivered")
	}

	o.status = StatusDelivered
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// ============================================================================
// Getters - 只读访问器
// ============================================================================
//
// DDD原则：字段私有，通过getter暴露只读访问
// 这样外部只能读取状态，不能直接修改，保证了封装性

func (o *Order) ID() string     { return o.id }
func (o *Order) UserID() string { return o.userID }

// Items 返回订单项的副本
// DDD原则：聚合内部实体不能被外部直接修改，返回副本保证封装性
func (o *Order) Items() []OrderItem {
	items := make([]OrderItem, len(o.items))
	copy(items, o.items)
	return items
}
func (o *Order) TotalAmount() shared.Money { return o.totalAmount }
func (o *Order) Status() Status            { return o.status }
func (o *Order) Version() int              { return o.version }
func (o *Order) CreatedAt() time.Time      { return o.createdAt }
func (o *Order) UpdatedAt() time.Time      { return o.updatedAt }

// ============================================================================
// 领域事件管理
// ============================================================================
//
// DDD原则：聚合根负责记录领域事件，UoW 负责保存到 outbox 表
// 事件流程：聚合状态变更 → 记录事件 → UoW 收集 → 保存到 outbox → Message Relay 异步发布

// PullEvents 获取并清空聚合根的事件列表
// 这是领域事件模式的标准实践：
// 1. 聚合根在状态变更时调用 recordEvent() 记录事件
// 2. UoW 在事务中调用 PullEvents() 获取事件并保存到 outbox 表
// 3. PullEvents 会清空事件列表，避免重复保存
func (o *Order) PullEvents() []shared.DomainEvent {
	events := make([]shared.DomainEvent, len(o.events))
	copy(events, o.events)
	o.events = make([]shared.DomainEvent, 0)
	return events
}

// recordEvent 记录领域事件
func (o *Order) recordEvent(event shared.DomainEvent) {
	o.events = append(o.events, event)
}

// OrderItem Getters - 允许读取，但不提供外部修改

func (item OrderItem) ID() string          { return item.id }
func (item OrderItem) ProductID() string   { return item.productID }
func (item OrderItem) ProductName() string { return item.productName }
func (item OrderItem) Quantity() int       { return item.quantity }
func (item OrderItem) UnitPrice() shared.Money    { return item.unitPrice }
func (item OrderItem) Subtotal() shared.Money     { return item.subtotal }

// 编译时检查 Order 实现了 AggregateRoot 接口
var _ shared.AggregateRoot = (*Order)(nil)
