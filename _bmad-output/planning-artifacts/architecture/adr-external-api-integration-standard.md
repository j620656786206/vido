# ADR: External API Integration Standard (Epic 12)

> **Status:** ACCEPTED
> **Date:** 2026-06-10
> **Deciders:** Alexyu (product owner), Winston (architect)
> **Origin:** retro-11-AI3 (Epic 11 retrospective action item — MED, prep)
> **Related PRD:** P2-022 (recommendations), P2-023 (streaming availability), P2-024 (trailers), P2-025 (Douban)
> **Related stories:** Epic 12 F-3, F-4, F-5, F-6 — sequenced AFTER this ADR, does NOT block F-1 / F-2
> **Codified as:** project-context.md **Rule 27 (External Integration Standard)**

---

## Context

Epic 12 (Rich Media Detail Page) stories **F-3 → F-6** are all third-party API
integrations:

| Story | External source | Nature |
|-------|----------------|--------|
| F-3 Recommendations | TMDB similar / recommendations endpoints | REST (authenticated) |
| F-4 Streaming availability | TMDB Watch Providers endpoint | REST (authenticated) |
| F-5 Trailer embeds | YouTube video (key supplied by TMDB) | client-side embed |
| F-6 Douban integration | Douban page scrape + review summary | HTML scrape (unauthenticated) |

Each of these shares the same four cross-cutting concerns — **rate limiting,
caching, graceful degradation on failure, and key/secret management.** The
Epic 11 retrospective (AI-3) flagged the risk: left unaligned, four stories would
each re-invent these concerns four different ways.

**The pattern already exists — it is just replicated, not shared.** Every
existing external client independently implements the same shape:

