# Story 7.1: Password/PIN Setup

Status: ready-for-dev

## Story

As a **first-time user**,
I want to **set up a password or PIN for Vido**,
So that **my media library is protected from unauthorized access**.

## Acceptance Criteria

1. **AC1: Password/PIN Setup During Wizard**
   - Given the user completes the setup wizard
   - When reaching the security step
   - Then they must set a password or PIN
   - And minimum requirements: 6+ characters (password) or 4+ digits (PIN)

2. **AC2: Secure Password Storage**
   - Given a password is set
   - When stored in the database
   - Then it is hashed using bcrypt (cost factor 12)
   - And the plaintext is never stored or logged

3. **AC3: Password Change**
   - Given the user wants to change their password
   - When accessing Settings > Security
   - Then they must enter current password first
   - And can then set a new password meeting minimum requirements

4. **AC4: Password Strength Indicator**
   - Given the user is entering a new password
   - When typing in the password field
   - Then a visual strength indicator shows weak/medium/strong
   - And minimum requirements are validated in real-time

## Tasks / Subtasks

- [ ] Task 1: Create Users Database Table & Migration (AC: 2)
  - [ ] 1.1: Create migration `XXX_create_users_table.sql` in `/apps/api/migrations/`
  - [ ] 1.2: Schema: `id TEXT PRIMARY KEY, password_hash TEXT NOT NULL, auth_type TEXT NOT NULL DEFAULT 'password', created_at TIMESTAMP, updated_at TIMESTAMP`
  - [ ] 1.3: Seed single-user row on first setup (single-user system, no username)
  - [ ] 1.4: Add `setup_completed BOOLEAN DEFAULT FALSE` to `settings` or `users` table

- [ ] Task 2: Create Auth Repository (AC: 2, 3)
  - [ ] 2.1: Create `/apps/api/internal/repository/auth_repository.go`
  - [ ] 2.2: Interface: `AuthRepositoryInterface` with `GetUser`, `UpdatePassword`, `IsSetupComplete`
  - [ ] 2.3: Implement `GetUser(ctx) (*models.User, error)` - single-user, no ID needed
  - [ ] 2.4: Implement `UpdatePassword(ctx, passwordHash string) error`
  - [ ] 2.5: Implement `IsSetupComplete(ctx) (bool, error)`
  - [ ] 2.6: Write repository tests with testify

- [ ] Task 3: Create Auth Service (AC: 1, 2, 3)
  - [ ] 3.1: Create `/apps/api/internal/services/auth_service.go`
  - [ ] 3.2: Interface: `AuthServiceInterface` with `SetupPassword`, `ChangePassword`, `ValidatePassword`, `IsSetupComplete`
  - [ ] 3.3: Implement `SetupPassword(ctx, password string, authType string) error` - hash with bcrypt cost 12
  - [ ] 3.4: Implement `ChangePassword(ctx, currentPassword, newPassword string) error` - verify current first
  - [ ] 3.5: Implement `ValidatePassword(ctx, password string) (bool, error)` - bcrypt compare
  - [ ] 3.6: Implement password validation rules: 6+ chars (password), 4+ digits (PIN)
  - [ ] 3.7: Write service tests (coverage >= 80%)

- [ ] Task 4: Create Auth Handler (AC: 1, 3)
  - [ ] 4.1: Create `/apps/api/internal/handlers/auth_handler.go`
  - [ ] 4.2: `POST /api/v1/auth/setup` - initial password/PIN setup
  - [ ] 4.3: `PUT /api/v1/auth/password` - change password (requires current password)
  - [ ] 4.4: `GET /api/v1/auth/status` - check if setup is complete (public endpoint)
  - [ ] 4.5: Use standard `ApiResponse` wrapper format
  - [ ] 4.6: Error codes: `AUTH_INVALID_CREDENTIALS`, `AUTH_WEAK_PASSWORD`, `VALIDATION_REQUIRED_FIELD`
  - [ ] 4.7: Write handler tests (coverage >= 70%)

- [ ] Task 5: Create User Model (AC: 2)
  - [ ] 5.1: Create `/apps/api/internal/models/user.go`
  - [ ] 5.2: Define `User` struct with `ID`, `PasswordHash`, `AuthType`, `CreatedAt`, `UpdatedAt`
  - [ ] 5.3: Define `SetupRequest` and `ChangePasswordRequest` DTOs
  - [ ] 5.4: Add JSON tags with snake_case

