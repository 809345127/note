package user

import (
	"net/http"

	"ddd-example/api/response"
	userapp "ddd-example/application/user"

	"github.com/gin-gonic/gin"
)

// Controller User controller
type Controller struct {
	userService *userapp.ApplicationService
}

// NewController Create user controller
func NewController(userService *userapp.ApplicationService) *Controller {
	return &Controller{
		userService: userService,
	}
}

// RegisterRoutes Register user routes
func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	{
		userGroup.POST("", c.CreateUser)
		userGroup.GET("", c.GetAllUsers)
		userGroup.GET("/:id", c.GetUser)
		userGroup.PUT("/:id/status", c.UpdateUserStatus)
		userGroup.GET("/:id/total-spent", c.GetUserTotalSpent)
	}
}

// CreateUser Create user
func (c *Controller) CreateUser(ctx *gin.Context) {
	var req userapp.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	user, err := c.userService.CreateUser(ctx.Request.Context(), req)
	if err != nil {
		response.HandleError(ctx, err, "Failed to create user", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, user, "User created successfully")
}

// GetUser Get user information
func (c *Controller) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		response.HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	user, err := c.userService.GetUser(ctx.Request.Context(), userID)
	if err != nil {
		response.HandleError(ctx, err, "User not found", http.StatusNotFound)
		return
	}

	response.HandleSuccess(ctx, user, "User retrieved successfully")
}

// GetAllUsers Get all users
func (c *Controller) GetAllUsers(ctx *gin.Context) {
	users, err := c.userService.GetAllUsers()
	if err != nil {
		response.HandleError(ctx, err, "Failed to get users", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, users, "Users retrieved successfully")
}

// UpdateUserStatusRequest Update user status request
type UpdateUserStatusRequest struct {
	Active bool `json:"active"`
}

// UpdateUserStatus Update user status
func (c *Controller) UpdateUserStatus(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		response.HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateUserStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	updateReq := userapp.UpdateUserStatusRequest{
		UserID: userID,
		Active: req.Active,
	}

	if err := c.userService.UpdateUserStatus(ctx.Request.Context(), updateReq); err != nil {
		response.HandleError(ctx, err, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, nil, "User status updated successfully")
}

// GetUserTotalSpent Get user total spent
func (c *Controller) GetUserTotalSpent(ctx *gin.Context) {
	userID := ctx.Param("id")
	if userID == "" {
		response.HandleError(ctx, gin.Error{}, "User ID is required", http.StatusBadRequest)
		return
	}

	req := userapp.GetUserTotalSpentRequest{UserID: userID}
	resp, err := c.userService.GetUserTotalSpent(ctx.Request.Context(), req)
	if err != nil {
		response.HandleError(ctx, err, "Failed to get user total spent", http.StatusInternalServerError)
		return
	}

	response.HandleSuccess(ctx, resp, "User total spent retrieved successfully")
}
