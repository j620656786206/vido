# Project Structure & Boundaries

## Overview

This section defines the **target project structure** after Phase 1-5 consolidation is complete. It maps all 94 functional requirements to specific directories and files, establishing clear architectural boundaries for AI agent implementation.

**Context:**
- This represents the **unified architecture** with single backend at `/apps/api`
- Root backend (`/cmd`, `/internal`) will be archived to `/archive/`
- Structure optimized for Nx monorepo with Go backend + React frontend

---

## Complete Project Directory Structure

**Target State (Post-Consolidation):**

```
vido/
├── README.md                           # Project overview and setup
├── package.json                        # Root package.json (Nx workspace)
├── nx.json                             # Nx configuration
├── tsconfig.base.json                  # Shared TypeScript config
├── .gitignore                          # Git ignore rules
├── .env.example                        # Environment variable template
├── docker-compose.yml                  # Docker orchestration
├── project-context.md                  # ⭐ AI agent quick reference
│
├── .github/                            # GitHub configuration
│   └── workflows/
│       ├── ci.yml                      # CI pipeline (tests, linting)
│       └── deploy.yml                  # Deployment automation
│
├── docs/                               # Project documentation
│   ├── README.md                       # API documentation overview
│   ├── AIR_SETUP.md                    # Air hot reload setup
│   ├── SWAGGO_SETUP.md                 # Swagger documentation setup
│   └── DEVELOPMENT.md                  # Development workflow guide
│
├── _bmad/                              # BMAD workflow system
│   ├── bmm/                            # BMAD workflows and agents
│   └── core/                           # Core BMAD utilities
│
├── ux-design.pen                       # UX design source (Pencil app)
├── _bmad-output/                       # BMAD generated artifacts
│   ├── screenshots/                    # Exported design screenshots
│   │   ├── flow-a-browse-desktop/      # Desktop browse flow
│   │   ├── flow-b-hover-detail-desktop/ # Desktop hover + detail flow
│   │   ├── flow-c-search-filter-settings-desktop/ # Desktop search/filter
│   │   ├── flow-d-browse-mobile/       # Mobile browse flow
│   │   ├── flow-e-interaction-mobile/  # Mobile interaction flow
│   │   └── flow-f-batch-settings-mobile/ # Mobile batch/settings flow
│   └── planning-artifacts/
│       ├── architecture.md             # ⭐ Complete architecture doc
│       ├── prd.md                      # Product requirements
│       ├── prd-validation-report.md   # PRD validation
│       └── ux-design-specification.md # UX design spec
│
├── archive/                            # ⚠️ DEPRECATED: Old root backend
│   ├── cmd/                            # Archived root entry point
│   └── internal/                       # Archived root backend code
│
├── apps/                               # Nx applications
│   │
│   ├── api/                            # ⭐ UNIFIED BACKEND (Go)
│   │   ├── main.go                     # Application entry point
│   │   ├── go.mod                      # Go module definition
│   │   ├── go.sum                      # Go dependencies lock
│   │   ├── .air.toml                   # Air hot reload config
│   │   ├── .env.example                # Backend env template
│   │   │
│   │   ├── docs/                       # Swagger generated docs
│   │   │   ├── docs.go                 # Swaggo generated code
│   │   │   ├── swagger.json            # OpenAPI spec (JSON)
│   │   │   └── swagger.yaml            # OpenAPI spec (YAML)
│   │   │
│   │   ├── migrations/                 # Database migrations (SQLite)
│   │   │   ├── 001_create_movies_table.sql          # ✅ Exists
│   │   │   ├── 002_create_series_table.sql          # ✅ Exists
│   │   │   ├── 003_create_settings_table.sql        # ✅ Exists
│   │   │   ├── 004_create_users_table.sql           # Phase 2.1
│   │   │   ├── 005_create_cache_entries_table.sql   # Phase 2.2
│   │   │   ├── 006_create_filename_mappings_table.sql # Phase 4.1
│   │   │   ├── 007_create_download_history_table.sql  # Phase 4
│   │   │   └── 008_add_fts5_search_index.sql        # Phase 4
│   │   │
│   │   └── internal/                   # Private application code
│   │       │
│   │       ├── handlers/               # HTTP request handlers (Gin)
│   │       │   ├── movie_handler.go            # Movie CRUD endpoints
│   │       │   ├── movie_handler_test.go       # Movie handler tests
│   │       │   ├── series_handler.go           # Series CRUD endpoints
│   │       │   ├── series_handler_test.go
│   │       │   ├── search_handler.go           # Search endpoints (FR1-10)
│   │       │   ├── search_handler_test.go
│   │       │   ├── auth_handler.go             # Login/logout endpoints
│   │       │   ├── auth_handler_test.go
│   │       │   ├── settings_handler.go         # Settings management
│   │       │   ├── settings_handler_test.go
│   │       │   ├── download_handler.go         # qBittorrent integration
│   │       │   ├── download_handler_test.go
│   │       │   ├── parser_handler.go           # Filename parsing endpoints
│   │       │   ├── parser_handler_test.go
│   │       │   └── health_handler.go           # Health check endpoint
│   │       │
│   │       ├── services/               # Business logic layer
│   │       │   ├── metadata_orchestrator.go    # Multi-source metadata (FR15-20)
│   │       │   ├── metadata_orchestrator_test.go
│   │       │   ├── search_service.go           # Search business logic
│   │       │   ├── search_service_test.go
│   │       │   ├── auth_service.go             # JWT authentication
│   │       │   ├── auth_service_test.go
│   │       │   ├── parser_service.go           # Filename parsing orchestration
│   │       │   ├── parser_service_test.go
│   │       │   ├── download_service.go         # Download monitoring
│   │       │   ├── download_service_test.go
│   │       │   └── backup_service.go           # Backup/restore logic
│   │       │
│   │       ├── repository/             # Data access layer (Repository pattern)
│   │       │   ├── movie_repository.go         # ✅ Exists
│   │       │   ├── movie_repository_test.go
│   │       │   ├── series_repository.go        # ✅ Exists
│   │       │   ├── series_repository_test.go
│   │       │   ├── user_repository.go          # User CRUD
│   │       │   ├── user_repository_test.go
│   │       │   ├── settings_repository.go      # ✅ Exists
│   │       │   ├── settings_repository_test.go
│   │       │   ├── cache_repository.go         # Cache persistence
│   │       │   ├── cache_repository_test.go
│   │       │   ├── filename_mapping_repository.go # Learning system
│   │       │   ├── filename_mapping_repository_test.go
│   │       │   └── repository.go               # Repository interface definitions
│   │       │
│   │       ├── models/                 # Domain models (Go structs)
│   │       │   ├── movie.go                    # ✅ Exists
│   │       │   ├── series.go                   # ✅ Exists
│   │       │   ├── user.go                     # User model
│   │       │   ├── settings.go                 # ✅ Exists
│   │       │   ├── cache_entry.go              # Cache entry model
│   │       │   ├── filename_mapping.go         # Filename mapping model
│   │       │   ├── download.go                 # Download history model
│   │       │   └── search_result.go            # Search result model
│   │       │
│   │       ├── middleware/             # HTTP middleware
│   │       │   ├── auth.go                     # JWT authentication (Phase 2.1)
│   │       │   ├── auth_test.go
│   │       │   ├── cors.go                     # CORS configuration
│   │       │   ├── recovery.go                 # Panic recovery
│   │       │   ├── request_id.go               # Request ID tracking
│   │       │   ├── logging.go                  # Request/response logging
│   │       │   └── rate_limit.go               # Rate limiting (optional)
│   │       │
│   │       ├── tmdb/                   # TMDb API client
│   │       │   ├── client.go                   # TMDb HTTP client
│   │       │   ├── client_test.go
│   │       │   ├── types.go                    # TMDb response types
│   │       │   ├── movies.go                   # Movie endpoints
│   │       │   ├── series.go                   # TV series endpoints
│   │       │   └── search.go                   # Search endpoints
│   │       │
│   │       ├── douban/                 # Douban scraper (Phase 4.2)
│   │       │   ├── scraper.go                  # Douban web scraper
│   │       │   ├── scraper_test.go
│   │       │   └── parser.go                   # HTML parsing
│   │       │
│   │       ├── wikipedia/              # Wikipedia API client (Phase 4.2)
│   │       │   ├── client.go                   # MediaWiki API client
│   │       │   ├── client_test.go
│   │       │   └── search.go                   # Wikipedia search
│   │       │
│   │       ├── qbittorrent/            # qBittorrent Web API client
│   │       │   ├── client.go                   # qBittorrent HTTP client
│   │       │   ├── client_test.go
│   │       │   ├── torrents.go                 # Torrent operations
│   │       │   └── auth.go                     # qBittorrent authentication
│   │       │
│   │       ├── ai/                     # AI provider abstraction (Phase 4.1)
│   │       │   ├── provider.go                 # Provider interface
│   │       │   ├── gemini.go                   # Google Gemini client
│   │       │   ├── gemini_test.go
│   │       │   ├── claude.go                   # Anthropic Claude client
│   │       │   ├── claude_test.go
│   │       │   └── types.go                    # AI response types
│   │       │
│   │       ├── parser/                 # Filename parser (Phase 4.1)
│   │       │   ├── regex_parser.go             # Regex-based parser
│   │       │   ├── regex_parser_test.go
│   │       │   ├── patterns.go                 # Filename patterns
│   │       │   ├── ai_parser.go                # AI-powered parser
│   │       │   ├── ai_parser_test.go
│   │       │   └── parser.go                   # Parser interface
│   │       │
│   │       ├── plugins/                # Plugin interfaces and manager
│   │       │   ├── manager.go                  # Plugin registration, health checks
│   │       │   ├── types.go                    # MediaServerPlugin, DownloaderPlugin, DVRPlugin interfaces
│   │       │   ├── plex/                       # Plex MediaServerPlugin implementation
│   │       │   ├── jellyfin/                   # Jellyfin MediaServerPlugin implementation
│   │       │   ├── sonarr/                     # Sonarr DVRPlugin implementation
│   │       │   ├── radarr/                     # Radarr DVRPlugin implementation
│   │       │   └── prowlarr/                   # Prowlarr indexer integration
│   │       │
│   │       ├── sse/                    # Server-Sent Events hub
│   │       │   ├── hub.go                      # Central event broadcaster
│   │       │   └── handler.go                  # HTTP handler for /api/v1/events
│   │       │
│   │       ├── subtitle/               # Subtitle engine pipeline
│   │       │   ├── engine.go                   # Pipeline orchestrator
│   │       │   ├── scorer.go                   # Multi-factor subtitle scoring
│   │       │   ├── converter.go                # OpenCC 簡繁轉換
│   │       │   ├── providers/                  # Subtitle source implementations
│   │       │   │   ├── assrt.go
│   │       │   │   ├── zimuku.go
│   │       │   │   └── opensub.go
│   │       │   └── detector.go                 # Content-based language detection
│   │       │
│   │       ├── scanner/                # Media library scanner
│   │       │   ├── scanner.go                  # Recursive file scanner
│   │       │   ├── watcher.go                  # File system watcher for scheduled scans
│   │       │   └── matcher.go                  # TMDB matching orchestrator
│   │       │
│   │       ├── cache/                  # Cache management (Phase 2.2)
│   │       │   ├── manager.go                  # Cache manager (tiered)
│   │       │   ├── manager_test.go
│   │       │   ├── memory.go                   # In-memory cache (bigcache)
│   │       │   ├── memory_test.go
│   │       │   ├── sqlite.go                   # SQLite persistent cache
│   │       │   └── sqlite_test.go
│   │       │
│   │       ├── tasks/                  # Background task queue (Phase 2.3)
│   │       │   ├── queue.go                    # Worker pool implementation
│   │       │   ├── queue_test.go
│   │       │   ├── ai_parsing_task.go          # AI parsing background task
│   │       │   ├── metadata_refresh_task.go    # Metadata refresh task
│   │       │   ├── backup_task.go              # Backup task
│   │       │   └── task.go                     # Task interface
│   │       │
│   │       ├── errors/                 # Unified error handling (Phase 2.4)
│   │       │   ├── app_error.go                # AppError type
│   │       │   ├── codes.go                    # Error code constants
│   │       │   ├── tmdb_errors.go              # TMDb error constructors
│   │       │   ├── ai_errors.go                # AI provider errors
│   │       │   ├── db_errors.go                # Database errors
│   │       │   └── auth_errors.go              # Authentication errors
│   │       │
│   │       ├── logger/                 # Logging configuration (Phase 1.1)
│   │       │   ├── slog.go                     # slog setup
│   │       │   └── sanitize.go                 # Sensitive data filtering
│   │       │
│   │       ├── database/               # Database connection & setup
│   │       │   ├── database.go                 # SQLite connection pool
│   │       │   ├── migrate.go                  # Migration runner
│   │       │   └── seed.go                     # Seed data (optional)
│   │       │
│   │       └── utils/                  # Shared utilities
│   │           ├── response.go                 # API response helpers
│   │           ├── validation.go               # Input validation helpers
│   │           └── crypto.go                   # Encryption utilities
│   │
│   └── web/                            # ⭐ FRONTEND (React + TypeScript)
│       ├── index.html                  # Vite entry HTML
│       ├── package.json                # Frontend dependencies
│       ├── vite.config.ts              # Vite configuration
│       ├── tailwind.config.js          # Tailwind CSS config (Phase 3.1)
│       ├── postcss.config.js           # PostCSS config
│       ├── vitest.config.ts            # Vitest test config (Phase 3.2)
│       ├── tsconfig.json               # TypeScript config
│       ├── .env.example                # Frontend env template
│       │
│       └── src/
│           ├── main.tsx                # ✅ Application entry point
│           ├── router.tsx              # ✅ TanStack Router config
│           ├── App.tsx                 # ✅ Root component
│           │
│           ├── routes/                 # Route components (TanStack Router)
│           │   ├── __root.tsx          # ✅ Root layout
│           │   ├── index.tsx           # ✅ Landing page
│           │   ├── search.tsx          # Search page (Phase 4.3)
│           │   ├── library.tsx         # Media library page (Phase 4.4)
│           │   ├── downloads.tsx       # Downloads monitor page
│           │   ├── settings.tsx        # Settings page
│           │   └── login.tsx           # Login page
│           │
│           ├── components/             # React components (feature-organized)
│           │   │
│           │   ├── search/             # Search feature (FR1-10)
│           │   │   ├── SearchBar.tsx
│           │   │   ├── SearchBar.spec.tsx
│           │   │   ├── FilterPanel.tsx
│           │   │   ├── FilterPanel.spec.tsx
│           │   │   ├── ResultsGrid.tsx
│           │   │   ├── ResultsGrid.spec.tsx
│           │   │   ├── ResultsList.tsx
│           │   │   ├── ResultsList.spec.tsx
│           │   │   └── SortControls.tsx
│           │   │
│           │   ├── library/            # Media library (FR38-46)
│           │   │   ├── MediaGrid.tsx
│           │   │   ├── MediaGrid.spec.tsx
│           │   │   ├── MediaList.tsx
│           │   │   ├── MediaList.spec.tsx
│           │   │   ├── MovieCard.tsx
│           │   │   ├── MovieCard.spec.tsx
│           │   │   ├── SeriesCard.tsx
│           │   │   ├── SeriesCard.spec.tsx
│           │   │   ├── FilterControls.tsx
│           │   │   └── VirtualScrollContainer.tsx # Virtual scrolling
│           │   │
│           │   ├── downloads/          # Download monitor (FR27-37)
│           │   │   ├── DownloadList.tsx
│           │   │   ├── DownloadList.spec.tsx
│           │   │   ├── DownloadItem.tsx
│           │   │   ├── DownloadItem.spec.tsx
│           │   │   ├── ProgressBar.tsx
│           │   │   ├── StatusBadge.tsx
│           │   │   └── QBitConnectionStatus.tsx
│           │   │
│           │   ├── parser/             # Filename parsing UI (FR11-26)
│           │   │   ├── ParserForm.tsx
│           │   │   ├── ParserForm.spec.tsx
│           │   │   ├── ParseResults.tsx
│           │   │   ├── ConfidenceScore.tsx
│           │   │   ├── ManualVerification.tsx
│           │   │   └── LearningIndicator.tsx
│           │   │
│           │   ├── settings/           # Settings pages (FR47-66)
│           │   │   ├── SettingsLayout.tsx
│           │   │   ├── GeneralSettings.tsx
│           │   │   ├── APIKeysSettings.tsx
│           │   │   ├── CacheSettings.tsx
│           │   │   ├── BackupSettings.tsx
│           │   │   └── QBitSettings.tsx
│           │   │
│           │   # auth/ — REMOVED in v4 (single-user, no auth)
│           │   │
│           │   └── ui/                 # Shared UI components
│           │       ├── Button.tsx
│           │       ├── Button.spec.tsx
│           │       ├── Input.tsx
│           │       ├── Modal.tsx
│           │       ├── Toast.tsx
│           │       ├── LoadingSpinner.tsx
│           │       ├── ErrorMessage.tsx
│           │       ├── Skeleton.tsx
│           │       └── Badge.tsx
│           │
│           ├── hooks/                  # Custom React hooks
│           │   ├── useSearch.ts        # Search query hook
│           │   ├── useSearch.spec.ts
│           │   ├── useMovieQuery.ts    # Movie data hook
│           │   ├── useSeriesQuery.ts   # Series data hook
│           │   ├── useAuth.ts          # Authentication hook
│           │   ├── useAuth.spec.ts
│           │   ├── useDownloadStatus.ts # Download monitoring hook
│           │   └── useParser.ts        # Filename parsing hook
│           │
│           ├── services/               # API client services
│           │   ├── movieService.ts     # Movie API client
│           │   ├── seriesService.ts    # Series API client
│           │   ├── searchService.ts    # Search API client
│           │   ├── authService.ts      # Auth API client
│           │   ├── downloadService.ts  # Download API client
│           │   ├── parserService.ts    # Parser API client
│           │   └── apiClient.ts        # Base HTTP client (fetch wrapper)
│           │
│           ├── stores/                 # Global state (Zustand - UI state only)
│           │   ├── authStore.ts        # Auth state (isAuthenticated, user)
│           │   ├── uiStore.ts          # UI state (theme, sidebar, etc.)
│           │   └── filterStore.ts      # Filter state (if needed)
│           │
│           ├── utils/                  # Utility functions
│           │   ├── formatDate.ts       # Date formatting
│           │   ├── formatDate.spec.ts
│           │   ├── sanitizeFilename.ts # Filename sanitization
│           │   ├── parseMovieTitle.ts  # Movie title parsing
│           │   └── apiError.ts         # API error handling
│           │
│           ├── types/                  # TypeScript type definitions
│           │   └── index.ts            # Re-export from shared-types
│           │
│           ├── styles/                 # Global styles
│           │   └── globals.css         # Tailwind directives + custom CSS
│           │
│           └── test/                   # Test utilities
│               ├── setup.ts            # Vitest setup
│               ├── mockData.ts         # Test data factories
│               └── testUtils.tsx       # Testing library wrappers
│
├── libs/                               # Shared libraries
│   └── shared-types/                   # TypeScript types (sync with Go)
│       ├── package.json
│       ├── tsconfig.json
│       └── src/
│           └── lib/
│               └── shared-types.ts     # ✅ Movie, Series, ApiResponse, etc.
│
└── data/                               # Runtime data (gitignored)
    ├── vido.db                         # SQLite database
    ├── vido.db-shm                     # SQLite shared memory
    ├── vido.db-wal                     # SQLite write-ahead log
    ├── cache/                          # File cache
    │   └── images/                     # Downloaded posters/backdrops
    └── backups/                        # Database backups
```

