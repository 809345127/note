package mysql

import (
	"context"
	"fmt"

	"ddd/domain/shared"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/mysql/po"

	"gorm.io/gorm"
)

// OutboxRepository MySQL/GORM implementation of outbox repository
// Implements transactional outbox pattern for reliable domain event publishing
type OutboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository Create outbox repository
func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// getDB returns the transaction from context if available, otherwise the default db
func (r *OutboxRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

// SaveEvent Save domain event to outbox table
// Uses transaction from context when called within UoW.Execute()
// Creates its own transaction when called standalone
func (r *OutboxRepository) SaveEvent(ctx context.Context, event shared.DomainEvent) error {
	// Validate event
	if err := shared.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid domain event: %w", err)
	}

	// Check if we're already in a UoW transaction
	if tx := persistence.TxFromContext(ctx); tx != nil {
		// Use the existing transaction from UoW
		return r.saveEventWithTx(tx, event)
	}

	// No UoW transaction - create our own for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveEventWithTx(tx, event)
	})
}

// saveEventWithTx performs the actual event save within a transaction
func (r *OutboxRepository) saveEventWithTx(tx *gorm.DB, event shared.DomainEvent) error {
	// Convert domain event to persistence object
	outboxPO, err := po.FromDomainEvent(event)
	if err != nil {
		return fmt.Errorf("failed to convert domain event: %w", err)
	}

	// Save to outbox table
	if err := tx.Create(outboxPO).Error; err != nil {
		return fmt.Errorf("failed to save event to outbox: %w", err)
	}

	return nil
}

// GetPendingEvents Get pending events for processing
// Used by OutboxProcessor to retrieve events for publishing
func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*po.OutboxEventPO, error) {
	var events []*po.OutboxEventPO
	db := r.getDB(ctx)

	err := db.Where("status = ?", string(po.EventStatusPending)).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending events: %w", err)
	}

	return events, nil
}

// MarkEventProcessing Mark event as being processed
// Used by OutboxProcessor to prevent concurrent processing
func (r *OutboxRepository) MarkEventProcessing(ctx context.Context, eventID string) error {
	db := r.getDB(ctx)
	result := db.Model(&po.OutboxEventPO{}).
		Where("id = ? AND status = ?", eventID, string(po.EventStatusPending)).
		Updates(map[string]interface{}{
			"status":     string(po.EventStatusProcessing),
			"updated_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("event not found or already being processed: %s", eventID)
	}

	return nil
}

// MarkEventPublished Mark event as successfully published
func (r *OutboxRepository) MarkEventPublished(ctx context.Context, eventID string) error {
	db := r.getDB(ctx)
	result := db.Model(&po.OutboxEventPO{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":     string(po.EventStatusPublished),
			"updated_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// MarkEventFailed Mark event as failed to publish
// Increments retry count for retry logic
func (r *OutboxRepository) MarkEventFailed(ctx context.Context, eventID string, maxRetries int) error {
	db := r.getDB(ctx)

	// First check current retry count
	var event po.OutboxEventPO
	if err := db.First(&event, "id = ?", eventID).Error; err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}

	newRetryCount := event.RetryCount + 1
	newStatus := string(po.EventStatusFailed)
	if newRetryCount < maxRetries {
		newStatus = string(po.EventStatusPending) // Retry later
	}

	result := db.Model(&po.OutboxEventPO{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":      newStatus,
			"retry_count": newRetryCount,
			"updated_at":  gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Compile-time interface implementation check
var _ shared.OutboxRepository = (*OutboxRepository)(nil)