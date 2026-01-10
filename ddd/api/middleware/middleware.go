package middleware

import (
	"net/http"
	"strconv"
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
	// DefaultMaxBodySize is the default maximum body size (1MB)
	DefaultMaxBodySize = 1 << 20 // 1MB
)

// MaxBodySizeMiddleware limits request body size to prevent DoS attacks
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

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
		c.Header("Vary", "Origin") // Required for proper CDN/proxy caching

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
		c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// limiterWithTime wraps a rate limiter with its creation time for cleanup
type limiterWithTime struct {
	limiter     *rate.Limiter
	createdAt   time.Time
}

// RateLimiter Rate limiter with automatic cleanup
type RateLimiter struct {
	limiters        sync.Map
	rate            rate.Limit
	burst           int
	lastCleanup     time.Time
	mu              sync.Mutex
	cleanupInterval time.Duration
	maxAge          time.Duration // Max age for a limiter before cleanup
	stopCh          chan struct{} // Channel to signal cleanup goroutine to stop
}

// NewRateLimiter Create rate limiter with automatic cleanup
func NewRateLimiter(r float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:            rate.Limit(r),
		burst:           burst,
		cleanupInterval: 5 * time.Minute,
		maxAge:          10 * time.Minute, // Limiter expires after 10 minutes
		lastCleanup:     time.Now(),
		stopCh:          make(chan struct{}),
	}
	// Start cleanup goroutine
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop Periodically cleans up old limiters
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

// Stop Stops the cleanup goroutine
// Should be called during server shutdown
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// cleanup Remove limiters that haven't been used recently
func (rl *RateLimiter) cleanup() {
	// Only cleanup if enough time has passed
	if time.Since(rl.lastCleanup) < rl.cleanupInterval {
		return
	}

	rl.lastCleanup = time.Now()

	// Iterate through all limiters and remove old ones
	now := time.Now()
	rl.limiters.Range(func(key, value interface{}) bool {
		entry := value.(*limiterWithTime)
		if now.Sub(entry.createdAt) > rl.maxAge {
			rl.limiters.Delete(key)
		}
		return true
	})
}

// getLimiter Get or create rate limiter for IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	entry := &limiterWithTime{
		limiter:   rate.NewLimiter(rl.rate, rl.burst),
		createdAt: time.Now(),
	}
	// Use LoadOrStore to prevent race condition
	actual, loaded := rl.limiters.LoadOrStore(ip, entry)
	if loaded {
		return actual.(*limiterWithTime).limiter
	}
	return entry.limiter
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
