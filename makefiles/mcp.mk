THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

.PHONY: mcp mcp-build mcp-examples mcp-test mcp-client-test mcp-integration-test

mcp-build: ## Build the combined Docker image for MCP
	@echo "$(GREEN)Building MCP Docker image mongo-mongodb-combined-mcp:v1...$(NC)"
	docker build -t mongo-mongodb-combined-mcp:v1 -f deployments/Dockerfile.mcp .

mcp: mcp-build ## Start MCP server for AI assistant integration
	@echo "$(GREEN)Starting MCP server...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp

mcp-examples: mcp-build ## Start MCP server with example migrations registered
	@echo "$(GREEN)Starting MCP server with examples...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-test: ## Test MCP server with example request
	@set -euo pipefail; \
		cleanup() { \
			docker compose -f $(COMPOSE_FILE_INTEGRATION) down -v >/dev/null 2>&1 || true; \
		}; \
		trap cleanup EXIT; \
		host_port=$${INTEGRATION_MONGO_PORT:-37017}; \
		echo "$(GREEN)Starting Mongo test container on port $$host_port...$(NC)"; \
		docker compose -f $(COMPOSE_FILE_INTEGRATION) up -d mongo; \
		if [ -z "$${MONGO_URL:-}" ]; then \
			export MONGO_URL="mongodb://localhost:$$host_port"; \
		fi; \
		echo "$(GREEN)Running MCP integration tests...$(NC)"; \
		cd $(REPO_ROOT) && go test -tags=integration ./mcp; \
		echo "$(GREEN)MCP integration tests finished.$(NC)"


mcp-client-test: mcp-build ## Test MCP server interactively
	@echo "$(GREEN)Testing MCP server interactively (Ctrl+C to exit)...$(NC)"
	@echo "Try these commands:"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"migration_status\",\"arguments\":{}}}"
	@echo ""
	@export MONGO_URL="mongodb://127.0.0.1:27018/test?connect=direct"; \
	export MONGO_DATABASE=test; \
	export MONGO_USERNAME=""; \
	export MONGO_PASSWORD=""; \
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-integration-test: ## Run MCP integration test (requires reachable MongoDB via env)
	@echo "$(GREEN)Running MCP integration test...$(NC)"
	@echo "Requires MONGO_URL (optional; defaults to mongodb://localhost:27017)"
	@cd $(REPO_ROOT) && go test -tags=integration ./mcp -run TestMCPIntegration_IndexingAndMigrations -count=1

.PHONY: mcp-config
mcp-config: mcp-build
	@echo "--- Copy the JSON below into your MCP config file ---"
	@echo "{"
	@echo "  \"mcpServers\": {"
	@echo "    \"mongo-tool\": {"
	@echo "      \"command\": \"$(shell pwd)/build/mongo-tool\","
	@echo "      \"args\": [\"mcp\"],"
	@echo "      \"env\": {"
	@echo "        \"MONGO_URI\": \"$(or $(MONGO_URI),mongodb://localhost:27017)\","
	@echo "        \"MONGO_DATABASE\": \"$(or $(MONGO_DATABASE),your_db_name)\""
	@echo "      }"
	@echo "    }"
	@echo "  }"
	@echo "}"
	@echo "------------------------------------------------------"
