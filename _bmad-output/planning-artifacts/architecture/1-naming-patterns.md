# 1. Naming Patterns

## 1.1 Database Naming Conventions

**MANDATORY Rules:**

**Table Naming:**
- ✅ **Pattern:** `snake_case`, **plural nouns**
- ✅ **Examples:** `movies`, `series`, `users`, `cache_entries`, `filename_mappings`
- ❌ **Anti-pattern:** `Movies`, `movie`, `Movie`

**Column Naming:**
- ✅ **Pattern:** `snake_case`
- ✅ **Examples:** `tmdb_id`, `created_at`, `user_id`, `release_date`
- ❌ **Anti-pattern:** `tmdbId`, `createdAt`, `userId`

**Primary Key:**
- ✅ **Pattern:** `id` (TEXT type for UUIDs)
- ✅ **Example:** `id TEXT PRIMARY KEY`
- ❌ **Anti-pattern:** `movie_id`, `{table}_id` for primary keys

**Foreign Key:**
- ✅ **Pattern:** `{referenced_table}_id`
- ✅ **Examples:** `user_id`, `movie_id`, `series_id`
- ❌ **Anti-pattern:** `fk_user`, `userId`

**Index Naming:**
- ✅ **Pattern:** `idx_{table}_{column}` or `idx_{table}_{column1}_{column2}`
- ✅ **Examples:** `idx_movies_tmdb_id`, `idx_movies_title`, `idx_cache_entries_expires_at`
- ❌ **Anti-pattern:** `movies_tmdb_index`, `title_idx`

**Migration File Naming:**
- ✅ **Pattern:** `{sequence}_{description}.sql`
- ✅ **Examples:** `001_create_movies_table.sql`, `004_create_users_table.sql`
- ❌ **Anti-pattern:** `create-movies.sql`, `1_movies.sql`

**Current Codebase Compliance:**
- ✅ **apps/api migrations:** Fully compliant with naming conventions
- ✅ **Existing tables:** `movies`, `series`, `settings` follow snake_case plural pattern
- ⚠️ **Migration needed:** No violations found, continue following existing pattern

---

## 1.2 API Naming Conventions

**MANDATORY Rules:**

