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

# Run integration tests with database
.PHONY: test-integration
test-integration: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Running integration tests..."
	@TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:5433/urlshortener?sslmode=disable" go test ./internal/store/ ./internal/usecase/ -v -run "Test.*Integration" -timeout 60s || (make docker-down; exit 1)
	@make docker-down

# Build and run the application (migrations will be applied automatically if DB is used)
.PHONY: run
run: build
	@echo "Starting application..."
	@./bin/shortener

# Build the application
.PHONY: build
build:
	@echo "Building application..."
	@mkdir -p bin
	@go build -o bin/shortener ./cmd/shortener

