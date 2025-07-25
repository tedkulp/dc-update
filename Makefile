# dc-update Makefile
# Simple build automation for development

.PHONY: build test clean install dev lint fmt help

# Default target
help:
	@echo "Available targets:"
	@echo "  build    - Build binary for current platform"
	@echo "  dev      - Run without building (for development)"
	@echo "  test     - Run all tests"
	@echo "  lint     - Run linter and static analysis"
	@echo "  fmt      - Format all Go code"
	@echo "  install  - Install globally"
	@echo "  clean    - Clean build artifacts"
	@echo "  release  - Test release build (requires GoReleaser)"

# Build binary for current platform
build:
	go build -o dc-update cmd/dc-update/main.go

# Run without building (development)
dev:
	go run cmd/dc-update/main.go $(ARGS)

# Run tests
test:
	go test ./...

# Lint and static analysis
lint:
	go vet ./...
	go fmt ./...

# Format code
fmt:
	go fmt ./...

# Install globally
install:
	go install ./cmd/dc-update

# Clean build artifacts
clean:
	rm -f dc-update
	rm -rf dist/

# Test release build (requires GoReleaser)
release:
	goreleaser build --snapshot --clean

# Dependencies
deps:
	go mod tidy
	go mod download