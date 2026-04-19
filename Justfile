# gfile build recipes

default:
    @just --list

# Build the gfile binary
build:
    go build -o gfile .

# Run tests (short mode)
test:
    go test -short ./...

# Run tests with race detector
race:
    go test -race -short ./...

# Generate test coverage report
coverage:
    mkdir -p cover/
    go test ./... -v -coverprofile cover/testCoverage.txt
    go tool cover -html=cover/testCoverage.txt -o cover/coverage.html

# One-time local install of pinned linter (matches CI)
install-tools:
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4

# Run linter
lint:
    golangci-lint run ./...

# Apply formatting fixes in place
fmt:
    golangci-lint fmt ./...

# Clean build artifacts
clean:
    rm -f gfile
    rm -rf cover/

# Run end-to-end single-PC bench (1 connection, 1 GB) N times and aggregate bandwidth
bench N="1":
    uv run scripts/bench.py -n {{N}} --size 1000 --connections 1

# Run end-to-end multi-PC bench (8 connections, 1 GB) N times and aggregate bandwidth.
# Flaky on macOS 15.x due to a kernel lo0 UDP bug that silently drops all packets
# for one random socket pair when many sockets are in play; ICE then times out.
# Fixed in macOS 26.
bench-multi N="1":
    uv run scripts/bench.py -n {{N}} --size 1000 --connections 8

# Run end-to-end CLI smoke test (real binary, real SDP exchange, fixture round-trip)
test-e2e:
    uv run scripts/e2e.py
