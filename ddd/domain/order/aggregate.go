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
	"fmt"
	"time"

	"ddd/domain/shared"

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

	// Dirty tracking for efficient persistence
	// These track changes since the aggregate was loaded/created
	addedItems   []OrderItem // Items added since load
	removedItems []OrderItem // Items removed since load
	isNew        bool        // True if this aggregate was newly created (not loaded from DB)
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
		return nil, ErrInvalidOrderState
	}

	if len(requests) == 0 {
		return nil, ErrEmptyOrderItems
	}

	// Create order items
	items := make([]OrderItem, len(requests))
	for i, req := range requests {
		if req.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}

		// Calculate subtotal with overflow check using Money.Multiply
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

	// Calculate total amount
	totalAmount := shared.NewMoney(0, "CNY")
	var err error
	for _, item := range items {
		totalAmount, err = totalAmount.Add(item.subtotal)
		if err != nil {
			return nil, err
		}
	}

	// Validate total amount is positive
	if totalAmount.Amount() <= 0 {
		return nil, ErrOrderTotalAmountNotPositive
	}

	// Generate UUID with proper error handling
	orderID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate order ID: %w", err)
	}

	now := time.Now()
	order := &Order{
		id:          orderID.String(),
		userID:      userID,
		items:       items,
		totalAmount: *totalAmount,
		status:      StatusPending,
		version:     0,
		createdAt:   now,
		updatedAt:   now,
		events:      make([]shared.DomainEvent, 0),
		// Dirty tracking: new aggregate has no changes to track yet
		addedItems:   nil,
		removedItems: nil,
		isNew:        true, // Mark as newly created
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
		events:      nil, // nil is more idiomatic than empty slice
		isNew:       false, // Mark as existing aggregate (not newly created)
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
		return ErrCannotModifyNonPendingOrder
	}

	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Calculate subtotal with overflow check using Money.Multiply
	subtotal, calcErr := unitPrice.Multiply(quantity)
	if calcErr != nil {
		return calcErr
	}

	id, idErr := uuid.NewV7()
	if idErr != nil {
		return fmt.Errorf("failed to generate order item ID: %w", idErr)
	}

	// Create new order item
	item := OrderItem{
		id:          id.String(),
		productID:   productID,
		productName: productName,
		quantity:    quantity,
		unitPrice:   unitPrice,
		subtotal:    *subtotal,
	}

	o.items = append(o.items, item)

	// Track addition for dirty tracking (only if not a new aggregate)
	// New aggregates will insert all items anyway, no need to track
	if !o.isNew {
		o.addedItems = append(o.addedItems, item)
	}

	// Recalculate total amount
	newTotal := shared.NewMoney(0, "CNY")
	var err error
	for _, it := range o.items {
		newTotal, err = newTotal.Add(it.subtotal)
		if err != nil {
			// Rollback add operation
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

// RemoveItem Remove order item through aggregate root
func (o *Order) RemoveItem(itemID string) error {
	if o.status != StatusPending {
		return ErrCannotModifyNonPendingOrder
	}

	// Find and delete order item
	found := false
	var removedItem OrderItem
	for i, item := range o.items {
		if item.id == itemID {
			removedItem = item
			// Remove element from slice
			o.items = append(o.items[:i], o.items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return ErrItemNotFound
	}

	// Track removal for dirty tracking (only if not a new aggregate)
	// For new aggregates, item was never persisted, so no need to track removal
	// Also check if the item was in addedItems (added then removed in same session)
	if !o.isNew {
		// Check if this item was added in current session
		wasAddedInSession := false
		for i, added := range o.addedItems {
			if added.id == itemID {
				// Remove from addedItems instead of tracking as removed
				o.addedItems = append(o.addedItems[:i], o.addedItems[i+1:]...)
				wasAddedInSession = true
				break
			}
		}
		// Only track as removed if it was originally loaded from DB
		if !wasAddedInSession {
			o.removedItems = append(o.removedItems, removedItem)
		}
	}

	// Recalculate total amount
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

// ============================================================================
// State Change Methods - Domain Behavior
// ============================================================================
//
// DDD Principle: State changes must go through aggregate root methods, not direct field modification
// Benefits:
// 1. Encapsulates business rules (e.g., state transition restrictions)
// 2. State changes do NOT immediately increment version - version is managed by persistence layer
// 3. Records domain events
// 4. Ensures aggregate internal consistency
// 5. Version is incremented after successful persistence (see incrementVersionForSave)

// Confirm Confirm order (status from PENDING -> CONFIRMED)
// Business rule: Only pending orders can be confirmed
func (o *Order) Confirm() error {
	if o.status != StatusPending {
		return ErrInvalidOrderStateTransition
	}

	o.status = StatusConfirmed
	o.updatedAt = time.Now()
	// Record domain event
	o.events = append(o.events, NewOrderConfirmedEvent(o.id))
	// Note: Version is NOT incremented here - it will be incremented after successful save
	// This ensures optimistic locking uses the correct version from database

	return nil
}

// Cancel Cancel order
// Business rule: Delivered or cancelled orders cannot be cancelled again
func (o *Order) Cancel(reason string) error {
	if o.status == StatusDelivered || o.status == StatusCancelled {
		return ErrInvalidOrderStateTransition
	}

	o.status = StatusCancelled
	o.updatedAt = time.Now()
	// Record domain event
	o.events = append(o.events, NewOrderCancelledEvent(o.id, reason))
	// Note: Version is NOT incremented here

	return nil
}

// Ship Ship order (status from CONFIRMED -> SHIPPED)
// Business rule: Only confirmed orders can be shipped
func (o *Order) Ship() error {
	if o.status != StatusConfirmed {
		return ErrInvalidOrderStateTransition
	}

	o.status = StatusShipped
	o.updatedAt = time.Now()
	// Record domain event
	o.events = append(o.events, NewOrderShippedEvent(o.id))
	// Note: Version is NOT incremented here

	return nil
}

// Deliver Deliver order (status from SHIPPED -> DELIVERED)
// Business rule: Only shipped orders can be marked as delivered
func (o *Order) Deliver() error {
	if o.status != StatusShipped {
		return ErrInvalidOrderStateTransition
	}

	o.status = StatusDelivered
	o.updatedAt = time.Now()
	// Record domain event
	o.events = append(o.events, NewOrderDeliveredEvent(o.id))
	// Note: Version is NOT incremented here

	return nil
}

// IncrementVersionForSave Increments the version after successful persistence
// This method is called by the repository after a successful save
// DDD Principle: Version management is controlled by the aggregate, triggered by persistence
func (o *Order) IncrementVersionForSave() {
	o.version++
	o.updatedAt = time.Now()
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
// Dirty Tracking - For Repository Layer Use Only
// ============================================================================
//
// DDD Principle: Aggregate tracks its own changes for efficient persistence
// Repository uses these methods to determine what needs to be inserted/deleted
// ⚠️ Note: These methods should only be used in repository implementation

// IsNew Returns true if this aggregate was newly created (not loaded from DB)
// Repository uses this to decide between INSERT ALL vs UPDATE with dirty tracking
func (o *Order) IsNew() bool { return o.isNew }

// AddedItems Returns items added since the aggregate was loaded
// Repository should INSERT these items
func (o *Order) AddedItems() []OrderItem {
	items := make([]OrderItem, len(o.addedItems))
	copy(items, o.addedItems)
	return items
}

// RemovedItems Returns items removed since the aggregate was loaded
// Repository should DELETE these items
func (o *Order) RemovedItems() []OrderItem {
	items := make([]OrderItem, len(o.removedItems))
	copy(items, o.removedItems)
	return items
}

// ClearDirtyTracking Clears all dirty tracking state after successful save
// Repository should call this after persisting changes successfully
func (o *Order) ClearDirtyTracking() {
	o.addedItems = nil
	o.removedItems = nil
	o.isNew = false
}

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

func (item OrderItem) ID() string              { return item.id }
func (item OrderItem) ProductID() string       { return item.productID }
func (item OrderItem) ProductName() string     { return item.productName }
func (item OrderItem) Quantity() int           { return item.quantity }
func (item OrderItem) UnitPrice() shared.Money { return item.unitPrice }
func (item OrderItem) Subtotal() shared.Money  { return item.subtotal }

// Compile-time check that Order implements AggregateRoot interface
var _ shared.AggregateRoot = (*Order)(nil)
