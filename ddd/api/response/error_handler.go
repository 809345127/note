package response

import (
	stdErrors "errors"
	"net/http"
	"runtime"

	"ddd/domain/shared"
	"ddd/pkg/errors"
	"ddd/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var httpStatusMap = map[errors.ErrorCode]int{
	errors.CodeInternal:   http.StatusInternalServerError,
	errors.CodeBadRequest: http.StatusBadRequest,
	errors.CodeNotFound:   http.StatusNotFound,
	errors.CodeConflict:   http.StatusConflict,
	errors.CodeForbidden:  http.StatusForbidden,
	errors.CodeValidation: http.StatusBadRequest,

	errors.CodeOrderNotFound:     http.StatusNotFound,
	errors.CodeInvalidOrderState: http.StatusUnprocessableEntity,
	errors.CodeConcurrentModify:  http.StatusConflict,

	errors.CodeUserNotFound:      http.StatusNotFound,
	errors.CodeUserNotActive:     http.StatusForbidden,
	errors.CodeUserTooYoung:      http.StatusForbidden,
	errors.CodeEmailAlreadyExist: http.StatusConflict,
}

func mapErrorCodeToHTTPStatus(code errors.ErrorCode) int {
	if status, ok := httpStatusMap[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

func GetRequestID(c *gin.Context) string {
	return getRequestID(c)
}

func captureStack(skip int) []string {
	var pcs [16]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	stack := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		frame, more := frames.Next()
		if frame.Function != "" {
			stack = append(stack, frame.Function)
		}
		if !more {
			break
		}
	}
	return stack
}

// HandleError 处理参数绑定等框架层错误。
func HandleError(c *gin.Context, err error, message string, code int) {
	requestID := getRequestID(c)

	logger.Error(message,
		zap.String("request_id", requestID),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Int("status", code),
		zap.Error(err))

	c.JSON(code, &Response{
		Success:   false,
		Error:     "BAD_REQUEST",
		Message:   message,
		Code:      code,
		RequestID: requestID,
	})
}

// HandleAppError 按应用错误码自动映射 HTTP 状态码。
func HandleAppError(c *gin.Context, err error) {
	requestID := getRequestID(c)
	appErr := errors.FromDomainError(err)
	httpStatus := mapErrorCodeToHTTPStatus(appErr.Code)
	stack := extractStack(err)

	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("error_code", string(appErr.Code)),
		zap.Int("http_status", httpStatus),
		zap.Strings("stack", stack),
	}
	if appErr.Err != nil {
		fields = append(fields, zap.Error(appErr.Err))
	}

	logger.Error(appErr.Message, fields...)

	userMessage := appErr.Message
	if appErr.Code == errors.CodeInternal {
		userMessage = "internal server error"
	}

	c.JSON(httpStatus, &Response{
		Success:   false,
		Error:     string(appErr.Code),
		Message:   userMessage,
		Code:      httpStatus,
		RequestID: requestID,
	})
}

func extractStack(err error) []string {
	var stacker shared.Stacker
	if stdErrors.As(err, &stacker) {
		if stack := stacker.Stack(); len(stack) > 0 {
			return stack
		}
	}
	return captureStack(4)
}
