THIS_MK := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILES_DIR := $(dir $(THIS_MK))
REPO_ROOT := $(abspath $(MAKEFILES_DIR)/..)

include $(MAKEFILES_DIR)/variables/vars.mk

.PHONY: docker-build docker-run docker-compose-up docker-compose-down security-scan

DOCKER_TAG=v0.1.0

docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f $(DOCKERFILE_LOCAL) .

docker-run: docker-build ## Run Docker container
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -it \
		-e MONGO_URL=mongodb://host.docker.internal:27017 \
		-e MONGO_DATABASE=test_db \
		$(DOCKER_IMAGE):$(DOCKER_TAG) status

start-tool: ## Start services with docker-compose
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -it \
		-e  "MONGO_URL=..." \
		-e  "MDB_MCP_CONNECTION_STRING=..." \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(NC)"
	docker-compose up -d --remove-orphans

docker-down: ## Stop services with docker-compose
	@echo "$(YELLOW)Stopping services with docker-compose...$(NC)"
	docker-compose down --remove-orphans -v

security-scan: ## Run security scan on Docker image
	@echo "$(GREEN)Running security scan...$(NC)"
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/src aquasec/trivy image $(DOCKER_IMAGE):$(DOCKER_TAG)

db-up: ## Start local MongoDB for testing
	@echo "$(GREEN)Starting local MongoDB...$(NC)"
	docker run --name mongo-migration-test -p 27017:27017 -d mongo:8.0 || \
	docker start mongo-migration-test

db-down: ## Stop local MongoDB
	@echo "$(YELLOW)Stopping local MongoDB...$(NC)"
	docker stop mongo-migration-test || true
	docker rm mongo-migration-test || true
