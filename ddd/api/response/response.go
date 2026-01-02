/*
Package response - API 层统一响应处理

设计原则:
1. HTTP 状态码映射放在 API 层，不污染领域层和应用层
2. 错误响应不暴露内部细节（堆栈、内部错误消息等）
3. 所有响应携带 RequestID 用于日志追踪
4. 内部错误统一返回 "internal server error"，真实错误只记录日志

堆栈提取策略:
1. 优先从领域错误（实现 shared.Stacker 接口）提取"错误发生点"堆栈
2. 如果错误不带堆栈，则在此处捕获"错误处理点"堆栈作为兜底

响应格式:

	成功: { success: true, data: {...}, message: "...", code: 200, request_id: "..." }
	失败: { success: false, error: "ERROR_CODE", message: "用户可见消息", code: 4xx/5xx, request_id: "..." }
*/
package response

import (
	"net/http"
	"runtime"

	"ddd/domain/shared"
	"ddd/pkg/errors"
	"ddd/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestIDKey context key for request id propagation
const RequestIDKey = "request_id"

// ============================================================================
// 响应结构体定义
// ============================================================================

// Response 通用响应结构
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`      // 错误码，不是错误详情
	Code      int         `json:"code"`                 // HTTP 状态码
	Message   string      `json:"message"`              // 用户可见消息
	RequestID string      `json:"request_id,omitempty"` // 请求追踪 ID
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Message    string      `json:"message"`
	Code       int         `json:"code"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// ============================================================================
// HTTP 状态码映射 (仅在 API 层)
// ============================================================================

// httpStatusMap 错误码到 HTTP 状态码的映射
// 这个映射只在 API 层使用，不暴露给其他层
var httpStatusMap = map[errors.ErrorCode]int{
	// 通用错误码
	errors.CodeInternal:   http.StatusInternalServerError,
	errors.CodeBadRequest: http.StatusBadRequest,
	errors.CodeNotFound:   http.StatusNotFound,
	errors.CodeConflict:   http.StatusConflict,
	errors.CodeForbidden:  http.StatusForbidden,
	errors.CodeValidation: http.StatusBadRequest,

	// 业务错误码 - 订单
	errors.CodeOrderNotFound:     http.StatusNotFound,
	errors.CodeInvalidOrderState: http.StatusUnprocessableEntity,
	errors.CodeConcurrentModify:  http.StatusConflict,

	// 业务错误码 - 用户
	errors.CodeUserNotFound:  http.StatusNotFound,
	errors.CodeUserNotActive: http.StatusForbidden,
}

// mapErrorCodeToHTTPStatus 将错误码映射为 HTTP 状态码
func mapErrorCodeToHTTPStatus(code errors.ErrorCode) int {
	if status, ok := httpStatusMap[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// ============================================================================
// 辅助函数
// ============================================================================

// getRequestID 从上下文获取请求 ID
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// captureStack 捕获调用栈（用于错误日志）
func captureStack(skip int) []string {
	var pcs [16]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stack []string
	for i := 0; i < 5; i++ { // 只取前 5 帧
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

// ============================================================================
// 错误处理函数
// ============================================================================

// HandleError 处理普通错误（非应用错误）
// 用于处理参数绑定等框架层错误
func HandleError(c *gin.Context, err error, message string, code int) {
	requestID := getRequestID(c)

	// 记录错误日志
	logger.Error(message,
		zap.String("request_id", requestID),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Int("status", code),
		zap.Error(err))

	response := &Response{
		Success:   false,
		Error:     "BAD_REQUEST",
		Message:   message,
		Code:      code,
		RequestID: requestID,
	}
	c.JSON(code, response)
}

// HandleAppError 处理应用层错误
// 自动映射 HTTP 状态码，记录完整错误日志，但不暴露内部细节给客户端
// 堆栈提取: 优先从错误中提取"发生点"堆栈，否则在此处捕获"处理点"堆栈
func HandleAppError(c *gin.Context, err error) {
	requestID := getRequestID(c)

	// 转换为应用错误
	appErr := errors.FromDomainError(err)

	// 获取 HTTP 状态码
	httpStatus := mapErrorCodeToHTTPStatus(appErr.Code)

	// 提取堆栈：优先从错误中提取"发生点"堆栈
	stack := extractStack(err)

	// 构建日志字段
	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("error_code", string(appErr.Code)),
		zap.Int("http_status", httpStatus),
		zap.Strings("stack", stack),
	}

	// 如果有内部错误，记录完整错误链
	if appErr.Err != nil {
		fields = append(fields, zap.Error(appErr.Err))
	}

	// 记录完整错误日志
	logger.Error(appErr.Message, fields...)

	// 构建响应 - 不暴露内部错误细节
	userMessage := appErr.Message
	// 内部错误不暴露真实消息
	if appErr.Code == errors.CodeInternal {
		userMessage = "internal server error"
	}

	response := &Response{
		Success:   false,
		Error:     string(appErr.Code),
		Message:   userMessage,
		Code:      httpStatus,
		RequestID: requestID,
	}
	c.JSON(httpStatus, response)
}

// extractStack 从错误中提取堆栈
// 优先提取"错误发生点"堆栈（如果错误实现了 Stacker 接口）
// 否则在此处捕获"错误处理点"堆栈作为兜底
func extractStack(err error) []string {
	// 尝试提取领域错误的堆栈（错误发生点）
	if stacker, ok := err.(shared.Stacker); ok {
		if stack := stacker.Stack(); len(stack) > 0 {
			return stack
		}
	}

	// 尝试从包装的内部错误中提取
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		if inner := unwrapper.Unwrap(); inner != nil {
			if stacker, ok := inner.(shared.Stacker); ok {
				if stack := stacker.Stack(); len(stack) > 0 {
					return stack
				}
			}
		}
	}

	// 兜底：捕获当前位置的堆栈（错误处理点）
	return captureStack(4) // skip: Callers, captureStack, extractStack, HandleAppError
}

// ============================================================================
// 成功响应函数
// ============================================================================

// HandleSuccess 处理成功响应 (200 OK)
func HandleSuccess(c *gin.Context, data interface{}, message string) {
	requestID := getRequestID(c)
	response := &Response{
		Success:   true,
		Data:      data,
		Message:   message,
		Code:      http.StatusOK,
		RequestID: requestID,
	}
	c.JSON(http.StatusOK, response)
}

// HandleCreated 处理创建成功响应 (201 Created)
func HandleCreated(c *gin.Context, data interface{}, message string) {
	requestID := getRequestID(c)
	response := &Response{
		Success:   true,
		Data:      data,
		Message:   message,
		Code:      http.StatusCreated,
		RequestID: requestID,
	}
	c.JSON(http.StatusCreated, response)
}

// HandleNoContent 处理无内容响应 (204 No Content)
func HandleNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// HandlePaginated 处理分页响应
func HandlePaginated(c *gin.Context, data interface{}, pagination Pagination, message string) {
	requestID := getRequestID(c)
	response := &PaginatedResponse{
		Success:    true,
		Data:       data,
		Pagination: pagination,
		Message:    message,
		Code:       http.StatusOK,
		RequestID:  requestID,
	}
	c.JSON(http.StatusOK, response)
}

// ============================================================================
// 兼容性函数 (Deprecated)
// ============================================================================

// NewSuccessResponse 创建成功响应
// Deprecated: 请使用 HandleSuccess
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Message: message,
		Code:    http.StatusOK,
	}
}

// NewErrorResponse 创建错误响应
// Deprecated: 请使用 HandleError 或 HandleAppError
func NewErrorResponse(err error, message string, code int) *Response {
	return &Response{
		Success: false,
		Error:   err.Error(),
		Message: message,
		Code:    code,
	}
}
