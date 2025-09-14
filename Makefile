# FileOps Makefile
# Cross-platform build system for the FileOps project

# Detect OS for cross-platform compatibility
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    RM := del /q /f
    RMDIR := rmdir /s /q
    MKDIR := mkdir
    PATHSEP := \\
    EXE_EXT := .exe
    NULL_DEVICE := NUL
    DATE_CMD := powershell -Command "Get-Date -Format 'yyyy-MM-dd_HH:mm:ss' -AsUTC"
else
    DETECTED_OS := $(shell uname -s)
    RM := rm -f
    RMDIR := rm -rf
    MKDIR := mkdir -p
    PATHSEP := /
    EXE_EXT := 
    NULL_DEVICE := /dev/null
    DATE_CMD := date -u '+%Y-%m-%d_%H:%M:%S'
endif

# Variables
APP_NAME := fileops
VERSION := $(shell git describe --tags --abbrev=0 2>$(NULL_DEVICE) || echo "v0.1.0")
BUILD_TIME := $(shell $(DATE_CMD) 2>$(NULL_DEVICE) || echo "unknown")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>$(NULL_DEVICE) || echo "unknown")
GO_VERSION := $(shell go version)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)"

# Directories
BUILD_DIR := build
DIST_DIR := dist
CMD_DIR := cmd$(PATHSEP)$(APP_NAME)

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
	@echo "Building $(APP_NAME) $(VERSION) for $(DETECTED_OS)..."
ifeq ($(OS),Windows_NT)
	@if not exist $(BUILD_DIR) $(MKDIR) $(BUILD_DIR)
else
	@$(MKDIR) $(BUILD_DIR)
endif
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)$(PATHSEP)$(APP_NAME)$(EXE_EXT) .$(PATHSEP)$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)$(PATHSEP)$(APP_NAME)$(EXE_EXT)"

# Build for all supported platforms
build-all: clean
	@echo "Building $(APP_NAME) $(VERSION) for all platforms..."
ifeq ($(OS),Windows_NT)
	@if not exist $(DIST_DIR) $(MKDIR) $(DIST_DIR)
else
	@$(MKDIR) $(DIST_DIR)
endif
	@echo "Building for linux/amd64..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$env:GOOS='linux'; $$env:GOARCH='amd64'; go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_linux_amd64 .$(PATHSEP)$(CMD_DIR)"
else
	@env GOOS=linux GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_linux_amd64 .$(PATHSEP)$(CMD_DIR)
endif
	@echo "Building for linux/arm64..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$env:GOOS='linux'; $$env:GOARCH='arm64'; go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_linux_arm64 .$(PATHSEP)$(CMD_DIR)"
else
	@env GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_linux_arm64 .$(PATHSEP)$(CMD_DIR)
endif
	@echo "Building for darwin/amd64..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$env:GOOS='darwin'; $$env:GOARCH='amd64'; go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_darwin_amd64 .$(PATHSEP)$(CMD_DIR)"
else
	@env GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_darwin_amd64 .$(PATHSEP)$(CMD_DIR)
endif
	@echo "Building for darwin/arm64..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$env:GOOS='darwin'; $$env:GOARCH='arm64'; go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_darwin_arm64 .$(PATHSEP)$(CMD_DIR)"
else
	@env GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_darwin_arm64 .$(PATHSEP)$(CMD_DIR)
endif
	@echo "Building for windows/amd64..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$env:GOOS='windows'; $$env:GOARCH='amd64'; go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_windows_amd64.exe .$(PATHSEP)$(CMD_DIR)"
else
	@env GOOS=windows GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)$(PATHSEP)$(APP_NAME)_$(VERSION)_windows_amd64.exe .$(PATHSEP)$(CMD_DIR)
endif
	@echo "Cross-compilation complete. Binaries in $(DIST_DIR)$(PATHSEP)"

# Development build with debug symbols
dev:
	@echo "Building development version..."
ifeq ($(OS),Windows_NT)
	@if not exist $(BUILD_DIR) $(MKDIR) $(BUILD_DIR)
else
	@$(MKDIR) $(BUILD_DIR)
endif
	go build -gcflags "all=-N -l" -o $(BUILD_DIR)$(PATHSEP)$(APP_NAME)-dev$(EXE_EXT) .$(PATHSEP)$(CMD_DIR)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
