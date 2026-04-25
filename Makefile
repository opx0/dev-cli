# dev-cli Makefile
# Development tasks for linting, testing, and building

.PHONY: all build build-mcp build-all test lint clean install

# Default target
all: lint test build-all

# Build the CLI binary
build:
	go build -o dev-cli .

# Build the MCP server binary
build-mcp:
	go build -o dev-mcp ./cmd/mcp/

# Build both binaries
build-all: build build-mcp

# Run tests with race detection (important for sync.RWMutex in Registry)
test:
	go test -v -race ./...

# Run short tests (skip integration tests that need Docker)
test-short:
	go test -v -short ./...

# Run linter
lint:
	golangci-lint run ./...

# Install golangci-lint if not present
install-lint:
	@which golangci-lint > /dev/null || \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin

# Clean build artifacts
clean:
	rm -f dev-cli dev-mcp
	go clean

# Install both binaries locally
install: build-all
	cp dev-cli $(GOPATH)/bin/ 2>/dev/null || cp dev-cli ~/go/bin/
	cp dev-mcp $(GOPATH)/bin/ 2>/dev/null || cp dev-mcp ~/go/bin/

# Run integration tests only (requires Docker)
test-integration:
	go test -v -race -run Integration ./...

# Check for issues without fixing
check: lint test-short

# Tidy dependencies
tidy:
	go mod tidy

# Full setup: Docker + Ollama + GPU + build + test
setup:
	@./setup.sh

# Start Ollama (assumes Docker is running)
ollama-up:
	docker compose -f infra/ollama/docker-compose.yml up -d

# Stop Ollama
ollama-down:
	docker compose -f infra/ollama/docker-compose.yml down
