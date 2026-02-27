package api

import (
	"ddd/api/middleware"
	"ddd/config"

	"github.com/gin-gonic/gin"
)

type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

type Router struct {
	cfg          *config.Config
	engine       *gin.Engine
	controllers  []ControllerRegister
	customRoutes []Route
}

type ControllerRegister interface {
	RegisterRoutes(api *gin.RouterGroup)
}

type MiddlewareRegister interface {
	Register(engine *gin.Engine)
}

func NewRouter(
	cfg *config.Config,
	controllers []ControllerRegister,
	middlewares []MiddlewareRegister,
	customRoutes []Route,
) *Router {
	engine := gin.New()

	_ = engine.SetTrustedProxies(nil)
	if len(cfg.Server.TrustedProxies) > 0 {
		_ = engine.SetTrustedProxies(cfg.Server.TrustedProxies)
	}

	engine.Use(middleware.RequestIDMiddleware())
	engine.Use(middleware.RecoveryMiddleware())
	engine.Use(middleware.LoggingMiddleware())
	engine.Use(middleware.MaxBodySizeMiddleware(middleware.DefaultMaxBodySize))

	for _, m := range middlewares {
		m.Register(engine)
	}

	engine.Use(middleware.RateLimitMiddleware(&cfg.Server.RateLimit))
	engine.Use(middleware.CORSMiddleware(&cfg.CORS))

	return &Router{
		cfg:          cfg,
		engine:       engine,
		controllers:  controllers,
		customRoutes: customRoutes,
	}
}

func (r *Router) SetupRoutes() {
	apiGroup := r.engine.Group("/api/v1")
	for _, c := range r.controllers {
		c.RegisterRoutes(apiGroup)
	}
	for _, route := range r.customRoutes {
		apiGroup.Handle(route.Method, route.Path, route.Handler)
	}

	r.engine.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    r.cfg.App.Name,
			"version": r.cfg.App.Version,
			"docs":    "/api/v1/docs",
			"health":  "/api/v1/health",
		})
	})
}

func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
