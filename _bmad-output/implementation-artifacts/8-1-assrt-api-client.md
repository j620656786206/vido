# Story 8.1: Assrt API Client

Status: done

## Story

As a **NAS media collector**,
I want **Vido to search and download subtitles from the Assrt subtitle source using their API**,
so that **I have access to one of the largest Chinese subtitle databases for matching Traditional Chinese subtitles to my media files**.

## Acceptance Criteria

1. **Given** an Assrt API key is configured via the secrets service, **When** `Search()` is called with a title and optional year, **Then** the provider returns a list of `SubtitleResult` entries parsed from the Assrt API response using the correct `native_name` key
2. **Given** an Assrt API key is NOT configured, **When** the provider is initialized or `Search()` is called, **Then** the provider returns an empty result set (not an error) and logs an info-level message that Assrt is skipped
3. **Given** valid search results exist, **When** results are returned, **Then** each `SubtitleResult` includes source="assrt", the subtitle ID, filename, language metadata, and download URL
4. **Given** a valid subtitle ID, **When** `Download()` is called, **Then** the provider fetches the subtitle file content as `[]byte` and returns it
5. **Given** the Assrt API rate limit (2 requests/second), **When** multiple requests are made in rapid succession, **Then** the client throttles requests using a rate limiter to stay within the limit
6. **Given** the Assrt API returns an HTTP error (4xx/5xx) or network timeout, **When** the error is encountered, **Then** the provider returns a wrapped error with context (status code, endpoint) and does NOT panic or crash
7. **Given** the Assrt API returns malformed JSON or unexpected response structure, **When** parsing fails, **Then** the provider returns an error describing the parse failure and the raw response is logged at debug level

## Tasks / Subtasks

- [x] Task 1: Define SubtitleResult and SubtitleQuery types (AC: 1, 3)
  - [x] 1.1: Create `apps/api/internal/subtitle/providers/provider.go` with `SubtitleProvider` interface, `SubtitleQuery` struct (Title, Year, ImdbID, Season, Episode, FileHash, Languages), and `SubtitleResult` struct (ID, Source, Filename, Language, DownloadURL, FileSize, UploadDate, Downloads, Group, Resolution, Format)
  - [x] 1.2: Write godoc comments for each type and method in the interface

- [x] Task 2: Implement Assrt API client struct (AC: 1, 2, 5)
  - [x] 2.1: Create `apps/api/internal/subtitle/providers/assrt.go` with `AssrtProvider` struct holding apiKey, httpClient, rateLimiter
  - [x] 2.2: Implement `NewAssrtProvider(ctx, secretsService) *AssrtProvider` constructor — fetch API key from secrets, if not found set `disabled=true`
  - [x] 2.3: Add `golang.org/x/time/rate` limiter initialized to 2 requests/second
  - [x] 2.4: Implement `Name() string` returning `"assrt"`

- [x] Task 3: Implement Search method (AC: 1, 2, 3, 5, 6, 7)
  - [x] 3.1: Implement `Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)` — if disabled, return `nil, nil`
  - [x] 3.2: Build API request URL: `https://api.assrt.net/v1/sub/search` with params `q` (title), `token` (API key)
  - [x] 3.3: Call `rateLimiter.Wait(ctx)` before making HTTP request
  - [x] 3.4: Parse response JSON — use correct key `native_name` (NOT `name`) for the Chinese title (P1-011 fix)
  - [x] 3.5: Map each result to `SubtitleResult` with source="assrt", extract language from metadata fields
  - [x] 3.6: Handle HTTP errors: wrap with status code and endpoint context
  - [x] 3.7: Handle JSON parse errors: log raw body at debug, return descriptive error

- [x] Task 4: Implement Download method (AC: 4, 5, 6)
  - [x] 4.1: Implement `Download(ctx context.Context, id string) ([]byte, error)` — fetch subtitle content by ID
  - [x] 4.2: Build download URL via `/sub/detail` endpoint, then extract download link from filelist
  - [x] 4.3: Call `rateLimiter.Wait(ctx)` before each HTTP request
  - [x] 4.4: Return raw subtitle file bytes
  - [x] 4.5: Handle errors (HTTP errors, empty response, timeout, no download URL)

