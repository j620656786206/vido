#!/bin/bash

# Environment Setup Script for Vido Nx Monorepo
# This script checks prerequisites and sets up the development environment

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Version comparison function (returns 0 if version1 >= version2)
version_ge() {
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Main script
print_header "Vido Environment Setup"

print_info "Checking prerequisites..."
echo ""

# Check Node.js
print_header "Checking Node.js"
if ! command_exists node; then
    print_error "Node.js is not installed"
    print_info "Please install Node.js 20.x or higher (LTS Iron)"
    print_info "Visit: https://nodejs.org/ or use nvm: https://github.com/nvm-sh/nvm"
    exit 1
fi

NODE_VERSION=$(node --version | sed 's/v//')
REQUIRED_NODE_VERSION="20.0.0"

print_info "Found Node.js version: $NODE_VERSION"

if version_ge "$NODE_VERSION" "$REQUIRED_NODE_VERSION"; then
    print_success "Node.js version is compatible (>= $REQUIRED_NODE_VERSION)"
else
    print_error "Node.js version $NODE_VERSION is too old"
    print_info "Required: Node.js >= $REQUIRED_NODE_VERSION (LTS Iron)"
    print_info "Please upgrade Node.js or use nvm: nvm install lts/iron"
    exit 1
fi

# Check npm
if ! command_exists npm; then
    print_error "npm is not installed"
    print_info "npm should come with Node.js. Please reinstall Node.js"
    exit 1
fi

NPM_VERSION=$(npm --version)
print_success "npm version: $NPM_VERSION"
echo ""

# Check Go
print_header "Checking Go"
if ! command_exists go; then
    print_error "Go is not installed"
    print_info "Please install Go 1.23 or higher"
    print_info "Visit: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_GO_VERSION="1.23"

print_info "Found Go version: $GO_VERSION"

if version_ge "$GO_VERSION" "$REQUIRED_GO_VERSION"; then
    print_success "Go version is compatible (>= $REQUIRED_GO_VERSION)"
else
    print_error "Go version $GO_VERSION is too old"
    print_info "Required: Go >= $REQUIRED_GO_VERSION"
    print_info "Please upgrade Go: https://golang.org/doc/install"
    exit 1
fi
echo ""

# Install dependencies
print_header "Installing Dependencies"
echo ""

# Install npm dependencies
print_info "Installing npm dependencies..."
if npm install; then
    print_success "npm dependencies installed successfully"
else
    print_error "Failed to install npm dependencies"
    exit 1
fi
echo ""

# Install Go dependencies
print_info "Installing Go dependencies for apps/api..."
cd apps/api
if go mod download; then
    print_success "Go dependencies downloaded successfully"
else
    print_error "Failed to download Go dependencies"
    exit 1
fi
cd ../..
echo ""

# Success message
print_header "Setup Complete!"
echo ""
print_success "All prerequisites met and dependencies installed"
echo ""
print_info "Next steps:"
echo "  1. Start the web app:    ${GREEN}nx serve web${NC}"
echo "  2. Start the API server: ${GREEN}nx serve api${NC}"
echo "  3. Build all projects:   ${GREEN}nx run-many -t build${NC}"
echo "  4. Run tests:            ${GREEN}nx run-many -t test${NC}"
echo ""
print_info "For more information, see the README.md"
echo ""
