.PHONY: mcp mcp-examples mcp-test mcp-client-test

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
