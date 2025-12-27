/*
Package logger - 日志包

设计原则:
1. 使用 zerolog 作为日志库（高性能、结构化）
2. 支持多种输出格式（JSON/Console）
3. 支持文件和标准输出
4. 提供 RequestID 支持，便于请求追踪

使用示例:

	// 基础使用
	logger.Info().Str("user_id", "123").Msg("user login")

	// 错误日志（推荐在 API 层使用 response.HandleAppError，它会自动记录）
	logger.Error().Err(err).Str("order_id", id).Msg("failed to get order")

	// 带 RequestID
	reqLogger := logger.WithRequestID(requestID)
	reqLogger.Info().Msg("processing request")
*/
package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"ddd/config"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init 初始化日志
func Init(cfg *config.LogConfig) error {
	// 设置时间格式
	zerolog.TimeFieldFormat = time.RFC3339

	// 设置日志级别
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// 设置输出目标
	var output io.Writer
	switch cfg.Output {
	case "file":
		// 确保日志目录存在
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		output = file
	default:
		output = os.Stdout
	}

	// 设置格式
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    false,
		}
	}

	log = zerolog.New(output).With().Timestamp().Caller().Logger()

	return nil
}

// parseLevel 解析日志级别
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Get 获取日志实例
func Get() *zerolog.Logger {
	return &log
}

// Debug 调试日志
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info 信息日志
func Info() *zerolog.Event {
	return log.Info()
}

// Warn 警告日志
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error 错误日志
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal 致命错误日志
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// WithRequestID 创建带请求 ID 的日志器
// 用于在整个请求处理过程中保持请求 ID
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithContext 创建带多个上下文字段的日志器
func WithContext(fields map[string]string) zerolog.Logger {
	ctx := log.With()
	for k, v := range fields {
		ctx = ctx.Str(k, v)
	}
	return ctx.Logger()
}
