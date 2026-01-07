# Vido API

Go-based REST API server for the Vido media management application.

## Structure

```
apps/api/
├── cmd/api/          # Application entry point
│   └── main.go       # Main server initialization
├── internal/         # Private application packages
│   └── (TBD)         # handlers, services, models, etc.
├── go.mod            # Go module definition
└── README.md         # This file
```

## Technology Stack

- **Framework**: Gin (HTTP web framework)
- **Language**: Go 1.23+

## Development

The API server will be configured with:
- Gin HTTP server
- CORS middleware
- Health check endpoints
- Configurable port (default: 8080)

Development commands will be configured via Nx targets in the next subtask.
