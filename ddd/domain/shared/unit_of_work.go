package shared

import "context"

// UnitOfWork Unit of Work Interface
// DDD principles:
// 1. Manage transaction boundaries
// 2. Track changes to aggregate roots
// 3. Collect and persist domain events (Outbox pattern)
// 4. Ensure aggregate consistency
//
// Usage pattern:
//
//	err := uow.Execute(ctx, func(ctx context.Context) error {
//	    // Create or load aggregate roots
//	    user, _ := userRepo.FindByID(ctx, userID)
//
//	    // Execute business operations
//	    user.Deactivate()
//
//	    // Save aggregate root (uses transaction from ctx)
//	    if err := userRepo.Save(ctx, user); err != nil {
//	        return err
//	    }
//
//	    // Register aggregate for event collection
//	    uow.RegisterDirty(user)
//	    return nil
//	})
type UnitOfWork interface {
	// Execute executes business operations in a transaction
	// - Begins transaction and injects it into context
	// - Executes the provided function
	// - Collects events from registered aggregates
	// - Saves events to outbox (if configured)
	// - Commits on success, rolls back on error
	Execute(ctx context.Context, fn func(ctx context.Context) error) error

	// RegisterNew registers a newly created aggregate root for event collection
	RegisterNew(aggregate AggregateRoot)

	// RegisterDirty registers a modified aggregate root for event collection
	RegisterDirty(aggregate AggregateRoot)

	// RegisterRemoved registers a deleted aggregate root for event collection
	RegisterRemoved(aggregate AggregateRoot)
}

// OutboxRepository Outbox Repository Interface
// Used to save domain events to outbox table, committed in the same transaction as business data
type OutboxRepository interface {
	// SaveEvent saves event to outbox table (in current transaction)
	SaveEvent(ctx context.Context, event DomainEvent) error
}
