package domain

import (
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrOrderNotFound = errors.New("order not found")
)

// UserDomainService 用户领域服务 - 处理跨实体的业务逻辑
type UserDomainService struct {
	userRepository UserRepository
	orderRepository OrderRepository
}

// NewUserDomainService 创建用户领域服务
func NewUserDomainService(userRepo UserRepository, orderRepo OrderRepository) *UserDomainService {
	return &UserDomainService{
		userRepository: userRepo,
		orderRepository: orderRepo,
	}
}

// CanUserPlaceOrder 检查用户是否可以下单
func (s *UserDomainService) CanUserPlaceOrder(userID string) (bool, error) {
	user, err := s.userRepository.FindByID(userID)
	if err != nil {
		return false, err
	}
	
	if !user.IsActive() {
		return false, ErrUserNotActive
	}
	
	if !user.CanMakePurchase() {
		return false, errors.New("user cannot make purchases")
	}
	
	// 检查用户是否有未完成的订单
	pendingOrders, err := s.orderRepository.FindByUserIDAndStatus(userID, OrderStatusPending)
	if err != nil {
		return false, err
	}
	
	// 如果用户有超过5个待处理订单，不允许继续下单
	if len(pendingOrders) >= 5 {
		return false, errors.New("user has too many pending orders")
	}
	
	return true, nil
}

// CalculateUserTotalSpent 计算用户总消费金额
func (s *UserDomainService) CalculateUserTotalSpent(userID string) (Money, error) {
	orders, err := s.orderRepository.FindByUserID(userID)
	if err != nil {
		return Money{}, err
	}
	
	total := NewMoney(0, "CNY")
	for _, order := range orders {
		if order.Status() == OrderStatusDelivered {
			total, _ = total.Add(order.TotalAmount())
		}
	}
	
	return *total, nil
}

// OrderDomainService 订单领域服务
type OrderDomainService struct {
	userRepository UserRepository
	orderRepository OrderRepository
}

// NewOrderDomainService 创建订单领域服务
func NewOrderDomainService(userRepo UserRepository, orderRepo OrderRepository) *OrderDomainService {
	return &OrderDomainService{
		userRepository: userRepo,
		orderRepository: orderRepo,
	}
}

// ProcessOrder 处理订单 - 完整的订单处理流程
func (s *OrderDomainService) ProcessOrder(orderID string) error {
	order, err := s.orderRepository.FindByID(orderID)
	if err != nil {
		return err
	}
	
	user, err := s.userRepository.FindByID(order.UserID())
	if err != nil {
		return err
	}
	
	// 验证用户状态
	if !user.IsActive() {
		return ErrUserNotActive
	}
	
	// 确认订单
	if err := order.Confirm(); err != nil {
		return err
	}
	
	// 保存更新后的订单
	if err := s.orderRepository.Save(order); err != nil {
		return err
	}
	
	return nil
}