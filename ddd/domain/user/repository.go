package user

import "context"

// Repository User repository interface
// DDD principles:
// 1. Repository only responsible for aggregate root persistence
// 2. Should not expose batch queries (like FindAll), such operations should be in query service
// 3. Use NextIdentity to generate ID, not directly in entity (facilitates testing and ID strategy adjustment)
// 4. Include context.Context to support timeout, cancellation and transaction
type Repository interface {
	// NextIdentity Generate new user ID (DDD recommends generating ID in repository)
	NextIdentity() string

	// Save Save or update user aggregate root (including all entities within aggregate)
	// If user.Version() == 0 means create, else update
	Save(ctx context.Context, user *User) error

	// FindByID Find user aggregate root by ID
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail Find user by email (business uniqueness constraint)
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Remove Logically delete user aggregate root (DDD recommends logical delete over physical delete)
	Remove(ctx context.Context, id string) error
}

// QueryService Query service interface (Q-side in CQRS pattern)
// DDD distinction: Command (modify) and Query (read) should be separated
// Repository handles command operations (load aggregate root, save aggregate root)
// Query service handles complex queries, not limited by aggregate boundaries
type QueryService interface {
	// SearchUsers Search users (supports pagination, sorting)
	SearchUsers(criteria SearchCriteria) ([]*User, error)

	// CountUsers Count users
	CountUsers(criteria SearchCriteria) (int, error)
}

// SearchCriteria Generic search criteria
type SearchCriteria struct {
	Filters   map[string]interface{}
	SortBy    string
	SortOrder string // ASC or DESC
	Page      int
	PageSize  int
}
