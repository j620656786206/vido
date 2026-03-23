# Story 8.1: Assrt API Client

Status: ready-for-dev

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

- [ ] Task 1: Define SubtitleResult and SubtitleQuery types (AC: 1, 3)
  - [ ] 1.1: Create `apps/api/internal/subtitle/providers/provider.go` with `SubtitleProvider` interface, `SubtitleQuery` struct (Title, Year, ImdbID, Season, Episode, FileHash), and `SubtitleResult` struct (ID, Source, Filename, Language, DownloadURL, FileSize, UploadDate, Downloads, Group)
  - [ ] 1.2: Write godoc comments for each type and method in the interface

- [ ] Task 2: Implement Assrt API client struct (AC: 1, 2, 5)
  - [ ] 2.1: Create `apps/api/internal/subtitle/providers/assrt.go` with `AssrtProvider` struct holding apiKey, httpClient, rateLimiter
  - [ ] 2.2: Implement `NewAssrtProvider(secretsService secrets.Service) *AssrtProvider` constructor — fetch API key from secrets, if not found set `disabled=true`
  - [ ] 2.3: Add `golang.org/x/time/rate` limiter initialized to 2 requests/second
  - [ ] 2.4: Implement `Name() string` returning `"assrt"`

- [ ] Task 3: Implement Search method (AC: 1, 2, 3, 5, 6, 7)
  - [ ] 3.1: Implement `Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)` — if disabled, return `nil, nil`
  - [ ] 3.2: Build API request URL: `https://api.assrt.net/v1/sub/search` with params `q` (title), `token` (API key)
  - [ ] 3.3: Call `rateLimiter.Wait(ctx)` before making HTTP request
  - [ ] 3.4: Parse response JSON — use correct key `native_name` (NOT `name`) for the Chinese title (P1-011 fix)
  - [ ] 3.5: Map each result to `SubtitleResult` with source="assrt", extract language from metadata fields
  - [ ] 3.6: Handle HTTP errors: wrap with status code and endpoint context
  - [ ] 3.7: Handle JSON parse errors: log raw body at debug, return descriptive error

- [ ] Task 4: Implement Download method (AC: 4, 5, 6)
  - [ ] 4.1: Implement `Download(ctx context.Context, id string) ([]byte, error)` — fetch subtitle content by ID
  - [ ] 4.2: Build download URL: `https://api.assrt.net/v1/sub/detail` with subtitle ID, then extract download link
  - [ ] 4.3: Call `rateLimiter.Wait(ctx)` before each HTTP request
  - [ ] 4.4: Return raw subtitle file bytes
  - [ ] 4.5: Handle errors (HTTP errors, empty response, timeout)

- [ ] Task 5: Write unit tests (AC: all)
  - [ ] 5.1: Create `apps/api/internal/subtitle/providers/assrt_test.go`
  - [ ] 5.2: Use `httptest.NewServer` to mock Assrt API responses
  - [ ] 5.3: Test successful search with `native_name` parsing (verify P1-011 fix)
  - [ ] 5.4: Test search when API key is not configured (disabled mode)
  - [ ] 5.5: Test rate limiter behavior (verify requests are throttled)
  - [ ] 5.6: Test HTTP error handling (4xx, 5xx, timeout)
  - [ ] 5.7: Test malformed JSON handling
  - [ ] 5.8: Test successful download
  - [ ] 5.9: Verify ≥80% code coverage

- [ ] Task 6: Build verification (AC: all)
  - [ ] 6.1: Run `go build ./...` — verify no compilation errors
  - [ ] 6.2: Run `go test ./internal/subtitle/...` — verify all tests pass
  - [ ] 6.3: Run `go vet ./internal/subtitle/...` — verify no vet issues

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
### Completion Notes List
### File List
