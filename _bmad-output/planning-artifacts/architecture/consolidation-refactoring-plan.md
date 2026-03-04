# Consolidation & Refactoring Plan

## Strategic Decision: Merge Backends into Unified Architecture

**Recommendation:** Consolidate both backends into `/apps/api` as the **single source of truth**

**Rationale:**
1. **Repository Pattern Already Exists** - Apps/api has database foundation
2. **Database Migrations Established** - Migration system in place
3. **Monorepo Structure** - Nx apps/api aligns with apps/web
4. **Less Refactoring Overhead** - Add features to apps/api vs rebuilding database in root

**Migration Strategy:**
- **Target:** `/apps/api` becomes the unified backend
- **Migrate FROM:** Root backend (`/cmd`, `/internal`)
- **Migrate TO:** Apps backend (`/apps/api`)

---

## Phase 1: Backend Consolidation (Priority: 🔴 Critical)

**Objective:** Merge root backend features into apps/api, deprecate root backend

### Step 1.1: Migrate Logging to slog

**Why:** Architectural Decision #6 specifies `slog`, NOT zerolog

**Actions:**
1. Replace zerolog imports with `log/slog`
2. Configure `slog.NewJSONHandler`
3. Implement sensitive data filtering (sanitizeAttr)
4. Update all logging calls to structured format

**Affected Files:**
- `/apps/api/internal/logger/` (create new package)
- All service files using logging

**Example Refactor:**
```go
// BEFORE (zerolog - non-compliant)
log.Info().Str("movie_id", id).Msg("Fetching movie")

// AFTER (slog - compliant)
slog.Info("Fetching movie", "movie_id", id)
```

---

### Step 1.2: Integrate Swaggo Documentation

**Actions:**
1. Install Swaggo: `go get -u github.com/swaggo/swag/cmd/swag`
2. Copy Swagger configuration from root backend
3. Add Swagger annotations to apps/api endpoints
4. Generate spec: `swag init -g cmd/main.go`
5. Add `/docs` endpoint to Gin router

**Affected Files:**
- `/apps/api/main.go` (add Swagger middleware)
- All API handler files (add annotations)
- `/apps/api/docs/` (generated folder)

**Example Annotation:**
```go
// @Summary Get movie by ID
// @Description Retrieve movie metadata from database
// @Tags movies
// @Accept json
// @Produce json
// @Param id path string true "Movie ID"
// @Success 200 {object} models.Movie
// @Failure 404 {object} AppError
// @Router /api/v1/movies/{id} [get]
func (h *MovieHandler) GetMovie(c *gin.Context) { ... }
```

---

### Step 1.3: Integrate TMDb Client

**Actions:**
1. Copy `/internal/tmdb/` package to `/apps/api/internal/tmdb/`
2. Refactor to use `slog` instead of zerolog
3. Integrate with repository layer:
   - Fetch from TMDb → Cache → Store in database
   - Implement cache-aside pattern

**Affected Files:**
- `/apps/api/internal/tmdb/` (migrated package)
- `/apps/api/internal/services/metadata.go` (new service)
- `/apps/api/internal/repository/movie.go` (update with TMDb integration)

**Architecture:**
```
HTTP Request → MetadataService → TMDb Client → Cache → Database Repository
```

---

### Step 1.4: Migrate Advanced Middleware

**Actions:**
1. Copy middleware from `/internal/middleware/` to `/apps/api/internal/middleware/`
2. Update imports to use apps/api module path
3. Apply middleware to Gin router in `main.go`

**Middleware to Migrate:**
- CORS configuration
- Error recovery middleware
- Request ID tracking
- Panic recovery

**Affected Files:**
- `/apps/api/internal/middleware/cors.go`
- `/apps/api/internal/middleware/recovery.go`
- `/apps/api/internal/middleware/request_id.go`
- `/apps/api/main.go` (apply middleware)

---

### Step 1.5: Configure Air Hot Reload for apps/api

**Actions:**
1. Copy `.air.toml` to `/apps/api/.air.toml`
2. Update paths to target `apps/api` directory
3. Update build command: `cd apps/api && go build -o ./tmp/main .`

**Configuration Changes:**
```toml
[build]
  cmd = "go build -o ./tmp/main ."
  bin = "tmp/main"
  include_dir = ["apps/api"]
  exclude_dir = ["tmp", "vendor", "node_modules"]
```

---

## Phase 2: Implement Missing Architectural Decisions (Priority: 🔴 Critical)

### Step 2.1: Implement JWT Authentication (Decision #3)

**Actions:**
1. Install `golang-jwt/jwt` v5.x
2. Create `users` table migration:
   ```sql
   CREATE TABLE users (
       id TEXT PRIMARY KEY,
       username TEXT UNIQUE NOT NULL,
       password_hash TEXT NOT NULL,
       pin_hash TEXT,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```
