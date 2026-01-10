/*
Package user - 用户领域错误定义

设计原则:
1. 使用哨兵错误(sentinel errors)支持 errors.Is() 类型安全判断
2. 错误构造函数在创建时捕获堆栈，便于定位错误发生点
3. 所有错误都支持错误链，可追溯根因
4. 使用 shared.ErrNotFound 作为底层错误，保持一致性

堆栈捕获:
- NewXxxError 构造函数内部调用 shared.CaptureStack(3)
- skip=3 跳过：runtime.Callers, CaptureStack, NewXxxError
- 堆栈从调用 NewXxxError 的位置开始（通常是 Repository）
*/
package user

import (
	"ddd/domain/shared"
	"errors"
	"fmt"
)

// ============================================================================
// 用户领域哨兵错误 (Sentinel Errors)
// 用于 errors.Is() 判断，是底层错误分类的标识
// ============================================================================

var (
	// ErrInvalidEmail 邮箱格式无效
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidName 姓名无效
	ErrInvalidName = errors.New("name cannot be empty")

	// ErrInvalidAge 年龄无效
	ErrInvalidAge = errors.New("age must be between 0 and 150")

	// ErrUserNotActive 用户未激活
	ErrUserNotActive = errors.New("user is not active")

	// ErrInsufficientFunds 资金不足
	ErrInsufficientFunds = errors.New("insufficient funds")

	// ErrConcurrentModification 并发修改冲突（乐观锁）
	ErrConcurrentModification = errors.New("user was modified by another transaction, please retry")

	// ErrUserNotFound 使用 shared.ErrNotFound 作为底层错误
	// 可用于: errors.Is(err, shared.ErrNotFound)

	// ErrEmailAlreadyExists 邮箱已存在（数据库唯一约束冲突）
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUserTooYoung 用户年龄不足，无法下单
	ErrUserTooYoung = errors.New("user is too young to make purchases")
)

// ============================================================================
// 用户领域错误构造函数
// 创建携带完整上下文和堆栈的结构化错误
// ============================================================================

// NewUserNotFoundError 创建用户未找到错误（带堆栈）
// 返回的错误支持:
//   - errors.Is(err, shared.ErrNotFound)
//   - err.(shared.Stacker).Stack() 获取堆栈
func NewUserNotFoundError(userID string) error {
	return &userDomainError{
		sentinel: shared.ErrNotFound,
		entity:   "user",
		message:  "user not found: " + userID,
		stack:    shared.CaptureStack(3),
	}
}

// NewConcurrentModificationError 创建并发修改错误
func NewConcurrentModificationError(userID string) error {
	return &userDomainError{
		sentinel: ErrConcurrentModification,
		entity:   "user",
		message:  "user " + userID + " was modified by another transaction, please retry",
		stack:    shared.CaptureStack(3),
	}
}

// NewInvalidEmailError 创建邮箱格式错误
func NewInvalidEmailError(email string) error {
	return &userDomainError{
		sentinel: ErrInvalidEmail,
		entity:   "user",
		field:    "email",
		message:  "invalid email format: " + email,
		stack:    shared.CaptureStack(3),
	}
}

// NewInvalidNameError 创建姓名为空错误
func NewInvalidNameError() error {
	return &userDomainError{
		sentinel: ErrInvalidName,
		entity:   "user",
		field:    "name",
		message:  "name cannot be empty",
		stack:    shared.CaptureStack(3),
	}
}

// NewInvalidAgeError 创建年龄无效错误
func NewInvalidAgeError(age int) error {
	return &userDomainError{
		sentinel: ErrInvalidAge,
		entity:   "user",
		field:    "age",
		message:  fmt.Sprintf("age must be between 0 and 150, got: %d", age),
		stack:    shared.CaptureStack(3),
	}
}

// NewUserNotActiveError 创建用户未激活错误
func NewUserNotActiveError(userID string) error {
	return &userDomainError{
		sentinel: ErrUserNotActive,
		entity:   "user",
		message:  "user " + userID + " is not active",
		stack:    shared.CaptureStack(3),
	}
}

// NewEmailAlreadyExistsError 创建邮箱已存在错误（带堆栈）
// 用于数据库唯一约束冲突场景
func NewEmailAlreadyExistsError(email string) error {
	return &userDomainError{
		sentinel: ErrEmailAlreadyExists,
		entity:   "user",
		field:    "email",
		message:  "email already exists: " + email,
		stack:    shared.CaptureStack(3),
	}
}

// ============================================================================
// 用户领域错误结构体（内部使用）
// 实现 error, Unwrap, Stacker 接口
// ============================================================================

// userDomainError 用户领域错误（带堆栈）
type userDomainError struct {
	sentinel error     // 哨兵错误，用于 errors.Is()
	entity   string    // 实体名
	field    string    // 字段名（可选）
	message  string    // 错误消息
	stack    []uintptr // 调用栈
}

func (e *userDomainError) Error() string {
	return e.message
}

func (e *userDomainError) Unwrap() error {
	return e.sentinel
}

// Stack 实现 shared.Stacker 接口
func (e *userDomainError) Stack() []string {
	if len(e.stack) == 0 {
		return nil
	}

	return shared.FormatStack(e.stack)
}
