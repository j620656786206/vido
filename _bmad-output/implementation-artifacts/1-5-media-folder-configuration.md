# Story 1.5: Media Folder Configuration

Status: ready-for-dev

## Story

As a **NAS user**,
I want to **configure which folders contain my media files**,
So that **Vido knows where to scan for movies and TV shows**.

## Acceptance Criteria

1. **Given** the user sets `VIDO_MEDIA_DIRS=/movies,/tv,/anime`
   **When** the application starts
   **Then** it validates that each path exists and is accessible
   **And** it stores the configured paths for future scanning operations

2. **Given** a configured media path does not exist
   **When** the application starts
   **Then** it logs a warning about the inaccessible path
   **And** it continues starting with the valid paths (graceful degradation)

3. **Given** no media directories are configured
   **When** the application starts
   **Then** it logs a notice that no media directories are set
   **And** the application starts successfully (search-only mode)

4. **Given** media directories are configured
   **When** a user views the settings page
   **Then** they see the list of configured media directories
   **And** they see the accessibility status of each directory (accessible/not found)

## Tasks / Subtasks

### Task 1: Create Media Config Package (AC: #1, #2, #3)
- [ ] 1.1 Create `apps/api/internal/media/config.go`
- [ ] 1.2 Implement `MediaConfig` struct with `Directories []MediaDirectory`
- [ ] 1.3 Implement `MediaDirectory` struct: `Path`, `Type`, `Status`, `FileCount`
- [ ] 1.4 Parse `VIDO_MEDIA_DIRS` comma-separated paths from environment

### Task 2: Implement Directory Validation (AC: #1, #2)
- [ ] 2.1 Create `apps/api/internal/media/validator.go`
- [ ] 2.2 Implement `ValidateDirectory(path string) (MediaDirectoryStatus, error)`
- [ ] 2.3 Check: path exists, is directory, is readable
- [ ] 2.4 Return status: `Accessible`, `NotFound`, `NotDirectory`, `NotReadable`

### Task 3: Implement Graceful Degradation (AC: #2, #3)
- [ ] 3.1 On startup, validate all configured directories
- [ ] 3.2 Log warning for each inaccessible directory (don't fail)
- [ ] 3.3 Store only valid directories for scanning operations
- [ ] 3.4 If no directories configured, log notice and continue (search-only mode)

### Task 4: Create Media Service (AC: #1, #4)
- [ ] 4.1 Create `apps/api/internal/services/media_service.go`
- [ ] 4.2 Implement `MediaServiceInterface` with:
  - `GetConfiguredDirectories() []MediaDirectory`
  - `ValidateDirectories() []MediaDirectory`
  - `GetDirectoryStatus(path string) MediaDirectoryStatus`
- [ ] 4.3 Cache directory validation results (refresh on demand)

### Task 5: Create Media Handler for Settings API (AC: #4)
- [ ] 5.1 Create `apps/api/internal/handlers/media_handler.go`
- [ ] 5.2 Implement `GET /api/v1/settings/media-directories` endpoint
- [ ] 5.3 Return list of directories with status and file counts
- [ ] 5.4 Follow existing handler patterns (service injection)

### Task 6: Update Config Integration (AC: #1)
- [ ] 6.1 Update `apps/api/internal/config/config.go` to include `MediaDirs`
- [ ] 6.2 Integrate with Story 1.3's environment variable system
- [ ] 6.3 Add media directory status to startup logging

### Task 7: Write Tests (AC: #1, #2, #3, #4)
- [ ] 7.1 Create `apps/api/internal/media/config_test.go`
- [ ] 7.2 Test comma-separated parsing
- [ ] 7.3 Test directory validation (mock filesystem)
- [ ] 7.4 Test graceful degradation with mixed valid/invalid paths
- [ ] 7.5 Test search-only mode (no directories configured)

## Dev Notes

### Current Implementation Status

**Does NOT Exist (to be created):**
- `apps/api/internal/media/` - Media configuration package
- `apps/api/internal/handlers/media_handler.go` - Media settings API
- Media directory validation logic

**Already Exists (extend):**
- `apps/api/internal/config/config.go` - Config struct (add MediaDirs)
- `apps/api/internal/services/` - Service layer pattern
- `apps/api/internal/handlers/` - Handler patterns

### Architecture Requirements

From `project-context.md`:

```
Rule 4: Layered Architecture
✅ Handler → Service → Repository → Database
❌ Handler → Repository (FORBIDDEN)
```

**Note:** Media config is read-only from environment, no database persistence needed.

### Media Directory Configuration Pattern

```go
// media/config.go
package media

type MediaDirectoryStatus string

const (
    StatusAccessible  MediaDirectoryStatus = "accessible"
    StatusNotFound    MediaDirectoryStatus = "not_found"
    StatusNotDir      MediaDirectoryStatus = "not_directory"
    StatusNotReadable MediaDirectoryStatus = "not_readable"
)

type MediaDirectory struct {
    Path      string               `json:"path"`
    Type      string               `json:"type,omitempty"` // movies, tv, anime, mixed
    Status    MediaDirectoryStatus `json:"status"`
    FileCount int                  `json:"file_count,omitempty"`
    Error     string               `json:"error,omitempty"`
}

type MediaConfig struct {
    Directories      []MediaDirectory `json:"directories"`
    ValidCount       int              `json:"valid_count"`
    TotalCount       int              `json:"total_count"`
    SearchOnlyMode   bool             `json:"search_only_mode"`
}

// LoadMediaConfig loads and validates media directories from environment
func LoadMediaConfig() *MediaConfig {
    rawDirs := os.Getenv("VIDO_MEDIA_DIRS")
    if rawDirs == "" {
        slog.Info("No media directories configured, running in search-only mode")
        return &MediaConfig{
            Directories:    []MediaDirectory{},
            SearchOnlyMode: true,
        }
    }

    paths := strings.Split(rawDirs, ",")
    config := &MediaConfig{
        Directories: make([]MediaDirectory, 0, len(paths)),
        TotalCount:  len(paths),
    }

    for _, p := range paths {
        path := strings.TrimSpace(p)
        if path == "" {
            continue
        }

        dir := validateDirectory(path)
        config.Directories = append(config.Directories, dir)

        if dir.Status == StatusAccessible {
            config.ValidCount++
        }
    }

    config.SearchOnlyMode = config.ValidCount == 0
    return config
}
```

### Directory Validation Pattern

```go
// media/validator.go
package media

import (
    "os"
    "path/filepath"
    "log/slog"
)

func validateDirectory(path string) MediaDirectory {
    dir := MediaDirectory{
        Path: path,
        Type: inferMediaType(path),
    }

    info, err := os.Stat(path)
    if err != nil {
        if os.IsNotExist(err) {
            dir.Status = StatusNotFound
            dir.Error = "directory does not exist"
            slog.Warn("Media directory not found",
                "path", path,
                "recommendation", "Check if the path is correctly mounted in Docker")
        } else {
            dir.Status = StatusNotReadable
            dir.Error = err.Error()
            slog.Warn("Media directory not accessible",
                "path", path,
                "error", err)
        }
        return dir
    }

    if !info.IsDir() {
        dir.Status = StatusNotDir
        dir.Error = "path is not a directory"
        slog.Warn("Media path is not a directory", "path", path)
        return dir
    }

    // Check readability by attempting to list contents
    entries, err := os.ReadDir(path)
    if err != nil {
        dir.Status = StatusNotReadable
        dir.Error = "cannot read directory contents"
        slog.Warn("Cannot read media directory", "path", path, "error", err)
        return dir
    }

    dir.Status = StatusAccessible
    dir.FileCount = len(entries)
    slog.Info("Media directory validated",
        "path", path,
        "type", dir.Type,
        "file_count", dir.FileCount)

    return dir
}

// inferMediaType guesses media type from path name
func inferMediaType(path string) string {
    base := strings.ToLower(filepath.Base(path))

    switch {
    case strings.Contains(base, "movie"):
        return "movies"
    case strings.Contains(base, "tv") || strings.Contains(base, "series"):
        return "tv"
    case strings.Contains(base, "anime"):
        return "anime"
    default:
        return "mixed"
    }
}
```

### Graceful Degradation Pattern

```go
// Startup logging with graceful degradation
func logMediaConfigStatus(config *MediaConfig) {
    if config.SearchOnlyMode {
        slog.Info("Running in search-only mode",
            "reason", "no accessible media directories configured",
            "recommendation", "Set VIDO_MEDIA_DIRS to enable library features")
        return
    }

    slog.Info("Media directories loaded",
        "total", config.TotalCount,
        "valid", config.ValidCount,
        "search_only_mode", config.SearchOnlyMode)

    for _, dir := range config.Directories {
        if dir.Status != StatusAccessible {
            slog.Warn("Media directory unavailable",
                "path", dir.Path,
                "status", dir.Status,
                "error", dir.Error)
        }
    }
}
```

### Media Service Pattern

```go
// services/media_service.go
package services

type MediaServiceInterface interface {
    GetConfiguredDirectories() []media.MediaDirectory
    RefreshDirectoryStatus() *media.MediaConfig
    GetConfig() *media.MediaConfig
}

type MediaService struct {
    config *media.MediaConfig
    mu     sync.RWMutex
}

func NewMediaService() *MediaService {
    config := media.LoadMediaConfig()
    media.logMediaConfigStatus(config)

    return &MediaService{
        config: config,
    }
}

func (s *MediaService) GetConfiguredDirectories() []media.MediaDirectory {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.config.Directories
}

func (s *MediaService) RefreshDirectoryStatus() *media.MediaConfig {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.config = media.LoadMediaConfig()
    return s.config
}

func (s *MediaService) GetConfig() *media.MediaConfig {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.config
}
```

### Media Handler Pattern

```go
// handlers/media_handler.go
package handlers

type MediaHandler struct {
    service services.MediaServiceInterface
}

func NewMediaHandler(service services.MediaServiceInterface) *MediaHandler {
    return &MediaHandler{service: service}
}

// GET /api/v1/settings/media-directories
func (h *MediaHandler) GetMediaDirectories(c *gin.Context) {
    config := h.service.GetConfig()
    SuccessResponse(c, config)
}

// POST /api/v1/settings/media-directories/refresh
func (h *MediaHandler) RefreshMediaDirectories(c *gin.Context) {
    config := h.service.RefreshDirectoryStatus()
    SuccessResponse(c, config)
}
```

### API Response Format

```json
{
  "success": true,
  "data": {
    "directories": [
      {
        "path": "/media/movies",
        "type": "movies",
        "status": "accessible",
        "file_count": 1234
      },
      {
        "path": "/media/tv",
        "type": "tv",
        "status": "accessible",
        "file_count": 567
      },
      {
        "path": "/media/downloads",
        "type": "mixed",
        "status": "not_found",
        "error": "directory does not exist"
      }
    ],
    "valid_count": 2,
    "total_count": 3,
    "search_only_mode": false
  }
}
```

### File Locations

| Component | Path |
|-----------|------|
| Media config | `apps/api/internal/media/config.go` |
| Directory validator | `apps/api/internal/media/validator.go` |
| Media service | `apps/api/internal/services/media_service.go` |
| Media handler | `apps/api/internal/handlers/media_handler.go` |
| Tests | `apps/api/internal/media/config_test.go` |

### Naming Conventions

From architecture documentation:

| Element | Pattern | Example |
|---------|---------|---------|
| Packages | lowercase singular | `media` |
| Structs | PascalCase | `MediaConfig`, `MediaDirectory` |
| Status constants | StatusPascalCase | `StatusAccessible`, `StatusNotFound` |
| Files | snake_case.go | `media_service.go` |

### Project Structure Notes

Target directory structure after this story:

```
apps/api/internal/
├── media/                       # NEW DIRECTORY
│   ├── config.go                # Media configuration
│   ├── validator.go             # Directory validation
│   └── config_test.go           # Tests
├── services/
│   └── media_service.go         # NEW: Media service
├── handlers/
│   └── media_handler.go         # NEW: Settings API
```

### Docker Volume Mapping

From Story 1.2 Docker Compose:
```yaml
volumes:
  - ${MEDIA_PATH:-./media}:/media:ro  # Read-only mount
```

**Note:** Media directories are mounted read-only. The application only reads metadata, never modifies files.

### Environment Variable

From Story 1.3:
```
VIDO_MEDIA_DIRS=/movies,/tv,/anime
```

- Comma-separated list of paths
- Each path is validated on startup
- Invalid paths are logged but don't block startup

### Testing Strategy

```go
func TestLoadMediaConfig_ValidDirectories(t *testing.T) {
    // Create temp directories
    dir1 := t.TempDir()
    dir2 := t.TempDir()

    t.Setenv("VIDO_MEDIA_DIRS", dir1+","+dir2)

    config := media.LoadMediaConfig()

    assert.Equal(t, 2, config.TotalCount)
    assert.Equal(t, 2, config.ValidCount)
    assert.False(t, config.SearchOnlyMode)
}

func TestLoadMediaConfig_MixedValidity(t *testing.T) {
    validDir := t.TempDir()
    invalidDir := "/nonexistent/path"

    t.Setenv("VIDO_MEDIA_DIRS", validDir+","+invalidDir)

    config := media.LoadMediaConfig()

    assert.Equal(t, 2, config.TotalCount)
    assert.Equal(t, 1, config.ValidCount)
    assert.False(t, config.SearchOnlyMode)

    // Check statuses
    assert.Equal(t, media.StatusAccessible, config.Directories[0].Status)
    assert.Equal(t, media.StatusNotFound, config.Directories[1].Status)
}

func TestLoadMediaConfig_NoDirectories(t *testing.T) {
    os.Unsetenv("VIDO_MEDIA_DIRS")

    config := media.LoadMediaConfig()

    assert.Equal(t, 0, config.TotalCount)
    assert.True(t, config.SearchOnlyMode)
}

func TestInferMediaType(t *testing.T) {
    tests := []struct {
        path     string
        expected string
    }{
        {"/media/movies", "movies"},
        {"/data/Movies", "movies"},
        {"/tv-shows", "tv"},
        {"/anime", "anime"},
        {"/downloads", "mixed"},
    }

    for _, tt := range tests {
        result := media.inferMediaType(tt.path)
        assert.Equal(t, tt.expected, result)
    }
}
```

### Previous Story Intelligence

From Story 1.3:
- `VIDO_MEDIA_DIRS` environment variable defined
- Config loading pattern with source tracking
- Validation pattern established

From Story 1.2:
- Docker volume mount: `/media:ro`
- Volume configuration in docker-compose.yml

**Dependency:** This story uses `VIDO_MEDIA_DIRS` from Story 1.3's config.

### References

- [Source: project-context.md#Rule 4: Layered Architecture]
- [Source: architecture.md#Graceful Degradation]
- [Source: epics.md#Story 1.5: Media Folder Configuration]
- [Source: apps/api/internal/config/config.go]

### NFR Traceability

| NFR | Requirement | Implementation |
|-----|-------------|----------------|
| FR49 | Configure media folder locations | VIDO_MEDIA_DIRS env var |
| NFR-R2 | Graceful degradation | Continue with valid paths only |
| NFR-U3 | Sensible defaults | Search-only mode when no dirs |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

