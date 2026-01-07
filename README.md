# Vido - Nx Monorepo

A full-stack media management application built with Nx monorepo architecture, featuring a React frontend and a high-performance Go backend with structured logging, error handling, OpenAPI/Swagger documentation, and shared TypeScript types.

## Project Overview

Vido is organized as an Nx monorepo that includes:

- **Frontend**: React application with Vite, TanStack Router, TanStack Query, and Tailwind CSS
- **Backend**: Go REST API with Gin framework, structured logging, error handling, CORS middleware, and automatic OpenAPI/Swagger documentation
- **Shared Libraries**: TypeScript type definitions shared between frontend and backend

## Backend Features

- **Gin Framework**: Fast HTTP framework with routing, middleware support, and JSON handling
- **Structured Logging**: JSON-formatted logs using zerolog with request ID tracking
- **Error Handling**: Consistent error response format with panic recovery
- **CORS Middleware**: Configurable cross-origin resource sharing
- **OpenAPI Documentation**: Auto-generated from code annotations with Swagger UI
- **Configuration Management**: Environment-based configuration with sensible defaults
- **Graceful Shutdown**: Proper signal handling and graceful server shutdown
- **Hot Reload**: Automatic recompilation on file changes using Air

## Prerequisites

Before getting started, ensure you have the following installed:

- **Node.js**: >= 20.x (LTS Iron)
  - Download: https://nodejs.org/
  - Or use nvm: `nvm install lts/iron`
- **Go**: >= 1.23
  - Download: https://golang.org/doc/install
- **npm**: Comes bundled with Node.js
- **Make**: (optional, for convenient backend commands)

## Quick Start

### Automated Setup

Run the initialization script to check prerequisites and install all dependencies:

```bash
./init.sh
```

This script will:
- Verify Node.js and Go versions
- Install npm dependencies
- Download Go module dependencies
- Install Go tools (swag for OpenAPI, air for hot reload)
- Create `.env` file from `.env.example` if it doesn't exist

### Manual Setup

If you prefer to set up manually:

```bash
# Install npm dependencies
npm install

# Install Go dependencies and tools
cd apps/api
go mod download
./scripts/install-swag.sh  # Install swag CLI
./scripts/install-air.sh   # Install Air for hot reload
cd ../..
```

## Project Structure

```
vido/
├── apps/
│   ├── web/              # React frontend application
│   │   ├── src/
│   │   ├── project.json  # Nx project configuration
│   │   ├── vite.config.mts
│   │   └── tailwind.config.js
│   │
│   └── api/              # Go backend application
│       ├── cmd/
│       │   └── api/      # Application entry point
│       ├── internal/
│       │   ├── config/   # Configuration management
│       │   ├── middleware/  # HTTP middleware
│       │   └── server/   # HTTP server and routes
│       ├── scripts/      # Utility scripts
│       │   ├── install-swag.sh     # Install swag CLI tool
│       │   ├── install-air.sh      # Install Air hot reload tool
│       │   └── generate-openapi.sh # Regenerate OpenAPI spec
│       ├── docs/         # Generated OpenAPI documentation
│       ├── api/          # OpenAPI spec
│       ├── .env.example  # Example environment configuration
│       ├── .air.toml     # Air hot reload configuration
│       ├── Makefile      # Development commands
│       ├── go.mod
│       └── project.json
│
├── libs/
│   └── shared-types/     # Shared TypeScript types
│       ├── src/
│       └── project.json
│
├── dist/                 # Build outputs
├── node_modules/
├── nx.json              # Nx workspace configuration
├── package.json         # Workspace dependencies
├── tsconfig.base.json   # Base TypeScript configuration
└── init.sh              # Environment setup script
```

## Available Commands

### Development

Start development servers:

```bash
# Start React frontend (http://localhost:4200)
nx serve web

# Start Go API server with hot reload (http://localhost:8080)
cd apps/api && make dev

# Run both concurrently (in separate terminals)
nx serve web & (cd apps/api && make dev)
```

### Building

Build projects for production:

```bash
# Build web app
nx build web

# Build API server
cd apps/api && make build

# Build shared types library
nx build shared-types

# Build all projects in parallel
nx run-many -t build
```

### Backend-Specific Commands

Navigate to `apps/api` and use these Makefile commands:

```bash
# Display available commands
make help

# Build the API binary (automatically runs go mod tidy)
make build

# Run the API server
make run

# Start hot reload development server (recommended for development)
make dev

# Run all tests
make test

# Regenerate OpenAPI specification
make swagger

# Update go.mod and go.sum
make tidy

# Clean build artifacts
make clean
```

### Testing

Run tests for projects:

```bash
# Test web app
nx test web

# Test API server
cd apps/api && make test

# Run tests with coverage
cd apps/api && go test -cover ./...

# Run tests for a specific package
cd apps/api && go test -v ./internal/config

# Run all tests
nx run-many -t test
```

### Linting and Formatting

```bash
# Lint all code
npm run lint

# Fix linting issues
npm run lint:fix

# Format code with Prettier
npm run format

# Check formatting
npm run format:check
```

## Development Workflow

### Working on the Frontend

1. Start the development server:
   ```bash
   nx serve web
   ```

2. The app will be available at http://localhost:4200 with hot module replacement enabled

3. Import shared types from `@vido/shared-types`:
   ```typescript
   import { Movie, ApiResponse } from '@vido/shared-types';
   ```

### Working on the Backend

1. Start the API server with hot reload:
   ```bash
   cd apps/api && make dev
   ```

2. The server will run at http://localhost:8080 with automatic recompilation on file changes

3. Health check endpoint: http://localhost:8080/health

4. View API documentation at: http://localhost:8080/swagger/index.html

### Hot Reload with Air

