ALL_PACKAGES := $(shell go list ./...)
EXAMPLE_PACKAGES := $(shell go list ./examples/...)
TEST_PACKAGES := $(filter-out $(EXAMPLE_PACKAGES), $(ALL_PACKAGES))

.PHONY: build clean test install deps

clear-cache: ## Clear build cache is sometimes needed in the pipeline
	@go clean -modcache # Clear module cache
	@go clean -cache # Clear build cache

build: deps ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-all: deps ## Build for multiple platforms
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
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

install: build ## Install the binary to GOBIN
	@echo "$(GREEN)Installing $(BINARY_NAME) to $(GOBIN)...$(NC)"
	@mkdir -p $(GOBIN)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	go clean

deps: ## Download Go modules
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy

test: ## Run tests for all non-example packages
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v $(TEST_PACKAGES)

test-library: ## Run library-specific tests
	@echo "$(GREEN)Running library tests...$(NC)"
	go test -v ./migration ./config

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out $(TEST_PACKAGES)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-examples: ## Test the examples
	@echo "$(GREEN)Testing examples...$(NC)"
	@go build -o examples/example examples/main.go
	@go build -o examples/library-example/library-example examples/library-example/main.go
	@echo "✅ Examples build successfully!"
	@echo "  - CLI example: examples/example"
	@echo "  - Library example: examples/library-example/library-example"

ci-build: clean build-all test ## Build and test for CI
	@echo "$(GREEN)CI build completed!$(NC)"
