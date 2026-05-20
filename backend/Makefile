.PHONY: run build test test-verbose test-coverage clean deps help

# Binary name
BINARY_NAME=newsaggregator
# Build directory
BUILD_DIR=bin
# Main package path
MAIN_PATH=./cmd/api

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## run: Run the application
run:
	@echo "Running application..."
	@go run $(MAIN_PATH)/main.go

## build: Build the application binary
build:
	@echo "Building application..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

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

## tidy: Clean up go.mod and go.sum
tidy:
	@echo "Tidying go.mod..."
	@go mod tidy
	@echo "Done"
