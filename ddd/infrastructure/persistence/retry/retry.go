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
	RetryPredicate                func(error) bool
}

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
func ExponentialBackoffWithJitter(attempt int, config Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	if config.JitterEnabled {
		jitterFactor := 0.8 + rand.Float64()*0.4
		delay = delay * jitterFactor
	}
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}
func IsRetryableError(err error, config Config) bool {
	if err == nil {
		return false
	}
	if config.RetryPredicate != nil && config.RetryPredicate(err) {
		return true
	}
	errStr := err.Error()
	if config.RetryOnConcurrentModification {
		if strings.Contains(errStr, "concurrent modification") ||
			errors.Is(err, order.ErrConcurrentModification) ||
			errors.Is(err, user.ErrConcurrentModification) {
			return true
		}
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1213:
			return config.RetryOnDeadlock
		case 1205:
			return config.RetryOnLockTimeout
		}
	}
	if strings.Contains(errStr, "deadlock") || strings.Contains(errStr, "lock wait timeout") {
		if config.RetryOnDeadlock {
			return true
		}
	}
	if errors.Is(err, gorm.ErrInvalidTransaction) ||
		(strings.Contains(errStr, "connection") && strings.Contains(errStr, "lost")) {
		return true
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false
	}

	return false
}
func ExecuteWithRetry(ctx context.Context, config Config, fn func(ctx context.Context) error) error {
	if !config.Enabled {
		return fn(ctx)
	}

	var lastErr error
	var attempt int

	for attempt = 1; attempt <= config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if !IsRetryableError(err, config) || attempt == config.MaxAttempts {
			break
		}

		delay := ExponentialBackoffWithJitter(attempt, config)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}

	return lastErr
}

func ExecuteWithAppConfig(ctx context.Context, appConfig *config.Config, fn func(ctx context.Context) error) error {

	retryConfig := FromAppConfig(appConfig)

	return ExecuteWithRetry(ctx, retryConfig, fn)

}
