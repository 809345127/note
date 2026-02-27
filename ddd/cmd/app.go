package cmd

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ddd/api"
	"ddd/config"
	"ddd/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App 表示可运行的 HTTP 应用。
type App struct {
	config *config.Config
	router *api.Router
	server *http.Server
	db     *gorm.DB
}

// NewApp 为兼容旧调用保留，内部统一走 Builder。
func NewApp(cfg *config.Config) *App {
	return NewBuilder(cfg).Build()
}

// Run 启动服务并在收到退出信号后优雅关闭。
func (a *App) Run() error {
	a.startHTTPServer()
	a.waitForShutdownSignal()

	logger.Info("Shutting down server...")

	if err := a.shutdownHTTPServer(); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	a.closeDatabase()

	logger.Info("Server exited properly")
	return nil
}

func (a *App) startHTTPServer() {
	go func() {
		logger.Info("Server started",
			zap.String("port", a.config.Server.Port),
			zap.String("health", "http://localhost:"+a.config.Server.Port+"/api/v1/health"))

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()
}

func (a *App) waitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func (a *App) shutdownHTTPServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Server.ShutdownTimeout)
	defer cancel()
	return a.server.Shutdown(ctx)
}

func (a *App) closeDatabase() {
	if a.db == nil {
		return
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error("Error closing database connection", zap.Error(err))
	}
}

func (a *App) GetServer() *http.Server {
	return a.server
}