---

## Architectural Boundaries

### API Boundaries

**External API Endpoints (Public Interface):**

```
Authentication:
  POST   /api/v1/auth/login          # Login with password/PIN
  POST   /api/v1/auth/logout         # Logout
  GET    /api/v1/auth/me             # Get current user

Movies:
  GET    /api/v1/movies              # List movies (paginated)
  GET    /api/v1/movies/{id}         # Get movie by ID
  POST   /api/v1/movies              # Create movie
  PUT    /api/v1/movies/{id}         # Update movie
  DELETE /api/v1/movies/{id}         # Delete movie

Series:
  GET    /api/v1/series              # List series (paginated)
  GET    /api/v1/series/{id}         # Get series by ID
  POST   /api/v1/series              # Create series
  PUT    /api/v1/series/{id}         # Update series
  DELETE /api/v1/series/{id}         # Delete series

Search:
  GET    /api/v1/search              # Search movies/series
  GET    /api/v1/search/suggestions  # Search suggestions (autocomplete)

Parser:
  POST   /api/v1/parser/filename     # Parse single filename
  POST   /api/v1/parser/batch        # Parse multiple filenames
  GET    /api/v1/parser/mappings     # Get learned filename mappings

Downloads:
  GET    /api/v1/downloads           # List downloads
  GET    /api/v1/downloads/{id}      # Get download status
  POST   /api/v1/downloads/{id}/pause   # Pause download
  POST   /api/v1/downloads/{id}/resume  # Resume download

Settings:
  GET    /api/v1/settings            # Get all settings
  GET    /api/v1/settings/{key}      # Get setting by key
  PUT    /api/v1/settings/{key}      # Update setting

System:
  GET    /api/v1/health              # Health check
  GET    /api/v1/docs                # Swagger UI
  GET    /api/v1/swagger.json        # OpenAPI spec
```

