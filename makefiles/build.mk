ifndef MAKEFILES_BUILD_MK
MAKEFILES_BUILD_MK := 1

THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

ALL_PACKAGES := $(shell cd $(REPO_ROOT) && go list ./...)
include $(MAKEFILES_DIR)/variables/vars.mk

GO_ENV ?=

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


install: build ## Install the binary to GOBIN
	@echo "$(GREEN)Installing $(BINARY_NAME) to $(GOBIN)...$(NC)"
	@mkdir -p $(GOBIN)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	$(GO_ENV) go clean

deps: ## Download Go modules
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go mod download
	cd $(REPO_ROOT) && $(GO_ENV) go mod tidy


ci-build: clean build-all test ## Build and test for CI
	@echo "$(GREEN)CI build completed!$(NC)"

endif # MAKEFILES_BUILD_MK