3. Implement authentication middleware (`/apps/api/internal/middleware/auth.go`)
4. Create login endpoint (`POST /api/v1/auth/login`)
5. Hash passwords with bcrypt (cost factor: 12)

**Affected Files:**
- `/apps/api/migrations/004_create_users_table.sql`
- `/apps/api/internal/middleware/auth.go`
- `/apps/api/internal/handlers/auth.go`
- `/apps/api/internal/services/auth.go`

**Testing Priority:** High (security critical)

---

### Step 2.2: Implement Caching System (Decision #4)

**Actions:**
1. Create `cache_entries` table migration:
   ```sql
   CREATE TABLE cache_entries (
       cache_key TEXT PRIMARY KEY,
       cache_value BLOB,
       created_at INTEGER,
       expires_at INTEGER,
       cache_type TEXT,
       hit_count INTEGER DEFAULT 0
   );
   CREATE INDEX idx_expires_at ON cache_entries(expires_at);
   ```
2. Install in-memory cache library: `go get github.com/allegro/bigcache/v3`
3. Implement `CacheManager` (`/apps/api/internal/cache/manager.go`)
4. Integrate with TMDb client (24-hour TTL)
5. Prepare for AI parsing (30-day TTL)

**Affected Files:**
- `/apps/api/migrations/005_create_cache_entries_table.sql`
- `/apps/api/internal/cache/manager.go`
- `/apps/api/internal/cache/memory.go`
- `/apps/api/internal/cache/sqlite.go`

---

### Step 2.3: Implement Background Task Queue (Decision #5)

**Actions:**
1. Create `TaskQueue` implementation (`/apps/api/internal/tasks/queue.go`)
2. Define task types: `AIParsingTask`, `MetadataRefreshTask`, `BackupTask`
3. Implement worker pool with 3-5 goroutines
4. Add retry logic with exponential backoff

**Affected Files:**
- `/apps/api/internal/tasks/queue.go`
- `/apps/api/internal/tasks/ai_parsing.go`
- `/apps/api/internal/tasks/metadata_refresh.go`
- `/apps/api/main.go` (initialize queue on startup)

---

### Step 2.4: Implement Unified Error Types (Decision #6)

**Actions:**
1. Create `AppError` type (`/apps/api/internal/errors/app_error.go`)
2. Define error codes (TMDB_*, AI_*, QBIT_*, DB_*, AUTH_*)
3. Create error constructor functions
4. Integrate with Gin error handling middleware

**Affected Files:**
- `/apps/api/internal/errors/app_error.go`
- `/apps/api/internal/errors/codes.go`
- All service files (return `AppError` instead of generic `error`)

---

## Phase 3: Frontend Alignment (Priority: 🟡 Important)

### Step 3.1: Configure Tailwind CSS (Decision #1)

**Actions:**
1. Install: `npm install -D tailwindcss postcss autoprefixer`
2. Initialize: `npx tailwindcss init -p`
3. Configure design tokens in `tailwind.config.js`
4. Add Tailwind directives to main CSS file

**Affected Files:**
- `/apps/web/tailwind.config.js` (create)
- `/apps/web/postcss.config.js` (create)
- `/apps/web/src/styles/global.css` (add directives)

---

### Step 3.2: Set Up Frontend Testing (Decision #2)

**Actions:**
1. Install Vitest: `npm install -D vitest @testing-library/react @testing-library/jest-dom`
2. Configure `vitest.config.ts`
3. Create test setup file with providers
4. Write first component test (smoke test)

**Affected Files:**
- `/apps/web/vitest.config.ts` (create)
- `/apps/web/src/test/setup.ts` (create)
- `/apps/web/src/app/app.spec.tsx` (create first test)

---

## Phase 4: Core Feature Implementation (Priority: 🔴 Critical)

### Step 4.1: Implement Filename Parser

**Actions:**
1. Create regex-based parser (`/apps/api/internal/parser/regex.go`)
2. Implement AI provider abstraction (`/apps/api/internal/ai/provider.go`)
3. Add Gemini client (`/apps/api/internal/ai/gemini.go`)
4. Add Claude client (`/apps/api/internal/ai/claude.go`)
5. Create filename mappings table:
   ```sql
   CREATE TABLE filename_mappings (
       id TEXT PRIMARY KEY,
       original_filename TEXT UNIQUE NOT NULL,
       parsed_title TEXT NOT NULL,
       parsed_year INTEGER,
       confidence_score REAL,
       source TEXT,  -- 'regex', 'ai_gemini', 'ai_claude', 'manual'
       created_at TIMESTAMP
   );
   ```

**Affected Files:**
- `/apps/api/internal/parser/` (new package)
- `/apps/api/internal/ai/` (new package)
- `/apps/api/migrations/006_create_filename_mappings_table.sql`

---

### Step 4.2: Implement Multi-Source Metadata Orchestrator

**Actions:**
1. Create orchestrator service (`/apps/api/internal/services/metadata_orchestrator.go`)
2. Implement TMDb → Douban → Wikipedia → AI → Manual fallback chain
3. Add circuit breaker pattern for external services
4. Integrate with caching system

