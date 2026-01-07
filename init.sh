#!/bin/bash

# Environment Setup Script for Vido Nx Monorepo
# This script checks prerequisites and sets up the development environment
# It is idempotent and safe to run multiple times

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

# Install swag CLI for OpenAPI generation
print_info "Installing swag CLI for OpenAPI generation..."
if command_exists swag; then
    SWAG_VERSION=$(swag --version 2>&1 || echo "unknown")
    print_warning "swag is already installed: $SWAG_VERSION"
    print_info "Reinstalling to ensure latest version..."
fi

if go install github.com/swaggo/swag/cmd/swag@latest; then
    if command_exists swag; then
        SWAG_VERSION=$(swag --version 2>&1 || echo "unknown")
        print_success"swag CLI installed successfully: $SWAG_VERSION"
    else
        print_warning "swag installed but not found in PATH"
        print_info "Add \$GOPATH/bin to your PATH:"
        print_info "export PATH=\$PATH:\$(go env GOPATH)/bin"
    fi
else
    print_error "Failed to install swag CLI"
    exit 1
fi
echo ""

# Install Air CLI for hot reload development
print_info "Installing Air for hot reload development..."
if command_exists air; then
    AIR_VERSION=$(air -v 2>&1 | head -n1 || echo "unknown")
    print_warning "Air is already installed: $AIR_VERSION"
    print_info "Reinstalling to ensure latest version..."
fi

if go install github.com/cosmtrek/air@latest; then
    if command_exists air; then
        AIR_VERSION=$(air -v 2>&1 | head -n1 || echo "unknown")
        print_success "Air CLI installed successfully: $AIR_VERSION"
    else
        print_warning "Air installed but not found in PATH"
        print_info "Add \$GOPATH/bin to your PATH:"
        print_info "export PATH=\$PATH:\$(go env GOPATH)/bin"
    fi
else
    print_error "Failed to install Air CLI"
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

# Verify Go modules
print_info "Verifying Go modules..."
if go mod verify; then
    print_success "Go modules verified successfully"
else
    print_error "Failed to verify Go modules"
    exit 1
fi

# Verify configuration files
print_info "Checking configuration files..."
if [ -f ".air.toml" ]; then
    print_success "Found .air.toml configuration file"
else
    print_warning "Warning: .air.toml not found"
fi

if [ -f ".env.example" ]; then
    print_success "Found .env.example file"
    if [ ! -f ".env" ]; then
        print_info "Creating .env from .env.example..."
        cp .env.example .env
        print_success "Created .env file"
    else
        print_info ".env file already exists"
    fi
else
    print_warning "Warning: .env.example not found"
fi

# Check if GOPATH/bin is in PATH
if command_exists swag && command_exists air; then
    print_success "All tools are accessible in PATH"
else
    print_warning "Warning: Some tools may not be in your PATH"
    echo ""
    print_info "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    print_info "export PATH=\$PATH:\$(go env GOPATH)/bin"
    echo ""
    print_info "Or run this command now (temporary for this session):"
    print_info "export PATH=\$PATH:\$(go env GOPATH)/bin"
fi
cd ../..
echo ""

# Success message
print_header "Setup Complete!"
echo ""
print_success "All prerequisites met and dependencies installed"
echo ""
print_info "Next steps:"
echo ""
print_info "For the web application:"
echo "  1. Start the web app:    ${GREEN}nx serve web${NC}"
echo "  2. Build all projects:   ${GREEN}nx run-many -t build${NC}"
echo "  3. Run tests:            ${GREEN}nx run-many -t test${NC}"
echo ""
print_info "For the Go backend API:"
echo "  1. Review your .env file and adjust settings if needed"
echo "     nano apps/api/.env"
echo ""
echo "  2. Build the API binary"
echo "     make -C apps/api build"
echo ""
echo "  3. Start the development server with hot reload"
echo "     make -C apps/api dev"
echo ""
echo "  4. View API documentation in your browser"
echo "     http://localhost:8080/swagger/index.html"
echo ""
print_info "Available Go backend commands:"
echo "  make -C apps/api build   - Build the API binary"
echo "  make -C apps/api run     - Run the API server"
echo "  make -C apps/api dev     - Start hot reload development server"
echo "  make -C apps/api test    - Run all tests"
echo "  make -C apps/api swagger - Regenerate OpenAPI specification"
echo "  make -C apps/api lint    - Run linter"
echo "  make -C apps/api clean   - Clean build artifacts"
echo ""
print_info "For more information, see the README.md"
echo ""