.PHONY: lint lint-check test build clean install goreleaser-build goreleaser-check help

# Default target
all: lint test build

# Linting and formatting (applies formatters and runs linters)
lint:
	golangci-lint run --fix ./...

# Linting without fixes (CI friendly)
lint-check:
	golangci-lint run ./...

# Testing
test:
	go test -v -race -coverprofile=coverage.out ./...

# Build the binary
build:
	go build -v .

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

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build"
	@echo "  lint             - Run golangci-lint with formatters and linters"
	@echo "  lint-check       - Run golangci-lint without fixes (CI friendly)"
	@echo "  test             - Run all tests with coverage"
	@echo "  build            - Build the cowpoke binary"
	@echo "  clean            - Clean build artifacts"
	@echo "  install          - Install/update dependencies"
	@echo "  goreleaser-build - Build with goreleaser (snapshot)"
	@echo "  goreleaser-check - Check goreleaser configuration"
	@echo "  help             - Show this help message"