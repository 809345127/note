/*
Package order - 订单领域错误定义

设计原则:
1. 使用哨兵错误(sentinel errors)支持 errors.Is() 类型安全判断
2. 错误构造函数在创建时捕获堆栈，便于定位错误发生点
3. 所有错误都支持错误链，可追溯根因
4. 不包含 HTTP 状态码等非领域概念

堆栈捕获:
- NewXxxError 构造函数内部调用 shared.CaptureStack(3)
- skip=3 跳过：runtime.Callers, CaptureStack, NewXxxError
- 堆栈从调用 NewXxxError 的位置开始（通常是 Repository）
*/
package order

import (
	"ddd/domain/shared"
	"errors"
)

// ============================================================================
// 订单领域哨兵错误 (Sentinel Errors)
// 用于 errors.Is() 判断，是底层错误分类的标识
// ============================================================================

var (
	// ErrOrderNotFound 订单未找到
	// 可用于: errors.Is(err, ErrOrderNotFound)
	ErrOrderNotFound = errors.New("order not found")

	// ErrConcurrentModification 并发修改冲突（乐观锁）
	// 当订单被其他事务修改时返回此错误，调用方应重试
	ErrConcurrentModification = errors.New("order was modified by another transaction, please retry")

	// ErrInvalidOrderState 无效的订单状态转换
	// 例如：已取消的订单不能确认
	ErrInvalidOrderState = errors.New("invalid order state transition")

	// ErrEmptyOrderItems 订单项为空
	ErrEmptyOrderItems = errors.New("order must have at least one item")

	// ErrUserCannotPlaceOrder 用户无法下单
	// 例如：用户未激活、被禁用等
	ErrUserCannotPlaceOrder = errors.New("user cannot place order")

	// ErrInvalidQuantity 无效的订单项数量
	ErrInvalidQuantity = errors.New("quantity must be positive")

	// ErrOrderTotalAmountNotPositive 订单总金额必须为正数
	ErrOrderTotalAmountNotPositive = errors.New("order total amount must be positive")

	// ErrCannotModifyNonPendingOrder 无法修改非待处理状态的订单
	ErrCannotModifyNonPendingOrder = errors.New("can only modify pending orders")

	// ErrItemNotFound 订单项不存在
	ErrItemNotFound = errors.New("item not found")

	// ErrInvalidOrderStateTransition 无效的订单状态转换
	ErrInvalidOrderStateTransition = errors.New("invalid order state transition")

	// ErrUserNotActiveForOrder 用户未激活，无法处理订单
	ErrUserNotActiveForOrder = errors.New("user is not active")
)

// ============================================================================
// 订单领域错误构造函数
// 创建携带完整上下文和堆栈的结构化错误
// ============================================================================

// NewOrderNotFoundError 创建订单未找到错误（带堆栈）
// 返回的错误支持:
//   - errors.Is(err, ErrOrderNotFound)
//   - err.(shared.Stacker).Stack() 获取堆栈
func NewOrderNotFoundError(orderID string) error {
	return &orderDomainError{
		sentinel: ErrOrderNotFound,
		entity:   "order",
		message:  "order not found: " + orderID,
		stack:    shared.CaptureStack(3),
	}
}

// NewConcurrentModificationError 创建并发修改错误
func NewConcurrentModificationError(orderID string) error {
	return &orderDomainError{
		sentinel: ErrConcurrentModification,
		entity:   "order",
		message:  "order " + orderID + " was modified by another transaction, please retry",
		stack:    shared.CaptureStack(3),
	}
}

// NewInvalidOrderStateError 创建无效状态转换错误
// currentState: 当前状态, targetState: 目标状态
func NewInvalidOrderStateError(currentState, targetState string) error {
	return &orderDomainError{
		sentinel: ErrInvalidOrderState,
		entity:   "order",
		message:  "cannot transition from " + currentState + " to " + targetState,
		stack:    shared.CaptureStack(3),
	}
}

// NewEmptyOrderItemsError 创建订单项为空错误
func NewEmptyOrderItemsError() error {
	return &orderDomainError{
		sentinel: ErrEmptyOrderItems,
		entity:   "order",
		field:    "items",
		message:  "order must have at least one item",
		stack:    shared.CaptureStack(3),
	}
}

// NewUserCannotPlaceOrderError 创建用户无法下单错误
func NewUserCannotPlaceOrderError(userID, reason string) error {
	return &orderDomainError{
		sentinel: ErrUserCannotPlaceOrder,
		entity:   "order",
		message:  "user " + userID + " cannot place order: " + reason,
		stack:    shared.CaptureStack(3),
	}
}

// ============================================================================
// 订单领域错误结构体（内部使用）
// 实现 error, Unwrap, Stacker 接口
// ============================================================================

// orderDomainError 订单领域错误（带堆栈）
type orderDomainError struct {
	sentinel error     // 哨兵错误，用于 errors.Is()
	entity   string    // 实体名
	field    string    // 字段名（可选）
	message  string    // 错误消息
	stack    []uintptr // 调用栈
}

func (e *orderDomainError) Error() string {
	return e.message
}

func (e *orderDomainError) Unwrap() error {
	return e.sentinel
}

// Stack 实现 shared.Stacker 接口
func (e *orderDomainError) Stack() []string {
	if len(e.stack) == 0 {
		return nil
	}

	return shared.FormatStack(e.stack)
}
