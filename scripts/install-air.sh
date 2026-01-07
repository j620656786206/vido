#!/bin/bash

# Script to install Air CLI for hot reload development
# This script is idempotent and safe to run multiple times

set -e  # Exit on error

echo "🔧 Installing Air for hot reload development..."
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

# Check if Air is already installed
if command -v air &> /dev/null; then
    AIR_VERSION=$(air -v 2>&1 | head -n1 || echo "unknown")
    echo "ℹ️  Air is already installed: $AIR_VERSION"
    echo "   Reinstalling to ensure latest version..."
    echo ""
fi

# Install Air CLI
echo "📦 Installing Air CLI..."
go install github.com/cosmtrek/air@latest

# Verify Air installation
if command -v air &> /dev/null; then
    AIR_VERSION=$(air -v 2>&1 | head -n1 || echo "unknown")
    echo "✅ Air CLI installed successfully: $AIR_VERSION"
else
    echo "⚠️  Air installed but not found in PATH"
    echo "   Add \$GOPATH/bin to your PATH:"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
    echo ""
    echo "   Or add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
fi
echo ""

# Verify .air.toml exists
if [ -f ".air.toml" ]; then
    echo "✅ Found .air.toml configuration file"
else
    echo "⚠️  Warning: .air.toml not found in current directory"
    echo "   Air will use default configuration"
fi
echo ""

echo "✅ Air setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'make dev' or 'air' to start hot reload development server"
echo "  2. Edit any .go file and Air will automatically rebuild and restart"
echo "  3. See docs/AIR_SETUP.md for detailed documentation"
