package order

import (
	"context"
)

type UserChecker interface {
	IsUserActive(ctx context.Context, userID string) (bool, error)
}
type DomainService struct {
	userChecker     UserChecker
	orderRepository Repository
}

func NewDomainService(userChecker UserChecker, orderRepo Repository) *DomainService {
	return &DomainService{
		userChecker:     userChecker,
		orderRepository: orderRepo,
	}
}
func (s *DomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	isActive, err := s.userChecker.IsUserActive(ctx, order.UserID())
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, ErrUserNotActiveForOrder
	}
	if order.Status() != StatusPending {
		return nil, ErrInvalidOrderStateTransition
	}

	return order, nil
}
func (s *DomainService) ValidateProcessOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	isActive, err := s.userChecker.IsUserActive(ctx, order.UserID())
	if err != nil {
		return err
	}
	if !isActive {
		return ErrUserNotActiveForOrder
	}
	if order.Status() != StatusPending {
		return ErrInvalidOrderStateTransition
	}

	return nil
}