- [ ] Task 6: Wire Up in Main (AC: 1, 3)
  - [ ] 6.1: Register auth routes in `/apps/api/cmd/api/main.go`
  - [ ] 6.2: Initialize auth repository, service, handler
  - [ ] 6.3: `GET /api/v1/auth/status` must be PUBLIC (no auth middleware)
  - [ ] 6.4: `POST /api/v1/auth/setup` must be PUBLIC (only works when setup incomplete)

- [ ] Task 7: Frontend - Password Setup Component (AC: 1, 4)
  - [ ] 7.1: Create `/apps/web/src/components/auth/PasswordSetup.tsx`
  - [ ] 7.2: Password/PIN toggle (radio or tab selection)
  - [ ] 7.3: Password field with show/hide toggle
  - [ ] 7.4: Confirm password field
  - [ ] 7.5: Real-time strength indicator (weak/medium/strong with color)
  - [ ] 7.6: Validation messages in zh-TW
  - [ ] 7.7: Submit button with loading state

- [ ] Task 8: Frontend - API Service (AC: 1, 3)
  - [ ] 8.1: Create `/apps/web/src/services/authService.ts`
  - [ ] 8.2: `setupPassword(password, authType)` - POST /api/v1/auth/setup
  - [ ] 8.3: `changePassword(current, new)` - PUT /api/v1/auth/password
  - [ ] 8.4: `getAuthStatus()` - GET /api/v1/auth/status
  - [ ] 8.5: Use TanStack Query mutations for setup/change operations

## Dev Notes

### Architecture Compliance

- **Layering:** Handler → Service → Repository (MANDATORY - Rule 4)
- **Single-user system:** No username field. The system has exactly one user. This prepares for multi-user in Growth phase (Epic 13) by using a `users` table.
- **bcrypt cost factor:** MUST be 12 (per architecture decision #3)
- **Password hashing:** Use `golang.org/x/crypto/bcrypt` package
- **Interface location:** `AuthServiceInterface` in services package, `AuthRepositoryInterface` in repository package (Rule 11)

### Existing Code to Reuse

- **Response helpers:** `/apps/api/internal/handlers/response.go` - `SuccessResponse()`, `ErrorResponse()`, `BadRequestError()`
- **Config pattern:** `/apps/api/internal/config/config.go` - follow existing env var pattern
- **Repository pattern:** Follow exact same pattern as `/apps/api/internal/repository/movie_repository.go`
- **Service pattern:** Follow exact same pattern as `/apps/api/internal/services/` existing services
- **Crypto package:** `/apps/api/internal/crypto/` already exists for AES-256 encryption (DO NOT use this for passwords - use bcrypt)

### Dependencies to Add

```
go get golang.org/x/crypto  # Already in go.mod (used for PBKDF2), bcrypt is in same package
```

Note: `golang.org/x/crypto` v0.44.0 is already in go.mod. bcrypt is at `golang.org/x/crypto/bcrypt`.

### Error Codes

- `AUTH_INVALID_CREDENTIALS` - Wrong current password during change
- `AUTH_WEAK_PASSWORD` - Password doesn't meet minimum requirements
- `AUTH_SETUP_ALREADY_COMPLETE` - Trying to run setup when already configured
- `VALIDATION_REQUIRED_FIELD` - Missing required fields

### Project Structure Notes

- All backend code in `/apps/api` (Rule 1)
- Logging with `slog` only (Rule 2)
- API responses use `ApiResponse` wrapper (Rule 3)
- Tests co-located: `auth_service_test.go` next to `auth_service.go` (Rule 9)
- API paths: `/api/v1/auth/*` (Rule 10)

### References

- [Source: architecture.md#Authentication Strategy: JWT (Stateless)] - bcrypt cost 12, password security
- [Source: architecture.md#Security Parameters] - httpOnly cookie, HS256 signing
- [Source: epics.md#Story 7.1] - Acceptance criteria and technical notes
- [Source: project-context.md#Rule 3] - API response format
- [Source: project-context.md#Rule 6] - Naming conventions

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
