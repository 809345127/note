package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ddd/domain/shared"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/retry"

	"gorm.io/gorm"
)

const DefaultTxTimeout = 30 * time.Second

type UnitOfWork struct {
	db               *gorm.DB
	aggregates       []shared.AggregateRoot
	outboxRepository *OutboxRepository
	retryConfig      retry.Config
}

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{
		db:               db,
		aggregates:       make([]shared.AggregateRoot, 0),
		outboxRepository: NewOutboxRepository(db),
		retryConfig:      retry.DefaultConfig,
	}
}

func (u *UnitOfWork) SetRetryConfig(config retry.Config) {
	u.retryConfig = config
}

// Execute 执行事务函数，并在提交前收集聚合事件写入 outbox。
func (u *UnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTxTimeout)
		defer cancel()
	}

	executeOnce := func(ctx context.Context) error {
		u.aggregates = make([]shared.AggregateRoot, 0)

		tx := u.db.WithContext(ctx).Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if tx.Error != nil {
			return fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}

		txCtx := persistence.ContextWithTx(ctx, tx)
		if err := fn(txCtx); err != nil {
			tx.Rollback()
			return err
		}

		for _, agg := range u.aggregates {
			events := agg.PullEvents()
			for _, event := range events {
				if err := u.outboxRepository.SaveEvent(txCtx, event); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to save event to outbox: %w", err)
				}
			}
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	}

	return retry.ExecuteWithRetry(ctx, u.retryConfig, executeOnce)
}

func (u *UnitOfWork) RegisterNew(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

func (u *UnitOfWork) RegisterDirty(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

func (u *UnitOfWork) RegisterRemoved(aggregate shared.AggregateRoot) {
	u.aggregates = append(u.aggregates, aggregate)
}

var _ shared.UnitOfWork = (*UnitOfWork)(nil)
