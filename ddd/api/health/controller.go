package health

import (
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"ddd/config"

	"github.com/gin-gonic/gin"
)

// Controller Health check controller
type Controller struct {
	config    *config.Config
	db        *sql.DB
	startTime time.Time
}

// NewController Create health check controller
// db parameter accepts *sql.DB or nil
func NewController(cfg *config.Config, db interface{}) *Controller {
	var sqlDB *sql.DB
	if db != nil {
		if d, ok := db.(*sql.DB); ok {
			sqlDB = d
		}
	}
	return &Controller{
		config:    cfg,
		db:        sqlDB,
		startTime: time.Now(),
	}
}

// RegisterRoutes Register health check routes
func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/health", c.Health)
	router.GET("/health/live", c.Liveness)
	router.GET("/health/ready", c.Readiness)
}

// HealthResponse Health check response
type HealthResponse struct {
	Status    string           `json:"status"`
	Version   string           `json:"version"`
	Uptime    string           `json:"uptime"`
	Timestamp string           `json:"timestamp"`
	Checks    map[string]Check `json:"checks,omitempty"`
	System    *SystemInfo      `json:"system,omitempty"`
}

// Check Check item
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfo System information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAlloc     uint64 `json:"mem_alloc_bytes"`
}

// Health Complete health check
func (c *Controller) Health(ctx *gin.Context) {
	checks := make(map[string]Check)
	overallStatus := "healthy"

	// Check database connection
	if c.db != nil {
		dbCheck := c.checkDatabase()
		checks["database"] = dbCheck
		if dbCheck.Status != "healthy" {
			overallStatus = "unhealthy"
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Version:   c.config.App.Version,
		Uptime:    time.Since(c.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	// Only expose system info in development mode
	if c.config.IsDevelopment() {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		response.System = &SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemAlloc:     memStats.Alloc,
		}
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	ctx.JSON(statusCode, response)
}

// Liveness Liveness check (Kubernetes liveness probe)
func (c *Controller) Liveness(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Readiness Readiness check (Kubernetes readiness probe)
func (c *Controller) Readiness(ctx *gin.Context) {
	// Check database connection
	if c.db != nil {
		if err := c.db.Ping(); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not_ready",
				"message": "database not available",
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// checkDatabase Check database connection
func (c *Controller) checkDatabase() Check {
	if c.db == nil {
		return Check{
			Status:  "unhealthy",
			Message: "database connection not initialized",
		}
	}

	start := time.Now()
	err := c.db.Ping()
	latency := time.Since(start)

	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: err.Error(),
			Latency: latency.String(),
		}
	}

	return Check{
		Status:  "healthy",
		Latency: latency.String(),
	}
}
