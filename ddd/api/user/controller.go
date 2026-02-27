package user

import (
	"net/http"

	"ddd/api/ctxutil"
	"ddd/api/response"
	userapp "ddd/application/user"
	"ddd/pkg/errors"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	userService *userapp.ApplicationService
}

func NewController(userService *userapp.ApplicationService) *Controller {
	return &Controller{userService: userService}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	userGroup := router.Group("/users")
	userGroup.POST("", c.CreateUser)
	userGroup.GET("/:id", c.GetUser)
	userGroup.PUT("/:id/status", c.UpdateUserStatus)
	userGroup.GET("/:id/total-spent", c.GetUserTotalSpent)
}

func (c *Controller) CreateUser(ctx *gin.Context) {
	var req userapp.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	user, err := c.userService.CreateUser(ctxutil.WithRequestID(ctx), req)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, user, "user created successfully")
}

func (c *Controller) GetUser(ctx *gin.Context) {
	userID, ok := requiredPathParam(ctx, "id", "user ID is required")
	if !ok {
		return
	}

	user, err := c.userService.GetUser(ctxutil.WithRequestID(ctx), userID)
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, user, "user retrieved successfully")
}

type UpdateUserStatusRequest struct {
	Active bool `json:"active"`
}

func (c *Controller) UpdateUserStatus(ctx *gin.Context) {
	userID, ok := requiredPathParam(ctx, "id", "user ID is required")
	if !ok {
		return
	}

	var req UpdateUserStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.HandleError(ctx, err, "invalid request parameters", http.StatusBadRequest)
		return
	}

	err := c.userService.UpdateUserStatus(ctxutil.WithRequestID(ctx), userapp.UpdateUserStatusRequest{
		UserID: userID,
		Active: req.Active,
	})
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, nil, "user status updated successfully")
}

func (c *Controller) GetUserTotalSpent(ctx *gin.Context) {
	userID, ok := requiredPathParam(ctx, "id", "user ID is required")
	if !ok {
		return
	}

	resp, err := c.userService.GetUserTotalSpent(ctxutil.WithRequestID(ctx), userapp.GetUserTotalSpentRequest{UserID: userID})
	if err != nil {
		response.HandleAppError(ctx, err)
		return
	}

	response.HandleSuccess(ctx, resp, "user total spent retrieved successfully")
}

func requiredPathParam(ctx *gin.Context, name, message string) (string, bool) {
	value := ctx.Param(name)
	if value == "" {
		response.HandleError(ctx, errors.BadRequest(message), message, http.StatusBadRequest)
		return "", false
	}
	return value, true
}
