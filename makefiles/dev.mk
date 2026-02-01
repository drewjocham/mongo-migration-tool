THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

.PHONY: dev-setup docs tidy version download-mockery
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

download-mockery:
	@echo "Ensuring Mockery $(MOCKERY_VERSION) is installed..."
	@GOFLAGS=-toolchain=local go install github.com/vektra/mockery/v2@"$(MOCKERY_VERSION)"
	@echo "Mockery installation complete."
