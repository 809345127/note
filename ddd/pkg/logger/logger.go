package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"ddd-example/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var log zerolog.Logger

// Init 初始化日志
func Init(cfg *config.LogConfig) error {
	// 设置错误堆栈
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339

	// 设置日志级别
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// 设置输出
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

// WithRequestID 添加请求ID
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithField 添加字段
func WithField(key string, value interface{}) zerolog.Logger {
	return log.With().Interface(key, value).Logger()
}
