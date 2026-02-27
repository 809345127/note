/*
Package order 定义订单聚合根及其行为。

说明：
- 订单及订单项的状态变更必须经由聚合根方法完成。
- 仓储重建聚合时使用 DTO 方法，不走业务构造函数。
*/
package order

import (
	"fmt"
	"time"

	"ddd/domain/shared"

	"github.com/google/uuid"
)

type Order struct {
	id           string
	userID       string
	items        []OrderItem
	totalAmount  shared.Money
	status       Status
	version      int
	createdAt    time.Time
	updatedAt    time.Time
	events       []shared.DomainEvent
	addedItems   []OrderItem
	removedItems []OrderItem
	isNew        bool
}

type OrderItem struct {
	id          string
	productID   string
	productName string
	quantity    int
	unitPrice   shared.Money
	subtotal    shared.Money
}

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusConfirmed Status = "CONFIRMED"
	StatusShipped   Status = "SHIPPED"
	StatusDelivered Status = "DELIVERED"
	StatusCancelled Status = "CANCELLED"
)

type ItemRequest struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
}

// NewOrder 创建新订单聚合。
func NewOrder(userID string, requests []ItemRequest) (*Order, error) {
	if userID == "" {
		return nil, ErrInvalidOrderState
	}
	if len(requests) == 0 {
		return nil, ErrEmptyOrderItems
	}

	items := make([]OrderItem, len(requests))
	for i, req := range requests {
		if req.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		subtotal, err := req.UnitPrice.Multiply(req.Quantity)
		if err != nil {
			return nil, err
		}

		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("failed to generate order item ID: %w", err)
		}

		items[i] = OrderItem{
			id:          id.String(),
			productID:   req.ProductID,
			productName: req.ProductName,
			quantity:    req.Quantity,
			unitPrice:   req.UnitPrice,
			subtotal:    *subtotal,
		}
	}

	totalAmount := shared.NewMoney(0, "CNY")
	var err error
	for _, item := range items {
		totalAmount, err = totalAmount.Add(item.subtotal)
		if err != nil {
			return nil, err
		}
	}
	if totalAmount.Amount() <= 0 {
		return nil, ErrOrderTotalAmountNotPositive
	}

	orderID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate order ID: %w", err)
	}

	now := time.Now()
	o := &Order{
		id:           orderID.String(),
		userID:       userID,
		items:        items,
		totalAmount:  *totalAmount,
		status:       StatusPending,
		version:      0,
		createdAt:    now,
		updatedAt:    now,
		events:       make([]shared.DomainEvent, 0),
		addedItems:   nil,
		removedItems: nil,
		isNew:        true,
	}
	o.events = append(o.events, NewOrderPlacedEvent(o.id, userID, o.totalAmount))
	return o, nil
}

