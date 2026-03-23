# Phase 0 Prerequisites for Epic 7-8

Technical specification for architecture prerequisites that must be completed before Epic 7 (Library Scanning) and Epic 8 (Subtitle Integration) development can begin.

## Current State Summary

- **Last migration**: `017_create_backups_table.go` (next = 018)
- **Movie struct**: Has `ParseStatus`, `FilePath`, `FileSize`, `MetadataSource` — no subtitle fields
- **Series struct**: Has `ParseStatus`, `FilePath`, `MetadataSource` — no subtitle fields
- **Repository methods**: CRUD, FindByID/TMDbID/IMDbID/FilePath, List, SearchByTitle, FullTextSearch, Upsert, Count, GetDistinctGenres, GetYearRange — no BulkCreate, no FindByParseStatus, no subtitle queries
- **SSE pattern**: Exists in `apps/api/internal/events/` (per-task `ChannelEmitter` with Subscribe/Unsubscribe/Emit) and `handlers/parse_progress_handler.go` — but scoped to parse tasks only, no global hub
- **Interfaces**: Defined in `apps/api/internal/repository/interfaces.go` with compile-time checks
- **Registry**: `apps/api/internal/repository/registry.go` — `Repositories` struct with factory functions

---

## 1. Database Migration

**File**: `apps/api/internal/database/migrations/018_add_subtitle_fields.go`

```sql
-- Movies table
ALTER TABLE movies ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
ALTER TABLE movies ADD COLUMN subtitle_path TEXT;
ALTER TABLE movies ADD COLUMN subtitle_language TEXT;
ALTER TABLE movies ADD COLUMN subtitle_last_searched TIMESTAMP;
ALTER TABLE movies ADD COLUMN subtitle_search_score REAL;

-- Series table
ALTER TABLE series ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
ALTER TABLE series ADD COLUMN subtitle_path TEXT;
ALTER TABLE series ADD COLUMN subtitle_language TEXT;
ALTER TABLE series ADD COLUMN subtitle_last_searched TIMESTAMP;
ALTER TABLE series ADD COLUMN subtitle_search_score REAL;

-- Indexes for subtitle queries
CREATE INDEX IF NOT EXISTS idx_movies_subtitle_status ON movies(subtitle_status);
CREATE INDEX IF NOT EXISTS idx_series_subtitle_status ON series(subtitle_status);
```

Follow the existing migration pattern: struct with `migrationBase`, `init()` registration via `Register()`, `Up(*sql.Tx) error` / `Down(*sql.Tx) error` methods. See `006_media_entities_enhancement.go` for the ALTER TABLE pattern.

---

## 2. Model Updates

### SubtitleStatus type (in `models/movie.go`, alongside `ParseStatus`)

```go
type SubtitleStatus string

const (
    SubtitleStatusNotSearched SubtitleStatus = "not_searched"
    SubtitleStatusSearching   SubtitleStatus = "searching"
    SubtitleStatusFound       SubtitleStatus = "found"
    SubtitleStatusNotFound    SubtitleStatus = "not_found"
)
```

### Movie struct — add after `MetadataSource` block

```go
// Subtitle tracking fields
SubtitleStatus      SubtitleStatus  `db:"subtitle_status" json:"subtitleStatus"`
SubtitlePath        sql.NullString  `db:"subtitle_path" json:"subtitlePath,omitempty"`
SubtitleLanguage    sql.NullString  `db:"subtitle_language" json:"subtitleLanguage,omitempty"`
SubtitleLastSearched sql.NullTime   `db:"subtitle_last_searched" json:"subtitleLastSearched,omitempty"`
SubtitleSearchScore sql.NullFloat64 `db:"subtitle_search_score" json:"subtitleSearchScore,omitempty"`
```

### Series struct — add after `MetadataSource` block

```go
// Subtitle tracking fields
SubtitleStatus      SubtitleStatus  `db:"subtitle_status" json:"subtitleStatus"`
SubtitlePath        sql.NullString  `db:"subtitle_path" json:"subtitlePath,omitempty"`
SubtitleLanguage    sql.NullString  `db:"subtitle_language" json:"subtitleLanguage,omitempty"`
SubtitleLastSearched sql.NullTime   `db:"subtitle_last_searched" json:"subtitleLastSearched,omitempty"`
SubtitleSearchScore sql.NullFloat64 `db:"subtitle_search_score" json:"subtitleSearchScore,omitempty"`
```

**Note**: `SubtitleStatus` type is defined in `models/movie.go` since that file already defines `ParseStatus` and `MetadataSource` — shared types for both Movie and Series.

---

## 3. Repository Interface Extensions

Add to `MovieRepositoryInterface` in `apps/api/internal/repository/interfaces.go`:

```go
// BulkCreate inserts multiple movies in a single transaction
// Needed by: Epic 7 (scan results batch insert)
BulkCreate(ctx context.Context, movies []*models.Movie) error

// FindByParseStatus retrieves movies matching a given parse status
// Needed by: Epic 7 (find pending/failed items for re-scan)
FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error)

// UpdateSubtitleStatus updates subtitle-related fields for a movie
// Needed by: Epic 8 (subtitle search results)
UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error

// FindBySubtitleStatus retrieves movies matching a given subtitle status
// Needed by: Epic 8 (find items needing subtitles)
FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error)

// FindNeedingSubtitleSearch retrieves movies that have not been searched for subtitles
// or were last searched before the given time threshold
// Needed by: Epic 8 (batch subtitle search scheduler)
FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error)
```

Add the same five methods to `SeriesRepositoryInterface` with `*models.Series` types.

### Implementation notes

