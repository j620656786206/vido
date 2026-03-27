# Story retro-8-TD3: Per-Provider Rate Limiter

Status: done

## Story

As a system administrator,
I want each subtitle provider to have its own proactive rate limiter,
so that batch operations don't trigger API rate limit errors.

## Acceptance Criteria

1. Zimuku provider has a `rate.Limiter` with appropriate limits (1 req/s, burst 1)
2. OpenSubtitles provider has a proactive `rate.Limiter` (5 req/s, burst 5) complementing existing 429 retry handling
3. All three providers use the `limiter.Wait(ctx)` pattern before API calls, matching Assrt's existing pattern
4. Rate limiter unit tests exist for Zimuku provider
5. Proactive rate limiter unit tests exist for OpenSubtitles provider (separate from existing 429 retry test)
6. Existing Assrt rate limiter test (`TestAssrtProvider_RateLimiter`) continues to pass
7. Batch processor's `DelayBetweenItems` (3s) remains unchanged as a secondary safeguard

## Tasks / Subtasks

- [x] Task 1: Add `rate.Limiter` to Zimuku provider (AC: 1, 3)
  - [x] 1.1 Add `rate.Limiter` field and constants to `zimuku.go` (1 req/s, burst 1)
  - [x] 1.2 Initialize limiter in `NewZimukuProvider` constructor
  - [x] 1.3 Add `limiter.Wait(ctx)` call before each HTTP request in `Search()` and `Download()`
- [x] Task 2: Add proactive `rate.Limiter` to OpenSubtitles provider (AC: 2, 3)
  - [x] 2.1 Add `rate.Limiter` field and constants to `opensub.go` (5 req/s, burst 5)
  - [x] 2.2 Initialize limiter in `NewOpenSubProvider` constructor
  - [x] 2.3 Add `limiter.Wait(ctx)` call before each HTTP request in `Search()` and `Download()`
  - [x] 2.4 Keep existing 429 retry logic intact — proactive limiter is defense layer 1, retry is defense layer 2
- [x] Task 3: Write rate limiter unit tests for Zimuku (AC: 4)
  - [x] 3.1 Add `TestZimukuProvider_RateLimiter` — verify requests are throttled to 1 req/s
  - [x] 3.2 Follow the same test pattern as `TestAssrtProvider_RateLimiter`
- [x] Task 4: Write proactive rate limiter unit tests for OpenSubtitles (AC: 5)
  - [x] 4.1 Add `TestOpenSubProvider_RateLimiter` — verify requests are throttled to 5 req/s
  - [x] 4.2 Ensure existing `TestOpenSubProvider_RateLimiting429` still passes alongside new test
- [x] Task 5: Run full test suite to verify no regressions (AC: 6, 7)
  - [x] 5.1 Run `nx test api` — all provider tests pass
  - [x] 5.2 Verify `TestAssrtProvider_RateLimiter` unchanged and passing
  - [x] 5.3 Verify batch processor `DelayBetweenItems` is still 3 seconds (no changes to `batch.go`)

## Dev Notes

### Rate Limit Values

| Provider | Rate | Burst | Rationale |
|----------|------|-------|-----------|
| Assrt | 2 req/s | 2 | Existing — do not change |
| OpenSubtitles | 5 req/s | 5 | API with auth; free tier allows ~5 req/s, paid ~40 req/10s |
| Zimuku | 1 req/s | 1 | Web scraping target — be conservative to avoid IP bans |

### Implementation Pattern (follow Assrt exactly)

All three providers should follow the identical pattern from `assrt.go`:

```go
// Constants
const (
    zimukuRateLimit = 1 // requests per second
    zimukuBurstLimit = 1
)

// Struct field
type ZimukuProvider struct {
    // ...existing fields...
    limiter *rate.Limiter
}

// Constructor
func NewZimukuProvider(...) *ZimukuProvider {
    return &ZimukuProvider{
        // ...existing fields...
        limiter: rate.NewLimiter(rate.Limit(zimukuRateLimit), zimukuBurstLimit),
    }
}

// Before each HTTP request
if err := p.limiter.Wait(ctx); err != nil {
    return nil, fmt.Errorf("rate limiter: %w", err)
}
```

### Package Dependency

`golang.org/x/time/rate` is already in `go.mod` — no new dependencies needed.

### Defense-in-Depth Strategy

The rate limiting architecture has three layers:

1. **Per-provider `rate.Limiter`** (this story) — prevents exceeding provider-specific API limits
2. **429 retry with backoff** (OpenSubtitles only, existing) — recovers if rate limit is hit despite proactive limiting
3. **Batch `DelayBetweenItems` = 3s** (existing) — coarse inter-item delay as final safeguard

Do NOT modify `DelayBetweenItems` in `batch.go` — it serves a different purpose (overall system pacing, not per-provider limiting).

### What NOT to Change

- Do NOT modify Assrt provider's rate limiter — it's already correct
- Do NOT modify batch processor's `DelayBetweenItems`
- Do NOT add rate limiting to the `SubtitleProvider` interface — keep it an internal concern of each provider
- Do NOT remove OpenSubtitles' existing 429 retry logic

### References

- [Source: apps/api/internal/subtitle/providers/assrt.go] — Reference implementation (lines 22-23, 47)
- [Source: apps/api/internal/subtitle/providers/opensub.go] — Needs proactive limiter (has 429 retry at line 36)
- [Source: apps/api/internal/subtitle/providers/zimuku.go] — Needs rate limiter (none currently)
- [Source: apps/api/internal/subtitle/batch.go] — `DelayBetweenItems` — do not change
- [Source: apps/api/internal/subtitle/providers/provider.go] — Provider interface — do not change
- [Source: epic-8-retro-2026-03-25.md#TD3] — Retro action item origin

## Change Log

- 2026-03-26: Story created — ready-for-dev
- 2026-03-27: Implementation complete — Zimuku 1 req/s, OpenSub 2 req/s (reduced from 5 per CR), removed auth double-token bug, all tests pass
- 2026-03-27: QA pass — all HTTP paths guarded, pattern consistent with Assrt
- 2026-03-27: CR fixes applied — rate constant, error format, auth mutex contention
- 2026-03-27: Status → done
