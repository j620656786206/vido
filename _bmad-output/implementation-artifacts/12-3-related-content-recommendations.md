# Story 12.3: Related Content Recommendations — TMDB Recommendations/Similar with "已有" Badges

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a media library user viewing a movie or TV series detail page,
I want to see a "相關推薦" section of related titles pulled from TMDB, with a badge on any title I already own,
so that I can discover similar content and immediately see what is already in my library versus what is new.

## Acceptance Criteria

1. **Given** a movie or TV detail page with a valid TMDB id (`tmdb_id > 0`), **when** the page loads, **then** a "相關推薦" recommendations section renders **below the credits/cast section** showing a responsive grid of related titles from TMDB.
2. **Given** the recommendations endpoint returns results, **then** each tile shows poster, title, release year, and rating — rendered by the **existing `PosterCard`** component (reuse, not reinvent).
3. **Given** a recommended title's TMDB id already exists in the local library (movies OR series, not soft-deleted), **then** the tile displays the **"已有" owned badge** via the existing `PosterCard isOwned` prop + `AvailabilityBadge` (Story 10-4 mechanism). Ownership is resolved with a **single batched lookup** (`FindOwnedTMDbIDs`), never per-tile N+1.
4. **Given** the TMDB `/recommendations` endpoint returns an empty list for the title, **when** recommendations are fetched, **then** the backend **falls back to the TMDB `/similar` endpoint** so the section is still populated when possible; the response indicates which source filled it (`source: "recommendations" | "similar"`).
5. **Given** both `/recommendations` and `/similar` return empty, **then** the section renders a quiet empty-state (or is omitted) — it MUST NOT render an error that breaks or blocks the rest of the page (Rule 27 Pillar 3 — fail-soft, enrichment-not-core).
6. **Given** the TMDB API is unavailable / times out / errors, **then** the recommendations section degrades to an empty-state with an optional retry affordance, and the **rest of the detail page renders unaffected** (per-section isolation, Rule 27 Pillar 3).
7. **Given** a user clicks/taps a recommendation tile, **then** they navigate to that title's detail page via the TMDB-numeric route (`/media/$type/$id`); an owned title shows its owned badge in the resulting `TMDbDetailView` (consistent with Story 10-4 — resolving owned→local-UUID is explicitly out of scope).
8. **Given** a mobile viewport, **then** the recommendations grid wraps/scrolls responsively and remains readable (reuse `MediaGrid`/`PosterCard` responsive layout — no new responsive CSS).
9. **Given** repeated visits to the same title, **then** recommendations are served from cache (24h TTL, cache checked **before** the rate limiter per Rule 27 Pillar 2), keeping warm detail-page load < 1.5s (Epic 12 success criterion).

## Tasks / Subtasks

### Backend