**Internal Service Boundaries:**

```
Handler Layer → Service Layer:
  - Handlers MUST call services, NEVER repositories directly
  - Request validation happens in handlers
  - Response formatting happens in handlers

Service Layer → Repository Layer:
  - Services contain business logic
  - Services orchestrate multiple repositories
  - Services handle caching logic

Repository Layer → Database:
  - Repositories perform CRUD operations
  - Repositories abstract database implementation
  - Repositories return domain models
```

---

### Component Boundaries

**Frontend Component Communication:**

```
Server State (TanStack Query):
  - All API data fetched via TanStack Query
  - Query keys follow hierarchical structure
  - Mutations trigger cache invalidation

Global Client State (Zustand):
  - Authentication state (isAuthenticated, user)
  - UI preferences (theme, sidebar state)
  - Filter state (if not in URL)

Local Component State (useState):
  - Form inputs
  - Toggle states
  - Modal open/close

Props Flow:
  - Parent → Child (unidirectional data flow)
  - Event handlers passed as callbacks
  - NO prop drilling (use context if needed)
```

**Component Hierarchy:**

```
App (Root)
├── Router (TanStack Router)
│   ├── __root Layout
│   │   ├── Header (navigation)
│   │   ├── Sidebar (optional)
│   │   └── Main Content Area
│   │       ├── Search Page
│   │       │   ├── SearchBar
│   │       │   ├── FilterPanel
│   │       │   └── ResultsGrid/ResultsList
│   │       ├── Library Page
│   │       │   ├── FilterControls
│   │       │   └── MediaGrid/MediaList
│   │       │       └── MovieCard/SeriesCard
│   │       ├── Downloads Page
│   │       │   └── DownloadList
│   │       │       └── DownloadItem
│   │       └── Settings Page
│   │           └── SettingsLayout
│   │               └── (Various Settings)
│   └── Login Page
│       └── LoginForm
└── Toast Container (global)
```

