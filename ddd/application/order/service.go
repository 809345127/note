/*
Package order Application Layer - Order Business Process Orchestration

Responsibilities of Application Layer:
1. Receive external requests (usually from Controller)
2. Call domain services for business rule validation
3. Call aggregate root methods to execute business operations
4. Use UoW to manage transactions and event collection (Outbox pattern)
5. Return results to caller

Important: Application services do not directly publish events!
- UoW collects events from aggregates and saves to outbox table before commit
- OutboxProcessor reads outbox table asynchronously and publishes to message queue
- This ensures atomicity of events and business data
*/
package order

import (
	"context"
	"errors"
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

// CreateOrder Create order
// Uses UoW to manage transaction and collect events from aggregate
func (s *ApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	var o *order.Order

	err := s.uow.Execute(ctx, func(ctx context.Context) error {
		// Check if user can place order
		canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(ctx, req.UserID)
		if err != nil {
			return err
		}
		if !canPlaceOrder {
			return errors.New("user cannot place order")
		}

		// Convert order item requests to domain model
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

		// Create order entity (aggregate root records events internally)
		o, err = order.NewOrder(req.UserID, requests)
		if err != nil {
			return err
		}

		// Save order (uses transaction from context)
		if err := s.orderRepo.Save(ctx, o); err != nil {
			return err
		}

		// Register aggregate with UoW for event collection
		s.uow.RegisterNew(o)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.convertToResponse(o), nil
}

// GetOrder Get order information
func (s *ApplicationService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	o, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
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
