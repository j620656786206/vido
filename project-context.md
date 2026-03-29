# Vido Project Context - AI Agent Quick Reference

> **Purpose:** Mandatory reading for all AI agents before implementing ANY code. This document ensures consistency across all implementations.

**Full Documentation:** See `_bmad-output/planning-artifacts/architecture/index.md` for complete architectural decisions and patterns (sharded into ~20 focused files).

**Last Updated:** 2026-03-28 (Rule 18: API boundary case transformation)
**Architecture Status:** ✅ Validated and Ready for Implementation (5,463 lines, 8 steps completed)

---

## 🚨 CRITICAL: Current Project State

### Dual Backend Architecture Problem

**The project currently has TWO separate Go backends with divided features:**

1. **Root Backend** (`/cmd` + `/internal`)
   - ✅ Has: Swagger, zerolog logging, TMDb client, advanced middleware
   - ❌ Missing: NO database, NO data persistence

2. **Apps Backend** (`/apps/api`)
   - ✅ Has: SQLite database, migrations, repository pattern
   - ❌ Missing: NO Swagger, NO structured logging, NO TMDb integration

### ⚠️ ALL NEW CODE MUST GO TO: `/apps/api`

**Consolidation Plan (5 Phases):**

**Phase 1: Backend Consolidation** (⭐ CURRENT PRIORITY)

- **Step 1.1:** Migrate TMDb client: `/internal/tmdb/` → `/apps/api/internal/tmdb/` (update to use slog)
- **Step 1.2:** Migrate Swagger: `/cmd/api/main.go` → `/apps/api/main.go` + `/apps/api/docs/`
- **Step 1.3:** Migrate middleware: `/internal/middleware/` → `/apps/api/internal/middleware/`

**Phase 2-5:** Implement architectural decisions, frontend alignment, core features, and testing.
See `_bmad-output/planning-artifacts/architecture/consolidation-refactoring-plan.md` for complete 5-phase roadmap.

**Root backend** (`/cmd`, `/internal`) will be archived to `/archive/` after Phase 1 completion.
**DO NOT add code to `/cmd` or root `/internal`** - these are deprecated.

---

## 🎯 Core Architectural Decisions (MANDATORY)

### 1. CSS Framework: Tailwind CSS v3.x

- **Use:** Utility-first classes for all styling
- **Config:** `/apps/web/tailwind.config.js`
- **Why:** Bundle size optimization, design system consistency

### 2. Testing Infrastructure

- **Backend:** Go testing + testify (coverage >80%)
- **Frontend:** Vitest + React Testing Library (coverage >70%)
- **E2E Feature-level:** Playwright (328 tests, runs in CI/nightly)
- **E2E Journey-level:** TestSprite (journey tests against deployed NAS at `http://192.168.50.52:8088`, manual trigger after deploy). 62 test cases across 6 P0 journeys, production server mode. Plan v4-regenerated 2026-03-27 for Epic 7+8. Test plan: `testsprite_tests/`. Baseline strategy: regenerate on deploy, mark `intentional-change` for bugfix breaks.
- **Pattern:** Co-located tests (`*_test.go`, `*.spec.tsx`)

### 3. Authentication — REMOVED (v4)

> **v4 Decision:** Vido v4 is single-user with no authentication required. Multi-user support is deferred to v5.0. All auth-related code, middleware, and configuration have been removed from scope.

### 4. Caching: Tiered (Memory + SQLite)

- **Tier 1:** In-memory (bigcache/ristretto) for hot data
- **Tier 2:** SQLite `cache_entries` table for persistent cache
- **TTL:** TMDb 24h, AI parsing 30d, images permanent

### 5. Background Tasks: Worker Pool

- **Implementation:** Goroutines + channels (NO external queue)
- **Workers:** 3-5 goroutines
- **Retry:** Exponential backoff (1s → 2s → 4s → 8s)

### 6. Error Handling: slog + Unified AppError

- **Logging:** Go `log/slog` (NOT zerolog, NOT fmt.Println)
- **Errors:** Custom `AppError` type with error codes
- **Format:** Structured JSON logs with sensitive data filtering

### 7. Plugin Architecture: Go Interfaces

**Decision:** Embedded plugin system using Go interfaces for external service integration.

**Interfaces:**

- `MediaServerPlugin` — Plex, Jellyfin (SyncLibrary, GetWatchHistory)
- `DownloaderPlugin` — qBittorrent, NZBGet (AddDownload, GetStatus, Pause, Remove)
- `DVRPlugin` — Sonarr, Radarr (AddMovie, AddSeries, GetQueue)
- Common: `Name()`, `TestConnection(config PluginConfig) error`