For the best development experience, use Air for automatic recompilation:

```bash
cd apps/api && make dev
```

Air will:
- Automatically rebuild your application when `.go` files change
- Restart the server with the new binary
- Display color-coded build and runtime logs
- Make changes visible in 1-2 seconds

Configuration is in `apps/api/.air.toml`. See `apps/api/docs/AIR_SETUP.md` for detailed documentation.

**Quick workflow:**
1. Run `make dev` in `apps/api` once
2. Edit any `.go` file and save
3. Air automatically rebuilds and restarts
4. Test your changes immediately!

### Adding Shared Types

1. Edit types in `libs/shared-types/src/lib/`

2. Export from `libs/shared-types/src/index.ts`

3. Build the library:
   ```bash
   nx build shared-types
   ```

4. Types are automatically available via `@vido/shared-types` path alias

### Adding New Backend Endpoints

1. Add your handler function in `apps/api/internal/server/router.go`

2. Document it with swaggo annotations:

```go
// GetUser retrieves a user by ID
// @Summary      Get user by ID
// @Description  Retrieve detailed information about a specific user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  User
// @Failure      404  {object}  middleware.ErrorResponse
// @Failure      500  {object}  middleware.ErrorResponse
// @Router       /api/v1/users/{id} [get]
func (s *Server) getUser(c *gin.Context) {
    // Handler implementation
}
```

3. Regenerate the OpenAPI spec:

```bash
cd apps/api && make swagger
```

## Technology Stack

### Frontend (apps/web)
- **Framework**: React 19
- **Build Tool**: Vite 7
- **Routing**: TanStack Router 1.x
- **Data Fetching**: TanStack Query 5.x
- **Styling**: Tailwind CSS 4.x
- **Testing**: Vitest

### Backend (apps/api)
- **Language**: Go 1.23
- **Framework**: Gin
- **Structured Logging**: zerolog with request ID tracking
- **Error Handling**: Consistent error response format with panic recovery
- **CORS**: gin-cors middleware
- **OpenAPI/Swagger**: Auto-generated documentation with swaggo
- **Hot Reload**: Air for automatic recompilation
- **Configuration**: Environment-based with sensible defaults

### Shared Libraries
- **shared-types**: TypeScript type definitions shared across the monorepo

### Build System
- **Monorepo Tool**: Nx 22.x
- **Package Manager**: npm with workspaces

## Port Configuration

- **Frontend (web)**: http://localhost:4200
- **Backend (api)**: http://localhost:8080

To change ports:

- **Web**: Set `VITE_PORT` environment variable or modify `apps/web/vite.config.mts`
- **API**: Set `PORT` environment variable in `apps/api/.env` before running `make dev` or `make run`

## Backend Configuration

Configuration is managed through environment variables in `apps/api/.env`. See `apps/api/.env.example` for available options:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `ENV` | `development` | Environment (development, production) |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `CORS_ORIGINS` | `*` | Allowed CORS origins (comma-separated) |
| `API_VERSION` | `v1` | API version |

## Error Handling

The server includes custom error types with consistent JSON responses:

```go
// Validation error (400)
c.Error(middleware.NewValidationError("Invalid email format"))

// Not found error (404)
c.Error(middleware.NewNotFoundError("User not found"))

// Internal error (500)
c.Error(middleware.NewInternalError("Database connection failed", err))

// Unauthorized error (401)
c.Error(middleware.NewUnauthorizedError("Authentication required"))

// Forbidden error (403)
c.Error(middleware.NewForbiddenError("Access denied"))
```

All errors return a consistent format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid email format"
  }
}
```

## OpenAPI/Swagger Documentation

The API documentation is automatically generated from code annotations using [swaggo](https://github.com/swaggo/swag).

### Regenerating the Spec

After modifying endpoint annotations:

```bash
cd apps/api && make swagger
```

This will:
1. Run `swag init` to regenerate the spec from code annotations
2. Generate `docs/swagger.json` and `docs/swagger.yaml`
3. Copy the spec to `api/openapi.json` for SDK generation

### Viewing Documentation

With the server running, visit:
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Raw JSON spec: `http://localhost:8080/swagger/doc.json`

## API Endpoints

### Health Check

```bash
GET /health
```

Returns the health status of the API server.

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-07T10:30:00Z"
}
```

### API Documentation

```bash
GET /swagger/index.html
```

Interactive Swagger UI for exploring and testing the API.

## Additional Resources

### Nx Documentation
- Nx Official Docs: https://nx.dev
- Nx Commands: https://nx.dev/nx-api/nx/documents/run
- Nx Plugins: https://nx.dev/plugin-features

### Framework Documentation
- React: https://react.dev
- Vite: https://vite.dev
- TanStack Router: https://tanstack.com/router
- TanStack Query: https://tanstack.com/query
- Tailwind CSS: https://tailwindcss.com
- Gin Framework: https://gin-gonic.com
- Go: https://golang.org
- swaggo: https://github.com/swaggo/swag

## Troubleshooting

### Node.js version issues
```bash
# Check your Node.js version
node --version

# Use nvm to switch to correct version
nvm use
```

### Go dependency issues
```bash
# Clean and reinstall Go dependencies
cd apps/api
go clean -modcache
go mod download
```

### Nx cache issues
```bash
# Clear Nx cache
nx reset
```

### Port already in use
```bash
# Kill process using port 4200 (web)
lsof -ti:4200 | xargs kill -9

# Kill process using port 8080 (api)
lsof -ti:8080 | xargs kill -9
```

## Contributing

1. Create a new branch for your feature
2. Make your changes following existing code patterns
3. Run linting and tests before committing
4. Build all projects to ensure nothing is broken
5. Submit a pull request

## License

MIT (Frontend and shared libraries) / Apache 2.0 (Backend)