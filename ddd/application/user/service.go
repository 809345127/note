package user

import (
	"context"
	"errors"
	"time"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
)

// ApplicationService User application service - coordinates user-related business processes
type ApplicationService struct {
	userRepo          user.Repository
	orderRepo         order.Repository
	userDomainService *user.DomainService
	uow               shared.UnitOfWork
}

// NewApplicationService Create user application service
func NewApplicationService(
	userRepo user.Repository,
	orderRepo order.Repository,
	uow shared.UnitOfWork,
) *ApplicationService {
	return &ApplicationService{
		userRepo:          userRepo,
		orderRepo:         orderRepo,
		userDomainService: user.NewDomainService(userRepo),
		uow:               uow,
	}
}

// CreateUserRequest Create user request DTO
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"required,min=0,max=150"`
}

// UserResponse User response DTO
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUser Create user
// DDD principle: Application service orchestrates business processes
// UoW manages transaction and collects events from aggregates (Outbox pattern)
func (s *ApplicationService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	var u *user.User

	err := s.uow.Execute(ctx, func(ctx context.Context) error {
		// Check if email already exists
		existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
		if existingUser != nil {
			return errors.New("email already exists")
		}

		// Create user entity (aggregate root records domain events upon creation)
		var err error
		u, err = user.NewUser(req.Name, req.Email, req.Age)
		if err != nil {
			return err
		}

		// Save user (uses transaction from context)
		if err := s.userRepo.Save(ctx, u); err != nil {
			return err
		}

		// Register aggregate with UoW for event collection
		s.uow.RegisterNew(u)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}

// GetUser Get user information
func (s *ApplicationService) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}

// UpdateUserStatusRequest Update user status request DTO
type UpdateUserStatusRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Active bool   `json:"active"`
}

// UpdateUserStatus Update user status
func (s *ApplicationService) UpdateUserStatus(ctx context.Context, req UpdateUserStatusRequest) error {
	return s.uow.Execute(ctx, func(ctx context.Context) error {
		u, err := s.userRepo.FindByID(ctx, req.UserID)
		if err != nil {
			return err
		}

		if req.Active {
			u.Activate()
		} else {
			u.Deactivate()
		}

		if err := s.userRepo.Save(ctx, u); err != nil {
			return err
		}

		s.uow.RegisterDirty(u)
		return nil
	})
}

// GetUserTotalSpentRequest Get user total spent request DTO
type GetUserTotalSpentRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// GetUserTotalSpentResponse Get user total spent response DTO
type GetUserTotalSpentResponse struct {
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

// GetUserTotalSpent Get user total spent amount
// Note: This is a cross-subdomain query, handled at application layer
func (s *ApplicationService) GetUserTotalSpent(ctx context.Context, req GetUserTotalSpentRequest) (*GetUserTotalSpentResponse, error) {
	orders, err := s.orderRepo.FindDeliveredOrdersByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	total := shared.NewMoney(0, "CNY")
	for _, o := range orders {
		total, _ = total.Add(o.TotalAmount())
	}

	return &GetUserTotalSpentResponse{
		UserID:      req.UserID,
		TotalAmount: total.Amount(),
		Currency:    total.Currency(),
	}, nil
}

// convertToResponse Convert user entity to response DTO
func (s *ApplicationService) convertToResponse(u *user.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID(),
		Name:      u.Name(),
		Email:     u.Email().Value(),
		Age:       u.Age(),
		IsActive:  u.IsActive(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

// CanUserPlaceOrder Check if user can place order (delegated to domain service)
func (s *ApplicationService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	return s.userDomainService.CanUserPlaceOrder(ctx, userID)
}
