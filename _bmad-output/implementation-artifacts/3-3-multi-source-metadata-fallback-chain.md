# Story 3.3: Multi-Source Metadata Fallback Chain

Status: done

## Story

As a **media collector**,
I want the **system to try multiple metadata sources automatically**,
So that **I always get metadata even when one source fails**.

## Acceptance Criteria

1. **AC1: TMDb to Douban Fallback**
   - Given a search query is made
   - When TMDb returns no results
   - Then the system automatically tries Douban within 1 second (NFR-R3)
   - And the user sees "TMDb ❌ → Searching Douban..." status

2. **AC2: Douban to Wikipedia Fallback**
   - Given both TMDb and Douban fail
   - When fallback continues
   - Then the system tries Wikipedia MediaWiki API
   - And respects Wikipedia rate limit (1 req/s, NFR-I14)

3. **AC3: Manual Fallback Option**
   - Given all automated sources fail
   - When the fallback chain completes
   - Then the user is offered manual search option
   - And the status shows all attempted sources: "TMDb ❌ → Douban ❌ → Wikipedia ❌ → Manual search"

4. **AC4: Source Indication**
   - Given any source succeeds
   - When metadata is found
   - Then the source is indicated (FR42): "Source: Douban"
   - And the fallback chain stops

5. **AC5: Circuit Breaker Protection**
   - Given a metadata source is consistently failing
   - When failure threshold is reached (e.g., 5 failures in 1 minute)
   - Then the circuit breaker opens and skips that source temporarily
   - And the system logs the circuit state change

## Tasks / Subtasks

- [x] Task 1: Create Metadata Provider Interface (AC: 1, 2, 4)
  - [x] 1.1: Create `/apps/api/internal/metadata/provider.go` with `MetadataProvider` interface
  - [x] 1.2: Define common `SearchRequest` and `SearchResult` types
  - [x] 1.3: Create `ProviderStatus` enum (available, unavailable, rate_limited)
  - [x] 1.4: Write interface tests

- [x] Task 2: Implement Circuit Breaker (AC: 5)
  - [x] 2.1: Create `/apps/api/internal/metadata/circuit_breaker.go`
  - [x] 2.2: Implement states: Closed (normal), Open (failing), Half-Open (testing)
  - [x] 2.3: Configure thresholds: 5 failures to open, 30s timeout to half-open
  - [x] 2.4: Add metrics for circuit state changes
  - [x] 2.5: Write circuit breaker tests

- [x] Task 3: Create Fallback Chain Orchestrator (AC: 1, 2, 3, 4)
  - [x] 3.1: Create `/apps/api/internal/metadata/orchestrator.go`
  - [x] 3.2: Implement ordered provider chain: TMDb → Douban → Wikipedia → Manual
  - [x] 3.3: Add <1 second timeout between fallbacks (NFR-R3)
  - [x] 3.4: Emit progress events for each source attempt
  - [x] 3.5: Return first successful result with source indication
  - [x] 3.6: Write orchestrator tests

- [x] Task 4: Adapt TMDb Service as Provider (AC: 1, 4)
  - [x] 4.1: Create `/apps/api/internal/metadata/tmdb_provider.go`
  - [x] 4.2: Wrap existing `TMDbService` to implement `MetadataProvider`
  - [x] 4.3: Map TMDb responses to common `SearchResult` format
  - [x] 4.4: Write adapter tests

- [x] Task 5: Create Douban Provider Stub (AC: 2, 4)
  - [x] 5.1: Create `/apps/api/internal/metadata/douban_provider.go`
  - [x] 5.2: Implement `MetadataProvider` interface (stub for now)
  - [x] 5.3: Return "not implemented" until Story 3.4
  - [x] 5.4: Add configuration for enable/disable

- [x] Task 6: Create Wikipedia Provider Stub (AC: 2, 4)
  - [x] 6.1: Create `/apps/api/internal/metadata/wikipedia_provider.go`
  - [x] 6.2: Implement `MetadataProvider` interface (stub for now)
  - [x] 6.3: Return "not implemented" until Story 3.5
  - [x] 6.4: Add rate limiter for 1 req/s (NFR-I14)

