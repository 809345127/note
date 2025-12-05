package user

import (
	"context"
	"errors"
	"time"

	"ddd-example/domain/order"
	"ddd-example/domain/shared"
	"ddd-example/domain/user"
)

// ApplicationService 用户应用服务 - 协调用户相关的业务流程
type ApplicationService struct {
	userRepo          user.Repository
	orderRepo         order.Repository
	userDomainService *user.DomainService
	eventPublisher    shared.DomainEventPublisher
}

// NewApplicationService 创建用户应用服务
func NewApplicationService(
	userRepo user.Repository,
	orderRepo order.Repository,
	eventPublisher shared.DomainEventPublisher,
) *ApplicationService {
	return &ApplicationService{
		userRepo:          userRepo,
		orderRepo:         orderRepo,
		userDomainService: user.NewDomainService(userRepo),
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
func (s *ApplicationService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	// 检查邮箱是否已存在
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// 创建用户实体（聚合根在创建时记录领域事件）
	u, err := user.NewUser(req.Name, req.Email, req.Age)
	if err != nil {
		return nil, err
	}

	// 保存用户（仓储只负责持久化，事件由 UoW 保存到 outbox 表）
	if err := s.userRepo.Save(ctx, u); err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}

// GetUser 获取用户信息
func (s *ApplicationService) GetUser(ctx context.Context, userID string) (*UserResponse, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.convertToResponse(u), nil
}

// GetAllUsers 获取所有用户
// 注意：该方法暴露了FindAll功能，违反了DDD聚合原则
// 在真实项目中，应该使用查询服务（Query Service）替代
// 或添加分页、过滤等限制，避免加载所有聚合根
func (s *ApplicationService) GetAllUsers() ([]*UserResponse, error) {
	// 在DDD中，仓储不应该提供FindAll方法
	// 这里为了在接口层演示，我们使用模拟数据
	// 实际应该通过UserQueryService.SearchUsers实现

	// 创建Mock数据用于测试（仅用于演示，生产环境应该移除）
	users := make([]*user.User, 0)

	// 这里假设有用户数据，实际应该查询数据库
	// 由于仓储接口已移除FindAll，此方法已不符合DDD原则
	// 建议在下一个迭代中重构为使用查询服务

	return s.convertUsersToResponses(users), nil
}

// convertUsersToResponses 转换用户列表为响应列表
func (s *ApplicationService) convertUsersToResponses(users []*user.User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, u := range users {
		responses[i] = s.convertToResponse(u)
	}
	return responses
}

// UpdateUserStatusRequest 更新用户状态请求DTO
type UpdateUserStatusRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Active bool   `json:"active"`
}

// UpdateUserStatus 更新用户状态
func (s *ApplicationService) UpdateUserStatus(ctx context.Context, req UpdateUserStatusRequest) error {
	u, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if req.Active {
		u.Activate()
	} else {
		u.Deactivate()
	}

	return s.userRepo.Save(ctx, u)
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
// 注意：这是一个跨子域的查询，放在应用层处理
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

// convertToResponse 将用户实体转换为响应DTO
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

// CanUserPlaceOrder 检查用户是否可以下单（委托给领域服务）
func (s *ApplicationService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	return s.userDomainService.CanUserPlaceOrder(ctx, userID)
}
