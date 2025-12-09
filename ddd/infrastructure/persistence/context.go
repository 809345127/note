package persistence

import (
	"context"

	"gorm.io/gorm"
)

// txKey is the context key for storing the transaction
type txKey struct{}

// TxFromContext retrieves the GORM transaction from context
// Returns nil if no transaction is present
func TxFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// ContextWithTx returns a new context with the GORM transaction attached
func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
