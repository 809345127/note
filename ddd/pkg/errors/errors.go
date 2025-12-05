package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode 错误码
type ErrorCode string

const (
	// 通用错误码
	CodeInternal       ErrorCode = "INTERNAL_ERROR"
	CodeBadRequest     ErrorCode = "BAD_REQUEST"
	CodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	CodeForbidden      ErrorCode = "FORBIDDEN"
	CodeNotFound       ErrorCode = "NOT_FOUND"
	CodeConflict       ErrorCode = "CONFLICT"
	CodeTooManyRequest ErrorCode = "TOO_MANY_REQUESTS"
	CodeValidation     ErrorCode = "VALIDATION_ERROR"

	// 业务错误码
	CodeUserNotFound     ErrorCode = "USER_NOT_FOUND"
	CodeUserNotActive    ErrorCode = "USER_NOT_ACTIVE"
	CodeEmailExists      ErrorCode = "EMAIL_EXISTS"
	CodeOrderNotFound    ErrorCode = "ORDER_NOT_FOUND"
	CodeInvalidOrderState ErrorCode = "INVALID_ORDER_STATE"
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatusCode 返回对应的HTTP状态码
func (e *AppError) HTTPStatusCode() int {
	switch e.Code {
	case CodeBadRequest, CodeValidation:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound, CodeUserNotFound, CodeOrderNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeEmailExists:
		return http.StatusConflict
	case CodeTooManyRequest:
		return http.StatusTooManyRequests
	case CodeUserNotActive, CodeInvalidOrderState:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// New 创建新错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 常用错误构造函数

func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message)
}

func NotFound(message string) *AppError {
	return New(CodeNotFound, message)
}

func Internal(message string) *AppError {
	return New(CodeInternal, message)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message)
}

func Conflict(message string) *AppError {
	return New(CodeConflict, message)
}

func TooManyRequests(message string) *AppError {
	return New(CodeTooManyRequest, message)
}

func Validation(message string) *AppError {
	return New(CodeValidation, message)
}

// 业务错误

func UserNotFound() *AppError {
	return New(CodeUserNotFound, "user not found")
}

func UserNotActive() *AppError {
	return New(CodeUserNotActive, "user is not active")
}

func EmailExists() *AppError {
	return New(CodeEmailExists, "email already exists")
}

func OrderNotFound() *AppError {
	return New(CodeOrderNotFound, "order not found")
}

func InvalidOrderState(message string) *AppError {
	return New(CodeInvalidOrderState, message)
}

// Is 检查是否为特定错误码
func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// AsAppError 将错误转换为 AppError
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	// 如果不是 AppError，包装为内部错误
	return Wrap(err, CodeInternal, "internal server error")
}

// MapDomainError 将领域错误映射为应用错误
func MapDomainError(err error) *AppError {
	if err == nil {
		return nil
	}

	// 已经是 AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// 根据错误信息映射
	msg := err.Error()
	switch msg {
	case "user not found":
		return UserNotFound()
	case "user is not active":
		return UserNotActive()
	case "email already exists":
		return EmailExists()
	case "order not found":
		return OrderNotFound()
	case "user cannot place order":
		return UserNotActive()
	default:
		// 检查是否包含特定关键词
		if contains(msg, "not found") {
			return NotFound(msg)
		}
		if contains(msg, "invalid") {
			return BadRequest(msg)
		}
		if contains(msg, "already exists") {
			return Conflict(msg)
		}
		return Wrap(err, CodeInternal, msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAny(s, substr))
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
