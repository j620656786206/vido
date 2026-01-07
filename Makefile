.PHONY: help build run dev test test-coverage test-race swagger lint clean tidy

# Default target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the API binary"
	@echo "  make run           - Run the API server"
	@echo "  make dev           - Start Air hot reload server"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make swagger       - Regenerate OpenAPI specification"
	@echo "  make lint          - Run linter (golangci-lint)"
	@echo "  make tidy          - Update go.mod and go.sum"
	@echo "  make clean         - Clean build artifacts"

# Update go.mod and go.sum
tidy:
	@echo "Updating go.mod and go.sum..."
	@go mod tidy
	@echo "Dependencies updated"

# Build the API binary
build: tidy
	@echo "Building API binary..."
	@mkdir -p bin
	@go build -o bin/api cmd/api/main.go
	@echo "Build complete: bin/api"

# Run the API server
run: build
	@echo "Starting API server..."
	@./bin/api

# Start Air hot reload server for development
dev:
	@echo "Starting Air hot reload server..."
	@if ! command -v air > /dev/null; then \
		echo "Error: Air is not installed. Please run: go install github.com/cosmtrek/air@latest"; \
		echo "Make sure $$GOPATH/bin is in your PATH"; \
		exit 1; \
	fi
	@air

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@echo ""
	@echo "=== Coverage Summary ==="
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "For detailed coverage report, run: go tool cover -html=coverage.out"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

# Regenerate OpenAPI specification
swagger:
	@echo "Regenerating OpenAPI specification..."
	@./scripts/generate-openapi.sh

# Run linter
lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Error: golangci-lint is not installed."; \
		echo "Please install it from: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	@golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf api/
	@echo "Clean complete"
