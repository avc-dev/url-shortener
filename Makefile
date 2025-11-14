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

