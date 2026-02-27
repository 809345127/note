package user

import (
	"context"

	"ddd/domain/shared"
)

// Repository 定义用户聚合的持久化接口。
type Repository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindBySpecification(ctx context.Context, spec shared.Specification[*User]) ([]*User, error)
	Remove(ctx context.Context, id string) error
}
