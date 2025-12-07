# DDD Practice Guide

This document is a practical guide for the DDD example project, explaining the correct implementation of each DDD pattern in the project in detail.

---

## Project Architecture

### Standard DDD Layered Architecture

```
ddd/
├── domain/                 # Domain Layer (Core layer, no dependencies on other layers)
│   ├── user.go             # User Aggregate Root
│   ├── order.go            # Order Aggregate Root
│   ├── value_objects.go    # Value Objects
│   ├── services.go         # Domain Services
│   ├── events.go           # Domain Events
│   ├── repositories.go     # Repository Interfaces
│   ├── aggregate.go        # Aggregate Marker Interface
│   ├── event_publisher.go  # Event Publisher Interface
│   ├── unit_of_work.go     # Unit of Work Interface
│   └── tx_unit_of_work.go  # Transactional Unit of Work
├── application/            # Application Layer (depends on domain layer)
├── api/                    # Presentation Layer (depends on application layer)
│   ├── router.go
│   ├── health/             # Health Check Controller
│   ├── user/               # User Controller
│   ├── order/              # Order Controller
│   ├── middleware/         # Request ID, Logging, Recovery, CORS, Rate Limiting
│   └── response/           # Unified Response Wrapper
├── infrastructure/         # Infrastructure Layer (implements interfaces defined in domain layer)
│   └── persistence/
│       ├── mocks/          # Mock Repository Implementations
│       └── mysql/          # MySQL Repository Implementations
└── cmd/                    # Application Entry Point
```

### Dependency Direction

```
┌─────────────────────────────────┐
│   Presentation Layer (api)      │
│   Handles HTTP requests/responses │
└────────────┬────────────────────┘
             │ Depends on
┌────────────▼────────────────────┐
│   Application Layer (service)   │
│   Orchestrates business processes, uses UnitOfWork │
└────────────┬────────────────────┘
             │ Depends on
┌────────────▼────────────────────┐
│   Domain Layer (domain)         │  ◄─ Core layer (pure, no dependencies)
│   Business logic, entities, aggregate roots │
└────────────┬────────────────────┘
             │ Dependency Inversion (via interfaces)
┌────────────▼────────────────────┐
│   Infrastructure Layer (infrastructure) │
│   Technical implementation (repositories, event publishing) │
└─────────────────────────────────┘
```

**Core Principle**: The domain layer is the core of the project, it doesn't depend on any frameworks or technical implementations. The infrastructure layer provides concrete implementations by implementing interfaces defined in the domain layer.

---

## Aggregate Root

Aggregate root is a core concept in DDD, it defines the consistency boundary for a group of related objects.

### Order Aggregate Root Example

```go
// domain/order.go
type Order struct {
    id          string
    userID      string
    items       []OrderItem           // Aggregate internal entity (private)
    totalAmount Money
    status      OrderStatus
    version     int                   // Optimistic locking version number
    createdAt   time.Time
    updatedAt   time.Time
    events      []DomainEvent         // Domain event list
}

// Aggregate internal entity (non-aggregate root, can only be accessed through Order)
type OrderItem struct {
    id          string  // Only unique within the aggregate
    productID   string
    productName string
    quantity    int
    unitPrice   Money
    subtotal    Money
}
```

### Aggregate Boundary Protection

All modifications to aggregate internal entities must go through the aggregate root:

```go
// Add order item through aggregate root method
func (o *Order) AddItem(productID, productName string, quantity int, unitPrice Money) error {
    // 1. Verify aggregate invariants
    if o.status != OrderStatusPending {
        return errors.New("can only add items to pending orders")
    }

    // 2. Create aggregate internal entity
    item := OrderItem{
        id:          uuid.New().String(),
        productID:   productID,
        productName: productName,
        quantity:    quantity,
        unitPrice:   unitPrice,
        subtotal:    *NewMoney(unitPrice.Amount() * int64(quantity), unitPrice.Currency()),
    }

    o.items = append(o.items, item)

    // 3. Maintain aggregate consistency
    o.recalculateTotalAmount()
    o.updatedAt = time.Now()

    return nil
}

// Items() returns copy to prevent external direct modification
func (o *Order) Items() []OrderItem {
    items := make([]OrderItem, len(o.items))
    copy(items, o.items)
    return items
}
```

