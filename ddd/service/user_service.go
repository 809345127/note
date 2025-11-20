package service

import (
	"errors"
	"fmt"
	"time"

	"ddd-example/domain"
)

// UserApplicationService 用户应用服务 - 协调用户相关的业务流程
type UserApplicationService struct {
	userRepo          domain.UserRepository
	userDomainService *domain.UserDomainService
	eventPublisher    domain.DomainEventPublisher
}

// NewUserApplicationService 创建用户应用服务
func NewUserApplicationService(
	userRepo domain.UserRepository,
	orderRepo domain.OrderRepository,
	eventPublisher domain.DomainEventPublisher,
) *UserApplicationService {
	return &UserApplicationService{
		userRepo:          userRepo,
		userDomainService: domain.NewUserDomainService(userRepo, orderRepo),
		eventPublisher:    eventPublisher,
	}
}

// CreateUserRequest 创建用户请求DTO
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age" binding:"required,min=0,max=150"`
}

// UserResponse 用户响应DTO
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUser 创建用户
func (s *UserApplicationService) CreateUser(req CreateUserRequest) (*UserResponse, error) {
	// 检查邮箱是否已存在
	existingUser, _ := s.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}
	
	// 创建用户实体
	user, err := domain.NewUser(req.Name, req.Email, req.Age)
	if err != nil {
		return nil, err
	}
	
	// 保存用户
	if err := s.userRepo.Save(user); err != nil {
		return nil, err
	}
	
	// 发布用户创建事件
	event := domain.NewUserCreatedEvent(user.ID(), user.Name(), user.Email().Value())
	if err := s.eventPublisher.Publish(event); err != nil {
		// 记录事件发布失败，但不影响用户创建的主流程
		fmt.Printf("Failed to publish user created event: %v\n", err)
	}
	
	return s.convertToResponse(user), nil
}

// GetUser 获取用户信息
func (s *UserApplicationService) GetUser(userID string) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	
	return s.convertToResponse(user), nil
}

// GetAllUsers 获取所有用户
func (s *UserApplicationService) GetAllUsers() ([]*UserResponse, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, err
	}
	
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = s.convertToResponse(user)
	}
	
	return responses, nil
}

// UpdateUserStatusRequest 更新用户状态请求DTO
type UpdateUserStatusRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Active bool   `json:"active"`
}

// UpdateUserStatus 更新用户状态
func (s *UserApplicationService) UpdateUserStatus(req UpdateUserStatusRequest) error {
	user, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		return err
	}
	
	if req.Active {
		user.Activate()
	} else {
		user.Deactivate()
	}
	
	return s.userRepo.Save(user)
}

// GetUserTotalSpentRequest 获取用户总消费请求DTO
type GetUserTotalSpentRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// GetUserTotalSpentResponse 获取用户总消费响应DTO
type GetUserTotalSpentResponse struct {
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

// GetUserTotalSpent 获取用户总消费金额
func (s *UserApplicationService) GetUserTotalSpent(req GetUserTotalSpentRequest) (*GetUserTotalSpentResponse, error) {
	totalAmount, err := s.userDomainService.CalculateUserTotalSpent(req.UserID)
	if err != nil {
		return nil, err
	}
	
	return &GetUserTotalSpentResponse{
		UserID:      req.UserID,
		TotalAmount: totalAmount.Amount(),
		Currency:    totalAmount.Currency(),
	}, nil
}

// convertToResponse 将用户实体转换为响应DTO
func (s *UserApplicationService) convertToResponse(user *domain.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID(),
		Name:      user.Name(),
		Email:     user.Email().Value(),
		Age:       user.Age(),
		IsActive:  user.IsActive(),
		CreatedAt: user.CreatedAt(),
		UpdatedAt: user.UpdatedAt(),
	}
}