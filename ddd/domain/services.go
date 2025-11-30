/*
领域服务（Domain Service）

领域服务用于处理不适合放在单个实体中的业务逻辑，通常是：
1. 跨多个聚合根的业务规则验证
2. 需要访问多个仓储的复杂业务计算
3. 无状态的业务规则

核心原则：领域服务只读不写
┌─────────────────────────────────────────────────────────────────┐
│ 操作类型        │ DomainService      │ ApplicationService       │
├─────────────────────────────────────────────────────────────────┤
│ 简单查询        │ ⚠️ 可以，建议传入   │ ✅ 查询后传入             │
│ 业务逻辑查询    │ ✅ 可主动查询       │ ✅ 也可以                 │
│ Save/Update    │ ❌ 绝对禁止         │ ✅ 唯一负责               │
│ Delete         │ ❌ 绝对禁止         │ ✅ 唯一负责               │
└─────────────────────────────────────────────────────────────────┘

与应用服务的区别：
┌─────────────────────────────────────────────────────────────────┐
│ 领域服务 (Domain Service)                                        │
│ - 位于领域层                                                     │
│ - 处理跨实体的业务规则                                            │
│ - 可依赖 Repository 接口查询数据                                  │
│ - 不调用 Save/Update/Delete（只读不写）                           │
│ - 不负责事务管理                                                  │
│ - 不发布事件                                                     │
├─────────────────────────────────────────────────────────────────┤
│ 应用服务 (Application Service)                                   │
│ - 位于应用层                                                     │
│ - 编排业务流程                                                   │
│ - 调用领域服务进行验证                                            │
│ - 负责调用 Save/Update/Delete 持久化                              │
│ - 管理事务边界                                                   │
└─────────────────────────────────────────────────────────────────┘
*/
package domain

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrOrderNotFound = errors.New("order not found")
)

// ============================================================================
// 用户领域服务
// ============================================================================

// UserDomainService 用户领域服务 - 处理跨实体的业务逻辑
type UserDomainService struct {
	userRepository  UserRepository
	orderRepository OrderRepository
}

// NewUserDomainService 创建用户领域服务
func NewUserDomainService(userRepo UserRepository, orderRepo OrderRepository) *UserDomainService {
	return &UserDomainService{
		userRepository:  userRepo,
		orderRepository: orderRepo,
	}
}

// CanUserPlaceOrder 检查用户是否可以下单
func (s *UserDomainService) CanUserPlaceOrder(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive() {
		return false, ErrUserNotActive
	}

	if !user.CanMakePurchase() {
		return false, errors.New("user cannot make purchases")
	}

	// TODO: 检查用户未完成订单数量
	// pendingOrders, err := s.orderRepository.FindByUserIDAndStatus(ctx, userID, OrderStatusPending)
	// if err != nil {
	//     return false, err
	// }
	// if len(pendingOrders) >= 5 {
	//     return false, errors.New("user has too many pending orders")
	// }

	return true, nil
}

// CalculateUserTotalSpent 计算用户总消费金额
func (s *UserDomainService) CalculateUserTotalSpent(ctx context.Context, userID string) (Money, error) {
	orders, err := s.orderRepository.FindDeliveredOrdersByUserID(ctx, userID)
	if err != nil {
		return Money{}, err
	}

	total := NewMoney(0, "CNY")
	for _, order := range orders {
		total, _ = total.Add(order.TotalAmount())
	}

	return *total, nil
}

// ============================================================================
// 订单领域服务
// ============================================================================

// OrderDomainService 订单领域服务
// DDD原则：领域服务可依赖 Repository 接口查询数据，但不调用 Save 持久化
type OrderDomainService struct {
	userRepository  UserRepository
	orderRepository OrderRepository
}

// NewOrderDomainService 创建订单领域服务
func NewOrderDomainService(userRepo UserRepository, orderRepo OrderRepository) *OrderDomainService {
	return &OrderDomainService{
		userRepository:  userRepo,
		orderRepository: orderRepo,
	}
}

// CanProcessOrder 检查订单是否可以处理
// DDD原则：领域服务只做业务规则验证，返回验证结果
// 实际的状态变更和持久化由应用服务负责
func (s *OrderDomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := s.orderRepository.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepository.FindByID(ctx, order.UserID())
	if err != nil {
		return nil, err
	}

	// 验证用户状态
	if !user.IsActive() {
		return nil, ErrUserNotActive
	}

	// 验证订单状态
	if order.Status() != OrderStatusPending {
		return nil, errors.New("only pending orders can be processed")
	}

	return order, nil
}
