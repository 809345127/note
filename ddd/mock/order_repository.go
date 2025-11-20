package mock

import (
	"ddd-example/domain"
	"errors"
	"sync"
)

// MockOrderRepository 订单仓储的Mock实现
type MockOrderRepository struct {
	orders map[string]*domain.Order
	mu     sync.RWMutex
}

// NewMockOrderRepository 创建Mock订单仓储
func NewMockOrderRepository() *MockOrderRepository {
	repo := &MockOrderRepository{
		orders: make(map[string]*domain.Order),
	}
	
	// 初始化一些测试数据
	repo.initializeTestData()
	return repo
}

// initializeTestData 初始化测试数据
func (r *MockOrderRepository) initializeTestData() {
	// 创建测试订单项
	item1 := domain.NewOrderItem("prod-1", "iPhone 15", 1, *domain.NewMoney(699900, "CNY"))
	item2 := domain.NewOrderItem("prod-2", "MacBook Pro", 1, *domain.NewMoney(1299900, "CNY"))
	item3 := domain.NewOrderItem("prod-3", "AirPods Pro", 2, *domain.NewMoney(199900, "CNY"))
	
	// 创建测试订单
	order1, _ := domain.NewOrder("user-1", []domain.OrderItem{item1, item2})
	order2, _ := domain.NewOrder("user-2", []domain.OrderItem{item3})
	order3, _ := domain.NewOrder("user-1", []domain.OrderItem{item1})
	
	// 设置不同的订单状态
	order1.Confirm()
	order1.Ship()
	
	order2.Confirm()
	
	// 设置固定ID以便测试
	r.orders["order-1"] = order1
	r.orders["order-2"] = order2
	r.orders["order-3"] = order3
}

func (r *MockOrderRepository) Save(order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.orders[order.ID()] = order
	return nil
}

func (r *MockOrderRepository) FindByID(id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	order, exists := r.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (r *MockOrderRepository) FindByUserID(userID string) ([]*domain.Order, error) {
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

func (r *MockOrderRepository) FindByUserIDAndStatus(userID string, status domain.OrderStatus) ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var orders []*domain.Order
	for _, order := range r.orders {
		if order.UserID() == userID && order.Status() == status {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *MockOrderRepository) FindAll() ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	orders := make([]*domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *MockOrderRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.orders, id)
	return nil
}