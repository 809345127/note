package mysql

import (
	"context"
	"fmt"
	"time"

	"ddd/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	DefaultMaxOpenConns    = 25
	DefaultMaxIdleConns    = 10
	DefaultConnMaxLifetime = 10 * time.Minute
	DefaultConnMaxIdleTime = 5 * time.Minute
)

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
	LogLevel        string        `mapstructure:"log_level" json:"log_level"`
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci&readTimeout=10s&writeTimeout=10s",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}
func (c *Config) parseLogLevel() gormlogger.LogLevel {
	switch c.LogLevel {
	case "debug":
		return gormlogger.Info
	case "info":
		return gormlogger.Info
	case "warn":
		return gormlogger.Warn
	case "error":
		return gormlogger.Error
	case "silent":
		return gormlogger.Silent
	default:
		return gormlogger.Warn
	}
}
func (c *Config) applyDefaults() {
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = DefaultMaxOpenConns
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
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
func (c *Config) Connect() (*gorm.DB, error) {
	c.applyDefaults()
	gormConfig := &gorm.Config{
		Logger: logger.NewGormLoggerAdapter(c.parseLogLevel()),
	}

	db, err := gorm.Open(mysql.Open(c.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
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
