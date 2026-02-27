package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Log      LogConfig      `mapstructure:"log"`
	CORS     CORSConfig     `mapstructure:"cors"`
}
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"`
}
type ServerConfig struct {
	Port            string          `mapstructure:"port"`
	ReadTimeout     time.Duration   `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration   `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration   `mapstructure:"shutdown_timeout"`
	TrustedProxies  []string        `mapstructure:"trusted_proxies"`
	RateLimit       RateLimitConfig `mapstructure:"rate_limit"`
}
type RateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	Rate    float64 `mapstructure:"rate"`
	Burst   int     `mapstructure:"burst"`
}
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	LogLevel        string        `mapstructure:"log_level"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Retry           RetryConfig   `mapstructure:"retry"`
}
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
type WorkerConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	BatchSize    int           `mapstructure:"batch_size"`
	MaxRetries   int           `mapstructure:"max_retries"`
}
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}
type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "dev" || c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "prod" || c.App.Env == "production"
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)
	configureConfigSource(v, configPath)
	configureEnvBinding(v)
	if err := readConfigFileIfPresent(v); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func configureConfigSource(v *viper.Viper, configPath string) {
	if configPath != "" {
		v.SetConfigFile(configPath)
		return
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
}

func configureEnvBinding(v *viper.Viper) {
	v.SetEnvPrefix("DDD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
}

func readConfigFileIfPresent(v *viper.Viper) error {
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	setAppDefaults(v)
	setServerDefaults(v)
	setDatabaseDefaults(v)
	setWorkerDefaults(v)
	setLogDefaults(v)
	setCORSDefaults(v)
}

func setAppDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "ddd")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "dev")
}

func setServerDefaults(v *viper.Viper) {
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.trusted_proxies", []string{})
	v.SetDefault("server.rate_limit.enabled", true)
	v.SetDefault("server.rate_limit.rate", 100)
	v.SetDefault("server.rate_limit.burst", 200)
}

func setDatabaseDefaults(v *viper.Viper) {
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "3306")
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "ddd_example")
	v.SetDefault("database.log_level", "warn")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.retry.enabled", true)
	v.SetDefault("database.retry.max_attempts", 3)
	v.SetDefault("database.retry.initial_delay", "100ms")
	v.SetDefault("database.retry.max_delay", "2s")
	v.SetDefault("database.retry.backoff_factor", 2.0)
	v.SetDefault("database.retry.jitter_enabled", true)
	v.SetDefault("database.retry.retry_on_concurrent_modification", true)
	v.SetDefault("database.retry.retry_on_deadlock", true)
	v.SetDefault("database.retry.retry_on_lock_timeout", true)
}

func setWorkerDefaults(v *viper.Viper) {
	v.SetDefault("worker.enabled", false)
	v.SetDefault("worker.poll_interval", "3s")
	v.SetDefault("worker.batch_size", 100)
	v.SetDefault("worker.max_retries", 5)
}

func setLogDefaults(v *viper.Viper) {
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.file_path", "logs/app.log")
}

func setCORSDefaults(v *viper.Viper) {
	v.SetDefault("cors.allow_origins", []string{"http://localhost:3000"})
	v.SetDefault("cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allow_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", 86400)
}
