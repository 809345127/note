/*
Package logger 提供项目统一日志能力。
*/
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ddd/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log       *zap.Logger
	atomLevel zap.AtomicLevel
)

func Init(cfg *config.LogConfig, env string) error {
	atomLevel = zap.NewAtomicLevelAt(parseLevel(cfg.Level))

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	switch cfg.Format {
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		if env == "dev" || env == "development" {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		}
	}

	if atomLevel.Level() == zapcore.DebugLevel && cfg.Format != "json" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	switch cfg.Output {
	case "file":
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0o755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		writeSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     7,
			Compress:   true,
		})
	default:
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, atomLevel)
	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func Get() *zap.Logger { return log }

func UpdateLevel(level string) {
	atomLevel.SetLevel(parseLevel(level))
}

func Sync() error {
	if log == nil {
		return nil
	}
	if err := log.Sync(); err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "inappropriate ioctl for device") &&
			!strings.Contains(errStr, "invalid argument") &&
			!strings.Contains(errStr, "bad file descriptor") {
			return err
		}
	}
	return nil
}

func With(fields ...zap.Field) *zap.Logger {
	if log != nil {
		return log.With(fields...)
	}
	return zap.NewNop()
}

func WithRequestID(requestID string) *zap.Logger {
	if log != nil {
		return log.With(zap.String("request_id", requestID))
	}
	return zap.NewNop()
}

func WithContext(fields map[string]any) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			zapFields = append(zapFields, zap.String(k, val))
		case int:
			zapFields = append(zapFields, zap.Int(k, val))
		case int64:
			zapFields = append(zapFields, zap.Int64(k, val))
		case int32:
			zapFields = append(zapFields, zap.Int32(k, val))
		case uint:
			zapFields = append(zapFields, zap.Uint(k, val))
		case uint64:
			zapFields = append(zapFields, zap.Uint64(k, val))
		case uint32:
			zapFields = append(zapFields, zap.Uint32(k, val))
		case float64:
			zapFields = append(zapFields, zap.Float64(k, val))
		case float32:
			zapFields = append(zapFields, zap.Float32(k, val))
		case bool:
			zapFields = append(zapFields, zap.Bool(k, val))
		case error:
			zapFields = append(zapFields, zap.Error(val))
		default:
			zapFields = append(zapFields, zap.Any(k, val))
		}
	}
	return log.With(zapFields...)
}

func Debug(msg string, fields ...zap.Field) {
	if log != nil {
		log.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...zap.Field) {
	if log != nil {
		log.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if log != nil {
		log.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if log != nil {
		log.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if log != nil {
		log.Fatal(msg, fields...)
	}
}
