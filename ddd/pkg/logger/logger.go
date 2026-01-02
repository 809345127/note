/*
Package logger - 日志包

设计原则:
1. 使用 zap 作为日志库（高性能、结构化）
2. 支持多种输出格式（JSON/Console）
3. 支持文件和标准输出
4. 提供 RequestID 支持，便于请求追踪

使用示例:

	// 基础使用
	logger.Info("user login", zap.String("user_id", "123"))

	// 错误日志（推荐在 API 层使用 response.HandleAppError，它会自动记录）
	logger.Error("failed to get order", zap.Error(err), zap.String("order_id", id))

	// 带 RequestID
	reqLogger := logger.With(zap.String("request_id", requestID))
	reqLogger.Info("processing request")
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

// Init 初始化日志
func Init(cfg *config.LogConfig, env string) error {
	// 设置日志级别
	atomLevel = zap.NewAtomicLevelAt(parseLevel(cfg.Level))

	// 创建编码器配置
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

	// 根据配置选择编码器格式
	var encoder zapcore.Encoder
	switch cfg.Format {
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		// 根据环境设置默认格式
		if env == "development" {
			// 开发环境默认使用控制台格式
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			// 生产环境默认使用 JSON 格式（K8s 最佳实践）
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		}
	}

	// 调试级别总是使用控制台格式以便于开发
	if atomLevel.Level() == zapcore.DebugLevel && cfg.Format != "json" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 根据配置选择输出目标
	var writeSyncer zapcore.WriteSyncer
	switch cfg.Output {
	case "file":
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		// 使用 lumberjack 进行日志轮转
		writeSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    10,   // 单个日志文件最大 10MB
			MaxBackups: 5,    // 保留 5 个旧日志文件
			MaxAge:     7,    // 日志文件最多保留 7 天
			Compress:   true, // 压缩旧日志文件
		})
	default: // 默认使用标准输出（K8s 最佳实践）
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, atomLevel)

	// 创建日志实例
	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// parseLevel 解析日志级别
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

// Get 获取日志实例
func Get() *zap.Logger {
	return log
}

// UpdateLevel 更新日志级别
func UpdateLevel(level string) {
	atomLevel.SetLevel(parseLevel(level))
}

// Sync 刷新日志缓冲区
func Sync() error {
	if log != nil {
		if err := log.Sync(); err != nil {
			// 忽略常见的无害 sync 错误
			errStr := err.Error()
			if !strings.Contains(errStr, "inappropriate ioctl for device") &&
				!strings.Contains(errStr, "invalid argument") &&
				!strings.Contains(errStr, "bad file descriptor") {
				return err
			}
		}
	}
	return nil
}

// With 添加上下文字段到日志器
func With(fields ...zap.Field) *zap.Logger {
	if log != nil {
		return log.With(fields...)
	}
	return zap.NewNop()
}

// WithRequestID 创建带请求 ID 的日志器
// 用于在整个请求处理过程中保持请求 ID
func WithRequestID(requestID string) *zap.Logger {
	if log != nil {
		return log.With(zap.String("request_id", requestID))
	}
	return zap.NewNop()
}

// WithContext 创建带多个上下文字段的日志器
// 支持多种数据类型的字段值
func WithContext(fields map[string]any) *zap.Logger {
	if log != nil {
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
				// 对于其他类型，尝试转换为字符串
				zapFields = append(zapFields, zap.Any(k, val))
			}
		}
		return log.With(zapFields...)
	}
	return zap.NewNop()
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	if log != nil {
		log.Debug(msg, fields...)
	}
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	if log != nil {
		log.Info(msg, fields...)
	}
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	if log != nil {
		log.Warn(msg, fields...)
	}
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	if log != nil {
		log.Error(msg, fields...)
	}
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	if log != nil {
		log.Fatal(msg, fields...)
	}
}
