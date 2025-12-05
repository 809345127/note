package mysql

import (
	"fmt"
	"time"

	"ddd-example/infrastructure/persistence/mysql/po"
	"ddd-example/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config MySQL配置
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

// DSN 生成数据源名称
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// Connect 连接到MySQL并返回GORM实例
func (c *Config) Connect() (*gorm.DB, error) {
	// 配置GORM日志
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	db, err := gorm.Open(mysql.Open(c.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层sql.DB以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
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

// AutoMigrate 自动迁移数据库表结构
// 注意：仅在开发环境使用，生产环境应使用迁移工具
func AutoMigrate(db *gorm.DB) error {
	logger.Info().Msg("Running database auto migration...")

	err := db.AutoMigrate(
		&po.UserPO{},
		&po.OrderPO{},
		&po.OrderItemPO{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	logger.Info().Msg("Database migration completed")
	return nil
}
