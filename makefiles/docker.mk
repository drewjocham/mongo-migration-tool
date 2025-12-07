.PHONY: docker-build docker-run docker-compose-up docker-compose-down security-scan

docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build ## Run Docker container
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -it \
		-e MONGO_URL=mongodb://host.docker.internal:27017 \
		-e MONGO_DATABASE=test_db \
		$(DOCKER_IMAGE):$(DOCKER_TAG) status

docker-compose-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(NC)"
	docker-compose up -d --remove-orphans

docker-compose-down: ## Stop services with docker-compose
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
