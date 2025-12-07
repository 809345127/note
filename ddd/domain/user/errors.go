package user

import "errors"

// Domain error definitions
// DDD principle: Domain layer defines business-related error types
// These errors express violations of business rules, not technical errors

var (
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidName       = errors.New("name cannot be empty")
	ErrInvalidAge        = errors.New("age must be between 0 and 150")
	ErrUserNotActive     = errors.New("user is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrUserNotFound      = errors.New("user not found")
)
