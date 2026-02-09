.PHONY: build test lint clean run server install help

VERSION ?= dev
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
BINARY := regctl

help:
	@echo ""
	@echo "  regctl — Domain Management CLI"
	@echo "  ==============================="
	@echo ""
	@echo "  make build    Build the binary"
	@echo "  make test     Run all tests"
	@echo "  make lint     Run go vet"
	@echo "  make run      Build and run the CLI"
	@echo "  make server   Build and start the API server"
	@echo "  make install  Install to /usr/local/bin"
	@echo "  make clean    Remove build artifacts"
	@echo "  make release  Build for all platforms"
	@echo ""

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) ./cmd/regctl

test:
	go test -v ./...

lint:
	go vet ./...

run: build
	./$(BINARY)

server: build
	./$(BINARY) server --port 8080

install: build
	@echo "Installing regctl to /usr/local/bin..."
	@if [ -w /usr/local/bin ]; then \
		cp $(BINARY) /usr/local/bin/$(BINARY); \
	else \
		sudo cp $(BINARY) /usr/local/bin/$(BINARY); \
	fi
	@echo "Done! Run 'regctl init' to get started."

clean:
	rm -f $(BINARY)
	rm -rf dist/

release:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/regctl
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/regctl
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/regctl
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/regctl
	@echo ""
	@echo "Release binaries built in dist/"