---

### Service Boundaries

**Backend Service Responsibilities:**

```
MetadataOrchestrator:
  - Coordinates TMDb, Douban, Wikipedia, AI sources
  - Implements fallback chain
  - Manages circuit breakers
  - Caches results

AuthService:
  - Validates credentials (bcrypt)
  - Generates JWT tokens
  - Verifies JWT tokens
  - Manages user sessions

ParserService:
  - Orchestrates regex parser → AI parser fallback
  - Manages learning system (filename mappings)
  - Calculates confidence scores
  - Handles manual verification

DownloadService:
  - Polls qBittorrent for status
  - Stores download history
  - Triggers notifications
  - Handles connection failures

BackupService:
  - Creates SQLite backups (atomic)
  - Verifies backup checksums
  - Manages backup retention
  - Schedules automatic backups
```

**Service Integration Patterns:**

```
Handler → Service (always):
  handler.GetMovie() → service.GetMovieByID() → repository.FindByID()

Service → Multiple Repositories:
  service.GetMovieWithMetadata() {
    movie ← movieRepo.FindByID()
    metadata ← metadataOrchestrator.Fetch()
    return merge(movie, metadata)
  }

Service → External API (with caching):
  service.FetchFromTMDb() {
    cached ← cacheManager.Get(key)
    if cached != nil { return cached }

    result ← tmdbClient.GetMovie()
    cacheManager.Set(key, result, 24h)
    return result
  }
```

