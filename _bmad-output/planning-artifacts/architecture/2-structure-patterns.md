# 2. Structure Patterns

## 2.1 Project Organization (Monorepo)

**MANDATORY Structure:**

```
vido/
├── apps/
│   ├── api/                    # Backend application (SINGLE SOURCE OF TRUTH)
│   │   ├── main.go
│   │   ├── internal/
│   │   │   ├── handlers/       # HTTP handlers (Gin controllers)
│   │   │   ├── services/       # Business logic layer
│   │   │   ├── repository/     # Data access layer (Repository pattern)
│   │   │   ├── models/         # Domain models (Go structs)
│   │   │   ├── middleware/     # HTTP middleware (auth, CORS, logging)
│   │   │   ├── tmdb/           # TMDb API client
│   │   │   ├── ai/             # AI provider abstraction (Gemini, Claude)
│   │   │   ├── parser/         # Filename parser (regex + AI)
│   │   │   ├── cache/          # Cache manager (tiered)
│   │   │   ├── tasks/          # Background task queue
│   │   │   ├── errors/         # Unified AppError types
│   │   │   └── logger/         # slog configuration
│   │   ├── migrations/         # Database migrations (SQLite)
│   │   ├── docs/               # Swagger generated docs
│   │   └── .air.toml           # Air hot reload config
│   │
│   └── web/                    # Frontend application (React)
│       ├── src/
│       │   ├── routes/         # TanStack Router route files
│       │   ├── components/     # React components (feature-organized)
│       │   │   ├── search/     # Search feature components
│       │   │   ├── library/    # Library feature components
│       │   │   ├── downloads/  # Downloads feature components
│       │   │   └── ui/         # Shared UI components
│       │   ├── hooks/          # Custom React hooks
│       │   ├── services/       # API client services
│       │   ├── stores/         # Global state (Zustand if needed)
│       │   ├── utils/          # Utility functions
│       │   └── test/           # Test utilities and setup
│       ├── tailwind.config.js
│       ├── vitest.config.ts
│       └── vite.config.ts
│
├── libs/
│   └── shared-types/           # Shared TypeScript types (sync with Go structs)
│       └── src/lib/shared-types.ts
│
├── archive/                    # DEPRECATED: Old root backend (after Phase 1)
│   ├── cmd/
│   └── internal/
│
├── docs/                       # Project documentation
├── docker-compose.yml
└── nx.json
```

**Critical Rules:**

1. **Single Backend Location:**
   - ✅ `/apps/api` is the ONLY backend
   - ❌ NO code in `/cmd` or root `/internal` (archive after migration)

2. **Feature-First Frontend Organization:**
   - ✅ Components organized by feature (e.g., `components/search/`)
   - ✅ Shared UI components in `components/ui/`
   - ❌ NOT by type (e.g., `components/buttons/`, `components/cards/`)

3. **Backend Layered Architecture:**
   - ✅ Handlers → Services → Repository → Database
   - ✅ Services contain business logic, repositories handle data access
   - ❌ Handlers MUST NOT directly access repositories (violates layering)

4. **Test Co-location:**
   - ✅ Backend: `{filename}_test.go` next to source file
   - ✅ Frontend: `{ComponentName}.spec.tsx` next to component
   - ❌ NO separate `tests/` directory

**Current Codebase Compliance:**
- ⚠️ **Dual backend exists:** Root `/cmd` + `/internal` AND `/apps/api`
- 🔴 **Migration required:** Consolidate into `/apps/api` during Phase 1
- ✅ **Frontend structure:** Follows Nx conventions, needs feature directories added

---

## 2.2 Backend File Structure Patterns

**Handler Files:**
- ✅ **Pattern:** `/apps/api/internal/handlers/{resource}_handler.go`
- ✅ **Examples:** `movie_handler.go`, `auth_handler.go`, `download_handler.go`
- ✅ **Struct naming:** `MovieHandler`, `AuthHandler`
- ❌ **Anti-pattern:** `movies.go`, `handler_movie.go`

**Service Files:**
- ✅ **Pattern:** `/apps/api/internal/services/{domain}_service.go`
- ✅ **Examples:** `metadata_service.go`, `auth_service.go`, `parser_service.go`
- ✅ **Struct naming:** `MetadataService`, `AuthService`
- ❌ **Anti-pattern:** `metadata.go`, `service_metadata.go`

**Repository Files:**
- ✅ **Pattern:** `/apps/api/internal/repository/{resource}_repository.go`
- ✅ **Examples:** `movie_repository.go`, `user_repository.go`, `cache_repository.go`
- ✅ **Interface naming:** `MovieRepository`, `UserRepository`
- ❌ **Anti-pattern:** `movies.go`, `movie_repo.go`

**Model Files:**
- ✅ **Pattern:** `/apps/api/internal/models/{resource}.go`
- ✅ **Examples:** `movie.go`, `user.go`, `settings.go`
- ✅ **Struct naming:** `Movie`, `User`, `Settings`
- ❌ **Anti-pattern:** `movie_model.go`, `models.go` (single file for all models)

**Middleware Files:**
- ✅ **Pattern:** `/apps/api/internal/middleware/{name}.go`
- ✅ **Examples:** `auth.go`, `cors.go`, `recovery.go`, `request_id.go`
- ❌ **Anti-pattern:** `auth_middleware.go`, `middleware.go`

**Test Files:**
- ✅ **Pattern:** `{filename}_test.go` co-located with source
- ✅ **Examples:** `movie_handler_test.go`, `cache_manager_test.go`
- ❌ **Anti-pattern:** `tests/movie_handler.go`, `movie_test.go` (omits layer)

