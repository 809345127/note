package order

import (
	"net/http"

	"ddd/api/ctxutil"
	"ddd/api/response"
	orderapp "ddd/application/order"
	"ddd/pkg/errors"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	orderService *orderapp.ApplicationService
}

func NewController(orderService *orderapp.ApplicationService) *Controller {
	return &Controller{orderService: orderService}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/orders")
	orderGroup.POST("", c.CreateOrder)
	orderGroup.GET("/:id", c.GetOrder)
	orderGroup.GET("/user/:userId", c.GetUserOrders)
	orderGroup.PUT("/:id/status", c.UpdateOrderStatus)
	orderGroup.POST("/:id/process", c.ProcessOrder)
}

func (c *Controller) CreateOrder(ctx *gin.Context) {
	var req orderapp.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	resp, err := c.orderService.CreateOrder(ctxutil.WithRequestID(ctx), req)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleCreated(ctx, resp, "order created successfully")
}

func (c *Controller) GetOrder(ctx *gin.Context) {
	orderID, ok := requiredPathParam(ctx, "id", "order ID is required")
	if !ok {
		return
	}

	resp, err := c.orderService.GetOrder(ctxutil.WithRequestID(ctx), orderID)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, resp, "order retrieved successfully")
}

func (c *Controller) GetUserOrders(ctx *gin.Context) {
	userID, ok := requiredPathParam(ctx, "userId", "user ID is required")
	if !ok {
		return
	}

	resp, err := c.orderService.GetUserOrders(ctxutil.WithRequestID(ctx), userID)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, resp, "user orders retrieved successfully")
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (c *Controller) UpdateOrderStatus(ctx *gin.Context) {
	orderID, ok := requiredPathParam(ctx, "id", "order ID is required")
	if !ok {
		return
	}

	var req UpdateOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	err := c.orderService.UpdateOrderStatus(ctxutil.WithRequestID(ctx), orderapp.UpdateOrderStatusRequest{
		OrderID: orderID,
		Status:  req.Status,
	})
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, nil, "order status updated successfully")
}

func (c *Controller) ProcessOrder(ctx *gin.Context) {
	orderID, ok := requiredPathParam(ctx, "id", "order ID is required")
	if !ok {
		return
	}

	if err := c.orderService.ProcessOrder(ctxutil.WithRequestID(ctx), orderID); err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, nil, "order processed successfully")
}

func requiredPathParam(ctx *gin.Context, name, message string) (string, bool) {
	value := ctx.Param(name)
	if value == "" {
		response.HandleError(ctx, errors.BadRequest(message), message, http.StatusBadRequest)
		return "", false
	}
	return value, true
}
