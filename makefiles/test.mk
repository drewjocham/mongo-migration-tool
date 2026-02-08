THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

ALL_PACKAGES := $(shell cd $(REPO_ROOT) && go list ./...)
EXAMPLE_PACKAGES := $(shell cd $(REPO_ROOT) && go list ./examples/...)
TEST_PACKAGES := $(filter-out $(EXAMPLE_PACKAGES), $(ALL_PACKAGES))
include $(MAKEFILES_DIR)/variables/vars.mk
include $(MAKEFILES_DIR)/build.mk

GO_ENV ?=
INTEGRATION_MONGO_PORT ?= 37017
COMPOSE_PROJECT_NAME ?= mm-it

.PHONY: test integration-test test-coverage test-examples

test: ## Run tests for all non-example packages
	@echo "$(GREEN)Running tests...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v $(TEST_PACKAGES)

test-library: ## Run library-specific tests
	@echo "$(GREEN)Running library tests...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v ./migration ./internal/config

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go test -v -coverprofile=coverage.out $(TEST_PACKAGES)
	cd $(REPO_ROOT) && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-examples: ## Test the examples
	@echo "$(GREEN)Testing examples...$(NC)"
	cd $(REPO_ROOT) && $(GO_ENV) go build -o examples/example ./examples
	cd $(REPO_ROOT) && $(GO_ENV) go build -o examples/library-example/library-example ./examples/library-example
	@echo "✅ Examples build successfully!"
	@echo "  - CLI example: examples/example"
	@echo "  - Library example: examples/library-example/library-example"

integration-test: build-all ## Run Docker-based CLI integration tests via docker compose
	@echo "$(GREEN)Running CLI integration tests with docker compose...$(NC)"
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INTEGRATION) build cli
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		$(DOCKER_COMPOSE) -f $(COMPOSE_FILE_INTEGRATION) up -d --wait --wait-timeout 180 mongo
	cd $(REPO_ROOT) && MONGO_URL="mongodb://admin:password@localhost:$(INTEGRATION_MONGO_PORT)/?authSource=admin" \
		INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) \
		$(GO_ENV) go test -v -tags=integration ./integration-tests; \
	status=$$?; \
	cd $(REPO_ROOT) && INTEGRATION_MONGO_PORT=$(INTEGRATION_MONGO_PORT) COMPOSE_PROJECT_NAME=$(COMPOSE_PROJECT_NAME) docker-compose -f $(COMPOSE_FILE_INTEGRATION) down -v; \
	exit $$status
