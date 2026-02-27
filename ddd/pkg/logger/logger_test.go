package logger

import (
	"os"
	"testing"
	"time"

	"ddd/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNilLoggerSafety(t *testing.T) {
	log = nil
	atomLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	Debug("test debug")
	Info("test info")
	Warn("test warn")
	Error("test error")
	testLogger := With(zap.String("key", "value"))
	if testLogger == nil {
		t.Error("With() returned nil logger")
	}
	testLogger.Info("test with")

	reqLogger := WithRequestID("test-id")
	if reqLogger == nil {
		t.Error("WithRequestID() returned nil logger")
	}
	reqLogger.Info("test with request id")

	ctxLogger := WithContext(map[string]interface{}{"test": "value"})
	if ctxLogger == nil {
		t.Error("WithContext() returned nil logger")
	}
	ctxLogger.Info("test with context")

	t.Log("✓ Nil logger safety tests passed")
}

func TestDevelopmentConfig(t *testing.T) {
	devConfig := &config.LogConfig{
		Level:    "debug",
		Format:   "",
		Output:   "stdout",
		FilePath: "logs/dev.log",
	}

	if err := Init(devConfig, "development"); err != nil {
		t.Fatalf("Failed to initialize development logger: %v", err)
	}
	defer Sync()

	Info("Development logger initialized", zap.String("env", "development"))
	Debug("Debug message should appear")
	Warn("Warning message with fields", zap.String("component", "test"), zap.Int("value", 42))

	t.Log("✓ Development config tests passed")
}

func TestWithContextTypes(t *testing.T) {
	if err := Init(&config.LogConfig{Level: "info", Output: "stdout"}, "development"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Sync()

	ctxFields := map[string]interface{}{
		"string_field": "test_value",
		"int_field":    123,
		"int64_field":  int64(1234567890),
		"float_field":  3.14,
		"bool_field":   true,
	}

	ctxLogger := WithContext(ctxFields)
	ctxLogger.Info("Context logger test")

	t.Log("✓ WithContext types tests passed")
}

func TestDynamicLogLevel(t *testing.T) {
	if err := Init(&config.LogConfig{Level: "debug", Output: "stdout"}, "development"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Sync()
	Debug("This debug message should be visible")
	UpdateLevel("info")
	Debug("This debug message should NOT be visible")
	Info("Info message should still be visible")
	UpdateLevel("debug")

	t.Log("✓ Dynamic log level tests passed")
}

func TestFileOutput(t *testing.T) {
	testFile := "logs/test_file.log"
	os.Remove(testFile)
	os.MkdirAll("logs", 0755)

	fileConfig := &config.LogConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: testFile,
	}

	if err := Init(fileConfig, "production"); err != nil {
		t.Fatalf("Failed to initialize file logger: %v", err)
	}
	defer Sync()
	defer os.Remove(testFile)

	Info("File logger initialized")
	Error("Error message to file")
	for i := 0; i < 10; i++ {
		Info("Log entry for test", zap.Int("entry", i))
	}
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Log file not created: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Fatal("Log file is empty")
	}

	t.Logf("✓ File output tests passed. File size: %d bytes", fileInfo.Size())
}

func TestProductionConfig(t *testing.T) {
	prodConfig := &config.LogConfig{
		Level:  "info",
		Format: "",
		Output: "stdout",
	}

	if err := Init(prodConfig, "production"); err != nil {
		t.Fatalf("Failed to initialize production logger: %v", err)
	}
	defer Sync()

	Info("Production logger initialized", zap.String("env", "production"))
	Warn("Production warning with structured fields",
		zap.String("service", "logger-test"),
		zap.Duration("uptime", 10*time.Second))

	t.Log("✓ Production config tests passed")
}

func TestSyncFunctionality(t *testing.T) {
	if err := Init(&config.LogConfig{Level: "info", Output: "stdout"}, "development"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("Test message before sync")

	if err := Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Log("✓ Sync functionality tests passed")
}
