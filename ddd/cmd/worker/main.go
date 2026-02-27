package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ddd/cmd"
	"ddd/config"
	"ddd/infrastructure/persistence/mysql"
	"ddd/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Worker startup failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := parseConfigPath()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := logger.Init(&cfg.Log, cfg.App.Env); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	if !cfg.Worker.Enabled {
		logger.Info("Outbox worker is disabled by config; exiting")
		return nil
	}

	db, err := cmd.NewMySQLConfig(cfg).Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	worker, err := mysql.NewOutboxWorker(
		mysql.NewOutboxRepository(db),
		&mysql.LoggingOutboxPublisher{},
		cfg.Worker.PollInterval,
		cfg.Worker.BatchSize,
		cfg.Worker.MaxRetries,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox worker: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("Outbox worker started",
		zap.Duration("poll_interval", cfg.Worker.PollInterval),
		zap.Int("batch_size", cfg.Worker.BatchSize),
		zap.Int("max_retries", cfg.Worker.MaxRetries),
	)

	if err := worker.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("outbox worker exited with error: %w", err)
	}

	logger.Info("Outbox worker stopped")
	return nil
}

func parseConfigPath() string {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()
	return configPath
}
