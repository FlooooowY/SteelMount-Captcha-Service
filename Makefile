# Makefile for SteelMount Captcha Service

.PHONY: test test-unit test-integration test-performance test-security test-all test-coverage test-bench test-specific clean help build run

# Default target
test: test-unit

# Build the application
build:
	@echo "Building captcha service..."
	@go build -o captcha-service.exe cmd/server/main.go

# Run the application
run:
	@echo "Starting captcha service..."
	@go run ./cmd/server/main.go

# Unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -v -timeout 30s ./tests/unit/...

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -timeout 60s ./tests/integration/...

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@go test -v -timeout 300s -run "Test.*Performance|Test.*RPS|Test.*Load" ./tests/performance/...

# Security tests
test-security:
	@echo "Running security tests..."
	@go test -v -timeout 120s ./tests/security/...

# All tests
test-all: test-unit test-integration test-performance test-security

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out -covermode=atomic ./tests/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection (requires CGO)
test-race:
	@echo "Running tests with race detection..."
	@CGO_ENABLED=1 go test -v -race -timeout 60s ./tests/...

# Run tests with benchmarks
test-bench:
	@echo "Running benchmarks..."
	@go test -v -bench=. -benchmem ./tests/performance/...

# Run specific test
test-specific:
	@echo "Running specific test: $(TEST)"
	@go test -v -race -timeout 60s -run "$(TEST)" ./tests/...

# Clean test artifacts and build files
clean:
	@echo "Cleaning artifacts..."
	@rm -f coverage.out coverage.html captcha-service.exe
	@go clean -testcache

# Lint code
lint:
	@echo "Running linter..."
	@go vet ./...
	@golangci-lint run || echo "golangci-lint not installed, skipping"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go-grpc_out=. proto/captcha/v1/captcha.proto
	@protoc --go_out=. --go-grpc_out=. proto/balancer/v1/balancer.proto

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t captcha-service .

# Docker run
docker-run:
	@echo "Running with Docker Compose..."
	@docker-compose up -d

# Docker stop
docker-stop:
	@echo "Stopping Docker Compose..."
	@docker-compose down

# Help
help:
	@echo "Available targets:"
	@echo "  build             - Build the captcha service binary"
	@echo "  run               - Run the captcha service"
	@echo "  test              - Run unit tests (default)"
	@echo "  test-unit         - Run unit tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-performance  - Run performance tests"
	@echo "  test-security     - Run security tests"
	@echo "  test-all          - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-race         - Run tests with race detection (requires CGO)"
	@echo "  test-bench        - Run benchmarks"
	@echo "  test-specific     - Run specific test (use TEST=TestName)"
	@echo "  clean             - Clean artifacts and build files"
	@echo "  lint              - Run code linter"
	@echo "  fmt               - Format code"
	@echo "  deps              - Download and tidy dependencies"
	@echo "  proto             - Generate protobuf files"
	@echo "  docker-build      - Build Docker image"
	@echo "  docker-run        - Run with Docker Compose"
	@echo "  docker-stop       - Stop Docker Compose"
	@echo "  help              - Show this help"