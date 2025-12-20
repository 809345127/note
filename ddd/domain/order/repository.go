package order

import (
	"context"

	"ddd/domain/shared"
)

// Repository Order repository interface
type Repository interface {
	// NextIdentity Generate new order ID
	NextIdentity() string

	// Save Save or update order aggregate root
	// If order.Version() == 0 means create, else update
	// Repository only handles persistence, events collected by UoW and saved to outbox table
	Save(ctx context.Context, order *Order) error

	// FindByID Find order aggregate root by ID
	FindByID(ctx context.Context, id string) (*Order, error)

	// FindByUserID Find user's orders (controlled query)
	FindByUserID(ctx context.Context, userID string) ([]*Order, error)

	// FindDeliveredOrdersByUserID Find user's delivered orders (controlled query in CQRS)
	FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*Order, error)

	// FindBySpecification Find orders by specification
	// Allows flexible query composition without repository method explosion
	FindBySpecification(ctx context.Context, spec shared.Specification[*Order]) ([]*Order, error)

	// Remove Logically delete order aggregate root
	Remove(ctx context.Context, id string) error
}