ifeq ($(OS),Windows_NT)
	@$(BUILD_DIR)$(PATHSEP)$(APP_NAME)$(EXE_EXT) --help
else
	@.$(PATHSEP)$(BUILD_DIR)$(PATHSEP)$(APP_NAME)$(EXE_EXT) --help
endif

# Install to GOPATH/bin
install:
	@echo "Installing $(APP_NAME) to GOPATH/bin..."
	go install $(LDFLAGS) .$(PATHSEP)$(CMD_DIR)

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

# Lint the code with comprehensive checks
lint:
	@echo "Running comprehensive linters..."
	@echo "- Running go vet..."
	go vet ./...
	@echo "- Running staticcheck..."
ifeq ($(OS),Windows_NT)
	@staticcheck -version 2>$(NULL_DEVICE) || go install honnef.co/go/tools/cmd/staticcheck@latest
else
	@which staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
endif
	@staticcheck ./...
	@echo "- Running golangci-lint..."
ifeq ($(OS),Windows_NT)
	@golangci-lint --version 2>$(NULL_DEVICE) || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
	@which golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
endif
	@golangci-lint run ./...

# Enhanced formatting with imports and code organization
fmt:
	@echo "Formatting code comprehensively..."
	@echo "- Running gofmt..."
	gofmt -w -s .
	@echo "- Installing/updating goimports..."
ifeq ($(OS),Windows_NT)
	@goimports -version 2>$(NULL_DEVICE) || go install golang.org/x/tools/cmd/goimports@latest
else
	@which goimports >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
endif
	@echo "→ Running goimports..."
	@goimports -w .
	@echo "→ Organizing modules..."
	go mod tidy

# Code quality checks
quality: lint fmt vet ineffassign misspell

# Static analysis with multiple tools
vet:
	@echo "Running go vet..."
	go vet ./...

# Check for inefficient assignments
ineffassign:
	@echo "Checking for inefficient assignments..."
ifeq ($(OS),Windows_NT)
	@ineffassign -help 2>$(NULL_DEVICE) || go install github.com/gordonklaus/ineffassign@latest
else
	@which ineffassign >/dev/null 2>&1 || go install github.com/gordonklaus/ineffassign@latest
endif
	@ineffassign ./...

# Check for misspellings
misspell:
	@echo "Checking for misspellings..."
ifeq ($(OS),Windows_NT)
	@misspell 2>$(NULL_DEVICE) || go install github.com/client9/misspell/cmd/misspell@latest
else
	@which misspell >/dev/null 2>&1 || go install github.com/client9/misspell/cmd/misspell@latest
endif
	@misspell -error .

# Check for unused code
deadcode:
	@echo "Checking for dead code..."
ifeq ($(OS),Windows_NT)
	@deadcode -help 2>$(NULL_DEVICE) || go install golang.org/x/tools/cmd/deadcode@latest
else
	@which deadcode >/dev/null 2>&1 || go install golang.org/x/tools/cmd/deadcode@latest
endif
	@deadcode ./...

# Vulnerability scanning
vuln:
	@echo "Scanning for vulnerabilities..."
ifeq ($(OS),Windows_NT)
	@govulncheck -version 2>$(NULL_DEVICE) || go install golang.org/x/vuln/cmd/govulncheck@latest
else
	@which govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
endif
	@govulncheck ./...

# Pre-commit checks (run before committing)
pre-commit: quality test security vuln
	@echo "✅ Pre-commit checks passed!"

# CI/CD quality pipeline
ci: deps quality test-cover security vuln
	@echo "✅ CI pipeline completed successfully!"

# Generate code (if using go generate)
generate:
	@echo "Generating code..."
	go generate ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
ifeq ($(OS),Windows_NT)
	@if exist $(BUILD_DIR) $(RMDIR) $(BUILD_DIR) 2>$(NULL_DEVICE)
	@if exist $(DIST_DIR) $(RMDIR) $(DIST_DIR) 2>$(NULL_DEVICE)
	@if exist coverage.out $(RM) coverage.out 2>$(NULL_DEVICE)
	@if exist coverage.html $(RM) coverage.html 2>$(NULL_DEVICE)
