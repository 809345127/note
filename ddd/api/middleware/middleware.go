package middleware

import (
	"net/http"
	"sync"
	"time"

	"ddd/api/response"
	"ddd/config"
	"ddd/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	// RequestIDHeader Request ID header
	RequestIDHeader = "X-Request-ID"
)

// RequestIDMiddleware Request ID middleware
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(response.RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// LoggingMiddleware Logging middleware
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get request ID
		requestID, _ := c.Get(response.RequestIDKey)
		reqID, _ := requestID.(string)

		// Create logger with request ID
		log := logger.WithRequestID(reqID)

		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Update path with query if needed
		if raw != "" {
			path = path + "?" + raw
		}

		// Prepare log fields
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			log.Error("HTTP Request", fields...)
		case c.Writer.Status() >= 400:
			log.Warn("HTTP Request", fields...)
		default:
			log.Info("HTTP Request", fields...)
		}
	}
}

// RecoveryMiddleware Recovery middleware (bug-fixed version)
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get(response.RequestIDKey)
				reqID, _ := requestID.(string)

				// Log panic
				logger.Error("Panic recovered",
					zap.String("request_id", reqID),
					zap.Any("error", recovered),
					zap.String("path", c.Request.URL.Path))

				// Return 500 error (call response method only once)
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
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

// CORSMiddleware CORS middleware (configurable version)
func CORSMiddleware(cfg *config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
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

		// Build allowed methods and headers
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

// RateLimiter Rate limiter
type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

// NewRateLimiter Create rate limiter
func NewRateLimiter(r float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.Limit(r),
		burst: burst,
	}
}

// getLimiter Get or create rate limiter for IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	if limiter, ok := rl.limiters.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters.Store(ip, limiter)
	return limiter
}

// RateLimitMiddleware Rate limiting middleware
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
			requestID, _ := c.Get(response.RequestIDKey)
			reqID, _ := requestID.(string)

			logger.Warn("Rate limit exceeded",
				zap.String("request_id", reqID),
				zap.String("client_ip", ip))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Response{
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
