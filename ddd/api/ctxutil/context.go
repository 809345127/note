package ctxutil

import (
	"context"

	"ddd/api/response"
	"ddd/infrastructure/persistence"

	"github.com/gin-gonic/gin"
)

func WithRequestID(ctx *gin.Context) context.Context {
	requestID := response.GetRequestID(ctx)
	return persistence.ContextWithRequestID(ctx.Request.Context(), requestID)
}
func RequestIDFromContext(ctx context.Context) string {
	return persistence.RequestIDFromContext(ctx)
}
