# ADR: Multi-Library Media Management

> **Status:** ACCEPTED
> **Date:** 2026-03-29
> **Deciders:** Alexyu (product owner), Winston (architect)
> **Related PRD:** `prd/prd-multi-library-amendment.md`

---

## Context

The current media folder configuration has a fundamental architectural disconnect:

1. **Setup Wizard** stores `media_folder_path` in settings DB → Scanner never reads it
2. **Scanner** reads `VIDO_MEDIA_DIRS` from environment → no content type information
3. **No per-folder type assignment** → all paths treated identically

Users need to designate folders by content type (movie vs series) to enable proper metadata matching and library organization — a baseline feature in all competing media servers (Plex, Jellyfin, Emby).

## Decision

### Route 2: Progressive Enhancement

Implement multi-library management with manual type assignment in Phase 1, reserving schema fields for future auto-detection capability.

### Data Model

**New tables:**

```sql
CREATE TABLE media_libraries (
    id TEXT PRIMARY KEY,                    -- UUID v4
    name TEXT NOT NULL,                     -- User-facing display name
    content_type TEXT NOT NULL,             -- "movie" | "series"
    auto_detect BOOLEAN NOT NULL DEFAULT false,  -- Phase 2 reserve
    sort_order INTEGER NOT NULL DEFAULT 0,  -- UI ordering
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE media_library_paths (
    id TEXT PRIMARY KEY,                    -- UUID v4
    library_id TEXT NOT NULL,               -- FK → media_libraries.id
    path TEXT NOT NULL UNIQUE,              -- Filesystem path (one path = one library)
    status TEXT NOT NULL DEFAULT 'unknown', -- accessible|not_found|not_readable|not_directory
    last_checked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (library_id) REFERENCES media_libraries(id) ON DELETE CASCADE
);
```

**Modified tables:**

```sql
ALTER TABLE movies ADD COLUMN library_id TEXT REFERENCES media_libraries(id);
ALTER TABLE series ADD COLUMN library_id TEXT REFERENCES media_libraries(id);

-- Phase 2 reserve (nullable, unused in Phase 1)
ALTER TABLE movies ADD COLUMN detected_type TEXT;
ALTER TABLE movies ADD COLUMN override_type TEXT;
ALTER TABLE series ADD COLUMN detected_type TEXT;
ALTER TABLE series ADD COLUMN override_type TEXT;
```

**Migration number:** 020

### API Design

New RESTful endpoints under `/api/v1/libraries`:

| Method | Path | Description | Request Body |
|--------|------|-------------|--------------|
| GET | `/api/v1/libraries` | List all libraries with paths and stats | — |
| POST | `/api/v1/libraries` | Create library | `{ name, content_type, paths[] }` |
| PUT | `/api/v1/libraries/:id` | Update library | `{ name?, content_type?, sort_order? }` |
| DELETE | `/api/v1/libraries/:id` | Delete library | Query: `?remove_media=true` |
| POST | `/api/v1/libraries/:id/paths` | Add path to library | `{ path }` |
| DELETE | `/api/v1/libraries/:id/paths/:pathId` | Remove path | — |
| POST | `/api/v1/libraries/:id/paths/refresh` | Refresh path statuses | — |

**Response format** (GET /libraries):

```json
{
  "libraries": [
    {
      "id": "uuid",
      "name": "我的電影",
      "content_type": "movie",
      "sort_order": 0,
      "paths": [
        {
          "id": "uuid",
          "path": "/media/movies",
          "status": "accessible",
          "last_checked_at": "2026-03-29T10:00:00Z"
        }
      ],
      "media_count": 32
    }
  ]
}
```

**Case transformation (Rule 18):** API boundary uses snake_case (Go structs use PascalCase, JS uses camelCase with automatic transformation).

### Service Layer Changes

**MediaLibraryRepository (NEW):**

```go
type MediaLibraryRepository interface {
    Create(ctx context.Context, library *MediaLibrary) error
    GetByID(ctx context.Context, id string) (*MediaLibrary, error)
    GetAll(ctx context.Context) ([]MediaLibrary, error)
    Update(ctx context.Context, library *MediaLibrary) error
    Delete(ctx context.Context, id string) error
    AddPath(ctx context.Context, path *MediaLibraryPath) error
    RemovePath(ctx context.Context, pathID string) error
    GetPathsByLibraryID(ctx context.Context, libraryID string) ([]MediaLibraryPath, error)
    UpdatePathStatus(ctx context.Context, pathID string, status string) error
}
```

**MediaService (MODIFIED):**

```
Before: NewMediaService(mediaDirs []string)
After:  NewMediaService(repo MediaLibraryRepository, fallbackDirs []string)
```

- Primary source: `MediaLibraryRepository` (DB)
- Fallback: `fallbackDirs` from `VIDO_MEDIA_DIRS` env var (creates default library on first use)
- `GetConfiguredDirectories()` now returns paths with library context

**ScannerService (MODIFIED):**

```
Before: NewScannerService(movieRepo, seriesRepo, mediaDirs []string, ...)
After:  NewScannerService(movieRepo, seriesRepo, libraryRepo MediaLibraryRepository, ...)
```

- Scanner iterates libraries, not raw paths
- Each scanned media item gets `library_id` assigned
- Library's `content_type` determines whether to create movie or series record

**SetupService (MODIFIED):**

- `CompleteSetup` creates `media_library` records instead of storing `media_folder_path`
- Input format changes: `[]{ path, content_type }` instead of single `string`

### Migration Strategy

Migration #020 runs automatically on app startup:

1. Read `media_folder_path` from `settings` table (if exists)
2. Read `VIDO_MEDIA_DIRS` from environment (if set)
3. Merge unique paths from both sources
4. For each path: create a `media_library` with:
   - Name: derived from folder basename (e.g., `/media/movies` → "movies")
   - Content type: `"movie"` (default — user adjusts in Settings UI)
5. Backfill `library_id` on existing `movies` and `series` records by matching file paths
6. Log migration results

**Zero-downtime guarantee:** Migration is additive (new tables + new nullable columns). Existing queries unaffected until service code is updated.

### Deprecation

| Deprecated | Replacement | Timeline |
|------------|-------------|----------|
| `settings.media_folder_path` | `media_libraries` + `media_library_paths` | Immediate |
| `VIDO_MEDIA_DIRS` env var (as primary) | DB-based libraries | Demoted to fallback; log deprecation warning |
| `GET /api/v1/settings/media-directories` | `GET /api/v1/libraries` | Keep both for 1 release cycle |

## Consequences

**Positive:**
- Users can assign content types per folder (matches Plex/Jellyfin baseline)
- Scanner makes correct movie vs series decisions based on library type
- Schema reserves fields for future auto-detection at near-zero cost
- Clean migration path from env-var-based to DB-based configuration

**Negative:**
- Setup Wizard requires rewrite (from single input to multi-library flow)
- ScannerSettings component requires significant modification
- `main.go` initialization changes (MediaService dependency injection)

**Risks:**
- Migration from env var to DB must handle edge cases (empty paths, duplicates)
- Path validation over network mounts may be slow → timeout handling needed

## Alternatives Considered

1. **Route 1 (Conservative):** Same scope minus schema reserve fields. Rejected: near-zero cost to include reserves.
2. **Route 3 (Aggressive):** Full auto-detection + dynamic views + tags. Rejected: solves a problem that rarely exists (99%+ users pre-organize folders).
3. **Keep env var only:** Add type syntax to env var (e.g., `VIDO_MEDIA_DIRS=movie:/movies,series:/tv`). Rejected: poor UX, no runtime management.
