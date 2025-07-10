.PHONY: dev lint test e2e release clean help

# Variables
GO_VERSION := 1.23
PROJECT := regctl
VERSION ?= dev
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Help
help:
	@echo "Available commands:"
	@echo "  make dev       - Start development environment"
	@echo "  make lint      - Run linters"
	@echo "  make test      - Run unit tests"
	@echo "  make e2e       - Run end-to-end tests"
	@echo "  make release   - Build release binaries"
	@echo "  make clean     - Clean build artifacts"

# Development
dev:
	@echo "Starting Cloudflare Workers development..."
	cd edge/workers && npm run dev

# Linting
lint:
	@echo "Running linters..."
	cd cmd/regctl && go vet ./...
	cd cmd/regctl && golangci-lint run
	cd edge/workers && npm run lint

# Testing
test:
	@echo "Running unit tests..."
	cd cmd/regctl && go test -v ./...
	cd edge/workers && npm test

# E2E Testing
e2e:
	@echo "Running e2e tests..."
	cd test && go test -v -tags=e2e ./...

# Release
release:
	@echo "Building release v$(VERSION)..."
	@mkdir -p dist
	cd cmd/regctl && GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../../dist/regctl-linux-amd64
	cd cmd/regctl && GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../../dist/regctl-darwin-amd64
	cd cmd/regctl && GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../../dist/regctl-darwin-arm64
	cd cmd/regctl && GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o ../../dist/regctl-windows-amd64.exe
	@echo "Release binaries built in dist/"

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf dist/
	cd edge/workers && rm -rf dist/ .wrangler/