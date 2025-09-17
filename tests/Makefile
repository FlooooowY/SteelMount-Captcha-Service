# Test Makefile for SteelMount Captcha Service

.PHONY: test test-unit test-integration test-performance test-security test-all clean

# Default target
test: test-unit

# Unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -timeout 30s ./tests/unit/...

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -timeout 60s ./tests/integration/...

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@go test -v -timeout 300s -run "Test.*Performance|Test.*RPS|Test.*Load" ./tests/performance/...

# Security tests
test-security:
	@echo "Running security tests..."
	@go test -v -race -timeout 60s ./tests/security/...

# All tests
test-all: test-unit test-integration test-performance test-security

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./tests/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with benchmarks
test-bench:
	@echo "Running benchmarks..."
	@go test -v -bench=. -benchmem ./tests/performance/...

# Run specific test
test-specific:
	@echo "Running specific test: $(TEST)"
	@go test -v -race -timeout 60s -run "$(TEST)" ./tests/...

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@go clean -testcache

# Help
help:
	@echo "Available targets:"
	@echo "  test              - Run unit tests (default)"
	@echo "  test-unit         - Run unit tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-performance  - Run performance tests"
	@echo "  test-security     - Run security tests"
	@echo "  test-all          - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-bench        - Run benchmarks"
	@echo "  test-specific     - Run specific test (use TEST=TestName)"
	@echo "  clean             - Clean test artifacts"
	@echo "  help              - Show this help"