- [ ] **Task 1: TMDB client — recommendations + similar endpoint wrappers** (AC: #1, #4)
  - [ ] 1.1 Add to `apps/api/internal/tmdb/movies.go`: `GetMovieRecommendations(ctx, movieID int) (*SearchResultMovies, error)` and `GetMovieSimilar(ctx, movieID int) (*SearchResultMovies, error)` (+ `…WithLanguage` variants threaded through the language fallback chain, mirroring the `GetMovieDetails`/`GetMovieDetailsWithLanguage` pattern at `movies.go:47-74`). Endpoints: `/movie/{id}/recommendations`, `/movie/{id}/similar`.
  - [ ] 1.2 Add to `apps/api/internal/tmdb/tv.go`: `GetTVRecommendations(ctx, tvID int) (*SearchResultTVShows, error)` and `GetTVSimilar(ctx, tvID int) (*SearchResultTVShows, error)` (+ `…WithLanguage`). Endpoints: `/tv/{id}/recommendations`, `/tv/{id}/similar`.
  - [ ] 1.3 **Reuse** existing response types — TMDB returns `{page, results[], total_pages, total_results}`, exactly `SearchResultMovies` / `SearchResultTVShows` (`tmdb/types.go:75-89`). Do NOT define new list types; tiles use the lightweight `Movie` / `TVShow` items (not the `*Details` variants).
  - [ ] 1.4 Thread all four through the chain exactly as 12-2 threaded `GetSeasonDetails`: `ClientInterface` (`client.go`) → `LanguageFallbackClient` `…WithFallback` (`fallback.go`) → `CacheService` cached wrapper (`cache.go`, 24h `DefaultCacheTTL`) → `TMDbService` (`services/tmdb_service.go`). Cache keys (Rule 27 Pillar 2, `{source}:{type}:{id}:{version}`): `tmdb:recommendations:movie:{id}:v1`, `tmdb:similar:movie:{id}:v1`, and `…:tv:…` equivalents.
  - [ ] 1.5 Rate limiting (Rule 27 Pillar 1): these ride the **existing shared TMDB limiter** (`client.go:16-23` `requestsPerInterval=40`, `rateLimitInterval=10s`) — ZERO new limiter, `limiter.Wait(ctx)` already first line of `doRequest`.
  - [ ] 1.6 Tests: client wrapper tests + fallback mock + cache test (mirror `tmdb/fallback_test.go`, `cache_test.go` additions from 12-2).

- [ ] **Task 2: RecommendationService — fallback chain + ownership annotation** (AC: #3, #4, #5)
  - [ ] 2.1 Create `apps/api/internal/services/recommendation_service.go` with `RecommendationService` holding deps: `TMDbServiceInterface` + `MovieRepositoryInterface` + `SeriesRepositoryInterface` (Rule 4 — handler→service→repo; the service owns the cross-domain ownership join so the handler never touches a repo directly).
  - [ ] 2.2 `GetMovieRecommendations(ctx, tmdbID int) (*RecommendationResult, error)`: call `tmdbService.GetMovieRecommendations`; **if `len(results)==0`, fall back to `GetMovieSimilar`** (AC #4); record `Source` = `"recommendations"` | `"similar"` | `""` (empty).
  - [ ] 2.3 `GetTVRecommendations(ctx, tmdbID int) (*RecommendationResult, error)`: same shape for TV.
  - [ ] 2.4 Ownership annotation: collect the result tmdb ids, call `movieRepo.FindOwnedTMDbIDs(ctx, ids)` (for movie type) or `seriesRepo.FindOwnedTMDbIDs(ctx, ids)` (for tv type) — **single batched query** (`movie_repository.go:907-968` / `series_repository.go:855-915`, Story 10-4). Build an owned-id set, stamp `IsOwned` per item. **This is REUSE of the Story 10-4 mechanism, not new drift.**
  - [ ] 2.5 Define `RecommendationItem` DTO (normalized across movie/TV: `ID int`, `MediaType string` `"movie"|"tv"`, `Title string` (TV → `Name`), `PosterPath *string`, `ReleaseDate string` (TV → `FirstAirDate`), `VoteAverage float64`, `IsOwned bool`) and `RecommendationResult { Items []RecommendationItem; Source string }`. Cap the rendered set at **18 items** (slice the TMDB page) to bound payload + ownership-lookup placeholders.
  - [ ] 2.6 Graceful degradation (Rule 27 Pillar 3): a TMDB error returns a Rule-7 `AppError` (reuse `TMDB_*` — `TMDB_TIMEOUT`/`TMDB_NOT_FOUND`/`TMDB_RATE_LIMIT`; **NO new prefix** per ADR Pillar 4); an ownership-lookup error MUST NOT fail the whole call — log and degrade to `IsOwned=false` for all (recommendations still render, just without badges). Service + ownership-merge unit tests.

- [ ] **Task 3: Handler endpoints + routes + wiring** (AC: #1, #5, #6)
  - [ ] 3.1 Add handler methods to the TMDB handler (`apps/api/internal/handlers/tmdb_handler.go`, sibling to the existing `GetMovieVideos` at `:440`): `GetMovieRecommendations`, `GetTVRecommendations`. Parse `:id` (TMDB numeric), validate `> 0`, call `RecommendationService`, return `{ "success": true, "data": { "results": [...], "source": "..." } }` (Rule 3).
  - [ ] 3.2 Register routes alongside the existing `…/videos` routes: `GET /api/v1/tmdb/movies/:id/recommendations` and `GET /api/v1/tmdb/tv/:id/recommendations`. **Rule 15 self-check**: confirm the exact existing TV videos route prefix (`/api/v1/tmdb/tv/:id/videos` vs `/tmdb/tvshows/...`) by grepping the handler's `RegisterRoutes` + `cmd/api/main.go`, and match it — do NOT assume.
  - [ ] 3.3 Wire `RecommendationService` construction in `apps/api/cmd/api/main.go` (inject `TMDbService`, `MovieRepository`, `SeriesRepository`); wire the handler dependency (Rule 15 — handler/service registered in DI, route added to router).
  - [ ] 3.4 Swaggo annotations on both new endpoints (Rule 15 Swagger sync); handler tests (success, empty→similar-fallback, TMDB-error→fail-soft, ownership-badge present).

### Frontend

- [ ] **Task 4: Types + service methods** (AC: #1, #2, #3, #4)
  - [ ] 4.1 Add to `apps/web/src/types/library.ts` (or `types/tmdb.ts` — match where sibling recommendation/card types live): `RecommendationItem { id: number; mediaType: 'movie' | 'tv'; title: string; posterPath: string | null; releaseDate?: string; voteAverage?: number; isOwned: boolean }` and `RecommendationsResponse { results: RecommendationItem[]; source: string }`.
  - [ ] 4.2 Add to `apps/web/src/services/libraryService.ts` (mirror `getSeasonEpisodes` at `:165`): `getMovieRecommendations(tmdbId: number)` → `fetchApi('/tmdb/movies/${tmdbId}/recommendations')` and `getTVRecommendations(tmdbId: number)` → `fetchApi('/tmdb/tv/${tmdbId}/recommendations')`. Case-transform is automatic via `fetchApi`/`snakeToCamel` (Rule 18).

- [ ] **Task 5: `useRecommendations` hook** (AC: #1, #9)
  - [ ] 5.1 Add to `apps/web/src/hooks/useMediaDetails.ts` (mirror `useSeriesSeasons` at `:29`): `useRecommendations(tmdbId: number, type: 'movie' | 'tv', enabled: boolean)` → `useQuery` calling the matching service method.
  - [ ] 5.2 Add a `detailKeys.recommendations(tmdbId, type)` entry to the query-key factory (`useMediaDetails.ts:12-24`).
  - [ ] 5.3 `staleTime: 24 * 60 * 60 * 1000` (24h — matches backend cache TTL; recs are stable). `enabled: enabled && tmdbId > 0` (gate on a valid TMDB id, like the existing `useMovieDetails`/`useTVShowDetails`).

- [ ] **Task 6: `RelatedContent` section component** (AC: #1, #2, #3, #5, #6, #8)
  - [ ] 6.1 Create `apps/web/src/components/media/RelatedContent.tsx`. Props: `items: RecommendationItem[]`, `isLoading`, `isError`, `onRetry`. Header: "相關推薦".
  - [ ] 6.2 Render tiles via the **existing `PosterCard`** (`id={String(item.id)}`, `type={item.mediaType}`, `title`, `posterPath`, `releaseDate`, `voteAverage`, **`isOwned={item.isOwned}`**) inside the existing **`MediaGrid`** layout (or a capped responsive grid reusing MediaGrid's class) — reuse-over-reinvent, no new card/grid.
  - [ ] 6.3 Loading skeleton (reuse `PosterCardSkeleton`); error → quiet `role="alert"` empty-state with retry (AC #6); empty results → render nothing or a muted "暫無推薦" line (AC #5). Never throw.
  - [ ] 6.4 Rule 21 header (component postdates the `.pen` design — Epic 12 not in `ux-design.pen`): use the design-coverage-gap form `// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page recommendations section postdates the .pen design` (NOT the deprecated `<screen-section — pending …>` placeholder; see Rule 21 + 19-8 sweep). Tiles inherit `PosterCard`'s own `.pen` link.
  - [ ] 6.5 Write `RelatedContent.spec.tsx` (renders tiles, owned badge present when `isOwned`, skeleton while loading, retry on error, empty-state). Rule 16 matchers (`toBeInTheDocument`, `toBeAttached` for any hover/transition).

- [ ] **Task 7: Integrate into the detail page** (AC: #1, #7)
  - [ ] 7.1 In `apps/web/src/routes/media/$type.$id.tsx`, render `<RelatedContent />` **below `CreditsSection`** in BOTH `LocalDetailView` (after credits at ~`:313`, above `SeasonAccordion`) AND `TMDbDetailView` (after credits at ~`:510`, as the final section).
  - [ ] 7.2 Resolve the TMDB id + type per view: `LocalDetailView` → `movie.tmdbId` / `series.tmdbId` + `type`; `TMDbDetailView` → the numeric route id + `type`. Pass to `useRecommendations(tmdbId, type, enabled)` (enable only when `tmdbId > 0`).
  - [ ] 7.3 Tile navigation is handled by `PosterCard`'s built-in link to `/media/$type/$id` (numeric → TMDbDetailView; owned items show the owned badge there). No extra nav wiring.

## Dev Notes

### Architecture Compliance

- **Rule 4 / Rule 11 (Layered Architecture, interface location):** `TMDbHandler.GetMovieRecommendations` → `RecommendationService.GetMovieRecommendations` → (`TMDbServiceInterface` for the TMDB call **and** `MovieRepositoryInterface`/`SeriesRepositoryInterface` for the ownership join). The handler NEVER calls a repo directly; the service owns the cross-domain join. Interfaces live in their owning packages.
- **Rule 5 (TanStack Query):** recommendations fetched via `useRecommendations` `useQuery`, gated `enabled: tmdbId > 0`.
- **Rule 6 (Naming):** endpoints `/api/v1/tmdb/movies/:id/recommendations`, `/api/v1/tmdb/tv/:id/recommendations`; Go `GetMovieRecommendations`/`GetMovieSimilar`; TS `getMovieRecommendations`/`useRecommendations`; JSON `is_owned`/`poster_path` (snake) ↔ `isOwned`/`posterPath` (camel via Rule 18).
- **Rule 7 (Error Codes) + Rule 27 Pillar 4:** reuse `TMDB_*` ONLY — **no new prefix** (ADR Decision/Pillar 4). Ownership-lookup failure is non-fatal (degrade, don't surface a `DB_*` error to the section).
- **Rule 10 (API Versioning):** all new routes under `/api/v1/`.
- **Rule 13 (Error Handling Completeness):** propagate TMDB errors as `AppError`; ownership-lookup error is logged-and-degraded (the one intentional "discard with comment" case — recommendations still render).
- **Rule 14 / Rule 27 Pillar 1:** TMDB client built once + reused; the existing 40/10s limiter is shared — F-3 adds ZERO new rate budget.
- **Rule 15 (Pre-commit Self-verification):** wire `RecommendationService` + handler in `main.go`; register both routes; Swaggo annotations; **verify the exact TV route prefix** (client method existing ≠ route registered — the Epic-10 `GetMovieVideos` precedent).
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` for tiles/badges; `toBeAttached()` for any hover state.
- **Rule 18 (Case Transform):** auto via `fetchApi`.
- **Rule 21 (Component↔Design):** `RelatedContent.tsx` uses the design-coverage-gap `// Design ref:` form (feature postdates the `.pen` design).
- **Rule 27 (External Integration Standard — the Five Pillars):** ✅ ① rate limit — reuses shared TMDB limiter (no new bucket) · ✅ ② cache — 24h tiered, checked before limiter, keys `tmdb:recommendations|similar:{movie|tv}:{id}:v1` · ✅ ③ degrade — per-section fail-soft, recommendations→similar fallback, page never fails · ✅ ④ error codes — reuse `TMDB_*`, no new prefix · ✅ ⑤ keys — existing TMDB `ClientConfig.APIKey`, no new secret. [Source: ADR `adr-external-api-integration-standard.md` Decision 1+2; project-context.md Rule 27]

### Cross-Stack Split Check (MANDATORY — Agreement 5 / Epic 9c Retro AI-1)

Backend tasks: **3** (Task 1 client wrappers, Task 2 service, Task 3 handler/wiring). Frontend tasks: **4** (Task 4 types/service, Task 5 hook, Task 6 component, Task 7 integration).

Threshold is "BOTH counts > 3". Backend = 3 (**not** > 3) → **NO split required. Single story.** (The backend is deliberately kept to 3 cohesive tasks; the TMDB wrappers + service + handler are tightly coupled and a backend-only split would create an API with no consumer.)

### Project Structure Notes

**Files to CREATE:**
- `apps/api/internal/services/recommendation_service.go` (+ `_test.go`)
- `apps/web/src/components/media/RelatedContent.tsx` (+ `RelatedContent.spec.tsx`)

**Files to MODIFY:**
- `apps/api/internal/tmdb/movies.go` — `GetMovieRecommendations[WithLanguage]`, `GetMovieSimilar[WithLanguage]`
- `apps/api/internal/tmdb/tv.go` — `GetTVRecommendations[WithLanguage]`, `GetTVSimilar[WithLanguage]`
- `apps/api/internal/tmdb/client.go` — `ClientInterface` new methods
- `apps/api/internal/tmdb/fallback.go` (+ `fallback_test.go`) — `…WithFallback` wrappers + mock
- `apps/api/internal/tmdb/cache.go` (+ `cache_test.go`) — cached wrappers (24h) + cache test
- `apps/api/internal/services/tmdb_service.go` — `TMDbServiceInterface` + impl methods (and update sibling service-test mocks that embed `TMDbServiceInterface`, as 12-2 did)
- `apps/api/internal/handlers/tmdb_handler.go` (+ `tmdb_handler_test.go`) — handler methods + routes
- `apps/api/cmd/api/main.go` — construct + wire `RecommendationService` and handler dep
- `apps/web/src/types/library.ts` (or `types/tmdb.ts`) — `RecommendationItem`, `RecommendationsResponse`
- `apps/web/src/services/libraryService.ts` — `getMovieRecommendations`, `getTVRecommendations`
- `apps/web/src/hooks/useMediaDetails.ts` — `useRecommendations` + `detailKeys.recommendations`
- `apps/web/src/routes/media/$type.$id.tsx` — render `<RelatedContent />` in both detail views

### Critical Implementation Details

1. **Recommendations vs Similar — server-side fallback (AC #4).** TMDB exposes two endpoints: `/recommendations` (behavior-aggregate, usually higher quality) and `/similar` (genre/keyword). The service calls `/recommendations` first and falls back to `/similar` only when recommendations is empty — the frontend sees ONE endpoint. The epic's "similar/recommended" is satisfied by surfacing both server-side. `source` in the response tells the UI which filled it (useful for a future "推薦" vs "類似" label; not required to display now).

2. **The "已有" badge IS Story 10-4, reused (AC #3).** `FindOwnedTMDbIDs(ctx, []int64) ([]int64, error)` already exists on **both** `MovieRepository` (`movie_repository.go:907-968`) and `SeriesRepository` (`series_repository.go:855-915`) — single `SELECT DISTINCT tmdb_id … WHERE tmdb_id IN (…) AND is_removed = 0`, dedup'd, no N+1. `PosterCard` already has an `isOwned` prop (`PosterCard.tsx:25-29`, "Story 10-4 — the user already owns this title") wired to `AvailabilityBadge`. F-3 adds NO new ownership mechanism — it calls the existing batch lookup and passes the existing prop. **This is REUSE; the AC-Drift check will hit Story 10-4 — cite it as reuse, not drift.**

3. **Two detail-view code paths (AC #1, #7).** The route splits into `LocalDetailView` (owned library item, local UUID, has `tmdbId` field) and `TMDbDetailView` (TMDB numeric id, from homepage explore/search). Recommendations are keyed by the **TMDB numeric id**, available in BOTH views — so the section renders in both (unlike 12-2's `SeasonAccordion`, which is local-only because it needs the local series UUID). Place it below `CreditsSection` in each.

4. **Tile navigation & owned-item resolution (AC #7).** `PosterCard` links to `/media/$type/$id` with `id` = TMDB numeric. Clicking an owned recommendation lands on `TMDbDetailView` (which has its own owned badge), NOT the local-UUID view. Resolving TMDB-id → local-UUID for owned recommendations is **explicitly out of scope** (matches Story 10-4 homepage-tile behavior). If a future story wants owned tiles to deep-link into the local detail view, that is a separate concern — see Discovery Triage below.

5. **Caching keeps the page < 1.5s (AC #9, Rule 27 Pillar 2).** Recommendations cached 24h, cache checked before `limiter.Wait`. First (cold) load pays one TMDB round-trip (~ up to 2 round-trips if recommendations is empty and it falls back to similar — both cached after). Warm loads are instant. The section fetches **lazily on the detail page** but is not gated behind a click (unlike the accordion); it loads with the page but its own query, so a slow/failed recs fetch never delays the core metadata render.

6. **DTO normalization (Task 2.5).** TMDB `Movie` uses `title`/`release_date`; `TVShow` uses `name`/`first_air_date`. The service flattens both into `RecommendationItem` with uniform `title`/`releaseDate` + a `mediaType` discriminator so the frontend has one tile shape. Cap at 18 items.

### Existing Code References

- TMDB endpoint-wrapper style: `apps/api/internal/tmdb/movies.go:47-74` (`GetMovieDetails`/`…WithLanguage`); `GetMovieVideos` already present (`movies.go`).
- TMDB chain precedent (12-2): `client.go:52-55` (interface), `tv.go:75-107` (`GetSeasonDetails`), `fallback.go:30-31`, `cache.go`, `services/tmdb_service.go:32-33` + arch comment `:60-61` (`TMDbService → CacheService → LanguageFallbackClient → Client`).
- Reusable list response types: `tmdb/types.go:75-89` (`SearchResultMovies`/`SearchResultTVShows`); item types `Movie`/`TVShow` `tmdb/types.go:5-54`.
- Rate limiter: `tmdb/client.go:16-23` (constants), `:105-107` (instantiation), `:121-125` (`limiter.Wait` first line of `doRequest`).
- Cache TTLs: `tmdb/cache.go:14-26` (`DefaultCacheTTL = 24h`).
- **Ownership batch lookup (the F-3 keystone):** `movie_repository.go:907-968` `FindOwnedTMDbIDs`; `series_repository.go:855-915` same. Schema: `movies.tmdb_id` + `idx_movies_tmdb_id` (migration 001); `series.tmdb_id` + `idx_series_tmdb_id` (migration 002).
- Handler/route precedent: `handlers/tmdb_handler.go:440` (`GetMovieVideos`), routes near `GetMovieVideos` registration.
- Frontend card reuse: `components/media/PosterCard.tsx:15-36` (props incl. `isOwned` `:25-29`); `components/media/MediaGrid.tsx:29-42` (responsive grid); `PosterCardSkeleton`, `AvailabilityBadge`.
- Hook + key-factory template: `hooks/useMediaDetails.ts:12-24` (`detailKeys`), `:29-37` (`useSeriesSeasons`), `:43-50` (`useSeasonEpisodes`).
- Service template: `services/libraryService.ts:21-36` (`fetchApi`), `:161-167` (`getSeriesSeasons`/`getSeasonEpisodes`).
- Detail route: `routes/media/$type.$id.tsx` — `LocalDetailView` credits `~:307-312` then `SeasonAccordion` `~:314-326`; `TMDbDetailView` credits `~:507-511`; owned-badge in `TMDbDetailView` `~:478-501`.
- Types: `types/library.ts` `LibraryMovie`/`LibrarySeries` (have `tmdbId`); TMDB `Movie`/`TVShow` in `types/tmdb.ts`.

### UX Design Note

Epic 12 has **no `ux-design.pen` screen** for the detail-page recommendations section (same gap 12-2 noted for the accordion). Follow these patterns:
- Section: "相關推薦" heading consistent with the `CreditsSection` heading style above it.
- Tiles: reuse `PosterCard` (already designed/baselined) inside `MediaGrid` layout — no new visual primitives.
- Owned badge: reuse `AvailabilityBadge` / `PosterCard isOwned` (Story 10-4 design).
- `RelatedContent.tsx` carries the Rule 21 design-coverage-gap `// Design ref:` header.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.** `RelatedContent.tsx` renders static tiles from server data (poster/title/year/rating/owned-flag); it performs no ambient-now read or date-boundary branching. The reused `PosterCard` `isNew` badge (which may read the clock) is **pre-existing and already governed by its own Rule-23 disposition** — this story neither adds nor modifies that logic. No new fixture-state baselines required.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-3 (P2-022)]
- [Source: _bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md — Decisions 1 (Five Pillars), 2 (per-story mapping), 3 (no shared package)]
- [Source: project-context.md — Rules 3, 4, 5, 6, 7, 10, 11, 13, 14, 15, 16, 18, 21, 27]
- [Source: apps/api/internal/repository/movie_repository.go:907-968 — FindOwnedTMDbIDs (Story 10-4)]
- [Source: apps/api/internal/repository/series_repository.go:855-915 — FindOwnedTMDbIDs (Story 10-4)]
- [Source: apps/api/internal/tmdb/movies.go:47-74 — endpoint-wrapper pattern]
- [Source: apps/api/internal/tmdb/types.go:75-89 — SearchResultMovies/SearchResultTVShows]
- [Source: apps/web/src/components/media/PosterCard.tsx:15-36 — isOwned prop (Story 10-4)]
- [Source: apps/web/src/hooks/useMediaDetails.ts:29-50 — useSeriesSeasons/useSeasonEpisodes templates (Story 12-2)]
- [Source: _bmad-output/implementation-artifacts/12-2-season-episode-list.md — TMDB chain threading precedent]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **Anticipated at authoring time (dev to confirm/triage during implementation):**
    - **③ backlog-with-carry-forward-link (candidate):** Owned recommendation tiles deep-link to the TMDB-numeric `TMDbDetailView`, not the local-UUID `MediaDetailView` (AC #7, Critical Detail #4). Resolving TMDB-id → local-UUID for owned tiles is out-of-scope and mirrors existing Story 10-4 homepage behavior. If product wants owned tiles to open the local detail view, the dev MUST file a `backlog` `sprint-status.yaml` entry AT DISCOVERY with a bidirectional link before close (do not leave it prose-only — Rule 24 ban). If product accepts current behavior, record `N/A`.
  - Otherwise, record each genuine in-flight discovery here with its lane (①/②/③) + tracked entry ID / added AC # before marking done. If none: `N/A — no out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List

## Change Log

| Date | Change |
|------|--------|
| 2026-06-11 | Story drafted (SM Bob, create-story yolo). F-3 — TMDB recommendations/similar with "已有" owned badges. Backend: TMDB endpoint wrappers (recommendations+similar, movie+TV) threaded through fallback/cache (24h) → `RecommendationService` (server-side similar-fallback + batched `FindOwnedTMDbIDs` ownership annotation, reuse Story 10-4) → TMDB handler routes. Frontend: `useRecommendations` hook + `RelatedContent` section reusing `PosterCard`/`MediaGrid`/`isOwned`, rendered below credits in both detail views. Rule 27 Five-Pillars compliant (rides existing TMDB limiter/cache/`TMDB_*` errors, no new infra/secret/prefix). Cross-stack split: backend 3 / frontend 4 → single story. |
