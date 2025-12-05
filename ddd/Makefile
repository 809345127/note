# DDD Example Application Makefile
# 用于简化开发和部署流程

.PHONY: help build run test clean docker-build docker-run docker-down lint fmt tidy migrate

# 变量定义
APP_NAME := ddd-example
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 默认目标
.DEFAULT_GOAL := help

## help: 显示帮助信息
help:
	@echo "DDD Example Application"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: 构建应用程序
build: ## 构建应用程序
	@echo "Building $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) .

## run: 运行应用程序
run: ## 运行应用程序
	go run .

## run-mysql: 使用MySQL运行应用程序
run-mysql: ## 使用MySQL运行
	DDD_DATABASE_TYPE=mysql go run .

## test: 运行测试
test: ## 运行所有测试
	go test -v -race -cover ./...

## test-coverage: 生成测试覆盖率报告
test-coverage: ## 生成覆盖率报告
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: 代码检查
lint: ## 运行代码检查
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: 格式化代码
fmt: ## 格式化代码
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## tidy: 整理依赖
tidy: ## 整理go.mod依赖
	go mod tidy

## clean: 清理构建产物
clean: ## 清理构建产物
	rm -rf bin/
	rm -f coverage.out coverage.html

## docker-build: 构建Docker镜像
docker-build: ## 构建Docker镜像
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

## docker-run: 运行Docker容器
docker-run: ## 运行Docker容器
	docker run -p 8080:8080 --name $(APP_NAME) $(APP_NAME):latest

## docker-compose-up: 启动所有服务（包括MySQL）
docker-compose-up: ## 启动Docker Compose服务
	docker-compose up -d

## docker-compose-down: 停止所有服务
docker-compose-down: ## 停止Docker Compose服务
	docker-compose down

## docker-compose-logs: 查看服务日志
docker-compose-logs: ## 查看Docker Compose日志
	docker-compose logs -f

## migrate: 运行数据库迁移（开发环境自动迁移）
migrate: ## 运行数据库迁移
	@echo "Note: Auto migration runs in development mode"
	@echo "For production, use a proper migration tool like golang-migrate"

## dev: 开发模式（热重载）
dev: ## 开发模式运行（需要air）
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to normal run..."; \
		go run .; \
	fi

## install-tools: 安装开发工具
install-tools: ## 安装开发工具
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest

## version: 显示版本信息
version: ## 显示版本信息
	@echo "App: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go: $(GO_VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