### Aggregate Marker Interface

```go
// domain/aggregate.go
type AggregateRoot interface {
    ID() string
    Version() int
    PullEvents() []DomainEvent  // Get and clear events
}

// Compile-time validation
var _ = IsAggregateRoot(&User{})
var _ = IsAggregateRoot(&Order{})
```

---

## Domain Event

Domain events record important events that occur in the business system, used for decoupling and asynchronous processing.

### Core Principles

1. **Aggregate roots generate events**: Record events to memory when state changes
2. **UoW saves events uniformly**: Before transaction commit, persist events with business data together (Outbox Pattern)
3. **Publish after transaction commit**: Background processes asynchronously read outbox table and publish to message queue

```
┌─────────────────────────────────────────────────────────┐
│  UnitOfWork.Execute()                                   │
│  ┌───────────────────────────────────────────────────┐  │
│  │ BEGIN TRANSACTION                                 │  │
│  │   repo.Save(order)    → Save aggregate root data  │  │
│  │   outbox.Save(events) → Save events to outbox table│ │
│  │ COMMIT                                            │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  Background process polls outbox table → Publish to message queue → Delete sent records│
└─────────────────────────────────────────────────────────┘
```

### Aggregate Root Generates Events

```go
// domain/order.go
func NewOrder(userID string, requests []OrderItemRequest) (*Order, error) {
    order := &Order{
        id:        uuid.New().String(),
        userID:    userID,
        status:    OrderStatusPending,
        events:    make([]DomainEvent, 0),
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }

    // ... Add order items

    // Record domain event
    order.events = append(order.events, NewOrderPlacedEvent(order.id, userID, order.totalAmount))

    return order, nil
}

// Record events when state changes
func (o *Order) Confirm() error {
    if o.status != OrderStatusPending {
        return errors.New("only pending orders can be confirmed")
    }
    o.status = OrderStatusConfirmed
    o.updatedAt = time.Now()
    o.version++

    // Record event
    o.events = append(o.events, NewOrderConfirmedEvent(o.id))
    return nil
}

// Get and clear events (for repository use only)
func (o *Order) PullEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}
```

### Why Not Publish Events Directly in Repository?