**Plugin Manager:** Registration at startup, per-plugin config in SQLite, health check scheduler.
**Location:** `/apps/api/internal/plugins/`

**Rules:**

- All plugin configs must pass `TestConnection()` before being saved
- Plugins must implement graceful degradation (feature disabled when plugin unavailable)
- Plugin health checks run at configurable intervals (default 60s)

### 8. Real-Time Events: SSE Hub

**Decision:** Server-Sent Events for real-time progress updates, replacing polling for downloads/scans/subtitles.

**Architecture:** Single Hub goroutine, fan-out to client channels via `http.Flusher`.
**Broadcast Event Types:** `scan_progress`, `scan_complete`, `scan_cancelled`, `subtitle_progress`, `subtitle_batch_progress`, `notification`
**Control Event Types:** `connected` (handshake), `ping` (keepalive)
**Location:** `/apps/api/internal/sse/`

**Rules:**

- SSE endpoint: `GET /api/v1/events`
- Buffered channels per client (capacity 100), drop on overflow via non-blocking send
- Hub internal channels: broadcast (256), register/unregister (64 each)
- Wire format: `event: {type}\ndata: {json}\n\n` — note: `{json}` is the full `Event` struct (`id`, `type`, `data`), so `type` appears both in the SSE event line and inside the JSON payload
- Reconnection (`Last-Event-ID`) not yet supported; `Event.ID` field exists but is not emitted as SSE `id:` line

**Lazy Connection Pattern** (`handler.go`):

1. Client HTTP request arrives at `GET /api/v1/events`
2. SSE headers are set (`text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`)
3. Client registers with Hub **after** HTTP handshake completes — lazy registration
4. Hub assigns UUID client ID, creates buffered channel (capacity 100)
5. Initial `connected` event sent with `clientId` to confirm handshake
6. Event streaming begins via `c.Stream()` loop
7. **Keepalive:** 30-second `ping` events (with timestamp payload) prevent proxy/client timeouts
8. On client disconnect, deferred `Unregister()` enqueues removal; Hub's `Run()` goroutine then closes the channel and deletes the client

**Non-blocking Broadcast** (`hub.go`):

- `Broadcast()` sends to Hub's broadcast channel (capacity 256) via `select...default` — drops event with warning log if full
- `Run()` goroutine fans out each broadcast to all registered clients via `select...default` — drops per-client if that client's channel is full
- `Close()` uses `atomic.Bool` for once-only shutdown, signals via `done` channel, closes all client channels

### 9a. Media Library Management (ADR 2026-03-29)

**Decision:** Multi-library system with per-folder content type assignment (Route 2 — Progressive Enhancement).

**Data Model:**

- `media_libraries` table: id, name, content_type (movie|series), auto_detect (Phase 2 reserve), sort_order
- `media_library_paths` table: id, library_id (FK), path (UNIQUE), status, last_checked_at
- `movies`/`series` tables: +library_id (FK), +detected_type (Phase 2), +override_type (Phase 2)
- Migration: #020

**API Endpoints:**

- `GET/POST /api/v1/libraries` — list/create libraries
- `PUT/DELETE /api/v1/libraries/:id` — update/delete library
- `POST/DELETE /api/v1/libraries/:id/paths` — add/remove paths
- `POST /api/v1/libraries/:id/paths/refresh` — refresh path statuses

**Service Changes:**

- `MediaService`: reads from `MediaLibraryRepository` (DB), fallback to `VIDO_MEDIA_DIRS` env var
- `ScannerService`: iterates libraries (not raw paths), assigns `library_id` + uses `content_type` for movie/series classification
- `SetupService`: creates library records instead of storing single `media_folder_path`

**Deprecation:**

- `settings.media_folder_path` → replaced by `media_libraries`
- `VIDO_MEDIA_DIRS` → demoted to fallback (log deprecation warning)

**ADR:** `architecture/adr-multi-library-media-management.md`

### 9b. Subtitle Engine Pipeline

**Decision:** Multi-source subtitle search with content-based language detection and OpenCC conversion.

**Pipeline:** search → score → download → post-process (OpenCC 簡繁轉換) → place
**Sources:** Assrt API, Zimuku scraper, OpenSubtitles API
**Scoring:** Language match 40% + Resolution match 20% + Source trust 20% + Group reputation 10% + Downloads 10%
**Location:** `/apps/api/internal/subtitle/`

**Rules:**

