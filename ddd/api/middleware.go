package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"ddd-example/config"
	"ddd-example/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const (
	// RequestIDHeader 请求ID头
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey 请求ID上下文键
	RequestIDKey = "request_id"
)

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 获取请求ID
		requestID, _ := c.Get(RequestIDKey)
		reqID, _ := requestID.(string)

		// 创建带请求ID的日志
		log := logger.WithRequestID(reqID)

		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 记录日志
		event := log.Info()
		if c.Writer.Status() >= 400 {
			event = log.Warn()
		}
		if c.Writer.Status() >= 500 {
			event = log.Error()
		}

		if raw != "" {
			path = path + "?" + raw
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Int("body_size", c.Writer.Size()).
			Msg("HTTP Request")
	}
}

// RecoveryMiddleware 恢复中间件（修复bug版本）
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get(RequestIDKey)
				reqID, _ := requestID.(string)

				// 记录panic日志
				logger.Error().
					Str("request_id", reqID).
					Interface("error", recovered).
					Str("path", c.Request.URL.Path).
					Msg("Panic recovered")

				// 返回500错误（只调用一次响应方法）
				c.AbortWithStatusJSON(http.StatusInternalServerError, Response{
					Success:   false,
					Error:     "internal server error",
					Message:   "An unexpected error occurred",
					Code:      http.StatusInternalServerError,
					RequestID: reqID,
				})
			}
		}()

		c.Next()
	}
}

// CORSMiddleware CORS中间件（可配置版本）
func CORSMiddleware(cfg *config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查是否允许的来源
		allowed := false
		for _, o := range cfg.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 构建允许的方法和头
		methods := ""
		for i, m := range cfg.AllowMethods {
			if i > 0 {
				methods += ", "
			}
			methods += m
		}

		headers := ""
		for i, h := range cfg.AllowHeaders {
			if i > 0 {
				headers += ", "
			}
			headers += h
		}

		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Max-Age", string(rune(cfg.MaxAge)))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiter 限流器
type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

// NewRateLimiter 创建限流器
func NewRateLimiter(r float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.Limit(r),
		burst: burst,
	}
}

// getLimiter 获取或创建IP的限流器
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	if limiter, ok := rl.limiters.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters.Store(ip, limiter)
	return limiter
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(cfg *config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := NewRateLimiter(cfg.Rate, cfg.Burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.getLimiter(ip)

		if !l.Allow() {
			requestID, _ := c.Get(RequestIDKey)
			reqID, _ := requestID.(string)

			logger.Warn().
				Str("request_id", reqID).
				Str("client_ip", ip).
				Msg("Rate limit exceeded")

			c.AbortWithStatusJSON(http.StatusTooManyRequests, Response{
				Success:   false,
				Error:     "rate_limit_exceeded",
				Message:   "Too many requests, please try again later",
				Code:      http.StatusTooManyRequests,
				RequestID: reqID,
			})
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// 使用channel监控超时
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			requestID, _ := c.Get(RequestIDKey)
			reqID, _ := requestID.(string)

			logger.Warn().
				Str("request_id", reqID).
				Str("path", c.Request.URL.Path).
				Msg("Request timeout")

			c.AbortWithStatusJSON(http.StatusGatewayTimeout, Response{
				Success:   false,
				Error:     "request_timeout",
				Message:   "Request timeout",
				Code:      http.StatusGatewayTimeout,
				RequestID: reqID,
			})
		}
	}
}

// GinLogger 返回用于Gin的zerolog适配器
func GinLogger(log *zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
