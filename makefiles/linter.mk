.PHONY: lint format vet

lint: ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(NC)"
	@cd $(REPO_ROOT) && golangci-lint run ./...

format: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	@cd $(REPO_ROOT) && gofmt -s -w .
	@cd $(REPO_ROOT) && goimports -w .

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@cd $(REPO_ROOT) && go vet ./...

ci-test: deps vet lint test ## Run all CI tests
	@echo "$(GREEN)All CI tests passed!$(NC)"
