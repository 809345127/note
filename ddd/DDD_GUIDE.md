# DDD Implementation Guide

This document covers implementation details with code examples. For theory, see [DDD_CONCEPTS.md](DDD_CONCEPTS.md).

## Project Architecture

```
ddd/
├── domain/                 # Core (no dependencies)
│   ├── shared/             # AggregateRoot, DomainEvent, UnitOfWork interfaces
│   ├── user/               # User aggregate
│   └── order/              # Order aggregate
├── application/            # Orchestration (depends on domain)
│   ├── user/
│   └── order/
├── api/                    # HTTP (depends on application)
│   ├── router.go
│   ├── user/
│   ├── order/
│   └── middleware/
└── infrastructure/         # Implementations (implements domain interfaces)
    └── persistence/
        └── mysql/
```

## Aggregate Root Implementation

### Structure

```go
// domain/order/order.go
type Order struct {
    id          string
    userID      string
    items       []OrderItem      // Internal entities (private)
    totalAmount Money
    status      OrderStatus
    version     int              // Optimistic locking
    events      []DomainEvent    // Pending events
    createdAt   time.Time
    updatedAt   time.Time
}

// Internal entity (only accessible through Order)
type OrderItem struct {
    id          string
    productID   string
    productName string
    quantity    int
    unitPrice   Money
    subtotal    Money
}
```

### Factory Method

```go
func NewOrder(userID string, items []OrderItemRequest) (*Order, error) {
    if len(items) == 0 {
        return nil, errors.New("order must have at least one item")
    }

    order := &Order{
        id:        uuid.Must(uuid.NewV7()).String(),
        userID:    userID,
        status:    OrderStatusPending,
        events:    make([]DomainEvent, 0),
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }

    for _, req := range items {
        if err := order.AddItem(req); err != nil {
            return nil, err
        }
    }

    // Record domain event
    order.events = append(order.events, NewOrderPlacedEvent(order.id, userID, order.totalAmount))

    return order, nil
}
```

### Boundary Protection

```go
// All modifications through aggregate root
func (o *Order) AddItem(req OrderItemRequest) error {
    if o.status != OrderStatusPending {
        return errors.New("can only add items to pending orders")
    }

    item := OrderItem{
        id:          uuid.Must(uuid.NewV7()).String(),
        productID:   req.ProductID,
        productName: req.ProductName,
        quantity:    req.Quantity,
        unitPrice:   req.UnitPrice,
        subtotal:    *NewMoney(req.UnitPrice.Amount()*int64(req.Quantity), req.UnitPrice.Currency()),
    }

    o.items = append(o.items, item)
    o.recalculateTotalAmount()
    o.updatedAt = time.Now()
    return nil
}

// Return copy to prevent external modification
func (o *Order) Items() []OrderItem {
    items := make([]OrderItem, len(o.items))
    copy(items, o.items)
    return items
}
```

### State Transitions

```go
func (o *Order) Confirm() error {
    if o.status != OrderStatusPending {
        return errors.New("only pending orders can be confirmed")
    }
    o.status = OrderStatusConfirmed
    o.updatedAt = time.Now()
    o.version++
    o.events = append(o.events, NewOrderConfirmedEvent(o.id))
    return nil
}

func (o *Order) Ship() error {
    if o.status != OrderStatusConfirmed {
        return errors.New("only confirmed orders can be shipped")
    }
    o.status = OrderStatusShipped
    o.updatedAt = time.Now()
    o.version++
    return nil
}
```

### Event Collection

```go
// Called by UoW before commit
func (o *Order) PullEvents() []DomainEvent {
    events := o.events
    o.events = nil
    return events
}
```

## Value Object Implementation

### Money

```go
// domain/shared/money.go
type Money struct {
    amount   int64   // In cents
    currency string
}

func NewMoney(amount int64, currency string) *Money {
    return &Money{amount: amount, currency: currency}
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return m.currency }

// Immutable: returns new instance
func (m Money) Add(other Money) (*Money, error) {
    if m.currency != other.currency {
        return nil, errors.New("currency mismatch")
    }
    return NewMoney(m.amount+other.amount, m.currency), nil
}

func (m Money) Multiply(factor int) *Money {
    return NewMoney(m.amount*int64(factor), m.currency)
}
```

### Email

```go
// domain/shared/email.go
type Email struct {
    value string
}

func NewEmail(value string) (*Email, error) {
    if !isValidEmail(value) {
        return nil, errors.New("invalid email format")
    }
    return &Email{value: value}, nil
}

func (e Email) Value() string { return e.value }

func (e Email) Equals(other Email) bool {
    return e.value == other.value
}
```

## Repository Implementation

### Interface (Domain Layer)

```go
// domain/order/repository.go
type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
    FindByUserID(ctx context.Context, userID string) ([]*Order, error)
    Remove(ctx context.Context, id string) error
}
```

### Implementation (Infrastructure Layer)

```go
// infrastructure/persistence/mysql/order_repository.go
type OrderRepository struct {
    db *gorm.DB
}

func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
    // Check for transaction in context
    db := r.db
    if tx := persistence.TxFromContext(ctx); tx != nil {
        db = tx
    }

    po := toOrderPO(order)
    return db.Save(po).Error
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
    var po OrderPO
    if err := r.db.Preload("Items").First(&po, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return toDomainOrder(&po), nil
}

// Logical deletion
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
    return r.db.Model(&OrderPO{}).Where("id = ?", id).
        Update("status", domain.OrderStatusCancelled).Error
}
```

## Unit of Work Implementation

### Interface (Domain Layer)

```go
// domain/shared/unit_of_work.go
type UnitOfWork interface {
    Execute(ctx context.Context, fn func(ctx context.Context) error) error
    RegisterNew(aggregate AggregateRoot)
    RegisterDirty(aggregate AggregateRoot)
}
```