- `BulkCreate`: Use a transaction (`tx.ExecContext` in a loop). Set timestamps and marshal genres per item. Reuse existing `Create` logic inside `tx`.
- `FindByParseStatus`: `SELECT ... FROM movies WHERE parse_status = ?`
- `UpdateSubtitleStatus`: Single `UPDATE movies SET subtitle_status=?, subtitle_path=?, subtitle_language=?, subtitle_search_score=?, subtitle_last_searched=?, updated_at=? WHERE id=?`
- `FindBySubtitleStatus`: `SELECT ... FROM movies WHERE subtitle_status = ?`
- `FindNeedingSubtitleSearch`: `SELECT ... FROM movies WHERE subtitle_status = 'not_searched' OR (subtitle_last_searched IS NOT NULL AND subtitle_last_searched < ?)`

Update the compile-time assertions at the bottom of `interfaces.go` — they will enforce implementation. Add implementations to both `movie_repository.go` and `series_repository.go`.

---

## 4. SSE Hub

The existing `events.ChannelEmitter` is scoped to parse tasks. Epic 7-8 need a global SSE hub for broadcasting multiple event types to all connected frontend clients.

### Package: `apps/api/internal/sse/`

#### `hub.go` — Core types

```go
package sse

type EventType string

const (
    EventScanProgress     EventType = "scan_progress"
    EventSubtitleProgress EventType = "subtitle_progress"
    EventNotification     EventType = "notification"
)

type Event struct {
    Type EventType   `json:"type"`
    Data interface{} `json:"data"`
}

type Client struct {
    ID     string
    Events chan Event
}

type Hub struct {
    mu         sync.RWMutex
    clients    map[string]*Client
    broadcast  chan Event
    register   chan *Client
    unregister chan *Client
    closed     bool
}
```

Key methods:
- `NewHub() *Hub` — creates hub, starts run loop in goroutine
- `Run()` — select loop: register/unregister clients, broadcast events
- `Register() *Client` — returns new client with buffered channel
- `Unregister(client *Client)` — removes client, closes channel
- `Broadcast(event Event)` — sends event to all clients (non-blocking per client)
- `Close()` — shuts down hub

#### `handler.go` — HTTP SSE endpoint

```go
// Handler returns a Gin handler for SSE streaming: GET /api/v1/events
func Handler(hub *Hub) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("Connection", "keep-alive")

        client := hub.Register()
        defer hub.Unregister(client)

        c.Stream(func(w io.Writer) bool {
            select {
            case event, ok := <-client.Events:
                if !ok { return false }
                data, _ := json.Marshal(event)
                c.SSEvent("message", string(data))
                return true
            case <-c.Request.Context().Done():
                return false
            }
        })
    }
}
```

### Wiring in `main.go`

```go
// After existing event emitter initialization:
sseHub := sse.NewHub()
defer sseHub.Close()

// Register SSE route (before apiV1 route group closing brace):
apiV1.GET("/events", sse.Handler(sseHub))

// Pass sseHub to services that need to emit events (scan service, subtitle service)
```

### CORS note

The existing CORS config already allows GET. SSE connections use standard HTTP, so no additional CORS changes are needed.

---

## 5. Implementation Order

### Step 1: Migration + Models (blocks everything)
- `018_add_subtitle_fields.go` — DB migration
- `SubtitleStatus` type + model fields in `movie.go` and `series.go`
- **Blocks**: Epic 7 and Epic 8

### Step 2: Repository Extensions (blocks Epic 7)
- Add `BulkCreate` and `FindByParseStatus` to interfaces + implementations
- Add `UpdateSubtitleStatus`, `FindBySubtitleStatus`, `FindNeedingSubtitleSearch` to interfaces + implementations
- Update compile-time assertions
- **`BulkCreate` + `FindByParseStatus`** block Epic 7
- **`UpdateSubtitleStatus` + `FindBySubtitleStatus` + `FindNeedingSubtitleSearch`** block Epic 8

### Step 3: SSE Hub (blocks Epic 7 progress UI)
- `apps/api/internal/sse/hub.go` + `handler.go`
- Wire into `main.go`
- **Blocks**: Epic 7 scan progress streaming, Epic 8 subtitle progress streaming
- Can be built in parallel with Step 2

### Dependency Graph

```
Step 1 (Migration + Models)
  |
  +---> Step 2 (Repository Extensions)
  |       |
  |       +---> Epic 7 (needs BulkCreate, FindByParseStatus)
  |       +---> Epic 8 (needs subtitle query methods)
  |
  +---> Step 3 (SSE Hub)
          |
          +---> Epic 7 (scan_progress events)
          +---> Epic 8 (subtitle_progress events)
```

Steps 2 and 3 are independent of each other and can be developed in parallel after Step 1 is complete.

---

## Files to Create/Modify

| Action | File |
|--------|------|
| CREATE | `apps/api/internal/database/migrations/018_add_subtitle_fields.go` |
| CREATE | `apps/api/internal/sse/hub.go` |
| CREATE | `apps/api/internal/sse/handler.go` |
| MODIFY | `apps/api/internal/models/movie.go` — add `SubtitleStatus` type + Movie fields |
| MODIFY | `apps/api/internal/models/series.go` — add Series subtitle fields |
| MODIFY | `apps/api/internal/repository/interfaces.go` — add 5 methods to each interface |
| MODIFY | `apps/api/internal/repository/movie_repository.go` — implement 5 new methods |
| MODIFY | `apps/api/internal/repository/series_repository.go` — implement 5 new methods |
| MODIFY | `apps/api/cmd/api/main.go` — wire SSE hub + route |
