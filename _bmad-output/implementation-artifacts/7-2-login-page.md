# Story 7.2: Login Page

Status: ready-for-dev

## Story

As a **returning user**,
I want to **log in with my password or PIN**,
So that **I can access my media library**.

## Acceptance Criteria

1. **AC1: Login Redirect**
   - Given the user is not authenticated
   - When accessing any Vido page
   - Then they are redirected to the login page
   - And the intended destination URL is preserved for post-login redirect

2. **AC2: Successful Login**
   - Given the login page is displayed
   - When the user enters correct credentials
   - Then they are authenticated and a JWT token is issued
   - And redirected to their intended destination (or home `/`)

3. **AC3: Failed Login**
   - Given incorrect credentials are entered
   - When login fails
   - Then error message: "密碼錯誤" (Invalid password)
   - And failed attempt is logged with slog (NFR-S13)
   - And the password field is cleared

4. **AC4: Login Rate Limiting**
   - Given 5 failed attempts in 15 minutes from same IP
   - When the limit is reached
   - Then login is temporarily blocked
   - And message: "嘗試次數過多，請在 X 分鐘後再試" (Too many attempts)
   - And the lockout duration is shown with countdown

## Tasks / Subtasks

- [ ] Task 1: Create Login Handler (AC: 2, 3)
  - [ ] 1.1: Add `POST /api/v1/auth/login` to auth handler
  - [ ] 1.2: Accept `{ "password": "..." }` body (no username - single-user)
  - [ ] 1.3: Validate password against bcrypt hash via AuthService
  - [ ] 1.4: On success: generate JWT token, set httpOnly cookie, return success
  - [ ] 1.5: On failure: return `AUTH_INVALID_CREDENTIALS` error, log attempt with slog
  - [ ] 1.6: Add `POST /api/v1/auth/logout` - clear httpOnly cookie
  - [ ] 1.7: Write handler tests

- [ ] Task 2: Implement JWT Token Generation (AC: 2)
  - [ ] 2.1: Add `golang-jwt/jwt` v5.x dependency
  - [ ] 2.2: Create `/apps/api/internal/auth/jwt.go` - JWT helper functions
  - [ ] 2.3: `GenerateToken(userID string) (string, error)` - HS256 signing
  - [ ] 2.4: `ValidateToken(tokenString string) (*Claims, error)` - validate and parse
  - [ ] 2.5: Claims: `user_id`, `exp` (24h), `iat`
  - [ ] 2.6: JWT secret from env var `JWT_SECRET` (minimum 32 bytes, auto-generate if missing)
  - [ ] 2.7: Set token in httpOnly, Secure (when HTTPS), SameSite=Lax cookie
  - [ ] 2.8: Write JWT tests

- [ ] Task 3: Implement Login Rate Limiting (AC: 4)
  - [ ] 3.1: Create `/apps/api/internal/middleware/login_rate_limiter.go`
  - [ ] 3.2: Track failed attempts per IP address (in-memory map with mutex)
  - [ ] 3.3: Rule: 5 attempts per 15-minute window (NFR-S13)
  - [ ] 3.4: Return 429 with `Retry-After` header when limit exceeded
  - [ ] 3.5: Auto-cleanup expired entries with periodic goroutine
  - [ ] 3.6: Apply middleware ONLY to `/api/v1/auth/login`
  - [ ] 3.7: Write rate limiter tests

- [ ] Task 4: Update Auth Service (AC: 2, 3)
  - [ ] 4.1: Add `Login(ctx, password string) (string, error)` to `AuthServiceInterface`
  - [ ] 4.2: Validate password with bcrypt, generate JWT on success
  - [ ] 4.3: Add `Logout(ctx)` method (cookie clearing handled in handler)
  - [ ] 4.4: Log failed login attempts with slog: `slog.Warn("Failed login attempt", "ip", ip)`
  - [ ] 4.5: Write login service tests

- [ ] Task 5: Frontend - Login Page (AC: 1, 2, 3, 4)
  - [ ] 5.1: Create `/apps/web/src/routes/login.tsx` (TanStack Router route)
  - [ ] 5.2: Simple login form: password input + submit button
  - [ ] 5.3: Auto-detect auth type (password vs PIN) from `/api/v1/auth/status`
  - [ ] 5.4: Show PIN pad for PIN auth type, text input for password
  - [ ] 5.5: Error display with zh-TW messages
  - [ ] 5.6: Loading state on submit button
  - [ ] 5.7: Rate limit feedback: show countdown timer when locked out
  - [ ] 5.8: Style with Tailwind CSS, dark theme consistent with app

