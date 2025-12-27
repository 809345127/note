/*
Package order - 订单应用服务层

职责:
1. 接收外部请求（来自 Controller）
2. 调用领域服务进行业务规则校验
3. 调用聚合根方法执行业务操作
4. 使用 UoW 管理事务和事件收集（Outbox 模式）
5. 返回结果给调用方

错误处理原则:
1. 领域错误直接向上传递，不在应用层转换
2. 基础设施错误（如数据库错误）使用 fmt.Errorf 包装添加上下文
3. API 层统一使用 errors.FromDomainError() 转换为应用错误
4. 这样保持了错误链完整，便于调试和日志追踪

事件处理:
- UoW 收集聚合产生的事件，在提交前保存到 outbox 表
- OutboxProcessor 异步读取 outbox 表并发布到消息队列
- 这确保了事件和业务数据的原子性
*/
package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
)

// userCheckerAdapter Adapter: adapts user.Repository to order.UserChecker
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

// ApplicationService Order application service - coordinates order-related business processes
type ApplicationService struct {
	orderRepo          order.Repository
	userRepo           user.Repository
	orderDomainService *order.DomainService
	userDomainService  *user.DomainService
	uow                shared.UnitOfWork
}

// NewApplicationService Create order application service
func NewApplicationService(
	orderRepo order.Repository,
	userRepo user.Repository,
	uow shared.UnitOfWork,
) *ApplicationService {
	userChecker := &userCheckerAdapter{userRepo: userRepo}
	return &ApplicationService{
		orderRepo:          orderRepo,
		userRepo:           userRepo,
		orderDomainService: order.NewDomainService(userChecker, orderRepo),
		userDomainService:  user.NewDomainService(userRepo),
		uow:                uow,
	}
}

// ============================================================================
// DTO Definitions - Data Transfer Objects
// ============================================================================

// CreateOrderRequest Create order request DTO
type CreateOrderRequest struct {
	UserID string             `json:"user_id" binding:"required"`
	Items  []OrderItemRequest `json:"items" binding:"required,min=1"`
}

// OrderItemRequest Order item request DTO
type OrderItemRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	ProductName string `json:"product_name" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	UnitPrice   int64  `json:"unit_price" binding:"required,min=0"`
	Currency    string `json:"currency" binding:"required"`
}

// OrderResponse Order response DTO
type OrderResponse struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Items       []OrderItemResponse `json:"items"`
	TotalAmount MoneyResponse       `json:"total_amount"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// OrderItemResponse Order item response DTO
type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	ProductName string        `json:"product_name"`
	Quantity    int           `json:"quantity"`
	UnitPrice   MoneyResponse `json:"unit_price"`
	Subtotal    MoneyResponse `json:"subtotal"`
}

// MoneyResponse Money response DTO
type MoneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// ============================================================================
// Application Service Methods - Business Process Orchestration
// ============================================================================

// CreateOrder 创建订单
// 使用 UoW 管理事务和收集聚合产生的领域事件
// 错误处理: 领域错误直接传递，基础设施错误添加上下文后传递
func (s *ApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	var o *order.Order

	err := s.uow.Execute(ctx, func(ctx context.Context) error {
		// 1. 检查用户是否可以下单（领域服务校验）
		canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(ctx, req.UserID)
		if err != nil {
			// 基础设施错误: 添加上下文后传递
			return fmt.Errorf("check user can place order: %w", err)
		}
		if !canPlaceOrder {
			// 领域错误: 使用领域错误构造函数
			return order.NewUserCannotPlaceOrderError(req.UserID, "user is not active")
		}

		// 2. 转换订单项请求为领域模型
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

		// 3. 创建订单聚合（聚合根内部记录领域事件）
		o, err = order.NewOrder(req.UserID, requests)
		if err != nil {
			// 领域错误: 直接传递（如订单项为空等）
			return err
		}

		// 4. 持久化订单（使用上下文中的事务）
		if err := s.orderRepo.Save(ctx, o); err != nil {
			// 基础设施错误: 添加上下文
			return fmt.Errorf("save order: %w", err)
		}

		// 5. 注册聚合到 UoW 用于事件收集
		s.uow.RegisterNew(o)
		return nil
	})

	if err != nil {
		// 直接返回错误，API 层统一处理
		return nil, err
	}

	return s.convertToResponse(o), nil
}

// GetOrder 获取订单信息
// 错误处理: 领域错误直接传递，API 层统一转换
func (s *ApplicationService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	// 从仓储获取订单
	// 仓储层会返回领域错误 (如 order.ErrOrderNotFound)
	o, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		// 最佳实践: 领域错误直接传递，不在应用层转换
		// API 层会使用 errors.FromDomainError() 统一处理
		// 这样保持了完整的错误链，便于调试
		return nil, err
	}

	return s.convertToResponse(o), nil
}

// GetUserOrders Get all orders for user
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

// UpdateOrderStatusRequest Update order status request DTO
type UpdateOrderStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=PENDING CONFIRMED SHIPPED DELIVERED CANCELLED"`
}

// UpdateOrderStatus Update order status
func (s *ApplicationService) UpdateOrderStatus(ctx context.Context, req UpdateOrderStatusRequest) error {
	o, err := s.orderRepo.FindByID(ctx, req.OrderID)
	if err != nil {
		return err
	}

	// Update order based on requested status
	switch order.Status(req.Status) {
	case order.StatusPending:
		// No action needed
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

// ProcessOrder Process order
// Uses UoW to manage transaction and collect events
func (s *ApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
	return s.uow.Execute(ctx, func(ctx context.Context) error {
		// 1. Verify if order can be processed through domain service
		o, err := s.orderDomainService.CanProcessOrder(ctx, orderID)
		if err != nil {
			return err
		}

		// 2. Execute status change (aggregate root method)
		if err := o.Confirm(); err != nil {
			return err
		}

		// 3. Save (uses transaction from context)
		if err := s.orderRepo.Save(ctx, o); err != nil {
			return err
		}

		// 4. Register aggregate for event collection
		s.uow.RegisterDirty(o)
		return nil
	})
}

// convertToResponse Convert order entity to response DTO
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
