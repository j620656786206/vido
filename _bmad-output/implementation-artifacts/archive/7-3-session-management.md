# Story 7.3: Session Management

Status: ready-for-dev

## Story

As an **authenticated user**,
I want **my session to be secure and persistent**,
So that **I don't have to log in repeatedly but remain protected**.

## Acceptance Criteria

1. **AC1: JWT Token Issuance**
   - Given the user logs in successfully
   - When a session is created
   - Then a cryptographically-signed JWT token is issued (NFR-S10)
   - And stored in httpOnly cookie with Secure and SameSite flags

2. **AC2: Session Validation**
   - Given a session is active
   - When the user makes requests
   - Then the session token is validated on every API call
   - And invalid/expired tokens return 401 Unauthorized

3. **AC3: Session Expiration**
   - Given the session expires (after 24 hours per architecture spec)
   - When the user tries to access Vido
   - Then they are redirected to login
   - And a message indicates session expiration: "登入已過期，請重新登入"

4. **AC4: Logout**
   - Given the user clicks "Logout"
   - When logout is processed
   - Then the httpOnly cookie is cleared (MaxAge=0)
   - And the user is redirected to login page
   - And client-side auth state is reset

5. **AC5: Auth Middleware Protection**
   - Given all API endpoints except public ones
   - When a request has no valid token
   - Then it returns 401 with `AUTH_TOKEN_INVALID` or `AUTH_TOKEN_EXPIRED`
   - And the response uses standard error format

## Tasks / Subtasks

- [ ] Task 1: Create Auth Middleware (AC: 2, 5)
  - [ ] 1.1: Create `/apps/api/internal/middleware/auth_middleware.go`
  - [ ] 1.2: Extract JWT from `vido_token` httpOnly cookie
  - [ ] 1.3: Validate token signature with `JWT_SECRET`
  - [ ] 1.4: Check token expiration
  - [ ] 1.5: Set `user_id` in Gin context: `c.Set("user_id", claims.UserID)`
  - [ ] 1.6: Return 401 with proper error code on failure
  - [ ] 1.7: Log auth failures with slog: `slog.Warn("Auth failed", "reason", reason, "ip", ip)`
  - [ ] 1.8: Write middleware tests with expired/invalid/missing token scenarios

- [ ] Task 2: Define Public vs Protected Routes (AC: 5)
  - [ ] 2.1: Update `/apps/api/cmd/api/main.go` route registration
  - [ ] 2.2: Public routes (NO auth middleware):
    - `GET /health`
    - `GET /api/v1/auth/status`
    - `POST /api/v1/auth/setup`
    - `POST /api/v1/auth/login`
  - [ ] 2.3: Protected routes (WITH auth middleware):
    - ALL other `/api/v1/*` routes
    - `POST /api/v1/auth/logout`
    - `PUT /api/v1/auth/password`
  - [ ] 2.4: Apply middleware at router group level: `apiV1.Use(middleware.AuthMiddleware())`
  - [ ] 2.5: Create separate group for public auth routes

- [ ] Task 3: Implement Token Refresh Logic (AC: 1, 3)
  - [ ] 3.1: Add middleware to check token age
  - [ ] 3.2: If token is older than 12h but still valid (within 24h), issue new token
  - [ ] 3.3: Set new token in httpOnly cookie automatically (sliding window)
  - [ ] 3.4: This prevents forced logout during active use
  - [ ] 3.5: Write refresh tests

- [ ] Task 4: Frontend - Session Expiry Handling (AC: 3)
  - [ ] 4.1: Update TanStack Query global error handler for 401
  - [ ] 4.2: On 401 response: clear auth store, redirect to `/login`
  - [ ] 4.3: Show toast: "登入已過期，請重新登入"
  - [ ] 4.4: Preserve current URL as `returnUrl` for post-login redirect
  - [ ] 4.5: Prevent multiple 401 redirects (debounce/flag)

- [ ] Task 5: Frontend - Logout Flow (AC: 4)
  - [ ] 5.1: Add logout button to app navigation/settings
  - [ ] 5.2: Call `POST /api/v1/auth/logout`
  - [ ] 5.3: Clear Zustand auth store state
  - [ ] 5.4: Invalidate all TanStack Query caches on logout
  - [ ] 5.5: Redirect to `/login`

- [ ] Task 6: CSRF Protection (AC: 2)
  - [ ] 6.1: SameSite=Lax cookie already prevents most CSRF
  - [ ] 6.2: For extra security on state-changing endpoints, verify `Origin` header matches expected host
  - [ ] 6.3: Add `Origin` check middleware for POST/PUT/DELETE requests
  - [ ] 6.4: Skip CSRF check for same-origin requests
  - [ ] 6.5: Write CSRF protection tests

## Dev Notes

### Architecture Compliance

- **JWT Decision #3:** Stateless JWT with HS256 signing
- **Token lifetime:** 24 hours with sliding window refresh at 12h
- **Cookie name:** `vido_token`
- **Claims:** `{ user_id, exp, iat }` - prepare `role` claim for Growth phase multi-user
- **Public endpoints:** Must be explicitly defined; everything else requires auth

### Critical: Route Organization Pattern

```go
// main.go route setup
router := gin.Default()
router.Use(corsMiddleware)

// Public routes - NO auth required
router.GET("/health", healthHandler)
public := router.Group("/api/v1/auth")
{
    public.GET("/status", authHandler.GetStatus)
    public.POST("/setup", authHandler.Setup)
    public.POST("/login", loginRateLimiter, authHandler.Login)
}

// Protected routes - auth middleware required
apiV1 := router.Group("/api/v1")
apiV1.Use(middleware.AuthMiddleware(jwtSecret))
{
    apiV1.POST("/auth/logout", authHandler.Logout)
    apiV1.PUT("/auth/password", authHandler.ChangePassword)
    // ... all other existing routes
}
```

### Existing Code Impact

- **ALL existing routes become protected:** Every `/api/v1/*` route currently has no auth. This story wraps them with auth middleware.
- **Existing tests may need update:** Tests that call protected endpoints will need to include a valid JWT cookie.
- **Test helper needed:** Create `testutil.GenerateTestToken()` for use in handler tests.

### Dependencies

- `github.com/golang-jwt/jwt/v5` - added in Story 7.2
- No additional dependencies needed

### Error Codes

- `AUTH_TOKEN_EXPIRED` - JWT past expiration
- `AUTH_TOKEN_INVALID` - Malformed or bad signature JWT
- `AUTH_REQUIRED` - No token provided at all

### CSRF Note

SameSite=Lax + httpOnly cookies provide strong CSRF protection. The Origin header check is an additional defense layer. Do NOT implement CSRF tokens (adds complexity for single-user self-hosted app).

### Project Structure Notes

- Middleware in `/apps/api/internal/middleware/` (new directory)
- Tests co-located: `auth_middleware_test.go` next to `auth_middleware.go`
- DO NOT create a separate `auth` package - middleware goes in `middleware` package, JWT helpers in `auth` package

### References

- [Source: architecture.md#Backend JWT Middleware (Gin)] - Exact middleware code pattern
- [Source: architecture.md#Authentication Strategy: JWT (Stateless)] - Token lifetime, signing
- [Source: architecture.md#Logout Mechanism] - Cookie clearing, state cleanup
- [Source: architecture.md#Multi-User Preparation] - Future `role` claim
- [Source: architecture.md#401 Unauthorized Handling] - Frontend 401 error handling
- [Source: epics.md#Story 7.3] - AC and technical notes
- [Source: project-context.md#Rule 4] - Handler → Service → Repository layering

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