---

### Data Boundaries

**Database Schema Boundaries:**

```
Core Entities:
  movies          # Movie metadata and state
  series          # TV series metadata and state
  users           # User accounts (Phase 2.1)

Supporting Entities:
  settings        # System configuration
  cache_entries   # Persistent cache (Phase 2.2)
  filename_mappings  # Learning system (Phase 4.1)
  download_history   # qBittorrent downloads

Relationships:
  movies.tmdb_id → TMDb API
  series.tmdb_id → TMDb API
  filename_mappings.parsed_title → movies.title (fuzzy match)
  download_history.movie_id → movies.id (optional FK)
```

**Data Access Patterns:**

```
Repository Pattern:
  interface MovieRepository {
    FindByID(ctx, id) → *Movie
    FindAll(ctx, filters) → []Movie
    Create(ctx, movie) → error
    Update(ctx, movie) → error
    Delete(ctx, id) → error
    Search(ctx, query) → []Movie  # Uses FTS5
  }

SQLite Implementation:
  type SQLiteMovieRepository struct {
    db *sql.DB
  }

PostgreSQL Implementation (future):
  type PostgresMovieRepository struct {
    db *sql.DB
  }
```

**Caching Boundaries:**

```
Tier 1 (Memory):
  - TMDb API responses (24h TTL)
  - Frequently accessed movies (LRU eviction)
  - qBittorrent status (5s TTL)

Tier 2 (SQLite cache_entries):
  - AI parsing results (30d TTL)
  - Douban scraping results (7d TTL)
  - Wikipedia results (7d TTL)

Tier 3 (File System):
  - Downloaded images (permanent)
  - Movie posters
  - Backdrop images
```

---

## Requirements to Structure Mapping

### Feature Area Mapping

**FR1-FR10: Media Search & Discovery**

```
Frontend:
  /apps/web/src/routes/search.tsx
  /apps/web/src/components/search/
    - SearchBar.tsx         (FR1: Search by title/keyword)
    - FilterPanel.tsx       (FR4: Filter by genre/year/rating)
    - SortControls.tsx      (FR5: Sort options)
    - ResultsGrid.tsx       (FR3: Grid view)
    - ResultsList.tsx       (FR3: List view)

Backend:
  /apps/api/internal/handlers/search_handler.go
  /apps/api/internal/services/search_service.go
  /apps/api/internal/repository/movie_repository.go
    - Search(query) method using FTS5

Database:
  Migration 008: Add FTS5 full-text search index
  Index on: movies.title, movies.original_title, movies.overview
```

**FR11-FR26: Filename Parsing & Metadata Retrieval**

```
Frontend:
  /apps/web/src/components/parser/
    - ParserForm.tsx           (FR13: Manual entry)
    - ParseResults.tsx         (FR21: Confidence display)
    - ManualVerification.tsx   (FR24: Manual verification)

Backend:
  /apps/api/internal/parser/
    - regex_parser.go          (FR11: Standard regex parsing)
    - ai_parser.go             (FR12: AI-powered parsing)
  /apps/api/internal/services/metadata_orchestrator.go
    - Fallback chain: TMDb → Douban → Wikipedia → AI (FR15-20)
  /apps/api/internal/tmdb/client.go
  /apps/api/internal/douban/scraper.go
  /apps/api/internal/wikipedia/client.go
  /apps/api/internal/ai/
    - gemini.go                (FR12: Gemini provider)
    - claude.go                (FR12: Claude provider)

Database:
  Migration 006: filename_mappings table (FR25-26: Learning system)
```

**FR27-FR37: Download Integration & Monitoring**

