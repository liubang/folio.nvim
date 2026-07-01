.PHONY: build build-all clean run fmt lint

BINARY := folio
GO_SRC := $(shell find . -name '*.go' -not -path './vendor/*')
LUA_SRC := $(shell find lua -name '*.lua')

# Default target: build the Go backend for the current platform.
build:
	go build -o $(BINARY) ./cmd/folio

# Cross-compile for all supported platforms.
build-all: clean
	GOOS=darwin  GOARCH=arm64 go build -o bin/$(BINARY)-darwin-arm64   ./cmd/folio
	GOOS=darwin  GOARCH=amd64 go build -o bin/$(BINARY)-darwin-amd64   ./cmd/folio
	GOOS=linux   GOARCH=arm64 go build -o bin/$(BINARY)-linux-arm64    ./cmd/folio
	GOOS=linux   GOARCH=amd64 go build -o bin/$(BINARY)-linux-amd64    ./cmd/folio
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY)-windows-amd64.exe ./cmd/folio

# Remove build artifacts.
clean:
	rm -f $(BINARY)
	rm -rf bin/

# Run the backend locally (for development).
run: build
	./$(BINARY) -port 0

# Format Go source code.
fmt:
	go fmt ./...

# Lint Go source code (requires golangci-lint).
lint:
	golangci-lint run ./...

# Format Lua source.
fmt-lua:
	stylua lua/
