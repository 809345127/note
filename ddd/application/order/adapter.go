package order

import (
	"context"

	"ddd/domain/user"
)

// userCheckerAdapter 将 user.Repository 适配为订单领域服务需要的用户状态查询接口。
type userCheckerAdapter struct {
	userRepo user.Repository
}

func (a *userCheckerAdapter) IsUserActive(ctx context.Context, userID string) (bool, error) {
	u, err := a.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.IsActive(), nil
}