- Language detection MUST analyze subtitle file content (not filename) — this fixes Bazarr's core zh-TW bug
- OpenCC conversion direction: s2twp (Simplified → Traditional with Taiwan phrases)
- CN content policy: Skip conversion when `production_countries` contains `CN` (mainland content keeps simplified subtitles — dialogue expressions match audio)
- Conversion is user-overridable: per-search toggle in subtitle dialog, global preference in settings
- Edge cases: Co-productions (multiple countries) default to convert (conservative); already-traditional subtitles pass through unchanged (idempotent)
- Subtitle files use `.zh-Hant.srt` or `.zh-Hans.srt` extension based on final language for Plex/Jellyfin compatibility

---

## 📋 MANDATORY Rules (ALL Agents MUST Follow)

### Rule 1: Single Backend Location

```
✅ ALL backend code → /apps/api
❌ NEVER add code to /cmd or root /internal (deprecated)
```

### Rule 2: Logging with slog ONLY

```go
// ✅ CORRECT
slog.Info("Fetching movie", "movie_id", id)
slog.Error("Failed to parse", "error", err, "filename", filename)

// ❌ WRONG
log.Println("Fetching movie")
fmt.Println("Error:", err)
```

### Rule 3: API Response Format

```json
// ✅ Success
{
  "success": true,
  "data": { ... }
}

// ✅ Error
{
  "success": false,
  "error": {
    "code": "TMDB_TIMEOUT",
    "message": "無法連線到 TMDb API，請稍後再試",
    "suggestion": "檢查網路連線或稍後重試。"
  }
}
```

### Rule 4: Layered Architecture

```
✅ Handler → Service → Repository → Database
❌ Handler → Repository (FORBIDDEN - skip service layer)
```

### Rule 5: TanStack Query for Server State

```typescript
// ✅ CORRECT - Use TanStack Query for API data
const { data: movie } = useQuery({
  queryKey: ['movies', 'detail', movieId],
  queryFn: () => movieService.getMovie(movieId),
});

// ❌ WRONG - Never use Zustand for server data
const movie = useMovieStore((state) => state.movie);
```

### Rule 6: Naming Conventions

```
Database:   snake_case plural (movies, media_files)
API Paths:  /api/v1/{resource} (plural: /api/v1/movies)
Go Files:   snake_case.go (movie_handler.go)
Go Structs: PascalCase (Movie, TMDbClient)
TS Files:   PascalCase.tsx (MovieCard.tsx)
TS Types:   PascalCase (Movie, ApiResponse<T>)
JSON Fields: snake_case (release_date, tmdb_id)
```

### Rule 7: Error Codes System

```
Format: {SOURCE}_{ERROR_TYPE}

TMDB_TIMEOUT, TMDB_NOT_FOUND, TMDB_RATE_LIMIT
AI_TIMEOUT, AI_QUOTA_EXCEEDED
DB_NOT_FOUND, DB_QUERY_FAILED
VALIDATION_REQUIRED_FIELD, VALIDATION_INVALID_FORMAT
SUBTITLE_NOT_FOUND, SUBTITLE_DOWNLOAD_FAILED, SUBTITLE_CONVERT_FAILED
PLUGIN_INIT_FAILED, PLUGIN_HEALTH_CHECK_FAILED, PLUGIN_NOT_CONFIGURED
SCANNER_PERMISSION_DENIED, SCANNER_PARSE_FAILED
SSE_CONNECTION_FAILED
LIBRARY_NOT_FOUND, LIBRARY_DUPLICATE_PATH, LIBRARY_PATH_NOT_ACCESSIBLE
LIBRARY_PATH_NOT_DIRECTORY, LIBRARY_DELETE_HAS_MEDIA
```

### Rule 8: Date/Time Format

```
API:      ISO 8601 with timezone → "2024-01-15T14:30:00Z"
Database: TIMESTAMP (created_at, updated_at)
Go:       time.Time (auto-marshals to ISO 8601)
Display:  toLocaleDateString('zh-TW') → "2024年1月15日"
```

### Rule 9: Test Co-location

```
✅ Backend: movie_handler.go → movie_handler_test.go (same dir)
✅ Frontend: MovieCard.tsx → MovieCard.spec.tsx (same dir)
❌ NO separate tests/ directory
```

### Rule 10: API Versioning

```
✅ /api/v1/movies
✅ /api/v1/events
❌ /movies (missing version)
❌ /api/movie (singular)
```

### Rule 11: Interface Location

```
✅ Define interfaces in services package (e.g., services.MovieServiceInterface)
✅ Handlers import and use interfaces from services package
✅ Repository interfaces in repository package (e.g., repository.MovieRepositoryInterface)
❌ Never duplicate interface definitions across packages
❌ Never define service interfaces in handlers package
```

### Rule 12: Pre-commit Hooks (--no-verify Forbidden)

```
✅ If a pre-commit hook fails, fix the issue and commit normally
✅ Only Alexyu can authorize --no-verify after reviewing the specific error
❌ Dev agents MUST NOT use --no-verify when committing
❌ NEVER bypass pre-commit hooks without explicit user approval
```

