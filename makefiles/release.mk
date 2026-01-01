.PHONY: release release-check deploy-dev deploy-prod

export GITHUB_TOKEN = "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"

release-check: clean ci-test build-all goreleaser-check ## Create a release build
	@echo "$(GREEN)Release build completed!$(NC)"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	./scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true ./scripts/deploy-migrations.sh auto

goreleaser-check:
	goreleaser release --skip=publish --snapshot --clean

release:
	goreleaser release --clean
