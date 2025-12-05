package mocks

import (
	"context"
	"errors"
	"sync"

	"ddd-example/domain/order"
	"ddd-example/domain/shared"

	"github.com/google/uuid"
)

// MockOrderRepository 订单仓储的Mock实现
// DDD原则：仓储只负责聚合根的持久化，不负责发布事件
// 事件发布由 UoW 保存到 outbox 表，后台服务异步发布
type MockOrderRepository struct {
	orders map[string]*order.Order
	mu     sync.RWMutex
}

// NewMockOrderRepository 创建Mock订单仓储
func NewMockOrderRepository() *MockOrderRepository {
	repo := &MockOrderRepository{
		orders: make(map[string]*order.Order),
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

		// 使用订单的实际ID作为key（而不是硬编码的key）
		r.orders[order1.ID()] = order1
		r.orders[order2.ID()] = order2
		r.orders[order3.ID()] = order3
	}
}

// createTestOrder 创建测试订单（仅用于Mock数据初始化）
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
		// 2个AirPods Pro
		requests = []order.ItemRequest{
			{
				ProductID:   "prod-3",
				ProductName: "AirPods Pro",
				Quantity:    2,
				UnitPrice:   *shared.NewMoney(199900, "CNY"),
			},
		}
	case "order-3-items":
		// 1个iPhone 15
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

// NextIdentity 生成新的订单ID
func (r *MockOrderRepository) NextIdentity() string {
	return "order-" + uuid.New().String()
}

func (r *MockOrderRepository) Save(ctx context.Context, o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.orders[o.ID()] = o

	// 注意：不在仓储中发布事件！
	// 事件由 UoW 在事务提交前保存到 outbox 表
	// 后台 OutboxProcessor 异步发布到消息队列

	return nil
}

func (r *MockOrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	o, exists := r.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	return o, nil
}

func (r *MockOrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var orders []*order.Order
	for _, o := range r.orders {
		if o.UserID() == userID {
			orders = append(orders, o)
		}
	}
	return orders, nil
}

func (r *MockOrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var orders []*order.Order
	for _, o := range r.orders {
		if o.UserID() == userID && o.Status() == order.StatusDelivered {
			orders = append(orders, o)
		}
	}
	return orders, nil
}

// Remove 逻辑删除订单
func (r *MockOrderRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	o, exists := r.orders[id]
	if !exists {
		return errors.New("order not found")
	}

	// 逻辑删除：标记为已取消
	return o.Cancel()
}
