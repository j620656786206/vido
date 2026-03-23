# Story 8.2: Zimuku Web Scraper

Status: ready-for-dev

## Story

As a **NAS media collector**,
I want **Vido to search and download subtitles from Zimuku (字幕庫) via web scraping**,
so that **I can access one of the most popular Chinese subtitle communities with a large Traditional Chinese subtitle catalogue**.

## Acceptance Criteria

1. **Given** a media title and optional year, **When** `Search()` is called, **Then** the provider scrapes the Zimuku search results page and returns a list of `SubtitleResult` entries with source="zimuku"
2. **Given** Zimuku search results HTML, **When** parsing the page, **Then** the scraper extracts subtitle title, language tags, download count, uploader/group, and detail page URL for each result
3. **Given** a valid subtitle ID (detail page URL), **When** `Download()` is called, **Then** the provider navigates to the detail page, extracts the download link, fetches the subtitle file, and returns its content as `[]byte`
4. **Given** Zimuku applies anti-scraping measures, **When** making HTTP requests, **Then** the client rotates User-Agent strings from a pool of common browser agents and applies random backoff between requests (1-3 seconds)
5. **Given** Zimuku returns a CAPTCHA challenge page, **When** the scraper detects it (by page structure or response code), **Then** it returns a specific `ErrCaptchaDetected` error and does NOT attempt to solve the CAPTCHA
6. **Given** Zimuku is unreachable or returns an HTTP error, **When** the error is encountered, **Then** the provider returns a wrapped error with context and does not block other providers
7. **Given** the search results page structure changes unexpectedly, **When** HTML parsing fails to find expected elements, **Then** the provider returns an `ErrParseFailure` error with details about what selectors failed

## Tasks / Subtasks

- [ ] Task 1: Implement Zimuku provider struct (AC: 4)
  - [ ] 1.1: Create `apps/api/internal/subtitle/providers/zimuku.go` with `ZimukuProvider` struct holding httpClient, userAgents []string, and a mutex for request serialization
  - [ ] 1.2: Implement `NewZimukuProvider() *ZimukuProvider` constructor — initialize with pool of 10+ common browser User-Agent strings
  - [ ] 1.3: Implement `Name() string` returning `"zimuku"`
  - [ ] 1.4: Implement `randomUserAgent() string` for rotation
  - [ ] 1.5: Implement `randomDelay(ctx context.Context) error` — sleep 1-3 seconds with context cancellation support

- [ ] Task 2: Implement HTML parsing helpers (AC: 2, 7)
  - [ ] 2.1: Add `github.com/PuerkitoBio/goquery` dependency for HTML parsing
  - [ ] 2.2: Implement `parseSearchResults(doc *goquery.Document) ([]SubtitleResult, error)` — extract subtitle entries from search results page
  - [ ] 2.3: Parse each result row: title text, language icons/tags, download count, group/uploader name, detail URL
  - [ ] 2.4: Map language tags to language codes (e.g., "繁體" → "zh-Hant", "簡體" → "zh-Hans", "雙語" → "zh")
  - [ ] 2.5: Return `ErrParseFailure` with selector details if expected elements are missing

- [ ] Task 3: Implement Search method (AC: 1, 2, 4, 5, 6, 7)
  - [ ] 3.1: Implement `Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)`
  - [ ] 3.2: Build search URL: `https://zimuku.org/search?q={title}` (URL-encode query)
  - [ ] 3.3: Apply random delay before request
  - [ ] 3.4: Set rotated User-Agent and common browser headers (Accept, Accept-Language, Referer)
  - [ ] 3.5: Detect CAPTCHA response: check for known CAPTCHA page patterns, return `ErrCaptchaDetected`
  - [ ] 3.6: Parse HTML response using goquery, call `parseSearchResults()`
  - [ ] 3.7: Map results to `SubtitleResult` with source="zimuku"

