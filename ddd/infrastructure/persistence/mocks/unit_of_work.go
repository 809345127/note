package mocks

import (
	"context"
	"fmt"

	"ddd/domain/shared"
)

// MockUnitOfWork is a mock implementation of UnitOfWork for testing
// It doesn't use real transactions but still collects and logs events
type MockUnitOfWork struct {
	aggregates []shared.AggregateRoot
}

// NewMockUnitOfWork creates a new MockUnitOfWork instance
func NewMockUnitOfWork() *MockUnitOfWork {
	return &MockUnitOfWork{
		aggregates: make([]shared.AggregateRoot, 0),
	}
}

// Execute runs the business logic without real transaction management
// It still collects events from registered aggregates for demonstration
func (u *MockUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Reset aggregates for this unit of work
	u.aggregates = make([]shared.AggregateRoot, 0)

	// Execute business logic (context passed through without modification in mock)
	if err := fn(ctx); err != nil {
		return err
	}

	// Collect and log events from registered aggregates (Outbox pattern simulation)
	for _, agg := range u.aggregates {
		events := agg.PullEvents()
		for _, event := range events {
			fmt.Printf("[MOCK OUTBOX] Event saved: %s (aggregate: %s, time: %s)\n",
				event.EventName(), agg.ID(), event.OccurredOn().Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

// RegisterNew registers a newly created aggregate root for event collection
func (u *MockUnitOfWork) RegisterNew(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// RegisterDirty registers a modified aggregate root for event collection
func (u *MockUnitOfWork) RegisterDirty(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// RegisterRemoved registers a deleted aggregate root for event collection
func (u *MockUnitOfWork) RegisterRemoved(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

// Compile-time check that MockUnitOfWork implements shared.UnitOfWork
var _ shared.UnitOfWork = (*MockUnitOfWork)(nil)
