/*
领域服务（Domain Service）

领域服务用于承载不适合放入单一实体的方法，典型场景：
1. 跨多个聚合根的业务规则校验
2. 依赖多个仓储的复杂业务计算
3. 无状态的领域规则逻辑

核心原则：领域服务只读，不直接写入持久化
*/
package user

import (
	"context"
)

type DomainService struct {
	userRepository Repository
}

func NewDomainService(userRepo Repository) *DomainService {
	return &DomainService{
		userRepository: userRepo,
	}
}
func (s *DomainService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive() {
		return false, ErrUserNotActive
	}
	if user.Age() < 18 {
		return false, ErrUserTooYoung
	}

	return true, nil
}
