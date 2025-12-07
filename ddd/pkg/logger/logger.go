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

// Init Initialize logger
func Init(cfg *config.LogConfig) error {
	// Set error stack
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339

	// Set log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Set output
	var output io.Writer
	switch cfg.Output {
	case "file":
		// Ensure log directory exists
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

	// Set format
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

// parseLevel Parse log level
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

// Get Get logger instance
func Get() *zerolog.Logger {
	return &log
}

// Debug Debug log
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info Info log
func Info() *zerolog.Event {
	return log.Info()
}

// Warn Warn log
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error Error log
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal Fatal error log
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// WithRequestID Add request ID
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithField Add field
func WithField(key string, value interface{}) zerolog.Logger {
	return log.With().Interface(key, value).Logger()
}
