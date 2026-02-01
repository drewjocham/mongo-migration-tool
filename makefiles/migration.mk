THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

.PHONY: create-migration migration-status migration-up migration-down

create-migration: build ## Create a new migration (usage: make create-migration DESC="description")
ifndef DESC
	@echo "$(RED)Error: DESC is required. Usage: make create-migration DESC=\"your description\"$(NC)"
	@exit 1
endif
	@echo "$(GREEN)Creating migration: $(DESC)$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) create "$(DESC)"

migration-status: build ## Show migration status
	@echo "$(GREEN)Checking migration status...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) status

migration-up: build ## Run all pending migrations
	@echo "$(GREEN)Running migrations...$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) up

migration-down: build ## Rollback migrations (usage: make migration-down VERSION="20231201_001")
ifndef VERSION
	@echo "$(RED)Error: VERSION is required. Usage: make migration-down VERSION=\"20231201_001\"$(NC)"
	@exit 1
endif
	@echo "$(YELLOW)Rolling back to version: $(VERSION)$(NC)"
	./$(BUILD_DIR)/$(BINARY_NAME) down --target=$(VERSION)