- [ ] Task 4: Implement Download method (AC: 3, 4, 5, 6)
  - [ ] 4.1: Implement `Download(ctx context.Context, id string) ([]byte, error)` — id is the detail page URL path
  - [ ] 4.2: Fetch detail page with anti-scraping headers
  - [ ] 4.3: Parse detail page to extract download link
  - [ ] 4.4: Apply random delay before download request
  - [ ] 4.5: Follow redirects to fetch actual subtitle file
  - [ ] 4.6: Return raw subtitle file bytes
  - [ ] 4.7: Handle CAPTCHA detection on detail/download pages

- [ ] Task 5: Define error types (AC: 5, 7)
  - [ ] 5.1: Define `ErrCaptchaDetected` as a sentinel error in zimuku.go
  - [ ] 5.2: Define `ErrParseFailure` struct error with Selector and Context fields
  - [ ] 5.3: Ensure both errors implement `error` interface and are distinguishable with `errors.Is()` / `errors.As()`

- [ ] Task 6: Write unit tests (AC: all)
  - [ ] 6.1: Create `apps/api/internal/subtitle/providers/zimuku_test.go`
  - [ ] 6.2: Create test HTML fixtures: search results page, detail page, CAPTCHA page
  - [ ] 6.3: Use `httptest.NewServer` to serve fixture HTML
  - [ ] 6.4: Test successful search parsing (verify all fields extracted correctly)
  - [ ] 6.5: Test language tag mapping (繁體, 簡體, 雙語, 英文)
  - [ ] 6.6: Test CAPTCHA detection returns `ErrCaptchaDetected`
  - [ ] 6.7: Test parse failure when HTML structure changes (missing expected selectors)
  - [ ] 6.8: Test successful download flow (detail page → download link → file)
  - [ ] 6.9: Test User-Agent rotation (verify different agents are used)
  - [ ] 6.10: Test HTTP error handling (timeout, 403, 500)
  - [ ] 6.11: Verify ≥80% code coverage

- [ ] Task 7: Build verification (AC: all)
  - [ ] 7.1: Run `go build ./...` — verify no compilation errors
  - [ ] 7.2: Run `go test ./internal/subtitle/...` — verify all tests pass
  - [ ] 7.3: Run `go vet ./internal/subtitle/...` — verify no vet issues

## Dev Notes

### Architecture & Patterns
- Must implement `SubtitleProvider` interface from `provider.go` (created in Story 8-1)
- Zimuku is a web scraping source (no official API), so reliability is inherently lower than API-based sources — the engine (Story 8-6+) accounts for this in source trust scoring
- Anti-scraping is essential: real-world Zimuku blocks automated access aggressively. User-Agent rotation + random delays are the minimum viable defense
- CAPTCHA is a known failure mode — the correct behavior is to fail gracefully, not attempt to solve it. The engine will fall back to other sources

### Project Structure Notes
- Provider file: `apps/api/internal/subtitle/providers/zimuku.go`
- Test file: `apps/api/internal/subtitle/providers/zimuku_test.go`
- HTML test fixtures can be embedded as string constants or placed in a `testdata/` directory
- Depends on `provider.go` types from Story 8-1

### HTML Parsing Strategy
- Use `goquery` (jQuery-like CSS selector API for Go) — it is the standard Go library for HTML scraping
- Zimuku search results are typically in a table or list structure; use CSS selectors to target result rows
- Language is indicated by icon classes or text labels (繁體中文, 簡體中文, 雙語, English)
- Download count is usually in a specific column/span

### Anti-Scraping Headers
```go
req.Header.Set("User-Agent", randomUA)
req.Header.Set("Accept", "text/html,application/xhtml+xml")
req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
req.Header.Set("Referer", "https://zimuku.org/")
```

### References
- PRD feature: P1-010 (multi-source search)
- `SubtitleProvider` interface: `apps/api/internal/subtitle/providers/provider.go`
- goquery: `github.com/PuerkitoBio/goquery`

## Dev Agent Record

### Agent Model Used
### Completion Notes List
### File List