### Rule 13: Error Handling Completeness

```go
// ✅ CORRECT — propagate ALL errors
result, err := s.repo.UpdateStatus(ctx, id, status)
if err != nil {
    return fmt.Errorf("update status: %w", err)
}

// ✅ CORRECT — log then return error
if err := s.repo.Save(ctx, item); err != nil {
    slog.Error("Failed to save item", "error", err, "id", item.ID)
    return err
}

// ❌ WRONG — swallowed error (silent failure)
result, err := s.repo.UpdateStatus(ctx, id, status)
if err != nil {
    slog.Error("update failed", "error", err)
    // BUG: no return! Continues with stale result
}

// ❌ WRONG — error ignored entirely
s.repo.UpdateStatus(ctx, id, status)
```

```
Every error return MUST be either:
  1. Propagated to caller (return err / return fmt.Errorf("context: %w", err))
  2. Explicitly logged AND execution halted (return after log)
  3. Intentionally discarded with comment explaining why (rare, needs justification)
Never log an error and continue executing as if it succeeded.
```

### Rule 14: Resource Lifecycle Management

```
Bounded Maps:
  ✅ In-memory maps/caches MUST have an upper bound or eviction policy
  ✅ Use sync.Map with periodic cleanup or fixed-size LRU
  ❌ Unbounded map[string]T that grows forever in long-running processes

Graceful Shutdown:
  ✅ Background goroutines MUST accept context.Context and honor cancellation
  ✅ Use errgroup or WaitGroup to ensure clean shutdown
  ❌ Goroutines that ignore context and run until process kill

Client Caching:
  ✅ Expensive clients (HTTP, DB, API) MUST be created once and reused
  ✅ Cache with config fingerprint — recreate only when config changes
  ❌ Creating new client instances per request or per poll cycle
```

### Rule 15: Pre-commit Self-verification

```
Before marking a story task complete, verify:

main.go Wiring:
  ✅ New handlers/services registered in main.go dependency injection
  ✅ New routes added to router setup
  ❌ Implementing handler but forgetting to wire it up

DB Column Sync:
  ✅ New model fields have corresponding migration ALTER/CREATE
  ✅ Repository INSERT/UPDATE SQL includes ALL model fields
  ❌ Adding model field but missing it in repository SQL or migration

Swagger:
  ✅ New/changed endpoints have updated Swaggo annotations
  ✅ Run swag init if annotations changed
  ❌ Changing API contract without updating docs
```

### Rule 16: Test Assertion Quality

```typescript
// ✅ CORRECT — specific DOM assertion
expect(screen.getByText('Movie Title')).toBeInTheDocument();

// ✅ CORRECT — use toBeAttached for CSS hover/transition elements
expect(overlay).toBeAttached();

// ✅ CORRECT — specific value assertion
expect(result).toEqual({ id: '1', title: 'Test' });

// ❌ WRONG — toBeTruthy for DOM presence (too vague)
expect(screen.getByText('Movie Title')).toBeTruthy();

// ❌ WRONG — toBeVisible for CSS hover-dependent elements (flaky)
expect(overlay).toBeVisible();

// ❌ WRONG — generic boolean for structured data
expect(!!result).toBe(true);
```

```
Use the MOST SPECIFIC assertion matcher available:
  - DOM presence: toBeInTheDocument() (not toBeTruthy)
  - CSS hover/transition elements: toBeAttached() (not toBeVisible)
  - Text content: toHaveTextContent() (not check innerHTML)
  - Equality: toEqual/toStrictEqual (not toBe for objects)
  - Errors: toThrow/toReject (not try-catch with toBeTruthy)
```

### Rule 17: Bilingual Documentation

```
All user-facing documentation MUST be bilingual (EN + zh-TW):

File Naming:
  ✅ doc-name.md (English, primary)
  ✅ doc-name.zh-TW.md (Traditional Chinese)
  ❌ doc-name.zh.md (wrong language tag)
  ❌ Chinese-only doc without English version

Scope:
  ✅ docs/ folder: installation guides, API references, event docs
  ✅ README.md + README.zh-TW.md (when user-facing)
  ❌ Internal docs (_bmad-output/, architecture/) — English only
  ❌ Code comments — English only

Translation Rules:
  ✅ Code blocks, URLs, file paths remain in English
  ✅ Technical terms keep English with optional Chinese annotation
  ✅ Tables preserve same structure in both languages

Reference: Epic 8 Agreement 6
```

### Rule 18: API Boundary Case Transformation

