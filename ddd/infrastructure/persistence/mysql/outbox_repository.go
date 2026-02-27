package mysql

import (
	"context"
	"fmt"

	"ddd/domain/shared"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/mysql/po"

	"gorm.io/gorm"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}
func (r *OutboxRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *OutboxRepository) SaveEvent(ctx context.Context, event shared.DomainEvent) error {
	if err := shared.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid domain event: %w", err)
	}
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return r.saveEventWithTx(tx, event)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveEventWithTx(tx, event)
	})
}
func (r *OutboxRepository) saveEventWithTx(tx *gorm.DB, event shared.DomainEvent) error {
	outboxPO, err := po.FromDomainEvent(event)
	if err != nil {
		return fmt.Errorf("failed to convert domain event: %w", err)
	}
	if err := tx.Create(outboxPO).Error; err != nil {
		return fmt.Errorf("failed to save event to outbox: %w", err)
	}

	return nil
}
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
func (r *OutboxRepository) MarkEventFailed(ctx context.Context, eventID string, maxRetries int) error {
	db := r.getDB(ctx)
	var event po.OutboxEventPO
	if err := db.First(&event, "id = ?", eventID).Error; err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}

	newRetryCount := event.RetryCount + 1
	newStatus := string(po.EventStatusFailed)
	if newRetryCount < maxRetries {
		newStatus = string(po.EventStatusPending)
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

var _ shared.OutboxRepository = (*OutboxRepository)(nil)
