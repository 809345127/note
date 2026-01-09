package user

import (
	"context"

	"ddd/domain/shared"
)

// Repository User repository interface
// DDD principles:
// 1. Repository only responsible for aggregate root persistence
// 2. Should not expose batch queries (like FindAll), such operations should be in query service
// 3. Include context.Context to support timeout, cancellation and transaction
type Repository interface {
	// Save Save or update user aggregate root (including all entities within aggregate)
	// If user.Version() == 0 means create, else update
	Save(ctx context.Context, user *User) error

	// FindByID Find user aggregate root by ID
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail Find user by email (business uniqueness constraint)
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindBySpecification Find users by specification
	// Allows flexible query composition without repository method explosion
	FindBySpecification(ctx context.Context, spec shared.Specification[*User]) ([]*User, error)

	// Remove Logically delete user aggregate root (DDD recommends logical delete over physical delete)
	Remove(ctx context.Context, id string) error
}
