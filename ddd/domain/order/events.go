package order

import (
	"time"

	"ddd/domain/shared"
)

type OrderPlacedEvent struct {
	orderID     string
	userID      string
	totalAmount shared.Money
	occurredOn  time.Time
}

func NewOrderPlacedEvent(orderID, userID string, totalAmount shared.Money) *OrderPlacedEvent {
	return &OrderPlacedEvent{
		orderID:     orderID,
		userID:      userID,
		totalAmount: totalAmount,
		occurredOn:  time.Now(),
	}
}

func (e *OrderPlacedEvent) EventName() string         { return "order.placed" }
func (e *OrderPlacedEvent) OccurredOn() time.Time     { return e.occurredOn }
func (e *OrderPlacedEvent) GetAggregateID() string    { return e.orderID }
func (e *OrderPlacedEvent) OrderID() string           { return e.orderID }
func (e *OrderPlacedEvent) UserID() string            { return e.userID }
func (e *OrderPlacedEvent) TotalAmount() shared.Money { return e.totalAmount }

type OrderConfirmedEvent struct {
	orderID    string
	occurredOn time.Time
}

func NewOrderConfirmedEvent(orderID string) *OrderConfirmedEvent {
	return &OrderConfirmedEvent{
		orderID:    orderID,
		occurredOn: time.Now(),
	}
}

func (e *OrderConfirmedEvent) EventName() string      { return "order.confirmed" }
func (e *OrderConfirmedEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *OrderConfirmedEvent) GetAggregateID() string { return e.orderID }
func (e *OrderConfirmedEvent) OrderID() string        { return e.orderID }

type OrderShippedEvent struct {
	orderID    string
	occurredOn time.Time
}

func NewOrderShippedEvent(orderID string) *OrderShippedEvent {
	return &OrderShippedEvent{
		orderID:    orderID,
		occurredOn: time.Now(),
	}
}

func (e *OrderShippedEvent) EventName() string      { return "order.shipped" }
func (e *OrderShippedEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *OrderShippedEvent) GetAggregateID() string { return e.orderID }
func (e *OrderShippedEvent) OrderID() string        { return e.orderID }

type OrderDeliveredEvent struct {
	orderID    string
	occurredOn time.Time
}

func NewOrderDeliveredEvent(orderID string) *OrderDeliveredEvent {
	return &OrderDeliveredEvent{
		orderID:    orderID,
		occurredOn: time.Now(),
	}
}

func (e *OrderDeliveredEvent) EventName() string      { return "order.delivered" }
func (e *OrderDeliveredEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *OrderDeliveredEvent) GetAggregateID() string { return e.orderID }
func (e *OrderDeliveredEvent) OrderID() string        { return e.orderID }

type OrderCancelledEvent struct {
	orderID    string
	reason     string
	occurredOn time.Time
}

func NewOrderCancelledEvent(orderID, reason string) *OrderCancelledEvent {
	return &OrderCancelledEvent{
		orderID:    orderID,
		reason:     reason,
		occurredOn: time.Now(),
	}
}

func (e *OrderCancelledEvent) EventName() string      { return "order.cancelled" }
func (e *OrderCancelledEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *OrderCancelledEvent) GetAggregateID() string { return e.orderID }
func (e *OrderCancelledEvent) OrderID() string        { return e.orderID }
func (e *OrderCancelledEvent) Reason() string         { return e.reason }