```
Frontend:
  /apps/web/src/routes/downloads.tsx
  /apps/web/src/components/downloads/
    - DownloadList.tsx         (FR27: Real-time status)
    - DownloadItem.tsx         (FR28: Individual download)
    - ProgressBar.tsx          (FR31: Progress display)
    - StatusBadge.tsx          (FR30: Status indicators)
    - QBitConnectionStatus.tsx (FR32: Connection health)

Backend:
  /apps/api/internal/qbittorrent/client.go
  /apps/api/internal/services/download_service.go
  /apps/api/internal/handlers/download_handler.go

Database:
  Migration 007: download_history table
  Polling: Every 5 seconds for active downloads (NFR-P8)
```

**FR38-FR46: Media Library Management**

```
Frontend:
  /apps/web/src/routes/library.tsx
  /apps/web/src/components/library/
    - MediaGrid.tsx            (FR39: Grid view)
    - MediaList.tsx            (FR39: List view)
    - MovieCard.tsx            (FR38: Display metadata)
    - SeriesCard.tsx
    - FilterControls.tsx       (FR42: Filters)
    - VirtualScrollContainer.tsx (NFR-P10: >1000 items)

Backend:
  /apps/api/internal/handlers/movie_handler.go
  /apps/api/internal/handlers/series_handler.go
  /apps/api/internal/repository/movie_repository.go
  /apps/api/internal/repository/series_repository.go

Database:
  Tables: movies, series (already exist)
```

**FR47-FR66: System Configuration & Management**

```
Frontend:
  /apps/web/src/routes/settings.tsx
  /apps/web/src/components/settings/
    - GeneralSettings.tsx      (FR49: Basic config)
    - APIKeysSettings.tsx      (FR50: TMDb, AI keys)
    - CacheSettings.tsx        (FR51: Cache management)
    - BackupSettings.tsx       (FR58: Backup/restore)
    - QBitSettings.tsx         (FR52: qBittorrent config)

Backend:
  /apps/api/internal/handlers/settings_handler.go
  /apps/api/internal/repository/settings_repository.go
  /apps/api/internal/services/backup_service.go
  /apps/api/internal/cache/manager.go

Database:
  Table: settings (exists)
  Migration 005: cache_entries table
```

~~**FR67-FR74: User Authentication & Access Control**~~ — **REMOVED in v4.** Single-user deployment, no authentication required. All auth components (login routes, auth middleware, JWT, bcrypt, user repository) are not needed.

**FR75-FR94: Growth Phase Features (Deferred)**

```
Structure Reserved:
  /apps/api/internal/subtitle/       # Subtitle management (FR75-80)
  /apps/api/internal/watcher/        # Watch folder (FR81-86)
  /apps/api/internal/webhook/        # Webhooks (FR89-90)
  /apps/api/internal/plex/           # Plex integration (FR91)
  /apps/api/internal/jellyfin/       # Jellyfin integration (FR91)

Status: Not implemented in MVP/1.0, structure defined for future
```

---

### Cross-Cutting Concerns Mapping

**Error Handling (All Components)**

```
Backend:
  /apps/api/internal/errors/
    - app_error.go         # Unified AppError type
    - codes.go             # Error code constants
    - *_errors.go          # Domain-specific error constructors

Frontend:
  /apps/web/src/utils/apiError.ts
  /apps/web/src/components/ui/ErrorMessage.tsx
  Global error boundary in App.tsx
```

**Logging (Backend Only)**

```
Backend:
  /apps/api/internal/logger/
    - slog.go              # slog configuration
    - sanitize.go          # Sensitive data filtering
  /apps/api/internal/middleware/logging.go

All handlers, services, repositories use slog:
  slog.Info("...", "key", value)
  slog.Error("...", "error", err, "context", data)
```

**Authentication (All Protected Endpoints)**

```
Backend Middleware Chain:
  Router → Logging → Recovery → CORS → Auth → Handler

Auth middleware:
  /apps/api/internal/middleware/auth.go
  - Extracts JWT from httpOnly cookie
  - Verifies JWT signature
  - Injects user context into request
  - Returns 401 if invalid

Frontend:
  /apps/web/src/stores/authStore.ts
  /apps/web/src/hooks/useAuth.ts
  Global 401 handler in TanStack Query config
```

**Caching (Performance Critical Paths)**

```
Backend:
  /apps/api/internal/cache/manager.go

Used by:
  - MetadataOrchestrator (TMDb, Douban, Wikipedia)
  - AIParser (30-day cache)
  - DownloadService (5-second qBittorrent status cache)

Cache key pattern:
  {source}:{type}:{identifier}:{version}
  Example: tmdb:movie:12345:v1
```

**Background Tasks (Async Operations)**

```
Backend:
  /apps/api/internal/tasks/
    - queue.go                  # Worker pool (3-5 workers)
    - ai_parsing_task.go        # AI parsing (10s async)
    - metadata_refresh_task.go  # Scheduled refresh
    - backup_task.go            # Scheduled backups

Integration:
  main.go initializes TaskQueue on startup
  Handlers submit tasks to queue
  Frontend polls /api/v1/tasks/{id} for status
```

---

## Integration Points

### Internal Communication Patterns

**Frontend ↔ Backend (HTTP/JSON):**

```
Request Flow:
  Component → TanStack Query → API Service → fetch → Backend Handler

Example:
  MovieCard.tsx
    → useQuery(['movies', 'detail', id])
    → movieService.getMovie(id)
    → fetch('/api/v1/movies/:id')
    → movie_handler.GetMovie()
    → service.GetMovieByID()
    → repository.FindByID()
```

**Service ↔ Service (Direct Function Calls):**

```
Within Backend:
  ParserService.ParseFilename()
    → RegexParser.Parse() (internal)
    → if low confidence: AIParser.Parse()
    → MetadataOrchestrator.FindMovie()
    → CacheManager.Set()
    → FilenameMapping.SaveMapping()

No inter-process communication needed (monolithic backend)
```

**Component ↔ Component (Props & Events):**

