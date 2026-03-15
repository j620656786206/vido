# Core Architectural Decisions

## Decision Priority Analysis

The following architectural decisions were made collaboratively, prioritized by their impact on implementation readiness and system architecture.

**🔴 Critical Decisions (Block Implementation):**

1. **CSS Framework:** Tailwind CSS
2. **Testing Infrastructure:** Go testing + testify (backend), Vitest + React Testing Library (frontend)
3. **Authentication Strategy:** JWT (Stateless)
4. **Caching Implementation:** In-Memory + SQLite tiered architecture
5. **Background Task Processing:** Lightweight Worker Pool + Channel
6. **Error Handling & Logging:** Structured logging (slog) + Unified error types

**🟡 Important Decisions (Shape Architecture):**
- Deferred to implementation phase based on specific component needs

**🟢 Deferred Decisions (Post-MVP):**
- CI/CD Platform (GitHub Actions recommended, but can be decided during setup)
- Monitoring & Observability Tools (Prometheus/Grafana or similar, post-1.0)

---

## 1. Frontend Styling: Tailwind CSS

**Decision:** Use Tailwind CSS as the primary styling solution

**Version:** Tailwind CSS v3.x (latest stable)

**Rationale:**
- **Bundle Size Optimization:** Atomic CSS approach minimizes final bundle size, aligning with <500KB gzipped target (NFR-P17)
- **Vite Integration:** Excellent first-class support with Vite build system
- **Development Velocity:** Utility-first approach accelerates component development
- **Design System Consistency:** Component-based design tokens ensure UI consistency
- **Responsive Design:** Built-in utilities support desktop-optimized and mobile-simplified requirements from UX spec

**Implementation Requirements:**

1. **Configuration Setup:**
   - Install: `npm install -D tailwindcss postcss autoprefixer`
   - Initialize: `npx tailwindcss init -p`
   - Configure `tailwind.config.js` with custom theme

2. **Design System Tokens:**
   ```javascript
   // tailwind.config.js
   module.exports = {
     theme: {
       extend: {
         colors: {
           primary: { /* Traditional Chinese UI colors */ },
           secondary: { /* ... */ },
         },
         screens: {
           'mobile': '320px',
           'tablet': '768px',
           'desktop': '1024px',
         },
       },
     },
   }
   ```

3. **Component Library Consideration:**
   - **Option:** Headless UI (by Tailwind Labs) for accessible components
   - **Alternative:** shadcn/ui for pre-built Tailwind components
   - **Decision:** Defer to implementation phase based on component needs

**Affects:**
- All frontend components and pages
- Design system documentation
- Storybook setup (if adopted)

**Alternatives Considered:**
- CSS Modules: More verbose, manual theme management
- CSS-in-JS: Runtime overhead, larger bundles

---

## 2. Testing Infrastructure

### Backend Testing: Go testing + testify

**Decision:** Use Go standard `testing` package with `testify` assertions and mocks

**Version:** 
- Go 1.21+ standard library `testing`
- testify v1.9.x

**Rationale:**
- **Zero Core Dependencies:** `testing` is part of Go standard library
- **Community Standard:** Widely adopted pattern in Go ecosystem
- **Sufficient Tooling:** testify provides rich assertions (`assert`, `require`) and mocking (`mock`, `suite`)
- **Air Integration:** Works seamlessly with Air hot reload during development
- **Simplicity:** Minimal learning curve for Go developers

**Coverage Target:** >80% (NFR-M1)

**Implementation Requirements:**

1. **Test Organization:**
   ```
   internal/
   ├── parser/
   │   ├── parser.go
   │   └── parser_test.go
   ├── metadata/
   │   ├── tmdb.go
   │   └── tmdb_test.go
   ```

2. **Testing Utilities:**
   - Test database fixtures (SQLite in-memory)
   - Mock HTTP clients for external APIs
   - Test helpers for common assertions

3. **CI Integration:**
   - Run tests: `go test ./...`
   - Coverage report: `go test -cover -coverprofile=coverage.out ./...`
   - Coverage gate: Fail if <80%

**Test Categories:**
- **Unit Tests:** Individual function/method testing
- **Integration Tests:** Database interactions, API client tests
- **Table-Driven Tests:** Leverage Go's table-driven testing pattern

