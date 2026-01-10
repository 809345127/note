package order

import (
	"context"
)

// UserChecker User status checker interface
// Used to break circular dependency between order and user packages
type UserChecker interface {
	IsUserActive(ctx context.Context, userID string) (bool, error)
}

// DomainService Order domain service
// DDD principle: Domain service can use Repository interfaces to query data but does not call Save for persistence
type DomainService struct {
	userChecker     UserChecker
	orderRepository Repository
}

// NewDomainService Create order domain service
func NewDomainService(userChecker UserChecker, orderRepo Repository) *DomainService {
	return &DomainService{
		userChecker:     userChecker,
		orderRepository: orderRepo,
	}
}

// CanProcessOrder Check if order can be processed
// DDD principle: Domain service only validates business rules, returns validation result
// Actual state changes and persistence handled by application service
func (s *DomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Validate user status
	isActive, err := s.userChecker.IsUserActive(ctx, order.UserID())
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, ErrUserNotActiveForOrder
	}

	// Validate order status
	if order.Status() != StatusPending {
		return nil, ErrInvalidOrderStateTransition
	}

	return order, nil
}

// ValidateProcessOrder Validate if order can be processed (without returning the order)
// This is useful when the caller needs to re-fetch the order for modification
func (s *DomainService) ValidateProcessOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Validate user status
	isActive, err := s.userChecker.IsUserActive(ctx, order.UserID())
	if err != nil {
		return err
	}
	if !isActive {
		return ErrUserNotActiveForOrder
	}

	// Validate order status
	if order.Status() != StatusPending {
		return ErrInvalidOrderStateTransition
	}

	return nil
}
