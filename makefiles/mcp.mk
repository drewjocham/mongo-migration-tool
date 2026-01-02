.PHONY: mcp mcp-build mcp-examples mcp-test mcp-client-test mcp-integration-test

mcp-build: ## Build the combined Docker image for MCP
	@echo "$(GREEN)Building MCP Docker image mongo-mongodb-combined-mcp:v1...$(NC)"
	docker build -t mongo-mongodb-combined-mcp:v1 -f Dockerfile.mcp .

mcp: build ## Start MCP server for AI assistant integration
	@echo "$(GREEN)Starting MCP server...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp

mcp-examples: build ## Start MCP server with example migrations registered
	@echo "$(GREEN)Starting MCP server with examples...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-test: ## Test MCP server with example request
	@set -euo pipefail; \
		cleanup() { \
			docker compose -f integration-compose.yml down -v >/dev/null 2>&1 || true; \
		}; \
		trap cleanup EXIT; \
		host_port=$${INTEGRATION_MONGO_PORT:-37017}; \
		echo "$(GREEN)Starting Mongo test container on port $$host_port...$(NC)"; \
		docker compose -f integration-compose.yml up -d mongo; \
		if [ -z "$${MONGO_URL:-}" ]; then \
			export MONGO_URL="mongodb://localhost:$$host_port"; \
		fi; \
		echo "$(GREEN)Running MCP integration tests...$(NC)"; \
		go test -tags=integration ./mcp; \
		echo "$(GREEN)MCP integration tests finished.$(NC)"


mcp-client-test: build ## Test MCP server interactively
	@echo "$(GREEN)Testing MCP server interactively (Ctrl+C to exit)...$(NC)"
	@echo "Try these commands:"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}"
	@echo "  {\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"migration_status\",\"arguments\":{}}}"
	@echo ""
	./$(BUILD_DIR)/$(BINARY_NAME) mcp --with-examples

mcp-integration-test: ## Run MCP integration test (requires reachable MongoDB via env)
	@echo "$(GREEN)Running MCP integration test...$(NC)"
	@echo "Requires MONGO_URL (optional; defaults to mongodb://localhost:27017)"
	@go test -tags=integration ./mcp -run TestMCPIntegration_IndexingAndMigrations -count=1