- [ ] Task 6: Frontend - Auth State Management (AC: 1, 2)
  - [ ] 6.1: Create `/apps/web/src/stores/authStore.ts` (Zustand - UI state only)
  - [ ] 6.2: State: `isAuthenticated`, `isSetupComplete`, `authType`
  - [ ] 6.3: Actions: `login()`, `logout()`, `checkAuth()`
  - [ ] 6.4: TanStack Query: `useAuthStatus()` hook for `/api/v1/auth/status`
  - [ ] 6.5: TanStack Query: `useLoginMutation()` for login flow

- [ ] Task 7: Frontend - Route Protection (AC: 1)
  - [ ] 7.1: Create `/apps/web/src/components/auth/AuthGuard.tsx`
  - [ ] 7.2: Check auth status on app load
  - [ ] 7.3: If not setup → redirect to setup wizard security step
  - [ ] 7.4: If not authenticated → redirect to `/login` with `returnUrl` param
  - [ ] 7.5: If authenticated → render children
  - [ ] 7.6: Handle 401 responses globally (TanStack Query `onError`)

- [ ] Task 8: Wire Up Routes (AC: 1, 2)
  - [ ] 8.1: Register login/logout routes in main.go
  - [ ] 8.2: `POST /api/v1/auth/login` - PUBLIC with rate limiter middleware
  - [ ] 8.3: `POST /api/v1/auth/logout` - requires auth
  - [ ] 8.4: Apply login rate limiter middleware to login route only

## Dev Notes

### Architecture Compliance

- **JWT Strategy:** Architecture Decision #3 - Stateless JWT
- **Signing:** HS256 (HMAC-SHA256) per architecture spec
- **Token expiration:** 24 hours
- **Cookie:** httpOnly, Secure (HTTPS), SameSite=Lax
- **JWT Secret:** Environment variable `JWT_SECRET`, minimum 32 bytes
- **Single-user:** No username field in login form - just password/PIN

### Dependencies to Add

```bash
go get github.com/golang-jwt/jwt/v5
```

### Critical: Cookie Configuration

```go
http.Cookie{
    Name:     "vido_token",
    Value:    tokenString,
    Path:     "/",
    HttpOnly: true,
    Secure:   isHTTPS, // Check X-Forwarded-Proto or config
    SameSite: http.SameSiteLaxMode,
    MaxAge:   86400, // 24 hours
}
```

### Existing Code to Reuse

- **Auth service from Story 7.1:** Extend `AuthServiceInterface` with `Login` method
- **Response helpers:** `/apps/api/internal/handlers/response.go`
- **Config pattern:** `/apps/api/internal/config/config.go` - add `JWT_SECRET` env var
- **Rate limiting pattern:** `golang.org/x/time/rate` already in go.mod (used by TMDb client), but login rate limiter needs IP-based tracking (use custom implementation with sync.Map)

### Frontend Auth Flow

```
App Load → GET /api/v1/auth/status
  → { setup_complete: false } → redirect to /setup
  → { setup_complete: true, authenticated: false } → redirect to /login
  → { setup_complete: true, authenticated: true } → render app
```

### Error Codes

- `AUTH_INVALID_CREDENTIALS` - Wrong password/PIN
- `AUTH_RATE_LIMITED` - Too many failed attempts
- `AUTH_SETUP_REQUIRED` - Setup not yet complete

### Project Structure Notes

- Login route: `/apps/web/src/routes/login.tsx` (TanStack Router file-based routing)
- Auth store: Zustand for `isAuthenticated` state (UI state only - Rule 5)
- Auth status query: TanStack Query for server state
- DO NOT store JWT in localStorage or React state - httpOnly cookie only
- Middleware directory: Create `/apps/api/internal/middleware/` (doesn't exist yet)

### References

- [Source: architecture.md#Authentication Strategy: JWT (Stateless)] - Full JWT implementation spec
- [Source: architecture.md#Login Flow] - bcrypt → JWT → httpOnly cookie flow
- [Source: architecture.md#401 Unauthorized Handling] - Frontend error handling
- [Source: epics.md#Story 7.2] - AC and technical notes
- [Source: project-context.md#Rule 3] - API response format
- [Source: project-context.md#Rule 5] - TanStack Query for server state

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
