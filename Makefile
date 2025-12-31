include makefiles/variables/vars.mk
include makefiles/variables/common.mk

include $(MAKEFILES_DIR)/build.mk
include $(MAKEFILES_DIR)/docker.mk
include $(MAKEFILES_DIR)/linter.mk
include $(MAKEFILES_DIR)/mcp.mk
include $(MAKEFILES_DIR)/release.mk
include $(MAKEFILES_DIR)/migration.mk
include $(MAKEFILES_DIR)/dev.mk
help: ## Show this help message
	@echo "MongoDB Migration Tool - Available commands:"
	@echo
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
