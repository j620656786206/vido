# Story 8.2: Zimuku Web Scraper

Status: review

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

- [x] Task 1: Implement Zimuku provider struct (AC: 4)
  - [x] 1.1: Create `zimuku.go` with `ZimukuProvider` struct holding httpClient, userAgents, skipDelays
  - [x] 1.2: Implement `NewZimukuProvider()` — 10 browser User-Agent strings
  - [x] 1.3: Implement `Name()` returning `"zimuku"`
  - [x] 1.4: Implement `randomUserAgent()` for rotation
  - [x] 1.5: Implement `randomDelay()` — 1-3s with context + skipDelays flag for tests

- [x] Task 2: Implement HTML parsing helpers (AC: 2, 7)
  - [x] 2.1: goquery already available in go.mod
  - [x] 2.2: Implement `parseSearchResults()` with CSS selectors
  - [x] 2.3: Parse: title, language, downloads, group, detail URL
  - [x] 2.4: `mapZimukuLanguage()`: 繁體→zh-Hant, 簡體→zh-Hans, 雙語→zh, English→en
  - [x] 2.5: Returns nil for empty results (graceful), ErrParseFailure for missing download links

- [x] Task 3: Implement Search method (AC: 1, 2, 4, 5, 6, 7)
  - [x] 3.1-3.7: Full implementation with anti-scraping, CAPTCHA detection, goquery parsing

- [x] Task 4: Implement Download method (AC: 3, 4, 5, 6)
  - [x] 4.1-4.7: Detail page → extract download link → fetch file, with CAPTCHA detection

- [x] Task 5: Define error types (AC: 5, 7)
  - [x] 5.1: `ErrCaptchaDetected` sentinel error
  - [x] 5.2: `ErrParseFailure` struct with Selector/Context fields
  - [x] 5.3: Compatible with `errors.Is()` and `errors.As()`

- [x] Task 6: Write unit tests (AC: all)
  - [x] 6.1-6.11: 17 tests with HTML fixtures, 82.8% coverage

- [x] Task 7: Build verification (AC: all)
  - [x] 7.1-7.3: build, test, vet all pass

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
Claude Opus 4.6 (1M context)

### Completion Notes List
- Implemented ZimukuProvider with goquery HTML scraping
- Anti-scraping: 10 UA rotation, random 1-3s delays, browser headers
- CAPTCHA detection via content analysis (captcha, 驗證碼, recaptcha keywords)
- ErrCaptchaDetected sentinel + ErrParseFailure struct error
- Language mapping: 繁體→zh-Hant, 簡體→zh-Hans, 雙語→zh, English→en
- io.LimitReader for OOM protection (2MB HTML, 50MB downloads)
- 17 tests with HTML fixtures, 82.8% coverage
- 🎨 UX Verification: SKIPPED — no UI changes

### File List
- apps/api/internal/subtitle/providers/zimuku.go (NEW)
- apps/api/internal/subtitle/providers/zimuku_test.go (NEW)
