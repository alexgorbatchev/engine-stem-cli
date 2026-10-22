set dotenv-load := false
set shell := ["bash", "-uc"]

binary_name := "engine-stem-cli"
output_dir := "./bin"
binary := output_dir + "/" + binary_name
export VERSION := env("VERSION", "dev")

# Default target: build the binary
default: build

# Run CLI in human mode (default)
run *args:
    go run ./cmd/{{ binary_name }} {{ args }}

# Run CLI in agent-facing token-conservative mode
run-ai *args:
    AGENT=1 go run ./cmd/{{ binary_name }} {{ args }}

# Build binary into ./bin
build:
    @mkdir -p {{ output_dir }}
    go build -trimpath -ldflags "-s -w -X github.com/alexgorbatchev/engine-stem-cli/internal/cli.Version=$VERSION" -o {{ binary }} ./cmd/{{ binary_name }}

# Build and verify a native, versioned release archive (VERSION=vX.Y.Z)
release-package: build
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
    test "$({{ binary }} --version)" = "$VERSION"
    target="$(go env GOOS)_$(go env GOARCH)"
    mkdir -p {{ output_dir }}/releases
    tar -czf "{{ output_dir }}/releases/{{ binary_name }}_${VERSION}_${target}.tar.gz" -C {{ output_dir }} {{ binary_name }} -C "{{ justfile_directory() }}" LICENSE README.md

# Verify module hygiene, static analysis, and race-enabled tests
ci:
    go mod tidy -diff
    go vet ./...
    go test -race -cover ./...

# Run unit tests and the publishing Git hook tests
test:
    go test -v ./...
    bash scripts/test-git-hooks.sh .githooks

# Run static checks, vet, and tests
check:
    go vet ./...
    go test ./...

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Run go vet
vet:
    go vet ./...

# Format Go code
fmt:
    go fmt ./...

# Run linters
lint:
    go vet ./...

# Install binary to GOPATH/bin
install:
    go install ./cmd/{{ binary_name }}

# Clean build artifacts
clean:
    rm -rf {{ output_dir }}

# Download project dependencies and install the publishing Git hooks
setup:
    go mod download
    git config core.hooksPath .githooks