- [x] Task 7: Create Metadata Service (AC: 1, 2, 3, 4, 5)
  - [x] 7.1: Create `/apps/api/internal/services/metadata_service.go`
  - [x] 7.2: Define `MetadataServiceInterface` in services package
  - [x] 7.3: Inject orchestrator with configured providers
  - [x] 7.4: Implement `SearchMetadata()` method
  - [x] 7.5: Wire up in `main.go`
  - [x] 7.6: Write service tests

- [x] Task 8: Update API Endpoints (AC: 3, 4)
  - [x] 8.1: Create `/api/v1/metadata/search` endpoint
  - [x] 8.2: Return fallback status in response
  - [x] 8.3: Include source indication in results
  - [x] 8.4: Write handler tests

### Review Follow-ups (AI)

- [ ] [AI-Review][LOW] TMDb provider does not pass Year filter to TMDb API (requires TMDbSearcher interface change) [tmdb_provider.go:132]

## Dev Notes

### Architecture Requirements

**ARCH-2: Multi-source metadata orchestrator**
- TMDb → Douban → Wikipedia → AI → Manual fallback chain
- Ordered provider execution with early termination on success

**ARCH-7: Circuit Breaker Pattern**
- Protect external service calls
- Implement fallback logic
- States: Closed → Open → Half-Open → Closed

**NFR-R3: Auto-fallback to Douban within <1s**
- Timeout between provider attempts must be <1 second

### Existing Codebase Integration

**MetadataSource Values (`/apps/api/internal/models/movie.go`):**
```go
const (
    MetadataSourceTMDb      MetadataSource = "tmdb"
    MetadataSourceDouban    MetadataSource = "douban"
    MetadataSourceWikipedia MetadataSource = "wikipedia"
    MetadataSourceManual    MetadataSource = "manual"
)
```

**Existing TMDb Service (`/apps/api/internal/services/tmdb_service.go`):**
- Already implements movie/TV search
- Returns zh-TW prioritized results
- Will be wrapped as a `MetadataProvider`

### Interface Design

```go
// MetadataProvider interface for all metadata sources
type MetadataProvider interface {
    Name() string
    Search(ctx context.Context, req *SearchRequest) (*SearchResult, error)
    IsAvailable() bool
    Source() models.MetadataSource
}

// SearchRequest common request format
type SearchRequest struct {
    Query     string
    Year      int    // Optional filter
    MediaType string // "movie" or "tv"
    Language  string // Preferred language (zh-TW)
}

// SearchResult common result format
type SearchResult struct {
    Items      []MetadataItem
    Source     models.MetadataSource
    TotalCount int
}

// MetadataItem normalized metadata
type MetadataItem struct {
    ID          string
    Title       string
    TitleZhTW   string // Traditional Chinese title
    Year        int
    Overview    string
    PosterURL   string
    MediaType   string
    Confidence  float64
    RawData     interface{} // Original provider response
}
```

### Circuit Breaker Design

```go
// CircuitBreaker states
type CircuitState int

const (
    StateClosed   CircuitState = iota // Normal operation
    StateOpen                          // Failing, skip provider
    StateHalfOpen                      // Testing recovery
)

// CircuitBreaker configuration
type CircuitBreakerConfig struct {
    FailureThreshold   int           // Failures to open (default: 5)
    SuccessThreshold   int           // Successes to close (default: 2)
    Timeout            time.Duration // Time in open state (default: 30s)
    HalfOpenMaxCalls   int           // Max calls in half-open (default: 1)
}

// CircuitBreaker interface
type CircuitBreaker interface {
    Execute(fn func() error) error
    State() CircuitState
    Reset()
}
```

### Orchestrator Design

