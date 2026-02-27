package shared

import "context"

// UnitOfWork 管理事务边界与聚合事件收集。
type UnitOfWork interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
	RegisterNew(aggregate AggregateRoot)
	RegisterDirty(aggregate AggregateRoot)
	RegisterRemoved(aggregate AggregateRoot)
}

type UnitOfWorkFactory interface {
	New() UnitOfWork
}

type OutboxRepository interface {
	SaveEvent(ctx context.Context, event DomainEvent) error
}