// ReconstructionDTO 仅供仓储层重建聚合使用。
type ReconstructionDTO struct {
	ID          string
	UserID      string
	Items       []OrderItem
	TotalAmount shared.Money
	Status      Status
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RebuildFromDTO 仅供仓储层调用。
func RebuildFromDTO(dto ReconstructionDTO) *Order {
	return &Order{
		id:          dto.ID,
		userID:      dto.UserID,
		items:       dto.Items,
		totalAmount: dto.TotalAmount,
		status:      dto.Status,
		version:     dto.Version,
		createdAt:   dto.CreatedAt,
		updatedAt:   dto.UpdatedAt,
		events:      nil,
		isNew:       false,
	}
}

type ItemReconstructionDTO struct {
	ID          string
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
	Subtotal    shared.Money
}

func RebuildItemFromDTO(dto ItemReconstructionDTO) OrderItem {
	return OrderItem{
		id:          dto.ID,
		productID:   dto.ProductID,
		productName: dto.ProductName,
		quantity:    dto.Quantity,
		unitPrice:   dto.UnitPrice,
		subtotal:    dto.Subtotal,
	}
}

func (o *Order) AddItem(productID, productName string, quantity int, unitPrice shared.Money) error {
	if o.status != StatusPending {
		return ErrCannotModifyNonPendingOrder
	}
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	subtotal, calcErr := unitPrice.Multiply(quantity)
	if calcErr != nil {
		return calcErr
	}

	id, idErr := uuid.NewV7()
	if idErr != nil {
		return fmt.Errorf("failed to generate order item ID: %w", idErr)
	}

	item := OrderItem{
		id:          id.String(),
		productID:   productID,
		productName: productName,
		quantity:    quantity,
		unitPrice:   unitPrice,
		subtotal:    *subtotal,
	}
	o.items = append(o.items, item)
	if !o.isNew {
		o.addedItems = append(o.addedItems, item)
	}

	newTotal := shared.NewMoney(0, "CNY")
	var err error
	for _, it := range o.items {
		newTotal, err = newTotal.Add(it.subtotal)
		if err != nil {
			o.items = o.items[:len(o.items)-1]
			if !o.isNew {
				o.addedItems = o.addedItems[:len(o.addedItems)-1]
			}
			return err
		}
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) RemoveItem(itemID string) error {
	if o.status != StatusPending {
		return ErrCannotModifyNonPendingOrder
	}

	found := false
	var removedItem OrderItem
	for i, item := range o.items {
		if item.id == itemID {
			removedItem = item
			o.items = append(o.items[:i], o.items[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return ErrItemNotFound
	}

	if !o.isNew {
		wasAddedInSession := false
		for i, added := range o.addedItems {
			if added.id == itemID {
				o.addedItems = append(o.addedItems[:i], o.addedItems[i+1:]...)
				wasAddedInSession = true
				break
			}
		}
		if !wasAddedInSession {
			o.removedItems = append(o.removedItems, removedItem)
		}
	}

	newTotal := shared.NewMoney(0, "CNY")
	for _, item := range o.items {
		var err error
		newTotal, err = newTotal.Add(item.subtotal)
		if err != nil {
			return err
		}
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()
	return nil
}

func (o *Order) Confirm() error {
	if o.status != StatusPending {
		return ErrInvalidOrderStateTransition
	}
	o.status = StatusConfirmed
	o.updatedAt = time.Now()
	o.events = append(o.events, NewOrderConfirmedEvent(o.id))
	return nil
}

func (o *Order) Cancel(reason string) error {
	if o.status == StatusDelivered || o.status == StatusCancelled {
		return ErrInvalidOrderStateTransition
	}
	o.status = StatusCancelled
	o.updatedAt = time.Now()
	o.events = append(o.events, NewOrderCancelledEvent(o.id, reason))
	return nil
}

func (o *Order) Ship() error {
	if o.status != StatusConfirmed {
		return ErrInvalidOrderStateTransition
	}
	o.status = StatusShipped
	o.updatedAt = time.Now()
	o.events = append(o.events, NewOrderShippedEvent(o.id))
	return nil
}

func (o *Order) Deliver() error {
	if o.status != StatusShipped {
		return ErrInvalidOrderStateTransition
	}
	o.status = StatusDelivered
	o.updatedAt = time.Now()
	o.events = append(o.events, NewOrderDeliveredEvent(o.id))
	return nil
}

func (o *Order) IncrementVersionForSave() {
	o.version++
	o.updatedAt = time.Now()
}

func (o *Order) ID() string     { return o.id }
func (o *Order) UserID() string { return o.userID }

func (o *Order) Items() []OrderItem {
	items := make([]OrderItem, len(o.items))
	copy(items, o.items)
	return items
}

func (o *Order) TotalAmount() shared.Money { return o.totalAmount }
func (o *Order) Status() Status            { return o.status }
func (o *Order) Version() int              { return o.version }
func (o *Order) CreatedAt() time.Time      { return o.createdAt }
func (o *Order) UpdatedAt() time.Time      { return o.updatedAt }

// 以下方法仅供仓储层使用。
func (o *Order) IsNew() bool { return o.isNew }

func (o *Order) AddedItems() []OrderItem {
	items := make([]OrderItem, len(o.addedItems))
	copy(items, o.addedItems)
	return items
}

func (o *Order) RemovedItems() []OrderItem {
	items := make([]OrderItem, len(o.removedItems))
	copy(items, o.removedItems)
	return items
}

func (o *Order) ClearDirtyTracking() {
	o.addedItems = nil
	o.removedItems = nil
	o.isNew = false
}

func (o *Order) PullEvents() []shared.DomainEvent {
	events := make([]shared.DomainEvent, len(o.events))
	copy(events, o.events)
	o.events = make([]shared.DomainEvent, 0)
	return events
}

func (item OrderItem) ID() string              { return item.id }
func (item OrderItem) ProductID() string       { return item.productID }
func (item OrderItem) ProductName() string     { return item.productName }
func (item OrderItem) Quantity() int           { return item.quantity }
func (item OrderItem) UnitPrice() shared.Money { return item.unitPrice }
func (item OrderItem) Subtotal() shared.Money  { return item.subtotal }

var _ shared.AggregateRoot = (*Order)(nil)
