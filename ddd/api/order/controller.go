/*
Package order - 订单 API 控制器

职责:
1. 接收 HTTP 请求，解析参数
2. 调用应用服务处理业务逻辑
3. 使用 response 包统一处理响应和错误

错误处理原则:
1. 参数绑定错误: 使用 response.HandleError 直接返回 400
2. 业务错误: 使用 response.HandleAppError 自动映射状态码
3. HandleAppError 会自动调用 errors.FromDomainError 转换错误
*/
package order

import (
	"net/http"

	"ddd/api/response"
	orderapp "ddd/application/order"
	"ddd/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Controller 订单控制器
type Controller struct {
	orderService *orderapp.ApplicationService
}

// NewController 创建订单控制器
func NewController(orderService *orderapp.ApplicationService) *Controller {
	return &Controller{
		orderService: orderService,
	}
}

// RegisterRoutes 注册订单路由
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

// CreateOrder 创建订单
// POST /api/v1/orders
func (c *Controller) CreateOrder(ctx *gin.Context) {
	var req orderapp.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 参数绑定错误: 直接返回 400
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	order, err := c.orderService.CreateOrder(ctx.Request.Context(), req)
	if err != nil {
		// 业务错误: HandleAppError 自动处理错误转换和状态码映射
		response.HandleAppError(ctx, err)
		return
	}

	// 创建成功返回 201
	response.HandleCreated(ctx, order, "order created successfully")
}

// GetOrder 获取订单信息
// GET /api/v1/orders/:id
//
// 错误处理示例（完整链路）:
//
//	Repository 返回: order.ErrOrderNotFound
//	     ↓
//	Service 直接传递: order.ErrOrderNotFound
//	     ↓
//	Controller 调用: response.HandleAppError(ctx, err)
//	     ↓
//	HandleAppError 内部:
//	  1. errors.FromDomainError(err) 转换为 AppError{Code: ORDER_NOT_FOUND}
//	  2. mapErrorCodeToHTTPStatus() 映射为 404
//	  3. 记录完整错误日志（含调用栈）
//	  4. 返回安全的 JSON 响应给客户端
func (c *Controller) GetOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		// 参数缺失: 直接返回 400
		response.HandleError(ctx, errors.BadRequest("order ID is required"), "order ID is required", http.StatusBadRequest)
		return
	}

	order, err := c.orderService.GetOrder(ctx.Request.Context(), orderID)
	if err != nil {
		// 业���错误: 自动映射
		// - order.ErrOrderNotFound -> 404
		// - 其他错误 -> 500
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, order, "order retrieved successfully")
}

// GetUserOrders 获取用户的所有订单
// GET /api/v1/orders/user/:userId
func (c *Controller) GetUserOrders(ctx *gin.Context) {
	userID := ctx.Param("userId")
	if userID == "" {
		response.HandleError(ctx, errors.BadRequest("user ID is required"), "user ID is required", http.StatusBadRequest)
		return
	}

	orders, err := c.orderService.GetUserOrders(ctx.Request.Context(), userID)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, orders, "user orders retrieved successfully")
}

// UpdateOrderStatusRequest 更新订单状态请求
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateOrderStatus 更新订单状态
// PUT /api/v1/orders/:id/status
func (c *Controller) UpdateOrderStatus(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		response.HandleError(ctx, errors.BadRequest("order ID is required"), "order ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	updateReq := orderapp.UpdateOrderStatusRequest{
		OrderID: orderID,
		Status:  req.Status,
	}

	if err := c.orderService.UpdateOrderStatus(ctx.Request.Context(), updateReq); err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, nil, "order status updated successfully")
}

// ProcessOrder 处理订单
// POST /api/v1/orders/:id/process
func (c *Controller) ProcessOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		response.HandleError(ctx, errors.BadRequest("order ID is required"), "order ID is required", http.StatusBadRequest)
		return
	}

	if err := c.orderService.ProcessOrder(ctx.Request.Context(), orderID); err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, nil, "order processed successfully")
}
