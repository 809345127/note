/*
Package logger - GORM 日志适配器测试
*/
package logger

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm/logger"
)

func TestGormLoggerAdapter(t *testing.T) {
	originalLogger := log
	defer func() { log = originalLogger }()
	testCases := []struct {
		name           string
		logLevel       logger.LogLevel
		expectedLevel  zapcore.Level
		shouldLogDebug bool
	}{
		{"Warn Level", logger.Warn, zapcore.WarnLevel, false},
		{"Info Level", logger.Info, zapcore.InfoLevel, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			testLogger := zap.New(core)
			log = testLogger
			adapter := NewGormLoggerAdapter(tc.logLevel)
			newAdapter := adapter.LogMode(logger.Info)
			if newAdapter == nil {
				t.Fatal("LogMode should return a new adapter")
			}
			adapter.Info(context.Background(), "test info message")
			adapter.Warn(context.Background(), "test warn message")
			adapter.Error(context.Background(), "test error message")
			begin := time.Now()
			adapter.Trace(context.Background(), begin, func() (string, int64) {
				return "SELECT * FROM users", 1
			}, nil)
			foundInfo := false
			foundWarn := false
			foundError := false
			foundTrace := false

			for _, logEntry := range logs.All() {
				switch logEntry.Message {
				case "test info message":
					foundInfo = true
				case "test warn message":
					foundWarn = true
				case "test error message":
					foundError = true
				case "SQL query executed":
					foundTrace = true
					hasSQL := false
					for _, field := range logEntry.Context {
						if field.Key == "sql" {
							hasSQL = true
							break
						}
					}
					if !hasSQL {
						t.Error("SQL query not found in trace log fields")
					}
				}
			}
			if tc.logLevel >= logger.Info {
				if !foundInfo {
					t.Error("Info message not found in logs")
				}
			} else {
				if foundInfo {
					t.Error("Info message should not be logged at Warn level")
				}
			}
			if tc.logLevel >= logger.Warn {
				if !foundWarn {
					t.Error("Warn message not found in logs")
				}
			}
			if tc.logLevel >= logger.Error {
				if !foundError {
					t.Error("Error message not found in logs")
				}
			}
			if tc.shouldLogDebug {
				if !foundTrace {
					t.Error("Trace message not found in logs for Info level")
				}
			} else {
				if foundTrace {
					t.Error("Trace message should not be logged at Warn level")
				}
			}
		})
	}
}
func TestGormLoggerAdapterWithConfig(t *testing.T) {
	originalLogger := log
	defer func() { log = originalLogger }()
	core, logs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)
	log = testLogger
	customConfig := &GormLoggerConfig{
		SlowThreshold:             10 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
		AddCaller:                 true,
	}
	adapter := NewGormLoggerAdapterWithConfig(logger.Info, customConfig)
	ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
	begin := time.Now()
	adapter.Trace(ctx, begin, func() (string, int64) {
		time.Sleep(15 * time.Millisecond)
		return "SELECT * FROM slow_table", 1
	}, nil)
	adapter.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 999", 0
	}, logger.ErrRecordNotFound)
	foundSlowQuery := false
	foundRequestID := false
	for _, logEntry := range logs.All() {
		if logEntry.Message == "Slow SQL query" {
			foundSlowQuery = true
			for _, field := range logEntry.Context {
				if field.Key == "request_id" && field.String == "test-request-123" {
					foundRequestID = true
					break
				}
			}
		}
		if logEntry.Message == "Database record not found" {
			t.Error("Record not found error should be ignored with custom config")
		}
	}

	if !foundSlowQuery {
		t.Error("Slow query should be logged with warn level")
	}

	if !foundRequestID {
		t.Error("Request ID should be propagated from context")
	}
}
