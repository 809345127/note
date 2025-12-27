package mocks

import (
	"context"
	"sync"

	"ddd/domain/order"
	"ddd/domain/shared"

	"github.com/google/uuid"
)

// MockOrderRepository Mock implementation of order repository
// DDD principle: Repository is only responsible for persistence of aggregate roots, not event publishing
// Events are saved to outbox table by UoW and published asynchronously by background services
type MockOrderRepository struct {
	orders map[string]*order.Order
	mu     sync.RWMutex
}

// NewMockOrderRepository Create Mock order repository
func NewMockOrderRepository() *MockOrderRepository {
	repo := &MockOrderRepository{
		orders: make(map[string]*order.Order),
	}

	// Initialize some test data
	repo.initializeTestData()
	return repo
}

// initializeTestData Initialize test data
func (r *MockOrderRepository) initializeTestData() {
	// Create test orders
	order1 := r.createTestOrder("order-1", "user-1", "order-1-items")
	order2 := r.createTestOrder("order-2", "user-2", "order-2-items")
	order3 := r.createTestOrder("order-3", "user-1", "order-3-items")

	if order1 != nil && order2 != nil && order3 != nil {
		// Set different order statuses
		order1.Confirm()
		order1.Ship()

		order2.Confirm()

		// Use the actual order ID as key (instead of hardcoded key)
		r.orders[order1.ID()] = order1
		r.orders[order2.ID()] = order2
		r.orders[order3.ID()] = order3
	}
}

// createTestOrder Create test order (only used for Mock data initialization)
func (r *MockOrderRepository) createTestOrder(id, userID, itemsType string) *order.Order {
	var requests []order.ItemRequest

	switch itemsType {
	case "order-1-items":
		// iPhone 15 + MacBook Pro
		requests = []order.ItemRequest{
			{
				ProductID:   "prod-1",
				ProductName: "iPhone 15",
				Quantity:    1,
				UnitPrice:   *shared.NewMoney(699900, "CNY"),
			},
			{
				ProductID:   "prod-2",
				ProductName: "MacBook Pro",
				Quantity:    1,
				UnitPrice:   *shared.NewMoney(1299900, "CNY"),
			},
		}
	case "order-2-items":
		// 2 AirPods Pro
		requests = []order.ItemRequest{
			{
				ProductID:   "prod-3",
				ProductName: "AirPods Pro",
				Quantity:    2,
				UnitPrice:   *shared.NewMoney(199900, "CNY"),
			},
		}
	case "order-3-items":
		// 1 iPhone 15
		requests = []order.ItemRequest{
			{
				ProductID:   "prod-1",
				ProductName: "iPhone 15",
				Quantity:    1,
				UnitPrice:   *shared.NewMoney(699900, "CNY"),
			},
		}
	}

	o, err := order.NewOrder(userID, requests)
	if err != nil {
		return nil
	}
	return o
}

// NextIdentity Generate new order ID
func (r *MockOrderRepository) NextIdentity() string {
	return "order-" + uuid.New().String()
}

func (r *MockOrderRepository) Save(ctx context.Context, o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check optimistic locking for existing orders
	if !o.IsNew() {
		existing, exists := r.orders[o.ID()]
		if exists && existing.Version() != o.Version() {
			return order.ErrConcurrentModification
		}
	}

	r.orders[o.ID()] = o

	// Clear dirty tracking after successful save
	o.ClearDirtyTracking()

	// Note: Do not publish events in repository!
	// Events are saved to outbox table by UoW before transaction commit
	// Background OutboxProcessor publishes to message queue asynchronously

	return nil
}

func (r *MockOrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	o, exists := r.orders[id]
	if !exists {
		// 使用带堆栈的错误构造函数
		return nil, order.NewOrderNotFoundError(id)
	}
	return o, nil
}

func (r *MockOrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := order.ByUserIDSpecification{UserID: userID}
	return r.FindBySpecification(ctx, spec)
}

func (r *MockOrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := shared.And(
		order.ByUserIDSpecification{UserID: userID},
		order.ByStatusSpecification{Status: order.StatusDelivered},
	)
	return r.FindBySpecification(ctx, spec)
}

func (r *MockOrderRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*order.Order]) ([]*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var orders []*order.Order
	for _, o := range r.orders {
		if spec.IsSatisfiedBy(ctx, o) {
			orders = append(orders, o)
		}
	}
	return orders, nil
}

// Remove Logical deletion of order
func (r *MockOrderRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	o, exists := r.orders[id]
	if !exists {
		return order.ErrOrderNotFound
	}

	// Logical deletion: mark as cancelled
	return o.Cancel()
}
