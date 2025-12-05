package api

import (
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"ddd-example/config"

	"github.com/gin-gonic/gin"
)

// HealthController 健康检查控制器
type HealthController struct {
	config    *config.Config
	db        *sql.DB
	startTime time.Time
}

// NewHealthController 创建健康检查控制器
// db 参数接受 *sql.DB 或 nil
func NewHealthController(cfg *config.Config, db interface{}) *HealthController {
	var sqlDB *sql.DB
	if db != nil {
		if d, ok := db.(*sql.DB); ok {
			sqlDB = d
		}
	}
	return &HealthController{
		config:    cfg,
		db:        sqlDB,
		startTime: time.Now(),
	}
}

// RegisterRoutes 注册健康检查路由
func (c *HealthController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/health", c.Health)
	router.GET("/health/live", c.Liveness)
	router.GET("/health/ready", c.Readiness)
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string           `json:"status"`
	Version   string           `json:"version"`
	Uptime    string           `json:"uptime"`
	Timestamp string           `json:"timestamp"`
	Checks    map[string]Check `json:"checks,omitempty"`
	System    *SystemInfo      `json:"system,omitempty"`
}

// Check 检查项
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAlloc     uint64 `json:"mem_alloc_bytes"`
}

// Health 完整健康检查
func (c *HealthController) Health(ctx *gin.Context) {
	checks := make(map[string]Check)
	overallStatus := "healthy"

	// 检查数据库连接
	if c.db != nil {
		dbCheck := c.checkDatabase()
		checks["database"] = dbCheck
		if dbCheck.Status != "healthy" {
			overallStatus = "unhealthy"
		}
	}

	// 获取系统信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response := HealthResponse{
		Status:    overallStatus,
		Version:   c.config.App.Version,
		Uptime:    time.Since(c.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
		System: &SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemAlloc:     memStats.Alloc,
		},
	}

	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	ctx.JSON(statusCode, response)
}

// Liveness 存活检查（Kubernetes liveness probe）
func (c *HealthController) Liveness(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Readiness 就绪检查（Kubernetes readiness probe）
func (c *HealthController) Readiness(ctx *gin.Context) {
	// 检查数据库连接
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

// checkDatabase 检查数据库连接
func (c *HealthController) checkDatabase() Check {
	if c.db == nil {
		return Check{
			Status:  "healthy",
			Message: "using mock database",
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
