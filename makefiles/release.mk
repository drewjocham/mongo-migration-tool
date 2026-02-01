THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

.PHONY: release release-check deploy-dev deploy-prod release-beta



release-check: clean ci-test build-all releaser-check ## Create a release build
	@echo "$(GREEN)Release build completed!$(NC)"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	$(ROOT_DIR)/scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true $(ROOT_DIR)/scripts/deploy-migrations.sh auto

releaser-check:
	cd $(ROOT_DIR) && goreleaser release --skip=publish --snapshot --clean

release:
	cd $(ROOT_DIR) && goreleaser release --clean

release-beta: ## Create and release a new beta version
	@echo "$(GREEN)Starting beta release process...$(NC)"
	$(ROOT_DIR)/scripts/release-beta.sh
