.PHONY: release release-check deploy-dev deploy-prod release-beta

release-check: clean ci-test build-all releaser-check ## Create a release build
	@echo "$(GREEN)Release build completed!$(NC)"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	./scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true ./scripts/deploy-migrations.sh auto

releaser-check:
	goreleaser release --skip=publish --snapshot --clean

release:
	goreleaser release --clean

release-beta: ## Create and release a new beta version
	@echo "$(GREEN)Starting beta release process...$(NC)"
	./scripts/release-beta.sh
