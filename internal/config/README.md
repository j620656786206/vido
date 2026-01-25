# Configuration Package

This package provides environment-based configuration for the Vido API server.

## Usage

```go
import "github.com/alexyu/vido/internal/config"

func main() {
    // Load configuration from environment variables
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Use configuration
    fmt.Printf("Starting server on port %s\n", cfg.Port)
    fmt.Printf("CORS origins: %v\n", cfg.CORSOrigins)
    fmt.Printf("Log level: %s\n", cfg.LogLevel)
}
```

## Configuration Variables

All configuration is loaded from environment variables with sensible defaults:

| Variable       | Description                     | Default                 | Example                                       |
| -------------- | ------------------------------- | ----------------------- | --------------------------------------------- |
| `PORT`         | Server port                     | `8080`                  | `8080`                                        |
| `ENV`          | Environment mode                | `development`           | `production`, `staging`, `development`        |
| `CORS_ORIGINS` | Comma-separated allowed origins | `http://localhost:3000` | `https://example.com,https://app.example.com` |
| `LOG_LEVEL`    | Logging level                   | `info`                  | `debug`, `info`, `warn`, `error`              |
| `API_VERSION`  | API version prefix              | `v1`                    | `v1`, `v2`                                    |

## Environment Files

The project uses `.env` files for local development:

- `.env.example` - Template with all available variables (committed to git)
- `.env` - Local environment file (git-ignored, create from example)

To set up your local environment:

```bash
cp .env.example .env
# Edit .env with your local values
```

## Helper Methods

The `Config` struct provides several helper methods:

- `IsDevelopment()` - Returns true if running in development mode
- `IsProduction()` - Returns true if running in production mode
- `GetPort()` - Returns port as an integer
- `GetAddress()` - Returns formatted address (e.g., `:8080`)

## Validation

The configuration loader validates:

- Log level must be one of: `debug`, `info`, `warn`, `error`
- CORS origins are trimmed of whitespace
- Invalid values return errors on load

## Testing

Run the configuration tests:

```bash
go test ./internal/config
```
