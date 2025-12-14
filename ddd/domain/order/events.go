package order

import (
	"time"

	"ddd/domain/shared"
)

// OrderPlacedEvent Order placed event
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

func (e *OrderPlacedEvent) EventName() string        { return "order.placed" }
func (e *OrderPlacedEvent) OccurredOn() time.Time    { return e.occurredOn }
func (e *OrderPlacedEvent) GetAggregateID() string   { return e.orderID }
func (e *OrderPlacedEvent) OrderID() string          { return e.orderID }
func (e *OrderPlacedEvent) UserID() string           { return e.userID }
func (e *OrderPlacedEvent) TotalAmount() shared.Money { return e.totalAmount }
