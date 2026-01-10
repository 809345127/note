package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"strings"
	"time"

	"ddd/config"
	"ddd/domain/order"
	"ddd/domain/user"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Config Retry configuration for the retry utility
type Config struct {
	Enabled                       bool
	MaxAttempts                   int
	InitialDelay                  time.Duration
	MaxDelay                      time.Duration
	BackoffFactor                 float64
	JitterEnabled                 bool
	RetryOnConcurrentModification bool
	RetryOnDeadlock               bool
	RetryOnLockTimeout            bool
	// RetryPredicate allows custom retry logic beyond built-in error detection
	RetryPredicate func(error) bool
}

// DefaultConfig Default retry configuration
var DefaultConfig = Config{
	Enabled:                       true,
	MaxAttempts:                   3,
	InitialDelay:                  100 * time.Millisecond,
	MaxDelay:                      2 * time.Second,
	BackoffFactor:                 2.0,
	JitterEnabled:                 true,
	RetryOnConcurrentModification: true,
	RetryOnDeadlock:               true,
	RetryOnLockTimeout:            true,
}

// FromAppConfig converts application configuration to retry utility configuration
func FromAppConfig(appConfig *config.Config) Config {
	dbConfig := appConfig.Database
	retryConfig := dbConfig.Retry

	return Config{
		Enabled:                       retryConfig.Enabled,
		MaxAttempts:                   retryConfig.MaxAttempts,
		InitialDelay:                  retryConfig.InitialDelay,
		MaxDelay:                      retryConfig.MaxDelay,
		BackoffFactor:                 retryConfig.BackoffFactor,
		JitterEnabled:                 retryConfig.JitterEnabled,
		RetryOnConcurrentModification: retryConfig.RetryOnConcurrentModification,
		RetryOnDeadlock:               retryConfig.RetryOnDeadlock,
		RetryOnLockTimeout:            retryConfig.RetryOnLockTimeout,
	}
}

// ExponentialBackoffWithJitter calculates delay with exponential backoff and jitter
// Uses the formula: delay = min(initial * factor^(attempt-1), max_delay) ± jitter
func ExponentialBackoffWithJitter(attempt int, config Config) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Exponential backoff: delay = initial * factor^(attempt-1)
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter (±20%)
	if config.JitterEnabled {
		jitterFactor := 0.8 + rand.Float64()*0.4 // Random between 0.8 and 1.2
		delay = delay * jitterFactor
	}

	// Ensure non-negative duration
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// IsRetryableError checks if an error is retryable based on configuration
func IsRetryableError(err error, config Config) bool {
	if err == nil {
		return false
	}

	// Check custom retry predicate first
	if config.RetryPredicate != nil && config.RetryPredicate(err) {
		return true
	}

	// Check for domain concurrent modification errors
	errStr := err.Error()
	if config.RetryOnConcurrentModification {
		if strings.Contains(errStr, "concurrent modification") ||
		   errors.Is(err, order.ErrConcurrentModification) ||
		   errors.Is(err, user.ErrConcurrentModification) {
			return true
		}
	}

	// Check for MySQL errors
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1213: // Deadlock
			return config.RetryOnDeadlock
		case 1205: // Lock wait timeout
			return config.RetryOnLockTimeout
		}
	}

	// Check for GORM deadlock errors
	if strings.Contains(errStr, "deadlock") || strings.Contains(errStr, "lock wait timeout") {
		if config.RetryOnDeadlock {
			return true
		}
	}

	// Check for connection-related errors that might be transient
	if errors.Is(err, gorm.ErrInvalidTransaction) ||
	   (strings.Contains(errStr, "connection") && strings.Contains(errStr, "lost")) {
		return true
	}

	// Unique constraint violations are NOT retryable (would always fail)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false
	}

	return false
}

// ExecuteWithRetry executes a function with retry logic
// Returns the original error if retries are exhausted or error is non-retryable
func ExecuteWithRetry(ctx context.Context, config Config, fn func(ctx context.Context) error) error {
	if !config.Enabled {
		return fn(ctx)
	}

	var lastErr error
	var attempt int

	for attempt = 1; attempt <= config.MaxAttempts; attempt++ {
		// Check if context is cancelled before attempting
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err, config) || attempt == config.MaxAttempts {
			break
		}

		// Calculate and wait for backoff

		delay := ExponentialBackoffWithJitter(attempt, config)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// Continue to next attempt (timer triggered, no need to stop)
			case <-ctx.Done():
				timer.Stop()  // Explicitly stop timer on context cancellation
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// ExecuteWithAppConfig executes a function with retry logic using application configuration

func ExecuteWithAppConfig(ctx context.Context, appConfig *config.Config, fn func(ctx context.Context) error) error {

	retryConfig := FromAppConfig(appConfig)

	return ExecuteWithRetry(ctx, retryConfig, fn)

}