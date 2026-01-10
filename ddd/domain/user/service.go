/*
Domain Service

Domain services handle business logic that doesn't fit well in single entities, typically:
1. Business rule validation across multiple aggregate roots
2. Complex business calculations requiring access to multiple repositories
3. Stateless business rules

Core principle: Domain service only reads, does not write
*/
package user

import (
	"context"
)

// DomainService User domain service - handles user-related business logic
type DomainService struct {
	userRepository Repository
}

// NewDomainService Create user domain service
func NewDomainService(userRepo Repository) *DomainService {
	return &DomainService{
		userRepository: userRepo,
	}
}

// CanUserPlaceOrder Check if user can place order
func (s *DomainService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive() {
		return false, ErrUserNotActive
	}

	// Directly check age since CanMakePurchase already checks isActive
	if user.Age() < 18 {
		return false, ErrUserTooYoung
	}

	return true, nil
}
