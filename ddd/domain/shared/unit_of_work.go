package shared

import "context"

// UnitOfWork Unit of Work Interface
// DDD principles:
// 1. Track changes to aggregate roots
// 2. Manage transaction boundaries
// 3. Coordinate repository save operations
// 4. Ensure aggregate consistency
//
// Usage pattern:
// uow := unitOfWorkFactory.New()
//
//	err := uow.Execute(func() error {
//	    // Load aggregate roots
//	    user, _ := userRepo.FindByID(userID)
//	    order, _ := orderRepo.FindByID(orderID)
//
//	    // Execute business operations
//	    user.Deactivate()
//	    order.Cancel()
//
//	    // Save (handled automatically during unit of work execution)
//	    uow.RegisterDirty(user)
//	    uow.RegisterDirty(order)
//
//	    return nil
//	})
type UnitOfWork interface {
	// Execute Execute business operations in transaction
	// Automatically handles begin, commit and rollback
	Execute(fn func() error) error

	// RegisterNew Register newly created aggregate root
	RegisterNew(aggregate AggregateRoot)

	// RegisterDirty Register modified aggregate root
	RegisterDirty(aggregate AggregateRoot)

	// RegisterClean Register clean aggregate root (unchanged)
	RegisterClean(aggregate AggregateRoot)

	// RegisterRemoved Register deleted aggregate root
	RegisterRemoved(aggregate AggregateRoot)
}

// UnitOfWorkFactory Unit of Work Factory
type UnitOfWorkFactory interface {
	// New Create new unit of work
	New() UnitOfWork
}

// TransactionManager Transaction Manager Interface
type TransactionManager interface {
	// Begin Begin transaction
	Begin() error

	// Commit Commit transaction
	Commit() error

	// Rollback Rollback transaction
	Rollback() error

	// InTransaction Whether in transaction
	InTransaction() bool
}

// OutboxRepository Outbox Repository Interface
// Used to save domain events to outbox table, committed in the same transaction as business data
type OutboxRepository interface {
	// SaveEvent Save event to outbox table (in current transaction)
	SaveEvent(ctx context.Context, event DomainEvent) error
}

// IsolationLevel Transaction Isolation Level
type IsolationLevel string

const (
	// ReadUncommitted Read Uncommitted
	ReadUncommitted IsolationLevel = "READ_UNCOMMITTED"
	// ReadCommitted Read Committed
	ReadCommitted IsolationLevel = "READ_COMMITTED"
	// RepeatableRead Repeatable Read
	RepeatableRead IsolationLevel = "REPEATABLE_READ"
	// Serializable Serializable (highest isolation level)
	Serializable IsolationLevel = "SERIALIZABLE"
)

// ExecuteEvent Execute Event
type ExecuteEvent struct {
	Type       ExecuteEventType
	Aggregates []AggregateRoot
	Error      error
}

// ExecuteEventType Execute Event Type
type ExecuteEventType string

const (
	EventBeforeCommit ExecuteEventType = "BEFORE_COMMIT"
	EventAfterCommit  ExecuteEventType = "AFTER_COMMIT"
	EventError        ExecuteEventType = "ERROR"
)
