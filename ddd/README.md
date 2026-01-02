# DDD Example Project - Go Implementation

A Domain-Driven Design example project in Go, demonstrating DDD patterns through user and order management.

## Documentation

| Document | Description |
|----------|-------------|
| **This file** | Quick start and API examples |
| [DDD_CONCEPTS.md](DDD_CONCEPTS.md) | DDD theory: concepts, principles, best practices |
| [DDD_GUIDE.md](DDD_GUIDE.md) | Implementation guide: code examples, patterns |

## Quick Start

### Requirements

- Go 1.24+

### Run

```bash
go mod download
go run main.go
# Or specify port: go run main.go -port 8080
```

## API Examples

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### Create User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Zhang San", "email": "zhangsan@example.com", "age": 25}'
```

### Get Users
```bash
curl http://localhost:8080/api/v1/users
curl http://localhost:8080/api/v1/users/user-1
```

### Create Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-1",
    "items": [{
      "product_id": "prod-1",
      "product_name": "iPhone 15",
      "quantity": 1,
      "unit_price": 699900,
      "currency": "CNY"
    }]
  }'
```

### Get User Orders
```bash
curl http://localhost:8080/api/v1/orders/user/user-1
```

## Project Structure

```
ddd/
├── api/                    # Presentation Layer
│   ├── router.go
│   ├── health/
│   ├── user/
│   ├── order/
│   ├── middleware/
│   └── response/
├── application/            # Application Layer
│   ├── user/
│   └── order/
├── domain/                 # Domain Layer (Core)
│   ├── shared/
│   ├── user/
│   └── order/
├── infrastructure/         # Infrastructure Layer
│   └── persistence/
│       └── mysql/
├── cmd/
└── main.go
```

## Test Data

Pre-loaded on startup:

**Users**: user-1 (Zhang San), user-2 (Li Si), user-3 (Wang Wu)

**Orders**: order-1 (shipped), order-2 (confirmed), order-3 (pending)
