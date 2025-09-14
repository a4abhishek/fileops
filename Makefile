# FileOps Makefile
# Build system for the FileOps project

# Variables
APP_NAME := fileops
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.GoVersion=$(GO_VERSION)"

# Directories
BUILD_DIR := build
DIST_DIR := dist
CMD_DIR := cmd/$(APP_NAME)

# Go build flags
GOFLAGS := -trimpath
TAGS := 

# Supported platforms for cross-compilation
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: all build build-all clean test test-race test-cover lint fmt deps help install dev run

# Default target
all: clean build

# Build for current platform
build:
	@echo "Building $(APP_NAME) $(VERSION) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all supported platforms
build-all: clean
	@echo "Building $(APP_NAME) $(VERSION) for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		output_name="$(APP_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then \
			output_name="$$output_name.exe"; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		env GOOS=$$GOOS GOARCH=$$GOARCH go build $(GOFLAGS) $(LDFLAGS) \
			-o $(DIST_DIR)/$$output_name ./$(CMD_DIR); \
	done
	@echo "Cross-compilation complete. Binaries in $(DIST_DIR)/"

# Development build with debug symbols
dev:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags "all=-N -l" -o $(BUILD_DIR)/$(APP_NAME)-dev ./$(CMD_DIR)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME) --help

# Install to GOPATH/bin
install:
	@echo "Installing $(APP_NAME) to GOPATH/bin..."
	go install $(LDFLAGS) ./$(CMD_DIR)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Lint the code
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/"; \
		go vet ./...; \
		go fmt ./...; \
	fi

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	go mod tidy

# Generate code (if using go generate)
generate:
	@echo "Generating code..."
	go generate ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out coverage.html

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it $(APP_NAME):$(VERSION)

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/archives
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		binary_name="$(APP_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then \
			binary_name="$$binary_name.exe"; \
			archive_name="$(APP_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.zip"; \
			cd $(DIST_DIR) && zip $$archive_name $$binary_name && cd ..; \
		else \
			archive_name="$(APP_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.tar.gz"; \
			cd $(DIST_DIR) && tar -czf $$archive_name $$binary_name && cd ..; \
		fi; \
		mv $(DIST_DIR)/$$archive_name $(DIST_DIR)/archives/; \
	done
	@echo "Release archives created in $(DIST_DIR)/archives/"

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Show project information
info:
	@echo "Project Information:"
	@echo "  Name: $(APP_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Go Version: $(GO_VERSION)"

# Help target
help:
	@echo "FileOps Build System"
	@echo "===================="
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build for current platform"
	@echo "  build-all    Build for all supported platforms"
	@echo "  dev          Build development version with debug symbols"
	@echo "  run          Build and run the application"
	@echo "  install      Install to GOPATH/bin"
	@echo "  test         Run tests"
	@echo "  test-race    Run tests with race detection"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  bench        Run benchmarks"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  generate     Run go generate"
	@echo "  deps         Install dependencies"
	@echo "  clean        Clean build artifacts"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Build and run Docker container"
	@echo "  release      Create release archives"
	@echo "  security     Run security scan"
	@echo "  info         Show project information"
	@echo "  help         Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                  # Build for current platform"
	@echo "  make test-cover             # Run tests with coverage"
	@echo "  make build-all release      # Build for all platforms and create archives"
