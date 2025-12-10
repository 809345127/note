# DDD Core Concepts

This document explains DDD theory and principles. For implementation details, see [DDD_GUIDE.md](DDD_GUIDE.md).

## What is DDD

Domain-Driven Design is a software development approach that:

> **Integrates business knowledge into software design to create models that accurately express business concepts.**

**Benefits:**
- Code directly reflects business concepts
- High cohesion, low coupling
- Centralized business logic, easy to maintain
- Domain logic can be tested independently

## Core Concepts

### Entity

Objects with **unique identity**. Even if all attributes are the same, different IDs mean different objects.

**Characteristics:**
- Has unique business identifier
- State changes over time
- Equality determined by identity, not attributes
- Contains business behavior methods

### Value Object

Objects described by their **attributes**, not identity. Two value objects with the same attributes are equal.

**Characteristics:**
- No unique identifier
- **Immutable** - operations return new instances
- Equality by value comparison
- Self-validating at creation

**Examples:** Email, Money, Address, DateRange

### Aggregate Root

The **entry point** of a consistency boundary for a group of related objects.

**Key Rules:**
1. External objects can only reference aggregate root, not internal entities
2. All modifications to internal entities must go through aggregate root
3. One transaction = one aggregate root
4. Aggregates reference each other by ID only

```
┌─────────────────────────────────┐
│  Order (Aggregate Root)         │
│  ┌───────────┐ ┌───────────┐   │
│  │ OrderItem │ │ OrderItem │   │  ← Internal entities
│  └───────────┘ └───────────┘   │    (only accessible via Order)
└─────────────────────────────────┘
```

### Domain Service

Handles business logic that **doesn't belong to any single entity**, typically involving multiple aggregates.

**Characteristics:**
- Stateless
- **Read-only** - validates and calculates, never calls Save()
- Contains complex cross-entity business rules

**When to use:**
- Logic involves multiple aggregates
- Logic doesn't naturally belong to any entity
- Complex validation spanning entities

### Domain Event

Records **important events** that occurred in the domain.

**Characteristics:**
- Represents something that **happened** (past tense naming)
- Immutable
- Contains event-related data
- Used for decoupling modules

**Examples:** UserCreated, OrderPlaced, PaymentReceived

### Repository

Provides **persistence abstraction** for aggregate roots.

**Key Rules:**
- Only persists aggregate roots (not internal entities)
- Provides domain-semantic interfaces
- Does NOT expose storage details (SQL, etc.)
- Does NOT publish events (that's UoW's job)

### Unit of Work

Manages **transaction boundaries** and coordinates persistence of multiple operations.

**Responsibilities:**
- Begin/commit/rollback transactions
- Collect domain events from aggregates
- Save events to outbox table (Outbox Pattern)

## Layered Architecture

```
┌─────────────────────────────────────────┐
│  Presentation Layer (api/)              │
│  HTTP handlers, middleware, responses   │
└─────────────────┬───────────────────────┘
                  │ depends on
┌─────────────────▼───────────────────────┐
│  Application Layer (application/)       │
│  Orchestrates business processes        │
│  Transaction management (via UoW)       │
└─────────────────┬───────────────────────┘
                  │ depends on
┌─────────────────▼───────────────────────┐
│  Domain Layer (domain/)    ◄── CORE     │
│  Entities, Value Objects, Domain Services│
│  NO external dependencies               │
└─────────────────┬───────────────────────┘
                  │ dependency inversion
┌─────────────────▼───────────────────────┐
│  Infrastructure Layer (infrastructure/) │
│  Repository implementations             │
│  Database, message queue, external APIs │
└─────────────────────────────────────────┘
```

**Core Principle:** Domain layer is pure - no framework dependencies.

## Application Service vs Domain Service

| Aspect | Application Service | Domain Service |
|--------|---------------------|----------------|
| **Responsibility** | Orchestrate business flow | Complex business rules |
| **Persistence** | Calls Save() | Never calls Save() |
| **Events** | Manages via UoW | Does not handle |
| **Dependencies** | Repo, DomainService, UoW | Repo (read-only) |
| **Called by** | Controller | Application Service |

**Decision Tree:**
```
Business logic to implement
    │
    ├─► Single entity operation → Entity method
    │
    ├─► Cross-entity validation/calculation → Domain Service
    │
    └─► Orchestration + persistence → Application Service
```

**Key Rule:** Domain Service is a "consultant" (advises), Application Service is a "manager" (decides and executes).

## Anemic Model vs DDD

### Anemic Model (Anti-pattern)

```go
// Entity only has data
type User struct {
    ID       string
    Name     string
    IsActive bool
}

// All logic in service layer
func (s *UserService) DeactivateUser(id string) error {
    user, _ := s.repo.FindByID(id)
    user.IsActive = false  // Direct field manipulation
    return s.repo.Save(user)
}
```

**Problems:**
- Low cohesion: business logic scattered
- Duplicate validation in multiple places
- Hard to test business rules in isolation

### DDD Rich Model

```go
// Entity contains behavior
type User struct {
    id       string  // private fields
    name     string
    isActive bool
}

func (u *User) Deactivate() {
    u.isActive = false
    u.updatedAt = time.Now()
    // Could also record domain event here
}
```

**Benefits:**
- High cohesion: logic with data
- Single source of truth for business rules
- Easy to unit test

## Best Practices

### 1. Keep Domain Model Pure

```go
// BAD: Domain depends on framework
type User struct {
    gin.Context  // NO!
}

// GOOD: Pure domain model
type User struct {
    id   string
    name string
}
```

### 2. Use Value Objects for Concepts

```go
// BAD: Primitive obsession
type Order struct {
    totalAmount int64
    currency    string
}

// GOOD: Value object with meaning
type Order struct {
    totalAmount Money  // Encapsulates currency logic
}
```

### 3. Protect Aggregate Boundaries

```go
// BAD: Exposing internals
order.Items[0].Quantity = 100  // Bypasses business rules

// GOOD: Through aggregate root
order.UpdateItemQuantity(itemID, 100)  // Validates invariants
```

### 4. Encapsulate State

```go
// BAD: Public fields
type User struct {
    ID   string
    Name string
}

// GOOD: Private fields + accessors
type User struct {
    id   string
    name string
}
func (u *User) ID() string { return u.id }
```

## Common Pitfalls

### 1. Over-Engineering

Don't use DDD for simple CRUD. A config table doesn't need aggregates and domain events.

### 2. Anemic Domain Model

If your entities only have getters/setters and all logic is in services, you're not doing DDD.

### 3. Domain Layer Dependencies

Domain layer should never import:
- Database drivers (`database/sql`)
- Web frameworks (`gin`, `echo`)
- Infrastructure packages

### 4. Ignoring Aggregate Boundaries

If you're directly modifying internal entities from outside, your aggregates aren't protecting consistency.

### 5. Domain Service Doing Persistence

If your domain service calls `repo.Save()`, it's actually an application service.

## Learning Resources

**Books:**
1. "Domain-Driven Design" - Eric Evans (The Blue Book)
2. "Implementing Domain-Driven Design" - Vaughn Vernon (The Red Book)
3. "Domain-Driven Design Distilled" - Vaughn Vernon (Quick intro)

**Online:**
- [DDD Community](https://dddcommunity.org/)
- [Martin Fowler's DDD Articles](https://martinfowler.com/tags/domain%20driven%20design.html)
