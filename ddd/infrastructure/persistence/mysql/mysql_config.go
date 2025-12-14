package mysql

import (
	"fmt"
	"time"

	"ddd/infrastructure/persistence/mysql/po"
	"ddd/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config MySQL configuration
type Config struct {
	Host            string
	Port            string
	Username        string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN Generate data source name
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// Connect Connect to MySQL and return GORM instance
func (c *Config) Connect() (*gorm.DB, error) {
	// Configure GORM logging
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
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

	// Set connection pool parameters
	if c.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(25)
	}

	if c.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(5)
	}

	if c.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}

	return db, nil
}

// AutoMigrate Auto migrate database schema
// Note: Only use in development environment, use migration tools in production
func AutoMigrate(db *gorm.DB) error {
	logger.Info().Msg("Running database auto migration...")

	err := db.AutoMigrate(
		&po.UserPO{},
		&po.OrderPO{},
		&po.OrderItemPO{},
		&po.OutboxEventPO{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	logger.Info().Msg("Database migration completed")
	return nil
}
