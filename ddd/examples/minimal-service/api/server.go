package api

import (
	"errors"
	"net/http"

	"ddd/examples/minimal-service/application"
	"ddd/examples/minimal-service/domain"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewServer(taskService *application.TaskService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	apiV1 := r.Group("/api/v1")
	apiV1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	apiV1.POST("/tasks", func(c *gin.Context) {
		var req application.CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: err.Error()})
			return
		}

		resp, err := taskService.CreateTask(c.Request.Context(), req)
		if err != nil {
			if errors.Is(err, domain.ErrTaskTitleEmpty) {
				c.JSON(http.StatusBadRequest, ErrorResponse{Code: "invalid_task", Message: err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "internal_error", Message: "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, resp)
	})

	apiV1.GET("/tasks", func(c *gin.Context) {
		resp, err := taskService.ListTasks(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Code: "internal_error", Message: "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"items": resp})
	})

	return r
}