```go
// Orchestrator manages the fallback chain
type Orchestrator struct {
    providers      []MetadataProvider
    circuitBreakers map[string]*CircuitBreaker
    logger         *slog.Logger
}

// Search executes the fallback chain
func (o *Orchestrator) Search(ctx context.Context, req *SearchRequest) (*SearchResult, *FallbackStatus) {
    status := &FallbackStatus{
        Attempts: []SourceAttempt{},
    }

    for _, provider := range o.providers {
        if !provider.IsAvailable() {
            continue
        }

        cb := o.circuitBreakers[provider.Name()]
        if cb.State() == StateOpen {
            status.Attempts = append(status.Attempts, SourceAttempt{
                Source:  provider.Source(),
                Skipped: true,
                Reason:  "circuit breaker open",
            })
            continue
        }

        // Execute with circuit breaker
        result, err := o.executeWithBreaker(ctx, provider, cb, req)

        status.Attempts = append(status.Attempts, SourceAttempt{
            Source:  provider.Source(),
            Success: err == nil && len(result.Items) > 0,
            Error:   err,
        })

        if err == nil && len(result.Items) > 0 {
            return result, status
        }

        // Wait <1s before next provider (NFR-R3)
        time.Sleep(100 * time.Millisecond)
    }

    return nil, status
}
```

### Project Structure Notes

**New Directory to Create:**
```
/apps/api/internal/metadata/
├── provider.go           # MetadataProvider interface
├── circuit_breaker.go    # Circuit breaker implementation
├── orchestrator.go       # Fallback chain orchestrator
├── tmdb_provider.go      # TMDb adapter
├── douban_provider.go    # Douban stub (until Story 3.4)
├── wikipedia_provider.go # Wikipedia stub (until Story 3.5)
└── *_test.go             # Tests for each file
```

**Files to Modify:**
- `/apps/api/internal/services/metadata_service.go` - New service
- `/apps/api/internal/handlers/metadata_handler.go` - New handler
- `/apps/api/cmd/api/main.go` - Wire up metadata service

### API Response Format

```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": "12345",
        "title": "Inception",
        "titleZhTW": "全面啟動",
        "year": 2010,
        "overview": "...",
        "posterUrl": "https://...",
        "mediaType": "movie"
      }
    ],
    "source": "tmdb",
    "fallbackStatus": {
      "attempts": [
        {"source": "tmdb", "success": true}
      ]
    }
  }
}
```

```json
{
  "success": true,
  "data": {
    "results": [...],
    "source": "douban",
    "fallbackStatus": {
      "attempts": [
        {"source": "tmdb", "success": false, "error": "no results"},
        {"source": "douban", "success": true}
      ]
    }
  }
}
```

### Testing Strategy

**Unit Tests:**
1. Circuit breaker state transitions
2. Provider interface implementations
3. Orchestrator fallback logic
4. Timeout enforcement

**Integration Tests:**
1. Full fallback chain execution
2. Circuit breaker recovery
3. API endpoint with mocked providers

**Test Scenarios:**
```go
var orchestratorTestCases = []struct {
    name           string
    tmdbResult     *SearchResult
    tmdbError      error
    doubanResult   *SearchResult
    expectedSource models.MetadataSource
}{
    {
        name:           "TMDb success",
        tmdbResult:     &SearchResult{Items: []MetadataItem{{Title: "Test"}}},
        expectedSource: models.MetadataSourceTMDb,
    },
    {
        name:           "TMDb fails, Douban succeeds",
        tmdbError:      errors.New("no results"),
        doubanResult:   &SearchResult{Items: []MetadataItem{{Title: "Test"}}},
        expectedSource: models.MetadataSourceDouban,
    },
}
```

**Coverage Targets:**
- Metadata package: ≥80%
- Metadata service: ≥80%

### Error Codes

Following project-context.md Rule 7:
- `METADATA_NO_RESULTS` - All sources returned no results
- `METADATA_ALL_FAILED` - All sources failed with errors
- `METADATA_TIMEOUT` - Fallback chain timeout exceeded
- `METADATA_CIRCUIT_OPEN` - All providers have open circuits

### Dependencies

**Story Dependencies:**
- Story 3.4 (Douban Web Scraper) - Douban provider stub will be replaced
- Story 3.5 (Wikipedia Fallback) - Wikipedia provider stub will be replaced