**Endpoint Paths:**
- ✅ **Pattern:** `/api/v{version}/{resource}` with **plural nouns**
- ✅ **Examples:**
  - `GET /api/v1/movies`
  - `GET /api/v1/movies/{id}`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/downloads`
- ❌ **Anti-pattern:** `/movie`, `/api/movie`, `/v1/movie`, `/getMovies`

**HTTP Methods:**
- ✅ **Pattern:** RESTful standard mapping
  - `GET` - Retrieve resource(s)
  - `POST` - Create new resource
  - `PUT` - Replace entire resource
  - `PATCH` - Partial update
  - `DELETE` - Remove resource
- ❌ **Anti-pattern:** `POST /api/v1/movies/update`, `GET /api/v1/movies/create`

**Route Parameters:**
- ✅ **Pattern:** `{parameter_name}` (Gin syntax)
- ✅ **Examples:** `/api/v1/movies/{id}`, `/api/v1/series/{id}/seasons/{season_number}`
- ❌ **Anti-pattern:** `:id`, `{movieId}`, `{movie-id}`

**Query Parameters:**
- ✅ **Pattern:** `snake_case`
- ✅ **Examples:** `?sort_by=release_date`, `?filter_genre=action`, `?page=1&per_page=20`
- ❌ **Anti-pattern:** `?sortBy=releaseDate`, `?filterGenre=action`

**HTTP Headers:**
- ✅ **Pattern:** `X-Vido-{Header-Name}` for custom headers
- ✅ **Examples:** `X-Vido-Request-ID`, `X-Vido-Client-Version`
- ❌ **Anti-pattern:** `Request-ID`, `client-version`

**Current Codebase Compliance:**
- ⚠️ **Root backend:** Endpoints not yet implemented (Swagger config exists)
- ⚠️ **Apps backend:** Only health check endpoint exists
- 🔴 **Migration needed:** Implement all endpoints following `/api/v1/{resource}` pattern during Phase 1-4

---

## 1.3 Code Naming Conventions

**Backend (Go) Naming:**

**Package Naming:**
- ✅ **Pattern:** `lowercase`, **singular nouns**, no underscores
- ✅ **Examples:** `tmdb`, `parser`, `middleware`, `cache`, `repository`
- ❌ **Anti-pattern:** `tmdb_client`, `Middleware`, `repositories`

**Struct Naming:**
- ✅ **Pattern:** `PascalCase`
- ✅ **Examples:** `Movie`, `TMDbClient`, `CacheManager`, `AppError`
- ❌ **Anti-pattern:** `movie`, `tmdbClient`, `cache_manager`

**Interface Naming:**
- ✅ **Pattern:** `PascalCase` with descriptive noun (NOT `-er` suffix unless idiomatic)
- ✅ **Examples:** `Repository`, `Cache`, `TaskQueue` (BUT: `Handler`, `Parser` acceptable)
- ❌ **Anti-pattern:** `IRepository`, `MovieRepositoryInterface`

**Function/Method Naming:**
- ✅ **Pattern:** `PascalCase` for exported, `camelCase` for unexported
- ✅ **Examples:** `GetMovieByID`, `CreateUser`, `parseFilename` (unexported)
- ❌ **Anti-pattern:** `get_movie_by_id`, `Createuser`

**Variable Naming:**
- ✅ **Pattern:** `camelCase` for locals, `PascalCase` for exported
- ✅ **Examples:** `movieID`, `tmdbClient`, `UserAgent` (exported const)
- ❌ **Anti-pattern:** `movie_id`, `TmdbClient` (local var)

**Constant Naming:**
- ✅ **Pattern:** `PascalCase` or `SCREAMING_SNAKE_CASE` for enum-like constants
- ✅ **Examples:** `DefaultCacheSize`, `MAX_RETRY_ATTEMPTS`
- ❌ **Anti-pattern:** `default_cache_size`, `maxRetryAttempts`

**File Naming:**
- ✅ **Pattern:** `snake_case.go`
- ✅ **Examples:** `tmdb_client.go`, `cache_manager.go`, `app_error.go`, `tmdb_client_test.go`
- ❌ **Anti-pattern:** `TMDbClient.go`, `cacheManager.go`, `tmdb-client.go`

**Current Codebase Compliance:**
- ✅ **Root backend:** Follows Go conventions (struct: `TMDbClient`, package: `tmdb`)
- ✅ **Apps backend:** Follows Go conventions (struct: `Movie`, package: `repository`)
- 🔴 **Migration needed:** Enforce `slog` usage (currently using `zerolog` in root backend)

---

**Frontend (TypeScript/React) Naming:**

**Component Naming:**
- ✅ **Pattern:** `PascalCase` for components
- ✅ **Examples:** `SearchBar`, `MovieCard`, `FilterPanel`, `ResultsGrid`
- ❌ **Anti-pattern:** `searchBar`, `movie-card`, `filter_panel`

**Component File Naming:**
- ✅ **Pattern:** `PascalCase.tsx` matching component name
- ✅ **Examples:** `SearchBar.tsx`, `MovieCard.tsx`, `FilterPanel.tsx`
- ❌ **Anti-pattern:** `search-bar.tsx`, `movie_card.tsx`, `searchBar.tsx`

**Hook Naming:**
- ✅ **Pattern:** `use{DescriptiveName}` in `camelCase`
- ✅ **Examples:** `useSearch`, `useMovieQuery`, `useAuth`, `useDownloadStatus`
- ❌ **Anti-pattern:** `UseSearch`, `movieQuery`, `authHook`

**Hook File Naming:**
- ✅ **Pattern:** `use{Name}.ts`
- ✅ **Examples:** `useSearch.ts`, `useMovieQuery.ts`, `useAuth.ts`
- ❌ **Anti-pattern:** `search.hook.ts`, `use-search.ts`, `movie-query.ts`

**Utility Function Naming:**
- ✅ **Pattern:** `camelCase`, descriptive verbs
- ✅ **Examples:** `formatDate`, `sanitizeFilename`, `parseMovieTitle`
- ❌ **Anti-pattern:** `FormatDate`, `format_date`, `date`

**Type/Interface Naming:**
- ✅ **Pattern:** `PascalCase`, descriptive nouns, **NO** `I` prefix
- ✅ **Examples:** `Movie`, `SearchResult`, `ApiResponse<T>`, `DownloadStatus`
- ❌ **Anti-pattern:** `IMovie`, `movieType`, `search_result`

**Enum Naming:**
- ✅ **Pattern:** `PascalCase` for enum, `SCREAMING_SNAKE_CASE` for values
- ✅ **Example:**
  ```typescript
  enum DownloadStatus {
    DOWNLOADING = 'DOWNLOADING',
    PAUSED = 'PAUSED',
    COMPLETED = 'COMPLETED',
    ERROR = 'ERROR',
  }
  ```
- ❌ **Anti-pattern:** `downloadStatus`, values like `downloading`, `Downloading`

**Constant Naming:**
- ✅ **Pattern:** `SCREAMING_SNAKE_CASE` for true constants
- ✅ **Examples:** `API_BASE_URL`, `MAX_SEARCH_RESULTS`, `DEFAULT_PAGE_SIZE`
- ❌ **Anti-pattern:** `apiBaseUrl`, `MaxSearchResults`, `default-page-size`

**CSS Class Naming (Tailwind):**
- ✅ **Pattern:** Tailwind utility classes, component-scoped classes use `kebab-case`
- ✅ **Examples:**
  - Tailwind: `className="flex items-center justify-between p-4 bg-gray-100"`
  - Custom: `className="movie-card-container"` (if absolutely needed)
- ❌ **Anti-pattern:** `className="movieCard"`, inline styles

**Current Codebase Compliance:**
- ✅ **Existing components:** Follow `PascalCase` naming (`App.tsx`, `NxWelcome.tsx`)
- ✅ **Shared types:** Follow TypeScript conventions (`Movie`, `ApiResponse<T>`)
- ⚠️ **Incomplete:** Most components don't exist yet, enforce patterns in Phase 4

---

## 1.4 Route/Path Naming

**Frontend Routes (TanStack Router):**
- ✅ **Pattern:** `kebab-case` for URL paths
- ✅ **Examples:**
  - `/search`
  - `/library`
  - `/downloads`
  - `/settings`
  - `/media/{id}` (route parameter)
- ❌ **Anti-pattern:** `/Search`, `/mediaLibrary`, `/download_list`

**Route File Naming (TanStack Router):**
- ✅ **Pattern:** Match route path, use `index.tsx` for root
- ✅ **Examples:**
  - `/routes/search.tsx` → `/search`
  - `/routes/library.tsx` → `/library`
  - `/routes/media/$id.tsx` → `/media/{id}`
  - `/routes/__root.tsx` → root layout
- ❌ **Anti-pattern:** `/routes/Search.tsx`, `/routes/media-detail.tsx`

**Current Codebase Compliance:**
- ✅ **Existing routes:** `__root.tsx`, `index.tsx` follow conventions
- ⚠️ **Incomplete:** Feature routes not implemented, enforce during Phase 4

---
