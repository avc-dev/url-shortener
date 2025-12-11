.PHONY: mocks
mocks:
	@echo "Generating mocks..."
	@mockery

.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

.PHONY: test-handler
test-handler:
	@echo "Running handler tests..."
	@go test -v ./internal/handler/...

.PHONY: clean-mocks
clean-mocks:
	@echo "Cleaning generated mocks..."
	@rm -rf internal/mocks

.PHONY: docker-up
docker-up:
	@echo "Starting PostgreSQL container..."
	@docker-compose up -d postgres

.PHONY: docker-down
docker-down:
	@echo "Stopping PostgreSQL container..."
	@docker-compose down

.PHONY: docker-logs
docker-logs:
	@echo "Showing PostgreSQL logs..."
	@docker-compose logs -f postgres