**Existing Dependencies:**
- TMDb service already implemented
- MetadataSource enum already defined

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.3]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-2]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-7]
- [Source: apps/api/internal/models/movie.go#MetadataSource]
- [Source: apps/api/internal/services/tmdb_service.go]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.1 & 3.2:**
- AI Provider abstraction can be integrated as a fallback option
- AI can generate alternative search keywords if all sources fail

**From Epic 2:**
- TMDb service patterns established
- Search handler patterns can be reused

## Dev Agent Record

### Agent Model Used

claude-opus-4-5-20251101

### Debug Log References

- Fixed import cycle between `metadata` and `services` packages by defining `TMDbSearcher` interface directly in metadata package
- Fixed circuit breaker test race conditions by using channels instead of slice appends for callback verification
- Added JSON tags to `FallbackStatus` and `SourceAttempt` structs for proper API response serialization

### Completion Notes List

1. **Task 1-3**: Created core metadata abstraction layer with provider interface, circuit breaker pattern (with configurable thresholds), and fallback chain orchestrator with <1s delay enforcement
2. **Task 4**: Adapted existing TMDb service as first provider in the chain, converting responses to common format
3. **Task 5-6**: Created Douban and Wikipedia provider stubs returning "not implemented" errors - ready for Stories 3.4 and 3.5
4. **Task 7**: Created MetadataService that wires orchestrator with all providers, supports enable/disable config per provider
5. **Task 8**: Created MetadataHandler with `/api/v1/metadata/search` and `/api/v1/metadata/providers` endpoints, wired up in main.go
6. Added configuration options: `ENABLE_DOUBAN`, `ENABLE_WIKIPEDIA`, `ENABLE_CIRCUIT_BREAKER`, `FALLBACK_DELAY_MS`, `CIRCUIT_BREAKER_FAILURE_THRESHOLD`, `CIRCUIT_BREAKER_TIMEOUT_SECONDS`

### File List

**New Files Created:**
- `apps/api/internal/metadata/provider.go` - MetadataProvider interface, SearchRequest, SearchResult, MetadataItem types
- `apps/api/internal/metadata/provider_test.go` - Interface and type tests
- `apps/api/internal/metadata/circuit_breaker.go` - Circuit breaker implementation with 3 states
- `apps/api/internal/metadata/circuit_breaker_test.go` - Circuit breaker state transition tests
- `apps/api/internal/metadata/orchestrator.go` - Fallback chain orchestrator with FallbackStatus tracking
- `apps/api/internal/metadata/orchestrator_test.go` - Orchestrator tests including timeout and cancellation
- `apps/api/internal/metadata/tmdb_provider.go` - TMDb adapter implementing MetadataProvider
- `apps/api/internal/metadata/tmdb_provider_test.go` - TMDb provider tests
- `apps/api/internal/metadata/douban_provider.go` - Douban stub provider
- `apps/api/internal/metadata/douban_provider_test.go` - Douban provider tests
- `apps/api/internal/metadata/wikipedia_provider.go` - Wikipedia stub provider with rate limiter
- `apps/api/internal/metadata/wikipedia_provider_test.go` - Wikipedia provider tests
- `apps/api/internal/services/metadata_service.go` - MetadataServiceInterface and implementation
- `apps/api/internal/services/metadata_service_test.go` - Service tests
- `apps/api/internal/handlers/metadata_handler.go` - HTTP handlers for metadata endpoints
- `apps/api/internal/handlers/metadata_handler_test.go` - Handler tests

**Modified Files:**
- `apps/api/internal/config/config.go` - Added metadata fallback chain configuration fields, loadBool helper, and config logging
- `apps/api/cmd/api/main.go` - Wired up MetadataService and MetadataHandler

### Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-01-23 | Code Review (claude-opus-4-5) | Added missing error codes `METADATA_ALL_FAILED` and `METADATA_CIRCUIT_OPEN` to provider.go |
| 2026-01-23 | Code Review (claude-opus-4-5) | Added metadata fallback chain config values to LogConfigSources() in config.go |
