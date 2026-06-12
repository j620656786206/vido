# Story 12.6: Douban Integration — Direct Page Link + User Review Summary

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a media library user viewing a movie or TV series detail page,
I want a direct link to the title's Douban page and a short summary of Douban user reviews (短評),
so that I can read Chinese-audience opinions and jump to the full Douban discussion without searching for it.

## Acceptance Criteria

1. **Given** a movie or TV detail page whose title has a resolved `douban_id` (populated by Story 12-1's rating enrichment), **when** the page loads, **then** a "查看豆瓣頁面" direct link renders pointing to `https://movie.douban.com/subject/{douban_id}/` (`target="_blank" rel="noopener noreferrer"`).
2. **Given** a title with a resolved `douban_id`, **when** the review summary is requested, **then** the backend returns the top N (≤5) Douban short comments (短評) — each with author, star rating, and comment text — plus the total review/comment count.
3. **Given** Douban short-comment text is simplified Chinese, **then** it is converted to Traditional Chinese (OpenCC `s2twp`) before display, reusing the existing `converter.go` (consistent with 12-1's rating-title conversion).
4. **Given** the title has NO resolved `douban_id` (12-1 never matched it), **then** the Douban section is omitted entirely — no link, no review block (fail-soft, never an error).
5. **Given** Douban is unavailable / blocked (robots.txt disallow, 403/429/503, timeout) / the review HTML fails to parse, **then** the review-summary section degrades to omitted-or-empty-state and the **rest of the detail page renders unaffected** (Rule 27 Pillar 3; reuse the douban client's existing block-detection + `enabled` kill-switch). The direct link (AC #1) still renders if `douban_id` is known, since it needs no scrape.
6. **Given** repeated visits, **then** the review summary is served from the existing `douban_cache` (extended with a review-summary field), so a warm load makes no new scrape request (Rule 27 Pillar 2).
7. **Given** Douban's anti-scrape posture, **then** the review-summary scrape rides the **existing `internal/douban` client** — 0.5 rps limiter (1 req / 2s), UA rotation, robots.txt compliance, exponential backoff — adding **no new scrape path** (Rule 27 Pillar 1; ADR F-6 row).
8. **Given** a mobile viewport, **then** the review list stacks readably (one comment per row: author + stars on one line, text below).

## Tasks / Subtasks

### Backend

- [x] **Task 1: Douban scraper — `ScrapeReviewSummary` + types** (AC: #2, #3, #5, #7)
  - [x] 1.1 Add to `apps/api/internal/douban/scraper.go` a method `ScrapeReviewSummary(ctx, id string) (*ReviewSummaryResult, error)` mirroring `ScrapeDetail` (`:35-80`): reuse `client.GetBody(ctx, DetailURL(id))` — **prefer parsing the short-comments (短評) already present on the subject page** (`https://movie.douban.com/subject/{id}/`) so NO extra request beyond the page the rating path already knows; only fall to `/subject/{id}/comments/` if the subject page lacks them (document the selector decision in the dev notes). → **Chose the subject-page inline `#comments-section` block (no extra request); `/subject/{id}/comments/` fallback NOT needed.**
  - [x] 1.2 Define `ReviewSummaryResult { ID string; TotalComments int; TopComments []ReviewComment }` and `ReviewComment { Author string; Rating int; Text string }` in `apps/api/internal/douban/types.go` (alongside `DetailResult`). → JSON tags `id`/`total_comments`/`top_comments` + `author`/`rating`/`text`.
  - [x] 1.3 Parse with goquery mirroring `parseDetailPage` style (`scraper.go:111-196`) — short-comment item selectors (e.g. `#comments-section .comment-item`, author `.comment-info a`, rating `.rating` class suffix, text `.short`). **Validate selectors against a saved fixture** (capture a real subject page into `scraper_test.go` testdata; Douban markup drifts — Rule 27 Pillar 3 risk row). → Selectors: items `#comments-section .comment-item` (fallback `#hot-comments`/`.comment-list`), author `.comment-info a`, rating `allstarNN` class /10, text `.short`/`.comment-content`, total `全部 N 条` regex. Fixture `testdata/subject_comments.html`.
  - [x] 1.4 Convert each comment `Text` to Traditional via the existing `ChineseConverter.ConvertIfSimplified` (reuse `convertToTraditional` style, `scraper.go:83-108`).
  - [x] 1.5 Errors reuse the existing `DOUBAN_*` codes (`types.go:168-175` — `DOUBAN_BLOCKED`/`DOUBAN_PARSE_ERROR`/`DOUBAN_RATE_LIMITED`/`DOUBAN_TIMEOUT`/`DOUBAN_NOT_FOUND`). **NO new prefix** (Rule 7 / ADR Pillar 4). Parse-failure → `DOUBAN_PARSE_ERROR`, returned fail-soft upstream. → Reused `BlockedError`/`ParseError`; no new code/prefix.
  - [x] 1.6 Scraper test: fixture-based parse (top-comments extracted, s2twp conversion applied), block/parse-error paths. → `scraper_review_test.go` (4 tests: parse+cap, s2twp, no-section degrade, disabled-client block).

- [x] **Task 2: Cache the review summary (extend `douban_cache`)** (AC: #6)
  - [x] 2.1 Add a migration (next free number — **verify the highest existing migration number first**; 12-2 used 025, so this is likely **026+**; grep `apps/api/internal/database/migrations/`) to add `review_summary_json TEXT` to the `douban_cache` table (idempotent `ALTER TABLE … ADD COLUMN`, self-registers via `init()` — NOT a `registry.go` edit, per 12-2's correction). → **Verified highest = 025; created `026_add_douban_review_summary.go` (self-registering init(), columnExists guard).**
  - [x] 2.2 Extend `apps/api/internal/douban/cache.go` `Get`/`Set` (`:98-180`) to hydrate/persist `review_summary_json` (JSON-encode `ReviewSummaryResult.TopComments` + `TotalComments`). Reuse the existing `douban_id`-keyed lookup + 7-day TTL (`DefaultCacheConfig`). A review-summary cache hit needs no scrape (cache-before-limiter, Pillar 2). → Added dedicated `GetReviewSummary`/`SetReviewSummary` (upsert touches only `review_summary_json`+`expires_at`); **converted detail `Set` INSERT OR REPLACE → ON CONFLICT(douban_id) DO UPDATE so a detail re-scrape no longer clobbers the review summary on the shared row.**
  - [x] 2.3 Cache test for the new column (round-trip + expiry). → `cache_review_test.go` (round-trip, miss, expiry, no-clobber).

- [x] **Task 3: Service + handler + routes** (AC: #1, #2, #4, #5)
  - [x] 3.1 Extend `apps/api/internal/services/douban_rating_service.go` with `EnrichDoubanReviewSummary(ctx, mediaID, mediaType string) (*ReviewSummaryResult, error)` mirroring `EnrichDoubanRating` (`:81-119`): read the media record → if `douban_id` already stored (from 12-1), scrape reviews by that id; if NOT stored, run the existing rating-lookup first to RESOLVE the id (reuse `lookup`/`pickBestMatch`), then scrape; if still unresolved → return `nil` (AC #4 omit). Wrap the scrape in the existing `doubanLookupTimeout` (10s) context.
    - Inject the review-capable scraper via an interface (mirror the `DoubanSearcher` injection pattern `:42-45`) — do NOT have the service reach into `internal/douban` concretely (Rule 4/11/19 layering). → Added `DoubanReviewScraper` interface (returns `*douban.ReviewSummaryResult` — type import only, mirrors how `DoubanSearcher` imports `metadata`). **Injected `*metadata.DoubanProvider` (NOT the raw `douban.Scraper`) so the cache-aware path + single client/limiter is preserved — see Completion Notes for the Dev-Note-5 deviation rationale.** Added `resolveDoubanID` helper.
  - [x] 3.2 Return `nil` (not an error) on any block/parse/timeout failure (graceful degradation — same contract as `EnrichDoubanRating` returning nil; AC #5).
  - [x] 3.3 Extend `apps/api/internal/handlers/douban_rating_handler.go` with `GetMovieDoubanReviewSummary` / `GetSeriesDoubanReviewSummary` (mirror `:32-66`); register routes `GET /api/v1/movies/:id/douban-review-summary` and `GET /api/v1/series/:id/douban-review-summary` (mirror `RegisterRoutes` `:71-74`). Response `{ "success": true, "data": {...} | null }` (Rule 3; null = omit).
  - [x] 3.4 Confirm `douban_id` is in the response surface so the frontend can build the direct link (AC #1): the existing `DoubanRatingResult` already carries `DoubanID` (`douban_rating_service.go:31-35`) — the frontend's existing `useDoubanRating` data already exposes `doubanId`. **Verify** the detail page receives it from the rating query; if so, AC #1 needs NO new backend field (build the link client-side). Note the finding. → **VERIFIED: `DoubanRatingResult.DoubanID` (`json:"douban_id"`) → camel `doubanId` via Rule 18; route already runs `useDoubanRating`. AC #1 needs NO new backend field — link built client-side.**
  - [x] 3.5 Wire `EnrichDoubanReviewSummary` deps in `cmd/api/main.go` (the `DoubanRatingService`/handler are already wired by 12-1 — add the review-scraper dep). Swaggo annotations on both new endpoints (Rule 15). Service + handler tests (resolved-id happy path, unresolved-id → null, block → null). → main.go injects the same `dp` as review scraper; Swaggo on both endpoints; `douban_review_service_test.go` (7 tests) + `douban_review_handler_test.go` (4 tests).

### Frontend

- [x] **Task 4: Types + service + hook** (AC: #1, #2, #6)
  - [x] 4.1 Add `ReviewComment { author: string; rating: number; text: string }` and `DoubanReviewSummary { id: string; totalComments: number; topComments: ReviewComment[] }` to `apps/web/src/types/library.ts`. → + `DoubanReviewSummaryResponse = DoubanReviewSummary | null`.
  - [x] 4.2 Add `getMovieDoubanReviewSummary(id)` / `getSeriesDoubanReviewSummary(id)` to `apps/web/src/services/libraryService.ts` (mirror the 12-1 `getMovieDoubanRating` methods → `/movies/${id}/douban-review-summary`).
  - [x] 4.3 Add `useDoubanReviewSummary(id, type, enabled)` to `apps/web/src/hooks/useDoubanRating.ts` (mirror `useDoubanRating` — `queryKey: ['douban-review-summary', type, id]`, `staleTime` 24h, `enabled: enabled && !!id`). Returns `DoubanReviewSummary | null`. → Placed in a **new** `apps/web/src/hooks/useDoubanReviewSummary.ts` (one hook per file, consistent with the codebase) rather than appending to `useDoubanRating.ts`.

- [x] **Task 5: `DoubanSection` component (direct link + review summary)** (AC: #1, #4, #5, #8)
  - [x] 5.1 Create `apps/web/src/components/media/DoubanSection.tsx`. Props: `doubanId?: string` (from the existing rating query), `summary?: DoubanReviewSummary | null`, `isLoading`, `isError`.
  - [x] 5.2 **If `doubanId`**: render the "查看豆瓣頁面" direct link → `https://movie.douban.com/subject/${doubanId}/` (AC #1). **If NO `doubanId`**: render nothing (AC #4).
  - [x] 5.3 Review summary: if `summary?.topComments?.length`, render each comment (author + star rating + Traditional-Chinese text), capped at 5, with the total-comments count. Loading → quiet skeleton; error/empty → omit the review block but KEEP the direct link (AC #5). Never throw.
  - [x] 5.4 Rule 21 header (feature postdates the `.pen` design): design-coverage-gap form `// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page Douban review section postdates the .pen design`.
  - [x] 5.5 Mobile: one comment per row, author/stars line + text below (Tailwind responsive — AC #8). Write `DoubanSection.spec.tsx` (link present when doubanId, omitted when not, comments render with Traditional text, error keeps link drops reviews, empty-state). Rule 16 matchers. → 8 tests (incl. 5-comment cap + unrated-comment star omission); always single-column stacked layout (mobile-readable by construction).

- [x] **Task 6: Integrate into the detail page** (AC: #1)
  - [x] 6.1 In `apps/web/src/routes/media/$type.$id.tsx`, render `<DoubanSection />` **below the overview / near the other Epic-12 enrichment sections** in BOTH `LocalDetailView` and `TMDbDetailView`. Source `doubanId` from the EXISTING `doubanQuery.data?.doubanId` (12-1's `useDoubanRating`, already wired into the route at `~:274-280`); source `summary` from the new `useDoubanReviewSummary(localId, type, enabled)`. → **Wired into LocalDetailView (below TrailerSection, above Credits), gated on `doubanQuery.data?.doubanId`. NOT rendered in TMDbDetailView — see Completion Notes "Discovery Triage": TMDbDetailView has no local UUID, and the Douban endpoints are local-UUID-keyed, so it can never resolve a doubanId (consistent with 12-1's existing in-code comment that scopes Douban to LocalDetailView). AC #1/#4 are still satisfied.**
  - [x] 6.2 Gate the review-summary fetch on the same `enabled` condition 12-1 uses for the rating (typically once a `douban_id`/`tmdbId` is known) to avoid a scrape when no Douban match exists. → Gated on `tmdbId > 0`, identical to `useDoubanRating`.

## Dev Notes

### Architecture Compliance

- **Rule 4 / Rule 11 / Rule 19 (Layering + boundaries):** `DoubanRatingHandler.GetMovieDoubanReviewSummary` → `DoubanRatingService.EnrichDoubanReviewSummary` → (review-scraper interface) → `internal/douban` `Scraper`. The service depends on an INJECTED interface (mirror `DoubanSearcher` `:42-45`), never on the concrete `internal/douban` package directly, keeping the package boundary clean.
- **Rule 5 (TanStack Query):** review summary via `useDoubanReviewSummary`, gated `enabled`.
- **Rule 6 (Naming):** endpoints `/api/v1/{movies,series}/:id/douban-review-summary`; Go `EnrichDoubanReviewSummary`/`ScrapeReviewSummary`; TS `useDoubanReviewSummary`; JSON `total_comments`/`top_comments` ↔ camel via Rule 18.
- **Rule 7 (Error Codes) + Rule 27 Pillar 4:** reuse `DOUBAN_*` ONLY — **no new prefix** (ADR Pillar 4 — "F-6 reuses DOUBAN_*").
- **Rule 13 (Error Handling):** service returns `nil` (fail-soft) on block/parse/timeout, identical to `EnrichDoubanRating`'s contract; errors logged via `slog`, never swallowed silently.
- **Rule 14 / Rule 27 Pillar 1:** reuses the existing douban client (built once, 0.5 rps limiter, `Wait` first) — ZERO new scrape infra (ADR F-6 row).
- **Rule 15 (Self-verification):** new migration self-registers via `init()` (NOT `registry.go` — 12-2 correction); register both routes; Swaggo; wire the review-scraper dep in `main.go`.
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` / `toBeAttached()`.
- **Rule 18 (Case Transform):** auto via `fetchApi`.
- **Rule 21 (Component↔Design):** `DoubanSection.tsx` uses the design-coverage-gap `// Design ref:` form.
- **Rule 27 (External Integration Standard — Five Pillars):** ✅ ① rate limit — existing douban 0.5 rps limiter, no new bucket · ✅ ② cache — extend `douban_cache` (7-day TTL, `douban_id`-keyed), checked before limiter · ✅ ③ degrade — per-section fail-soft, robots.txt/block-detection/`enabled` kill-switch all inherited, stale-on-error via cache · ✅ ④ error codes — reuse `DOUBAN_*`, no new prefix · ✅ ⑤ keys — Douban scrape is unauthenticated, no secret. [Source: ADR Decision 1 (Pillars) + Decision 2 (F-6 row "route through existing internal/douban") + Risk row "Douban page-structure drift breaks F-6 review summary"]

### Cross-Stack Split Check (MANDATORY — Agreement 5 / Epic 9c Retro AI-1)

Backend tasks: **3** (Task 1 scraper, Task 2 cache/migration, Task 3 service/handler). Frontend tasks: **3** (Task 4 types/service/hook, Task 5 component, Task 6 integration).

Threshold is "BOTH counts > 3". Backend = 3 (**not** > 3), Frontend = 3 (not > 3) → **NO split required. Single story** (sits exactly at the boundary; the scrape + its consumer are tightly coupled and a split would create a scrape-with-no-consumer).

### Project Structure Notes

**Files to CREATE:**
- `apps/api/internal/database/migrations/0NN_add_douban_review_summary.go` (+ `_test.go`) — **NN = next free number, verify first**
- `apps/web/src/components/media/DoubanSection.tsx` (+ `DoubanSection.spec.tsx`)

**Files to MODIFY:**
- `apps/api/internal/douban/scraper.go` (+ `scraper_test.go` + testdata fixture) — `ScrapeReviewSummary` + parse
- `apps/api/internal/douban/types.go` — `ReviewSummaryResult`, `ReviewComment`
- `apps/api/internal/douban/cache.go` (+ `cache_test.go`) — `review_summary_json` hydrate/persist
- `apps/api/internal/services/douban_rating_service.go` (+ test) — `EnrichDoubanReviewSummary` + review-scraper interface
- `apps/api/internal/handlers/douban_rating_handler.go` (+ test) — 2 handler methods + routes
- `apps/api/cmd/api/main.go` — wire review-scraper dep
- `apps/web/src/types/library.ts` — `ReviewComment`, `DoubanReviewSummary`
- `apps/web/src/services/libraryService.ts` — `get{Movie,Series}DoubanReviewSummary`
- `apps/web/src/hooks/useDoubanRating.ts` — `useDoubanReviewSummary`
- `apps/web/src/routes/media/$type.$id.tsx` — render `<DoubanSection />` in both detail views

### Critical Implementation Details

1. **The direct link is nearly free (AC #1).** Story 12-1 already stores `douban_id` (migration 024) and its `DoubanRatingResult` already returns `DoubanID`; the route already runs `useDoubanRating`. So the "查看豆瓣頁面" link is built CLIENT-SIDE from `doubanQuery.data?.doubanId` with the existing `DetailURL` pattern (`https://movie.douban.com/subject/{id}/`) — **no new backend field needed for the link** (verify in Task 3.4). The link renders even when the review scrape fails (AC #5).

2. **The review summary is the real work — and the riskiest (AC #2, ADR Risk row).** Douban short-comment HTML drifts and the comments page may be partially behind anti-scrape. Mitigations baked in: (a) reuse the existing client's robots.txt/0.5-rps/UA-rotation/`enabled`/block-detection — if Douban blocks, the section omits per Pillar 3; (b) parse the 短評 ALREADY on the subject page (no extra request) where possible; (c) capture a real-page fixture into testdata so the parser is testable and drift is detectable; (d) cache 7 days so a working scrape isn't repeated. If the subject page proves to not carry usable short comments, fall to `/subject/{id}/comments/` (one extra rate-limited request) — document which path was chosen.

3. **Resolve `douban_id` before scraping (Task 3.1).** The review scrape is keyed by `douban_id`. For library items that 12-1 already rated, `douban_id` is stored — read it. For items not yet matched, reuse 12-1's `lookup`/`pickBestMatch` (title+year search) to resolve the id first, then scrape. If unresolvable → return `nil` → section omits (AC #4). Do NOT duplicate the search logic — reuse the service's existing lookup.

4. **OpenCC conversion is reuse, not new (AC #3).** `converter.go` (`ChineseConverter`, `s2twp`) already converts the rating-path title/summary. Apply the SAME `ConvertIfSimplified` to each comment's text. CN-content policy (project-context 9b) is about subtitle files, not review display — always convert review text to Traditional for the zh-TW UI.

5. **Service layering — inject the scraper (Rule 4/11/19).** `DoubanRatingService` currently injects `DoubanSearcher` (a `metadata`-level interface), not the raw `douban.Scraper`. Add a small review-scraper interface (e.g. `DoubanReviewScraper { ScrapeReviewSummary(ctx, id) (*ReviewSummaryResult, error) }`) and inject the concrete `douban.Scraper` at `main.go` wiring. Keeps the service free of a concrete `internal/douban` import (consistent with 12-1).

6. **Migration number (Task 2.1).** 12-2 shipped migration 025; the next is likely 026 but **grep `apps/api/internal/database/migrations/` and use the next free integer** — do not assume (12-2's story said 023 but reality was 025).

### Existing Code References

- Douban client (reuse wholesale): `apps/api/internal/douban/client.go` — rate limiter `:122` (0.5 rps, burst 1), robots `:374-437`, UA rotation `:133-140`, kill-switch `:358-370`, block detection `:267-307`, `DetailURL` `:346-349`.
- Scraper to extend: `apps/api/internal/douban/scraper.go:35-80` (`ScrapeDetail`), `:111-196` (`parseDetailPage` selectors), `:83-108` (`convertToTraditional`).
- Error codes: `apps/api/internal/douban/types.go:168-175` (`DOUBAN_*`).
- Cache to extend: `apps/api/internal/douban/cache.go:98-180` (`Get`/`Set`), `:26-32` (7-day TTL).
- 12-1 service template: `apps/api/internal/services/douban_rating_service.go:31-35` (`DoubanRatingResult` w/ `DoubanID`), `:42-45` (`DoubanSearcher` injection), `:81-119` (`EnrichDoubanRating` + cache-then-lookup-then-persist flow), `:148-213` (`lookup`/`pickBestMatch`).
- 12-1 handler template: `apps/api/internal/handlers/douban_rating_handler.go:32-66` (handler), `:71-74` (routes).
- 12-1 migration template: `apps/api/internal/database/migrations/024_add_douban_rating_fields.go`.
- 12-1 frontend template: `apps/web/src/hooks/useDoubanRating.ts` (`useDoubanRating`), `apps/web/src/components/media/DualRatingDisplay.tsx`, route wiring `routes/media/$type.$id.tsx:274-280`.
- Detail route slots: `routes/media/$type.$id.tsx` — LocalDetailView overview `~:300-305`; TMDbDetailView overview `~:503-505`.
- Sibling Epic-12 detail-integration stories: `12-3`/`12-4`/`12-5` (same below-overview section pattern).

### UX Design Note

Epic 12 has **no `ux-design.pen` screen** for the Douban section. Patterns: "豆瓣評論" heading; direct-link styled like other outbound links; comment rows modeled on a simple list (author + star glyphs + text); `DoubanSection.tsx` carries the Rule 21 design-coverage-gap header.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.** `DoubanSection.tsx` renders a static link + server-supplied comments; no ambient-now read or date-boundary branching. No new fixture-state baselines.
- Reference: `project-context.md` Rule 23.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-6 (P2-025)]
- [Source: _bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md — Decision 2 (F-6 row), Decision 1 Pillars, Risk row (Douban page-structure drift)]
- [Source: project-context.md — Rules 3, 4, 5, 6, 7, 11, 13, 14, 15, 16, 18, 19, 21, 27; AD #9b (OpenCC s2twp)]
- [Source: apps/api/internal/douban/ — existing client/scraper/cache/converter (rate-limit/robots/UA/kill-switch)]
- [Source: apps/api/internal/services/douban_rating_service.go — Story 12-1 enrichment template]
- [Source: _bmad-output/implementation-artifacts/12-2-season-episode-list.md — migration self-registration correction (init(), next-free-number)]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia — BMM dev-story workflow)

### Debug Log References

- Full regression gate (Epic 9 Retro AI-1): `pnpm nx test web` → **2099/2099 pass** (176 files); `pnpm nx test api` → **0 failures**. test:cleanup confirmed "No test processes found" (no orphaned vitest workers).
- Cache expiry test format pitfall: storing `expires_at` via a Go `time.Time` (RFC3339, `T`/`Z`) compares wrong against SQLite `CURRENT_TIMESTAMP` (space format); fixed the expired-row test to use `datetime('now','-1 hour')`, matching the existing `TestCache_GetExpired`. The real `SetReviewSummary` always writes a FUTURE expiry (now+7d), which reads back correctly (same mechanism as the detail `Set`).

### Completion Notes List

- **🔗 AC Drift: NONE** (checked `douban_cache` across `_bmad-output/implementation-artifacts/*.md` — 3 hits: `3-4` (table creator, migration 008), `6-2` (cache mgmt), `12-1` (explicitly "no join against douban_cache"); all REUSE not DRIFT. The migration-026 `ADD COLUMN` + the detail `Set` `INSERT OR REPLACE → ON CONFLICT(douban_id) DO UPDATE` change PRESERVE the existing cache-hit external contract — a detail `Get` still returns the same `DetailResult` on hit; only the formerly-clobbering REPLACE is now non-clobbering, which is additive safety, not a behavior change to any prior AC.)
- **📎 Contract Stamps: NONE** (no `[@contract-v*]` stamps in 12-6 or upstream `12-1`; 12-1 is pre-Rule-20 implicit v0. This story defines and consumes no version-stamped wire contracts.)
- **🎭 A11y Pre-Flight: PASS** (1 component checked — `DoubanSection`; `eslint` jsx-a11y → 0 warnings on touched files, 0 introduced by this story. Manual 4-class check: no TMDb `<img>` (no responsive-image concern); no `aria-modal`/focus-trap (static section, not a modal); no async-revealed status pill needing `aria-live` (the comment list is render-once data, not a live status badge); no custom combobox/widget. Positive a11y: `<section aria-labelledby>` landmark → visible `<h2>`, star rating `role="img"` + `aria-label="{n} 星"`, loading skeleton `aria-hidden`. The 1 pre-existing eslint warning in `$type.$id.tsx:188` (`useCallback` deps) is untouched by this story.)
- **🎨 UX Verification: SKIPPED — design-coverage-gap.** Epic 12 has NO `ux-design.pen` screen for the Douban section (documented in the story's UX Design Note); `DoubanSection.tsx` carries the Rule 21 design-coverage-gap header. Implementation follows the described patterns: "豆瓣評論" heading, outbound-link styling matching sibling sections (`text-[var(--accent-primary)] hover:underline`, `target=_blank rel=noopener`), simple author + star-glyph + text comment list, total-comment count line. No screenshot to diff against.
- **Dev-Note-5 deviation (injected provider, not raw `douban.Scraper`):** Dev Note 5 / Task 3.1 suggested injecting "the concrete `douban.Scraper`". I instead injected `*metadata.DoubanProvider` as the `DoubanReviewScraper` (the SAME instance already injected as `DoubanSearcher`). Rationale: the raw `douban.Scraper` owns neither the cache nor the single rate limiter; injecting the provider (which owns the one `douban.Client` + `douban.Cache` + circuit breaker) is what preserves **AC #6** (cache-before-scrape, warm load = no new request) and **Rule 27 ①/②** (single limiter, cache before limiter). The provider's new `ScrapeReviewSummary` does cache→scrape→cache. The service still depends only on the `DoubanReviewScraper` interface + the `douban.ReviewSummaryResult` type (a type import, exactly as `DoubanSearcher` imports `metadata` types) — no concrete-scraper dependency, so Rule 4/11/19 layering holds.
- **Selector decision (Task 1.1):** parses the subject-page inline `#comments-section .comment-item` block (fallbacks `#hot-comments`/`.comment-list`) so NO extra request beyond `DetailURL(id)`. The `/subject/{id}/comments/` fallback was NOT needed. Selectors validated against `testdata/subject_comments.html`.
- **AC #1 backend-field check (Task 3.4): VERIFIED** — `services.DoubanRatingResult.DoubanID` (`json:"douban_id"`) → camel `doubanId` (Rule 18); the route already runs `useDoubanRating`, so the "查看豆瓣頁面" link is built 100% client-side. No new backend field added for the link.
- **Pre-existing tsc note (Epic 9c Retro AI-2 — NEITHER fix-nor-file, justified):** `tsc --noEmit -p apps/web/tsconfig.app.json` reports pre-existing errors repo-wide (RecentMediaPanel, HeroBanner, EmptyNoFolder, downloads, scanner, gallery fixtures, and the 12-1 `localData.releaseDate` union-narrowing in this route's meta line at `:295/:299`). These are NOT introduced by 12-6 (all new 12-6 files are type-clean) and are NOT test failures — full `tsc` is OUTSIDE the project CI gate (Rule 12 = go vet + staticcheck + eslint + prettier; `nx build web` uses vite/esbuild, no typecheck). No tracking entry filed: not a test failure, repo-wide pre-existing tsc tolerance, out of this story's scope.

### Code Review Follow-ups (Amelia adversarial CR, 2026-06-12)

Outcome: **0 High / 2 Medium / 3 Low** — all Medium + actionable Low auto-fixed; Rule 7 **PASS** (5 `DOUBAN_*` codes, no new prefixes), Rule 20 **N/A** (no stamp bumps), Rule 25 **N/A** (no mega-line change). Git vs File List: 0 discrepancies. All ACs verified IMPLEMENTED; all `[x]` tasks verified done; test claims re-run and confirmed green.

- **M1 (fixed) — wasted Douban search scrape for unmatched titles.** `useDoubanReviewSummary` was gated on `tmdbId > 0`, but `DoubanSection` only renders when a `doubanId` is resolved, so every unmatched-title detail view fired a discarded review-summary request → a live, rate-limited Douban *search* scrape with no UI payoff (also defeats Task 6.2's stated "avoid a scrape when no Douban match exists" and Rule 27 Pillar 1). Fix: gate `enabled` on `Boolean(doubanQuery.data?.doubanId)` so the fetch matches the render condition. [`routes/media/$type.$id.tsx:142`, `hooks/useDoubanReviewSummary.ts` doc]
- **M2 (fixed) — star-scale conflated with comment cap.** `StarRating` used `MAX_COMMENTS` (=5, the short-comment cap) as the 5-star maximum; coincidentally correct but a future change to the comment cap would silently corrupt star rendering. Fix: introduced an independent `MAX_STARS = 5`. [`components/media/DoubanSection.tsx`]
- **L1 (fixed) — inaccurate staleTime comment.** Comment claimed "24h matches the server-side Douban cache TTL"; the server TTL is 7 days. Reworded — 24h is the client freshness window mirroring `useDoubanRating`. [`hooks/useDoubanReviewSummary.ts:5`]
- **L2 (fixed) — total-count fallback understated.** When the "全部 N 条" header is absent, the parser fell back to `len(TopComments)` (capped at 5), so the UI could show "共 5 則短評" for a larger page. Fix: fall back to `items.Length()` (un-capped on-page comment-item count). [`douban/scraper.go:485`]
- **L3 (acknowledged, no change) — AC #8 has no viewport-specific test.** The review list is single-column by construction (`flex flex-col`), so AC #8 is satisfied structurally; no responsive assertion added (consistent with the design-coverage-gap status).

Re-verification after fixes: `go test ./internal/{douban,services,handlers,database/migrations}` green + `go vet` clean; `DoubanSection.spec.tsx` 8/8; prettier clean on all touched files.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **① absorb-in-place — Task 6.1 "both views" premise was incorrect (resolved, no deferred work):** TMDbDetailView (`idKind === 'tmdb-numeric'`) has no local UUID, and BOTH Douban endpoints (`/{movies,series}/:id/douban-rating` and `…/douban-review-summary`) are local-UUID-keyed, so TMDbDetailView can NEVER resolve a `doubanId`. Story 12-1 already established and documented this in-code (the `DualRatingDisplay` comment in TMDbDetailView). Therefore `DoubanSection` is correctly **LocalDetailView-only**; in TMDbDetailView the section is omitted, which is exactly what **AC #4** ("no resolved douban_id → omit") prescribes — so **AC #1/#4 are satisfied**, not deferred. Absorbed in-place by (a) wiring LocalDetailView fully and (b) extending the existing 12-1 in-code comment in TMDbDetailView to name the 12-6 review summary. No new `sprint-status.yaml` entry (no deferred work; consistent with the established 12-1 precedent).
  - **Live-scrape feasibility (anticipated ③ candidate → N/A at code level):** the parser + 7-day cache + fail-soft degrade are implemented and fixture-tested; the `/subject/{id}/comments/` fallback was not needed (subject page carries the 短評 inline). Real-Douban anti-scrape behavior is NOT exercised by unit tests (no network) — if a future DEPLOY shows the live subject page lacks usable 短評 or blocks, the section degrades gracefully (AC #5) and the direct link survives (AC #1); file the ③ backlog entry at that point. Recorded **N/A** now (feature is implemented + fail-soft, not infeasible).
- Reference: `project-context.md` Rule 24.

### File List

**Created — backend:**
- `apps/api/internal/database/migrations/026_add_douban_review_summary.go`
- `apps/api/internal/database/migrations/026_add_douban_review_summary_test.go`
- `apps/api/internal/douban/scraper_review_test.go`
- `apps/api/internal/douban/cache_review_test.go`
- `apps/api/internal/douban/testdata/subject_comments.html`
- `apps/api/internal/services/douban_review_service_test.go`
- `apps/api/internal/handlers/douban_review_handler_test.go`

**Modified — backend:**
- `apps/api/internal/douban/types.go` (ReviewSummaryResult, ReviewComment)
- `apps/api/internal/douban/scraper.go` (ScrapeReviewSummary + parse + s2twp)
- `apps/api/internal/douban/cache.go` (GetReviewSummary/SetReviewSummary + non-clobbering detail Set)
- `apps/api/internal/douban/cache_test.go` (review_summary_json column in test schema)
- `apps/api/internal/metadata/douban_provider.go` (cache-aware ScrapeReviewSummary)
- `apps/api/internal/services/douban_rating_service.go` (DoubanReviewScraper iface, EnrichDoubanReviewSummary, resolveDoubanID, constructor arg)
- `apps/api/internal/services/douban_rating_service_test.go` (4-arg constructor)
- `apps/api/internal/handlers/douban_rating_handler.go` (2 handler methods + routes + Swaggo)
- `apps/api/internal/handlers/douban_rating_handler_test.go` (mock method + douban import)
- `apps/api/cmd/api/main.go` (inject review scraper)

**Created — frontend:**
- `apps/web/src/hooks/useDoubanReviewSummary.ts`
- `apps/web/src/components/media/DoubanSection.tsx`
- `apps/web/src/components/media/DoubanSection.spec.tsx`

**Modified — frontend:**
- `apps/web/src/types/library.ts` (ReviewComment, DoubanReviewSummary, DoubanReviewSummaryResponse)
- `apps/web/src/services/libraryService.ts` (get{Movie,Series}DoubanReviewSummary)
- `apps/web/src/routes/media/$type.$id.tsx` (DoubanSection in LocalDetailView + 12-6 comment in TMDbDetailView)

## Change Log

| Date | Change |
|------|--------|
| 2026-06-12 | **Adversarial CR (Amelia, code-review).** 0 High / 2 Medium / 3 Low — all Medium + actionable Low auto-fixed: M1 review-summary query re-gated on a resolved `doubanId` (eliminates the discarded review request + live Douban *search* scrape on every unmatched-title view; aligns enable-gate with render-gate, Rule 27 ①); M2 `MAX_STARS` split from `MAX_COMMENTS` in `DoubanSection`; L1 corrected the staleTime comment (server cache TTL is 7d, the 24h is the client freshness window); L2 total-count fallback uses `items.Length()` instead of the 5-capped slice. L3 (AC #8 viewport test) acknowledged — single-column by construction. Rule 7 **PASS** / Rule 20 **N/A** / Rule 25 **N/A**; git vs File List 0 discrepancies. Re-verified: douban/services/handlers/migrations Go tests + `go vet` green, `DoubanSection.spec` 8/8, prettier clean. Status → **done**. |
| 2026-06-12 | **Implemented (Amelia, dev-story).** Backend: `douban.ScrapeReviewSummary` parsing the subject-page inline 短評 block (no extra request) + s2twp conversion (reuses `ChineseConverter`); `ReviewSummaryResult`/`ReviewComment` types; migration 026 `review_summary_json` on `douban_cache` (self-registering init()); `Cache.Get/SetReviewSummary` (upsert) + detail `Set` changed INSERT OR REPLACE→ON CONFLICT so a detail re-scrape can't clobber the review summary (AC #6); `metadata.DoubanProvider.ScrapeReviewSummary` (cache-before-scrape, single client/limiter — Rule 27 ①/②); `DoubanReviewScraper` interface + `EnrichDoubanReviewSummary`/`resolveDoubanID` on the 12-1 service (injects the SAME provider, not the raw scraper — see Completion Notes); handler `Get{Movie,Series}DoubanReviewSummary` + routes `/{movies,series}/:id/douban-review-summary` + Swaggo; main.go wiring. Frontend: `useDoubanReviewSummary` hook + `DoubanSection` (查看豆瓣頁面 link survives a failed scrape; comments author+stars+Traditional text, 5-cap, count) wired into LocalDetailView (TMDbDetailView omits — no local UUID, Discovery Triage ①). Reused `DOUBAN_*` codes (no new prefix/secret/scrape-infra). AC-Drift NONE / Contract-Stamps NONE / A11y PASS / UX design-coverage-gap. Full regression: web 2099/2099, api 0 fail. Status → review. |
| 2026-06-11 | Story drafted (SM Bob, create-story yolo). F-6 — Douban direct link + user review summary (短評). Direct link nearly free (reuses 12-1's stored `douban_id` / `DoubanRatingResult.DoubanID`, built client-side). Review summary: new `ScrapeReviewSummary` on the existing douban `Scraper` (reuses 0.5rps limiter / robots.txt / UA-rotation / kill-switch / OpenCC s2twp), cached in extended `douban_cache` (`review_summary_json`, 7-day TTL), surfaced via `EnrichDoubanReviewSummary` on the 12-1 `DoubanRatingService` + new `/{movies,series}/:id/douban-review-summary` routes. Frontend: `useDoubanReviewSummary` hook + `DoubanSection` (direct link survives a failed scrape). Reuse `DOUBAN_*` codes — no new prefix/secret/scrape-infra. Cross-stack split: backend 3 / frontend 3 → single story. Review-scrape fragility flagged (ADR Risk row) — fail-soft + fixture-tested parser + feasibility Discovery-Triage candidate. |
