package mysql

import (
	"context"
	"fmt"

	"ddd/domain/shared"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/retry"

	"gorm.io/gorm"
)

// UnitOfWork implements the Unit of Work pattern with GORM
// It manages database transactions and collects domain events from aggregates
type UnitOfWork struct {
	db               *gorm.DB
	aggregates       []shared.AggregateRoot
	outboxRepository *OutboxRepository
	retryConfig      retry.Config
}

// NewUnitOfWork creates a new UnitOfWork instance
func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{
		db:               db,
		aggregates:       make([]shared.AggregateRoot, 0),
		outboxRepository: NewOutboxRepository(db),
		retryConfig:      retry.DefaultConfig,
	}
}

// SetRetryConfig updates the retry configuration for this UnitOfWork
func (u *UnitOfWork) SetRetryConfig(config retry.Config) {
	u.retryConfig = config
}

// Execute runs the business logic inside a database transaction
// It:
// 1. Begins a transaction
// 2. Injects the transaction into context for repositories to use
// 3. Executes the business function
// 4. Collects events from registered aggregates and logs them (Outbox pattern simulation)
// 5. Commits on success, rolls back on error
// 6. Automatically retries on retryable errors (concurrent modification, deadlocks, etc.)
func (u *UnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Define the transaction execution function that will be retried
	executeOnce := func(ctx context.Context) error {
		// Reset aggregates for this attempt
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
				// Save event to outbox table using the transaction context
				if err := u.outboxRepository.SaveEvent(txCtx, event); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to save event to outbox: %w", err)
				}
			}
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	// Execute with retry logic
	return retry.ExecuteWithRetry(ctx, u.retryConfig, executeOnce)
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
