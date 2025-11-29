# DDD示例项目 - Go语言实现

这是一个使用Go语言实现的领域驱动设计(DDD)示例项目，专为熟悉贫血模式的开发者设计，通过具体实现展示DDD的核心概念和实践方法。

## 📚 文档导航

| 文档 | 说明 |
|------|------|
| **本文件 (README.md)** | 项目概述、快速开始、API示例 |
| [DDD_CONCEPTS.md](DDD_CONCEPTS.md) | DDD核心概念详解、贫血模式对比、最佳实践、常见误区 |
| [DDD_GUIDE.md](DDD_GUIDE.md) | 实践指南：聚合根、领域事件、工作单元、仓储模式 |

## 🎯 项目目标

- **展示DDD核心概念**: 实体、值对象、聚合根、领域服务、领域事件、仓储、工作单元
- **对比贫血模式与DDD**: 帮助开发者理解架构转变（详见 [DDD_CONCEPTS.md](DDD_CONCEPTS.md#从贫血模式到ddd)）
- **提供完整示例**: 包含用户管理和订单管理的完整业务流程
- **可运行代码**: 支持 Mock 和 MySQL 两种存储方式

## 🏗️ 项目架构

项目严格遵循DDD分层架构（详细架构图见 [DDD_CONCEPTS.md](DDD_CONCEPTS.md#分层架构)）：

```
ddd/
├── api/                        # 表示层 (Presentation Layer)
│   ├── user_controller.go
│   ├── order_controller.go
│   ├── health_controller.go
│   ├── router.go
│   ├── middleware.go
│   └── response.go
├── service/                    # 应用层 (Application Layer)
│   ├── user_service.go
│   └── order_service.go
├── domain/                     # 领域层 (Domain Layer) - 核心层
│   ├── user.go                 # 用户聚合根
│   ├── order.go                # 订单聚合根
│   ├── value_objects.go        # 值对象 (Email, Money, OrderItem)
│   ├── services.go             # 领域服务
│   ├── events.go               # 领域事件
│   ├── repositories.go         # 仓储接口
│   ├── aggregate.go            # 聚合标记接口
│   ├── event_publisher.go      # 事件发布接口
│   ├── unit_of_work.go         # 工作单元接口
│   └── tx_unit_of_work.go      # 事务工作单元
├── infrastructure/             # 基础设施层 (Infrastructure Layer)
│   └── persistence/
│       ├── mocks/              # Mock 实现
│       │   ├── user_repository.go
│       │   ├── order_repository.go
│       │   └── event_publisher.go
│       └── mysql/              # MySQL 实现
│           ├── mysql_config.go
│           ├── user_repository.go
│           └── order_repository.go
├── util/
│   └── utils.go
├── cmd/
│   └── app.go
├── main.go
├── go.mod
├── README.md                   # 本文件
├── DDD_CONCEPTS.md             # DDD概念详解
└── DDD_GUIDE.md                # DDD实践指南
```

## 🧩 DDD核心概念

> 📖 详细概念讲解请参阅 [DDD_CONCEPTS.md](DDD_CONCEPTS.md)，实现细节请参阅 [DDD_GUIDE.md](DDD_GUIDE.md)

| 概念 | 说明 | 项目实现 |
|------|------|----------|
| **聚合根** | 一组相关对象的一致性边界入口 | `User`, `Order` |
| **实体** | 有唯一标识，包含业务逻辑 | `User`, `Order` |
| **值对象** | 不可变，通过值比较 | `Email`, `Money`, `OrderItem` |
| **领域服务** | 跨实体的业务规则 | `UserDomainService`, `OrderDomainService` |
| **领域事件** | 记录业务系统中的重要事件 | `UserCreatedEvent`, `OrderPlacedEvent` |
| **仓储** | 聚合根的持久化抽象 | `UserRepository`, `OrderRepository` |
| **工作单元** | 事务边界管理 | `UnitOfWork` |

## 🚀 快速开始

### 环境要求

- Go 1.21 或更高版本
- Git

### 安装依赖

```bash
cd /home/shize/note/ddd
go mod download
```

### 运行项目

```bash
go run main.go
```

或使用指定端口：

```bash
go run main.go -port 8080
```

### 测试API

项目启动后，可以通过以下方式测试API：

#### 1. 健康检查
```bash
curl http://localhost:8080/api/v1/health
```

#### 2. 创建用户
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "email": "zhangsan@example.com",
    "age": 25
  }'
```

#### 3. 获取用户列表
```bash
curl http://localhost:8080/api/v1/users
```

#### 4. 获取指定用户
```bash
curl http://localhost:8080/api/v1/users/user-1
```

#### 5. 创建订单
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

#### 6. 获取用户订单
```bash
curl http://localhost:8080/api/v1/orders/user/user-1
```

## 📖 核心业务流程

### 用户创建流程

1. **API层**接收创建用户请求
2. **应用服务**验证邮箱唯一性
3. **领域层**创建用户实体（包含业务规则验证）
4. **仓储层**保存用户数据
5. **领域事件**发布用户创建事件
6. **API层**返回创建结果

### 订单创建流程

1. **API层**接收创建订单请求
2. **应用服务**检查用户是否可以下单（通过领域服务）
3. **领域服务**验证用户状态、年龄、待处理订单数量等
4. **领域层**创建订单实体
5. **仓储层**保存订单数据
6. **领域事件**发布订单创建事件
7. **API层**返回创建结果

## 🔍 从贫血模式到DDD

> 📖 详细对比和代码示例请参阅 [DDD_CONCEPTS.md - 从贫血模式到DDD](DDD_CONCEPTS.md#从贫血模式到ddd)

| 贫血模式 | DDD模式 |
|---------|---------|
| 实体只包含数据 | 实体包含业务逻辑 |
| 业务逻辑分散在服务层 | 业务逻辑封装在领域层 |
| 低内聚、难维护 | 高内聚、易测试 |

## 🧪 测试数据

项目启动时会自动创建以下测试数据：

### 测试用户
- **user-1**: 张三 (zhangsan@example.com, 25岁)
- **user-2**: 李四 (lisi@example.com, 30岁)  
- **user-3**: 王五 (wangwu@example.com, 35岁)

### 测试订单
- **order-1**: user-1的订单，包含iPhone 15和MacBook Pro（已发货）
- **order-2**: user-2的订单，包含2个AirPods Pro（已确认）
- **order-3**: user-1的订单，包含iPhone 15（待处理）

## 🔧 扩展方向

- **CQRS模式**: 分离命令和查询职责（详见 [DDD_GUIDE.md - 进阶主题](DDD_GUIDE.md#进阶主题)）
- **事件溯源**: 通过事件序列重建聚合状态
- **更多限界上下文**: 商品管理、库存管理、支付管理

## 📚 学习资源

> 📖 更多学习资源请参阅 [DDD_CONCEPTS.md - 学习资源](DDD_CONCEPTS.md#学习资源)

**推荐书籍**:
- 《领域驱动设计》- Eric Evans
- 《实现领域驱动设计》- Vaughn Vernon