package api

import (
	"net/http"

	"ddd-example/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 通用响应结构
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Message: message,
		Code:    http.StatusOK,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err error, message string, code int) *Response {
	return &Response{
		Success: false,
		Error:   err.Error(),
		Message: message,
		Code:    code,
	}
}

// getRequestID 从上下文获取请求ID
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// HandleError 统一错误处理
func HandleError(c *gin.Context, err error, message string, code int) {
	requestID := getRequestID(c)
	response := &Response{
		Success:   false,
		Error:     err.Error(),
		Message:   message,
		Code:      code,
		RequestID: requestID,
	}
	c.JSON(code, response)
}

// HandleAppError 处理应用错误（自动映射HTTP状态码）
func HandleAppError(c *gin.Context, err error) {
	requestID := getRequestID(c)
	appErr := errors.MapDomainError(err)

	response := &Response{
		Success:   false,
		Error:     string(appErr.Code),
		Message:   appErr.Message,
		Code:      appErr.HTTPStatusCode(),
		RequestID: requestID,
	}
	c.JSON(appErr.HTTPStatusCode(), response)
}

// HandleSuccess 统一成功处理
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

// HandleCreated 创建成功处理
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

// HandleNoContent 无内容响应
func HandleNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
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

// HandlePaginated 分页响应处理
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
