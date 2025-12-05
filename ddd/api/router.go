package api

import (
	"ddd-example/config"

	"github.com/gin-gonic/gin"
)

// Router 路由配置
type Router struct {
	engine           *gin.Engine
	config           *config.Config
	healthController *HealthController
	userController   *UserController
	orderController  *OrderController
}

// NewRouter 创建路由配置
func NewRouter(
	cfg *config.Config,
	healthController *HealthController,
	userController *UserController,
	orderController *OrderController,
) *Router {
	// 根据环境设置Gin模式
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 添加中间件（顺序很重要）
	engine.Use(RequestIDMiddleware())        // 1. 首先生成请求ID
	engine.Use(RecoveryMiddleware())         // 2. 恢复中间件
	engine.Use(LoggingMiddleware())          // 3. 日志中间件
	engine.Use(CORSMiddleware(&cfg.CORS))    // 4. CORS
	engine.Use(RateLimitMiddleware(&cfg.Server.RateLimit)) // 5. 限流

	return &Router{
		engine:           engine,
		config:           cfg,
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
			"name":    r.config.App.Name,
			"version": r.config.App.Version,
			"env":     r.config.App.Env,
			"docs":    "/api/v1/docs",
			"health":  "/api/v1/health",
		})
	})
}

// GetEngine 获取Gin引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
