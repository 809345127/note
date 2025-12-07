# DDD Example Project - Go Implementation

This is a Domain-Driven Design (DDD) example project implemented in Go, designed for developers familiar with the Anemic Model pattern. It demonstrates DDD core concepts and practical implementation through concrete examples.

## 📚 Documentation Navigation

| Document | Description |
|----------|-------------|
| **This file (README.md)** | Project overview, quick start, API examples |
| [DDD_CONCEPTS.md](DDD_CONCEPTS.md) | DDD core concepts explained in detail, Anemic Model comparison, best practices, common pitfalls |
| [DDD_GUIDE.md](DDD_GUIDE.md) | Practical guide: Aggregate Roots, Domain Events, Unit of Work, Repository patterns |

## 🎯 Project Objectives

- **Demonstrate DDD Core Concepts**: Entities, Value Objects, Aggregate Roots, Domain Services, Domain Events, Repositories, Unit of Work
- **Compare Anemic Model vs DDD**: Help developers understand architectural transformation (see [DDD_CONCEPTS.md](DDD_CONCEPTS.md#from-anemic-model-to-ddd))
- **Provide Complete Examples**: Complete business processes including user management and order management
- **Working Code**: Supports both Mock and MySQL storage options

## 🏗️ Project Architecture

The project strictly follows DDD layered architecture (detailed architecture diagram in [DDD_CONCEPTS.md](DDD_CONCEPTS.md#layered-architecture)):

```
ddd/
├── api/                        # Presentation Layer
│   ├── router.go               # Route entry, aggregates middleware and controllers
│   ├── health/
│   │   └── controller.go       # Health check controller
│   ├── user/
│   │   └── controller.go       # User controller
│   ├── order/
│   │   └── controller.go       # Order controller
│   ├── middleware/
│   │   └── middleware.go       # Request ID, Recovery, Logging, CORS, Rate Limiting
│   └── response/
│       └── response.go         # Unified response and pagination structure
├── application/                # Application Layer
│   ├── user/
│   │   └── service.go
│   └── order/
│       └── service.go
├── domain/                     # Domain Layer - Core Layer
│   ├── shared/                 # Common Value Objects, Error Definitions
│   ├── user/                   # User Aggregate
│   └── order/                  # Order Aggregate
├── infrastructure/             # Infrastructure Layer
│   └── persistence/
│       ├── mocks/              # Mock Implementation
│       └── mysql/              # MySQL Implementation
├── pkg/                        # Common Components (logger, errors, etc.)
├── util/                       # Utility Functions
├── cmd/                        # Application Entry Point
│   └── app.go
├── main.go
├── go.mod
├── README.md                   # This file
├── DDD_CONCEPTS.md             # DDD Concepts Explained
└── DDD_GUIDE.md                # DDD Practical Guide
```

## 🧩 DDD Core Concepts

> 📖 For detailed concept explanations, please refer to [DDD_CONCEPTS.md](DDD_CONCEPTS.md); for implementation details, see [DDD_GUIDE.md](DDD_GUIDE.md)

| Concept | Description | Implementation |
|---------|-------------|----------------|
| **Aggregate Root** | The entry point of a consistency boundary for a group of related objects | `User`, `Order` |
| **Entity** | Objects with unique identities, containing business logic | `User`, `Order` |
| **Value Object** | Immutable objects identified by their attributes | `Email`, `Money`, `OrderItem` |
| **Domain Service** | Business rules spanning multiple entities | `UserDomainService`, `OrderDomainService` |
| **Domain Event** | Recording important events in the business system | `UserCreatedEvent`, `OrderPlacedEvent` |
| **Repository** | Persistence abstraction for aggregate roots | `UserRepository`, `OrderRepository` |
| **Unit of Work** | Transaction boundary management | `UnitOfWork` |

## 🚀 Quick Start

### Environment Requirements

- Go 1.21 or higher
- Git

### Install Dependencies

```bash
cd /home/shize/note/ddd
go mod download
```

### Run Project

```bash
go run main.go
```

Or specify a port:

```bash
go run main.go -port 8080
```

### Test API

After the project starts, you can test the API using:

#### 1. Health Check
```bash
curl http://localhost:8080/api/v1/health
```

#### 2. Create User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Zhang San",
    "email": "zhangsan@example.com",
    "age": 25
  }'
```

#### 3. Get User List
```bash
curl http://localhost:8080/api/v1/users
```

#### 4. Get Specific User
```bash
curl http://localhost:8080/api/v1/users/user-1
```

#### 5. Create Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-1",
    "items": [
      {
        "product_id": "prod-1",
        "product_name": "iPhone 15",
        "quantity": 1,
        "unit_price": 699900,
        "currency": "CNY"
      }
    ]
  }'
```

#### 6. Get User Orders
```bash
curl http://localhost:8080/api/v1/orders/user/user-1
```

## 📖 Core Business Process Flows

### User Creation Process

1. **API Layer** receives user creation request
2. **Application Service** validates email uniqueness
3. **Domain Layer** creates user entity (includes business rule validation)
4. **Repository Layer** saves user data
5. **Domain Event** publishes user creation event
6. **API Layer** returns creation result

### Order Creation Process

1. **API Layer** receives order creation request
2. **Application Service** checks if user can place order (via domain service)
3. **Domain Service** validates user status, age, pending orders count, etc.
4. **Domain Layer** creates order entity
5. **Repository Layer** saves order data
6. **Domain Event** publishes order creation event
7. **API Layer** returns creation result

## 🔍 From Anemic Model to DDD

> 📖 For detailed comparison and code examples, see [DDD_CONCEPTS.md - From Anemic Model to DDD](DDD_CONCEPTS.md#from-anemic-model-to-ddd)

| Anemic Model | DDD Pattern |
|--------------|-------------|
| Entities only contain data | Entities contain business logic |
| Business logic scattered in service layer | Business logic encapsulated in domain layer |
| Low cohesion, hard to maintain | High cohesion, easy to test |

## 🧪 Test Data

The project automatically creates the following test data on startup:

### Test Users
- **user-1**: Zhang San (zhangsan@example.com, 25 years old)
- **user-2**: Li Si (lisi@example.com, 30 years old)
- **user-3**: Wang Wu (wangwu@example.com, 35 years old)

### Test Orders
- **order-1**: User-1's order, includes iPhone 15 and MacBook Pro (shipped)
- **order-2**: User-2's order, includes 2 AirPods Pro (confirmed)
- **order-3**: User-1's order, includes iPhone 15 (pending)

## 🔧 Extension Directions

- **CQRS Pattern**: Separate command and query responsibilities (see [DDD_GUIDE.md - Advanced Topics](DDD_GUIDE.md#advanced-topics))
- **Event Sourcing**: Reconstruct aggregate state through event sequences
- **More Bounded Contexts**: Product management, inventory management, payment management

## 📚 Learning Resources

> 📖 For more learning resources, see [DDD_CONCEPTS.md - Learning Resources](DDD_CONCEPTS.md#learning-resources)

**Recommended Books**:
- "Domain-Driven Design" - Eric Evans
- "Implementing Domain-Driven Design" - Vaughn Vernon
