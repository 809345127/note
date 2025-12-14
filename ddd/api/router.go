package api

import (
	"ddd/api/health"
	"ddd/api/middleware"
	"ddd/api/order"
	"ddd/api/user"
	"ddd/config"

	"github.com/gin-gonic/gin"
)

// Router Route configuration
type Router struct {
	engine           *gin.Engine
	config           *config.Config
	healthController *health.Controller
	userController   *user.Controller
	orderController  *order.Controller
}

// NewRouter Create route configuration
func NewRouter(
	cfg *config.Config,
	healthController *health.Controller,
	userController *user.Controller,
	orderController *order.Controller,
) *Router {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Add middleware (order is important)
	engine.Use(middleware.RequestIDMiddleware())                      // 1. Generate request ID first
	engine.Use(middleware.RecoveryMiddleware())                       // 2. Recovery middleware
	engine.Use(middleware.LoggingMiddleware())                        // 3. Logging middleware
	engine.Use(middleware.CORSMiddleware(&cfg.CORS))                  // 4. CORS
	engine.Use(middleware.RateLimitMiddleware(&cfg.Server.RateLimit)) // 5. Rate limiting

	return &Router{
		engine:           engine,
		config:           cfg,
		healthController: healthController,
		userController:   userController,
		orderController:  orderController,
	}
}

// SetupRoutes Set up all routes
func (r *Router) SetupRoutes() {
	// Set API route group
	apiGroup := r.engine.Group("/api/v1")
	{
		// Register controller routes
		r.healthController.RegisterRoutes(apiGroup)
		r.userController.RegisterRoutes(apiGroup)
		r.orderController.RegisterRoutes(apiGroup)
	}

	// Set root path route
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

// GetEngine Get Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
