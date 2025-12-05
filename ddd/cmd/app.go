package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ddd-example/api"
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

// App 应用程序结构体
type App struct {
	config *config.Config
	router *api.Router
	server *http.Server
	db     *gorm.DB
}

// NewApp 创建应用程序
func NewApp(cfg *config.Config) *App {
	// 初始化日志
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

	// 事件发布器（用于事件订阅/处理）
	eventPublisher := mocks.NewMockEventPublisher()

	// 根据配置选择仓储实现
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

		// 测试连接
		sqlDB, err := db.DB()
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to get underlying sql.DB")
		}
		if err := sqlDB.Ping(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to ping MySQL")
		}

		logger.Info().Msg("Connected to MySQL successfully")

		// 开发环境自动迁移
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

	// 创建应用服务
	userService := userapp.NewApplicationService(userRepo, orderRepo, eventPublisher)
	orderService := orderapp.NewApplicationService(orderRepo, userRepo, eventPublisher)

	// 创建控制器（健康检查需要传入sql.DB用于检查连接）
	var sqlDB interface{}
	if db != nil {
		sqlDB, _ = db.DB()
	}
	healthController := api.NewHealthController(cfg, sqlDB)
	userController := api.NewUserController(userService)
	orderController := api.NewOrderController(orderService)

	// 创建路由
	router := api.NewRouter(cfg, healthController, userController, orderController)
	router.SetupRoutes()

	// 创建HTTP服务器
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

// Run 运行应用程序
func (a *App) Run() error {
	// 启动服务器
	go func() {
		logger.Info().
			Str("port", a.config.Server.Port).
			Str("health", "http://localhost:"+a.config.Server.Port+"/api/v1/health").
			Msg("Server started")

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
		return err
	}

	// 关闭数据库连接
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

// GetServer 获取服务器实例（用于测试）
func (a *App) GetServer() *http.Server {
	return a.server
}
