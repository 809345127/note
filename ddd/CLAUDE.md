# CLAUDE.md

该文件为 Claude Code（claude.ai/code）在本仓库协作时提供说明。

## 项目概览

这是一个使用 Go 实现的 DDD 示例项目，通过用户管理与订单管理场景演示 DDD 模式，主存储为 MySQL。

## 常用命令

```bash
# 运行应用（MySQL 存储）
go run . -port 8080
```

## 架构

项目采用严格 DDD 分层与依赖倒置：

```text
表现层 (api/) → 应用层 (application/) → 领域层 (domain/) ← 基础设施层 (infrastructure/)
```

### 分层结构

- `api/`：HTTP 处理器、中间件、响应封装
  - `router.go`：路由聚合与初始化
  - `{domain}/controller.go`：按限界上下文组织控制器
  - `middleware/`：请求 ID、日志、恢复、CORS、限流
  - `response/`：统一响应与分页结构
- `application/`：编排业务流程的应用服务
  - 服务依赖仓储接口、领域服务、UnitOfWork
  - 聚合写操作通过 `UoW.Execute()` 包裹
  - 包含请求/响应 DTO
- `domain/`：核心业务逻辑（不依赖框架）
  - `shared/`：公共接口（AggregateRoot、DomainEvent、UnitOfWork、Money）
  - `user/`：用户聚合（实体、值对象、仓储接口、领域服务、事件）
  - `order/`：订单聚合（含 OrderItem、仓储接口、领域服务、事件）
- `infrastructure/persistence/`：仓储与 UoW 实现
  - `context.go`：基于 context 的事务传播
  - `mysql/`：基于 GORM 的 MySQL 实现（UoW、仓储、PO）

### 关键 DDD 模式

1. 聚合根：`User` 与 `Order` 负责一致性边界
2. 值对象：`Email`、`Money`、`OrderItem`，不可变、按值相等
3. 领域服务：跨实体业务逻辑（如 `UserDomainService.CanUserPlaceOrder`）
4. 领域事件：`UserCreatedEvent`、`OrderPlacedEvent` 由聚合产生，UoW 统一收集
5. 仓储：仅抽象聚合根持久化
6. 工作单元：事务管理 + 基于 context 的事务传播 + 事件收集
7. 规格模式：可复用查询条件，支持 AND/OR/NOT 组合

### Unit Of Work 用法

应用服务使用 UoW 管理事务并收集事件：

```go
err := s.uow.Execute(ctx, func(ctx context.Context) error {
    // 创建聚合（内部记录事件）
    user, err := user.NewUser(name, email, age)
    if err != nil {
        return err
    }

    // 保存（从 context 获取事务）
    if err := s.userRepo.Save(ctx, user); err != nil {
        return err
    }

    // 注册以便事件收集
    s.uow.RegisterNew(user)
    return nil
})
```

仓储通过 `persistence.TxFromContext(ctx)` 判断是否处于事务中。

### 关键设计规则

- 领域层不依赖其他层或任何框架
- 聚合内部产生事件，UoW 负责写入 outbox
- 领域服务只读校验，不调用 `Save()`
- 应用服务负责流程编排、调用领域服务并落库
- 实体字段保持私有，通过访问器保护不变量
- 聚合内部对象修改必须经过聚合根方法