### Frontend Testing: Vitest + React Testing Library

**Decision:** Use Vitest as test runner with React Testing Library for component testing

**Version:**
- Vitest v1.x
- React Testing Library v14.x
- @testing-library/jest-dom for DOM matchers

**Rationale:**
- **Vite Native Integration:** Uses same Vite config, extremely fast test execution
- **Jest API Compatibility:** Familiar API for developers with Jest experience
- **TypeScript First-Class Support:** Native TypeScript support without additional configuration
- **React Testing Best Practices:** RTL encourages testing user behavior over implementation details
- **HMR for Tests:** Hot reload for test files during development

**Coverage Target:** >70% (NFR-M2)

**Implementation Requirements:**

1. **Vitest Configuration:**
   ```typescript
   // vitest.config.ts
   import { defineConfig } from 'vitest/config'
   import react from '@vitejs/plugin-react'
   
   export default defineConfig({
     plugins: [react()],
     test: {
       environment: 'jsdom',
       setupFiles: ['./src/test/setup.ts'],
       coverage: {
         provider: 'v8',
         reporter: ['text', 'json', 'html'],
         threshold: {
           lines: 70,
           functions: 70,
           branches: 70,
           statements: 70,
         },
       },
     },
   })
   ```

2. **Testing Utilities:**
   - Custom render wrapper with TanStack Query provider
   - Mock router setup for TanStack Router
   - Test data factories
   - MSW (Mock Service Worker) for API mocking

3. **Test Categories:**
   - **Component Tests:** User interactions, rendering, props
   - **Hook Tests:** Custom React hooks with `@testing-library/react-hooks`
   - **Integration Tests:** Multi-component workflows

### E2E Testing: Dual-Layer Strategy (Playwright + TestSprite)

**Decision:** Two-tier E2E testing — Playwright for feature-level tests, TestSprite for PRD acceptance-criteria journey tests

**Tier 1 — Playwright (Active, 328 test cases):**
- Feature-level E2E tests co-located with stories
- Cross-browser testing with auto-wait mechanisms
- Runs in CI nightly or when related story changes
- 25 spec files covering individual features (API + UI)

**Tier 2 — TestSprite (Installed, deferred until Epic 5+6 complete):**
- AI-powered journey-level testing via MCP server
- Tests run in TestSprite cloud sandbox (requires external access to app)
- 62 test cases generated across 9 categories, mapping to 6 P0 user journeys
- Test plan: `testsprite_tests/testsprite_frontend_test_plan.json`

**6 P0 Critical User Journeys (TestSprite):**
1. 搜尋 → 瀏覽 → 查看詳情
2. 檔名解析 → 元資料匹配 → 手動修正
3. 下載監控全流程
4. 連線健康 → 降級 → 恢復
5. 媒體庫瀏覽與互動
6. qBittorrent 連線設定

**TestSprite Activation Prerequisites (target: after Epic 5+6):**
- [ ] Epic 5 (Media Library Management) fully complete
- [ ] Epic 6 (System Config & Backup) complete
- [ ] Seed data script (`scripts/seed-test-data.sh`) for test database
- [ ] External access via ngrok/cloudflared tunnel or staging deployment
- [ ] Run in production mode to unlock all 62 tests (dev mode limits to 15)

**Initial Test Run (2026-03-15):** 15/62 tests executed in dev mode, 3 passed, 12 failed due to infrastructure gaps (cloud sandbox cannot reach localhost, empty DB, zh-TW label mismatches). No app bugs identified.

**TestSprite Account:** Free plan, 150 credits, MCP server installed at project scope

---

## 3. Authentication Strategy: JWT (Stateless)

**Decision:** Implement JSON Web Token (JWT) based stateless authentication

**Version:** golang-jwt/jwt v5.x

**Rationale:**
- **Stateless Architecture:** Server doesn't need to maintain session state, simplifying self-hosted deployment (no Redis required)
- **Scalability Preparation:** Easier horizontal scaling for future multi-user scenarios (FR73)
- **API-First Design:** Natural fit for RESTful API architecture (NFR-I16)
- **Standard Protocol:** Widely adopted, mature libraries, extensive documentation
- **Self-Hosted Simplicity:** Zero external dependencies beyond the application itself

