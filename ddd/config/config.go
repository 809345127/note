package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config Application Configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

// AppConfig Application Configuration
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"` // development, staging, production
}

// ServerConfig Server Configuration
type ServerConfig struct {
	Port            string          `mapstructure:"port"`
	ReadTimeout     time.Duration   `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration   `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration   `mapstructure:"shutdown_timeout"`
	RateLimit       RateLimitConfig `mapstructure:"rate_limit"`
}

// RateLimitConfig Rate Limiting Configuration
type RateLimitConfig struct {
	Enabled  bool    `mapstructure:"enabled"`
	Rate     float64 `mapstructure:"rate"`      // Requests per second
	Burst    int     `mapstructure:"burst"`     // Burst capacity
}

// DatabaseConfig Database Configuration
type DatabaseConfig struct {
	Type            string        `mapstructure:"type"` // mysql, mock
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Retry           RetryConfig   `mapstructure:"retry"`
}

// RetryConfig Retry configuration for optimistic concurrency control
type RetryConfig struct {
	Enabled                       bool          `mapstructure:"enabled"`
	MaxAttempts                   int           `mapstructure:"max_attempts"`
	InitialDelay                  time.Duration `mapstructure:"initial_delay"`
	MaxDelay                      time.Duration `mapstructure:"max_delay"`
	BackoffFactor                 float64       `mapstructure:"backoff_factor"`
	JitterEnabled                 bool          `mapstructure:"jitter_enabled"`
	RetryOnConcurrentModification bool          `mapstructure:"retry_on_concurrent_modification"`
	RetryOnDeadlock               bool          `mapstructure:"retry_on_deadlock"`
	RetryOnLockTimeout            bool          `mapstructure:"retry_on_lock_timeout"`
}

// LogConfig Log Configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, console
	Output     string `mapstructure:"output"`      // stdout, file
	FilePath   string `mapstructure:"file_path"`
}

// CORSConfig CORS Configuration
type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// IsDevelopment Whether it's development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction Whether it's production environment
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// Load Load Configuration
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Configuration file settings
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// Read environment variables
	v.SetEnvPrefix("DDD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Use default values when config file doesn't exist
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults Set default configuration
func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "ddd")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "development")

	// Server
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.rate_limit.enabled", true)
	v.SetDefault("server.rate_limit.rate", 100)
	v.SetDefault("server.rate_limit.burst", 200)

	// Database
	v.SetDefault("database.type", "mock")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "3306")
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "ddd_example")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	// Retry configuration defaults
	v.SetDefault("database.retry.enabled", true)
	v.SetDefault("database.retry.max_attempts", 3)
	v.SetDefault("database.retry.initial_delay", "100ms")
	v.SetDefault("database.retry.max_delay", "2s")
	v.SetDefault("database.retry.backoff_factor", 2.0)
	v.SetDefault("database.retry.jitter_enabled", true)
	v.SetDefault("database.retry.retry_on_concurrent_modification", true)
	v.SetDefault("database.retry.retry_on_deadlock", true)
	v.SetDefault("database.retry.retry_on_lock_timeout", true)

	// Log
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.file_path", "logs/app.log")

	// CORS
	v.SetDefault("cors.allow_origins", []string{"http://localhost:3000"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", 86400)
}
