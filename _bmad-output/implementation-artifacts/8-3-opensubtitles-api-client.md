# Story 8.3: OpenSubtitles API Client

Status: review

## Story

As a **NAS media collector**,
I want **Vido to search and download subtitles from OpenSubtitles using their REST API**,
so that **I can access the world's largest subtitle database with hash-based matching for accurate subtitle-to-file pairing**.

## Acceptance Criteria

1. **Given** OpenSubtitles API credentials are configured via the secrets service, **When** `Search()` is called with an IMDB ID and optional file hash, **Then** the provider returns a list of `SubtitleResult` entries from the OpenSubtitles REST API v1 with source="opensubtitles"
2. **Given** a media file hash is provided in the query, **When** searching, **Then** the provider includes the hash parameter for hash-based matching (higher accuracy than title-only search)
3. **Given** valid search results, **When** results are returned, **Then** each `SubtitleResult` includes subtitle ID, filename, language code, download count, uploader, and file format
4. **Given** a valid subtitle file ID, **When** `Download()` is called, **Then** the provider requests a download link from the API, fetches the file, and returns its content as `[]byte`
5. **Given** the API requires authentication, **When** the auth token expires or is missing, **Then** the provider automatically re-authenticates using stored credentials and retries the original request
6. **Given** OpenSubtitles API credentials are NOT configured, **When** the provider is initialized, **Then** it operates in disabled mode and returns empty results (not errors)
7. **Given** the API returns rate limiting (HTTP 429), **When** the response is received, **Then** the provider respects the Retry-After header and retries after the specified delay
8. **Given** the API returns an error (4xx/5xx other than 429), **When** the error is encountered, **Then** the provider returns a wrapped error with status code and endpoint context

## Tasks / Subtasks

- [ ] Task 1: Implement OpenSubtitles provider struct (AC: 5, 6)
  - [ ] 1.1: Create `apps/api/internal/subtitle/providers/opensub.go` with `OpenSubProvider` struct holding apiKey, username, password, httpClient, authToken, tokenExpiry, and mutex for token refresh
  - [ ] 1.2: Implement `NewOpenSubProvider(secretsService secrets.Service) *OpenSubProvider` constructor — fetch API key, username, password from secrets; if not found set `disabled=true`
  - [ ] 1.3: Implement `Name() string` returning `"opensubtitles"`

- [ ] Task 2: Implement authentication (AC: 5)
  - [ ] 2.1: Implement `authenticate(ctx context.Context) error` — POST to `/api/v1/login` with username/password, store JWT token and expiry
  - [ ] 2.2: Implement `ensureAuth(ctx context.Context) error` — check token validity, call `authenticate()` if expired or missing
  - [ ] 2.3: Use sync.Mutex to prevent concurrent authentication attempts
  - [ ] 2.4: Set token expiry conservatively (refresh 5 minutes before actual expiry)

- [ ] Task 3: Implement Search method (AC: 1, 2, 3, 6, 7, 8)
  - [ ] 3.1: Implement `Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)` — if disabled, return `nil, nil`
  - [ ] 3.2: Call `ensureAuth(ctx)` before making API request
  - [ ] 3.3: Build search request: `GET /api/v1/subtitles` with params: `imdb_id`, `languages` (comma-separated), `moviehash` (if available), `season_number`, `episode_number`
  - [ ] 3.4: Set headers: `Api-Key`, `Authorization: Bearer {token}`, `Content-Type: application/json`
  - [ ] 3.5: Handle HTTP 429: parse `Retry-After` header, sleep, retry (max 1 retry)
  - [ ] 3.6: Parse response JSON, map to `SubtitleResult` with source="opensubtitles"
  - [ ] 3.7: Handle other HTTP errors with wrapped context

- [ ] Task 4: Implement Download method (AC: 4, 5, 7, 8)
  - [ ] 4.1: Implement `Download(ctx context.Context, id string) ([]byte, error)`
  - [ ] 4.2: Call `ensureAuth(ctx)` before request
  - [ ] 4.3: POST to `/api/v1/download` with `{"file_id": id}` to get download link
  - [ ] 4.4: Fetch the actual subtitle file from the returned download URL
  - [ ] 4.5: Handle 429 rate limiting with retry
  - [ ] 4.6: Return raw subtitle file bytes

