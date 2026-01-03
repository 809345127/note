package api

import (
	"ddd/api/middleware"

	"github.com/gin-gonic/gin"
)

// Route represents a custom route
type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

// Router Route configuration
type Router struct {
	engine      *gin.Engine
	config      any
	controllers []ControllerRegister
	middlewares []MiddlewareRegister
	customRoutes []Route
}

// ControllerRegister is an interface for registering controllers
type ControllerRegister interface {
	RegisterRoutes(api *gin.RouterGroup)
}

// MiddlewareRegister is an interface for registering middleware
type MiddlewareRegister interface {
	Register(engine *gin.Engine)
}

// NewRouter Create route configuration
func NewRouter(
	cfg any,
	controllers []ControllerRegister,
	middlewares []MiddlewareRegister,
	customRoutes []Route,
) *Router {
	// Get config as interface{} to avoid import cycle
	config := cfg

	engine := gin.New()

	// Add middleware (order is important)
	engine.Use(middleware.RequestIDMiddleware()) // 1. Generate request ID first
	engine.Use(middleware.RecoveryMiddleware())  // 2. Recovery middleware
	engine.Use(middleware.LoggingMiddleware())   // 3. Logging middleware

	// Get CORS config dynamically
	// Note: In a real implementation, you'd pass the config properly
	engine.Use(middleware.CORSMiddleware(nil)) // 4. CORS (configurable)

	// Apply custom middleware
	for _, m := range middlewares {
		m.Register(engine)
	}

	return &Router{
		engine:      engine,
		config:      config,
		controllers: controllers,
		middlewares: middlewares,
		customRoutes: customRoutes,
	}
}

// SetupRoutes Set up all routes
func (r *Router) SetupRoutes() {
	// Set API route group
	apiGroup := r.engine.Group("/api/v1")
	{
		// Register all controllers dynamically
		for _, c := range r.controllers {
			c.RegisterRoutes(apiGroup)
		}

		// Register custom routes
		for _, route := range r.customRoutes {
			apiGroup.Handle(route.Method, route.Path, route.Handler)
		}
	}

	// Set root path route
	r.engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "DDD Application",
			"version": "1.0.0",
			"docs":    "/api/v1/docs",
			"health":  "/api/v1/health",
		})
	})
}

// GetEngine Get Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
