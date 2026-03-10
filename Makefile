.PHONY: build build-all build-linux build-darwin build-windows test test-integration lint release clean format install install-hooks setup help

BINARY_NAME=x
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -s -w"

# Default target
all: build

# Show help
help:
	@echo "Available targets:"
	@echo "  build          Build for current platform"
	@echo "  build-all      Build for all platforms (Linux, macOS, Windows)"
	@echo "  test           Run tests with race detection and coverage"
	@echo "  test-integration Run integration tests (requires API access)"
	@echo "  lint           Run golangci-lint"
	@echo "  format         Format code with gofmt and goimports"
	@echo "  release        Build optimized release binary"
	@echo "  clean          Remove build artifacts"
	@echo "  install        Install binary locally"
	@echo "  install-hooks  Install git hooks"
	@echo "  setup          Run setup script to initialize project"
	@echo "  help           Show this help message"

# Build for current platform
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)/main.go

# Build for all platforms
build-all: build-linux build-darwin build-windows

# Linux builds
build-linux:
	@echo "Building for Linux..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)/main.go

# macOS builds
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)/main.go

# Windows builds
build-windows:
	@echo "Building for Windows..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/$(BINARY_NAME)/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-arm64.exe ./cmd/$(BINARY_NAME)/main.go

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests (requires API access)
test-integration:
	go test -v -tags=integration ./...

# Run linter
lint:
	golangci-lint run ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
	go mod verify

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out
	rm -f coverage.html
	go clean -cache

# Release build (optimized)
release: clean
	mkdir -p bin
	CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)/main.go
	@echo "Release binary built: bin/$(BINARY_NAME)"

# Development build with debug info
dev:
	go build -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)/main.go

# Install locally
install: build
	go install ./cmd/$(BINARY_NAME)

# Format code
format:
	@echo "Formatting code..."
	@gofmt -w -s .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not installed. Install: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Run go vet
vet:
	go vet ./...

# Check for security issues with gosec (if installed)
security:
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# Run all checks (format, vet, lint, test)
check: format vet lint test

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@git config core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "Hooks installed from .githooks/"

# Run setup script
setup:
	@chmod +x scripts/setup.sh
	@./scripts/setup.sh

# Generate completions
completions:
	@mkdir -p completions
	@./bin/$(BINARY_NAME) completion bash > completions/$(BINARY_NAME).bash
	@./bin/$(BINARY_NAME) completion zsh > completions/_$(BINARY_NAME)
	@./bin/$(BINARY_NAME) completion fish > completions/$(BINARY_NAME).fish
	@./bin/$(BINARY_NAME) completion powershell > completions/_$(BINARY_NAME).ps1
	@echo "Shell completions generated in completions/"

# Dry-run release with GoReleaser
release-dry:
	goreleaser release --snapshot --clean

# Build changelog (requires git-chglog)
changelog:
	@if command -v git-chglog >/dev/null 2>&1; then \
		git-chglog -o CHANGELOG.md; \
	else \
		echo "git-chglog not installed. Install: go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest"; \
	fi

# Homebrew formula packaging
homebrew-formula:
	@echo "Homebrew formula available at: homebrew/x-cli.rb"
	@echo "To use locally:"
	@echo "  brew install --formula homebrew/x-cli.rb"

# Snap package build
snap:
	snapcraft

# Snap clean
snap-clean:
	snapcraft clean

# Docker build
docker-build:
	docker build -t x-cli:$(VERSION) .

# Docker test
docker-test:
	docker run --rm x-cli:$(VERSION) x --help

# Distribution packages
dist-packages: release
	@echo "Creating distribution packages..."
	@mkdir -p dist/packages
	
	# Create tarballs
	cd dist && tar -czf packages/x-cli-$(VERSION)-linux-amd64.tar.gz x-linux-amd64
	cd dist && tar -czf packages/x-cli-$(VERSION)-linux-arm64.tar.gz x-linux-arm64
	cd dist && tar -czf packages/x-cli-$(VERSION)-darwin-amd64.tar.gz x-darwin-amd64
	cd dist && tar -czf packages/x-cli-$(VERSION)-darwin-arm64.tar.gz x-darwin-arm64
	cd dist && zip packages/x-cli-$(VERSION)-windows-amd64.zip x-windows-amd64.exe
	
	# Generate checksums
	cd dist/packages && sha256sum *.tar.gz *.zip > checksums.txt
	
	@echo "Distribution packages created in dist/packages/"

# Full release (build, test, package)
full-release: check dist-packages
	@echo "Full release complete!"
	@echo "Packages available in: dist/packages/"
	@echo "Checksums: dist/packages/checksums.txt"
