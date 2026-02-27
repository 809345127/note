package order

import (
	"context"

	"ddd/domain/shared"
)

type Repository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
	FindByUserID(ctx context.Context, userID string) ([]*Order, error)
	FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*Order, error)
	FindBySpecification(ctx context.Context, spec shared.Specification[*Order]) ([]*Order, error)
	Remove(ctx context.Context, id string) error
}
