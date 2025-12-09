/*
Package order Order subdomain - Core layer of DDD architecture

The domain layer is the core of the entire application, containing:
- Aggregate Roots: Entities that maintain consistency boundaries
- Entity: Objects with unique identity
- Value Objects: Immutable objects identified by their attributes
- Domain Services: Business logic spanning multiple entities
- Domain Events: Important events recorded in the business system
- Repository Interfaces: Abstraction for aggregate root persistence

DDD Core Principles:
1. Domain layer does not depend on any other layer (pure business logic)
2. All fields are private, behavior exposed through methods
3. Business rules encapsulated within entities and value objects
*/
package order

import (
	"errors"
	"time"

	"ddd-example/domain/shared"

	"github.com/google/uuid"
)

// Order Order aggregate root
// Order acts as the aggregate root, maintaining the consistency boundary of orders
// All modifications to Order and OrderItem must go through the Order aggregate root
type Order struct {
	id          string
	userID      string
	items       []OrderItem
	totalAmount shared.Money
	status      Status
	version     int // Optimistic lock version number for concurrency control
	createdAt   time.Time
	updatedAt   time.Time

	// Domain event list for recording events within the aggregate
	events []shared.DomainEvent
}

// OrderItem Order item - Entity within the aggregate (non-aggregate root)
// OrderItem is part of the aggregate, has no global unique identifier, can only be accessed through Order
type OrderItem struct {
	id          string // Unique identifier for OrderItem within the aggregate
	productID   string
	productName string
	quantity    int
	unitPrice   shared.Money
	subtotal    shared.Money
}

// Status Order status enum
type Status string

const (
	StatusPending   Status = "PENDING"   // Pending
	StatusConfirmed Status = "CONFIRMED" // Confirmed
	StatusShipped   Status = "SHIPPED"   // Shipped
	StatusDelivered Status = "DELIVERED" // Delivered
	StatusCancelled Status = "CANCELLED" // Cancelled
)

// PostOptions Create order options
type PostOptions struct {
	UserID string
	Items  []ItemRequest
}

// ItemRequest Create order item request
type ItemRequest struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
}

// ============================================================================
// Factory Methods - Creating Aggregate Roots
// ============================================================================
//
// DDD Principle: Use factory methods to create aggregate roots, not direct struct literals
// Advantages:
// 1. Encapsulates creation logic and validation rules
// 2. Ensures aggregate root is in valid state when created
// 3. Can record domain events during creation

// NewOrder Create new Order aggregate root
// This is the only entry point for creating Order, ensuring all business rules are met during order creation
func NewOrder(userID string, requests []ItemRequest) (*Order, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if len(requests) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	// Create order items
	items := make([]OrderItem, len(requests))
	for i, req := range requests {
		if req.Quantity <= 0 {
			return nil, errors.New("quantity must be positive")
		}

		items[i] = OrderItem{
			id:          uuid.New().String(),
			productID:   req.ProductID,
			productName: req.ProductName,
			quantity:    req.Quantity,
			unitPrice:   req.UnitPrice,
			subtotal:    *shared.NewMoney(req.UnitPrice.Amount()*int64(req.Quantity), req.UnitPrice.Currency()),
		}
	}

	// Calculate total amount
	totalAmount := shared.NewMoney(0, "CNY")
	var err error
	for _, item := range items {
		totalAmount, err = totalAmount.Add(item.subtotal)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	order := &Order{
		id:          uuid.New().String(),
		userID:      userID,
		items:       items,
		totalAmount: *totalAmount,
		status:      StatusPending,
		version:     0,
		createdAt:   now,
		updatedAt:   now,
		events:      make([]shared.DomainEvent, 0),
	}

	// Record domain event
	order.events = append(order.events, NewOrderPlacedEvent(order.id, userID, order.totalAmount))

	return order, nil
}

// ============================================================================
// ReconstructionDTO - For Repository Layer Use Only
// ============================================================================
//
// DDD Principle: Aggregate root reconstruction from database requires special handling
// Since fields are private, the repository layer needs a way to reconstruct aggregate roots
// Using DTO + Factory method pattern, rather than exposing setters or using reflection

// ReconstructionDTO Order reconstruction data transfer object
// Limited to repository layer usage, for reconstructing Order aggregate root from database
// This is a special design that maintains encapsulation of the domain model
// ⚠️ Note: This DTO should only be used in repository implementation, not called from application layer
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

// RebuildFromDTO Reconstruct Order aggregate root from DTO
// This is a factory method specifically for repository layer to reconstruct aggregate root
// ⚠️ Note: This method should only be used in repository implementation, not called from application layer
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
		events:      []shared.DomainEvent{},
	}
}

// ItemReconstructionDTO Order item reconstruction data transfer object
type ItemReconstructionDTO struct {
	ID          string
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   shared.Money
	Subtotal    shared.Money
}

// RebuildItemFromDTO Rebuild OrderItem from DTO
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

// ============================================================================
// Aggregate Root Behavior Methods - Managing Entities Within Aggregate
// ============================================================================
//
// DDD Principle: Entities within an aggregate (OrderItem) can only be accessed through the aggregate root (Order)
// External code cannot directly create or modify OrderItem

