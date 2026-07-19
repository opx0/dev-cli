# dev-cli Makefile
# Development tasks for linting, testing, and building

.PHONY: all build test test-short lint check clean install tidy test-integration install-lint

# Default target
all: check build

# Build the CLI binary
build:
	go build -o dev-cli .

# Run tests with race detection
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
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.12.2

# Clean build artifacts
clean:
	rm -f dev-cli
	go clean

# Install the CLI locally
install: build
	install -Dm755 dev-cli "$$(go env GOPATH)/bin/dev-cli"

# Run integration tests only (requires Docker)
test-integration:
	go test -v -race -run Integration ./...

# Check for issues without changing files
check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go mod tidy -diff
	go vet ./...
	go test -short ./...
	golangci-lint run ./...

# Tidy dependencies
tidy:
	go mod tidy
