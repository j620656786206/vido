# Story 7.4: API Authentication

Status: ready-for-dev

## Story

As a **developer integrating with Vido**,
I want **API endpoints to be authenticated**,
So that **only authorized requests can access data**.

## Acceptance Criteria

1. **AC1: Unauthenticated Request Rejection**
   - Given an API request is made
   - When no authentication token is provided (no cookie, no header)
   - Then the request is rejected with 401 Unauthorized
   - And response body: `{ "success": false, "error": { "code": "AUTH_REQUIRED" } }`

2. **AC2: Session Cookie Authentication**
   - Given a valid session cookie exists (from browser login)
   - When making API requests from the browser
   - Then the session cookie authenticates the request
   - And CSRF protection is enforced (SameSite + Origin check)

3. **AC3: API Token Authentication**
   - Given an API token is generated
   - When used in `Authorization: Bearer <token>` header
   - Then the request is authenticated
   - And the token has configurable expiration (default 30 days)

4. **AC4: API Token Management**
   - Given the user is authenticated
   - When they access Settings > API Tokens
   - Then they can generate new API tokens with custom names
   - And view existing tokens (masked)
   - And revoke any token

5. **AC5: Dual Auth Support**
   - Given both cookie and Bearer token are valid auth methods
   - When the auth middleware processes a request
   - Then it checks cookie first, then Bearer header
   - And both methods use the same JWT validation

## Tasks / Subtasks

- [ ] Task 1: Extend Auth Middleware for Bearer Token (AC: 1, 2, 3, 5)
  - [ ] 1.1: Update `/apps/api/internal/middleware/auth_middleware.go`
  - [ ] 1.2: Check order: (1) `vido_token` cookie → (2) `Authorization: Bearer` header
  - [ ] 1.3: Both use same `ValidateToken()` function from auth package
  - [ ] 1.4: Set `auth_method` in Gin context: "cookie" or "bearer"
  - [ ] 1.5: Write tests for both auth methods and precedence

- [ ] Task 2: Create API Tokens Table & Migration (AC: 3, 4)
  - [ ] 2.1: Create migration `XXX_create_api_tokens_table.sql`
  - [ ] 2.2: Schema: `id TEXT PRIMARY KEY, name TEXT NOT NULL, token_hash TEXT NOT NULL, last_four TEXT NOT NULL, expires_at TIMESTAMP, created_at TIMESTAMP, revoked_at TIMESTAMP`
  - [ ] 2.3: Index: `idx_api_tokens_token_hash` for fast lookup
  - [ ] 2.4: Store hashed token (SHA-256), show only last 4 chars

- [ ] Task 3: Create API Token Repository (AC: 3, 4)
  - [ ] 3.1: Create `/apps/api/internal/repository/api_token_repository.go`
  - [ ] 3.2: Interface: `APITokenRepositoryInterface`
  - [ ] 3.3: Methods: `Create`, `GetByTokenHash`, `ListActive`, `Revoke`, `CleanupExpired`
  - [ ] 3.4: Write repository tests

- [ ] Task 4: Create API Token Service (AC: 3, 4)
  - [ ] 4.1: Create `/apps/api/internal/services/api_token_service.go`
  - [ ] 4.2: Interface: `APITokenServiceInterface`
  - [ ] 4.3: `GenerateToken(name string, expiresInDays int) (plainToken string, error)` - returns plain token ONCE
  - [ ] 4.4: `ValidateToken(plainToken string) (bool, error)` - hash and lookup
  - [ ] 4.5: `ListTokens() ([]APITokenSummary, error)` - name, last_four, created_at, expires_at
  - [ ] 4.6: `RevokeToken(tokenID string) error`
  - [ ] 4.7: Write service tests (coverage >= 80%)

- [ ] Task 5: Create API Token Handler (AC: 4)
  - [ ] 5.1: Create `/apps/api/internal/handlers/api_token_handler.go`
  - [ ] 5.2: `POST /api/v1/auth/tokens` - generate new token (return plain text ONCE)
  - [ ] 5.3: `GET /api/v1/auth/tokens` - list active tokens (masked)
  - [ ] 5.4: `DELETE /api/v1/auth/tokens/:id` - revoke token
  - [ ] 5.5: All token endpoints require authentication (protected routes)
  - [ ] 5.6: Write handler tests (coverage >= 70%)

