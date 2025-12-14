package order

import "errors"

// Domain error definitions
var (
	ErrOrderNotFound         = errors.New("order not found")
	ErrConcurrentModification = errors.New("order was modified by another transaction, please retry")
)
