/*
Package errors 定义应用层错误码及领域错误映射。

约束：
- 错误码用于跨层语义表达，不绑定 HTTP。
- 领域错误到应用错误的映射在本包完成。
*/
package errors

import (
	"errors"
	"fmt"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"
)

type ErrorCode string

const (
	CodeInternal   ErrorCode = "INTERNAL_ERROR"
	CodeBadRequest ErrorCode = "BAD_REQUEST"
	CodeNotFound   ErrorCode = "NOT_FOUND"
	CodeConflict   ErrorCode = "CONFLICT"
	CodeForbidden  ErrorCode = "FORBIDDEN"
	CodeValidation ErrorCode = "VALIDATION_ERROR"

	CodeOrderNotFound     ErrorCode = "ORDER_NOT_FOUND"
	CodeInvalidOrderState ErrorCode = "INVALID_ORDER_STATE"
	CodeConcurrentModify  ErrorCode = "CONCURRENT_MODIFY"

	CodeUserNotFound      ErrorCode = "USER_NOT_FOUND"
	CodeUserNotActive     ErrorCode = "USER_NOT_ACTIVE"
	CodeUserTooYoung      ErrorCode = "USER_TOO_YOUNG"
	CodeEmailAlreadyExist ErrorCode = "EMAIL_ALREADY_EXISTS"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     fmt.Errorf("%s: %w", message, err),
	}
}

func BadRequest(message string) *AppError { return New(CodeBadRequest, message) }
func NotFound(message string) *AppError   { return New(CodeNotFound, message) }
func Internal(message string) *AppError   { return New(CodeInternal, message) }
func Conflict(message string) *AppError   { return New(CodeConflict, message) }
func Forbidden(message string) *AppError  { return New(CodeForbidden, message) }
func Validation(message string) *AppError { return New(CodeValidation, message) }

func OrderNotFound() *AppError                   { return New(CodeOrderNotFound, "order not found") }
func InvalidOrderState(message string) *AppError { return New(CodeInvalidOrderState, message) }
func UserNotFound() *AppError                    { return New(CodeUserNotFound, "user not found") }
func UserNotActive() *AppError                   { return New(CodeUserNotActive, "user is not active") }

func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(err, CodeInternal, "internal server error")
}

// FromDomainError 将领域层错误转换为应用层错误码。
func FromDomainError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		return &AppError{Code: CodeOrderNotFound, Message: err.Error(), Err: err}
	case errors.Is(err, order.ErrConcurrentModification):
		return &AppError{Code: CodeConcurrentModify, Message: "please retry your operation", Err: err}
	case errors.Is(err, order.ErrInvalidOrderState), errors.Is(err, order.ErrInvalidOrderStateTransition):
		return &AppError{Code: CodeInvalidOrderState, Message: err.Error(), Err: err}
	case errors.Is(err, order.ErrUserCannotPlaceOrder), errors.Is(err, order.ErrUserNotActiveForOrder):
		return &AppError{Code: CodeUserNotActive, Message: err.Error(), Err: err}
	case errors.Is(err, order.ErrEmptyOrderItems), errors.Is(err, order.ErrInvalidQuantity), errors.Is(err, order.ErrOrderTotalAmountNotPositive):
		return &AppError{Code: CodeValidation, Message: err.Error(), Err: err}

	case errors.Is(err, user.ErrEmailAlreadyExists):
		return &AppError{Code: CodeEmailAlreadyExist, Message: "email already exists", Err: err}
	case errors.Is(err, user.ErrUserNotActive):
		return &AppError{Code: CodeUserNotActive, Message: err.Error(), Err: err}
	case errors.Is(err, user.ErrUserTooYoung):
		return &AppError{Code: CodeUserTooYoung, Message: err.Error(), Err: err}
	case errors.Is(err, user.ErrInvalidEmail), errors.Is(err, user.ErrInvalidName), errors.Is(err, user.ErrInvalidAge):
		return &AppError{Code: CodeValidation, Message: err.Error(), Err: err}
	case errors.Is(err, user.ErrConcurrentModification):
		return &AppError{Code: CodeConcurrentModify, Message: "please retry your operation", Err: err}

	case errors.Is(err, shared.ErrNotFound):
		return &AppError{Code: CodeNotFound, Message: err.Error(), Err: err}
	case errors.Is(err, shared.ErrConflict):
		return &AppError{Code: CodeConflict, Message: err.Error(), Err: err}
	case errors.Is(err, shared.ErrInvalidInput):
		return &AppError{Code: CodeValidation, Message: err.Error(), Err: err}
	case errors.Is(err, shared.ErrForbidden):
		return &AppError{Code: CodeForbidden, Message: err.Error(), Err: err}
	default:
		return &AppError{Code: CodeInternal, Message: "internal server error", Err: err}
	}
}
