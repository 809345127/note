package domain

import (
	"time"
)

// DomainEvent 领域事件接口
type DomainEvent interface {
	EventName() string
	OccurredOn() time.Time
	GetAggregateID() string
}

// UserCreatedEvent 用户创建事件
type UserCreatedEvent struct {
	userID    string
	name      string
	email     string
	occurredOn time.Time
}

func NewUserCreatedEvent(userID, name, email string) *UserCreatedEvent {
	return &UserCreatedEvent{
		userID:     userID,
		name:       name,
		email:      email,
		occurredOn: time.Now(),
	}
}

func (e *UserCreatedEvent) EventName() string      { return "user.created" }
func (e *UserCreatedEvent) OccurredOn() time.Time { return e.occurredOn }
func (e *UserCreatedEvent) GetAggregateID() string { return e.userID }
func (e *UserCreatedEvent) UserID() string         { return e.userID }
func (e *UserCreatedEvent) Name() string           { return e.name }
func (e *UserCreatedEvent) Email() string          { return e.email }

// OrderPlacedEvent 订单创建事件
type OrderPlacedEvent struct {
	orderID    string
	userID     string
	totalAmount Money
	occurredOn time.Time
}

func NewOrderPlacedEvent(orderID, userID string, totalAmount Money) *OrderPlacedEvent {
	return &OrderPlacedEvent{
		orderID:    orderID,
		userID:     userID,
		totalAmount: totalAmount,
		occurredOn: time.Now(),
	}
}

func (e *OrderPlacedEvent) EventName() string      { return "order.placed" }
func (e *OrderPlacedEvent) OccurredOn() time.Time { return e.occurredOn }
func (e *OrderPlacedEvent) GetAggregateID() string { return e.orderID }
func (e *OrderPlacedEvent) OrderID() string         { return e.orderID }
func (e *OrderPlacedEvent) UserID() string          { return e.userID }
func (e *OrderPlacedEvent) TotalAmount() Money      { return e.totalAmount }