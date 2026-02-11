THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

ROOT_DIR := $(REPO_ROOT)

.PHONY: release release-check releaser-check deploy-dev deploy-prod release-beta sync-library

deploy-dev: ## Deploy to development environment
	@echo "$(GREEN)Deploying to development...$(NC)"
	$(ROOT_DIR)/scripts/deploy-migrations.sh auto

deploy-prod: ## Deploy to production environment
	@echo "$(GREEN)Deploying to production...$(NC)"
	REQUIRE_SIGNED_IMAGES=true $(ROOT_DIR)/scripts/deploy-migrations.sh auto

release-check:
	cd $(ROOT_DIR) && goreleaser release --skip=publish --skip=docker --snapshot --clean

release:
	cd $(ROOT_DIR) && goreleaser release --clean

release-beta: ## Create and release a new beta version
	@echo "$(GREEN)Starting beta release process...$(NC)"
	$(ROOT_DIR)/scripts/release-beta.sh

sync-library:
	MODULE=github.com/drewjocham/mongo-migration-tool; \
	VERSION=v1.0.0; \
	curl -sS "https://proxy.golang.org/$${MODULE}/@v/$${VERSION}.info"
