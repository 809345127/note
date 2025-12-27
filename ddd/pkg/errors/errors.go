/*
Package errors - 应用层错误定义

设计原则:
1. 只使用标准库 errors，不依赖第三方包
2. 应用层错误码用于跨层通信，不含 HTTP 概念
3. HTTP 状态码映射放在 API 层（api/response）
4. 使用 errors.Is() 进行类型安全的错误判断，不用字符串匹配
5. 堆栈追踪由日志层在记录时捕获，不存储在错误结构体中

错误流转:

	Domain Error (领域层)
	     ↓ errors.Is() 判断
	AppError (应用层) - 本包定义
	     ↓ API 层映射
	HTTP Response (表示层)
*/
package errors

import (
	"errors"
	"fmt"

	"ddd/domain/order"
	"ddd/domain/shared"
)

// ============================================================================
// 应用层错误码 (Application Error Codes)
// 用于跨层通信的标准化错误分类
// ============================================================================

// ErrorCode 错误码类型
type ErrorCode string

const (
	// 通用错误码
	CodeInternal   ErrorCode = "INTERNAL_ERROR"   // 内部错误（未知错误）
	CodeBadRequest ErrorCode = "BAD_REQUEST"      // 请求参数错误
	CodeNotFound   ErrorCode = "NOT_FOUND"        // 资源未找到
	CodeConflict   ErrorCode = "CONFLICT"         // 资源冲突
	CodeForbidden  ErrorCode = "FORBIDDEN"        // 禁止访问
	CodeValidation ErrorCode = "VALIDATION_ERROR" // 参数校验失败

	// 业务错误码 - 订单相关
	CodeOrderNotFound     ErrorCode = "ORDER_NOT_FOUND"     // 订单未找到
	CodeInvalidOrderState ErrorCode = "INVALID_ORDER_STATE" // 无效订单状态
	CodeConcurrentModify  ErrorCode = "CONCURRENT_MODIFY"   // 并发修改冲突

	// 业务错误码 - 用户相关
	CodeUserNotFound  ErrorCode = "USER_NOT_FOUND"  // 用户未找到
	CodeUserNotActive ErrorCode = "USER_NOT_ACTIVE" // 用户未激活
)

// ============================================================================
// 应用层错误结构体 (Application Error)
// ============================================================================

// AppError 应用层错误
// 注意: 不包含 Stack 字段，堆栈由日志层按需捕获
type AppError struct {
	// Code 错误码，用于程序判断
	Code ErrorCode

	// Message 用户可见的错误消息
	Message string

	// Err 原始错误，用于日志记录和错误链追踪
	// 使用 json:"-" 确保不会序列化到响应中
	Err error
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现错误链，支持 errors.Is() 和 errors.As()
func (e *AppError) Unwrap() error {
	return e.Err
}

// ============================================================================
// 应用层错误构造函数
// ============================================================================

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误，添加应用层上下文
// 使用标准库 fmt.Errorf 包装，保留完整错误链
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     fmt.Errorf("%s: %w", message, err),
	}
}

// ============================================================================
// 便捷构造函数 - 通用错误
// ============================================================================

// BadRequest 创建请求参数错误
func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message)
}

// NotFound 创建资源未找到错误
func NotFound(message string) *AppError {
	return New(CodeNotFound, message)
}

// Internal 创建内部错误
func Internal(message string) *AppError {
	return New(CodeInternal, message)
}

// Conflict 创建资源冲突错误
func Conflict(message string) *AppError {
	return New(CodeConflict, message)
}

// Forbidden 创建禁止访问错误
func Forbidden(message string) *AppError {
	return New(CodeForbidden, message)
}

// Validation 创建参数校验错误
func Validation(message string) *AppError {
	return New(CodeValidation, message)
}

// ============================================================================
// 便捷构造函数 - 业务错误
// ============================================================================

// OrderNotFound 创建订单未找到错误
func OrderNotFound() *AppError {
	return New(CodeOrderNotFound, "order not found")
}

// InvalidOrderState 创建无效订单状态错误
func InvalidOrderState(message string) *AppError {
	return New(CodeInvalidOrderState, message)
}

// UserNotFound 创建用户未找到错误
func UserNotFound() *AppError {
	return New(CodeUserNotFound, "user not found")
}

// UserNotActive 创建用户未激活错误
func UserNotActive() *AppError {
	return New(CodeUserNotActive, "user is not active")
}

// ============================================================================
// 错误判断和转换函数
// ============================================================================

// Is 判断是否为特定错误码
func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// AsAppError 将错误转换为 AppError
// 如果已经是 AppError 则直接返回，否则包装为内部错误
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	// 未知错误包装为内部错误
	return Wrap(err, CodeInternal, "internal server error")
}

// ============================================================================
// 领域错误映射 (Domain Error Mapping)
// 使用 errors.Is() 进行类型安全判断，替代字符串匹配
// ============================================================================

// FromDomainError 将领域错误映射为应用错误
// 这是领域层到应用层的错误转换边界
func FromDomainError(err error) *AppError {
	if err == nil {
		return nil
	}

	// 1. 已经是 AppError，直接返回
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// 2. 使用 errors.Is() 判断领域错误类型 - 类型安全，无字符串匹配
	switch {
	// 订单领域错误
	case errors.Is(err, order.ErrOrderNotFound):
		return &AppError{Code: CodeOrderNotFound, Message: err.Error(), Err: err}

	case errors.Is(err, order.ErrConcurrentModification):
		return &AppError{Code: CodeConcurrentModify, Message: "please retry your operation", Err: err}

	case errors.Is(err, order.ErrInvalidOrderState):
		return &AppError{Code: CodeInvalidOrderState, Message: err.Error(), Err: err}

	case errors.Is(err, order.ErrUserCannotPlaceOrder):
		return &AppError{Code: CodeUserNotActive, Message: err.Error(), Err: err}

	// 共享领域错误
	case errors.Is(err, shared.ErrNotFound):
		return &AppError{Code: CodeNotFound, Message: err.Error(), Err: err}

	case errors.Is(err, shared.ErrConflict):
		return &AppError{Code: CodeConflict, Message: err.Error(), Err: err}

	case errors.Is(err, shared.ErrInvalidInput):
		return &AppError{Code: CodeValidation, Message: err.Error(), Err: err}

	case errors.Is(err, shared.ErrForbidden):
		return &AppError{Code: CodeForbidden, Message: err.Error(), Err: err}

	default:
		// 3. 未知错误 - 不暴露内部细节，统一返回内部错误
		return &AppError{
			Code:    CodeInternal,
			Message: "internal server error",
			Err:     err, // 保留原始错误用于日志
		}
	}
}
