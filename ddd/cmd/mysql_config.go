package cmd

import (
	"ddd/config"
	"ddd/infrastructure/persistence/mysql"
)

func NewMySQLConfig(cfg *config.Config) *mysql.Config {
	return &mysql.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Username:        cfg.Database.Username,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		LogLevel:        cfg.Database.LogLevel,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}
}
