package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Order 订单实体
type Order struct {
	id          string
	userID      string
	items       []OrderItem
	totalAmount Money
	status      OrderStatus
	createdAt   time.Time
	updatedAt   time.Time
}

// OrderItem 订单项
type OrderItem struct {
	productID string
	productName string
	quantity  int
	unitPrice Money
	subtotal  Money
}

// OrderStatus 订单状态枚举
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// NewOrder 创建新订单
func NewOrder(userID string, items []OrderItem) (*Order, error) {
	if len(items) == 0 {
		return nil, errors.New("order must have at least one item")
	}
	
	// 计算总金额
	totalAmount := NewMoney(0, "CNY")
	for _, item := range items {
		totalAmount, _ = totalAmount.Add(item.subtotal)
	}
	
	now := time.Now()
	return &Order{
		id:          uuid.New().String(),
		userID:      userID,
		items:       items,
		totalAmount: *totalAmount,
		status:      OrderStatusPending,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// 订单领域行为
func (o *Order) Confirm() error {
	if o.status != OrderStatusPending {
		return errors.New("only pending orders can be confirmed")
	}
	o.status = OrderStatusConfirmed
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) Cancel() error {
	if o.status == OrderStatusDelivered || o.status == OrderStatusCancelled {
		return errors.New("cannot cancel delivered or cancelled orders")
	}
	o.status = OrderStatusCancelled
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) Ship() error {
	if o.status != OrderStatusConfirmed {
		return errors.New("only confirmed orders can be shipped")
	}
	o.status = OrderStatusShipped
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) Deliver() error {
	if o.status != OrderStatusShipped {
		return errors.New("only shipped orders can be delivered")
	}
	o.status = OrderStatusDelivered
	o.updatedAt = time.Now()
	return nil
}

// 获取器方法
func (o *Order) ID() string          { return o.id }
func (o *Order) UserID() string      { return o.userID }
func (o *Order) Items() []OrderItem  { return o.items }
func (o *Order) TotalAmount() Money  { return o.totalAmount }
func (o *Order) Status() OrderStatus { return o.status }
func (o *Order) CreatedAt() time.Time { return o.createdAt }
func (o *Order) UpdatedAt() time.Time { return o.updatedAt }

// OrderItem 获取器方法
func (item OrderItem) ProductID() string { return item.productID }
func (item OrderItem) ProductName() string { return item.productName }
func (item OrderItem) Quantity() int { return item.quantity }
func (item OrderItem) UnitPrice() Money { return item.unitPrice }
func (item OrderItem) Subtotal() Money { return item.subtotal }

// NewOrderItem 创建订单项
func NewOrderItem(productID, productName string, quantity int, unitPrice Money) OrderItem {
	subtotal := NewMoney(unitPrice.Amount()*int64(quantity), unitPrice.Currency())
	return OrderItem{
		productID:   productID,
		productName: productName,
		quantity:    quantity,
		unitPrice:   unitPrice,
		subtotal:    *subtotal,
	}
}