### Implementation with Outbox Pattern

```go
// infrastructure/persistence/mysql/unit_of_work.go
type UnitOfWork struct {
    db         *gorm.DB
    aggregates []domain.AggregateRoot
}

func (uow *UnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
    uow.aggregates = nil

    return uow.db.Transaction(func(tx *gorm.DB) error {
        // Put transaction in context
        ctx = persistence.ContextWithTx(ctx, tx)

        // Execute business logic
        if err := fn(ctx); err != nil {
            return err
        }

        // Collect and save events to outbox
        for _, agg := range uow.aggregates {
            events := agg.PullEvents()
            for _, event := range events {
                payload, _ := json.Marshal(event)
                outbox := OutboxPO{
                    EventType:   event.EventName(),
                    AggregateID: agg.ID(),
                    Payload:     payload,
                    CreatedAt:   time.Now(),
                }
                if err := tx.Create(&outbox).Error; err != nil {
                    return err
                }
            }
        }

        return nil
    })
}

func (uow *UnitOfWork) RegisterNew(agg domain.AggregateRoot) {
    uow.aggregates = append(uow.aggregates, agg)
}
```

## Domain Event Flow

```
Aggregate state change → Record event in memory
            ↓
      UoW.Execute()
            ↓
┌───────────────────────────┐
│ BEGIN TRANSACTION         │
│   repo.Save(aggregate)    │
│   outbox.Save(events)     │
│ COMMIT                    │
└───────────────────────────┘
            ↓
   Background process polls outbox
            ↓
   Publish to message queue
```

### Why Not Publish in Repository?

```go
// BAD: Event sent before transaction commits
func (r *OrderRepository) Save(ctx context.Context, order *Order) error {
    err := r.db.Save(order).Error
    if err != nil {
        return err
    }
    r.publisher.Publish(order.PullEvents())  // Transaction might rollback!
    return nil
}
```

If subsequent operations fail, the transaction rolls back but events are already sent.

### Outbox Table

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

### Message Relay (Background Process)

```go
// infrastructure/outbox_processor.go
func (p *OutboxProcessor) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.processOutbox()
        }
    }
}

func (p *OutboxProcessor) processOutbox() {
    rows, _ := p.db.Query(
        "SELECT id, event_type, payload FROM outbox WHERE published_at IS NULL LIMIT 100",
    )
    defer rows.Close()

    for rows.Next() {
        var id int64
        var eventType string
        var payload []byte
        rows.Scan(&id, &eventType, &payload)

        if err := p.mq.Publish(eventType, payload); err != nil {
            continue
        }

        p.db.Exec("UPDATE outbox SET published_at = NOW() WHERE id = ?", id)
    }
}
```

## Application Service Pattern

```go
// application/order/service.go
type OrderApplicationService struct {
    orderRepo          domain.OrderRepository
    userRepo           domain.UserRepository
    orderDomainService *domain.OrderDomainService
    uow                domain.UnitOfWork
}

func (s *OrderApplicationService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
    var order *domain.Order

    err := s.uow.Execute(ctx, func(ctx context.Context) error {
        // 1. Validate via domain service (read-only)
        if err := s.orderDomainService.ValidateCanPlaceOrder(ctx, req.UserID); err != nil {
            return err
        }

        // 2. Create aggregate (records events internally)
        var err error
        order, err = domain.NewOrder(req.UserID, req.Items)
        if err != nil {
            return err
        }

        // 3. Save aggregate
        if err := s.orderRepo.Save(ctx, order); err != nil {
            return err
        }

        // 4. Register for event collection
        s.uow.RegisterNew(order)
        return nil
    })

    if err != nil {
        return nil, err
    }

    return s.toResponse(order), nil
}
```

## Domain Service Pattern

```go
// domain/order/service.go
type OrderDomainService struct {
    userRepo  UserRepository
    orderRepo OrderRepository
}

// Read-only: validates but never calls Save()
func (s *OrderDomainService) ValidateCanPlaceOrder(ctx context.Context, userID string) error {
    user, err := s.userRepo.FindByID(ctx, userID)
    if err != nil {
        return err
    }

    if !user.IsActive() {
        return errors.New("user is not active")
    }

    if !user.CanMakePurchase() {
        return errors.New("user cannot make purchases")
    }

    // Check pending orders limit
    orders, err := s.orderRepo.FindPendingByUserID(ctx, userID)
    if err != nil {
        return err
    }

    if len(orders) >= 5 {
        return errors.New("too many pending orders")
    }

    return nil
}

// Complex calculation across aggregates
func (s *OrderDomainService) CalculateUserTotalSpent(ctx context.Context, userID string) (Money, error) {
    orders, err := s.orderRepo.FindDeliveredByUserID(ctx, userID)
    if err != nil {
        return Money{}, err
    }

    total := NewMoney(0, "CNY")
    for _, order := range orders {
        total, _ = total.Add(order.TotalAmount())
    }
    return *total, nil
}
```

## Summary

| Component | Location | Responsibility |
|-----------|----------|----------------|
| Aggregate Root | domain/ | Business rules, event generation |
| Value Object | domain/shared/ | Immutable concepts |
| Repository Interface | domain/ | Persistence contract |
| Repository Impl | infrastructure/ | Database operations |
| Domain Service | domain/ | Cross-entity validation (read-only) |
| Application Service | application/ | Orchestration, Save(), UoW |
| UnitOfWork | infrastructure/ | Transaction + event collection |
| Message Relay | infrastructure/ | Outbox polling, event publishing |

**Key Rules:**
1. Domain layer has no external dependencies
2. Aggregates generate events, UoW saves them
3. Domain services never call Save()
4. All persistence through UoW transaction
