/*
Package logger - GORM logger adapter tests
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

// TestGormLoggerAdapter tests the GORM logger adapter functionality
func TestGormLoggerAdapter(t *testing.T) {
	// Save the original logger and restore it after the test
	originalLogger := log
	defer func() { log = originalLogger }()

	// Test cases
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
			// Create a fresh zap observer for each test case
			core, logs := observer.New(zapcore.DebugLevel)
			testLogger := zap.New(core)
			log = testLogger

			// Test with default configuration
			adapter := NewGormLoggerAdapter(tc.logLevel)

			// Test LogMode
			newAdapter := adapter.LogMode(logger.Info)
			if newAdapter == nil {
				t.Fatal("LogMode should return a new adapter")
			}

			// Test Info method
			adapter.Info(context.Background(), "test info message")

			// Test Warn method
			adapter.Warn(context.Background(), "test warn message")

			// Test Error method
			adapter.Error(context.Background(), "test error message")

			// Test Trace method
			begin := time.Now()
			adapter.Trace(context.Background(), begin, func() (string, int64) {
				return "SELECT * FROM users", 1
			}, nil)

			// Check for specific log messages
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
					// Check that SQL is included in the fields
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

			// Verify expected logs were found based on log level
			if tc.logLevel <= logger.Info {
				if !foundInfo {
					t.Error("Info message not found in logs")
				}
			} else {
				// At Warn level or higher, Info should be filtered out
				if foundInfo {
					t.Error("Info message should not be logged at Warn level")
				}
			}

			// Warn should always be logged when level is Warn or lower
			if tc.logLevel <= logger.Warn {
				if !foundWarn {
					t.Error("Warn message not found in logs")
				}
			}

			// Error should always be logged when level is Error or lower
			if tc.logLevel <= logger.Error {
				if !foundError {
					t.Error("Error message not found in logs")
				}
			}

			// Trace should only be logged at Info level
			if tc.shouldLogDebug {
				if !foundTrace {
					t.Error("Trace message not found in logs for Info level")
				}
			} else {
				// At Warn level, Trace should be filtered out
				if foundTrace {
					t.Error("Trace message should not be logged at Warn level")
				}
			}
		})
	}
}

// TestGormLoggerAdapterWithConfig tests the GORM logger adapter with custom configuration
func TestGormLoggerAdapterWithConfig(t *testing.T) {
	// Save the original logger and restore it after the test
	originalLogger := log
	defer func() { log = originalLogger }()

	// Create a fresh zap observer
	core, logs := observer.New(zapcore.DebugLevel)
	testLogger := zap.New(core)
	log = testLogger

	// Create custom configuration
	customConfig := &GormLoggerConfig{
		SlowThreshold:             10 * time.Millisecond, // Very low threshold for testing
		IgnoreRecordNotFoundError: true,
		AddCaller:                 true,
	}

	// Test with custom configuration
	adapter := NewGormLoggerAdapterWithConfig(logger.Info, customConfig)

	// Test with context containing request_id
	ctx := context.WithValue(context.Background(), "request_id", "test-request-123")

	// Test Trace with slow query (should trigger warn)
	begin := time.Now()
	adapter.Trace(ctx, begin, func() (string, int64) {
		// Simulate slow query by sleeping
		time.Sleep(15 * time.Millisecond)
		return "SELECT * FROM slow_table", 1
	}, nil)

	// Test record not found error (should be ignored)
	adapter.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 999", 0
	}, logger.ErrRecordNotFound)

	// Check logs
	foundSlowQuery := false
	foundRequestID := false
	for _, logEntry := range logs.All() {
		if logEntry.Message == "Slow SQL query" {
			foundSlowQuery = true
			// Check for request_id in context
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
