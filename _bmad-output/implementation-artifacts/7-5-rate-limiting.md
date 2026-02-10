# Story 7.5: Rate Limiting

Status: ready-for-dev

## Story

As a **system administrator**,
I want **API rate limiting**,
So that **the system is protected from abuse**.

## Acceptance Criteria

1. **AC1: General API Rate Limiting**
   - Given API requests are made
   - When rate exceeds 100 requests/minute from same IP (NFR-S12)
   - Then subsequent requests return 429 Too Many Requests
   - And `Retry-After` header indicates seconds until next allowed request

2. **AC2: User-Friendly Rate Limit Response**
   - Given rate limit is hit
   - When the user sees the error in the UI
   - Then a friendly message explains: "請求頻率過高，請稍後再試"
   - And the response includes `retry_after` seconds in body

3. **AC3: Tiered Rate Limits**
   - Given different endpoint categories
   - When rate limits are applied
   - Then more restrictive limits for sensitive operations:
     - Login: 5 req/15 min per IP (Story 7.2, already implemented)
     - Write operations (POST/PUT/DELETE): 60 req/min per IP
     - Read operations (GET): 100 req/min per IP
   - And health check endpoint is exempt from rate limiting

4. **AC4: Rate Limit Headers**
   - Given any API response
   - When rate limiting is active
   - Then response includes headers:
     - `X-RateLimit-Limit`: max requests allowed
     - `X-RateLimit-Remaining`: requests remaining in window
     - `X-RateLimit-Reset`: UTC epoch when window resets

## Tasks / Subtasks

- [ ] Task 1: Create Rate Limiter Core (AC: 1, 3)
  - [ ] 1.1: Create `/apps/api/internal/middleware/rate_limiter.go`
  - [ ] 1.2: Implement token bucket or sliding window algorithm per IP
  - [ ] 1.3: Use `golang.org/x/time/rate` for core limiter (already in go.mod)
  - [ ] 1.4: Create `RateLimiterConfig` struct: `{ RequestsPerMinute, BurstSize }`
  - [ ] 1.5: Create IP-indexed limiter store with `sync.Map`
  - [ ] 1.6: Auto-cleanup stale entries (goroutine every 10 minutes)
  - [ ] 1.7: Write core rate limiter tests

- [ ] Task 2: Create Rate Limiter Middleware (AC: 1, 3, 4)
  - [ ] 2.1: Create `RateLimitMiddleware(config RateLimiterConfig) gin.HandlerFunc`
  - [ ] 2.2: Extract client IP from `c.ClientIP()` (handles X-Forwarded-For)
  - [ ] 2.3: Check rate limit for IP
  - [ ] 2.4: On limit exceeded: return 429 with `Retry-After` header
  - [ ] 2.5: Always set rate limit info headers (`X-RateLimit-*`)
  - [ ] 2.6: Error response: `{ "success": false, "error": { "code": "RATE_LIMITED", "message": "...", "retry_after": N } }`
  - [ ] 2.7: Write middleware tests

- [ ] Task 3: Apply Tiered Rate Limits (AC: 3)
  - [ ] 3.1: Define rate limit configs:
    - `ReadConfig`: 100 req/min, burst 10
    - `WriteConfig`: 60 req/min, burst 5
    - Login config: already in Story 7.2 (5 req/15 min)
  - [ ] 3.2: Update main.go route groups:
    - GET routes: `ReadConfig` rate limiter
    - POST/PUT/DELETE routes: `WriteConfig` rate limiter
    - `/health`: NO rate limiter
    - Public auth routes: separate rate limits (login already has its own)
  - [ ] 3.3: Approach: Apply at router group level with method-based config
  - [ ] 3.4: Write integration tests for tiered limits

- [ ] Task 4: Frontend - Rate Limit Handling (AC: 2)
  - [ ] 4.1: Update API service base to detect 429 responses
  - [ ] 4.2: Parse `Retry-After` header or `retry_after` from response body
  - [ ] 4.3: Show toast notification: "請求頻率過高，請稍後再試"
  - [ ] 4.4: Auto-retry after `Retry-After` seconds for TanStack Query (optional, up to 1 retry)
  - [ ] 4.5: Disable rapid-fire button clicks during rate limit cooldown

