.PHONY: build run test clean generate proto lint vet benchmark docker

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=captcha-service
BINARY_UNIX=$(BINARY_NAME)_unix

# Directories
PROTO_DIR=proto
PB_DIR=pb
CMD_DIR=cmd
INTERNAL_DIR=internal

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./$(CMD_DIR)/server

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./$(CMD_DIR)/server

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./$(CMD_DIR)/server
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Generate protobuf files
generate: proto
	$(GOCMD) generate ./...

# Generate protobuf Go code
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/balancer/v1/balancer.proto
	protoc --go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/captcha/v1/captcha.proto

# Run tests
test:
	$(GOTEST) -v ./...

# Run unit tests
test-unit:
	$(GOTEST) -v -short ./...

# Run integration tests
test-integration:
	$(GOTEST) -v -tags=integration ./...

# Run benchmarks
test-benchmark:
	$(GOTEST) -bench=. -benchmem ./...

# Run linter
lint:
	golangci-lint run

# Run go vet
vet:
	$(GOCMD) vet ./...

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Install development tools
install-tools:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u google.golang.org/protobuf/cmd/protoc-gen-go
	$(GOGET) -u google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Docker build
docker:
	docker build -t steelmount-captcha:latest .

# Docker run
docker-run:
	docker run -p 38000-40000:38000-40000 steelmount-captcha:latest

# Format code
fmt:
	$(GOCMD) fmt ./...

# Check code formatting
fmt-check:
	@if [ $$(gofmt -s -l . | wc -l) -gt 0 ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -s -l .; \
		exit 1; \
	fi

# Full check (lint, vet, test, fmt-check)
check: fmt-check vet lint test

# Development setup
dev-setup: install-tools deps generate
	@echo "Development environment ready!"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-linux   - Build for Linux"
	@echo "  run           - Build and run the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  generate      - Generate protobuf files"
	@echo "  proto         - Generate protobuf Go code"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-benchmark - Run benchmarks"
	@echo "  lint          - Run linter"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  docker        - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  fmt           - Format code"
	@echo "  fmt-check     - Check code formatting"
	@echo "  check         - Run all checks"
	@echo "  dev-setup     - Setup development environment"
	@echo "  help          - Show this help"