**Security Parameters:**
- **Token Expiration:** 24 hours (balances security with user experience)
- **Storage:** httpOnly cookie (prevents XSS attacks)
- **Signing Algorithm:** HS256 (HMAC-SHA256)
- **Secret Key:** Environment variable `JWT_SECRET` (minimum 32 bytes)

**Implementation Requirements:**

1. **Backend JWT Middleware (Gin):**
   ```go
   // middleware/auth.go
   func AuthMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           tokenString := extractTokenFromCookie(c)
           token, err := jwt.Parse(tokenString, keyFunc)
           
           if err != nil || !token.Valid {
               c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
               return
           }
           
           // Inject user context
           claims := token.Claims.(jwt.MapClaims)
           c.Set("user_id", claims["user_id"])
           c.Next()
       }
   }
   ```

2. **Login Flow:**
   - User submits password/PIN → Validate with bcrypt hash (NFR-S9)
   - Generate JWT with claims: `{user_id, exp, iat}`
   - Set httpOnly cookie with JWT
   - Return success response

3. **Frontend Integration:**
   - TanStack Query auth mutation
   - Automatic retry on 401 (redirect to login)
   - Token refresh mechanism (if implementing shorter expiration)

4. **Password Security:**
   - Use bcrypt for password hashing (cost factor: 12)
   - Minimum password requirements: 8 characters (configurable)
   - Support for PIN (4-6 digits) as alternative (NFR-S9)

**Logout Mechanism:**
- Clear httpOnly cookie
- Client-side state cleanup

**Multi-User Preparation:**
- JWT already supports multiple users
- Add `role` claim when implementing RBAC (Growth phase)
- Database schema includes `users` table (architecture ready)

**Alternatives Considered:**
- **Session-based:** Requires session store (Redis/database), stateful, harder to scale
- **Hybrid (JWT + Refresh Token):** Added complexity, deferred to Growth phase if needed

---

## 4. Caching Strategy: Tiered In-Memory + SQLite

**Decision:** Implement tiered caching using in-memory cache (hot data) and SQLite (persistent cache)

**Implementation:** Custom CacheManager with multiple cache tiers

**Rationale:**
- **Zero External Dependencies:** No Redis required, simplifies self-hosted deployment (aligns with architecture principle)
- **Performance Optimization:** Memory cache provides <1ms access for hot data
- **Cost Control:** 30-day AI parsing cache dramatically reduces API costs (per-file <$0.05, per-user/month <$2)
- **Persistence:** SQLite ensures cache survives restarts for expensive operations (AI parsing)
- **Tiered Strategy:** Optimized for different data characteristics and access patterns

**Cache Tiers:**

**Tier 1: In-Memory Cache (Hot Data)**
- **Technology:** bigcache or ristretto (Go libraries)
- **Use Cases:** 
  - TMDb API responses (24-hour TTL, NFR-I7)
  - Frequently accessed metadata
  - qBittorrent status (5-second refresh, reduce API calls)
- **Capacity:** Configurable (default: 100MB)
- **Eviction:** LRU (Least Recently Used)

**Tier 2: SQLite Persistent Cache (Cold Data)**
- **Table:** `cache_entries`
  ```sql
  CREATE TABLE cache_entries (
      cache_key TEXT PRIMARY KEY,
      cache_value BLOB,
      created_at INTEGER,
      expires_at INTEGER,
      cache_type TEXT,  -- 'ai_parsing', 'metadata', 'image_meta'
      hit_count INTEGER DEFAULT 0
  );
  CREATE INDEX idx_expires_at ON cache_entries(expires_at);
  ```
- **Use Cases:**
  - AI parsing results (30-day TTL, NFR-I10)
  - Image metadata (permanent)
  - Infrequently accessed metadata
- **Capacity:** Limited by disk space (negligible)

**Tier 3: File System (Permanent Storage)**
- **Location:** `./data/images/` directory
- **Use Cases:**
  - Downloaded poster images
  - Background images
  - Thumbnail caches
- **Management:** Lazy loading, cleanup on media deletion

**Cache Key Design:**
```
{source}:{type}:{identifier}:{version}

Examples:
- tmdb:movie:12345:v1
- ai:filename:hash_abc123:v1
- douban:movie:1234567:v1
```

