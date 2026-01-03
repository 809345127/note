/*
Package logger - GORM logger adapter

This adapter implements GORM's logger interface and delegates to our custom
Zap-based logger, allowing GORM database operations to be logged using the
same logging infrastructure as the rest of the application.
*/
package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// GormLoggerConfig defines configuration options for GORM logger adapter

type GormLoggerConfig struct {
	SlowThreshold             time.Duration // Threshold for slow queries
	IgnoreRecordNotFoundError bool          // Whether to ignore record not found errors
	AddCaller                 bool          // Whether to add caller information
}

// DefaultGormLoggerConfig returns default configuration
func DefaultGormLoggerConfig() *GormLoggerConfig {
	return &GormLoggerConfig{
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		AddCaller:                 true,
	}
}

// GormLoggerAdapter implements the gorm.logger.Interface
// and delegates to our custom Zap-based logger

type GormLoggerAdapter struct {
	logLevel logger.LogLevel
	logger   *zap.Logger
	config   *GormLoggerConfig
}

// NewGormLoggerAdapter creates a new GORM logger adapter
func NewGormLoggerAdapter(logLevel logger.LogLevel) *GormLoggerAdapter {
	return NewGormLoggerAdapterWithConfig(logLevel, DefaultGormLoggerConfig())
}

// NewGormLoggerAdapterWithConfig creates a new GORM logger adapter with custom configuration
func NewGormLoggerAdapterWithConfig(logLevel logger.LogLevel, config *GormLoggerConfig) *GormLoggerAdapter {
	return &GormLoggerAdapter{
		logLevel: logLevel,
		logger:   log,
		config:   config,
	}
}

// LogMode sets the log level for the adapter
func (l *GormLoggerAdapter) LogMode(logLevel logger.LogLevel) logger.Interface {
	return &GormLoggerAdapter{
		logLevel: logLevel,
		logger:   l.logger,
		config:   l.config,
	}
}

// extractContextFields extracts common fields from context
func (l *GormLoggerAdapter) extractContextFields(ctx context.Context) []zap.Field {
	var fields []zap.Field

	// Extract request_id if present
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields = append(fields, zap.String("request_id", requestID))
	}

	return fields
}

// getLoggerWithFields returns logger with additional fields and caller info if configured
func (l *GormLoggerAdapter) getLoggerWithFields(ctx context.Context) *zap.Logger {
	logger := l.logger

	// Add context fields
	if ctxFields := l.extractContextFields(ctx); len(ctxFields) > 0 {
		logger = logger.With(ctxFields...)
	}

	// Add caller information
	if l.config.AddCaller {
		logger = logger.WithOptions(zap.AddCaller())
	}

	return logger
}

// Info logs information messages
func (l *GormLoggerAdapter) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel <= logger.Info {
		logger := l.getLoggerWithFields(ctx)
		logger.Info(fmt.Sprintf(msg, args...))
	}
}

// Warn logs warning messages
func (l *GormLoggerAdapter) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel <= logger.Warn {
		logger := l.getLoggerWithFields(ctx)
		logger.Warn(fmt.Sprintf(msg, args...))
	}
}

// Error logs error messages
func (l *GormLoggerAdapter) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel <= logger.Error {
		logger := l.getLoggerWithFields(ctx)
		logger.Error(fmt.Sprintf(msg, args...))
	}
}

// Trace logs SQL queries and their execution details
func (l *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// Only log trace at Info level or higher (Info/Debug)
	if l.logLevel < logger.Info {
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

	if err != nil {
		// Skip record not found error if configured
		if err == logger.ErrRecordNotFound && l.config.IgnoreRecordNotFoundError {
			return
		}
		log.Error("Database operation failed", append(fields, zap.Error(err))...)
		return
	}

	// Log slow queries as warnings
	if elapsed > l.config.SlowThreshold {
		log.Warn("Slow SQL query", append(fields, zap.String("type", "slow_query"))...)
	} else {
		log.Debug("SQL query executed", fields...)
	}
}
