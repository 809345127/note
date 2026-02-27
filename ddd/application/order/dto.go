package order

import "time"

// CreateOrderRequest 表示创建订单的入参。
type CreateOrderRequest struct {
	UserID string             `json:"user_id" binding:"required"`
	Items  []OrderItemRequest `json:"items" binding:"required,min=1"`
}

// OrderItemRequest 表示创建订单时的单个商品项。
type OrderItemRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	ProductName string `json:"product_name" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	UnitPrice   int64  `json:"unit_price" binding:"required,min=0"`
	Currency    string `json:"currency" binding:"required"`
}

// UpdateOrderStatusRequest 表示更新订单状态入参。
type UpdateOrderStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=PENDING CONFIRMED SHIPPED DELIVERED CANCELLED"`
	Reason  string `json:"reason"`
}

// OrderResponse 表示订单返回模型。
type OrderResponse struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Items       []OrderItemResponse `json:"items"`
	TotalAmount MoneyResponse       `json:"total_amount"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// OrderItemResponse 表示订单项返回模型。
type OrderItemResponse struct {
	ProductID   string        `json:"product_id"`
	ProductName string        `json:"product_name"`
	Quantity    int           `json:"quantity"`
	UnitPrice   MoneyResponse `json:"unit_price"`
	Subtotal    MoneyResponse `json:"subtotal"`
}

// MoneyResponse 表示金额返回模型。
type MoneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}
