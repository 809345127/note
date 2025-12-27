/*
Package shared - 领域层共享错误定义

设计原则:
1. 领域层定义哨兵错误(sentinel errors)，用于 errors.Is() 类型安全判断
2. DomainError 在创建时捕获堆栈，但延迟格式化（按需打印）
3. 领域错误不包含 HTTP 状态码等传输层概念
4. 使用标准库 errors，不依赖第三方包

堆栈捕获策略:
- 捕获时机：错误创建时（构造函数内）
- 格式化时机：日志打印时（Stack() 方法）
- 这样既能精确定位错误发生点，又避免了不必要的格式化开销
*/
package shared

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ============================================================================
// 哨兵错误 (Sentinel Errors)
// 用于 errors.Is() 判断错误类型，不携带具体信息
// ============================================================================

var (
	// ErrNotFound 资源未找到
	ErrNotFound = errors.New("not found")

	// ErrConflict 资源冲突（如并发修改、唯一约束冲突）
	ErrConflict = errors.New("conflict")

	// ErrInvalidInput 无效输入（参数校验失败）
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized 未授权
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden 禁止访问（已授权但无权限）
	ErrForbidden = errors.New("forbidden")
)

// ============================================================================
// 领域错误结构体 (Domain Error)
// 携带业务上下文和发生点堆栈，支持 errors.Is() 和 errors.As()
// ============================================================================

// DomainError 领域错误 - 携带业务上下文和堆栈的结构化错误
type DomainError struct {
	// Err 底层哨兵错误，用于 errors.Is() 判断
	Err error

	// Entity 发生错误的实体名称（如 "order", "user"）
	Entity string

	// Message 人类可读的错误描述
	Message string

	// Field 可选：发生错误的字段名（用于校验错误）
	Field string

	// stack 调用栈帧（私有），在创建时捕获，按需格式化
	stack []uintptr
}

// Error 实现 error 接口
func (e *DomainError) Error() string {
	return e.Message
}

// Unwrap 实现错误链，支持 errors.Is() 和 errors.As()
func (e *DomainError) Unwrap() error {
	return e.Err
}

// Stack 按需格式化堆栈（只在打印日志时调用）
// 返回格式化后的堆栈字符串切片，每个元素是一个调用帧
func (e *DomainError) Stack() []string {
	return FormatStack(e.stack)
}

// ============================================================================
// 堆栈捕获辅助函数
// ============================================================================

// CaptureStack 捕获当前调用栈（导出供子领域包使用）
// skip: 跳过的帧数（通常为 3：Callers, CaptureStack, NewXxxError）
func CaptureStack(skip int) []uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[:n]
}

// FormatStack 格式化堆栈帧为字符串切片（导出供子领域包使用）
// 过滤 runtime 内部帧，最多返回 10 帧
func FormatStack(stack []uintptr) []string {
	if len(stack) == 0 {
		return nil
	}

	frames := runtime.CallersFrames(stack)
	var result []string
	for {
		frame, more := frames.Next()
		// 过滤掉 runtime 内部帧
		if !strings.Contains(frame.File, "runtime/") {
			result = append(result, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		}
		if !more || len(result) > 10 {
			break
		}
	}
	return result
}

// ============================================================================
// 领域错误构造函数
// 在创建时捕获堆栈，提供语义化的错误创建方式
// ============================================================================

// NewNotFoundError 创建"未找到"领域错误
// 堆栈从调用此函数的位置开始捕获
func NewNotFoundError(entity string) error {
	return &DomainError{
		Err:     ErrNotFound,
		Entity:  entity,
		Message: entity + " not found",
		stack:   CaptureStack(3),
	}
}

// NewConflictError 创建"冲突"领域错误
func NewConflictError(entity, message string) error {
	return &DomainError{
		Err:     ErrConflict,
		Entity:  entity,
		Message: message,
		stack:   CaptureStack(3),
	}
}

// NewValidationError 创建"校验失败"领域错误
func NewValidationError(entity, field, reason string) error {
	return &DomainError{
		Err:     ErrInvalidInput,
		Entity:  entity,
		Field:   field,
		Message: reason,
		stack:   CaptureStack(3),
	}
}

// NewForbiddenError 创建"禁止访问"领域错误
func NewForbiddenError(entity, reason string) error {
	return &DomainError{
		Err:     ErrForbidden,
		Entity:  entity,
		Message: reason,
		stack:   CaptureStack(3),
	}
}

// ============================================================================
// Stacker 接口
// 用于 API 层统一提取堆栈
// ============================================================================

// Stacker 可提供堆栈的错误接口
type Stacker interface {
	Stack() []string
}