**Implementation Architecture:**

```go
type CacheManager struct {
    memoryCache  MemoryCache     // Tier 1: bigcache/ristretto
    dbCache      SQLiteCache     // Tier 2: SQLite
    fsCache      FileSystemCache // Tier 3: File system
}

func (cm *CacheManager) Get(key string) (interface{}, error) {
    // Try memory cache first
    if value, found := cm.memoryCache.Get(key); found {
        return value, nil
    }
    
    // Fallback to SQLite
    if value, found := cm.dbCache.Get(key); found {
        // Promote to memory cache
        cm.memoryCache.Set(key, value, ttl)
        return value, nil
    }
    
    return nil, ErrCacheMiss
}
```

**Cache Patterns:**

1. **Cache-Aside (Lazy Loading):**
   ```go
   func GetMovieMetadata(id string) (*Metadata, error) {
       key := fmt.Sprintf("tmdb:movie:%s:v1", id)
       
       // Try cache first
       if cached, err := cacheManager.Get(key); err == nil {
           return cached.(*Metadata), nil
       }
       
       // Cache miss - fetch from API
       metadata, err := tmdbClient.GetMovie(id)
       if err != nil {
           return nil, err
       }
       
       // Store in cache
       cacheManager.Set(key, metadata, 24*time.Hour)
       return metadata, nil
   }
   ```

2. **Write-Through (AI Parsing):**
   ```go
   func ParseFilename(filename string) (*ParseResult, error) {
       result, err := aiProvider.Parse(filename)
       if err != nil {
           return nil, err
       }
       
       // Immediately cache result (30 days)
       key := fmt.Sprintf("ai:filename:%s:v1", hash(filename))
       cacheManager.Set(key, result, 30*24*time.Hour)
       
       return result, nil
   }
   ```

**TTL Management:**
- **Background Cleanup Goroutine:** Runs every hour, removes expired entries from SQLite
- **Memory Cache:** Auto-eviction via LRU
- **Manual Invalidation:** API endpoint for cache clearing (admin feature)

**Monitoring:**
- Cache hit rate metrics (for performance dashboard)
- Cache size monitoring
- Eviction rate tracking

**Affects:**
- All external API integrations (TMDb, Douban, Wikipedia, AI providers)
- Performance optimization for large media libraries
- Cost optimization for AI API usage

**Alternatives Considered:**
- **Redis:** External dependency, over-engineering for single-user self-hosted scenario
- **SQLite Only:** Poor performance for hot data
- **Memory Only:** Loss of expensive AI results on restart

---

## 5. Background Task Processing: Lightweight Worker Pool

**Decision:** Implement lightweight background task processing using Go's native concurrency primitives (goroutines + channels)

**Implementation:** Worker Pool pattern with buffered channels

**Rationale:**
- **Zero Dependencies:** Pure Go implementation using goroutines and channels
- **Simplicity:** Straightforward implementation and debugging
- **Go-Native:** Leverages Go's excellent concurrency model
- **Sufficient for Requirements:** Handles AI parsing (10s non-blocking), retries, and scheduled tasks
- **Resource Efficient:** Configurable worker count prevents resource exhaustion

**Architecture:**

```go
type TaskQueue struct {
    taskChan   chan Task           // Buffered channel (e.g., 100 capacity)
    workers    int                 // Configurable worker count (3-5)
    wg         sync.WaitGroup      // Graceful shutdown
    ctx        context.Context     // Cancellation support
    cancel     context.CancelFunc
}

type Task interface {
    Execute(ctx context.Context) error
    GetType() TaskType
    GetPriority() int
    ShouldRetry(err error) bool
    GetMaxRetries() int
}

type TaskType int
const (
    TaskAIParsing TaskType = iota  // High priority
    TaskMetadataRefresh             // Medium priority
    TaskBackup                      // Low priority, scheduled
)
```

**Worker Pool Implementation:**

