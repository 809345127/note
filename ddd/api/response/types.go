package response

// RequestIDKey 是 gin context 中保存请求 ID 的键。
const RequestIDKey = "request_id"

// Response 是统一响应结构。
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
}

// PaginatedResponse 是分页响应结构。
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Message    string      `json:"message"`
	Code       int         `json:"code"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Pagination 表示分页信息。
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}
