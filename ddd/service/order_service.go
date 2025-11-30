/*
Package service 应用层 - 业务流程编排

应用层的职责：
1. 接收外部请求（通常来自Controller）
2. 调用领域服务进行业务规则验证
3. 调用聚合根方法执行业务操作
4. 调用仓储保存聚合根（UoW 会将事件保存到 outbox 表）
5. 返回结果给调用方

重要：应用服务不直接发布事件！
- 事件由 UoW 在事务提交前保存到 outbox 表
- 后台 OutboxProcessor 异步读取 outbox 表并发布到消息队列
- 这保证了事件与业务数据的原子性

应用服务 vs 领域服务：
┌─────────────────────────────────────────────────────────────────┐
│ 应用服务 (Application Service)                                   │
│ - 编排业务流程（调用多个领域对象和服务）                             │
│ - 负责调用Save持久化                                              │
│ - 管理事务边界（通过UnitOfWork或数据库事务）                         │
│ - 不包含业务规则（业务规则在领域层）                                 │
│ - 不直接发布事件（事件由 UoW 保存到 outbox）                        │
├─────────────────────────────────────────────────────────────────┤
│ 领域服务 (Domain Service)                                        │
│ - 处理跨实体的业务规则                                            │
│ - 不负责持久化（不调用Save）                                       │
│ - 被应用服务调用                                                  │
└─────────────────────────────────────────────────────────────────┘
*/
package service

import (
	"context"
	"errors"
	"time"

	"ddd-example/domain"
)

// ============================================================================
// 订单应用服务
// ============================================================================

// OrderApplicationService 订单应用服务 - 协调订单相关的业务流程
type OrderApplicationService struct {
	orderRepo          domain.OrderRepository
	userRepo           domain.UserRepository
	orderDomainService *domain.OrderDomainService
	userDomainService  *domain.UserDomainService
	eventPublisher     domain.DomainEventPublisher
}

// NewOrderApplicationService 创建订单应用服务
func NewOrderApplicationService(
	orderRepo domain.OrderRepository,
	userRepo domain.UserRepository,
	eventPublisher domain.DomainEventPublisher,
) *OrderApplicationService {
	return &OrderApplicationService{
		orderRepo:          orderRepo,
		userRepo:           userRepo,
		orderDomainService: domain.NewOrderDomainService(userRepo, orderRepo),
		userDomainService:  domain.NewUserDomainService(userRepo, orderRepo),
		eventPublisher:     eventPublisher,
	}
}

// ============================================================================
// DTO定义 - 数据传输对象
// ============================================================================
//
// DDD原则：DTO用于层间数据传输，与领域模型分离
// - Request DTO：接收外部输入
// - Response DTO：返回给外部调用方
// 这样可以：
// 1. 保护领域模型不被外部直接访问
// 2. 灵活调整API结构而不影响领域模型
// 3. 添加特定于展示层的验证和格式化

// CreateOrderRequest 创建订单请求DTO
type CreateOrderRequest struct {
	UserID string             `json:"user_id" binding:"required"`
	Items  []OrderItemRequest `json:"items" binding:"required,min=1"`
}

// OrderItemRequest 订单项请求DTO
type OrderItemRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	ProductName string `json:"product_name" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	UnitPrice   int64  `json:"unit_price" binding:"required,min=0"`
	Currency    string `json:"currency" binding:"required"`
}

// OrderResponse 订单响应DTO
type OrderResponse struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Items       []OrderItemResponse `json:"items"`
	TotalAmount MoneyResponse       `json:"total_amount"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// OrderItemResponse 订单项响应DTO
type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	ProductName string        `json:"product_name"`
	Quantity    int           `json:"quantity"`
	UnitPrice   MoneyResponse `json:"unit_price"`
	Subtotal    MoneyResponse `json:"subtotal"`
}

// MoneyResponse 金额响应DTO
type MoneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// ============================================================================
// 应用服务方法 - 业务流程编排
// ============================================================================

