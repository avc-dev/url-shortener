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

# Check if migrate tool is installed
.PHONY: check-migrate
check-migrate:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate tool is not installed. Please install it from https://github.com/golang-migrate/migrate/tree/master/cmd/migrate" >&2; exit 1; }

# Setup migrations
.PHONY: migrate-up
migrate-up: check-migrate
	migrate -path migrations/schema -database "${POSTGRES_URL}" -verbose up

# Uninstall migrations
.PHONY: migrate-down
migrate-down: check-migrate
	migrate -path migrations/schema -database "${POSTGRES_URL}" -verbose down

