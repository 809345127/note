/*
领域服务（Domain Service）

领域服务用于处理不适合放在单个实体中的业务逻辑，通常是：
1. 跨多个聚合根的业务规则验证
2. 需要访问多个仓储的复杂业务计算
3. 无状态的业务规则

核心原则：领域服务只读不写
*/
package user

import (
	"context"
	"errors"
)

// DomainService 用户领域服务 - 处理用户相关的业务逻辑
type DomainService struct {
	userRepository Repository
}

// NewDomainService 创建用户领域服务
func NewDomainService(userRepo Repository) *DomainService {
	return &DomainService{
		userRepository: userRepo,
	}
}

// CanUserPlaceOrder 检查用户是否可以下单
func (s *DomainService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive() {
		return false, ErrUserNotActive
	}

	if !user.CanMakePurchase() {
		return false, errors.New("user cannot make purchases")
	}

	return true, nil
}