- [x] Task 5: Write unit tests (AC: all)
  - [x] 5.1: Create `apps/api/internal/subtitle/providers/assrt_test.go`
  - [x] 5.2: Use `httptest.NewServer` to mock Assrt API responses
  - [x] 5.3: Test successful search with `native_name` parsing (verify P1-011 fix)
  - [x] 5.4: Test search when API key is not configured (disabled mode)
  - [x] 5.5: Test rate limiter behavior (verify requests are throttled)
  - [x] 5.6: Test HTTP error handling (4xx, 5xx, timeout)
  - [x] 5.7: Test malformed JSON handling
  - [x] 5.8: Test successful download
  - [x] 5.9: Verify ≥80% code coverage — achieved 82.4%

- [x] Task 6: Build verification (AC: all)
  - [x] 6.1: Run `go build ./...` — no compilation errors
  - [x] 6.2: Run `go test ./internal/subtitle/...` — 13 tests pass
  - [x] 6.3: Run `go vet ./internal/subtitle/...` — no vet issues

## Dev Notes

### Architecture & Patterns
- The `SubtitleProvider` interface is the contract all three providers (Assrt, Zimuku, OpenSubtitles) must implement — defined once in `provider.go` and shared
- Rate limiting uses `golang.org/x/time/rate` — a token bucket limiter; call `Wait(ctx)` which blocks until a token is available or context is cancelled
- The secrets service (`apps/api/internal/secrets/`) manages encrypted API keys — use `secretsService.Get("assrt_api_key")` to retrieve
- Assrt being OPTIONAL is critical: when no API key exists, the provider must gracefully degrade (return empty, not error) so the engine can proceed with other sources

### Project Structure Notes
- All provider files go under `apps/api/internal/subtitle/providers/`
- The `SubtitleProvider` interface in `provider.go` will be used by Story 8-6+ (engine/scorer) — design it to be stable
- HTTP client should use a shared `http.Client` with reasonable timeouts (10s connect, 30s total)

### Key Implementation Detail: P1-011 Fix
The existing Assrt integration had a bug using the wrong response key. The Assrt API returns Chinese titles under the `native_name` field, NOT `name`. Ensure all response parsing uses `native_name` for the localized title.

### Assrt API Reference
- Base URL: `https://api.assrt.net/v1/`
- Auth: `token` query parameter
- Search endpoint: `GET /sub/search?q={query}&token={key}`
- Detail endpoint: `GET /sub/detail?id={id}&token={key}`

### References
- PRD features: P1-010 (multi-source search), P1-011 (native_name fix)
- Secrets service: `apps/api/internal/secrets/`
- SSE hub: `apps/api/internal/sse/` (used by engine in later stories)

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Completion Notes List
- Created shared `SubtitleProvider` interface, `SubtitleQuery`, and `SubtitleResult` types in `provider.go` — stable contract for all 3 providers
- Implemented `AssrtProvider` with optional API key (disabled mode returns empty, not error)
- P1-011 fix applied: response parsing uses `native_name` key for Chinese titles
- Rate limiter: `golang.org/x/time/rate` at 2 req/s with token bucket
- Download uses two-step flow: detail API → extract download URL from filelist → fetch file
- 13 unit tests with httptest mock server, 82.4% coverage
- Full regression suite passes (all existing packages cached + new tests green)
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Code Review Findings (applied 2026-03-23)
- **HIGH — Unbounded io.ReadAll (OOM risk)**: All 3 `io.ReadAll` calls had no size limit. Fixed: API JSON responses capped at 1MB via `io.LimitReader`, subtitle downloads capped at 50MB.
- **HIGH — API key leakable in error messages**: Go's `http` package can include full URLs (with `token=` query param) in error messages. Fixed: added `sanitizeTokenError()` helper that redacts token values from error strings.
- **MEDIUM — Rate limiter burst=1 too restrictive**: Token bucket burst was 1 but rate is 2 req/s. Fixed: burst set to 2 (matching rate) via `assrtRateBurst` const.
- **MEDIUM — No validation of empty search title**: `Search()` accepted empty Title, sending meaningless API requests. Fixed: early return with descriptive error.
- **MEDIUM — Flaky context cancellation test (5s block)**: Test server handler slept 5s unconditionally, blocking `server.Close()`. Fixed: handler now listens on `r.Context().Done()` to exit promptly.
- **LOW — No User-Agent header**: Requests lacked identification. Fixed: all requests now send `User-Agent: Vido/1.0 (NAS Media Manager)`.
- **LOW — Dead code in test helper**: Unused `origBaseURL` variable removed.

### File List
- apps/api/internal/subtitle/providers/provider.go (NEW)
- apps/api/internal/subtitle/providers/assrt.go (NEW)
- apps/api/internal/subtitle/providers/assrt_test.go (NEW)