```
React Component Communication:
  Parent → Child: Props
  Child → Parent: Callback functions
  Global: Context API (minimal use)
  Server State: TanStack Query (shared cache)

Example:
  SearchPage
    → <SearchBar onSearch={handleSearch} />
    → <FilterPanel filters={filters} onChange={setFilters} />
    → <ResultsGrid movies={movies} />
```

---

### External Integrations

**TMDb API (Primary Metadata Source)**

```
Integration Point:
  /apps/api/internal/tmdb/client.go

Communication:
  HTTP GET → api.themoviedb.org/3/
  Authentication: API key in query param
  Rate Limiting: 40 req/10s (NFR-I6)
  Caching: 24-hour TTL (NFR-I7)

Error Handling:
  Timeout (>5s) → TMDB_TIMEOUT
  Rate limit → TMDB_RATE_LIMIT (wait 10s, retry)
  Not found → TMDB_NOT_FOUND (fallback to Douban)
```

**Douban (Secondary Metadata Source)**

```
Integration Point:
  /apps/api/internal/douban/scraper.go

Communication:
  HTTP GET → movie.douban.com/
  Method: Web scraping (no official API)
  Rate Limiting: 1 req/2s (conservative)
  Caching: 7-day TTL

Challenges:
  - Anti-scraping detection (User-Agent rotation)
  - HTML structure changes (fragile selectors)
  - Fallback to Wikipedia if blocked
```

**Wikipedia MediaWiki API (Tertiary Source)**

```
Integration Point:
  /apps/api/internal/wikipedia/client.go

Communication:
  HTTP GET → zh.wikipedia.org/w/api.php
  Rate Limiting: 1 req/s (NFR-I14)
  Caching: 7-day TTL

Usage:
  Search for movie title → extract infobox data
  Fallback if TMDb + Douban both fail
```

**AI Providers (Gemini & Claude)**

```
Integration Points:
  /apps/api/internal/ai/gemini.go
  /apps/api/internal/ai/claude.go

Communication:
  HTTP POST → generativelanguage.googleapis.com (Gemini)
  HTTP POST → api.anthropic.com (Claude)
  Authentication: API key in header (user-provided)
  Timeout: 15 seconds (NFR-I12)
  Caching: 30-day TTL (NFR-I10)

Cost Optimization:
  Only invoke if regex parser fails (confidence <0.7)
  Cache all results aggressively
  Target: <$0.05 per file (NFR-Cost1)
```

**qBittorrent Web API v2**

```
Integration Point:
  /apps/api/internal/qbittorrent/client.go

Communication:
  HTTP GET/POST → {user_configured_host}:{port}/api/v2/
  Authentication: Cookie-based session
  Polling: Every 5 seconds for active downloads (NFR-P8)

Error Handling:
  Connection failed → QBIT_CONNECTION_FAILED
  Auth failed → QBIT_AUTH_FAILED
  Circuit breaker: 5 consecutive failures → disable for 1 minute
```

---

### Data Flow Diagram

```
User Input (Frontend)
    ↓
TanStack Query (Cache Check)
    ↓
API Service (fetch)
    ↓
Backend Handler (Validation)
    ↓
Service Layer (Business Logic)
    ↓
┌─────────────┬─────────────┬──────────────┐
│             │             │              │
Repository   Cache         External API   Background Task
(Database)   (Memory/DB)   (TMDb/AI)      (Worker Pool)
    ↓            ↓              ↓              ↓
SQLite       bigcache       HTTP Client    Goroutines
Database     + cache_       + Retry        + Channels
             entries        + Circuit      + Exponential
             table          Breaker        Backoff
```

**Specific Data Flows:**

**1. Search Flow (FR1):**
```
User types in SearchBar
  → useSearch hook triggers TanStack Query
  → searchService.search(query, filters)
  → GET /api/v1/search?q={query}&genre={genre}
  → search_handler.Search()
  → search_service.SearchMovies()
  → movie_repository.Search() [Uses FTS5 index]
  → SQLite database query
  → Results wrapped in ApiResponse<T>
  → Frontend displays ResultsGrid
```

**2. Filename Parsing Flow (FR11-12):**
```
User uploads filename "電影.1080p.BluRay.x264.mkv"
  → POST /api/v1/parser/filename
  → parser_handler.ParseFilename()
  → parser_service.Parse()
    ├─→ regex_parser.Parse()  [Confidence: 0.4]
    └─→ [Low confidence] ai_parser.Parse()
        ├─→ cache.Get("ai:filename:hash:v1") [Cache miss]
        ├─→ gemini.ParseFilename() [10s async task]
        ├─→ cache.Set("ai:filename:hash:v1", result, 30d)
        └─→ Return parsed result [Confidence: 0.9]
  → metadata_orchestrator.FindMovie(parsed_title)
    ├─→ tmdb.SearchMovie() [zh-TW priority]
    ├─→ [Not found] douban.SearchMovie()
    └─→ [Not found] wikipedia.SearchMovie()
  → filename_mapping.SaveMapping() [Learning system]
  → Return ParseResult with metadata
```

**3. Download Monitoring Flow (FR27):**
```
DownloadService polling (every 5s)
  → qbittorrent.GetTorrents()
  → Compare with previous state
  → If changed: update download_history table
  → Frontend polls GET /api/v1/downloads
  → TanStack Query auto-refetch (refetchInterval: 5000ms)
  → DownloadList updates with new status
```

---

## File Organization Patterns

### Configuration Files Organization

