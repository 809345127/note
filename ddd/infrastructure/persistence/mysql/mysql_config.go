package mysql

import (
	"context"
	"fmt"
	"time"

	"ddd/infrastructure/persistence/mysql/po"
	"ddd/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Default connection pool settings based on GORM best practices
const (
	DefaultMaxOpenConns    = 25
	DefaultMaxIdleConns    = 10
	DefaultConnMaxLifetime = 10 * time.Minute
	DefaultConnMaxIdleTime = 5 * time.Minute
)

// Config MySQL configuration
type Config struct {
	Host            string        `mapstructure:"host" json:"host"`
	Port            string        `mapstructure:"port" json:"port"`
	Username        string        `mapstructure:"username" json:"username"`
	Password        string        `mapstructure:"password" json:"password"`
	Database        string        `mapstructure:"database" json:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"`
	LogLevel        string        `mapstructure:"log_level" json:"log_level"` // debug, info, warn, error
}

// DSN Generate data source name with optimized settings for MySQL 8+
func (c *Config) DSN() string {
	// Using standard MySQL 8+ compatible DSN with proper charset and collation
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci&readTimeout=10s&writeTimeout=10s&allowPublicKeyRetrieval=true",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// parseLogLevel converts string log level to GORM logger level
func (c *Config) parseLogLevel() gormlogger.LogLevel {
	switch c.LogLevel {
	case "debug":
		return gormlogger.Info // GORM uses Info level for debug logging
	case "warn":
		return gormlogger.Warn
	case "error":
		return gormlogger.Error
	case "silent":
		return gormlogger.Silent
	default:
		return gormlogger.Info
	}
}

// applyDefaults sets default values for connection pool settings
func (c *Config) applyDefaults() {
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = DefaultMaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	// Ensure MaxIdleConns doesn't exceed MaxOpenConns
	if c.MaxIdleConns > c.MaxOpenConns {
		c.MaxIdleConns = c.MaxOpenConns
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = DefaultConnMaxLifetime
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = DefaultConnMaxIdleTime
	}
}

// Connect establishes connection to MySQL with optimized pool settings
func (c *Config) Connect() (*gorm.DB, error) {
	c.applyDefaults()

	// Configure GORM with custom logger and optimized settings
	gormConfig := &gorm.Config{
		Logger: logger.NewGormLoggerAdapter(c.parseLogLevel()),
	}

	db, err := gorm.Open(mysql.Open(c.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool - GORM best practices
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(c.ConnMaxIdleTime)

	logger.Info("Database connected",
		zap.String("host", c.Host),
		zap.String("database", c.Database),
		zap.Int("max_open_conns", c.MaxOpenConns),
		zap.Int("max_idle_conns", c.MaxIdleConns),
		zap.Duration("conn_max_lifetime", c.ConnMaxLifetime),
	)

	return db, nil
}

// Ping verifies database connection is still alive
func (c *Config) Ping(ctx context.Context) error {
	db, err := c.Connect()
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// AutoMigrate Auto migrate database schema
// Note: Only use in development environment, use migration tools in production
func AutoMigrate(db *gorm.DB) error {
	logger.Info("Running database auto migration...")

	err := db.AutoMigrate(
		&po.UserPO{},
		&po.OrderPO{},
		&po.OrderItemPO{},
		&po.OutboxEventPO{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	logger.Info("Database migration completed")
	return nil
}
