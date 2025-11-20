package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return "[DDD-Example] " +
			param.TimeStamp.Format(time.RFC3339) + " | " +
			param.Method + " " +
			param.Path + " " +
			param.Request.Proto + " " +
			strconv.Itoa(param.StatusCode) + " " +
			param.Latency.String() + " " +
			param.ClientIP + " " +
			param.ErrorMessage +
			"\n"
	})
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err,
				"message": "Internal server error",
				"code":    500,
			})
		}
		c.AbortWithStatus(500)
	})
}