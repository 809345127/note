package api

import (
	"ddd-example/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	userService *service.UserApplicationService
}

// NewUserController 创建用户控制器
func NewUserController(userService *service.UserApplicationService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// RegisterRoutes 注册用户路由
func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	{
		userGroup.POST("", c.CreateUser)
		userGroup.GET("", c.GetAllUsers)
		userGroup.GET("/:id", c.GetUser)
		userGroup.PUT("/:id/status", c.UpdateUserStatus)
		userGroup.GET("/:id/total-spent", c.GetUserTotalSpent)
	}
}

// CreateUser 创建用户
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req service.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	user, err := c.userService.CreateUser(ctx.Request.Context(), req)
	if err != nil {
		HandleError(ctx, err, "Failed to create user", http.StatusInternalServerError)
		return
	}

	HandleSuccess(ctx, user, "User created successfully")
}

// GetUser 获取用户信息
func (c *UserController) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	user, err := c.userService.GetUser(ctx.Request.Context(), userID)
	if err != nil {
		HandleError(ctx, err, "User not found", http.StatusNotFound)
		return
	}

	HandleSuccess(ctx, user, "User retrieved successfully")
}

// GetAllUsers 获取所有用户
func (c *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := c.userService.GetAllUsers()
	if err != nil {
		HandleError(ctx, err, "Failed to get users", http.StatusInternalServerError)
		return
	}
	
	HandleSuccess(ctx, users, "Users retrieved successfully")
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Active bool `json:"active"`
}

// UpdateUserStatus 更新用户状态
func (c *UserController) UpdateUserStatus(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}
	
	var req UpdateUserStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}
	
	updateReq := service.UpdateUserStatusRequest{
		UserID: userID,
		Active: req.Active,
	}

	if err := c.userService.UpdateUserStatus(ctx.Request.Context(), updateReq); err != nil {
		HandleError(ctx, err, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	HandleSuccess(ctx, nil, "User status updated successfully")
}

// GetUserTotalSpent 获取用户总消费
func (c *UserController) GetUserTotalSpent(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	req := service.GetUserTotalSpentRequest{UserID: userID}
	response, err := c.userService.GetUserTotalSpent(ctx.Request.Context(), req)
	if err != nil {
		HandleError(ctx, err, "Failed to get user total spent", http.StatusInternalServerError)
		return
	}

	HandleSuccess(ctx, response, "User total spent retrieved successfully")
}