- [ ] Task 5: Implement file hash calculation utility (AC: 2)
  - [ ] 5.1: Implement `CalculateOpenSubHash(filePath string) (string, error)` — OpenSubtitles hash algorithm (first+last 64KB, size-based)
  - [ ] 5.2: Export as a package-level function for use by the engine
  - [ ] 5.3: Write tests with known hash values

- [ ] Task 6: Write unit tests (AC: all)
  - [ ] 6.1: Create `apps/api/internal/subtitle/providers/opensub_test.go`
  - [ ] 6.2: Use `httptest.NewServer` to mock OpenSubtitles API
  - [ ] 6.3: Test authentication flow (login, token storage, token refresh)
  - [ ] 6.4: Test search with IMDB ID only
  - [ ] 6.5: Test search with IMDB ID + file hash (verify hash parameter is sent)
  - [ ] 6.6: Test search with season/episode for TV shows
  - [ ] 6.7: Test disabled mode when credentials are missing
  - [ ] 6.8: Test HTTP 429 rate limiting with Retry-After
  - [ ] 6.9: Test token expiry triggers re-authentication
  - [ ] 6.10: Test download flow (get link → fetch file)
  - [ ] 6.11: Test HTTP error handling (401 re-auth, 4xx, 5xx)
  - [ ] 6.12: Test file hash calculation with known values
  - [ ] 6.13: Verify ≥80% code coverage

- [ ] Task 7: Build verification (AC: all)
  - [ ] 7.1: Run `go build ./...` — verify no compilation errors
  - [ ] 7.2: Run `go test ./internal/subtitle/...` — verify all tests pass
  - [ ] 7.3: Run `go vet ./internal/subtitle/...` — verify no vet issues

## Dev Notes

### Architecture & Patterns
- Must implement `SubtitleProvider` interface from `provider.go` (created in Story 8-1)
- OpenSubtitles REST API v1 is the modern API (not the legacy XML-RPC API) — use `https://api.opensubtitles.com/api/v1/` as base URL
- Auth token management: JWT tokens from `/login` have an expiry. The provider must transparently handle re-auth without leaking auth concerns to the engine
- Hash-based matching is OpenSubtitles' killer feature — when a file hash matches, subtitle accuracy is near 100%

### Project Structure Notes
- Provider file: `apps/api/internal/subtitle/providers/opensub.go`
- Test file: `apps/api/internal/subtitle/providers/opensub_test.go`
- The hash function may be called by the engine before search — consider placing it as a public function or in a shared util
- Depends on `provider.go` types from Story 8-1

### OpenSubtitles API Reference
- Base URL: `https://api.opensubtitles.com/api/v1/`
- Auth: `POST /login` → JWT token; pass as `Authorization: Bearer {token}` header
- API key: `Api-Key` header on all requests
- Search: `GET /subtitles?imdb_id={id}&languages=zh-cn,zh-tw&moviehash={hash}`
- Download: `POST /download` with `{"file_id": 123}` → returns `{"link": "https://..."}`
- Rate limit: 429 with `Retry-After` header

### OpenSubtitles Hash Algorithm
The hash is computed by combining: file size (as a 64-bit little-endian integer) + sum of first 64KB (as 64-bit words) + sum of last 64KB (as 64-bit words). This is a well-documented algorithm with reference implementations available.

### References
- PRD feature: P1-010 (multi-source search)
- `SubtitleProvider` interface: `apps/api/internal/subtitle/providers/provider.go`
- Secrets service: `apps/api/internal/secrets/`
- OpenSubtitles API docs: https://opensubtitles.stoplight.io/docs/opensubtitles-api

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Completion Notes List
- OpenSubtitles REST API v1 with JWT auth token management
- Transparent re-auth: expired tokens auto-refresh before API calls
- Hash-based matching via CalculateOpenSubHash (first+last 64KB)
- HTTP 429 rate limiting: parse Retry-After header, wait and retry once
- Disabled mode: empty results when API key missing (not errors)
- 16 unit tests, 80.6% coverage, zero regressions
- 🎨 UX Verification: SKIPPED — no UI changes

### File List
- apps/api/internal/subtitle/providers/opensub.go (NEW)
- apps/api/internal/subtitle/providers/opensub_test.go (NEW)
