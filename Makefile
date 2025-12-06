
# Shared Variables
include makefiles/variables/vars.mk
include makefiles/variables/common.mk


.PHONY: help build clean test docker-build docker-run install deps lint format vet mcp mcp-examples mcp-test mcp-client-test

help: ## Show this help message
	@echo "MongoDB Migration Tool - Available commands:"
	@echo
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: download-mockery
_download-mockery:
	@echo "Ensuring Mockery $(MOCKERY_VERSION) is installed..."
	@GOFLAGS=-toolchain=local go install github.com/vektra/mockery/v2@"$(MOCKERY_VERSION)"
	@echo "Mockery installation complete."

.PHONY: download-mockery
download-mockery:
	@echo "Ensuring Mockery $(MOCKERY_VERSION) is installed..."
	@GOFLAGS=-toolchain=local go install github.com/vektra/mockery/v2@"$(MOCKERY_VERSION)"
	@echo "Mockery installation complete."

deps: ## Download Go modules
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy

build: deps ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

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
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	go clean

test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v $(shell go list ./... | grep -v /examples)

test-library: ## Run library-specific tests
	@echo "$(GREEN)Running library tests...$(NC)"
	go test -v ./migration ./config

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out $(shell go list ./... | grep -v /examples)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-examples: ## Test the examples
	@echo "$(GREEN)Testing examples...$(NC)"
	@go build -o examples/example examples/main.go
	@go build -o examples/library-example/library-example examples/library-example/main.go
	@echo "✅ Examples build successfully!"
	@echo "  - CLI example: examples/example"
	@echo "  - Library example: examples/library-example/library-example"

lint: ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run ./config/... ./migration/... ./mcp/... ./cmd/...

format: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet $(shell go list ./... | grep -v /examples)

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

create-migration: ## Create a new migration (usage: make create-migration DESC="description")
ifndef DESC
	@echo "$(RED)Error: DESC is required. Usage: make create-migration DESC=\"your description\"$(NC)"
	@exit 1
endif
	@echo "$(GREEN)Creating migration: $(DESC)$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) create "$(DESC)"

migration-status: build ## Show migration status
	@echo "$(GREEN)Checking migration status...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) status

migration-up: build ## Run all pending migrations
	@echo "$(GREEN)Running migrations...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) up

migration-down: build ## Rollback migrations (usage: make migration-down VERSION="20231201_001")
ifndef VERSION
	@echo "$(RED)Error: VERSION is required. Usage: make migration-down VERSION=\"20231201_001\"$(NC)"
	@exit 1
endif
	@echo "$(YELLOW)Rolling back to version: $(VERSION)$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) down $(VERSION)

dev-setup: ## Set up development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin; \
	fi
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@echo "Development environment ready!"

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	./scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true ./scripts/deploy-migrations.sh auto

ci-test: deps vet lint test ## Run all CI tests
	@echo "$(GREEN)All CI tests passed!$(NC)"

ci-build: clean build-all test ## Build and test for CI
	@echo "$(GREEN)CI build completed!$(NC)"

release: clean ci-test build-all ## Create a release build
	@echo "$(GREEN)Release build completed!$(NC)"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

db-up: ## Start local MongoDB for testing
	@echo "$(GREEN)Starting local MongoDB...$(NC)"
	docker run --name mongo-migration-test -p 27017:27017 -d mongo:7.0 || \
	docker start mongo-migration-test

db-down: ## Stop local MongoDB
	@echo "$(YELLOW)Stopping local MongoDB...$(NC)"
	docker stop mongo-migration-test || true
	docker rm mongo-migration-test || true

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

tidy: ## Tidy Go modules for all services
	@echo "🧹 Tidying all workspace modules with Go $(GO_COMPAT_VERSION) compatibility..."
	go mod tidy -go=$(GO_COMPAT_VERSION) && cd $(REPO_ROOT);
	@echo "✅ All modules tidied"

version: ## Show version information
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD)"
	@echo "Build date: $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"

.PHONY: tidy

mcp: build ## Start MCP server for AI assistant integration
	@echo "$(GREEN)Starting MCP server...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp

mcp-examples: build ## Start MCP server with example migrations registered
	@echo "$(GREEN)Starting MCP server with examples...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-test: build ## Test MCP server with example request
	@echo "$(GREEN)Testing MCP server...$(NC)"
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-client-test: build ## Test MCP server interactively
	@echo "$(GREEN)Testing MCP server interactively (Ctrl+C to exit)...$(NC)"
	@echo "Try these commands:"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"migration_status\",\"arguments\":{}}}"
	@echo ""
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples
