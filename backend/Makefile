.PHONY: run build test test-verbose test-coverage clean deps vapid help

# Build directory
BUILD_DIR=bin

# Version info
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## run: Run the API
run:
	@echo "Running API..."
	@go run ./cmd/api/main.go

## build: Build the API binary
build:
	@echo "Building API..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/newsaggregator ./cmd/api/main.go
	@echo "Binary built at $(BUILD_DIR)/newsaggregator"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test ./...

## test-verbose: Run all tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## clean: Remove build artifacts and temporary files
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies updated"

## vapid: Generate a VAPID keypair for Web Push (run once)
vapid:
	@go run ./cmd/vapidgen

## tidy: Clean up go.mod and go.sum
tidy:
	@echo "Tidying go.mod..."
	@go mod tidy
	@echo "Done"