// AddItem Add order item through aggregate root
// This is an important DDD principle: aggregate internal entities can only be accessed and modified through the aggregate root
// Parameters: productID, productName string, quantity int, unitPrice shared.Money
func (o *Order) AddItem(productID, productName string, quantity int, unitPrice shared.Money) error {
	// Verify if current status allows modification
	if o.status != StatusPending {
		return errors.New("can only add items to pending orders")
	}

	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	// Create new order item
	item := OrderItem{
		id:          uuid.New().String(),
		productID:   productID,
		productName: productName,
		quantity:    quantity,
		unitPrice:   unitPrice,
		subtotal:    *shared.NewMoney(unitPrice.Amount()*int64(quantity), unitPrice.Currency()),
	}

	o.items = append(o.items, item)

	// Recalculate total amount
	newTotal := shared.NewMoney(0, "CNY")
	var err error
	for _, it := range o.items {
		newTotal, err = newTotal.Add(it.subtotal)
		if err != nil {
			// Rollback add operation
			o.items = o.items[:len(o.items)-1]
			return err
		}
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()

	return nil
}

// RemoveItem Remove order item through aggregate root
func (o *Order) RemoveItem(itemID string) error {
	if o.status != StatusPending {
		return errors.New("can only remove items from pending orders")
	}

	// Find and delete order item
	found := false
	for i, item := range o.items {
		if item.id == itemID {
			// Remove element from slice
			o.items = append(o.items[:i], o.items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return errors.New("item not found")
	}

	// Recalculate total amount
	newTotal := shared.NewMoney(0, "CNY")
	for _, item := range o.items {
		newTotal, _ = newTotal.Add(item.subtotal)
	}
	o.totalAmount = *newTotal
	o.updatedAt = time.Now()

	return nil
}

// ============================================================================
// State Change Methods - Domain Behavior
// ============================================================================
//
// DDD Principle: State changes must go through aggregate root methods, not direct field modification
// Benefits:
// 1. Encapsulates business rules (e.g., state transition restrictions)
// 2. Automatically maintains version numbers (optimistic locking)
// 3. Records domain events
// 4. Ensures aggregate internal consistency

// Confirm Confirm order (status from PENDING -> CONFIRMED)
// Business rule: Only pending orders can be confirmed
func (o *Order) Confirm() error {
	if o.status != StatusPending {
		return errors.New("only pending orders can be confirmed")
	}

	o.status = StatusConfirmed
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Cancel Cancel order
// Business rule: Delivered or cancelled orders cannot be cancelled again
func (o *Order) Cancel() error {
	if o.status == StatusDelivered || o.status == StatusCancelled {
		return errors.New("cannot cancel delivered or cancelled orders")
	}

	o.status = StatusCancelled
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Ship Ship order (status from CONFIRMED -> SHIPPED)
// Business rule: Only confirmed orders can be shipped
func (o *Order) Ship() error {
	if o.status != StatusConfirmed {
		return errors.New("only confirmed orders can be shipped")
	}

	o.status = StatusShipped
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// Deliver Deliver order (status from SHIPPED -> DELIVERED)
// Business rule: Only shipped orders can be marked as delivered
func (o *Order) Deliver() error {
	if o.status != StatusShipped {
		return errors.New("only shipped orders can be delivered")
	}

	o.status = StatusDelivered
	o.updatedAt = time.Now()
	o.version++

	return nil
}

// ============================================================================
// Getters - Read-only Accessors
// ============================================================================
//
// DDD Principle: Fields are private, exposed through getters for read-only access
// This ensures external code can only read state, not modify directly, maintaining encapsulation

func (o *Order) ID() string     { return o.id }
func (o *Order) UserID() string { return o.userID }

// Items Return copy of order items
// DDD Principle: Aggregate internal entities cannot be modified directly by external code, returning copy ensures encapsulation
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

// ============================================================================
// Domain Event Management
// ============================================================================
//
// DDD Principle: Aggregate root records domain events, UoW saves to outbox table
// Event flow: Aggregate state change → Record event → UoW collects → Save to outbox → Message Relay publishes asynchronously

// PullEvents Get and clear aggregate root's event list
// This is standard practice for domain event pattern:
// 1. Aggregate root records events when state changes
// 2. UoW calls PullEvents() in transaction to get events and save to outbox table
// 3. PullEvents clears event list to avoid duplicate saves
func (o *Order) PullEvents() []shared.DomainEvent {
	events := make([]shared.DomainEvent, len(o.events))
	copy(events, o.events)
	o.events = make([]shared.DomainEvent, 0)
	return events
}

// OrderItem Getters - Allow reading but no external modification

func (item OrderItem) ID() string          { return item.id }
func (item OrderItem) ProductID() string   { return item.productID }
func (item OrderItem) ProductName() string { return item.productName }
func (item OrderItem) Quantity() int       { return item.quantity }
func (item OrderItem) UnitPrice() shared.Money    { return item.unitPrice }
func (item OrderItem) Subtotal() shared.Money     { return item.subtotal }

// Compile-time check that Order implements AggregateRoot interface
var _ shared.AggregateRoot = (*Order)(nil)
