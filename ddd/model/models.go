package model

import (
	"time"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserModel 用户数据模型（用于数据库或外部存储）
type UserModel struct {
	BaseModel
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	IsActive bool   `json:"is_active"`
}

// OrderModel 订单数据模型
type OrderModel struct {
	BaseModel
	UserID      string       `json:"user_id"`
	TotalAmount int64        `json:"total_amount"`
	Currency    string       `json:"currency"`
	Status      string       `json:"status"`
	Items       []OrderItemModel `json:"items"`
}

// OrderItemModel 订单项数据模型
type OrderItemModel struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Currency    string `json:"currency"`
	Subtotal    int64  `json:"subtotal"`
}

// PaginationModel 分页数据模型
type PaginationModel struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Pages    int `json:"pages"`
}

// ApiResponseModel API响应数据模型
type ApiResponseModel struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
}

// ErrorModel 错误数据模型
type ErrorModel struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// ValidationErrorModel 验证错误数据模型
type ValidationErrorModel struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}