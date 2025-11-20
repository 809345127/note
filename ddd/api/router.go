package api

import (
	"github.com/gin-gonic/gin"
)

// Router 路由配置
type Router struct {
	engine           *gin.Engine
	healthController *HealthController
	userController   *UserController
	orderController  *OrderController
}

// NewRouter 创建路由配置
func NewRouter(
	healthController *HealthController,
	userController *UserController,
	orderController *OrderController,
) *Router {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	
	engine := gin.New()
	
	// 添加中间件
	engine.Use(LoggingMiddleware())
	engine.Use(RecoveryMiddleware())
	engine.Use(CORSMiddleware())
	
	return &Router{
		engine:           engine,
		healthController: healthController,
		userController:   userController,
		orderController:  orderController,
	}
}

// SetupRoutes 设置所有路由
func (r *Router) SetupRoutes() {
	// 设置API路由组
	apiGroup := r.engine.Group("/api/v1")
	{
		// 注册控制器路由
		r.healthController.RegisterRoutes(apiGroup)
		r.userController.RegisterRoutes(apiGroup)
		r.orderController.RegisterRoutes(apiGroup)
	}
	
	// 设置根路径路由
	r.engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to DDD Example API",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
		})
	})
}

// GetEngine 获取Gin引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}