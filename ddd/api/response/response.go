package response

import (
	"net/http"

	"ddd/pkg/errors"

	"github.com/gin-gonic/gin"
)

// RequestIDKey context key for request id propagation
const RequestIDKey = "request_id"

// Response Common response structure
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
}

// NewSuccessResponse Create success response
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Message: message,
		Code:    http.StatusOK,
	}
}

// NewErrorResponse Create error response
func NewErrorResponse(err error, message string, code int) *Response {
	return &Response{
		Success: false,
		Error:   err.Error(),
		Message: message,
		Code:    code,
	}
}

// getRequestID Get request ID from context
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// HandleError Unified error handling
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

// HandleAppError Handle application error (automatically maps HTTP status code)
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

// HandleSuccess Unified success handling
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

// HandleCreated Create success handling
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

// HandleNoContent No content response
func HandleNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// PaginatedResponse Paginated response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Message    string      `json:"message"`
	Code       int         `json:"code"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Pagination Pagination information
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// HandlePaginated Paginated response handling
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
