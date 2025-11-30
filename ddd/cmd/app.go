package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ddd-example/api"
	"ddd-example/domain"
	"ddd-example/infrastructure/persistence/mocks"
	"ddd-example/infrastructure/persistence/mysql"
	"ddd-example/service"

	"github.com/gin-gonic/gin"
)

// App 应用程序结构体
type App struct {
	router *api.Router
	server *gin.Engine
}

// NewApp 创建应用程序
func NewApp() *App {
	// 根据环境变量选择仓储实现
	dbType := os.Getenv("DB_TYPE")

	var userRepo domain.UserRepository
	var orderRepo domain.OrderRepository

	// 注意：事件发布器现在只用于事件订阅/处理，不再由仓储使用
	// 仓储不直接发布事件，事件由 UoW 保存到 outbox 表
	// 后台 OutboxProcessor 异步发布到消息队列
	eventPublisher := mocks.NewMockEventPublisher()

	if dbType == "mysql" {
		// 使用MySQL实现
		fmt.Println("🗄️  Using MySQL persistence layer...")
		config := mysql.NewConfig()
		config.Port = "3307" // 使用Docker MySQL的端口

		db, err := config.Connect()
		if err != nil {
			log.Fatalf("❌ Failed to connect to MySQL: %v", err)
		}

		// 测试数据库连接
		if err := db.Ping(); err != nil {
			log.Fatalf("❌ Failed to ping MySQL: %v", err)
		}

		fmt.Println("✅ Connected to MySQL successfully")

		// 仓储只负责持久化，不负责发布事件
		userRepo = mysql.NewUserRepository(db)
		orderRepo = mysql.NewOrderRepository(db)
	} else {
		// 使用Mock实现（默认）
		fmt.Println("💾  Using Mock persistence layer...")
		// 仓储只负责持久化，不负责发布事件
		userRepo = mocks.NewMockUserRepository()
		orderRepo = mocks.NewMockOrderRepository()
	}

	// 创建应用服务
	// 注意：eventPublisher 现在主要用于订阅事件，实际发布由 OutboxProcessor 完成
	userService := service.NewUserApplicationService(userRepo, orderRepo, eventPublisher)
	orderService := service.NewOrderApplicationService(orderRepo, userRepo, eventPublisher)

	// 创建控制器
	healthController := api.NewHealthController()
	userController := api.NewUserController(userService)
	orderController := api.NewOrderController(orderService)

	// 创建路由
	router := api.NewRouter(healthController, userController, orderController)
	router.SetupRoutes()

	return &App{
		router: router,
		server: router.GetEngine(),
	}
}

// Run 运行应用程序
func (a *App) Run(port string) {
	// 设置优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nShutting down server...")

		// 这里可以添加清理逻辑
		fmt.Println("Server stopped")
		os.Exit(0)
	}()

	fmt.Printf("Server starting on port %s...\n", port)
	fmt.Printf("API Documentation: http://localhost:%s/api/v1/docs\n", port)
	fmt.Printf("Health Check: http://localhost:%s/api/v1/health\n", port)

	if err := a.server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// GetServer 获取服务器实例（用于测试）
func (a *App) GetServer() *gin.Engine {
	return a.server
}