- [ ] Task 5: Rate Limit Configuration (AC: 1, 3)
  - [ ] 5.1: Make rate limits configurable via environment variables:
    - `VIDO_RATE_LIMIT_READ`: default 100
    - `VIDO_RATE_LIMIT_WRITE`: default 60
  - [ ] 5.2: Add to config.go with proper defaults
  - [ ] 5.3: Log rate limit config on startup with slog
  - [ ] 5.4: Write config tests

- [ ] Task 6: Wire Up in Main (AC: all)
  - [ ] 6.1: Update route registration with rate limit middleware
  - [ ] 6.2: Exempt health check from all rate limiting
  - [ ] 6.3: Apply correct tier to each route group
  - [ ] 6.4: Ensure login rate limiter (Story 7.2) and general rate limiter can coexist

## Dev Notes

### Architecture Compliance

- **NFR-S12:** 100 requests/minute per IP (general)
- **NFR-S13:** 5 failed auth attempts per 15 min (login - already in Story 7.2)
- **Algorithm:** Token bucket using `golang.org/x/time/rate` (already a dependency)
- **No external dependencies:** In-memory rate limiting (suitable for single-instance deployment)

### Token Bucket Implementation

```go
import "golang.org/x/time/rate"

type IPRateLimiter struct {
    limiters sync.Map // map[string]*rate.Limiter
    rate     rate.Limit
    burst    int
}

func (rl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    limiter, exists := rl.limiters.Load(ip)
    if !exists {
        l := rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters.Store(ip, l)
        return l
    }
    return limiter.(*rate.Limiter)
}
```

### Route Group Structure After All Auth Stories

```go
router := gin.Default()
router.Use(corsMiddleware)

// Health - no auth, no rate limit
router.GET("/health", healthHandler)

// Public auth routes - specific rate limits
publicAuth := router.Group("/api/v1/auth")
publicAuth.Use(readRateLimiter)
{
    publicAuth.GET("/status", authHandler.GetStatus)
    publicAuth.POST("/setup", writeRateLimiter, authHandler.Setup)
    publicAuth.POST("/login", loginRateLimiter, authHandler.Login)
}

// Protected routes
apiV1 := router.Group("/api/v1")
apiV1.Use(authMiddleware)
{
    // Read routes - 100 req/min
    apiV1Read := apiV1.Group("")
    apiV1Read.Use(readRateLimiter)
    // ... GET routes

    // Write routes - 60 req/min
    apiV1Write := apiV1.Group("")
    apiV1Write.Use(writeRateLimiter)
    // ... POST/PUT/DELETE routes
}
```

**Note:** The exact route grouping may need adjustment based on Gin's middleware stacking. An alternative is a single rate limiter middleware that checks the HTTP method to determine the limit tier.

### Existing Code to Reuse

- **`golang.org/x/time/rate`:** Already in go.mod (used by TMDb client for external API rate limiting). Same package, different use case.
- **Login rate limiter from Story 7.2:** Coexists with general rate limiter (different scope)
- **Response helpers:** `ErrorResponse()` for 429 responses
- **Config pattern:** Add env vars following existing pattern in `config.go`

### Error Codes

- `RATE_LIMITED` - General API rate limit exceeded
- `AUTH_RATE_LIMITED` - Login rate limit (from Story 7.2)

### Important: IP Detection

```go
// Gin's ClientIP() handles X-Forwarded-For for reverse proxy setups
ip := c.ClientIP()
```

This is critical for Docker/reverse proxy deployments where all traffic comes from proxy IP. Ensure `gin.SetTrustedProxies()` is configured properly.

### Project Structure Notes

- Rate limiter middleware in `/apps/api/internal/middleware/rate_limiter.go`
- Tests: `rate_limiter_test.go` (same directory)
- Config additions: `config.go`
- No new services or repositories needed (middleware-only feature)

### References

- [Source: architecture.md#Security Requirements Satisfaction] - Rate limiting middleware requirements
- [Source: architecture.md#NFR-S12] - 100 req/min per IP
- [Source: architecture.md#NFR-S13] - Auth rate limiting (5 per 15 min)
- [Source: epics.md#Story 7.5] - AC and technical notes
- [Source: project-context.md#Rule 3] - API response format
- [Source: project-context.md#Rule 7] - Error code format

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
