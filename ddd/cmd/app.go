package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ddd-example/api"
	healthapi "ddd-example/api/health"
	orderapi "ddd-example/api/order"
	userapi "ddd-example/api/user"
	orderapp "ddd-example/application/order"
	userapp "ddd-example/application/user"
	"ddd-example/config"
	"ddd-example/domain/order"
	"ddd-example/domain/user"
	"ddd-example/infrastructure/persistence/mocks"
	"ddd-example/infrastructure/persistence/mysql"
	"ddd-example/pkg/logger"

	"gorm.io/gorm"
)

// App Application structure
type App struct {
	config *config.Config
	router *api.Router
	server *http.Server
	db     *gorm.DB
}

// NewApp Create application
func NewApp(cfg *config.Config) *App {
	// Initialize logger
	if err := logger.Init(&cfg.Log); err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	logger.Info().
		Str("app", cfg.App.Name).
		Str("version", cfg.App.Version).
		Str("env", cfg.App.Env).
		Msg("Starting application")

	var userRepo user.Repository
	var orderRepo order.Repository
	var db *gorm.DB

	// Event publisher (for event subscription/processing)
	eventPublisher := mocks.NewMockEventPublisher()

	// Select repository implementation based on configuration
	if cfg.Database.Type == "mysql" {
		logger.Info().Msg("Using MySQL/GORM persistence layer")

		mysqlConfig := &mysql.Config{
			Host:            cfg.Database.Host,
			Port:            cfg.Database.Port,
			Username:        cfg.Database.Username,
			Password:        cfg.Database.Password,
			Database:        cfg.Database.Database,
			MaxOpenConns:    cfg.Database.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		}

		var err error
		db, err = mysqlConfig.Connect()
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to connect to MySQL")
		}

		// Test connection
		sqlDB, err := db.DB()
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get underlying sql.DB")
		}
		if err := sqlDB.Ping(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to ping MySQL")
		}

		logger.Info().Msg("Connected to MySQL successfully")

		// Auto migration in development environment
		if cfg.IsDevelopment() {
			if err := mysql.AutoMigrate(db); err != nil {
				logger.Fatal().Err(err).Msg("Failed to auto migrate")
			}
		}

		userRepo = mysql.NewUserRepository(db)
		orderRepo = mysql.NewOrderRepository(db)
	} else {
		logger.Info().Msg("Using Mock persistence layer")
		userRepo = mocks.NewMockUserRepository()
		orderRepo = mocks.NewMockOrderRepository()
	}

	// Create application services
	userService := userapp.NewApplicationService(userRepo, orderRepo, eventPublisher)
	orderService := orderapp.NewApplicationService(orderRepo, userRepo, eventPublisher)

	// Create controllers (health check needs sql.DB for connection check)
	var sqlDB interface{}
	if db != nil {
		sqlDB, _ = db.DB()
	}
	healthController := healthapi.NewController(cfg, sqlDB)
	userController := userapi.NewController(userService)
	orderController := orderapi.NewController(orderService)

	// Create router
	router := api.NewRouter(cfg, healthController, userController, orderController)
	router.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router.GetEngine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		config: cfg,
		router: router,
		server: server,
		db:     db,
	}
}

// Run runs the application
func (a *App) Run() error {
	// Start server
	go func() {
		logger.Info().
			Str("port", a.config.Server.Port).
			Str("health", "http://localhost:"+a.config.Server.Port+"/api/v1/health").
			Msg("Server started")

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
		return err
	}

	// Close database connection
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Error().Err(err).Msg("Error closing database connection")
			}
		}
	}

	logger.Info().Msg("Server exited properly")
	return nil
}

// GetServer Get server instance (for testing)
func (a *App) GetServer() *http.Server {
	return a.server
}
