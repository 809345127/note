package mysql

import (
	"context"
	"fmt"

	"ddd-example/domain/shared"
	"ddd-example/infrastructure/persistence"

	"gorm.io/gorm"
)

// UnitOfWork implements the Unit of Work pattern with GORM
// It manages database transactions and collects domain events from aggregates
type UnitOfWork struct {
	db         *gorm.DB
	aggregates []shared.AggregateRoot
}

// NewUnitOfWork creates a new UnitOfWork instance
func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{
		db:         db,
		aggregates: make([]shared.AggregateRoot, 0),
	}
}

// Execute runs the business logic inside a database transaction
// It:
// 1. Begins a transaction
// 2. Injects the transaction into context for repositories to use
// 3. Executes the business function
// 4. Collects events from registered aggregates and logs them (Outbox pattern simulation)
// 5. Commits on success, rolls back on error
func (u *UnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Reset aggregates for this unit of work
	u.aggregates = make([]shared.AggregateRoot, 0)

	// Begin transaction
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Create context with transaction
	txCtx := persistence.ContextWithTx(ctx, tx)

	// Execute business logic
	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}

	// Collect and process events from registered aggregates (Outbox pattern)
	for _, agg := range u.aggregates {
		events := agg.PullEvents()
		for _, event := range events {
			// In a real implementation, this would save to an outbox table
			// For this demo, we log the event
			fmt.Printf("[OUTBOX] Event saved in transaction: %s (aggregate: %s)\n",
				event.EventName(), agg.ID())
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RegisterNew registers a newly created aggregate root for event collection
func (u *UnitOfWork) RegisterNew(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// RegisterDirty registers a modified aggregate root for event collection
func (u *UnitOfWork) RegisterDirty(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// RegisterRemoved registers a deleted aggregate root for event collection
func (u *UnitOfWork) RegisterRemoved(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// Compile-time check that UnitOfWork implements shared.UnitOfWork
var _ shared.UnitOfWork = (*UnitOfWork)(nil)