**Incorrect Example (Don't do this):**

```go
func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
    _, err := r.db.ExecContext(ctx, `INSERT INTO users ...`)
    if err != nil {
        return err
    }
    // ❌ Error: Transaction may not be committed yet, or subsequent operations may fail causing rollback
    // But the event has already been sent, causing data inconsistency!
    r.eventPublisher.Publish(user.PullEvents())
    return nil
}
```

**Problem:** Transactions are managed at the application service layer (through UoW), and repositories are just one part of the transaction. If events are published directly in repo.Save(), the following issues may occur:
1. Events are sent out before the transaction is committed
2. Subsequent operations fail causing transaction rollback, but events have already been consumed

**Correct Approach:** Let UoW uniformly save events to outbox table before transaction commit, and have background processes publish asynchronously after transaction commit.

---

## Unit of Work

The Unit of Work pattern is used to manage transaction boundaries, ensure consistency of multiple aggregate operations, and uniformly handle persistence of domain events.

### Interface Definition

```go
// domain/unit_of_work.go
type UnitOfWork interface {
    Execute(fn func() error) error  // Automatically manage transactions
    RegisterNew(aggregate AggregateRoot)
    RegisterDirty(aggregate AggregateRoot)
    RegisterRemoved(aggregate AggregateRoot)
}
```

### UoW Implementation (with Outbox Pattern)

```go
// infrastructure/unit_of_work.go
type UnitOfWorkImpl struct {
    db         *sql.DB
    tx         *sql.Tx
    aggregates []domain.AggregateRoot
}

func (uow *UnitOfWorkImpl) Execute(fn func() error) error {
    // 1. Start transaction
    tx, err := uow.db.Begin()
    if err != nil {
        return err
    }
    uow.tx = tx
    uow.aggregates = nil

    // 2. Execute business logic
    if err := fn(); err != nil {
        tx.Rollback()
        return err
    }

    // 3. Collect all aggregate root events, save to outbox table (same transaction)
    for _, agg := range uow.aggregates {
        events := agg.PullEvents()
        for _, event := range events {
            payload, _ := json.Marshal(event)
            _, err := tx.Exec(
                "INSERT INTO outbox (event_type, aggregate_id, payload, created_at) VALUES (?, ?, ?, NOW())",
                event.EventName(), agg.ID(), payload,
            )
            if err != nil {
                tx.Rollback()
                return err
            }
        }
    }

    // 4. Commit transaction (business data + events committed together, ensuring atomicity)
    return tx.Commit()
}

func (uow *UnitOfWorkImpl) RegisterNew(agg domain.AggregateRoot) {
    uow.aggregates = append(uow.aggregates, agg)
}
```

### Application Layer Usage Example

```go
// application/order/service.go
func (s *OrderApplicationService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
    var order *domain.Order

    // Use Unit of Work to manage transaction
    err := s.uow.Execute(func() error {
        // 1. Verify if user can place order
        user, err := s.userRepo.FindByID(req.UserID)
        if err != nil {
            return err
        }

        if !user.CanMakePurchase() {
            return errors.New("user cannot make purchases")
        }

        // 2. Create order aggregate root (aggregate root records events internally)
        order, err = domain.NewOrder(req.UserID, req.Items)
        if err != nil {
            return err
        }

        // 3. Save aggregate root
        if err := s.orderRepo.Save(order); err != nil {
            return err
        }

        // 4. Register with Unit of Work (UoW will collect events before commit)
        s.uow.RegisterNew(order)

        return nil
    })

    // Execute handles automatically:
    // - Start transaction (Begin)
    // - Execute business operations
    // - Collect aggregate root events, save to outbox table
    // - Commit transaction (business data + events atomically committed)
    // - Rollback transaction on failure

    if err != nil {
        return nil, err
    }

    return s.convertToResponse(order), nil
}
```

### Outbox Table Structure

```sql
CREATE TABLE outbox (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    aggregate_id VARCHAR(36) NOT NULL,
    payload JSON NOT NULL,
    created_at TIMESTAMP NOT NULL,
    published_at TIMESTAMP NULL,
    INDEX idx_unpublished (published_at, created_at)
);
```

### Event Publishing: Message Relay (Independent Background Service)

**Important: Application Service is not responsible for Publishing Events, only for Saving Events to the outbox table.**

Actual publishing is completed by an independent background service (Message Relay). Common implementation methods:

| Method | Latency | Complexity | Description |
|--------|---------|------------|-------------|
| Polling | Seconds | Low | Simple but with latency |
| CDC (Change Data Capture) | Milliseconds | High | Recommended, like Debezium |
| JIT Polling | Controllable | Medium | Hybrid approach, balancing latency and simplicity |

#### JIT Polling Implementation (Recommended)

```go
// infrastructure/outbox_processor.go
type OutboxProcessor struct {
    db           *sql.DB
    messageQueue MessageQueue
    triggerCh    chan struct{}  // Used to receive immediate processing notifications
}

func NewOutboxProcessor(db *sql.DB, mq MessageQueue) *OutboxProcessor {
    return &OutboxProcessor{
        db:           db,
        messageQueue: mq,
        triggerCh:    make(chan struct{}, 1),  // With buffer to avoid blocking
    }
}

// NotifyNewMessages notifies the processor of new messages, can process immediately (non-blocking)
func (p *OutboxProcessor) NotifyNewMessages() {
    select {
    case p.triggerCh <- struct{}{}:
    default:  // Notification already waiting, no need to repeat
    }
}

// Run starts the background processing loop
func (p *OutboxProcessor) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)  // Periodic fallback
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-p.triggerCh:    // Immediate trigger
            p.processOutbox()
        case <-ticker.C:       // Periodic fallback (prevent notification loss)
            p.processOutbox()
        }
    }
}

func (p *OutboxProcessor) processOutbox() {
    rows, err := p.db.Query(
        "SELECT id, event_type, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at LIMIT 100",
    )
    if err != nil {
        log.Printf("Failed to query outbox: %v", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var id int64
        var eventType string
        var payload []byte
        if err := rows.Scan(&id, &eventType, &payload); err != nil {
            continue
        }

        // Publish to message queue
        if err := p.messageQueue.Publish(eventType, payload); err != nil {
            log.Printf("Failed to publish event %d: %v", id, err)
            continue
        }

        // Mark as published
        p.db.Exec("UPDATE outbox SET published_at = NOW() WHERE id = ?", id)
    }
}
```

#### Application Service with JIT Polling

```go
// application/order/service.go
type OrderApplicationService struct {
    uow             domain.UnitOfWork
    orderRepo       domain.OrderRepository
    outboxProcessor *infrastructure.OutboxProcessor  // Holds processor reference
}

func (s *OrderApplicationService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
    var order *domain.Order

    err := s.uow.Execute(func() error {
        // ... Business logic (create order, save aggregate root)
        // UoW will save events to outbox table in transaction
        return nil
    })

    if err != nil {
        return nil, err
    }

    // After transaction success, notify processor of new messages (non-blocking, optional)
    // Note: This is just a "notification", not direct publishing
    s.outboxProcessor.NotifyNewMessages()

    return s.convertToResponse(order), nil
}
```

### Responsibility Summary

```
┌─────────────────────────────────────────────────────────────────────┐
│  Role                      │  Responsibility                          │
├─────────────────────────────────────────────────────────────────────┤
│  Aggregate Root            │  Generate events, temporarily store in memory │
│  UoW                       │  Transaction management, Save Event to outbox table │
│  Application Service       │  Business orchestration, can notify processor (not direct publish) │
│  Message Relay (Independent Process) │  Read from outbox, Publish to message queue │
└─────────────────────────────────────────────────────────────────────┘
```

**Core Principle: Application Service does not directly Publish Events.**

---

## Repository

Repository provides persistence abstraction for aggregate roots.

### Interface Design Principles

```go
// domain/repositories.go
type OrderRepository interface {
    // ID Generation
    NextIdentity() string

    // Basic Operations (aggregate root level)
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
    Remove(ctx context.Context, id string) error  // Logical deletion

    // Controlled Queries (limited scope)
    FindByUserID(ctx context.Context, userID string) ([]*Order, error)
}
```

### Repository Responsibilities

**Should Do**:
- Only persist aggregate roots (save entire aggregate together)
- Provide query interfaces with domain semantics
- Ensure atomic operations of aggregates

**Should Not Do**:
- Expose underlying storage details (SQL statements, etc.)
- Allow bypassing aggregate root to modify internal entities
- Provide batch operations (like FindAll)
- Contain business logic
- Directly publish domain events (this is UoW's responsibility)

### Logical Deletion

DDD recommends using logical deletion to preserve business history:

```go
// infrastructure/persistence/mysql/order_repository.go
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
    // Logical deletion: mark as cancelled
    _, err := r.db.ExecContext(ctx, `
        UPDATE orders SET status = ?, updated_at = NOW() WHERE id = ?
    `, string(domain.OrderStatusCancelled), id)
    return err
}
```

---

## Domain Service vs Application Service

### Domain Service

Handles cross-entity business rules, **not responsible for persistence and event publishing**:

```go
// domain/services.go
type OrderDomainService struct {
    userRepository  UserRepository
    orderRepository OrderRepository
}

// Only does business rule validation, returns validation result
func (s *OrderDomainService) CanProcessOrder(ctx context.Context, orderID string) (*Order, error) {
    order, err := s.orderRepository.FindByID(ctx, orderID)
    if err != nil {
        return nil, err
    }

    user, err := s.userRepository.FindByID(ctx, order.UserID())
    if err != nil {
        return nil, err
    }

    // Cross-entity business rule validation
    if !user.IsActive() {
        return nil, ErrUserNotActive
    }
    if order.Status() != OrderStatusPending {
        return nil, errors.New("only pending orders can be processed")
    }

    return order, nil
}
```

### Application Service

Orchestrates business processes, **responsible for calling Save**:

```go
// application/order/service.go
type OrderApplicationService struct {
    orderRepo          domain.OrderRepository
    orderDomainService *domain.OrderDomainService
}

func (s *OrderApplicationService) ProcessOrder(ctx context.Context, orderID string) error {
    // 1. Validate through domain service
    order, err := s.orderDomainService.CanProcessOrder(ctx, orderID)
    if err != nil {
        return err
    }

    // 2. Modify aggregate root status
    if err := order.Confirm(); err != nil {
        return err
    }

    // 3. Persist (repository only handles persistence, events are saved to outbox table by UoW)
    return s.orderRepo.Save(ctx, order)
}
```

### Responsibility Comparison

| Feature       | Domain Service             | Application Service                       |
|--------------|---------------------------|-------------------------------------------|
| **Responsibility** | Cross-entity business rule validation | Orchestrate business processes |
| **Persistence** | Does not call Save          | Calls Save                                |
| **Event Handling** | Not responsible          | Save events to outbox table through UoW   |
| **Called By** | Application Service        | Presentation Layer (Controller)           |

---

## Value Object

Value objects are immutable and identified by their value rather than identity.

### Money Value Object

```go
// domain/value_objects.go
type Money struct {
    amount   int64   // in cents
    currency string
}

func NewMoney(amount int64, currency string) *Money {
    return &Money{amount: amount, currency: currency}
}

func (m Money) Amount() int64     { return m.amount }
func (m Money) Currency() string  { return m.currency }

// Value objects are immutable, operations return new instances
func (m Money) Add(other Money) (*Money, error) {
    if m.currency != other.currency {
        return nil, errors.New("currency mismatch")
    }
    return NewMoney(m.amount + other.amount, m.currency), nil
}
```

### Email Value Object

```go
type Email struct {
    value string
}

func NewEmail(value string) (*Email, error) {
    // Validate email format
    if !isValidEmail(value) {
        return nil, errors.New("invalid email format")
    }
    return &Email{value: value}, nil
}

func (e Email) Value() string { return e.value }
```

---

## Best Practices Summary

### Layer Responsibilities

| Layer | Responsibility | Dependencies |
|-------|----------------|--------------|
| Presentation Layer | HTTP request handling, parameter validation | Application Layer |
| Application Layer | Business process orchestration, calls Save | Domain Layer |
| Domain Layer | Business logic, aggregate roots, domain services | No dependencies |
| Infrastructure Layer | Technical implementation, repositories, event publishing | Domain Layer interfaces |

### Event Flow (Outbox Pattern)

```
Aggregate root state change → Record event to memory
                    ↓
              UoW.Execute()
                    ↓
    ┌───────────────────────────────┐
    │ BEGIN TRANSACTION             │
    │   repo.Save(aggregate root)   │
    │   outbox.Save(event)          │
    │ COMMIT                        │
    └───────────────────────────────┘
                    ↓
        Background process polls outbox table
                    ↓
        Publish to message queue → Event handler
```

**Key Points:**
- Aggregate roots generate events, UoW saves uniformly
- Events and business data are persisted in the same transaction, ensuring atomicity
- Background processes publish asynchronously, ensuring eventual consistency

### Code Quality Check

1. **Domain classes don't depend on frameworks**
2. **Cannot bypass aggregate root to modify internal entities**
3. **Business rules are inside entities**
4. **Repository interfaces are concise**

---

## Advanced Topics

### CQRS Pattern

Separate command (write) and query (read) models:

```go
// Command model: through aggregate root
orderRepo.Save(order)

// Query model: dedicated query service
type OrderQueryService interface {
    SearchOrders(criteria OrderSearchCriteria) ([]OrderDTO, error)
}
```

### Event Sourcing

Reconstruct aggregate state through event sequence:

```go
type EventStore interface {
    SaveEvents(aggregateID string, events []DomainEvent, version int) error
    LoadEvents(aggregateID string) ([]DomainEvent, error)
}
```

---

## References

### Recommended Books

1. **"Domain-Driven Design"** - Eric Evans (Foundational DDD work)
2. **"Implementing Domain-Driven Design"** - Vaughn Vernon (Practical guide, highly recommended)
3. **"Domain-Driven Design Distilled"** - Vaughn Vernon (Quick start guide)

### Design Patterns

- Aggregate Pattern - Maintains consistency boundaries
- Repository Pattern - Persistence abstraction
- Unit of Work Pattern - Transaction management + Event collection
- Domain Event Pattern - Decoupling and asynchronous processing
- **Outbox Pattern** - Events and data atomically persisted, background asynchronous publishing
