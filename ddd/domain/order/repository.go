package order

import "context"

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

	// Remove Logically delete order aggregate root
	Remove(ctx context.Context, id string) error
}

// QueryService Query service interface (Q-side in CQRS pattern)
type QueryService interface {
	// SearchOrders Search orders
	SearchOrders(criteria SearchCriteria) ([]*Order, error)
}

// SearchCriteria Generic query criteria
type SearchCriteria struct {
	Filters   map[string]interface{}
	SortBy    string
	SortOrder string // ASC or DESC
	Page      int
	PageSize  int
}
