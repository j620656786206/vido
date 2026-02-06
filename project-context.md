# Vido Project Context - AI Agent Quick Reference

> **Purpose:** Mandatory reading for all AI agents before implementing ANY code. This document ensures consistency across all implementations.

**Full Documentation:** See `_bmad-output/planning-artifacts/architecture.md` for complete architectural decisions and patterns.

**Last Updated:** 2026-02-06
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
See `_bmad-output/planning-artifacts/architecture.md` Section "Consolidation & Refactoring Plan" for complete 5-phase roadmap.

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
- **Pattern:** Co-located tests (`*_test.go`, `*.spec.tsx`)

### 3. Authentication: JWT Stateless

- **Library:** `golang-jwt/jwt` v5.x
- **Storage:** httpOnly cookies
- **Expiration:** 24 hours
- **Password Hashing:** bcrypt (cost factor 12)

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
Database:   snake_case plural (movies, users)
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
AUTH_INVALID_CREDENTIALS, AUTH_TOKEN_EXPIRED
VALIDATION_REQUIRED_FIELD, VALIDATION_INVALID_FORMAT
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
✅ /api/v1/auth/login
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
        └── architecture.md     # Complete architecture doc
```

---

## 📝 Naming Conventions Quick Reference

### Database (SQLite)

| Element     | Pattern                | Example                       | ❌ Anti-pattern       |
| ----------- | ---------------------- | ----------------------------- | --------------------- |
| Tables      | snake_case plural      | `movies`, `users`             | `Movies`, `movie`     |
| Columns     | snake_case             | `tmdb_id`, `created_at`       | `tmdbId`, `createdAt` |
| Primary Key | `id`                   | `id TEXT PRIMARY KEY`         | `movie_id`            |
| Foreign Key | `{table}_id`           | `user_id`, `movie_id`         | `fk_user`, `userId`   |
| Indexes     | `idx_{table}_{column}` | `idx_movies_tmdb_id`          | `movies_tmdb_index`   |
| Migrations  | `{seq}_{desc}.sql`     | `001_create_movies_table.sql` | `create-movies.sql`   |

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
| Hooks            | use + camelCase | `useSearch`, `useAuth`        | `UseSearch`, `searchHook` |
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
interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  login: (credentials: Credentials) => Promise<void>;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  user: null,
  login: async (credentials) => {
    /* ... */
  },
  logout: () => set({ isAuthenticated: false, user: null }),
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

### Session-Aware Process Management

E2E tests (Playwright) automatically start Go backend and Vite dev server. To prevent orphaned processes from consuming CPU after tests crash or complete:

**Automatic Cleanup (Built-in):**
- `globalSetup` creates a session-specific tracking file
- `globalTeardown` cleans up only processes from the current session
- Safe for multiple Claude Code sessions running tests in parallel

**Manual Cleanup Commands:**
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
- Processes on ports 8080, 4200

---

## ✅ Pre-Commit Checklist

Before committing code, verify:

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

**Testing (Definition of Done):**

- [ ] `go test ./...` passes with no failures
- [ ] Services test coverage ≥ 80%
- [ ] Handlers test coverage ≥ 70%
- [ ] Tests co-located with source files (`*_test.go`, `*.spec.tsx`)

**Integration (Definition of Done):**

- [ ] New Services/Handlers wired up in `main.go`
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

- All Rules (1-11) must be followed
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

- **Architecture Decisions:** `_bmad-output/planning-artifacts/architecture.md`
- **PRD:** `_bmad-output/planning-artifacts/prd.md`
- **UX Design:** `_bmad-output/planning-artifacts/ux-design-specification.md`

**Key Sections in architecture.md:**

- Core Architectural Decisions (Step 4)
- Implementation Patterns & Consistency Rules (Step 5)
- Current Implementation Analysis (Brownfield Assessment)
- Consolidation & Refactoring Plan (5 Phases)

---

## ✅ Architecture Validation Summary

**Validation Status:** COMPLETE (2026-01-12)

The complete architecture has been validated for:

- ✅ **Coherence:** All 6 architectural decisions work together without conflicts
- ✅ **Coverage:** All 94 functional requirements are architecturally supported
- ✅ **Readiness:** 47 implementation patterns ensure AI agent consistency

**Key Deliverables:**

- 6 architectural decisions documented with versions and rationale
- 47 implementation patterns preventing AI agent conflicts (see architecture.md)
- 400+ files/directories defined in complete project structure
- 5-phase consolidation roadmap from current to target state

**Confidence Level:** HIGH - Ready for implementation with comprehensive guidance.

---

## 🚀 Implementation Workflow

1. **Read this file FIRST** before implementing any feature
2. **Check architecture.md** for specific pattern details if needed
3. **Follow the consolidation plan** (Phase 1-5) for refactoring
4. **Verify checklist** before committing code
5. **Write tests** alongside implementation (TDD encouraged)

---

**Questions or clarifications?** Refer to the full architecture document or ask the user.

**Last reminder:** ALL new backend code goes to `/apps/api`. The root backend is deprecated.
