# Verus Gateway Makefile

# Variables
BINARY_NAME=verus-gateway
MAIN_PATH=./cmd/gateway
BUILD_DIR=./bin
COVERAGE_FILE=coverage.out
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-w -s"

# Version information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags with version info
VERSION_FLAGS=-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'
BUILD_FLAGS=$(LDFLAGS) -ldflags="$(VERSION_FLAGS)"

.PHONY: all build clean test test-coverage test-integration lint fmt vet run install dev help docker-build docker-run

## help: Display this help message
help:
	@echo "Verus Gateway - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Run tests and build binary
all: test build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✓ All builds complete"

## clean: Remove build artifacts and cache
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	@$(GO) clean -cache -testcache
	@echo "✓ Clean complete"

## test: Run unit tests
test:
	@echo "Running unit tests..."
	$(GO) test $(GOFLAGS) -race -timeout 30s ./...
	@echo "✓ Tests complete"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GO) test $(GOFLAGS) -race -tags=integration -timeout 5m ./tests/integration/...
	@echo "✓ Integration tests complete"

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## lint: Run linter (golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "✓ Lint complete"

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✓ Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "✓ Vet complete"

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy
	@echo "✓ Tidy complete"

## run: Run the application (development mode)
run:
	@echo "Running $(BINARY_NAME)..."
	$(GO) run $(MAIN_PATH)

## dev: Run with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@which air > /dev/null || (echo "air not installed. Install: go install github.com/cosmtrek/air@latest" && exit 1)
	air

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(BUILD_FLAGS) $(MAIN_PATH)
	@echo "✓ Install complete"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t verus-gateway:$(VERSION) -t verus-gateway:latest .
	@echo "✓ Docker build complete"

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env verus-gateway:latest

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	docker compose -f docker-compose.yml up -d

## docker-compose-down: Stop services with docker-compose
docker-compose-down:
	docker compose -f docker-compose.yml down

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "✓ Dependencies downloaded"

## verify: Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify
	@echo "✓ Dependencies verified"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "✓ All checks passed"

## ci: Run CI pipeline (used by GitHub Actions)
ci: check test-coverage
	@echo "✓ CI pipeline complete"

## version: Display version information
version:
	@echo "Version:     $(VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Build Time:  $(BUILD_TIME)"

## setup-tools: Install development tools
setup-tools:
	@echo "Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/cosmtrek/air@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	@echo "✓ Tools installed"

.DEFAULT_GOAL := help
