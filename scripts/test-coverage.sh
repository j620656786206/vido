#!/bin/bash

set -e

echo "Running tests with coverage and race detection..."
echo ""

# Run tests with race detector and coverage
echo "=== Running tests with race detector ==="
go test -race -v ./...
echo ""

# Generate coverage report
echo "=== Generating coverage report ==="
go test -coverprofile=coverage.out ./...
echo ""

# Display overall coverage
echo "=== Overall Coverage ==="
go tool cover -func=coverage.out | tail -1
echo ""

# Display coverage by package
echo "=== Coverage by Package ==="
go tool cover -func=coverage.out | grep -E "^github.com/alexyu/vido/internal/(config|handlers|middleware|tmdb)/" | awk '{print $1 " " $3}'
echo ""

# Check TMDb package coverage specifically
echo "=== TMDb Package Coverage ==="
cd internal/tmdb
go test -cover
cd ../..
echo ""

# Generate HTML coverage report
echo "=== Generating HTML coverage report ==="
go tool cover -html=coverage.out -o coverage.html
echo "HTML coverage report generated: coverage.html"
echo ""

echo "✓ All tests passed with coverage analysis complete"
