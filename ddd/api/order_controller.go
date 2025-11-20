package api

import (
	"ddd-example/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OrderController 订单控制器
type OrderController struct {
	orderService *service.OrderApplicationService
}

// NewOrderController 创建订单控制器
func NewOrderController(orderService *service.OrderApplicationService) *OrderController {
	return &OrderController{
		orderService: orderService,
	}
}

// RegisterRoutes 注册订单路由
func (c *OrderController) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/orders")
	{
		orderGroup.POST("", c.CreateOrder)
		orderGroup.GET("/:id", c.GetOrder)
		orderGroup.GET("/user/:userId", c.GetUserOrders)
		orderGroup.PUT("/:id/status", c.UpdateOrderStatus)
		orderGroup.POST("/:id/process", c.ProcessOrder)
	}
}

// CreateOrder 创建订单
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var req service.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}
	
	order, err := c.orderService.CreateOrder(req)
	if err != nil {
		HandleError(ctx, err, "Failed to create order", http.StatusInternalServerError)
		return
	}
	
	HandleSuccess(ctx, order, "Order created successfully")
}

// GetOrder 获取订单信息
func (c *OrderController) GetOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}
	
	order, err := c.orderService.GetOrder(orderID)
	if err != nil {
		HandleError(ctx, err, "Order not found", http.StatusNotFound)
		return
	}
	
	HandleSuccess(ctx, order, "Order retrieved successfully")
}

// GetUserOrders 获取用户的所有订单
func (c *OrderController) GetUserOrders(ctx *gin.Context) {
	userID := ctx.Param("userId")
	if userID == "" {
		HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}
	
	orders, err := c.orderService.GetUserOrders(userID)
	if err != nil {
		HandleError(ctx, err, "Failed to get user orders", http.StatusInternalServerError)
		return
	}
	
	HandleSuccess(ctx, orders, "User orders retrieved successfully")
}

// UpdateOrderStatusRequest 更新订单状态请求
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateOrderStatus 更新订单状态
func (c *OrderController) UpdateOrderStatus(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}
	
	var req UpdateOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}
	
	updateReq := service.UpdateOrderStatusRequest{
		OrderID: orderID,
		Status:  req.Status,
	}
	
	if err := c.orderService.UpdateOrderStatus(updateReq); err != nil {
		HandleError(ctx, err, "Failed to update order status", http.StatusInternalServerError)
		return
	}
	
	HandleSuccess(ctx, nil, "Order status updated successfully")
}

// ProcessOrder 处理订单
func (c *OrderController) ProcessOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}
	
	if err := c.orderService.ProcessOrder(orderID); err != nil {
		HandleError(ctx, err, "Failed to process order", http.StatusInternalServerError)
		return
	}
	
	HandleSuccess(ctx, nil, "Order processed successfully")
}