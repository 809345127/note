package shared

import "context"

// UnitOfWork 工作单元接口
// DDD原则：
// 1. 跟踪聚合根的变化
// 2. 管理事务边界
// 3. 协调仓储的保存操作
// 4. 保证聚合的一致性
//
// 使用模式：
// uow := unitOfWorkFactory.New()
//
//	err := uow.Execute(func() error {
//	    // 加载聚合根
//	    user, _ := userRepo.FindByID(userID)
//	    order, _ := orderRepo.FindByID(orderID)
//
//	    // 执行业务操作
//	    user.Deactivate()
//	    order.Cancel()
//
//	    // 保存（在工作单元执行时自动处理）
//	    uow.RegisterDirty(user)
//	    uow.RegisterDirty(order)
//
//	    return nil
//	})
type UnitOfWork interface {
	// Execute 在事务中执行业务操作
	// 自动处理begin、commit和rollback
	Execute(fn func() error) error

	// RegisterNew 注册新建的聚合根
	RegisterNew(aggregate AggregateRoot)

	// RegisterDirty 注册被修改的聚合根
	RegisterDirty(aggregate AggregateRoot)

	// RegisterClean 注册干净的聚合根（未改变）
	RegisterClean(aggregate AggregateRoot)

	// RegisterRemoved 注册被删除的聚合根
	RegisterRemoved(aggregate AggregateRoot)
}

// UnitOfWorkFactory 工作单元工厂
type UnitOfWorkFactory interface {
	// New 创建新的工作单元
	New() UnitOfWork
}

// TransactionManager 事务管理器接口
type TransactionManager interface {
	// Begin 开始事务
	Begin() error

	// Commit 提交事务
	Commit() error

	// Rollback 回滚事务
	Rollback() error

	// InTransaction 是否处于事务中
	InTransaction() bool
}

// OutboxRepository Outbox 仓储接口
// 用于保存领域事件到 outbox 表，与业务数据同事务提交
type OutboxRepository interface {
	// SaveEvent 保存事件到 outbox 表（在当前事务中）
	SaveEvent(ctx context.Context, event DomainEvent) error
}

// IsolationLevel 事务隔离级别
type IsolationLevel string

const (
	// ReadUncommitted 未提交读
	ReadUncommitted IsolationLevel = "READ_UNCOMMITTED"
	// ReadCommitted 提交读
	ReadCommitted IsolationLevel = "READ_COMMITTED"
	// RepeatableRead 可重复读
	RepeatableRead IsolationLevel = "REPEATABLE_READ"
	// Serializable 可串行化（最高隔离级别）
	Serializable IsolationLevel = "SERIALIZABLE"
)

// ExecuteEvent 执行事件
type ExecuteEvent struct {
	Type       ExecuteEventType
	Aggregates []AggregateRoot
	Error      error
}

// ExecuteEventType 执行事件类型
type ExecuteEventType string

const (
	EventBeforeCommit ExecuteEventType = "BEFORE_COMMIT"
	EventAfterCommit  ExecuteEventType = "AFTER_COMMIT"
	EventError        ExecuteEventType = "ERROR"
)
