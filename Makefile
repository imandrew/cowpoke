.PHONY: lint test build clean install goreleaser-build goreleaser-check help

# Default target
all: lint test build

# Linting
lint:
	golangci-lint run

# Testing
test:
	go test -v -race -coverprofile=coverage.out ./...

# Build the binary
build:
	go build -v ./cmd/cowpoke

# Clean build artifacts
clean:
	rm -f cowpoke coverage.out
	rm -rf dist/

# Install dependencies
install:
	go mod download
	go mod tidy

# GoReleaser build (snapshot)
goreleaser-build:
	goreleaser build --snapshot --clean

# GoReleaser check configuration
goreleaser-check:
	goreleaser check

# Format code with gofumpt
fmt:
	gofumpt -l -w .

# Check formatting without making changes
fmt-check:
	gofumpt -l .

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build"
	@echo "  lint             - Run golangci-lint"
	@echo "  test             - Run all tests with coverage"
	@echo "  build            - Build the cowpoke binary"
	@echo "  clean            - Clean build artifacts"
	@echo "  install          - Install/update dependencies"
	@echo "  goreleaser-build - Build with goreleaser (snapshot)"
	@echo "  goreleaser-check - Check goreleaser configuration"
	@echo "  fmt              - Format code with gofumpt"
	@echo "  fmt-check        - Check code formatting (CI friendly)"
	@echo "  help             - Show this help message"