else
	@$(RMDIR) $(BUILD_DIR) $(DIST_DIR) 2>$(NULL_DEVICE) || true
	@$(RM) coverage.out coverage.html 2>$(NULL_DEVICE) || true
endif

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
ifeq ($(OS),Windows_NT)
	@if not exist $(DIST_DIR)$(PATHSEP)archives $(MKDIR) $(DIST_DIR)$(PATHSEP)archives
	@echo "Note: Archive creation requires manual zip/tar for cross-platform compatibility"
	@echo "Binaries available in $(DIST_DIR)$(PATHSEP) for manual archiving"
else
	@$(MKDIR) $(DIST_DIR)/archives
	@echo "Creating tar.gz archives for Unix platforms..."
	@cd $(DIST_DIR) && tar -czf archives/$(APP_NAME)_$(VERSION)_linux_amd64.tar.gz $(APP_NAME)_$(VERSION)_linux_amd64
	@cd $(DIST_DIR) && tar -czf archives/$(APP_NAME)_$(VERSION)_linux_arm64.tar.gz $(APP_NAME)_$(VERSION)_linux_arm64
	@cd $(DIST_DIR) && tar -czf archives/$(APP_NAME)_$(VERSION)_darwin_amd64.tar.gz $(APP_NAME)_$(VERSION)_darwin_amd64
	@cd $(DIST_DIR) && tar -czf archives/$(APP_NAME)_$(VERSION)_darwin_arm64.tar.gz $(APP_NAME)_$(VERSION)_darwin_arm64
	@cd $(DIST_DIR) && tar -czf archives/$(APP_NAME)_$(VERSION)_windows_amd64.tar.gz $(APP_NAME)_$(VERSION)_windows_amd64.exe
	@echo "Release archives created in $(DIST_DIR)/archives/"
endif

# Security scan
security:
	@echo "Running security scan..."
ifeq ($(OS),Windows_NT)
	@gosec -version 2>$(NULL_DEVICE) || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
else
	@which gosec >/dev/null 2>&1 || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
endif
	@gosec ./...

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
	@echo "🔨 Build Targets:"
	@echo "  build        Build for current platform"
	@echo "  build-all    Build for all supported platforms"
	@echo "  dev          Build development version with debug symbols"
	@echo "  run          Build and run the application"
	@echo "  install      Install to GOPATH/bin"
	@echo ""
	@echo "🧪 Testing Targets:"
	@echo "  test         Run tests"
	@echo "  test-race    Run tests with race detection"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  bench        Run benchmarks"
	@echo ""
	@echo "✨ Code Quality Targets:"
	@echo "  quality      Run all quality checks (lint + fmt + vet + ineffassign + misspell)"
	@echo "  lint         Run comprehensive linters (staticcheck + golangci-lint)"
	@echo "  fmt          Format code with gofmt + goimports"
	@echo "  vet          Run go vet static analysis"
	@echo "  ineffassign  Check for inefficient assignments"
	@echo "  misspell     Check for misspellings"
	@echo "  deadcode     Check for unused code"
	@echo ""
	@echo "🔒 Security Targets:"
	@echo "  security     Run security scan with gosec"
	@echo "  vuln         Scan for known vulnerabilities"
	@echo ""
	@echo "🚀 Workflow Targets:"
	@echo "  pre-commit   Run all pre-commit checks (quality + test + security + vuln)"
	@echo "  ci           Run full CI pipeline (deps + quality + test-cover + security + vuln)"
	@echo ""
	@echo "🛠️  Utility Targets:"
	@echo "  generate     Run go generate"
	@echo "  deps         Install dependencies"
	@echo "  clean        Clean build artifacts"
	@echo "  info         Show project information"
	@echo ""
	@echo "🐳 Docker Targets:"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Build and run Docker container"
	@echo ""
	@echo "📦 Release Targets:"
	@echo "  release      Create release archives"
	@echo ""
	@echo "💡 Examples:"
	@echo "  make quality              # Run all code quality checks"
	@echo "  make pre-commit           # Run pre-commit validation"
	@echo "  make ci                   # Run full CI pipeline"
	@echo "  make test-cover           # Run tests with coverage"
	@echo "  make build-all release    # Build for all platforms and create archives"
