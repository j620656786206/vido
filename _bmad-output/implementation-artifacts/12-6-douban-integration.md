# Story 12.6: Douban Integration — Direct Page Link + User Review Summary

Status: ready-for-dev

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

- [ ] **Task 1: Douban scraper — `ScrapeReviewSummary` + types** (AC: #2, #3, #5, #7)
  - [ ] 1.1 Add to `apps/api/internal/douban/scraper.go` a method `ScrapeReviewSummary(ctx, id string) (*ReviewSummaryResult, error)` mirroring `ScrapeDetail` (`:35-80`): reuse `client.GetBody(ctx, DetailURL(id))` — **prefer parsing the short-comments (短評) already present on the subject page** (`https://movie.douban.com/subject/{id}/`) so NO extra request beyond the page the rating path already knows; only fall to `/subject/{id}/comments/` if the subject page lacks them (document the selector decision in the dev notes).
  - [ ] 1.2 Define `ReviewSummaryResult { ID string; TotalComments int; TopComments []ReviewComment }` and `ReviewComment { Author string; Rating int; Text string }` in `apps/api/internal/douban/types.go` (alongside `DetailResult`).
  - [ ] 1.3 Parse with goquery mirroring `parseDetailPage` style (`scraper.go:111-196`) — short-comment item selectors (e.g. `#comments-section .comment-item`, author `.comment-info a`, rating `.rating` class suffix, text `.short`). **Validate selectors against a saved fixture** (capture a real subject page into `scraper_test.go` testdata; Douban markup drifts — Rule 27 Pillar 3 risk row).
  - [ ] 1.4 Convert each comment `Text` to Traditional via the existing `ChineseConverter.ConvertIfSimplified` (reuse `convertToTraditional` style, `scraper.go:83-108`).
  - [ ] 1.5 Errors reuse the existing `DOUBAN_*` codes (`types.go:168-175` — `DOUBAN_BLOCKED`/`DOUBAN_PARSE_ERROR`/`DOUBAN_RATE_LIMITED`/`DOUBAN_TIMEOUT`/`DOUBAN_NOT_FOUND`). **NO new prefix** (Rule 7 / ADR Pillar 4). Parse-failure → `DOUBAN_PARSE_ERROR`, returned fail-soft upstream.
  - [ ] 1.6 Scraper test: fixture-based parse (top-comments extracted, s2twp conversion applied), block/parse-error paths.

- [ ] **Task 2: Cache the review summary (extend `douban_cache`)** (AC: #6)
  - [ ] 2.1 Add a migration (next free number — **verify the highest existing migration number first**; 12-2 used 025, so this is likely **026+**; grep `apps/api/internal/database/migrations/`) to add `review_summary_json TEXT` to the `douban_cache` table (idempotent `ALTER TABLE … ADD COLUMN`, self-registers via `init()` — NOT a `registry.go` edit, per 12-2's correction).
  - [ ] 2.2 Extend `apps/api/internal/douban/cache.go` `Get`/`Set` (`:98-180`) to hydrate/persist `review_summary_json` (JSON-encode `ReviewSummaryResult.TopComments` + `TotalComments`). Reuse the existing `douban_id`-keyed lookup + 7-day TTL (`DefaultCacheConfig`). A review-summary cache hit needs no scrape (cache-before-limiter, Pillar 2).
  - [ ] 2.3 Cache test for the new column (round-trip + expiry).

- [ ] **Task 3: Service + handler + routes** (AC: #1, #2, #4, #5)
  - [ ] 3.1 Extend `apps/api/internal/services/douban_rating_service.go` with `EnrichDoubanReviewSummary(ctx, mediaID, mediaType string) (*ReviewSummaryResult, error)` mirroring `EnrichDoubanRating` (`:81-119`): read the media record → if `douban_id` already stored (from 12-1), scrape reviews by that id; if NOT stored, run the existing rating-lookup first to RESOLVE the id (reuse `lookup`/`pickBestMatch`), then scrape; if still unresolved → return `nil` (AC #4 omit). Wrap the scrape in the existing `doubanLookupTimeout` (10s) context.
    - Inject the review-capable scraper via an interface (mirror the `DoubanSearcher` injection pattern `:42-45`) — do NOT have the service reach into `internal/douban` concretely (Rule 4/11/19 layering).
  - [ ] 3.2 Return `nil` (not an error) on any block/parse/timeout failure (graceful degradation — same contract as `EnrichDoubanRating` returning nil; AC #5).
  - [ ] 3.3 Extend `apps/api/internal/handlers/douban_rating_handler.go` with `GetMovieDoubanReviewSummary` / `GetSeriesDoubanReviewSummary` (mirror `:32-66`); register routes `GET /api/v1/movies/:id/douban-review-summary` and `GET /api/v1/series/:id/douban-review-summary` (mirror `RegisterRoutes` `:71-74`). Response `{ "success": true, "data": {...} | null }` (Rule 3; null = omit).
  - [ ] 3.4 Confirm `douban_id` is in the response surface so the frontend can build the direct link (AC #1): the existing `DoubanRatingResult` already carries `DoubanID` (`douban_rating_service.go:31-35`) — the frontend's existing `useDoubanRating` data already exposes `doubanId`. **Verify** the detail page receives it from the rating query; if so, AC #1 needs NO new backend field (build the link client-side). Note the finding.
  - [ ] 3.5 Wire `EnrichDoubanReviewSummary` deps in `cmd/api/main.go` (the `DoubanRatingService`/handler are already wired by 12-1 — add the review-scraper dep). Swaggo annotations on both new endpoints (Rule 15). Service + handler tests (resolved-id happy path, unresolved-id → null, block → null).

### Frontend

- [ ] **Task 4: Types + service + hook** (AC: #1, #2, #6)
  - [ ] 4.1 Add `ReviewComment { author: string; rating: number; text: string }` and `DoubanReviewSummary { id: string; totalComments: number; topComments: ReviewComment[] }` to `apps/web/src/types/library.ts`.
  - [ ] 4.2 Add `getMovieDoubanReviewSummary(id)` / `getSeriesDoubanReviewSummary(id)` to `apps/web/src/services/libraryService.ts` (mirror the 12-1 `getMovieDoubanRating` methods → `/movies/${id}/douban-review-summary`).
  - [ ] 4.3 Add `useDoubanReviewSummary(id, type, enabled)` to `apps/web/src/hooks/useDoubanRating.ts` (mirror `useDoubanRating` — `queryKey: ['douban-review-summary', type, id]`, `staleTime` 24h, `enabled: enabled && !!id`). Returns `DoubanReviewSummary | null`.

- [ ] **Task 5: `DoubanSection` component (direct link + review summary)** (AC: #1, #4, #5, #8)
  - [ ] 5.1 Create `apps/web/src/components/media/DoubanSection.tsx`. Props: `doubanId?: string` (from the existing rating query), `summary?: DoubanReviewSummary | null`, `isLoading`, `isError`.
  - [ ] 5.2 **If `doubanId`**: render the "查看豆瓣頁面" direct link → `https://movie.douban.com/subject/${doubanId}/` (AC #1). **If NO `doubanId`**: render nothing (AC #4).
  - [ ] 5.3 Review summary: if `summary?.topComments?.length`, render each comment (author + star rating + Traditional-Chinese text), capped at 5, with the total-comments count. Loading → quiet skeleton; error/empty → omit the review block but KEEP the direct link (AC #5). Never throw.
  - [ ] 5.4 Rule 21 header (feature postdates the `.pen` design): design-coverage-gap form `// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page Douban review section postdates the .pen design`.
  - [ ] 5.5 Mobile: one comment per row, author/stars line + text below (Tailwind responsive — AC #8). Write `DoubanSection.spec.tsx` (link present when doubanId, omitted when not, comments render with Traditional text, error keeps link drops reviews, empty-state). Rule 16 matchers.

- [ ] **Task 6: Integrate into the detail page** (AC: #1)
  - [ ] 6.1 In `apps/web/src/routes/media/$type.$id.tsx`, render `<DoubanSection />` **below the overview / near the other Epic-12 enrichment sections** in BOTH `LocalDetailView` and `TMDbDetailView`. Source `doubanId` from the EXISTING `doubanQuery.data?.doubanId` (12-1's `useDoubanRating`, already wired into the route at `~:274-280`); source `summary` from the new `useDoubanReviewSummary(localId, type, enabled)`.
  - [ ] 6.2 Gate the review-summary fetch on the same `enabled` condition 12-1 uses for the rating (typically once a `douban_id`/`tmdbId` is known) to avoid a scrape when no Douban match exists.

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **Anticipated at authoring time (dev to confirm/triage):**
    - **③ backlog candidate:** If the subject page does NOT carry usable 短評 and the `/subject/{id}/comments/` page is robots-disallowed or anti-scrape-blocked, the review-summary feature may be infeasible. If so, ship the AC #1 direct link, omit the review block gracefully, and file a **③ backlog** `sprint-status.yaml` entry documenting the scrape-feasibility limitation (bidirectional link) rather than a prose-only note (Rule 24 ban). If reviews scrape fine, record `N/A`.
  - Otherwise record each in-flight discovery with its lane (①/②/③) + tracked entry ID before marking done. If none: `N/A — no out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List

## Change Log

| Date | Change |
|------|--------|
| 2026-06-11 | Story drafted (SM Bob, create-story yolo). F-6 — Douban direct link + user review summary (短評). Direct link nearly free (reuses 12-1's stored `douban_id` / `DoubanRatingResult.DoubanID`, built client-side). Review summary: new `ScrapeReviewSummary` on the existing douban `Scraper` (reuses 0.5rps limiter / robots.txt / UA-rotation / kill-switch / OpenCC s2twp), cached in extended `douban_cache` (`review_summary_json`, 7-day TTL), surfaced via `EnrichDoubanReviewSummary` on the 12-1 `DoubanRatingService` + new `/{movies,series}/:id/douban-review-summary` routes. Frontend: `useDoubanReviewSummary` hook + `DoubanSection` (direct link survives a failed scrape). Reuse `DOUBAN_*` codes — no new prefix/secret/scrape-infra. Cross-stack split: backend 3 / frontend 3 → single story. Review-scrape fragility flagged (ADR Risk row) — fail-soft + fixture-tested parser + feasibility Discovery-Triage candidate. |
