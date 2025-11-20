package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
}

// Response 通用响应结构
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
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

// HandleError 统一错误处理
func HandleError(c *gin.Context, err error, message string, code int) {
	response := NewErrorResponse(err, message, code)
	c.JSON(code, response)
}

// HandleSuccess 统一成功处理
func HandleSuccess(c *gin.Context, data interface{}, message string) {
	response := NewSuccessResponse(data, message)
	c.JSON(http.StatusOK, response)
}