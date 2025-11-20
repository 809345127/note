package api

import (
	"github.com/gin-gonic/gin"
)

// HealthController 健康检查控制器
type HealthController struct{}

// NewHealthController 创建健康检查控制器
func NewHealthController() *HealthController {
	return &HealthController{}
}

// RegisterRoutes 注册健康检查路由
func (c *HealthController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/health", c.HealthCheck)
}

// HealthCheck 健康检查
func (c *HealthController) HealthCheck(ctx *gin.Context) {
	HandleSuccess(ctx, gin.H{
		"status": "healthy",
		"service": "ddd-example",
		"version": "1.0.0",
	}, "Service is healthy")
}