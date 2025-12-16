# Calculate the path to the variables file dynamically
BUILD_MK_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(abspath $(dir $(BUILD_MK_PATH)))
include $(MAKEFILES_DIR)/variables/vars.mk

ALL_PACKAGES := $(shell go list $(REPO_ROOT)/...)
EXAMPLE_PACKAGES := $(shell go list $(REPO_ROOT)/examples/...)
TEST_PACKAGES := $(filter-out $(EXAMPLE_PACKAGES), $(ALL_PACKAGES))

GO_ENV ?= GOWORK=off
INTEGRATION_MONGO_PORT ?= 37017
COMPOSE_PROJECT_NAME ?= mm-it

.PHONY: build clean test install deps integration-test

clear-cache: ## Clear build cache is sometimes needed in the pipeline
	@$(GO_ENV) go clean -modcache # Clear module cache
	@$(GO_ENV) go clean -cache # Clear build cache

build: deps ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO_ENV) CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(REPO_ROOT)

build-all: deps ## Build for multiple platforms
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
	# Linux amd64
	$(GO_ENV) GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(REPO_ROOT)
	# Linux arm64
	$(GO_ENV) GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(REPO_ROOT)
	# macOS amd64
	$(GO_ENV) GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(REPO_ROOT)
	# macOS arm64
	$(GO_ENV) GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(REPO_ROOT)
	# Windows amd64
	$(GO_ENV) GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(REPO_ROOT)

install: build ## Install the binary to GOBIN
	@echo "$(GREEN)Installing $(BINARY_NAME) to $(GOBIN)...$(NC)"
	@mkdir -p $(GOBIN)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	$(GO_ENV) go clean

deps: ## Download Go modules
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go mod download
	cd $(REPO_ROOT) && $(GO_ENV) go mod tidy

test: ## Run tests for all non-example packages
	@echo "$(GREEN)Running tests...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v $(TEST_PACKAGES)

test-library: ## Run library-specific tests
	@echo "$(GREEN)Running library tests...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v ./migration ./config

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v -coverprofile=coverage.out $(TEST_PACKAGES)
	cd $(REPO_ROOT) && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-examples: ## Test the examples
	@echo "$(GREEN)Testing examples...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go build -o examples/example examples/main.go
	cd $(REPO_ROOT) && $(GO_ENV) go build -o examples/library-example/library-example examples/library-example/main.go
	@echo "✅ Examples build successfully!"
	@echo "  - CLI example: examples/example"
	@echo "  - Library example: examples/library-example/library-example"

integration-test: ## Run Docker-based CLI integration tests via docker compose
	@echo "$(GREEN)Running CLI integration tests with docker compose...$(NC)"
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker-compose -f integration-compose.yml build cli
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker-compose -f integration-compose.yml up -d mongo
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		$(GO_ENV) go test -v -tags=integration ./integration; \
	status=$$?; \
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker-compose -f integration-compose.yml down -v; \
	exit $$status

ci-build: clean build-all test ## Build and test for CI
	@echo "$(GREEN)CI build completed!$(NC)"


