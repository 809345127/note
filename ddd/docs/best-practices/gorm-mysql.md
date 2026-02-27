# GORM + MySQL 最佳实践（评审基线）

## 强制规则

1. 仓储方法必须接收 `context.Context`。
2. 事务边界由 Unit Of Work 统一管理，禁止控制器/应用层临时拼事务。
3. 乐观锁必须基于 `version` 字段，并返回领域级并发冲突错误。
4. 生产库表结构由外部系统管理，应用不在启动阶段改表。
5. 学习基线固定启用 outbox；生产项目可按需改为特性开关。
6. 列表查询需避免 N+1。

## 拒绝合并场景

1. 控制器/应用层直接使用 `*gorm.DB`。
2. 启动时无条件执行 `AutoMigrate`。
3. 未做错误映射，直接把 SQL/驱动错误透传到 API。
4. 高频查询字段缺失唯一索引或检索索引。

## 正向参考文件

- `infrastructure/persistence/mysql/unit_of_work.go`
- `infrastructure/persistence/mysql/user_repository.go`
- `infrastructure/persistence/mysql/order_repository.go`
