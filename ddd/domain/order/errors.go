/*
Package order 定义订单领域错误。
*/
package order

import (
	"ddd/domain/shared"
	"errors"
)

var (
	ErrOrderNotFound               = errors.New("order not found")
	ErrConcurrentModification      = errors.New("order was modified by another transaction, please retry")
	ErrInvalidOrderState           = errors.New("invalid order state transition")
	ErrEmptyOrderItems             = errors.New("order must have at least one item")
	ErrUserCannotPlaceOrder        = errors.New("user cannot place order")
	ErrInvalidQuantity             = errors.New("quantity must be positive")
	ErrOrderTotalAmountNotPositive = errors.New("order total amount must be positive")
	ErrCannotModifyNonPendingOrder = errors.New("can only modify pending orders")
	ErrItemNotFound                = errors.New("item not found")
	ErrInvalidOrderStateTransition = errors.New("invalid order state transition")
	ErrUserNotActiveForOrder       = errors.New("user is not active")
)

func NewOrderNotFoundError(orderID string) error {
	return &orderDomainError{
		sentinel: ErrOrderNotFound,
		entity:   "order",
		message:  "order not found: " + orderID,
		stack:    shared.CaptureStack(3),
	}
}

func NewConcurrentModificationError(orderID string) error {
	return &orderDomainError{
		sentinel: ErrConcurrentModification,
		entity:   "order",
		message:  "order " + orderID + " was modified by another transaction, please retry",
		stack:    shared.CaptureStack(3),
	}
}

func NewInvalidOrderStateError(currentState, targetState string) error {
	return &orderDomainError{
		sentinel: ErrInvalidOrderState,
		entity:   "order",
		message:  "cannot transition from " + currentState + " to " + targetState,
		stack:    shared.CaptureStack(3),
	}
}

func NewEmptyOrderItemsError() error {
	return &orderDomainError{
		sentinel: ErrEmptyOrderItems,
		entity:   "order",
		field:    "items",
		message:  "order must have at least one item",
		stack:    shared.CaptureStack(3),
	}
}

func NewUserCannotPlaceOrderError(userID, reason string) error {
	return &orderDomainError{
		sentinel: ErrUserCannotPlaceOrder,
		entity:   "order",
		message:  "user " + userID + " cannot place order: " + reason,
		stack:    shared.CaptureStack(3),
	}
}

type orderDomainError struct {
	sentinel error
	entity   string
	field    string
	message  string
	stack    []uintptr
}

func (e *orderDomainError) Error() string   { return e.message }
func (e *orderDomainError) Unwrap() error   { return e.sentinel }
func (e *orderDomainError) Stack() []string { return shared.FormatStack(e.stack) }
