package order

import "errors"

// 领域错误定义
var (
	ErrOrderNotFound = errors.New("order not found")
)
