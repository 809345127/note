package service

import (
	"ddd-example/domain"
	"errors"
	"fmt"
	"time"
)

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

// CreateOrderRequest 创建订单请求DTO
type CreateOrderRequest struct {
	UserID string      `json:"user_id" binding:"required"`
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
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	Items       []OrderItemResponse `json:"items"`
	TotalAmount MoneyResponse    `json:"total_amount"`
	Status      string           `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
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

// CreateOrder 创建订单
func (s *OrderApplicationService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
	// 检查用户是否可以下单
	canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(req.UserID)
	if err != nil {
		return nil, err
	}
	if !canPlaceOrder {
		return nil, errors.New("user cannot place order")
	}
	
	// 转换订单项
	orderItems := make([]domain.OrderItem, len(req.Items))
	for i, item := range req.Items {
		unitPrice := domain.NewMoney(item.UnitPrice, item.Currency)
		orderItems[i] = domain.NewOrderItem(item.ProductID, item.ProductName, item.Quantity, *unitPrice)
	}
	
	// 创建订单实体
	order, err := domain.NewOrder(req.UserID, orderItems)
	if err != nil {
		return nil, err
	}
	
	// 保存订单
	if err := s.orderRepo.Save(order); err != nil {
		return nil, err
	}
	
	// 发布订单创建事件
	event := domain.NewOrderPlacedEvent(order.ID(), order.UserID(), order.TotalAmount())
	if err := s.eventPublisher.Publish(event); err != nil {
		fmt.Printf("Failed to publish order placed event: %v\n", err)
	}
	
	return s.convertToResponse(order), nil
}

// GetOrder 获取订单信息
func (s *OrderApplicationService) GetOrder(orderID string) (*OrderResponse, error) {
	order, err := s.orderRepo.FindByID(orderID)
	if err != nil {
		return nil, err
	}
	
	return s.convertToResponse(order), nil
}

// GetUserOrders 获取用户的所有订单
func (s *OrderApplicationService) GetUserOrders(userID string) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.FindByUserID(userID)
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
func (s *OrderApplicationService) UpdateOrderStatus(req UpdateOrderStatusRequest) error {
	order, err := s.orderRepo.FindByID(req.OrderID)
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
	
	return s.orderRepo.Save(order)
}

// ProcessOrder 处理订单
func (s *OrderApplicationService) ProcessOrder(orderID string) error {
	return s.orderDomainService.ProcessOrder(orderID)
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