- [ ] Task 6: Integrate API Token Validation in Middleware (AC: 3, 5)
  - [ ] 6.1: When Bearer token is not a valid JWT (not from login), check API token table
  - [ ] 6.2: Auth check order: (1) Cookie JWT → (2) Bearer JWT → (3) Bearer API token hash
  - [ ] 6.3: API tokens: hash the bearer value with SHA-256, look up in `api_tokens` table
  - [ ] 6.4: Check `revoked_at IS NULL` and `expires_at > NOW()`
  - [ ] 6.5: Write integration tests for all 3 auth methods

- [ ] Task 7: Frontend - API Token Management UI (AC: 4)
  - [ ] 7.1: Create `/apps/web/src/components/settings/APITokenManager.tsx`
  - [ ] 7.2: List existing tokens with name, last 4 chars, expiry, created date
  - [ ] 7.3: "Generate New Token" button → dialog with name input + expiry selector
  - [ ] 7.4: Show generated token ONCE with copy button and warning
  - [ ] 7.5: Revoke button with confirmation dialog
  - [ ] 7.6: Use TanStack Query for token list
  - [ ] 7.7: Use TanStack mutations for create/revoke

- [ ] Task 8: Wire Up in Main (AC: all)
  - [ ] 8.1: Register API token routes (protected group)
  - [ ] 8.2: Initialize API token repository, service, handler
  - [ ] 8.3: Pass API token service to auth middleware for Bearer validation

## Dev Notes

### Architecture Compliance

- **Dual auth:** Cookie (browser) + Bearer token (API clients) - both use JWT validation first, then API token hash lookup as fallback
- **API tokens are NOT JWTs:** They are random strings hashed with SHA-256 for storage. This distinguishes them from session JWTs.
- **Token display:** Plain text shown exactly ONCE at generation. Only `last_four` stored for display.
- **Interface location:** Rule 11 - service interfaces in services package

### Token Generation Pattern

```go
// Generate a random 32-byte API token
tokenBytes := make([]byte, 32)
crypto/rand.Read(tokenBytes)
plainToken := base64.URLEncoding.EncodeToString(tokenBytes) // ~43 chars
tokenHash := sha256.Sum256([]byte(plainToken))
lastFour := plainToken[len(plainToken)-4:]
```

### Auth Middleware Flow

```
Request → Extract Auth
  → Cookie "vido_token"? → Validate JWT → ✅ Proceed (auth_method=cookie)
  → Header "Authorization: Bearer X"?
    → Is valid JWT? → ✅ Proceed (auth_method=bearer_jwt)
    → Hash X → Lookup in api_tokens table → ✅ Proceed (auth_method=bearer_api_token)
  → None found → 401 AUTH_REQUIRED
```

### Existing Code to Reuse

- **Auth middleware from Story 7.3:** Extend, don't rewrite
- **JWT helpers from Story 7.2:** `auth.ValidateToken()`
- **Repository pattern:** Same as existing repositories in `/apps/api/internal/repository/`
- **Crypto package:** DO NOT use `/apps/api/internal/crypto/` for token hashing (that's AES-256 for secrets). Use standard `crypto/sha256`.

### Error Codes

- `AUTH_REQUIRED` - No authentication provided
- `AUTH_TOKEN_INVALID` - Invalid cookie JWT or Bearer token
- `AUTH_TOKEN_EXPIRED` - JWT or API token expired
- `AUTH_TOKEN_REVOKED` - API token has been revoked

### Project Structure Notes

- API token routes under `/api/v1/auth/tokens` (protected)
- New files: `api_token_repository.go`, `api_token_service.go`, `api_token_handler.go`
- Migration for `api_tokens` table
- All in `/apps/api` (Rule 1)

### References

- [Source: architecture.md#Authentication Strategy: JWT (Stateless)] - JWT validation
- [Source: architecture.md#Frontend Integration] - TanStack Query auth mutation
- [Source: epics.md#Story 7.4] - AC and technical notes
- [Source: project-context.md#Rule 3] - API response format
- [Source: project-context.md#Rule 4] - Handler → Service → Repository layering
- [Source: project-context.md#Rule 6] - Naming conventions (snake_case DB)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
