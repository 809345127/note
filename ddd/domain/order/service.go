package order

import (
	"context"
	"errors"
)

// UserChecker 用户状态检查接口
// 用于打破 order 和 user 包之间的循环依赖
type UserChecker interface {
	IsUserActive(ctx context.Context, userID string) (bool, error)
}

// DomainService 订单领域服务
// DDD原则：领域服务可依赖 Repository 接口查询数据，但不调用 Save 持久化
type DomainService struct {
	userChecker     UserChecker
	orderRepository Repository
}

// NewDomainService 创建订单领域服务
func NewDomainService(userChecker UserChecker, orderRepo Repository) *DomainService {
	return &DomainService{
		userChecker:     userChecker,
		orderRepository: orderRepo,
	}
}

// CanProcessOrder 检查订单是否可以处理
// DDD原则：领域服务只做业务规则验证，返回验证结果
// 实际的状态变更和持久化由应用服务负责
func (s *DomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 验证用户状态
	isActive, err := s.userChecker.IsUserActive(ctx, order.UserID())
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, errors.New("user is not active")
	}

	// 验证订单状态
	if order.Status() != StatusPending {
		return nil, errors.New("only pending orders can be processed")
	}

	return order, nil
}
