package service

import (
	"context"
	"errors"
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
// DDD原则：应用服务负责编排业务流程，事件由 UoW 保存到 outbox 表，后台 Message Relay 异步发布
func (s *UserApplicationService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	// 检查邮箱是否已存在
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// 创建用户实体（聚合根在创建时记录领域事件）
	user, err := domain.NewUser(req.Name, req.Email, req.Age)
	if err != nil {
		return nil, err
	}

	// 保存用户（仓储只负责持久化，事件由 UoW 保存到 outbox 表）
	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return s.convertToResponse(user), nil
}

// GetUser 获取用户信息
func (s *UserApplicationService) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(user), nil
}

// GetAllUsers 获取所有用户
// 注意：该方法暴露了FindAll功能，违反了DDD聚合原则
// 在真实项目中，应该使用查询服务（Query Service）替代
// 或添加分页、过滤等限制，避免加载所有聚合根
func (s *UserApplicationService) GetAllUsers() ([]*UserResponse, error) {
	// 在DDD中，仓储不应该提供FindAll方法
	// 这里为了在接口层演示，我们使用模拟数据
	// 实际应该通过UserQueryService.SearchUsers实现

	// 创建Mock数据用于测试（仅用于演示，生产环境应该移除）
	users := make([]*domain.User, 0)

	// 这里假设有用户数据，实际应该查询数据库
	// 由于仓储接口已移除FindAll，此方法已不符合DDD原则
	// 建议在下一个迭代中重构为使用查询服务

	return s.convertUsersToResponses(users), nil
}

// convertUsersToResponses 转换用户列表为响应列表
func (s *UserApplicationService) convertUsersToResponses(users []*domain.User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = s.convertToResponse(user)
	}
	return responses
}

// UpdateUserStatusRequest 更新用户状态请求DTO
type UpdateUserStatusRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Active bool   `json:"active"`
}

// UpdateUserStatus 更新用户状态
func (s *UserApplicationService) UpdateUserStatus(ctx context.Context, req UpdateUserStatusRequest) error {
	user, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if req.Active {
		user.Activate()
	} else {
		user.Deactivate()
	}

	return s.userRepo.Save(ctx, user)
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
func (s *UserApplicationService) GetUserTotalSpent(ctx context.Context, req GetUserTotalSpentRequest) (*GetUserTotalSpentResponse, error) {
	totalAmount, err := s.userDomainService.CalculateUserTotalSpent(ctx, req.UserID)
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
