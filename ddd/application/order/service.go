/*
Package order 应用层 - 订单业务流程编排

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
*/
package order

import (
	"context"
	"errors"
	"time"

	"ddd-example/domain/order"
	"ddd-example/domain/shared"
	"ddd-example/domain/user"
)

// userCheckerAdapter 适配器：将 user.Repository 适配为 order.UserChecker
type userCheckerAdapter struct {
	userRepo user.Repository
}

func (a *userCheckerAdapter) IsUserActive(ctx context.Context, userID string) (bool, error) {
	u, err := a.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.IsActive(), nil
}

// ApplicationService 订单应用服务 - 协调订单相关的业务流程
type ApplicationService struct {
	orderRepo          order.Repository
	userRepo           user.Repository
	orderDomainService *order.DomainService
	userDomainService  *user.DomainService
	eventPublisher     shared.DomainEventPublisher
}

// NewApplicationService 创建订单应用服务
func NewApplicationService(
	orderRepo order.Repository,
	userRepo user.Repository,
	eventPublisher shared.DomainEventPublisher,
) *ApplicationService {
	userChecker := &userCheckerAdapter{userRepo: userRepo}
	return &ApplicationService{
		orderRepo:          orderRepo,
		userRepo:           userRepo,
		orderDomainService: order.NewDomainService(userChecker, orderRepo),
		userDomainService:  user.NewDomainService(userRepo),
		eventPublisher:     eventPublisher,
	}
}

// ============================================================================
// DTO定义 - 数据传输对象
// ============================================================================

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
func (s *ApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	// 检查用户是否可以下单
	canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !canPlaceOrder {
		return nil, errors.New("user cannot place order")
	}

	// 转换订单项请求为领域模型
	requests := make([]order.ItemRequest, len(req.Items))
	for i, item := range req.Items {
		unitPrice := shared.NewMoney(item.UnitPrice, item.Currency)
		requests[i] = order.ItemRequest{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   *unitPrice,
		}
	}

	// 创建订单实体
	o, err := order.NewOrder(req.UserID, requests)
	if err != nil {
		return nil, err
	}

	// 保存订单
	if err := s.orderRepo.Save(ctx, o); err != nil {
		return nil, err
	}

	return s.convertToResponse(o), nil
}

// GetOrder 获取订单信息
func (s *ApplicationService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	o, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(o), nil
}

// GetUserOrders 获取用户的所有订单
func (s *ApplicationService) GetUserOrders(ctx context.Context, userID string) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = s.convertToResponse(o)
	}

	return responses, nil
}

// UpdateOrderStatusRequest 更新订单状态请求DTO
type UpdateOrderStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=PENDING CONFIRMED SHIPPED DELIVERED CANCELLED"`
}

// UpdateOrderStatus 更新订单状态
func (s *ApplicationService) UpdateOrderStatus(ctx context.Context, req UpdateOrderStatusRequest) error {
	o, err := s.orderRepo.FindByID(ctx, req.OrderID)
	if err != nil {
		return err
	}

	// 根据请求的状态更新订单
	switch order.Status(req.Status) {
	case order.StatusPending:
		// 不需要任何操作
	case order.StatusConfirmed:
		if err := o.Confirm(); err != nil {
			return err
		}
	case order.StatusShipped:
		if err := o.Ship(); err != nil {
			return err
		}
	case order.StatusDelivered:
		if err := o.Deliver(); err != nil {
			return err
		}
	case order.StatusCancelled:
		if err := o.Cancel(); err != nil {
			return err
		}
	default:
		return errors.New("invalid order status")
	}

	return s.orderRepo.Save(ctx, o)
}

// ProcessOrder 处理订单
func (s *ApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
	// 1. 通过领域服务验证订单是否可以处理
	o, err := s.orderDomainService.CanProcessOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. 执行状态变更（聚合根方法，内部会记录事件）
	if err := o.Confirm(); err != nil {
		return err
	}

	// 3. 保存（UoW 会将事件保存到 outbox 表，后台异步发布）
	return s.orderRepo.Save(ctx, o)
}

// convertToResponse 将订单实体转换为响应DTO
func (s *ApplicationService) convertToResponse(o *order.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(o.Items()))
	for i, item := range o.Items() {
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
		ID:     o.ID(),
		UserID: o.UserID(),
		Items:  items,
		TotalAmount: MoneyResponse{
			Amount:   o.TotalAmount().Amount(),
			Currency: o.TotalAmount().Currency(),
		},
		Status:    string(o.Status()),
		CreatedAt: o.CreatedAt(),
		UpdatedAt: o.UpdatedAt(),
	}
}
