# dev-cli Makefile
# Development tasks for linting, testing, and building

.PHONY: all build test lint clean install

# Default target
all: lint test build

# Build the CLI binary
build:
	go build -o dev-cli .

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
	rm -f dev-cli
	go clean

# Install the CLI locally
install: build
	cp dev-cli $(GOPATH)/bin/ 2>/dev/null || cp dev-cli ~/go/bin/

# Run integration tests only (requires Docker)
test-integration:
	go test -v -race -run Integration ./...

# Check for issues without fixing
check: lint test-short

# Tidy dependencies
tidy:
	go mod tidy