```
All frontend services MUST transform data at the API boundary:

Response (backend → frontend):
  ✅ snakeToCamel(data.data) on every API response
  Already enforced via shared fetchApi in libraryService.ts

Request (frontend → backend):
  ✅ JSON.stringify(camelToSnake(params)) on every POST/PUT body
  ❌ JSON.stringify(params) — sends camelCase keys, backend rejects or ignores

Implementation:
  import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

  // Response: always transform
  return snakeToCamel<T>(data.data);

  // Request: always transform body
  body: JSON.stringify(camelToSnake(params))

Reference: Bugfix sprint 2026-03-28 audit — 4 services found missing camelToSnake
```

---

## 🏗️ Project Structure

```
vido/
├── apps/
│   ├── api/                    # ⭐ SINGLE BACKEND (unified)
│   │   ├── main.go
│   │   ├── internal/
│   │   │   ├── handlers/       # HTTP handlers (Gin)
│   │   │   ├── services/       # Business logic
│   │   │   ├── repository/     # Data access (Repository pattern)
│   │   │   ├── models/         # Domain models
│   │   │   ├── middleware/     # HTTP middleware
│   │   │   ├── tmdb/           # TMDb API client
│   │   │   ├── ai/             # AI provider abstraction
│   │   │   ├── parser/         # Filename parser
│   │   │   ├── cache/          # Cache manager
│   │   │   ├── tasks/          # Background task queue
│   │   │   ├── plugins/        # Plugin interfaces and manager
│   │   │   ├── sse/            # Server-Sent Events hub
│   │   │   ├── subtitle/       # Subtitle engine pipeline
│   │   │   ├── scanner/        # Media library scanner
│   │   │   ├── errors/         # Unified AppError
│   │   │   └── logger/         # slog config
│   │   ├── migrations/         # SQLite migrations
│   │   └── .air.toml
│   │
│   └── web/                    # Frontend (React)
│       ├── src/
│       │   ├── routes/         # TanStack Router
│       │   ├── components/     # Feature-organized
│       │   │   ├── search/
│       │   │   ├── library/
│       │   │   ├── downloads/
│       │   │   └── ui/         # Shared UI
│       │   ├── hooks/          # Custom hooks
│       │   ├── services/       # API clients
│       │   ├── stores/         # Zustand (UI state only)
│       │   └── utils/
│       └── tailwind.config.js
│
├── libs/
│   └── shared-types/           # TypeScript types
│
├── archive/                    # ⚠️ DEPRECATED (old root backend)
│   ├── cmd/
│   └── internal/
│
├── project-context.md          # ⭐ THIS FILE
└── _bmad-output/
    └── planning-artifacts/
        └── architecture/       # Complete architecture doc (sharded)
            └── index.md
```

---

## 📝 Naming Conventions Quick Reference

### Database (SQLite)

| Element     | Pattern                | Example                       | ❌ Anti-pattern         |
| ----------- | ---------------------- | ----------------------------- | ----------------------- |
| Tables      | snake_case plural      | `movies`, `media_files`       | `Movies`, `movie`       |
| Columns     | snake_case             | `tmdb_id`, `created_at`       | `tmdbId`, `createdAt`   |
| Primary Key | `id`                   | `id TEXT PRIMARY KEY`         | `movie_id`              |
| Foreign Key | `{table}_id`           | `library_id`, `movie_id`      | `fk_library`, `movieId` |
| Indexes     | `idx_{table}_{column}` | `idx_movies_tmdb_id`          | `movies_tmdb_index`     |
| Migrations  | `{seq}_{desc}.sql`     | `001_create_movies_table.sql` | `create-movies.sql`     |

### Backend (Go)

| Element    | Pattern              | Example                         | ❌ Anti-pattern             |
| ---------- | -------------------- | ------------------------------- | --------------------------- |
| Packages   | lowercase singular   | `tmdb`, `parser`, `cache`       | `tmdb_client`, `Middleware` |
| Structs    | PascalCase           | `Movie`, `TMDbClient`           | `movie`, `tmdbClient`       |
| Interfaces | PascalCase           | `Repository`, `Cache`           | `IRepository`               |
| Functions  | PascalCase/camelCase | `GetMovieByID`, `parseFilename` | `get_movie_by_id`           |
| Files      | snake_case.go        | `tmdb_client.go`                | `TMDbClient.go`             |

### Frontend (TypeScript/React)

