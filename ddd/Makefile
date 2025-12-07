# DDD Example Application Makefile
# For simplifying development and deployment processes

.PHONY: help build run test clean docker-build docker-run docker-down lint fmt tidy migrate

# Variable definitions
APP_NAME := ddd-example
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := help

## help: Show help information
help:
	@echo "DDD Example Application"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Build application
build: ## Build application
	@echo "Building $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) .

## run: Run application
run: ## Run application
	go run .

## run-mysql: Run application with MySQL
run-mysql: ## Run with MySQL
	DDD_DATABASE_TYPE=mysql go run .

## test: Run tests
test: ## Run all tests
	go test -v -race -cover ./...

## test-coverage: Generate test coverage report
test-coverage: ## Generate coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Code linting
lint: ## Run code linting
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format code
fmt: ## Format code
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## tidy: Tidy dependencies
tidy: ## Tidy go.mod dependencies
	go mod tidy

## clean: Clean build artifacts
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

## docker-build: Build Docker image
docker-build: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

## docker-run: Run Docker container
docker-run: ## Run Docker container
	docker run -p 8080:8080 --name $(APP_NAME) $(APP_NAME):latest

## docker-compose-up: Start all services (including MySQL)
docker-compose-up: ## Start Docker Compose services
	docker-compose up -d

## docker-compose-down: Stop all services
docker-compose-down: ## Stop Docker Compose services
	docker-compose down

## docker-compose-logs: View service logs
docker-compose-logs: ## View Docker Compose logs
	docker-compose logs -f

## migrate: Run database migration (auto-migration in development)
migrate: ## Run database migration
	@echo "Note: Auto migration runs in development mode"
	@echo "For production, use a proper migration tool like golang-migrate"

## dev: Development mode (hot reload)
dev: ## Run in development mode (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to normal run..."; \
		go run .; \
	fi

## install-tools: Install development tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest

## version: Show version information
version: ## Show version information
	@echo "App: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go: $(GO_VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
