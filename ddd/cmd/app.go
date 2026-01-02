package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ddd/api"
	healthapi "ddd/api/health"
	orderapi "ddd/api/order"
	userapi "ddd/api/user"
	orderapp "ddd/application/order"
	userapp "ddd/application/user"
	"ddd/config"
	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
	"ddd/infrastructure/persistence/mysql"
	"ddd/pkg/logger"

	"go.uber.org/zap"
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
	if err := logger.Init(&cfg.Log, cfg.App.Env); err != nil {
		// 使用默认 logger 记录初始化失败
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting application",
		zap.String("app", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env))

	var userRepo user.Repository
	var orderRepo order.Repository
	var uow shared.UnitOfWork
	var db *gorm.DB

	// Use MySQL/GORM persistence layer
	logger.Info("Using MySQL/GORM persistence layer")

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
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get underlying sql.DB", zap.Error(err))
	}
	if err := sqlDB.Ping(); err != nil {
		logger.Fatal("Failed to ping MySQL", zap.Error(err))
	}

	logger.Info("Connected to MySQL successfully")

	// Auto migration in development environment
	if cfg.IsDevelopment() {
		if err := mysql.AutoMigrate(db); err != nil {
			logger.Fatal("Failed to auto migrate", zap.Error(err))
		}
	}

	userRepo = mysql.NewUserRepository(db)
	orderRepo = mysql.NewOrderRepository(db)
	uow = mysql.NewUnitOfWork(db)

	// Create application services with UoW for transaction management
	userService := userapp.NewApplicationService(userRepo, orderRepo, uow)
	orderService := orderapp.NewApplicationService(orderRepo, userRepo, uow)

	// Create controllers (health check needs sql.DB for connection check)
	var healthDB interface{}
	if db != nil {
		healthDB, _ = db.DB()
	}
	healthController := healthapi.NewController(cfg, healthDB)
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
		logger.Info("Server started",
			zap.String("port", a.config.Server.Port),
			zap.String("health", "http://localhost:"+a.config.Server.Port+"/api/v1/health"))

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	// Close database connection
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Error("Error closing database connection", zap.Error(err))
			}
		}
	}

	logger.Info("Server exited properly")
	return nil
}

// GetServer Get server instance (for testing)
func (a *App) GetServer() *http.Server {
	return a.server
}
