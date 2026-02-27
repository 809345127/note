package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleSuccess(c *gin.Context, data interface{}, message string) {
	requestID := getRequestID(c)
	c.JSON(http.StatusOK, &Response{
		Success:   true,
		Data:      data,
		Message:   message,
		Code:      http.StatusOK,
		RequestID: requestID,
	})
}

func HandleCreated(c *gin.Context, data interface{}, message string) {
	requestID := getRequestID(c)
	c.JSON(http.StatusCreated, &Response{
		Success:   true,
		Data:      data,
		Message:   message,
		Code:      http.StatusCreated,
		RequestID: requestID,
	})
}

func HandleNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func HandlePaginated(c *gin.Context, data interface{}, pagination Pagination, message string) {
	requestID := getRequestID(c)
	c.JSON(http.StatusOK, &PaginatedResponse{
		Success:    true,
		Data:       data,
		Pagination: pagination,
		Message:    message,
		Code:       http.StatusOK,
		RequestID:  requestID,
	})
}

// NewSuccessResponse 已废弃，仅用于兼容旧调用。
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{Success: true, Data: data, Message: message, Code: http.StatusOK}
}

// NewErrorResponse 已废弃，仅用于兼容旧调用。
func NewErrorResponse(err error, message string, code int) *Response {
	return &Response{Success: false, Error: err.Error(), Message: message, Code: code}
}
