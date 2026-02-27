# DDD 最佳实践（评审基线）

## 强制规则

1. 依赖方向必须是 `api -> application -> domain <- infrastructure`。
2. 领域层禁止引入 Gin/GORM/MySQL 包。
3. 聚合字段必须私有，状态只能通过行为方法修改。
4. 应用服务负责用例编排，业务规则放在实体/领域服务。
5. 仓储只持久化聚合根，不持久化任意对象图。
6. 领域事件由 UoW 收集，并按需写入 outbox。

## 拒绝合并场景

1. 控制器直接调用仓储。
2. 领域结构体暴露可变公有字段。
3. 领域包导入基础设施或框架包。
4. 同一业务规则在控制器和领域重复实现。

## 正向参考文件

- `application/order/service.go`
- `domain/order/aggregate.go`
- `infrastructure/persistence/mysql/order_repository.go`
