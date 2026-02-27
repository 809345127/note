/*
Package logger 提供 GORM 到 Zap 的日志适配。
*/
package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ddd/infrastructure/persistence"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type GormLoggerConfig struct {
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
	AddCaller                 bool
}

func DefaultGormLoggerConfig() *GormLoggerConfig {
	return &GormLoggerConfig{
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		AddCaller:                 true,
	}
}

type GormLoggerAdapter struct {
	logLevel logger.LogLevel
	logger   *zap.Logger
	config   *GormLoggerConfig
}

func NewGormLoggerAdapter(logLevel logger.LogLevel) *GormLoggerAdapter {
	return NewGormLoggerAdapterWithConfig(logLevel, DefaultGormLoggerConfig())
}

func NewGormLoggerAdapterWithConfig(logLevel logger.LogLevel, config *GormLoggerConfig) *GormLoggerAdapter {
	if config == nil {
		config = DefaultGormLoggerConfig()
	}
	baseLogger := log
	if baseLogger == nil {
		baseLogger = zap.NewNop()
	}
	return &GormLoggerAdapter{logLevel: logLevel, logger: baseLogger, config: config}
}

func (l *GormLoggerAdapter) LogMode(logLevel logger.LogLevel) logger.Interface {
	return &GormLoggerAdapter{logLevel: logLevel, logger: l.logger, config: l.config}
}

func (l *GormLoggerAdapter) extractContextFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 1)
	if requestID := persistence.RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	return fields
}

func (l *GormLoggerAdapter) getLoggerWithFields(ctx context.Context) *zap.Logger {
	loggerInstance := l.logger
	if loggerInstance == nil {
		loggerInstance = zap.NewNop()
	}
	if ctxFields := l.extractContextFields(ctx); len(ctxFields) > 0 {
		loggerInstance = loggerInstance.With(ctxFields...)
	}
	if l.config.AddCaller {
		loggerInstance = loggerInstance.WithOptions(zap.AddCaller())
	}
	return loggerInstance
}

func (l *GormLoggerAdapter) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= logger.Info {
		l.getLoggerWithFields(ctx).Info(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLoggerAdapter) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.getLoggerWithFields(ctx).Warn(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLoggerAdapter) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= logger.Error {
		l.getLoggerWithFields(ctx).Error(fmt.Sprintf(msg, args...))
	}
}

func (l *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	sql, rows := fc()
	elapsed := time.Since(begin)
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
	}
	log := l.getLoggerWithFields(ctx)

	if err != nil && l.logLevel >= logger.Error {
		if errors.Is(err, logger.ErrRecordNotFound) && l.config.IgnoreRecordNotFoundError {
			return
		}
		log.Error("Database operation failed", append(fields, zap.Error(err))...)
		return
	}

	if l.config.SlowThreshold != 0 && elapsed > l.config.SlowThreshold && l.logLevel >= logger.Warn {
		log.Warn("Slow SQL query", append(fields, zap.String("type", "slow_query"))...)
		return
	}

	if l.logLevel >= logger.Info {
		log.Info("SQL query executed", fields...)
	}
}
