package order

import (
	"context"
	"fmt"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
)

// ApplicationService 编排订单相关用例。
type ApplicationService struct {
	orderRepo          order.Repository
	orderDomainService *order.DomainService
	userDomainService  *user.DomainService
	uowFactory         shared.UnitOfWorkFactory
}

func NewApplicationService(
	orderRepo order.Repository,
	userRepo user.Repository,
	uowFactory shared.UnitOfWorkFactory,
) *ApplicationService {
	userChecker := &userCheckerAdapter{userRepo: userRepo}
	return &ApplicationService{
		orderRepo:          orderRepo,
		orderDomainService: order.NewDomainService(userChecker, orderRepo),
		userDomainService:  user.NewDomainService(userRepo),
		uowFactory:         uowFactory,
	}
}

func (s *ApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	var o *order.Order
	uow := s.uowFactory.New()

	err := uow.Execute(ctx, func(ctx context.Context) error {
		canPlaceOrder, err := s.userDomainService.CanUserPlaceOrder(ctx, req.UserID)
		if err != nil {
			return fmt.Errorf("check user can place order: %w", err)
		}
		if !canPlaceOrder {
			return order.NewUserCannotPlaceOrderError(req.UserID, "user is not active")
		}

		o, err = order.NewOrder(req.UserID, toItemRequests(req.Items))
		if err != nil {
			return err
		}

		if err := s.orderRepo.Save(ctx, o); err != nil {
			return fmt.Errorf("save order: %w", err)
		}

		uow.RegisterNew(o)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return toOrderResponse(o), nil
}

func (s *ApplicationService) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	o, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return toOrderResponse(o), nil
}

func (s *ApplicationService) GetUserOrders(ctx context.Context, userID string) ([]*OrderResponse, error) {
	orders, err := s.orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]*OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = toOrderResponse(o)
	}
	return responses, nil
}

func (s *ApplicationService) UpdateOrderStatus(ctx context.Context, req UpdateOrderStatusRequest) error {
	uow := s.uowFactory.New()
	return uow.Execute(ctx, func(ctx context.Context) error {
		o, err := s.orderRepo.FindByID(ctx, req.OrderID)
		if err != nil {
			return err
		}

		if err := applyStatusTransition(o, req.Status, req.Reason); err != nil {
			return err
		}

		if err := s.orderRepo.Save(ctx, o); err != nil {
			return err
		}

		uow.RegisterDirty(o)
		return nil
	})
}

func (s *ApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
	uow := s.uowFactory.New()
	return uow.Execute(ctx, func(ctx context.Context) error {
		if err := s.orderDomainService.ValidateProcessOrder(ctx, orderID); err != nil {
			return err
		}

		o, err := s.orderRepo.FindByID(ctx, orderID)
		if err != nil {
			return err
		}

		if err := o.Confirm(); err != nil {
			return err
		}

		if err := s.orderRepo.Save(ctx, o); err != nil {
			return err
		}

		uow.RegisterDirty(o)
		return nil
	})
}

func applyStatusTransition(o *order.Order, status, reason string) error {
	switch order.Status(status) {
	case order.StatusPending:
		return nil
	case order.StatusConfirmed:
		return o.Confirm()
	case order.StatusShipped:
		return o.Ship()
	case order.StatusDelivered:
		return o.Deliver()
	case order.StatusCancelled:
		return o.Cancel(reason)
	default:
		return shared.NewValidationError("order", "status", "invalid order status")
	}
}
