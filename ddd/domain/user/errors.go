package user

import "errors"

// 领域错误定义
// DDD原则：领域层定义业务相关的错误类型
// 这些错误表达业务规则的违反，而非技术错误

var (
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidName       = errors.New("name cannot be empty")
	ErrInvalidAge        = errors.New("age must be between 0 and 150")
	ErrUserNotActive     = errors.New("user is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrUserNotFound      = errors.New("user not found")
)