| Element          | Pattern         | Example                       | ❌ Anti-pattern           |
| ---------------- | --------------- | ----------------------------- | ------------------------- |
| Components       | PascalCase      | `SearchBar`, `MovieCard`      | `searchBar`, `search-bar` |
| Component Files  | PascalCase.tsx  | `SearchBar.tsx`               | `search-bar.tsx`          |
| Hooks            | use + camelCase | `useSearch`, `useLibrary`     | `UseSearch`, `searchHook` |
| Hook Files       | use{Name}.ts    | `useSearch.ts`                | `search.hook.ts`          |
| Types/Interfaces | PascalCase      | `Movie`, `ApiResponse<T>`     | `IMovie`, `movieType`     |
| Constants        | SCREAMING_SNAKE | `API_BASE_URL`, `MAX_RETRIES` | `apiBaseUrl`              |

### API Endpoints

| Element | Pattern                    | Example                        | ❌ Anti-pattern              |
| ------- | -------------------------- | ------------------------------ | ---------------------------- |
| Paths   | /api/v{version}/{resource} | `/api/v1/movies`               | `/movie`, `/getMovies`       |
| Methods | RESTful                    | `GET`, `POST`, `PUT`, `DELETE` | `POST /api/v1/movies/update` |
| Params  | {param_name}               | `/api/v1/movies/{id}`          | `/api/v1/movies/:id`         |
| Query   | snake_case                 | `?sort_by=release_date`        | `?sortBy=releaseDate`        |

---

## 🔧 Error Handling Pattern

### Backend (Go)

```go
// Step 1: Create AppError
func (s *MovieService) GetMovieByID(ctx context.Context, id string) (*Movie, error) {
    movie, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, NewDBNotFoundError(err) // AppError
        }
        return nil, NewDBQueryError(err)
    }
    return movie, nil
}

// Step 2: Log with slog
func (h *MovieHandler) GetMovie(c *gin.Context) {
    movie, err := h.service.GetMovieByID(c.Request.Context(), id)
    if err != nil {
        var appErr *AppError
        if !errors.As(err, &appErr) {
            appErr = NewInternalError(err)
        }

        slog.Error("Failed to get movie",
            "error_code", appErr.Code,
            "movie_id", id,
            "error", err,
        )

        ErrorResponse(c, appErr)
        return
    }

    SuccessResponse(c, movie)
}
```

### Frontend (TypeScript)

```typescript
const { data, error, isError } = useQuery({
  queryKey: ['movies', 'detail', movieId],
  queryFn: () => movieService.getMovie(movieId),
  onError: (error: ApiError) => {
    toast.error(error.message, {
      description: error.suggestion,
    });
    console.error(`[${error.code}]`, error.details);
  },
});

if (isError) {
  return <ErrorMessage error={error} />;
}
```

---

## 🔄 State Management Pattern

### Server State (TanStack Query) ✅

```typescript
// Query keys with hierarchy
const movieKeys = {
  all: ['movies'] as const,
  lists: () => [...movieKeys.all, 'list'] as const,
  list: (filters: string) => [...movieKeys.lists(), { filters }] as const,
  details: () => [...movieKeys.all, 'detail'] as const,
  detail: (id: string) => [...movieKeys.details(), id] as const,
};

// Usage
const { data: movie } = useQuery({
  queryKey: movieKeys.detail(movieId),
  queryFn: () => fetchMovie(movieId),
});
```

### Global Client State (Zustand) - UI State ONLY

```typescript
// ✅ ONLY for UI state, NOT server data
interface UIState {
  sidebarOpen: boolean;
  viewMode: 'grid' | 'list';
  toggleSidebar: () => void;
  setViewMode: (mode: 'grid' | 'list') => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: true,
  viewMode: 'grid',
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  setViewMode: (mode) => set({ viewMode: mode }),
}));
```

### Local Component State (useState)

```typescript
// ✅ Form inputs, toggles, local UI state
const [isOpen, setIsOpen] = useState(false);
const [searchTerm, setSearchTerm] = useState('');
```

---

## 🧪 Testing Patterns

### Backend (Go)

```go
// movie_handler_test.go (co-located with movie_handler.go)

func TestMovieHandler_GetMovie(t *testing.T) {
    tests := []struct {
        name       string
        movieID    string
        wantStatus int
        wantError  string
    }{
        {
            name:       "success",
            movieID:    "valid-id",
            wantStatus: http.StatusOK,
        },
        {
            name:       "not found",
            movieID:    "invalid-id",
            wantStatus: http.StatusNotFound,
            wantError:  "DB_NOT_FOUND",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Frontend (TypeScript)

```typescript
// MovieCard.spec.tsx (co-located with MovieCard.tsx)

import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MovieCard } from './MovieCard';