```go
func (tq *TaskQueue) Start() {
    for i := 0; i < tq.workers; i++ {
        tq.wg.Add(1)
        go tq.worker(i)
    }
}

func (tq *TaskQueue) worker(id int) {
    defer tq.wg.Done()
    
    for {
        select {
        case task := <-tq.taskChan:
            if err := tq.executeWithRetry(task); err != nil {
                logger.Error("Task failed", "worker", id, "task", task.GetType(), "error", err)
            }
        case <-tq.ctx.Done():
            return
        }
    }
}

func (tq *TaskQueue) executeWithRetry(task Task) error {
    var err error
    maxRetries := task.GetMaxRetries()
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        err = task.Execute(tq.ctx)
        
        if err == nil {
            return nil // Success
        }
        
        if !task.ShouldRetry(err) {
            return err // Non-retryable error
        }
        
        if attempt < maxRetries {
            // Exponential backoff: 1s, 2s, 4s, 8s
            backoff := time.Duration(1<<uint(attempt)) * time.Second
            time.Sleep(backoff)
        }
    }
    
    return fmt.Errorf("task failed after %d retries: %w", maxRetries, err)
}
```

**Task Types:**

1. **AI Parsing Task** (High Priority)
   - Max retries: 3
   - Retry on: Network errors, timeouts
   - Non-blocking UI: Immediate task submission, progress tracking via WebSocket/polling

2. **Metadata Refresh Task** (Medium Priority)
   - Max retries: 5
   - Retry on: All errors except 404
   - Scheduled: Nightly for all media items

3. **Backup Task** (Low Priority)
   - Max retries: 2
   - Scheduled: Configurable (default: daily at 3 AM)
   - Retry on: Disk space errors (after cleanup)

**Configuration:**
```go
type TaskQueueConfig struct {
    WorkerCount    int           // Default: 3
    QueueCapacity  int           // Default: 100
    ShutdownTimeout time.Duration // Default: 30s
}
```

**Graceful Shutdown:**
```go
func (tq *TaskQueue) Stop() error {
    tq.cancel() // Signal all workers to stop
    
    done := make(chan struct{})
    go func() {
        tq.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-time.After(30 * time.Second):
        return errors.New("shutdown timeout: workers still processing")
    }
}
```

**Progress Tracking:**
- Frontend polls `/api/v1/tasks/{task_id}` for AI parsing progress
- Task metadata stored in memory (map[string]TaskStatus)
- Progress updates via structured logging

**Trade-offs Acknowledged:**
- ⚠️ **No Persistence:** In-flight tasks lost on restart (acceptable - can re-trigger)
- ⚠️ **Manual Retry Logic:** No built-in retry framework (simple to implement)
- ⚠️ **No Priority Queue:** FIFO processing (sufficient for current requirements)

**Future Enhancement Path:**
- If persistence needed: Add `background_tasks` SQLite table
- If complex scheduling needed: Integrate lightweight scheduler (e.g., `robfig/cron`)
- If distributed: Migrate to asynq or similar (requires Redis)

**Affects:**
- AI filename parsing (FR15, FR22)
- Automatic metadata refresh
- Scheduled backups (FR58, NFR-R7)
- Any long-running operations

**Alternatives Considered:**
- **asynq (Redis-based):** External dependency, over-engineering for single-user scenario
- **Worker Pool + SQLite Persistence:** Added complexity, deferred unless restart-resilience becomes critical

---

## 6. Error Handling & Logging: Structured Logging with Unified Error Types

**Decision:** Implement structured logging using Go's `slog` standard library and custom unified error types for consistent error handling

**Version:** 
- Go 1.21+ `log/slog` (standard library)
- Custom `AppError` type

**Rationale:**
- **Standard Library:** `slog` is part of Go 1.21+, zero external dependencies
- **Structured Logging:** JSON-formatted logs enable querying and analysis
- **Observability:** Facilitates debugging and monitoring in production
- **User Experience:** Unified error types ensure consistent, actionable error messages (NFR-U4, NFR-U5)
- **Security:** Built-in sensitive data filtering prevents API key leakage (NFR-S4)

**Architecture:**

### Unified Error Type

```go
type AppError struct {
    Code       string  // Error code (e.g., "TMDB_TIMEOUT", "AI_QUOTA_EXCEEDED")
    Message    string  // User-friendly message (Traditional Chinese)
    Details    string  // Technical details (for logging)
    Suggestion string  // Troubleshooting hint
    HTTPStatus int     // HTTP status code
    Err        error   // Original error (wrapped)
}

func (e *AppError) Error() string {
    return e.Message
}

func (e *AppError) Unwrap() error {
    return e.Err
}
```

