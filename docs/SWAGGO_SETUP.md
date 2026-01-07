# Swaggo Setup Instructions

This document provides instructions for setting up Swaggo (swag) for OpenAPI specification generation.

## Prerequisites

- Go 1.21 or later
- Git

## Installation Steps

### 1. Install swag CLI

The swag CLI tool is required to generate OpenAPI specifications from Go annotations.

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Verify the installation:

```bash
swag --version
```

### 2. Install Go Dependencies

Install the gin-swagger and related dependencies:

```bash
go mod download
```

This will download:
- `github.com/swaggo/gin-swagger` - Gin middleware for serving Swagger UI
- `github.com/swaggo/files` - Static file handler for Swagger UI assets
- `github.com/swaggo/swag` - Core swag library for annotation parsing

### 3. Verify Installation

Check that all dependencies are installed:

```bash
go mod verify
```

## Configuration

### API Annotations in main.go

The main.go file now includes OpenAPI annotations:

- `@title` - API title
- `@version` - API version
- `@description` - API description
- `@host` - Default API host (can be overridden)
- `@BasePath` - API base path prefix
- `@schemes` - Supported schemes (http, https)
- `@securityDefinitions` - Authentication schemes

### Generating OpenAPI Spec

To generate the OpenAPI specification:

```bash
swag init -g cmd/api/main.go -o docs
```

This will create:
- `docs/swagger.json` - OpenAPI 2.0 specification in JSON format
- `docs/swagger.yaml` - OpenAPI 2.0 specification in YAML format
- `docs/docs.go` - Go package for embedding the spec

## Usage

### Accessing Swagger UI

Once the server is running with Swagger UI configured (in subtask 4.2), you can access:

- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/swagger/doc.json`

### Updating Documentation

After modifying API annotations:

1. Run `swag init` to regenerate the spec
2. Restart the server to load the updated documentation

## Troubleshooting

### swag command not found

If you get "command not found" after installing swag, ensure that `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.) to make it permanent.

### Import errors

If you encounter import errors, run:

```bash
go mod tidy
```

This will clean up the go.mod file and ensure all dependencies are properly tracked.

## Next Steps

- Subtask 4.2: Add OpenAPI annotations to endpoints
- Subtask 4.3: Create OpenAPI generation script and Makefile target