```
Root Level (Monorepo):
  /.gitignore                 # Git ignore rules
  /.env.example               # Environment template (all keys documented)
  /docker-compose.yml         # Multi-container orchestration
  /nx.json                    # Nx workspace config
  /tsconfig.base.json         # Shared TS config

Backend (/apps/api):
  /.air.toml                  # Air hot reload config
  /.env.example               # Backend-specific env vars
  /go.mod                     # Go module dependencies
  /go.sum                     # Dependency lock file

Frontend (/apps/web):
  /vite.config.ts             # Vite build config
  /tailwind.config.js         # Tailwind CSS config
  /postcss.config.js          # PostCSS processing
  /vitest.config.ts           # Test runner config
  /tsconfig.json              # TypeScript config
  /.env.example               # Frontend-specific env vars
```

### Source Code Organization

```
Backend (Feature-Based within Layers):
  /internal/handlers/         # All HTTP handlers
  /internal/services/         # All business logic
  /internal/repository/       # All data access
  /internal/{domain}/         # Domain-specific packages
    - tmdb/                   # TMDb integration
    - ai/                     # AI provider abstraction
    - parser/                 # Filename parsing

Frontend (Feature-Based):
  /components/{feature}/      # Feature-specific components
  /components/ui/             # Shared UI primitives
  /hooks/                     # Shared hooks
  /services/                  # API client services
  /routes/                    # Route components
```

### Test Organization

```
Backend (Co-located):
  /internal/handlers/movie_handler.go
  /internal/handlers/movie_handler_test.go

  /internal/services/search_service.go
  /internal/services/search_service_test.go

Frontend (Co-located):
  /components/search/SearchBar.tsx
  /components/search/SearchBar.spec.tsx

  /hooks/useSearch.ts
  /hooks/useSearch.spec.ts

Test Utilities:
  /apps/web/src/test/setup.ts         # Vitest global setup
  /apps/web/src/test/mockData.ts      # Test data factories
  /apps/web/src/test/testUtils.tsx    # Custom render functions
```

### Asset Organization

```
Static Assets (Frontend):
  /apps/web/public/
    ├── favicon.ico
    ├── logo.svg
    └── assets/
        ├── icons/
        └── images/

Dynamic Assets (Runtime):
  /data/cache/images/
    ├── posters/            # Movie posters from TMDb
    │   └── {tmdb_id}.jpg
    ├── backdrops/          # Backdrop images
    │   └── {tmdb_id}.jpg
    └── avatars/            # User avatars (future)
```

---

## Development Workflow Integration

### Development Server Structure

```
Start Development:
  # Backend (Air hot reload)
  cd apps/api
  air                       # Watches *.go files, rebuilds on change

  # Frontend (Vite dev server)
  cd apps/web
  npm run dev               # HMR enabled, port 5173

  # Nx (monorepo orchestration)
  nx run-many --target=serve --all  # Start all apps in parallel

Development URLs:
  Frontend:  http://localhost:5173
  Backend:   http://localhost:8080
  Swagger:   http://localhost:8080/api/v1/docs
```

### Build Process Structure

```
Backend Build:
  cd apps/api
  go build -o ./dist/vido-api main.go

  Output:
    /apps/api/dist/vido-api         # Single binary

Frontend Build:
  cd apps/web
  npm run build

  Output:
    /apps/web/dist/
      ├── index.html
      ├── assets/
      │   ├── index-{hash}.js       # Main bundle
      │   ├── vendor-{hash}.js      # Vendor chunk
      │   └── *.css                 # Compiled CSS

Nx Build (All Apps):
  nx run-many --target=build --all

  Output:
    /dist/
      ├── apps/api/
      └── apps/web/
```

### Deployment Structure

```
Docker Deployment:
  docker-compose.yml defines:
    - vido-api (backend service)
    - vido-web (frontend served via nginx)
    - volumes for /data (database, cache, backups)

Production Structure:
  /opt/vido/
    ├── vido-api              # Go binary
    ├── web/                  # Frontend static files
    ├── data/
    │   ├── vido.db           # SQLite database
    │   ├── cache/            # Image cache
    │   └── backups/          # DB backups
    └── .env                  # Production env vars

Environment Variables Required:
  # Backend
  PORT=8080
  DATABASE_PATH=/opt/vido/data/vido.db
  JWT_SECRET=<random-32-bytes>
  TMDB_API_KEY=<user-provided>

  # Optional
  GEMINI_API_KEY=<user-provided>
  CLAUDE_API_KEY=<user-provided>
  QBITTORRENT_URL=http://localhost:8081
  QBITTORRENT_USERNAME=admin
  QBITTORRENT_PASSWORD=<user-provided>
```

---

## Summary

**Project Structure Readiness:**

| Aspect | Status | Notes |
|--------|--------|-------|
| Directory Structure | ✅ Complete | All paths defined |
| API Boundaries | ✅ Complete | All endpoints mapped |
| Component Boundaries | ✅ Complete | Communication patterns defined |
| Service Boundaries | ✅ Complete | Responsibilities documented |
| Data Boundaries | ✅ Complete | Schema & access patterns |
| Requirements Mapping | ✅ Complete | All 94 FRs mapped to structure |
| Integration Points | ✅ Complete | Internal & external defined |
| File Organization | ✅ Complete | Config, source, test, assets |
| Development Workflow | ✅ Complete | Dev, build, deploy processes |

**Implementation Sequence:**

1. **Phase 1:** Consolidate root backend into `/apps/api` structure
2. **Phase 2:** Implement missing directories/files per architectural decisions
3. **Phase 3:** Align frontend structure with Tailwind + Vitest
4. **Phase 4:** Build core features per requirements mapping
5. **Phase 5:** Test coverage across all layers

**Structure Compliance with Patterns:**

All directory and file naming follows the patterns defined in Step 5:
- ✅ Backend files: `snake_case.go`
- ✅ Frontend files: `PascalCase.tsx`
- ✅ Tests co-located: `*_test.go`, `*.spec.tsx`
- ✅ Feature-first organization
- ✅ Layered backend architecture (handlers → services → repositories)

**Ready for Implementation:** Complete project structure blueprint established.
