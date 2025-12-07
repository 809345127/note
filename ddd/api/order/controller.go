package order

import (
	"net/http"

	"ddd-example/api/response"
	orderapp "ddd-example/application/order"

	"github.com/gin-gonic/gin"
)

// Controller Order controller
type Controller struct {
	orderService *orderapp.ApplicationService
}

// NewController Create order controller
func NewController(orderService *orderapp.ApplicationService) *Controller {
	return &Controller{
		orderService: orderService,
	}
}

// RegisterRoutes Register order routes
func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/orders")
	{
		orderGroup.POST("", c.CreateOrder)
		orderGroup.GET("/:id", c.GetOrder)
		orderGroup.GET("/user/:userId", c.GetUserOrders)
		orderGroup.PUT("/:id/status", c.UpdateOrderStatus)
		orderGroup.POST("/:id/process", c.ProcessOrder)
	}
}

// CreateOrder Create order
func (c *Controller) CreateOrder(ctx *gin.Context) {
	var req orderapp.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	order, err := c.orderService.CreateOrder(ctx.Request.Context(), req)
	if err != nil {
		response.HandleError(ctx, err, "Failed to create order", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, order, "Order created successfully")
}

// GetOrder Get order information
func (c *Controller) GetOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		response.HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}

	order, err := c.orderService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		response.HandleError(ctx, err, "Order not found", http.StatusNotFound)
		return
	}

	response.HandleSuccess(ctx, order, "Order retrieved successfully")
}

// GetUserOrders Get all orders for a user
func (c *Controller) GetUserOrders(ctx *gin.Context) {
	userID := ctx.Param("userId")
	if userID == "" {
		response.HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	orders, err := c.orderService.GetUserOrders(ctx.Request.Context(), userID)
	if err != nil {
		response.HandleError(ctx, err, "Failed to get user orders", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, orders, "User orders retrieved successfully")
}

// UpdateOrderStatusRequest Update order status request
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateOrderStatus Update order status
func (c *Controller) UpdateOrderStatus(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		response.HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	updateReq := orderapp.UpdateOrderStatusRequest{
		OrderID: orderID,
		Status:  req.Status,
	}

	if err := c.orderService.UpdateOrderStatus(ctx.Request.Context(), updateReq); err != nil {
		response.HandleError(ctx, err, "Failed to update order status", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, nil, "Order status updated successfully")
}

// ProcessOrder Process order
func (c *Controller) ProcessOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		response.HandleError(ctx, gin.Error{}, "Order ID is required", http.StatusBadRequest)
		return
	}

	if err := c.orderService.ProcessOrder(ctx.Request.Context(), orderID); err != nil {
		response.HandleError(ctx, err, "Failed to process order", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, nil, "Order processed successfully")
}