describe('MovieCard', () => {
  it('renders movie title', async () => {
    const queryClient = new QueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <MovieCard movieId="test-id" />
      </QueryClientProvider>
    );

    expect(await screen.findByText('Test Movie')).toBeInTheDocument();
  });
});
```

---

## 🧹 Test Process Cleanup

### Process Lifecycle Rule

**All test-related child processes MUST terminate when the parent process exits.** This applies to:

- **Unit tests (Vitest):** Uses `pool: 'forks'` so workers are child processes that can be force-killed on exit. `teardownTimeout: 5000` prevents indefinite hangs from uncleaned timers/listeners.
- **E2E tests (Playwright):** `globalSetup`/`globalTeardown` track and clean up spawned servers (Go backend, Vite dev server) per session.
- **Go backend:** Started as background process during E2E; cleaned up by teardown.
- **Vite dev server:** Started as background process during E2E; cleaned up by teardown.

### Automatic Cleanup (Built-in)

- Vitest `pool: 'forks'` ensures workers exit even with open handles
- Playwright `globalTeardown` cleans up only processes from the current session
- Safe for multiple Claude Code sessions running tests in parallel
- `nx run web:test` automatically runs `test:cleanup:all` after vitest exits (configured in `apps/web/project.json`)

### Developer Responsibility (MANDATORY)

After **every** test execution — whether via `nx run web:test`, direct `vitest`, or any other method:

1. Run `pnpm run test:cleanup` to verify no orphaned processes remain
2. If orphaned processes are found, run `pnpm run test:cleanup:all` immediately
3. Test execution is NOT considered complete until cleanup verification passes
4. This rule applies regardless of test pass/fail outcome

### Manual Cleanup Commands

```bash
# List orphaned test processes
pnpm run test:cleanup

# Force kill ALL test processes (use with caution)
pnpm run test:cleanup:all
```

**Session Files Location:** `node_modules/.cache/vido-test-sessions/`

**What Gets Cleaned Up:**

- Go backend (`go run ./cmd/api`)
- Vite dev server (`nx serve web`)
- Vitest workers (`node (vitest N)`)
- Playwright test runners
- Processes on ports 8080, 4200

---

## 🧪 TestSprite Journey Test Workflow

### Manual Trigger (After NAS Deploy)

1. **Start localhost proxy:** `node -e "const n=require('net');const s=n.createServer(c=>{const r=n.connect(8088,'192.168.50.52');c.pipe(r);r.pipe(c);c.on('error',()=>r.destroy());r.on('error',()=>c.destroy())});s.listen(8088,'127.0.0.1',()=>console.log('Proxy ready'))" &`
2. **Verify proxy:** `curl -s -o /dev/null -w "%{http_code}" http://localhost:8088/` (expect 200)
3. **Run TestSprite:** `node $(npm root)/.cache/@testsprite/testsprite-mcp/dist/index.js generateCodeAndExecute` or use TestSprite MCP tools in Claude Code
4. **Review results:** Check `testsprite_tests/tmp/raw_report.md` and TestSprite dashboard links
5. **Compare with baseline:** Check `testsprite_tests/testsprite-mcp-test-report.md` for expected pass/fail
6. **Kill proxy when done:** `kill $(lsof -ti:8088)`

### Baseline Strategy

- **Current baseline:** 2026-03-28, 14/30 passed (46.7%)
- After bugfix-1 + bugfix-3: expected ~73%
- When a previously-passing TC fails after a deploy → **regression**, investigate immediately
- When a previously-failing TC passes after a bugfix → **intentional change**, update baseline report

### Key Files

- Test plan: `testsprite_tests/testsprite_frontend_test_plan.json` (40 TCs)
- Test report: `testsprite_tests/testsprite-mcp-test-report.md`
- Raw results: `testsprite_tests/tmp/raw_report.md`
- Config: `testsprite_tests/tmp/config.json`
- Credits: 150/month (Free plan), check via TestSprite MCP `testsprite_check_account_info`

---

## ✅ Pre-Commit Checklist

Before committing code, verify:

**Format & Lint (MANDATORY):**

- [ ] Run `pnpm run format:check` and fix any issues with `pnpm exec prettier --write <files>`
- [ ] Run `pnpm run lint` to check for linting errors

**Code Location & Architecture:**

- [ ] All new code is in `/apps/api` (backend) or `/apps/web` (frontend)
- [ ] No code added to deprecated `/cmd` or root `/internal`
- [ ] Handler → Service → Repository layering respected
- [ ] Interfaces defined in correct package (Rule 11)

**Code Quality:**

- [ ] Logging uses `slog` (NOT zerolog, fmt.Println, or log.Print)
- [ ] API responses use `ApiResponse<T>` wrapper format
- [ ] Error codes follow `{SOURCE}_{ERROR_TYPE}` pattern
- [ ] Dates are ISO 8601 strings in JSON
- [ ] Naming conventions followed (see tables above)
- [ ] Frontend service POST/PUT bodies use `camelToSnake()` (Rule 18)
- [ ] Frontend service responses use `snakeToCamel()` (Rule 18)
- [ ] No swallowed errors — every error propagated or logged+returned (Rule 13)
- [ ] In-memory maps/caches have upper bounds (Rule 14)
- [ ] Background goroutines honor context cancellation (Rule 14)

**Testing (Definition of Done):**

- [ ] `go test ./...` passes with no failures
- [ ] Services test coverage ≥ 80%
- [ ] Handlers test coverage ≥ 70%
- [ ] Tests co-located with source files (`*_test.go`, `*.spec.tsx`)
- [ ] Assertions use specific matchers — `toBeInTheDocument`, `toBeAttached` (Rule 16)

**Integration (Definition of Done):**

- [ ] New Services/Handlers wired up in `main.go` (Rule 15)
- [ ] New model fields reflected in migration SQL and repository (Rule 15)
- [ ] Swagger annotations updated for new/changed endpoints (Rule 15)
- [ ] No binary files or sensitive data staged
- [ ] TanStack Query used for server state (NOT Zustand)

---

## 🤝 Team Agreements (Epic 1 Retrospective)

**Established: 2026-01-17**

These agreements were established during Epic 1 retrospective to improve development quality:

### Agreement 1: 標記完成 = 驗證完成

> "Marking a task complete means it has been **verified**, not just implemented."

- Before marking a task `[x]`, run the code and confirm it works
- Don't rely solely on Code Review to catch unfinished work
- If unsure, test it manually before marking complete

### Agreement 2: 左移品質檢查

> "Shift quality checks LEFT - catch issues during implementation, not review."

- Run `go test -cover` during implementation, not just before commit
- Check coverage targets (Services ≥80%, Handlers ≥70%) while coding
- Code Review should focus on architecture and design, not basic issues

### Agreement 3: project-context.md 是聖經

> "This file is the single source of truth. Read it before implementing."

- All Rules (1-17) must be followed
- When in doubt, check this file first
- Update this file when new patterns are established

---

## 🎯 Quick Decision Guide

### When to use what?

| Use Case               | Technology/Pattern                                    |
| ---------------------- | ----------------------------------------------------- |
| Backend HTTP framework | Gin                                                   |
| Backend logging        | `log/slog` (NOT zerolog)                              |
| Backend testing        | Go testing + testify                                  |
| Backend ORM            | **None** - Use repository pattern with `database/sql` |
| Database               | SQLite with WAL mode                                  |
| API documentation      | Swaggo (OpenAPI/Swagger)                              |
| Frontend framework     | React 19 + TypeScript                                 |
| Frontend routing       | TanStack Router                                       |
| Server state           | TanStack Query v5                                     |
| Client state (UI only) | Zustand                                               |
| Frontend styling       | Tailwind CSS v3.x                                     |
| Frontend testing       | Vitest + React Testing Library                        |
| Build tool (frontend)  | Vite                                                  |
| Monorepo               | Nx                                                    |

---

## 🔗 Complete Documentation

**For full details, see:**

- **Architecture Decisions:** `_bmad-output/planning-artifacts/architecture/index.md`
- **PRD:** `_bmad-output/planning-artifacts/prd.md`
- **UX Design:** `_bmad-output/planning-artifacts/ux-design-specification.md`

**Key Sections in architecture/:**

- Core Architectural Decisions (Step 4)
- Implementation Patterns & Consistency Rules (Step 5)
- Current Implementation Analysis (Brownfield Assessment)
- Consolidation & Refactoring Plan (5 Phases)

---

## ✅ Architecture Validation Summary

**Validation Status:** COMPLETE (2026-01-12)

The complete architecture has been validated for:

- ✅ **Coherence:** All 9 architectural decisions work together without conflicts
- ✅ **Coverage:** All 94 functional requirements are architecturally supported
- ✅ **Readiness:** 47 implementation patterns ensure AI agent consistency

**Key Deliverables:**

- 9 architectural decisions documented with versions and rationale
- 47 implementation patterns preventing AI agent conflicts (see architecture/)
- 400+ files/directories defined in complete project structure
- 5-phase consolidation roadmap from current to target state

**Confidence Level:** HIGH - Ready for implementation with comprehensive guidance.

---

## 🚀 Implementation Workflow

1. **Read this file FIRST** before implementing any feature
2. **Check architecture/** for specific pattern details if needed
3. **Follow the consolidation plan** (Phase 1-5) for refactoring
4. **Verify checklist** before committing code
5. **Write tests** alongside implementation (TDD encouraged)

---

**Questions or clarifications?** Refer to the full architecture document or ask the user.

**Last reminder:** ALL new backend code goes to `/apps/api`. The root backend is deprecated.
