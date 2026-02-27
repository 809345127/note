package mysql

import (
	"context"
	"fmt"
	"time"

	"ddd/pkg/logger"

	"go.uber.org/zap"
)

type OutboxPublisher interface {
	Publish(ctx context.Context, eventType, payload string) error
}
type LoggingOutboxPublisher struct{}

func (p *LoggingOutboxPublisher) Publish(ctx context.Context, eventType, payload string) error {
	logger.Info("Outbox event published",
		zap.String("event_type", eventType),
		zap.String("payload", payload),
	)
	return nil
}

type OutboxWorker struct {
	repository   *OutboxRepository
	publisher    OutboxPublisher
	pollInterval time.Duration
	batchSize    int
	maxRetries   int
}

func NewOutboxWorker(
	repository *OutboxRepository,
	publisher OutboxPublisher,
	pollInterval time.Duration,
	batchSize int,
	maxRetries int,
) (*OutboxWorker, error) {
	if repository == nil {
		return nil, fmt.Errorf("outbox repository is required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("outbox publisher is required")
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be positive")
	}
	if batchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}
	if maxRetries <= 0 {
		return nil, fmt.Errorf("max retries must be positive")
	}

	return &OutboxWorker{
		repository:   repository,
		publisher:    publisher,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		maxRetries:   maxRetries,
	}, nil
}
func (w *OutboxWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				logger.Error("Outbox batch processing failed", zap.Error(err))
			}
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) error {
	events, err := w.repository.GetPendingEvents(ctx, w.batchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		if err := w.repository.MarkEventProcessing(ctx, event.ID); err != nil {
			logger.Warn("Skip outbox event due to lock contention",
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
			continue
		}

		if err := w.publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			failErr := w.repository.MarkEventFailed(ctx, event.ID, w.maxRetries)
			if failErr != nil {
				logger.Error("Failed to mark outbox event as failed",
					zap.String("event_id", event.ID),
					zap.Error(failErr),
				)
			}
			continue
		}

		if err := w.repository.MarkEventPublished(ctx, event.ID); err != nil {
			logger.Error("Failed to mark outbox event as published",
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
		}
	}

	return nil
}