---

## 2.3 Frontend File Structure Patterns

**Component Files:**
- ✅ **Pattern:** `/apps/web/src/components/{feature}/{ComponentName}.tsx`
- ✅ **Examples:**
  - `components/search/SearchBar.tsx`
  - `components/library/MovieCard.tsx`
  - `components/ui/Button.tsx`
- ❌ **Anti-pattern:** `components/SearchBar.tsx` (no feature grouping)

**Hook Files:**
- ✅ **Pattern:** `/apps/web/src/hooks/use{Name}.ts`
- ✅ **Examples:** `hooks/useSearch.ts`, `hooks/useAuth.ts`
- ❌ **Anti-pattern:** `hooks/search.ts`, `hooks/useSearchHook.ts`

**Service Files:**
- ✅ **Pattern:** `/apps/web/src/services/{resource}Service.ts`
- ✅ **Examples:** `services/movieService.ts`, `services/authService.ts`
- ❌ **Anti-pattern:** `services/movies.ts`, `api/movieApi.ts`

**Route Files:**
- ✅ **Pattern:** `/apps/web/src/routes/{path}.tsx`
- ✅ **Examples:** `routes/search.tsx`, `routes/library.tsx`, `routes/__root.tsx`
- ❌ **Anti-pattern:** `pages/Search.tsx`, `routes/search/index.tsx`

**Test Files:**
- ✅ **Pattern:** `{ComponentName}.spec.tsx` co-located with component
- ✅ **Examples:** `SearchBar.spec.tsx`, `MovieCard.spec.tsx`
- ❌ **Anti-pattern:** `__tests__/SearchBar.test.tsx`, `SearchBar.test.tsx`

---

## 2.4 Plugin Package Structure Pattern

**MANDATORY Structure for `/apps/api/internal/plugins/`:**

```
plugins/
├── manager.go          # Plugin registration, health checks, lifecycle
├── types.go            # Plugin, MediaServerPlugin, DownloaderPlugin, DVRPlugin interfaces
├── plex/
│   ├── plex.go         # Plex MediaServerPlugin implementation
│   └── plex_test.go
├── jellyfin/
│   ├── jellyfin.go     # Jellyfin MediaServerPlugin implementation
│   └── jellyfin_test.go
├── sonarr/
│   ├── sonarr.go       # Sonarr DVRPlugin implementation
│   └── sonarr_test.go
├── radarr/
│   ├── radarr.go       # Radarr DVRPlugin implementation
│   └── radarr_test.go
└── prowlarr/
    ├── prowlarr.go     # Prowlarr indexer integration
    └── prowlarr_test.go
```

**Critical Rules:**
- ✅ All plugins implement the base `Plugin` interface (`Name()`, `TestConnection()`)
- ✅ Each plugin in its own sub-package
- ✅ Plugin registration happens in `main.go` at startup
- ❌ NO dynamic plugin loading (Go plugins or shared libraries)

---

## 2.5 Subtitle Engine Package Pattern

**MANDATORY Structure for `/apps/api/internal/subtitle/`:**

```
subtitle/
├── engine.go           # Pipeline orchestrator (search → score → download → convert → place)
├── engine_test.go
├── scorer.go           # Multi-factor subtitle scoring
├── scorer_test.go
├── converter.go        # OpenCC 簡繁轉換 integration
├── converter_test.go
├── detector.go         # Content-based language detection
├── detector_test.go
└── providers/          # Subtitle source implementations
    ├── provider.go     # SubtitleProvider interface
    ├── assrt.go        # Assrt API provider
    ├── assrt_test.go
    ├── zimuku.go       # Zimuku scraper provider
    ├── zimuku_test.go
    ├── opensub.go      # OpenSubtitles API provider
    └── opensub_test.go
```

**Critical Rules:**
- ✅ Each provider implements `SubtitleProvider` interface
- ✅ Providers are independent and can be enabled/disabled
- ✅ Scoring weights are configurable (not hard-coded)
- ❌ Provider files MUST NOT import from other provider packages

---

## 2.6 SSE Handler Package Pattern

**MANDATORY Structure for `/apps/api/internal/sse/`:**

```
sse/
├── hub.go              # Central event broadcaster (single goroutine)
├── hub_test.go
├── handler.go          # HTTP handler for GET /api/v1/events
└── handler_test.go
```

**Critical Rules:**
- ✅ Single hub instance, initialized at application startup
- ✅ Hub runs as a single goroutine, fan-out to client channels
- ✅ Handler uses `http.Flusher` interface for streaming
- ❌ NO WebSocket fallback (SSE is sufficient for server→client push)

---

## 2.7 Configuration File Organization

**Backend Configuration:**
- ✅ **Air config:** `/apps/api/.air.toml`
- ✅ **Environment template:** `/apps/api/.env.example`
- ✅ **Actual env:** `/apps/api/.env` (gitignored)
- ❌ **Anti-pattern:** Root-level `.air.toml`, `config/air.toml`

**Frontend Configuration:**
- ✅ **Vite config:** `/apps/web/vite.config.ts`
- ✅ **Tailwind config:** `/apps/web/tailwind.config.js`
- ✅ **Vitest config:** `/apps/web/vitest.config.ts`
- ✅ **TypeScript config:** `/apps/web/tsconfig.json`
- ❌ **Anti-pattern:** Root-level configs (conflicts with monorepo)

**Monorepo Configuration:**
- ✅ **Nx config:** `/nx.json` (root level)
- ✅ **Docker compose:** `/docker-compose.yml` (root level)
- ✅ **Root tsconfig:** `/tsconfig.base.json` (Nx convention)

---
