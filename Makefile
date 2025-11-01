BINARY_NAME=mongo-essential
DOCKER_IMAGE=mongo-migration-tool
DOCKER_TAG?=latest
GO_VERSION=1.24.0

BUILD_DIR=./build
LOCAL_BINARY=$(BUILD_DIR)/$(BINARY_NAME) # Standardized local binary path
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)"

GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Define packages to include/exclude for unit tests
UNIT_TEST_PKGS=$(shell go list ./... | grep -v /examples | grep -v /test/integration)

.PHONY: help build clean test docker-build docker-run install deps lint format vet dev-setup install-tools migration-status migration-up migration-down

help: ## Show this help message
	@echo "MongoDB Migration Tool - Available commands:"
	@echo
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

#---

## 🛠️ Setup and Build

deps: ## Download Go modules
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy

build: deps ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(LOCAL_BINARY) .

build-all: deps ## Build for multiple platforms
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	mkdir -p $(BUILD_DIR)
	# Linux amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	# Linux arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	# macOS amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	# macOS arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

install: build ## Install the binary to GOPATH/bin
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	cp $(LOCAL_BINARY) $(GOPATH)/bin/$(BINARY_NAME)

clean: ## Clean build artifacts and test cache
	@echo "$(YELLOW)Cleaning build artifacts and test cache...$(NC)"
	rm -rf $(BUILD_DIR)
	go clean -testcache
	rm -f coverage.out coverage.html
	go clean

## 🧪 Testing

test: ## Run unit tests
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test -v -short $(UNIT_TEST_PKGS)

test-unit: test ## Alias for test

test-integration: ## Run integration tests (requires MongoDB)
	@echo "$(GREEN)Running integration tests...$(NC)"
	@echo "$(YELLOW)Note: This requires MongoDB running on localhost:27017$(NC)"
	go test -v -tags=integration ./test/integration/...

test-integration-docker: ## Run integration tests in Docker
	@echo "$(GREEN)Running integration tests in Docker...$(NC)"
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down -v

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "$(GREEN)All tests completed!$(NC)"

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -short -coverprofile=coverage.out -covermode=atomic $(UNIT_TEST_PKGS)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "$(GREEN)Coverage summary:$(NC)"
	@go tool cover -func=coverage.out | tail -1

test-coverage-full: ## Run all tests with coverage (including integration)
	@echo "$(GREEN)Running all tests with coverage...$(NC)"
	@echo "$(YELLOW)Note: This requires MongoDB running on localhost:27017$(NC)"
	go test -v -tags=integration -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "$(GREEN)Coverage summary:$(NC)"
	@go tool cover -func=coverage.out | tail -1

test-race: ## Run tests with race detector
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	go test -v -short -race $(UNIT_TEST_PKGS)

#---

## 🧹 Quality and Formatting

install-tools: ## Install necessary Go tools (lint, formatters, etc.)
	@echo "$(GREEN)Installing Go tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "Go tools installed."

format: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	gofmt -s -w .
	goimports -w .

lint-fix: format ## Format code and attempt to fix linting issues
	@echo "$(GREEN)Attempting to fix linting issues...$(NC)"
	golangci-lint run --fix ./...

lint: ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run ./config/... ./migration/... ./mcp/... ./cmd/...

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet $(shell go list ./... | grep -v /examples)

# ---

## 🐳 Docker

docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build ## Run Docker container
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -it \
		-e MONGO_URL=mongodb://host.docker.internal:27017 \
		-e MONGO_DATABASE=test_db \
		$(DOCKER_IMAGE):$(DOCKER_TAG) status

docker-compose-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(NC)"
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "$(YELLOW)Stopping services with docker-compose...$(NC)"
	docker-compose down

#---

## 🚀 Migration & Development

create-migration: build ## Create a new migration (usage: make create-migration DESC="description")
ifndef DESC
	@echo "$(RED)Error: DESC is required. Usage: make create-migration DESC=\"your description\"$(NC)"
	@exit 1
endif
	@echo "$(GREEN)Creating migration: $(DESC)$(NC)"
	./$(LOCAL_BINARY) create "$(DESC)"

migration-status: build ## Show migration status
	@echo "$(GREEN)Checking migration status...$(NC)"
	./$(LOCAL_BINARY) status

migration-up: build ## Run all pending migrations
	@echo "$(GREEN)Running migrations...$(NC)"
	./$(LOCAL_BINARY) up

migration-down: build ## Rollback migrations (usage: make migration-down VERSION="20231201_001")
ifndef VERSION
	@echo "$(RED)Error: VERSION is required. Usage: make migration-down VERSION=\"20231201_001\"$(NC)"
	@exit 1
endif
	@echo "$(YELLOW)Rolling back to version: $(VERSION)$(NC)"
	./$(LOCAL_BINARY) down $(VERSION)

dev-setup: install-tools ## Set up development environment
	@echo "$(GREEN)Development environment ready!$(NC)"

db-up: ## Start local MongoDB for testing
	@echo "$(GREEN)Starting local MongoDB...$(NC)"
	docker run --name mongo-migration-test -p 27017:27017 -d mongo:7.0 || \
	docker start mongo-migration-test

db-down: ## Stop local MongoDB
	@echo "$(YELLOW)Stopping local MongoDB...$(NC)"
	docker stop mongo-migration-test || true
	docker rm mongo-migration-test || true

ci-test: deps vet test-coverage test-race ## Run all CI tests
	@echo "$(GREEN)All CI tests passed!$(NC)"

ci-test-full: deps vet lint test-coverage-full test-race ## Run all CI tests with linting and integration
	@echo "$(GREEN)All CI tests passed!$(NC)"

release: clean ci-test build-all ## Create a release build
	@echo "$(GREEN)Release build completed!$(NC)"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

version: ## Show version information
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD)"
	@echo "Build date: $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"

# Remaining targets (mcp, deploy, etc.) retain their original logic but use the updated variables.

mcp: build ## Start MCP server for AI assistant integration
	@echo "$(GREEN)Starting MCP server...$(NC)"
	./$(LOCAL_BINARY) mcp

mcp-examples: build ## Start MCP server with example migrations registered
	@echo "$(GREEN)Starting MCP server with examples...$(NC)"
	./$(LOCAL_BINARY) mcp --with-examples

mcp-test: build ## Test MCP server with example request
	@echo "$(GREEN)Testing MCP server...$(NC)"
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./$(LOCAL_BINARY) mcp --with-examples

mcp-client-test: build ## Test MCP server interactively
	@echo "$(GREEN)Testing MCP server interactively (Ctrl+C to exit)...$(NC)"
	@echo "Try these commands:"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"migration_status\",\"arguments\":{}}}"
	@echo ""
	./$(LOCAL_BINARY) mcp --with-examples

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	./scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true ./scripts/deploy-migrations.sh auto

security-scan: ## Run security scan on Docker image
	@echo "$(GREEN)Running security scan...$(NC)"
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/src aquasec/trivy image $(DOCKER_IMAGE):$(DOCKER_TAG)

docs: ## Generate documentation
	@echo "$(GREEN)Generating documentation...$(NC)"
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "Install godoc with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi
