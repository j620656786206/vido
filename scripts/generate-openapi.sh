#!/bin/bash

# Script to regenerate OpenAPI specification and prepare for SDK generation
# Usage: ./scripts/generate-openapi.sh

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Regenerating OpenAPI specification...${NC}"

# Check if swag is installed
if ! command -v swag &> /dev/null; then
    echo -e "${RED}Error: swag CLI is not installed${NC}"
    echo "Please run: ./scripts/install-swag.sh"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go 1.21 or later from https://golang.org/dl/"
    exit 1
fi

# Navigate to project root (script is in scripts/ directory)
cd "$(dirname "$0")/.."

echo "Running swag init..."
swag init -g cmd/api/main.go -o docs

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ OpenAPI spec generated successfully${NC}"
else
    echo -e "${RED}✗ Failed to generate OpenAPI spec${NC}"
    exit 1
fi

# Create api directory if it doesn't exist
echo "Creating api/ directory for SDK generation..."
mkdir -p api

# Copy swagger.json to api/openapi.json for SDK generation
echo "Copying swagger.json to api/openapi.json..."
cp docs/swagger.json api/openapi.json

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ OpenAPI spec copied to api/openapi.json${NC}"
else
    echo -e "${RED}✗ Failed to copy OpenAPI spec${NC}"
    exit 1
fi

# Print file sizes for verification
echo ""
echo "Generated files:"
echo "  docs/swagger.json   : $(wc -c < docs/swagger.json | xargs) bytes"
echo "  docs/swagger.yaml   : $(wc -c < docs/swagger.yaml | xargs) bytes"
echo "  api/openapi.json    : $(wc -c < api/openapi.json | xargs) bytes"

echo ""
echo -e "${GREEN}✓ OpenAPI specification regeneration complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Review the spec at: http://localhost:8080/swagger/index.html (when server is running)"
echo "  2. Use api/openapi.json for TypeScript SDK generation"
echo "  3. Commit the changes: git add docs/ api/ && git commit -m 'Update OpenAPI spec'"