// CreateOrder 创建订单
// 应用服务方法的典型流程：
// 1. 调用领域服务验证业务规则
// 2. 转换DTO为领域对象
// 3. 调用聚合根工厂方法创建实体
// 4. 调用仓储Save保存（UoW 会将事件保存到 outbox 表）
// 5. 转换领域对象为Response DTO返回
// 注意：应用服务不直接发布事件，事件由后台 OutboxProcessor 异步发布
func (s *OrderApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	// 检查用户是否可以下单
	canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !canPlaceOrder {
		return nil, errors.New("user cannot place order")
	}

	// 转换订单项请求为领域模型
	requests := make([]domain.OrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		unitPrice := domain.NewMoney(item.UnitPrice, item.Currency)
		requests[i] = domain.OrderItemRequest{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   *unitPrice,
		}
	}

	// 创建订单实体
	order, err := domain.NewOrder(req.UserID, requests)
	if err != nil {
		return nil, err
	}

	// 保存订单
	if err := s.orderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	// 注意：不在这里发布事件！
	// 事件已由 UoW 保存到 outbox 表，后台 OutboxProcessor 会异步发布

	return s.convertToResponse(order), nil
}

// GetOrder 获取订单信息
func (s *OrderApplicationService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(order), nil
}

// GetUserOrders 获取用户的所有订单
func (s *OrderApplicationService) GetUserOrders(ctx context.Context, userID string) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = s.convertToResponse(order)
	}

	return responses, nil
}

// UpdateOrderStatusRequest 更新订单状态请求DTO
type UpdateOrderStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=PENDING CONFIRMED SHIPPED DELIVERED CANCELLED"`
}

// UpdateOrderStatus 更新订单状态
func (s *OrderApplicationService) UpdateOrderStatus(ctx context.Context, req UpdateOrderStatusRequest) error {
	order, err := s.orderRepo.FindByID(ctx, req.OrderID)
	if err != nil {
		return err
	}

	// 根据请求的状态更新订单
	switch domain.OrderStatus(req.Status) {
	case domain.OrderStatusPending:
		// 不需要任何操作
	case domain.OrderStatusConfirmed:
		if err := order.Confirm(); err != nil {
			return err
		}
	case domain.OrderStatusShipped:
		if err := order.Ship(); err != nil {
			return err
		}
	case domain.OrderStatusDelivered:
		if err := order.Deliver(); err != nil {
			return err
		}
	case domain.OrderStatusCancelled:
		if err := order.Cancel(); err != nil {
			return err
		}
	default:
		return errors.New("invalid order status")
	}

	return s.orderRepo.Save(ctx, order)
}

// ProcessOrder 处理订单
// DDD原则：应用服务负责编排流程（调用领域服务验证、修改状态、保存）
// 注意：应用服务不直接发布事件
func (s *OrderApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
	// 1. 通过领域服务验证订单是否可以处理
	order, err := s.orderDomainService.CanProcessOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. 执行状态变更（聚合根方法，内部会记录事件）
	if err := order.Confirm(); err != nil {
		return err
	}

	// 3. 保存（UoW 会将事件保存到 outbox 表，后台异步发布）
	return s.orderRepo.Save(ctx, order)
}

// convertToResponse 将订单实体转换为响应DTO
func (s *OrderApplicationService) convertToResponse(order *domain.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(order.Items()))
	for i, item := range order.Items() {
		items[i] = OrderItemResponse{
			ProductID:   item.ProductID(),
			ProductName: item.ProductName(),
			Quantity:    item.Quantity(),
			UnitPrice: MoneyResponse{
				Amount:   item.UnitPrice().Amount(),
				Currency: item.UnitPrice().Currency(),
			},
			Subtotal: MoneyResponse{
				Amount:   item.Subtotal().Amount(),
				Currency: item.Subtotal().Currency(),
			},
		}
	}

	return &OrderResponse{
		ID:     order.ID(),
		UserID: order.UserID(),
		Items:  items,
		TotalAmount: MoneyResponse{
			Amount:   order.TotalAmount().Amount(),
			Currency: order.TotalAmount().Currency(),
		},
		Status:    string(order.Status()),
		CreatedAt: order.CreatedAt(),
		UpdatedAt: order.UpdatedAt(),
	}
}
