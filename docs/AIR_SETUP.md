# Air Hot Reload Setup

This document explains how to use Air for automatic recompilation and hot reload during development.

## What is Air?

Air (cosmtrek/air) is a live reload tool for Go applications. It watches your `.go` files and automatically rebuilds and restarts your application when changes are detected, dramatically improving the development experience.

## Installation

### Automatic Installation (Recommended)

Run the installation script:

```bash
./scripts/install-air.sh
```

This script will:
- Check for Go installation
- Install Air CLI via `go install`
- Verify the installation
- Check for `.air.toml` configuration

### Manual Installation

```bash
go install github.com/cosmtrek/air@latest
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Add this line to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) to make it permanent.

## Configuration

The project includes a pre-configured `.air.toml` file at the root directory with the following settings:

### Key Configuration Options

- **Build Command**: `go build -o ./tmp/main ./cmd/api`
- **Binary Location**: `./tmp/main` (excluded from git)
- **Watch Extensions**: `.go`, `.tpl`, `.tmpl`, `.html`
- **Excluded Directories**: `assets`, `tmp`, `vendor`, `testdata`, `docs`, `api`, `bin`, `.git`, `.auto-claude`
- **Excluded Files**: `*_test.go` (tests don't trigger rebuild)
- **Build Delay**: 1000ms (debounce for rapid file changes)
- **Build Errors**: Logged to `build-errors.log`

### Configuration File Structure

```toml
[build]
  cmd = "go build -o ./tmp/main ./cmd/api"
  bin = "./tmp/main"
  delay = 1000
  exclude_dir = ["vendor", "tmp", "docs", ...]
  exclude_regex = ["_test.go"]
  include_ext = ["go", "tpl", "tmpl", "html"]

[color]
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"
```

## Usage

### Start Hot Reload Server

Use the Makefile (recommended):

```bash
make dev
```

Or run Air directly:

```bash
air
```

### What Happens

1. Air compiles your application to `./tmp/main`
2. Air starts the compiled binary
3. Air watches for file changes
4. On change detection:
   - Air stops the running binary
   - Air recompiles the application
   - Air restarts the new binary
   - Changes are live in ~1-2 seconds

### Development Workflow

1. Start Air: `make dev`
2. Make changes to any `.go` file
3. Save the file
4. Air automatically rebuilds and restarts
5. Test your changes immediately
6. Repeat!

### Stopping the Server

Press `Ctrl+C` to stop Air and the running application.

## Troubleshooting

### Air Command Not Found

**Problem**: `bash: air: command not found`

**Solution**:
1. Verify installation: `go install github.com/cosmtrek/air@latest`
2. Check PATH: `echo $PATH | grep "$(go env GOPATH)/bin"`
3. If not in PATH, add it:
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

### Build Errors

**Problem**: Air shows build errors and doesn't restart

**Solution**:
1. Check `build-errors.log` in the project root
2. Fix the compilation errors in your code
3. Air will automatically retry the build on next save

### Port Already in Use

**Problem**: `bind: address already in use`

**Solution**:
1. Stop any other instances of the API server
2. Find and kill the process using the port:
   ```bash
   lsof -ti:8080 | xargs kill -9
   ```
3. Or change the PORT in your `.env` file

### Air Not Detecting Changes

**Problem**: Saving files doesn't trigger rebuild

**Solution**:
1. Verify you're editing files with `.go` extension
2. Check that the file isn't in an excluded directory
3. Ensure the file isn't matching `exclude_regex` patterns
4. Try restarting Air

## Performance Tips

### Optimizing Build Times

1. **Exclude unnecessary directories**: The `.air.toml` already excludes common directories
2. **Use build caching**: Go's build cache is enabled by default
3. **Increase delay**: If you make rapid changes, increase `delay` in `.air.toml`

### Reducing CPU Usage

1. **Exclude test files**: Already configured with `exclude_regex = ["_test.go"]`
2. **Limit watched directories**: Only watch directories you're actively editing
3. **Use polling mode**: Set `poll = true` on some file systems (slower but more reliable)

## Comparison with Standard Workflow

### Without Air

```bash
# Edit code
vim internal/server/router.go

# Rebuild
make build

# Run
./bin/api

# Repeat for every change
```

### With Air

```bash
# Start once
make dev

# Edit code - Air handles rebuild and restart automatically
vim internal/server/router.go
# Changes live in 1-2 seconds!
```

## Integration with Other Tools

### With Swagger

Air watches all `.go` files, including annotation changes. However, you need to manually regenerate the OpenAPI spec:

```bash
# In a separate terminal
make swagger
```

### With Tests

Air excludes test files by default. To run tests automatically, use a separate watcher or run tests manually:

```bash
# Manually
make test

# Watch mode (using entr or similar)
find . -name "*.go" | entr -c go test ./...
```

## Advanced Configuration

### Custom Build Flags

Edit `.air.toml` to add build flags:

```toml
[build]
  cmd = "go build -tags debug -race -o ./tmp/main ./cmd/api"
```

### Multiple Build Targets

To watch multiple services, create separate `.air.toml` files:

```bash
air -c .air.api.toml    # For API server
air -c .air.worker.toml # For background worker
```

### Custom Exclude Patterns

Add more patterns to exclude:

```toml
[build]
  exclude_regex = ["_test.go", "_mock.go", ".*\\.tmp"]
```

## Environment Variables

Air inherits environment variables from your shell. To use a `.env` file:

```bash
# Load .env before starting Air
export $(cat .env | xargs) && make dev
```

Or use a tool like `direnv` for automatic environment loading.

## References

- [Air GitHub Repository](https://github.com/cosmtrek/air)
- [Air Documentation](https://github.com/cosmtrek/air/blob/master/README.md)
- [Air Configuration Reference](https://github.com/cosmtrek/air/blob/master/air_example.toml)

## Related Documentation

- [Swaggo Setup](./SWAGGO_SETUP.md) - OpenAPI specification generation
- [README](./README.md) - General documentation generation
- [Project README](../README.md) - Main project documentation
