# Vido - Nx Monorepo

A full-stack media management application built with Nx monorepo architecture, featuring a React frontend and Go backend with shared TypeScript types.

## Project Overview

Vido is organized as an Nx monorepo that includes:

- **Frontend**: React application with Vite, TanStack Router, TanStack Query, and Tailwind CSS
- **Backend**: Go REST API with Gin framework
- **Shared Libraries**: TypeScript type definitions shared between frontend and backend

## Prerequisites

Before getting started, ensure you have the following installed:

- **Node.js**: >= 20.x (LTS Iron)
  - Download: https://nodejs.org/
  - Or use nvm: `nvm install lts/iron`
- **Go**: >= 1.23
  - Download: https://golang.org/doc/install
- **npm**: Comes bundled with Node.js

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

### Manual Setup

If you prefer to set up manually:

```bash
# Install npm dependencies
npm install

# Install Go dependencies
cd apps/api
go mod download
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
│       ├── cmd/api/      # Application entry point
│       ├── internal/     # Private packages
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

# Start Go API server (http://localhost:3000)
nx serve api

# Run both concurrently (in separate terminals)
nx serve web & nx serve api
```

### Building

Build projects for production:

```bash
# Build web app
nx build web

# Build API server
nx build api

# Build shared types library
nx build shared-types

# Build all projects in parallel
nx run-many -t build
```

### Testing

Run tests for projects:

```bash
# Test web app
nx test web

# Test API server
nx test api

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

### Go-Specific Commands

Additional commands for the Go backend:

```bash
# Format Go code
nx fmt api

# Run Go vet
nx lint api

# Tidy Go modules
nx tidy api
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

1. Start the API server:
   ```bash
   nx serve api
   ```

2. The server will run at http://localhost:3000 in debug mode

3. Health check endpoint: http://localhost:3000/health

### Adding Shared Types

1. Edit types in `libs/shared-types/src/lib/`

2. Export from `libs/shared-types/src/index.ts`

3. Build the library:
   ```bash
   nx build shared-types
   ```

4. Types are automatically available via `@vido/shared-types` path alias

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
- **CORS**: gin-cors middleware

### Shared Libraries
- **shared-types**: TypeScript type definitions shared across the monorepo

### Build System
- **Monorepo Tool**: Nx 22.x
- **Package Manager**: npm with workspaces

## Port Configuration

- **Frontend (web)**: http://localhost:4200
- **Backend (api)**: http://localhost:3000

To change ports:

- **Web**: Set `VITE_PORT` environment variable or modify `vite.config.mts`
- **API**: Set `PORT` environment variable before running `nx serve api`

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

# Kill process using port 3000 (api)
lsof -ti:3000 | xargs kill -9
```

## Contributing

1. Create a new branch for your feature
2. Make your changes following existing code patterns
3. Run linting and tests before committing
4. Build all projects to ensure nothing is broken
5. Submit a pull request

## License

MIT
