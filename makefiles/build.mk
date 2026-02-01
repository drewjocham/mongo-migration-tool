THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

ALL_PACKAGES := $(shell cd $(REPO_ROOT) && go list ./...)
EXAMPLE_PACKAGES := $(shell cd $(REPO_ROOT) && go list ./examples/...)
TEST_PACKAGES := $(filter-out $(EXAMPLE_PACKAGES), $(ALL_PACKAGES))
include $(MAKEFILES_DIR)/variables/vars.mk

GO_ENV ?=
INTEGRATION_MONGO_PORT ?= 37017
COMPOSE_PROJECT_NAME ?= mm-it

.PHONY: build clean test install deps integration-test

clear-cache: ## Clear build cache is sometimes needed in the pipeline
	@$(GO_ENV) go clean -modcache
	@$(GO_ENV) go clean -cache

build: deps ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	cd $(REPO_ROOT) && $(GO_ENV) CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# --- Configuration ---
BINARY_NAME  := mongo-migration
BIN_DIR     := $(REPO_ROOT)/bin
MAIN_PACKAGE := ./cmd

.PHONY: build-all
build-all: deps ## Build for all supported platforms
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p $(BIN_DIR)
	@$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
		$(eval BINARY := $(BIN_DIR)/$(BINARY_NAME)-$(OS)-$(ARCH)$(if $(filter windows,$(OS)),.exe)) \
		echo "$(YELLOW)  > Building $(OS)/$(ARCH)...$(NC)"; \
		cd $(REPO_ROOT) && GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 \
		go build $(LDFLAGS) -o $(BINARY) $(MAIN_PACKAGE); \
	)
	@echo "$(GREEN)Done! Binaries are in $(BIN_DIR)$(NC)"

.PHONY: clean
clean: ## Remove build artifacts
	@echo "$(RED)Cleaning $(BIN_DIR)...$(NC)"
	@rm -rf $(BIN_DIR)

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
		docker-compose -f $(COMPOSE_FILE_INTEGRATION) build cli
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		docker-compose -f $(COMPOSE_FILE_INTEGRATION) up -d mongo
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		$(GO_ENV) go test -v -tags=integration ./integration; \
	status=$$?; \
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker-compose -f $(COMPOSE_FILE_INTEGRATION) down -v; \
	exit $$status

ci-build: clean build-all test ## Build and test for CI
	@echo "$(GREEN)CI build completed!$(NC)"
