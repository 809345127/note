package user

import (
	"context"
	"fmt"
	"time"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
)

type ApplicationService struct {
	userRepo          user.Repository
	orderRepo         order.Repository
	userDomainService *user.DomainService
	uowFactory        shared.UnitOfWorkFactory
}

func NewApplicationService(
	userRepo user.Repository,
	orderRepo order.Repository,
	uowFactory shared.UnitOfWorkFactory,
) *ApplicationService {
	return &ApplicationService{
		userRepo:          userRepo,
		orderRepo:         orderRepo,
		userDomainService: user.NewDomainService(userRepo),
		uowFactory:        uowFactory,
	}
}

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"required,min=0,max=150"`
}
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *ApplicationService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	var u *user.User
	uow := s.uowFactory.New()

	err := uow.Execute(ctx, func(ctx context.Context) error {
		var err error
		u, err = user.NewUser(req.Name, req.Email, req.Age)
		if err != nil {
			return err
		}
		if err := s.userRepo.Save(ctx, u); err != nil {
			return err
		}
		uow.RegisterNew(u)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}
func (s *ApplicationService) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}

type UpdateUserStatusRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Active bool   `json:"active"`
}

func (s *ApplicationService) UpdateUserStatus(ctx context.Context, req UpdateUserStatusRequest) error {
	uow := s.uowFactory.New()
	return uow.Execute(ctx, func(ctx context.Context) error {
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

		uow.RegisterDirty(u)
		return nil
	})
}

type GetUserTotalSpentRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
type GetUserTotalSpentResponse struct {
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

func (s *ApplicationService) GetUserTotalSpent(ctx context.Context, req GetUserTotalSpentRequest) (*GetUserTotalSpentResponse, error) {
	orders, err := s.orderRepo.FindDeliveredOrdersByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	total := shared.NewMoney(0, "CNY")
	for _, o := range orders {
		if o.TotalAmount().Currency() != total.Currency() {
			return nil, fmt.Errorf("mixed currencies not supported: %s vs %s", o.TotalAmount().Currency(), total.Currency())
		}
		var addErr error
		total, addErr = total.Add(o.TotalAmount())
		if addErr != nil {
			return nil, addErr
		}
	}

	return &GetUserTotalSpentResponse{
		UserID:      req.UserID,
		TotalAmount: total.Amount(),
		Currency:    total.Currency(),
	}, nil
}
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
func (s *ApplicationService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	return s.userDomainService.CanUserPlaceOrder(ctx, userID)
}
