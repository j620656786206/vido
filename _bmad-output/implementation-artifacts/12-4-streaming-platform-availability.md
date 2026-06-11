# Story 12.4: Streaming Platform Availability — TMDB Watch Providers

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a media library user viewing a movie or TV series detail page,
I want to see which streaming platforms (Netflix, Disney+, Apple TV+, etc.) the title is available on in my region,
so that I can decide where to watch it without leaving the app.

## Acceptance Criteria

1. **Given** a movie or TV detail page with a valid TMDB id (`tmdb_id > 0`), **when** the page loads, **then** a "可在哪裡觀看" streaming-availability section renders (below the overview, above the credits) showing the platform logos available for the title in the configured region (default **TW**).
2. **Given** TMDB returns watch-provider data for the region, **then** the section shows each provider's logo (via TMDB `logo_path`) and name, grouped by monetization type — **訂閱 (flatrate)** shown primarily, with **租借 (rent)** / **購買 (buy)** shown when present.
3. **Given** the region has a TMDB watch `link`, **then** the section provides a "前往 TMDB 觀看頁" outbound link (TMDB's `Results[region].link`).
4. **Given** the title has NO watch-provider data for the region (TMDB `results` omits it or returns empty groups), **then** the section renders a quiet empty-state (e.g. "此區域暫無串流資訊") or is omitted — it MUST NOT error or break the page (Rule 27 Pillar 3 — enrichment-not-core, fail-soft).
5. **Given** the TMDB API is unavailable / times out / errors, **then** the section degrades to an empty-state (optional retry) and the **rest of the detail page renders unaffected** (per-section isolation, Rule 27 Pillar 3).
6. **Given** TMDB/JustWatch licensing terms, **then** the section displays the **"資料來源：JustWatch"** attribution required for TMDB watch-provider data.
7. **Given** a mobile viewport, **then** the provider logos wrap responsively and remain tappable/readable (flex-wrap row, no new responsive primitives).
8. **Given** repeated visits to the same title, **then** watch-provider data is served from cache (24h TTL, cache checked **before** the rate limiter per Rule 27 Pillar 2), keeping warm detail-page load < 1.5s.

## Tasks / Subtasks

### Backend

- [x] **Task 1: Re-add TMDB `GetWatchProviders` client method + types + cache (24h)** (AC: #1, #2, #3, #8)
  - [x] 1.1 Re-create `apps/api/internal/tmdb/watch_providers.go` faithfully restoring the code **removed in commit `fdbf249` (Story 11-1 dead-code deletion)** — now that F-4 is the real consumer (ADR Decision 2 / Risk row). Restore: `WatchProvider` (`ProviderID`, `ProviderName`, `LogoPath *string`, `DisplayPriority`), `WatchProviderRegion` (`Link`, `Flatrate`, `Rent`, `Buy`), `WatchProvidersResponse` (`ID`, `Results map[string]WatchProviderRegion`), and `GetWatchProviders(ctx, mediaType string, id int, region string) (*WatchProvidersResponse, error)` (endpoint `/{media_type}/{id}/watch/providers`, validates `mediaType ∈ {movie,tv}` + `id>0`, filters `Results` to the single `region` when non-empty). **DO NOT** restore the dead `TWWatchProviderIDs` hardcoded map — providers resolve dynamically (ADR "add only what F-4 renders"; YAGNI). ✅ Restored minus `TWWatchProviderIDs`.
  - [x] 1.2 Add `GetWatchProviders` to `ClientInterface` (`client.go`, alongside `GetMovieVideos` at `:56`).
  - [x] 1.3 **Caching (Rule 27 Pillar 2 — diverges from `GetMovieVideos` which is uncached):** thread a cached wrapper through `cache.go` at **24h TTL** (`DefaultCacheTTL`) — ADR Pillar 2 table mandates 24h for watch providers (catalogs change daily at most). Cache key `tmdb:watchproviders:{movie|tv}:{id}:{region}:v1` (include region — provider availability is region-specific). Cache checked BEFORE `limiter.Wait`. ✅ Raw client injected via `SetProvidersClient` (bypasses fallback).
  - [x] 1.4 **NOT** added to `LanguageFallbackClient` (`fallback.go`) — watch-provider names/data are language-neutral (same call as `GetMovieVideos`); no language fallback needed.
  - [x] 1.5 Add `GetWatchProviders` to `TMDbServiceInterface` + `TMDbService` impl (`services/tmdb_service.go`, mirror `GetMovieVideos` at `:385-407`): nil-client guard, `slog` error logging, returns the cached result. Region defaults to **"TW"** when caller passes empty (single tuning point).
  - [x] 1.6 Restore + adapt tests from the removed `watch_providers_test.go` (region filtering, bad media-type/id validation) + cache test (24h) + service test. Reuse `TMDB_*` Rule-7 error codes (Pillar 4 — **no new prefix**).

- [x] **Task 2: Handler endpoints + routes + wiring** (AC: #1, #4, #5)
  - [x] 2.1 Add handler methods to `apps/api/internal/handlers/tmdb_handler.go` (sibling to `GetMovieVideos` at `:300-327`): `GetMovieWatchProviders`, `GetTVWatchProviders`. Parse `:id` (TMDB numeric, validate `>0`), read optional `?region=` query (default "TW"), call `service.GetWatchProviders(ctx, "movie"|"tv", id, region)`, return `{ "success": true, "data": { ...WatchProvidersResponse } }` (Rule 3). On TMDB error, map via the existing `handleTMDbError` helper so the section can fail-soft. ✅ Region default applied at the service layer (single tuning point); handler forwards the raw query.
  - [x] 2.2 Register routes alongside the existing videos routes (`tmdb_handler.go:473-475`): `GET /api/v1/tmdb/movies/:id/watch/providers` and `GET /api/v1/tmdb/tv/:id/watch/providers`. **Rule 15 self-check**: confirm the exact existing `/tv/:id/videos` group prefix and match it. ✅ Registered inside the same `tmdbGroup` block; route-registration test asserts both paths.
  - [x] 2.3 Add `GetWatchProviders` to the handler's `TMDbServiceInterface` (`tmdb_handler.go:13-26`). Swaggo annotations on both endpoints (Rule 15 Swagger sync). Wiring in `cmd/api/main.go` is already satisfied. ✅ No new DI — no generated swagger artifacts are committed (annotations source-only).
  - [x] 2.4 Handler tests: success (region populated), region-empty→empty-state, TMDB-error→fail-soft mapping. ✅ Plus invalid-id→400.

### Frontend

- [x] **Task 3: Types + service method** (AC: #1, #2, #3)
  - [x] 3.1 Add to `apps/web/src/types/library.ts` (or `types/tmdb.ts`): `WatchProvider { providerId: number; providerName: string; logoPath: string | null; displayPriority: number }`, `WatchProviderRegion { link: string; flatrate?: WatchProvider[]; rent?: WatchProvider[]; buy?: WatchProvider[] }`, `WatchProvidersResponse { id: number; results: Record<string, WatchProviderRegion> }`. ✅ Added to `types/library.ts`.
  - [x] 3.2 Add to `apps/web/src/services/libraryService.ts` (mirror `getSeasonEpisodes` at `:165`): `getMovieWatchProviders(tmdbId: number, region = 'TW')` → `fetchApi('/tmdb/movies/${tmdbId}/watch/providers?region=${region}')` and `getTVWatchProviders(tmdbId, region = 'TW')` → `/tmdb/tv/...`. Case-transform auto via `fetchApi`/`snakeToCamel` (Rule 18). ✅ **Rule 18 caveat VERIFIED:** `snakeToCamel`'s `_([a-z])` rewrite leaves uppercase region map keys ("TW"/"US") untouched → `results` stays a faithful `Record<string, WatchProviderRegion>`; inner `provider_id`/`logo_path`/`display_priority` camelCase correctly.

- [x] **Task 4: `useWatchProviders` hook** (AC: #1, #8)
  - [x] 4.1 Add to `apps/web/src/hooks/useMediaDetails.ts` (mirror `useSeriesSeasons` at `:29`): `useWatchProviders(tmdbId, type, enabled, region = 'TW')` → `useQuery` calling the matching service method.
  - [x] 4.2 Add `detailKeys.watchProviders(tmdbId, type, region)` to the query-key factory.
  - [x] 4.3 `staleTime: 24 * 60 * 60 * 1000` (24h — matches backend cache). `enabled: enabled && tmdbId > 0`.

- [x] **Task 5: `StreamingAvailability` section component** (AC: #1, #2, #3, #4, #6, #7)
  - [x] 5.1 Create `apps/web/src/components/media/StreamingAvailability.tsx`. Props: `region?`, `regionCode?`, `isLoading`, `isError`, `onRetry`. Header: "可在哪裡觀看".
  - [x] 5.2 Flex-wrap rows (modeled on `TechBadgeGroup`). Group flatrate / rent / buy with `訂閱` / `租借` / `購買` sub-labels; logos via `getImageUrl(logoPath, 'w92')` with `alt={providerName}`. Sort each group by `displayPriority`. ✅ Logo-missing → readable name chip (never a broken image).
  - [x] 5.3 The `link` → "前往 TMDB 觀看頁" outbound `<a target="_blank" rel="noopener noreferrer">` (AC #3).
  - [x] 5.4 **JustWatch attribution (AC #6):** "資料來源：JustWatch" rendered beneath the providers.
  - [x] 5.5 Loading skeleton; error → quiet `role="alert"` empty-state with retry (AC #5); no providers → muted "此區域暫無串流資訊" (AC #4). Never throws (region undefined → empty-state).
  - [x] 5.6 Rule 21 header — design-coverage-gap `// Design ref:` form present.
  - [x] 5.7 Write `StreamingAvailability.spec.tsx` — 10 tests (groups/logos, displayPriority sort, logo fallback, outbound link rel/target, JustWatch attribution, skeleton, retry on error, empty-state, undefined region). Rule 16 matchers (`toBeInTheDocument`).

- [x] **Task 6: Integrate into the detail page** (AC: #1)
  - [x] 6.1 In `apps/web/src/routes/media/$type.$id.tsx`, render `<StreamingAvailability />` **below the overview, above `CreditsSection`** in BOTH `LocalDetailView` AND `TMDbDetailView` — consistent placement.
  - [x] 6.2 Resolve the TMDB id + type per view (LocalDetailView `localData.tmdbId`; TMDbDetailView numeric route id). `useWatchProviders(tmdbId, type, tmdbId > 0, 'TW')`; extract `data?.results?.['TW']` for the component.

## Dev Notes

### Architecture Compliance

- **Rule 4 / Rule 11 (Layered Architecture):** `TMDbHandler.GetMovieWatchProviders` → `TMDbService.GetWatchProviders` → `CacheService` → `Client.GetWatchProviders`. No new service needed (unlike 12-3 — there is no cross-domain ownership join here; this is a pure TMDB passthrough like `GetMovieVideos`).
- **Rule 5 (TanStack Query):** fetched via `useWatchProviders` `useQuery`, gated `enabled: tmdbId > 0`.
- **Rule 6 (Naming):** endpoints `/api/v1/tmdb/movies/:id/watch/providers`, `/api/v1/tmdb/tv/:id/watch/providers`; Go `GetWatchProviders`; TS `getMovieWatchProviders`/`useWatchProviders`; JSON `provider_id`/`logo_path` (snake) ↔ `providerId`/`logoPath` (camel via Rule 18).
- **Rule 7 (Error Codes) + Rule 27 Pillar 4:** reuse `TMDB_*` ONLY — **no new prefix** (ADR Pillar 4 — "F-3/F-4 reuse TMDB_*").
- **Rule 10 (API Versioning):** all new routes under `/api/v1/`.
- **Rule 13 (Error Handling):** propagate TMDB errors as `AppError` via `handleTMDbError`; the frontend section fails soft.
- **Rule 14 / Rule 27 Pillar 1:** rides the **existing shared TMDB limiter** (40/10s) — F-4 adds ZERO new rate budget (ADR Pillar 1).
- **Rule 15 (Pre-commit Self-verification):** register both routes; Swaggo annotations; **verify the exact `/tv/:id/...` route prefix**; confirm `main.go` wiring (existing `TMDbService`/`TMDbHandler` already wired — F-4 adds methods, not new DI unless a dep is missing).
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` for logos/links; `toBeAttached()` for hover states.
- **Rule 18 (Case Transform):** auto via `fetchApi`. **Watch out:** TMDB `results` is a map keyed by region code ("TW"); `snakeToCamel` must not mangle the region-code keys — confirm the case-transform only touches object property names it recognizes, OR keep `results` as `Record<string, …>` and only transform the inner `WatchProviderRegion` shape. **Flag for the dev to verify in Task 3.**
- **Rule 21 (Component↔Design):** `StreamingAvailability.tsx` uses the design-coverage-gap `// Design ref:` form.
- **Rule 27 (External Integration Standard — Five Pillars):** ✅ ① rate limit — shared TMDB limiter, no new bucket · ✅ ② cache — 24h tiered, before limiter, key `tmdb:watchproviders:{movie|tv}:{id}:{region}:v1` (note: diverges from videos' no-cache — ADR Pillar 2 table explicitly sets 24h here) · ✅ ③ degrade — per-section fail-soft, empty-state on no-data/error, page never fails · ✅ ④ error codes — reuse `TMDB_*`, no new prefix · ✅ ⑤ keys — existing TMDB `ClientConfig.APIKey`, no new secret. [Source: ADR `adr-external-api-integration-standard.md` Decision 1 Pillar 2 table + Decision 2 F-4 row + Risk row "F-4 re-introduces the dead-code 11-1 removed"]

### Cross-Stack Split Check (MANDATORY — Agreement 5 / Epic 9c Retro AI-1)

Backend tasks: **2** (Task 1 client+types+cache+service, Task 2 handler/routes). Frontend tasks: **4** (Task 3 types/service, Task 4 hook, Task 5 component, Task 6 integration).

Threshold is "BOTH counts > 3". Backend = 2 (**not** > 3) → **NO split required. Single story.** (Backend is lighter than 12-3 because there is no ownership join — watch providers is a pure TMDB passthrough like `GetMovieVideos`.)

### Project Structure Notes

**Files to CREATE:**
- `apps/api/internal/tmdb/watch_providers.go` (+ `watch_providers_test.go`) — **restore from commit `fdbf249`** (minus the dead `TWWatchProviderIDs` map)
- `apps/web/src/components/media/StreamingAvailability.tsx` (+ `StreamingAvailability.spec.tsx`)

**Files to MODIFY:**
- `apps/api/internal/tmdb/client.go` — `ClientInterface` `GetWatchProviders`
- `apps/api/internal/tmdb/cache.go` (+ `cache_test.go`) — cached wrapper (24h) + cache test
- `apps/api/internal/services/tmdb_service.go` — `TMDbServiceInterface` + impl (+ update sibling service-test mocks embedding `TMDbServiceInterface`, as 12-2 did)
- `apps/api/internal/handlers/tmdb_handler.go` (+ `tmdb_handler_test.go`) — handler methods + routes + interface method
- `apps/web/src/types/library.ts` (or `types/tmdb.ts`) — `WatchProvider`, `WatchProviderRegion`, `WatchProvidersResponse`
- `apps/web/src/services/libraryService.ts` — `getMovieWatchProviders`, `getTVWatchProviders`
- `apps/web/src/hooks/useMediaDetails.ts` — `useWatchProviders` + `detailKeys.watchProviders`
- `apps/web/src/routes/media/$type.$id.tsx` — render `<StreamingAvailability />` in both detail views

### Critical Implementation Details

1. **This is a restoration, not a green-field build (Task 1).** The exact `GetWatchProviders` + its types + tests existed and were deleted in commit `fdbf249` (`fix(11-1): … remove dead watch-provider code`). Recover the verbatim code with `git show fdbf249:apps/api/internal/tmdb/watch_providers.go` and `git show fdbf249^:apps/api/internal/tmdb/watch_providers.go` (the `^` parent has the file pre-deletion). Re-add faithfully; the ONLY intentional omission is the dead `TWWatchProviderIDs` shorthand map (providers come dynamically from the live response — ADR "add only what F-4 renders"). **The AC-Drift / reuse check will surface Story 11-1 — cite it as a deliberate re-introduction (the ADR Risk row sanctions it: "It now has a real consumer (F-4); YAGNI is satisfied").**

2. **Watch providers diverge from videos on caching (Task 1.3).** `GetMovieVideos` is deliberately UNcached (small, ephemeral). Watch providers ARE cached 24h — the ADR Pillar 2 table is explicit: "TMDB watch providers (F-4) | 24 h | provider catalogs change daily at most." Do NOT copy the no-cache decision from videos; cache it. Region is part of the cache key.

3. **Region = TW (Task 1.5 / 6.2).** TMDB's `/watch/providers` returns a `results` map keyed by every region the title is available in. F-4 filters to **TW** (Taiwan — matches the zh-TW product locale). The removed `GetWatchProviders` already does single-region filtering via its `region` param; default it to "TW" at the service layer (single tuning point) and pass `?region=TW` from the frontend. A future story could add a region picker — out of scope (see Discovery Triage).

4. **JustWatch attribution is mandatory (AC #6).** TMDB's watch-provider data is sourced from JustWatch; TMDB's terms require displaying the JustWatch attribution wherever this data appears. Render "資料來源：JustWatch" in the section. This is a compliance requirement, not optional polish.

5. **`snakeToCamel` vs the region-keyed map (Rule 18 caveat).** The response's `results` is `map[string]WatchProviderRegion` keyed by region code. Verify the frontend `snakeToCamel` transform does not corrupt the "TW"/"US" map keys (it should only camelCase known object property names, not arbitrary map keys). If it does, keep `results` typed as `Record<string, WatchProviderRegion>` and transform only the inner shape. Verify during Task 3.

6. **Two detail-view code paths (Task 6).** Like 12-3, watch providers are keyed by the TMDB numeric id, available in BOTH `LocalDetailView` (`tmdbId` field) and `TMDbDetailView` (numeric route id) — render in both, below the overview / above credits, consistently.

### Existing Code References

- **Removed code to restore:** commit `fdbf249` deleted `apps/api/internal/tmdb/watch_providers.go` (75 lines) + `watch_providers_test.go` (108 lines). Recover via `git show fdbf249^:apps/api/internal/tmdb/watch_providers.go`.
- TMDB sub-resource endpoint template: `tmdb/movies.go:281-298` (`GetMovieVideos`); chain — `client.go:56` (interface), `cache.go:14-26` (TTL constants; videos no-cache comment `:44-45`), `services/tmdb_service.go:385-407` (`GetMovieVideos` impl), `handlers/tmdb_handler.go:300-327` (handler) + `:473-475` (route registration).
- Rate limiter: `tmdb/client.go:16-23` (constants), `:121-125` (`limiter.Wait` first line of `doRequest`).
- Region default precedent: `tmdb/movies.go:202-213` (discover `watch_region` defaults to "TW").
- Frontend image helper: `apps/web/src/lib/image.ts:5-8` — `getImageUrl(path, size)`; works for `logo_path` (`getImageUrl(logoPath, 'w92')`).
- Badge-row component pattern: `components/media/TechBadgeGroup.tsx` (build array → flex-wrap render); `components/media/TechBadge.tsx`.
- Hook + key-factory template: `hooks/useMediaDetails.ts:12-24` (`detailKeys`), `:29-37` (`useSeriesSeasons`).
- Service template: `services/libraryService.ts:21-36` (`fetchApi`), `:161-167` (`getSeriesSeasons`/`getSeasonEpisodes`).
- Detail route slots: `routes/media/$type.$id.tsx` — LocalDetailView overview `~:300-305` → credits `~:307-312`; TMDbDetailView overview `~:503-505` → credits `~:507-511`.
- Sibling F-3 story (same TMDB-detail integration shape): `_bmad-output/implementation-artifacts/12-3-related-content-recommendations.md`.

### UX Design Note

Epic 12 has **no `ux-design.pen` screen** for the streaming-availability section. Follow these patterns:
- Section: "可在哪裡觀看" heading consistent with sibling detail-page section headings.
- Logos: TMDB provider logos via `getImageUrl(logoPath, 'w92')`, flex-wrap row modeled on `TechBadgeGroup`.
- Attribution: "資料來源：JustWatch" muted text below the logos.
- `StreamingAvailability.tsx` carries the Rule 21 design-coverage-gap `// Design ref:` header.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.** `StreamingAvailability.tsx` renders static provider logos/names/links from server data; no ambient-now read or date-boundary branching. No new fixture-state baselines required.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-4 (P2-023)]
- [Source: _bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md — Decision 1 (Pillar 2 24h watch-providers row), Decision 2 (F-4 row), Risk row (re-introduce 11-1 dead code)]
- [Source: project-context.md — Rules 3, 4, 5, 6, 7, 10, 13, 14, 15, 16, 18, 21, 27]
- [Source: git commit fdbf249 — removed watch_providers.go (Story 11-1 dead-code deletion)]
- [Source: apps/api/internal/tmdb/movies.go:281-298 — GetMovieVideos sub-resource template]
- [Source: apps/api/internal/handlers/tmdb_handler.go:300-327,473-475 — handler + route template]
- [Source: apps/web/src/lib/image.ts:5-8 — getImageUrl for logos]
- [Source: apps/web/src/components/media/TechBadgeGroup.tsx — badge-row component model]
- [Source: _bmad-output/implementation-artifacts/12-3-related-content-recommendations.md — sibling F-3 TMDB-detail integration]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / BMM dev-story workflow)

### Debug Log References

- Full Go backend suite green: `go test ./...` (all packages pass).
- Full web suite green: `nx test web` — 173 files / 2073 tests pass, no orphaned processes.
- `StreamingAvailability.spec.tsx` — 10/10 pass.

### Completion Notes List

- 🔗 AC Drift: NONE (checked: `watch.provider`/`WatchProvider`/`watch/providers`/`fdbf249` across `_bmad-output/implementation-artifacts/*.md` — only Story 11-1 references the symbol, and that was the dead-code DELETION this story deliberately reverses. The external HTTP wire contract was never shipped to a consumer in 11-1, so re-adding it is not a behavior DRIFT of any prior shipped AC. ADR Risk row explicitly sanctions the re-introduction: "It now has a real consumer (F-4); YAGNI is satisfied.")
- ♻️ Re-introduced dead code (NOT a discovery — flagged per Critical Detail #1): restoring `GetWatchProviders` + its types/tests reverses Story 11-1 CR-M1's deletion (commit `fdbf249`). This is ADR-sanctioned (Decision 2 F-4 row + Risk row). The ONLY intentional omission is the dead `TWWatchProviderIDs` hardcoded map (+ its `TestTWWatchProviderIDs`) — providers resolve dynamically from the live response (YAGNI). A future dead-code/drift audit should NOT re-flag `GetWatchProviders`.
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story or in the upstream Story 11-1 it references — 11-1 is pre-Rule-20 / implicit v0. This story defines a passthrough TMDB wire shape consumed only by its own frontend; no cross-story contract versioning applies.)
- 🎭 A11y Pre-Flight: PASS (2 components checked — `StreamingAvailability.tsx` + the `$type.$id.tsx` render sites; `eslint` on both → 0 jsx-a11y warnings introduced, 1 pre-existing `react-hooks/exhaustive-deps` warning at `$type.$id.tsx:171` unrelated to this story. The 4 recurring a11y classes: (1) responsive images — N/A, provider logos are fixed 40×40 `w92` thumbnails, not responsive hero images; srcSet would be over-engineering for tiny logos; (2) modal focus — N/A, no modal; (3) aria-live — error state carries `role="alert"`; the provider grid mirrors the sibling 12-3 RelatedContent which intentionally omits aria-live on the result grid; (4) custom-widget keyboard/ARIA — N/A, only native `<a>`/`<img>`/`<button>`. Section uses `aria-labelledby` to the visible `<h2>`; outbound link has `rel="noopener noreferrer"`; logos have `alt={providerName}`.)
- 🎨 UX Verification: design-coverage-gap (Rule 21) — Epic 12 has **no `ux-design.pen` screen** for the streaming-availability section, so there is no design screenshot to diff against. `StreamingAvailability.tsx` carries the `// Design ref:` design-gap header; visual approach follows the documented patterns (heading consistent with sibling detail sections, flex-wrap logo rows modeled on `TechBadgeGroup`, muted "資料來源：JustWatch" attribution). No discrepancies possible against a non-existent screen.
- 🧱 Pre-existing failures observed (NOT introduced by this story): (a) `nx typecheck web` fails with ~90 errors in `routes/test/-gallery.fixtures.tsx` + `releaseDate`-on-union errors in `$type.$id.tsx` — all present on `main` (verified via `git stash`); typecheck is not a CI gate (main itself does not pass it). (b) Three touched Go files (`tmdb/client.go`, `tmdb/fallback_test.go`, `services/enrichment_nfo_test.go`) were already gofmt-dirty on `HEAD` (verified) — the project does not enforce gofmt; `gofmt -w` was deliberately NOT run to avoid injecting unrelated struct-realignment churn. My added lines are internally gofmt-consistent. Per Epic-9c Retro AI-2: these are large/unrelated pre-existing conditions — documented here, not silently skipped; no new tracking entry filed because they are platform-wide and outside this story's scope.
- 🔌 Caching design note: `CacheService` gained a raw `providersClient ClientInterface` injected via the new exported `SetProvidersClient` setter (wired in `NewTMDbService`). This mirrors the existing `SetContentFilter` injection precedent and keeps the 16 existing `NewCacheService` test call-sites unchanged. Watch providers deliberately bypass `LanguageFallbackClient` (Task 1.4 — data is language-neutral) while still being cached 24h (Task 1.3, diverging from the uncached `GetMovieVideos`).
- 🔍 Adversarial CR (2026-06-11, Amelia) — fixes applied (auto-fix path):
  - **MEDIUM #1 — region case normalization:** `TMDbService.GetWatchProviders` now `strings.ToUpper`s the region after the empty→TW default. A lowercase `?region=tw` previously missed TMDb's uppercase `Results` keys (silent empty) AND polluted the cache under a distinct lowercase key. Added `TestTMDbService_GetWatchProviders_NormalizesRegionToUpper`.
  - **LOW #2/#3 — dead prop + multi-source 'TW' literal:** removed the never-destructured `regionCode?` prop from `StreamingAvailability` (ironic dead code in a restore story); introduced a single module-level `WATCH_REGION = 'TW'` const in `$type.$id.tsx` used for BOTH the `useWatchProviders` arg AND the `results?.[WATCH_REGION]` lookup in both detail views — no more drift between hook region and lookup key.
  - **LOW #4 — halfwidth comma in error string:** NOT changed — `無法載入串流資訊,請稍後再試。` matches its direct siblings (`SeasonAccordion`/`RelatedContent`); deferred to a future codebase-wide punctuation sweep, not this story.
  - Verification: Go `tmdb`/`services`/`handlers` green (5/5 watch-provider service tests incl. new normalization test); `StreamingAvailability.spec.tsx` 10/10; prettier + gofmt clean on touched files. Pre-existing flaky `TestScannerService_SSEBroadcast_ScanCancelled` (order-dependent, passes in isolation) is unrelated to this story.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **Region picker (anticipated lane ③):** Region is hardcoded to **TW** (service-layer default, single tuning point; `?region=` is already plumbed end-to-end). Product has NOT requested a user-selectable region picker, so per the authoring guidance this is recorded as **N/A** (TW-only accepted for Epic 12). The plumbing (`region` param on client/service/handler/hook/service-method + region in cache key) means a future region-picker story is a pure-frontend add with no backend change.
  - **Re-introduced dead code:** see Completion Notes ♻️ — ADR-sanctioned, not a discovery.
  - No other out-of-scope work discovered.
- Reference: `project-context.md` Rule 24.

- **Did this story discover any work outside its current scope?**
  - **Anticipated at authoring time (dev to confirm/triage during implementation):**
    - **③ backlog-with-carry-forward-link (candidate):** Region is hardcoded to **TW**. If product wants a user-selectable region picker (US / JP / etc.), the dev MUST file a `backlog` `sprint-status.yaml` entry AT DISCOVERY with a bidirectional link before close (Rule 24 ban on prose-only mentions). If TW-only is accepted, record `N/A`.
    - **Re-introduced dead code (NOT a discovery, but flag in Completion Notes):** restoring `GetWatchProviders` reverses Story 11-1's deletion — note this explicitly as ADR-sanctioned (Risk row), so a future drift/dead-code audit doesn't re-flag it.
  - Otherwise record each genuine in-flight discovery here with its lane (①/②/③) + tracked entry ID / added AC # before marking done. If none: `N/A — no out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List

**Backend — created:**
- `apps/api/internal/tmdb/watch_providers.go` (restored from `fdbf249^`, minus `TWWatchProviderIDs`)
- `apps/api/internal/tmdb/watch_providers_test.go` (restored, minus `TestTWWatchProviderIDs`)

**Backend — modified:**
- `apps/api/internal/tmdb/client.go` (`ClientInterface.GetWatchProviders`)
- `apps/api/internal/tmdb/cache.go` (`providersClient` field + `SetProvidersClient` setter + cached `GetWatchProviders` + interface method)
- `apps/api/internal/tmdb/cache_test.go` (3 watch-providers cache tests: miss-then-hit/24h-TTL, region-in-key, nil-client)
- `apps/api/internal/tmdb/fallback_test.go` (`MockClient.GetWatchProviders` + configurable fields/counter)
- `apps/api/internal/services/tmdb_service.go` (`TMDbServiceInterface` + impl + `SetProvidersClient` wiring in `NewTMDbService`)
- `apps/api/internal/services/tmdb_service_test.go` (`MockCacheService.GetWatchProviders` + 4 service tests; `fmt` import)
- `apps/api/internal/services/enrichment_nfo_test.go` (`mockTMDbServiceForNFO.GetWatchProviders`)
- `apps/api/internal/services/explore_block_service_test.go` (`mockTMDbServiceForExplore.GetWatchProviders`)
- `apps/api/internal/handlers/tmdb_handler.go` (interface method + `GetMovieWatchProviders`/`GetTVWatchProviders` handlers + routes + Swaggo)
- `apps/api/internal/handlers/tmdb_handler_test.go` (`MockTMDbService.GetWatchProviders` + 5 handler tests + route assertions; `fmt` import)

**Frontend — created:**
- `apps/web/src/components/media/StreamingAvailability.tsx`
- `apps/web/src/components/media/StreamingAvailability.spec.tsx`

**Frontend — modified:**
- `apps/web/src/types/library.ts` (`WatchProvider`, `WatchProviderRegion`, `WatchProvidersResponse`)
- `apps/web/src/services/libraryService.ts` (`getMovieWatchProviders`, `getTVWatchProviders`)
- `apps/web/src/hooks/useMediaDetails.ts` (`useWatchProviders` + `detailKeys.watchProviders`)
- `apps/web/src/routes/media/$type.$id.tsx` (`useWatchProviders` + `<StreamingAvailability />` in both detail views)

**AC drift reference:**
- `_bmad-output/implementation-artifacts/11-1-multi-dimensional-filter-engine.md` (dead-code deletion reversed — see Completion Notes ♻️; not a drift)

## Change Log

| Date | Change |
|------|--------|
| 2026-06-11 | Task 1 (backend client+cache+service): **restored** `GetWatchProviders` + types + tests from commit `fdbf249^` (minus dead `TWWatchProviderIDs`); threaded a 24h region-keyed cache through `CacheService` (raw client injected via new `SetProvidersClient`, bypassing language fallback). Tests: client (restored), cache (miss→hit/24h/region-key/nil-client), service (region-default-TW, validation, error propagation). |
| 2026-06-11 | Task 2 (handler): added `GetMovieWatchProviders`/`GetTVWatchProviders` + routes `/tmdb/{movies,tv}/:id/watch/providers` + handler `TMDbServiceInterface` method + Swaggo. Tests: success (movie/tv), empty-region forward, invalid-id→400, TMDB-error→mapped status, route registration. Full Go suite green. |
| 2026-06-11 | Tasks 3–6 (frontend): added `WatchProvider*` types, `getMovie/TVWatchProviders` service methods, `useWatchProviders` hook + `detailKeys.watchProviders`, `StreamingAvailability` section (flatrate/rent/buy logo groups via `getImageUrl w92`, displayPriority sort, TMDB watch link, mandatory JustWatch attribution, loading/error/empty fail-soft states) rendered below overview / above credits in both detail views. Region TW. `StreamingAvailability.spec.tsx` 10 tests; full web suite (2073) green. Rule 18 region-key caveat verified safe. |
| 2026-06-11 | Adversarial CR (Amelia, code-review workflow): fixed MEDIUM region case-normalization (`strings.ToUpper` + test) and LOW dead-prop/multi-source cleanup (removed `regionCode`, single `WATCH_REGION` const). Status → done. Go + web tests green; prettier/gofmt clean. |
| 2026-06-11 | Story drafted (SM Bob, create-story yolo). F-4 — TMDB Watch Providers streaming availability. Backend: **restore** `GetWatchProviders` + types + tests removed in commit fdbf249 (Story 11-1), thread through cache (24h, region-keyed) → `TMDbService` → TMDB handler routes (`/tmdb/{movies,tv}/:id/watch/providers?region=TW`). Frontend: `useWatchProviders` hook + `StreamingAvailability` section (provider logos via `getImageUrl`, flatrate/rent/buy groups, TMDB watch link, mandatory JustWatch attribution), rendered below overview in both detail views. Region default TW. Rule 27 Five-Pillars compliant (rides existing TMDB limiter, 24h cache per ADR Pillar 2, reuse `TMDB_*` — no new infra/secret/prefix). Cross-stack split: backend 2 / frontend 4 → single story. |