**Affected Files:**
- `/apps/api/internal/services/metadata_orchestrator.go`
- `/apps/api/internal/douban/` (new Douban scraper)
- `/apps/api/internal/wikipedia/` (new Wikipedia client)

---

### Step 4.3: Build Media Search UI

**Actions:**
1. Create search page (`/apps/web/src/routes/search.tsx`)
2. Create search components:
   - `SearchBar.tsx`
   - `FilterPanel.tsx`
   - `ResultsGrid.tsx`
   - `ResultsList.tsx`
3. Implement TanStack Query hooks for search API
4. Add Tailwind styling

**Affected Files:**
- `/apps/web/src/routes/search.tsx`
- `/apps/web/src/components/search/` (new directory)
- `/apps/web/src/hooks/useSearch.ts`

---

### Step 4.4: Build Media Library UI

**Actions:**
1. Create library page (`/apps/web/src/routes/library.tsx`)
2. Create library components:
   - `MediaGrid.tsx`
   - `MediaList.tsx`
   - `FilterControls.tsx`
3. Implement virtual scrolling for >1,000 items
4. Add batch operation controls

**Affected Files:**
- `/apps/web/src/routes/library.tsx`
- `/apps/web/src/components/library/` (new directory)
- `/apps/web/src/hooks/useLibrary.ts`

---

## Phase 5: Testing & Quality Gates (Priority: 🟡 Important)

### Step 5.1: Backend Testing

**Actions:**
1. Write repository tests with SQLite in-memory
2. Write service tests with mocked dependencies
3. Write API endpoint tests
4. Achieve >80% coverage

**Test Files:**
- `/apps/api/internal/repository/*_test.go`
- `/apps/api/internal/services/*_test.go`
- `/apps/api/internal/handlers/*_test.go`

---

### Step 5.2: Frontend Testing

**Actions:**
1. Write component tests for all UI components
2. Write hook tests for custom hooks
3. Write integration tests for critical user flows
4. Achieve >70% coverage

**Test Files:**
- `/apps/web/src/components/**/*.spec.tsx`
- `/apps/web/src/hooks/*.spec.ts`

---

## Deprecation Plan for Root Backend

**Timeline:** After Phase 1 completion

**Actions:**
1. Add deprecation notice to `/cmd/api/main.go`
2. Update documentation to point to `/apps/api`
3. Archive root backend code to `/archive/` directory
4. Update CI/CD to build apps/api only

**Files to Archive:**
- `/cmd/api/` → `/archive/cmd/api/`
- `/internal/` → `/archive/internal/`

---

## Refactoring Effort Estimate

**Total Estimated Effort:** 15-20 development days

**Breakdown by Phase:**
- Phase 1 (Backend Consolidation): 5 days
- Phase 2 (Architectural Decisions): 5 days
- Phase 3 (Frontend Alignment): 2 days
- Phase 4 (Core Features): 6 days
- Phase 5 (Testing): 3 days

**Critical Path:**
Phase 1 → Phase 2 (Steps 2.1-2.4) → Phase 4 (Steps 4.1-4.4)

**Parallelizable Work:**
- Phase 3 (Tailwind + Vitest) can run parallel to Phase 2
- Phase 5 (Testing) should be continuous throughout

---

## Risk Mitigation

**Risk 1: Data Loss During Consolidation**
- **Mitigation:** Use feature branches, test migration thoroughly, backup existing databases

**Risk 2: Breaking Changes in Dependencies**
- **Mitigation:** Pin dependency versions in go.mod and package.json

**Risk 3: Regression in Existing Features**
- **Mitigation:** Write tests BEFORE refactoring, use TDD approach

**Risk 4: Incomplete Migration**
- **Mitigation:** Use checklist tracking (TodoWrite tool), verify each step

---

## Success Criteria

**Phase 1 Complete When:**
- [ ] Single backend at `/apps/api` with ALL features
- [ ] Swaggo documentation accessible at `/docs`
- [ ] slog structured logging operational
- [ ] Air hot reload functional
- [ ] Root backend archived

**Phase 2 Complete When:**
- [ ] JWT authentication protecting all endpoints
- [ ] Caching system operational with hit rate >80%
- [ ] Background tasks processing AI parsing without blocking UI
- [ ] Unified error types returning consistent messages

**Phase 3 Complete When:**
- [ ] Tailwind CSS configured and design tokens defined
- [ ] Vitest test suite running with >10 passing tests

**Phase 4 Complete When:**
- [ ] Filename parser (regex + AI) operational
- [ ] Multi-source metadata fallback working
- [ ] Search UI functional with filters
- [ ] Media library UI functional with virtual scrolling

**Phase 5 Complete When:**
- [ ] Backend test coverage >80%
- [ ] Frontend test coverage >70%
- [ ] CI pipeline running all tests

---