**Error Code System:**

Error codes follow the pattern: `{SOURCE}_{ERROR_TYPE}`

- **TMDB_*** - TMDb API errors
  - `TMDB_TIMEOUT` - API request timeout
  - `TMDB_RATE_LIMIT` - Rate limit exceeded
  - `TMDB_NOT_FOUND` - Movie/TV show not found
  - `TMDB_AUTH_FAILED` - Invalid API key

- **AI_*** - AI provider errors
  - `AI_TIMEOUT` - AI parsing timeout (>10s)
  - `AI_QUOTA_EXCEEDED` - User's API quota exhausted
  - `AI_INVALID_RESPONSE` - Unparseable AI response
  - `AI_PROVIDER_ERROR` - Generic provider error

- **QBIT_*** - qBittorrent errors
  - `QBIT_CONNECTION_FAILED` - Cannot connect to qBittorrent
  - `QBIT_AUTH_FAILED` - Invalid credentials
  - `QBIT_TORRENT_NOT_FOUND` - Torrent not found

- **DB_*** - Database errors
  - `DB_CONNECTION_FAILED` - Database connection error
  - `DB_QUERY_FAILED` - Query execution error
  - `DB_CONSTRAINT_VIOLATION` - Constraint violation

- **AUTH_*** - Authentication errors
  - `AUTH_INVALID_CREDENTIALS` - Wrong password/PIN
  - `AUTH_TOKEN_EXPIRED` - JWT expired
  - `AUTH_TOKEN_INVALID` - Malformed JWT

**Example Error Construction:**

```go
func NewTMDbTimeoutError(err error) *AppError {
    return &AppError{
        Code:       "TMDB_TIMEOUT",
        Message:    "無法連線到 TMDb API，請稍後再試",
        Details:    fmt.Sprintf("TMDb API request timed out: %v", err),
        Suggestion: "檢查網路連線或稍後重試。如果問題持續，請確認 TMDb API 狀態。",
        HTTPStatus: http.StatusGatewayTimeout,
        Err:        err,
    }
}
```

### Structured Logging Configuration

```go
// Initialize logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: getLogLevel(), // From env: DEBUG, INFO, WARN, ERROR
    ReplaceAttr: sanitizeAttr, // Filter sensitive data
}))

slog.SetDefault(logger)
```

**Log Level Guidelines:**

- **ERROR:** Requires immediate attention
  - External API failures
  - Database errors
  - Authentication failures
  - Unrecoverable errors

- **WARN:** Potentially problematic but recoverable
  - Retry attempts succeeding
  - Fallback mechanisms activated
  - Deprecated API usage
  - Resource threshold warnings (e.g., 8,000 media items)

- **INFO:** Important business events
  - User login/logout
  - AI parsing completed
  - Media item added/deleted
  - Cache invalidation events

- **DEBUG:** Development and troubleshooting
  - API request/response details
  - Cache hit/miss
  - Database query execution
  - Worker task processing

**Sensitive Data Filtering:**

```go
func sanitizeAttr(groups []string, a slog.Attr) slog.Attr {
    // Filter API keys from URLs
    if a.Key == "url" {
        if urlStr, ok := a.Value.Any().(string); ok {
            a.Value = slog.StringValue(sanitizeURL(urlStr))
        }
    }
    
    // Remove sensitive fields entirely
    if a.Key == "api_key" || a.Key == "password" || a.Key == "token" {
        return slog.Attr{} // Omit attribute
    }
    
    return a
}

func sanitizeURL(urlStr string) string {
    u, err := url.Parse(urlStr)
    if err != nil {
        return "[INVALID_URL]"
    }
    
    // Remove query parameters containing keys
    q := u.Query()
    q.Del("api_key")
    q.Del("key")
    q.Del("token")
    u.RawQuery = q.Encode()
    
    return u.String()
}
```

**Usage Examples:**

