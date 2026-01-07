#!/bin/bash

# Script to install swag CLI and download Go dependencies
# This script is idempotent and safe to run multiple times

set -e  # Exit on error

echo "🔧 Installing Swaggo for OpenAPI specification generation..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    echo "   Please install Go 1.21 or later from https://golang.org/dl/"
    exit 1
fi

# Get Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Found Go: $GO_VERSION"
echo ""

# Install swag CLI
echo "📦 Installing swag CLI..."
go install github.com/swaggo/swag/cmd/swag@latest

# Verify swag installation
if command -v swag &> /dev/null; then
    SWAG_VERSION=$(swag --version 2>&1 || echo "unknown")
    echo "✅ swag CLI installed successfully: $SWAG_VERSION"
else
    echo "⚠️  swag installed but not found in PATH"
    echo "   Add \$GOPATH/bin to your PATH:"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
fi
echo ""

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download

echo "✅ Dependencies downloaded successfully"
echo ""

# Verify dependencies
echo "🔍 Verifying Go modules..."
go mod verify

echo ""
echo "✅ Swaggo setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'swag init -g cmd/api/main.go -o docs' to generate OpenAPI spec"
echo "  2. See docs/SWAGGO_SETUP.md for detailed documentation"
