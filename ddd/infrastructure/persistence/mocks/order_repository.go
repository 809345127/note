package mocks

import (
	"context"
	"ddd-example/domain"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
)

// MockOrderRepository 订单仓储的Mock实现
// DDD原则：仓储负责聚合根的持久化，并在保存后发布领域事件
type MockOrderRepository struct {
	orders         map[string]*domain.Order
	mu             sync.RWMutex
	eventPublisher domain.DomainEventPublisher
}

// NewMockOrderRepository 创建Mock订单仓储
// eventPublisher: 事件发布器，仓储在Save后发布聚合根产生的事件
func NewMockOrderRepository(eventPublisher domain.DomainEventPublisher) *MockOrderRepository {
	repo := &MockOrderRepository{
		orders:         make(map[string]*domain.Order),
		eventPublisher: eventPublisher,
	}

	// 初始化一些测试数据
	repo.initializeTestData()
	return repo
}

// initializeTestData 初始化测试数据
func (r *MockOrderRepository) initializeTestData() {
	// 创建测试订单
	order1 := r.createTestOrder("order-1", "user-1", "order-1-items")
	order2 := r.createTestOrder("order-2", "user-2", "order-2-items")
	order3 := r.createTestOrder("order-3", "user-1", "order-3-items")

	if order1 != nil && order2 != nil && order3 != nil {
		// 设置不同的订单状态
		order1.Confirm()
		order1.Ship()

		order2.Confirm()

		r.orders["order-1"] = order1
		r.orders["order-2"] = order2
		r.orders["order-3"] = order3
	}
}

// createTestOrder 创建测试订单（仅用于Mock数据初始化）
func (r *MockOrderRepository) createTestOrder(id, userID, itemsType string) *domain.Order {
	var requests []domain.OrderItemRequest

	switch itemsType {
	case "order-1-items":
		// iPhone 15 + MacBook Pro
		requests = []domain.OrderItemRequest{
			{
				ProductID:   "prod-1",
				ProductName: "iPhone 15",
				Quantity:    1,
				UnitPrice:   *domain.NewMoney(699900, "CNY"),
			},
			{
				ProductID:   "prod-2",
				ProductName: "MacBook Pro",
				Quantity:    1,
				UnitPrice:   *domain.NewMoney(1299900, "CNY"),
			},
		}
	case "order-2-items":
		// 2个AirPods Pro
		requests = []domain.OrderItemRequest{
			{
				ProductID:   "prod-3",
				ProductName: "AirPods Pro",
				Quantity:    2,
				UnitPrice:   *domain.NewMoney(199900, "CNY"),
			},
		}
	case "order-3-items":
		// 1个iPhone 15
		requests = []domain.OrderItemRequest{
			{
				ProductID:   "prod-1",
				ProductName: "iPhone 15",
				Quantity:    1,
				UnitPrice:   *domain.NewMoney(699900, "CNY"),
			},
		}
	}

	order, err := domain.NewOrder(userID, requests)
	if err != nil {
		return nil
	}
	return order
}

// NextIdentity 生成新的订单ID
func (r *MockOrderRepository) NextIdentity() string {
	return "order-" + uuid.New().String()
}

func (r *MockOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.orders[order.ID()] = order

	// DDD原则：仓储在保存成功后发布聚合根产生的领域事件
	r.publishEvents(order.PullEvents())

	return nil
}

// publishEvents 发布领域事件
func (r *MockOrderRepository) publishEvents(events []domain.DomainEvent) {
	if r.eventPublisher == nil {
		return
	}
	for _, event := range events {
		if err := r.eventPublisher.Publish(event); err != nil {
			log.Printf("[WARN] Failed to publish event %s: %v", event.EventName(), err)
		}
	}
}

func (r *MockOrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (r *MockOrderRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var orders []*domain.Order
	for _, order := range r.orders {
		if order.UserID() == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *MockOrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var orders []*domain.Order
	for _, order := range r.orders {
		if order.UserID() == userID && order.Status() == domain.OrderStatusDelivered {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

// Remove 逻辑删除订单
func (r *MockOrderRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, exists := r.orders[id]
	if !exists {
		return errors.New("order not found")
	}

	// 逻辑删除：标记为已取消
	return order.Cancel()
}