```go
// Error logging with context
slog.Error("TMDb API request failed",
    "error_code", "TMDB_TIMEOUT",
    "movie_id", movieID,
    "retry_count", retryCount,
    "url", sanitizeURL(apiURL),
    "error", err,
)

// Info logging for business events
slog.Info("AI parsing completed",
    "filename", filename,
    "duration_ms", duration.Milliseconds(),
    "result", "success",
    "provider", "gemini",
)

// Debug logging (only in development)
slog.Debug("Cache hit",
    "cache_key", cacheKey,
    "cache_tier", "memory",
    "ttl_remaining", ttlRemaining,
)
```

### Frontend Error Handling

**TanStack Query Error Handling:**

```typescript
const { data, error, isError } = useQuery({
  queryKey: ['movie', movieId],
  queryFn: fetchMovie,
  onError: (error: AppError) => {
    // Display user-friendly toast
    toast.error(error.message, {
      description: error.suggestion,
    });
    
    // Log technical details
    console.error(`[${error.code}]`, error.details);
  },
});
```

**Global Error Boundary:**

```typescript
class ErrorBoundary extends React.Component {
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log to backend error tracking
    logger.error('React error boundary caught error', {
      error: error.message,
      componentStack: errorInfo.componentStack,
    });
    
    // Display fallback UI
    this.setState({ hasError: true });
  }
}
```

**401 Unauthorized Handling:**

```typescript
// TanStack Query global config
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      onError: (error) => {
        if (error.status === 401) {
          // Clear auth state and redirect to login
          authStore.logout();
          router.navigate('/login');
        }
      },
    },
  },
});
```

**Affects:**
- All backend services and API endpoints
- Frontend error display and user feedback
- Observability and debugging capabilities
- Security (prevents sensitive data leakage)

**Alternatives Considered:**
- **zap/zerolog:** High-performance alternatives, but `slog` is standard library (preferred)
- **Plain text logging:** Difficult to query and analyze in production
- **No unified error types:** Inconsistent error messages, poor user experience

---

## Decision Impact Analysis

**Implementation Sequence:**

The following order is recommended for implementing these architectural decisions:

1. **Error Handling & Logging** (Foundation)
   - Establish `AppError` types and error codes
   - Configure `slog` with sensitive data filtering
   - Required by all subsequent components

2. **Authentication** (Early Security)
   - Implement JWT middleware
   - Password hashing with bcrypt
   - Blocks API endpoint implementation

3. **Testing Infrastructure** (Quality Enabler)
   - Set up Vitest + React Testing Library
   - Configure Go testing + testify
   - Enables TDD for subsequent features

4. **Caching** (Performance Foundation)
   - Implement CacheManager with tiered strategy
   - Required for external API integrations
   - Critical for cost control

5. **Background Tasks** (Non-Blocking Operations)
   - Implement Worker Pool
   - Define task types (AI parsing, backups)
   - Enables asynchronous AI parsing

6. **Frontend Styling** (UI Development)
   - Configure Tailwind CSS
   - Establish design tokens
   - Enables component development

**Cross-Component Dependencies:**

```
Error Handling & Logging
    ↓
Authentication ←→ Caching
    ↓              ↓
Background Tasks ←┘
    ↓
All External Integrations (TMDb, AI, qBittorrent)
```

- **Error Handling** is foundational - all components depend on it
- **Authentication** and **Caching** are independent but both required for API layer
- **Background Tasks** depends on error handling and caching (for retry logic and result storage)
- **External Integrations** depend on all above (auth, caching, error handling, background processing)

**Critical Path:**

For MVP implementation to proceed, the following must be decided and implemented:

1. ✅ Error Handling & Logging
2. ✅ Authentication Strategy
3. ✅ Caching Implementation
4. ✅ Testing Infrastructure

These four decisions are blockers for core feature development.

**Deferred Decisions (Can be made during implementation):**

- **Component Library:** Headless UI vs shadcn/ui (can decide when building first complex component)
- **CI/CD Platform:** GitHub Actions vs alternatives (can set up when ready for automation)
- **Monitoring Tools:** Prometheus vs alternatives (post-1.0 concern)
- **E2E Framework:** Playwright setup (deferred to 1.0 phase)

---

## Decision Implementation Roadmap

With core architectural decisions finalized, the following sections analyze the **current codebase state** and provide a **consolidation & refactoring plan** to align with these decisions.
