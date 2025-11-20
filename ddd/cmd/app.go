package cmd

import (
	"ddd-example/api"
	"ddd-example/mock"
	"ddd-example/service"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

// App 应用程序结构体
type App struct {
	router *api.Router
	server *gin.Engine
}

// NewApp 创建应用程序
func NewApp() *App {
	// 创建Mock仓储
	userRepo := mock.NewMockUserRepository()
	orderRepo := mock.NewMockOrderRepository()
	eventPublisher := mock.NewMockEventPublisher()
	
	// 创建应用服务
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