| Client | Rate limiter | Cache | Degradation | Error codes |
|--------|-------------|-------|-------------|-------------|
| `internal/tmdb` | `rate.Every(10s/40)`, burst 40 (TMDB's published 40 req / 10 s) | `cache.go` (24 h TTL) | `fallback.go` | `errors.go` → `TMDB_*` (Rule 7) |
| `internal/douban` | `0.5 rps` (1 req / 2 s), burst 1 | `cache.go` (search 24 h) | robots.txt compliance + UA rotation + exp-backoff `1s→16s ×2` + jitter `100–500ms`, `enabled` kill-switch, metrics | `DOUBAN_*` (Rule 7) |
| `internal/wikipedia` | `1 rps` | (client cache) | retry | `WIKIPEDIA_*` (Rule 7) |
| `internal/subtitle/providers/*` | per-provider `rate.Limiter` | SQLite | source-isolated parallel search | `SUBTITLE_*` |

This ADR does **not** introduce new infrastructure. It **codifies the existing,
battle-tested shape as the single standard** every Epic 12 integration must
follow, so the four stories converge instead of fragmenting. It reuses the
already-decided foundations:

- **AD #4** — Tiered cache (in-memory + SQLite), key pattern `{source}:{type}:{id}:{version}`
- **AD #6** — `AppError` + `slog` with `sanitizeAttr` (strips `api_key`/`key`/`token` from logs)
- **Rule 7** — `{SOURCE}_{ERROR_TYPE}` error-code system
- **graceful-degradation-execution-playbook.md** — circuit-breaker / fallback-chain consensus
- Worker-pool exponential backoff (`1s → 2s → 4s → 8s`) from AD #5

---

## Decision 1: The Five Pillars — every external integration MUST provide all five

Any code path that calls a third-party network service (or surfaces third-party
data to a story) MUST satisfy all five pillars. This is the checklist the
dev-story and code-review workflows enforce via Rule 27.

### Pillar 1 — Rate limiting (`golang.org/x/time/rate`)

- One `*rate.Limiter` **per upstream service**, constructed once at client init,
  reused for the process lifetime (Rule 14 — clients cached and reused, never
  per-request).
- Limit set to the upstream's **published / observed** ceiling, with the constant
  named and commented at the call site (precedent: `tmdb/client.go`
  `requestsPerInterval = 40`, `rateLimitInterval = 10s`).
- `limiter.Wait(ctx)` is the FIRST line of the request method, so context
  cancellation short-circuits before any socket is opened.

| Service | Limit | Source |
|---------|-------|--------|
| TMDB (all endpoints, incl. F-3/F-4) | 40 req / 10 s | published; **already enforced** by the shared `internal/tmdb` limiter |
| Douban (F-6) | 0.5 rps (1 req / 2 s) | conservative anti-scrape; **already enforced** |
| YouTube (F-5) | **n/a — no backend call** | see Decision 4 |

> **Key consequence:** F-3 and F-4 add ZERO new rate-limit budget — they ride the
> existing TMDB limiter. Re-using one client = one shared 40/10s bucket across
> details, search, discover, recommendations, watch-providers. No new bucket to
> tune or starve.

### Pillar 2 — Caching (tiered, per AD #4)

- Every upstream response cached under the canonical key
  `{source}:{type}:{id}:{version}` (e.g. `tmdb:recommendations:603:v1`,
  `tmdb:watchproviders:603:v1`, `douban:review:1234567:v1`).
- TTL by data volatility:

| Data | TTL | Rationale |
|------|-----|-----------|
| TMDB recommendations / similar (F-3) | 24 h | tracks AD #4 TMDB default; recs change slowly |
| TMDB watch providers (F-4) | 24 h | provider catalogs change daily at most |
| Douban review summary (F-6) | 24 h (search) / longer for stable pages | matches existing `douban/cache.go` |
| TMDB video keys (F-5) | rides existing movie/TV details cache | already cached via `GetMovieVideos` |

- A cache hit MUST NOT consult the rate limiter (cache is checked before
  `limiter.Wait`). This is what keeps the detail page < 1.5 s (Epic 12 success
  criterion) on warm content.

### Pillar 3 — Graceful degradation (fail-soft, never fail-page)

External data on the detail page is **enrichment, not core content.** A failure
in any pillar-3 source MUST degrade the section, never the page.

- **Per-section isolation:** F-3/F-4/F-5/F-6 each render independently. One source
  timing out / 404-ing / being blocked hides ONLY its section (empty-state or
  omitted), the rest of the detail page renders. (Mirrors the subtitle engine's
  source-isolation and Douban's existing `enabled` kill-switch.)
- **Bounded retry:** transient errors use the established exponential backoff
  (`1s → 2s → 4s → 8s`), capped, context-aware. Non-retryable (404, auth) fail
  immediately — no retry storm.
- **Stale-on-error:** if the upstream is down but a stale cache entry exists,
  serve stale rather than nothing (cache-aside already supports this — return the
  last-good value on fetch error).
- **Scraper-specific (F-6):** Douban already implements robots.txt compliance, UA
  rotation, block detection, and a global `enabled` flag — F-6 MUST route through
  the existing `internal/douban` client and inherit these. No new scrape path.

### Pillar 4 — Error codes (Rule 7, `{SOURCE}_{ERROR_TYPE}`)

- Backend-originated failures surface a Rule-7 `AppError`. **No new prefix is
  required for Epic 12:** F-3/F-4 reuse `TMDB_*` (`TMDB_RATE_LIMIT`,
  `TMDB_TIMEOUT`, `TMDB_NOT_FOUND`); F-6 reuses `DOUBAN_*`. F-5 makes no backend
  call → no error code.
- If a genuinely new failure mode appears (e.g. a watch-providers-specific
  shape), add the new code under the EXISTING `TMDB_` prefix and sync it with
  `code-review/instructions.xml` Step 3's Rule-7 wire-format grep (the
  retro-10-AI3 mandatory check) — do not invent a new prefix.

### Pillar 5 — Key / secret management

- API keys live in app settings / env, injected via the client's `ClientConfig`
  at construction (precedent: `tmdb.ClientConfig.APIKey`) — never hardcoded,
  never committed.
- Logs are sanitized by the existing `slog` `sanitizeAttr` (strips `api_key` /
  `key` / `token` and scrubs them from logged URLs — AD #6, NFR-S4). Any new
  client MUST log through `slog` so it inherits this.
- **Epic 12 adds no new secret:** TMDB key already exists; Douban needs none
  (scrape); YouTube needs none (Decision 4).

---

## Decision 2: Per-story integration mapping

| Story | Integration approach | New code | New infra |
|-------|---------------------|----------|-----------|
| **F-3** Recommendations | Extend `internal/tmdb` with `GetMovieRecommendations` / `GetMovieSimilar` / `GetTVRecommendations` / `GetTVSimilar` endpoint wrappers (mirror `movies.go` style). Cross-reference "已有" badges against the local library by TMDB id. | endpoint wrappers + service + handler | none — rides existing limiter/cache/errors |
| **F-4** Streaming availability | Extend `internal/tmdb` with `GetWatchProviders(movie/tv)`. **Note:** a `GetWatchProviders` was *removed as dead code in 11-1 (YAGNI)*; F-4 is the real consumer — re-add it now that a production path exists. | endpoint wrapper + service + handler | none — rides existing TMDB client |
| **F-5** Trailer embeds | **Backend already exposes `GetMovieVideos` / `GetTVShowVideos` → `VideosResponse`.** Surface the YouTube video `key` in the detail response; frontend embeds it (Decision 4). Fallback: link to TMDB video page when no YouTube key. | response field plumbing only | **none** — no new endpoint, no new key |
| **F-6** Douban integration | Route through existing `internal/douban` client; add a review-summary scrape method + direct-link field. Inherits robots.txt / rate-limit / degradation. | scrape method + service + handler | none — rides existing Douban client |

---

## Decision 3: NO shared `internal/externalapi` package (YAGNI)

We **codify the convention** (Rule 27) rather than refactoring the four existing
clients' replicated limiter/cache/fallback into a shared base package.

**Rationale:**
- The existing `tmdb` / `douban` / `wikipedia` clients already work and are fully
  tested. A cross-cutting refactor of multiple shipped, covered files carries
  real regression + test-rewrite cost and exceeds the minimal need of F-3..F-6.
- Three of the four Epic 12 integrations (F-3/F-4/F-5) ride the **already-shared**
  `internal/tmdb` client — the de-duplication that matters most is already done.
- The retro culture explicitly rewards strict YAGNI (11-1 deleted exported-but-
  unused `GetWatchProviders`). A speculative abstraction layer is the opposite of
  that.

**Future-enhancement trigger (recorded, not acted on):** if a *fifth* independent
external client appears and again hand-rolls the same limiter+cache+fallback
triplet, that third duplication is the signal to extract a shared
`internal/externalapi` base — not before.

---

## Decision 4: YouTube via client-side embed, NO YouTube Data API

F-5 trailers are embedded **client-side** using the YouTube video `key` that TMDB
already returns from its `/videos` endpoint (already wrapped by
`GetMovieVideos` / `GetTVShowVideos`).

- Frontend renders `https://www.youtube-nocookie.com/embed/{key}` in an `iframe`
  (privacy-enhanced, no-cookie domain).
- **No backend YouTube call, no YouTube Data API key, no YouTube quota, no
  YouTube rate limiter.** YouTube is therefore exempt from Pillars 1/4/5 — it has
  no backend surface.
- **Fallback chain (Pillar 3):** YouTube key present → embed. No YouTube key but a
  TMDB video link exists → link out to the TMDB video page. Neither → omit the
  trailer section (empty-state).

This is the "boring technology" choice: TMDB is already the single source of the
video key, so a second provider integration buys nothing.

---

## Consequences

### Positive
- Four stories converge on one proven shape; reviewers check one Rule-27 checklist
  instead of re-deriving rate-limit/cache/degradation correctness four times.
- F-3/F-4/F-5 add **zero** new external-service infra (no new key, limiter, quota,
  or error prefix) — they extend the existing `internal/tmdb` client.
- Detail page stays < 1.5 s on warm content (cache-before-limiter) and never
  fails the whole page on a single dead source (per-section degradation).
- The standard is **enforced, not aspirational** — per Epic 11 Retro Insight 1
  (passive docs rot, codified checks stick), it lands as Rule 27 with a
  dev-story / code-review hook, mirroring the AC-Drift / Rule-7 checks that
  demonstrably stuck.

### Negative / trade-offs
- The four existing clients keep their replicated limiter/cache code (Decision 3).
  Accepted: de-dup deferred until a real third duplication appears.
- Douban review-summary scraping remains inherently fragile (HTML structure can
  change). Mitigated by Pillar 3 fail-soft + the existing block-detection metrics.

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| F-4 watch-providers re-introduces the dead-code that 11-1 removed | Low | It now has a real consumer (F-4); YAGNI is satisfied. Add only what F-4 renders. |
| Douban page-structure drift breaks F-6 review summary | Medium | Per-section fail-soft (Pillar 3) — empty review section, page still renders; metrics surface block/timeout rate. |
| Shared TMDB 40/10s bucket saturates when detail page fans out (details+recs+providers+videos in parallel) | Medium | All four are cached 24 h; first warm load primes them. If contention observed, raise burst or stagger calls — single tuning point, not four. |
| YouTube embed blocked by network / ad-block | Low | Pillar-3 fallback to TMDB video link, then omit. |

---

## Alternatives Considered

1. **Build a shared `internal/externalapi` base package now** — Rejected (Decision 3): cross-cutting refactor of shipped, tested clients exceeds F-3..F-6's minimal need; the YAGNI culture rewards deferring it to a real third duplication.
2. **Add a YouTube Data API integration for F-5** — Rejected (Decision 4): TMDB already supplies the video key; a second provider adds a key, quota, limiter, and failure mode for zero added capability.
3. **New `YOUTUBE_` / watch-provider error prefix** — Rejected (Pillar 4): F-5 has no backend call; F-3/F-4 reuse `TMDB_*`. Inventing prefixes violates Rule 7's `SOURCE = uppercase(package)` uniformity.
4. **Per-story ad-hoc integration (the no-ADR baseline)** — Rejected: this is precisely the four-way fragmentation retro-11-AI3 exists to prevent.
5. **Fail the detail page when an enrichment source is down** — Rejected: external data is enrichment, not core content; Pillar 3 mandates per-section degradation.

---

## Implementation Sequence

1. **This ADR + Rule 27** land first (retro-11-AI3 deliverable) — no code.
2. F-1 / F-2 proceed independently (no dependency on this ADR).
3. F-3 → F-4 → F-5 → F-6 each cite Rule 27 in their Dev Notes and satisfy the
   five-pillar checklist; code-review verifies it.
```
