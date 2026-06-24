# Story (UX3): Discover Contextual Facet Counts — Backend

Status: ready-for-dev

<!-- Story key: ux3-discover-facet-aggregation-be (BE half of facet-aggregation).
     Source: tech-spec-ux3-discover-facet-aggregation.md (Tasks 1–5, AC1–AC7).
     Two BE↔FE contract decisions ratified via Party Mode 2026-06-24 (Winston/John/Murat/Bob): Q1=A, Q2=A. -->

## Story

As a **Discover-v2 user drilling through the TMDb-scale catalog**,
I want **the backend to return, for my current filter selection, the contextual result count for every candidate facet value (how many results if I ADD that facet)**,
so that **the Discover rail can show per-chip counts and grey out dead-end (0-result) facets — without me having to click each one to find out.**

(Backend half only. The frontend `FacetCountChip` revival is the separate consumer story `ux3-discover-facet-aggregation-fe`.)

## Acceptance Criteria

> Contract surface AC1/AC8/AC9 are stamped **`[@contract-v1]`** (Rule 20). The consumer `ux3-discover-facet-aggregation-fe` MUST `confirmed against [@contract-v1]` the request/response shape.

1. **AC1 (happy + contract) `[@contract-v1]`:** Given a base filter selection (existing discover query params) and per-dimension candidate value lists, when `GET /api/v1/tmdb/discover/facet-counts` is called, then it returns `ApiResponse<FacetCounts>` where `counts[dim][value]` = the summed **movie + tv** `total_results` for `(base params + that facet value added)`. Response shape:
   ```json
   { "success": true, "data": { "counts": { "genre": { "28": 340 }, "region": { "TW": 88 }, "rating": { "8": 95 }, "platform": { "8": 540 } }, "partial": false } }
   ```
   Outer key = dimension (`genre|region|rating|platform`); inner key = the facet value **as the FE supplied it** (string).
2. **AC2 (contextual semantics):** Given `genre=28` (動作) is already in the base selection, when facet-counts is requested with `region_values` including `KR`, then `counts.region.KR` reflects (動作 ∩ 韓國), NOT 韓國-alone. "Add" semantics per dimension: **multi-select** (`genre`→`GenreIDs`, `platform`→`WatchProviders`) **append** the value; **single-select** (`region`→`Region`, `rating`→`VoteAverageGte`) **replace/set** the value.
3. **AC3 (cache reuse) — reworded from tech-spec:** Given the same **normalized** facet sub-query (see AC9) was computed within the 1h TTL, when requested again, then **NO new TMDb call is made** — it is served from the cache via the existing `CacheService.DiscoverMovies/DiscoverTVShows` path. (Note: AR-F2 normalization makes count cache keys **distinct from the grid's** keys — counts share cache entries with other counts, not with the result grid. Asserted via mock call-capture.)
4. **AC4 (rate isolation):** Given a fan-out across many facet values, when it runs, then concurrent facet-counts TMDb calls never exceed `N` via `errgroup` `g.SetLimit(N)` (N≈4, tuned in impl), so interactive TMDb calls (detail / search / homepage) are not starved.
5. **AC5 (time-budget partial):** Given some sub-queries are cold and the fan-out exceeds the ~800ms budget, when the endpoint responds, then it returns the counts resolved within budget with `partial: true` (unresolved facet keys omitted) and never blocks beyond the budget. Cached facets (1h) resolve instantly, so steady state is rarely partial.
6. **AC6 (fail-soft):** Given one facet sub-query errors (e.g. `TMDB_TIMEOUT`), when computing, then that single facet is omitted and the rest still return — the endpoint never fails the page (Rule 27 ③ / Rule 13: the error is swallowed+logged at the per-facet boundary, not propagated).
7. **AC7 (validation passthrough — no new error code):** Given `year_gte > year_lte` (or `vote_gte > vote_lte`, or a local-only `sort` key), when facet-counts is requested, then it returns the **existing** `TMDB_INVALID_YEAR_RANGE` / `TMDB_INVALID_VOTE_RANGE` / `TMDB_UNSUPPORTED_SORT` error (Rule 7 — **NO new error code**; reuses `parseDiscoverParams`).
8. **AC8 (candidate ownership — Q1=A) `[@contract-v1]`:** Given the request supplies per-dimension candidate value CSVs, when counts are computed, then the BE computes counts **ONLY for the FE-supplied candidate values** and holds **NO facet inventory of its own** (the curated 18 genres / 5 regions / 4 ratings / 3 platforms remain owned by FE `discoverFilters.ts`). A dimension with no `*_values` supplied is **absent** from `counts` (not enumerated server-side).
9. **AC9 (count-param normalization — AR-F2/F3) `[@contract-v1]`:** Given any facet sub-query, when it is issued, then `SortBy` is set to `""`, `Page` to `1`, and `Language` to an explicit pinned value **before** calling Discover — so all sorts/pages share one cache entry (AR-F2) and the fallback-language chain is not fanned out (AR-F3). Counts are documented as per-locale.
10. **AC10 (AR-F3 fallback-bypass verification — tracked):** Given the explicit-`Language` requirement of AC9, the dev MUST **verify** whether `LanguageFallbackClient.DiscoverMoviesWithFallback` short-circuits the language chain when `Language` is non-empty (vs. still looping). The finding is documented in the Dev Agent Record. If the fallback layer **cannot** be cleanly made to issue a single `/discover` call per probe, the dev **expands scope per Rule 24** (new sub-task — e.g. route count probes through the raw `Client` / `providersClient`) rather than silently fanning out N language calls per facet.

## Tasks / Subtasks

- [ ] **Task 1 — `FacetCounts` response type (AC: #1)**
  - [ ] File: `apps/api/internal/tmdb/types.go`.
  - [ ] `type FacetCounts struct { Counts map[string]map[string]int \`json:"counts"\`; Partial bool \`json:"partial"\` }`.
- [ ] **Task 2 — `DiscoverFacetCounts` on `CacheService` (AC: #1, #2, #3, #4, #5, #6, #8, #9, #10)**
  - [ ] File: `apps/api/internal/tmdb/cache.go` (+ add the method to `CacheServiceInterface`, `:45-77`).
  - [ ] Signature (Q1=A, contract): `DiscoverFacetCounts(ctx context.Context, base DiscoverParams, candidates map[string][]string) (*FacetCounts, error)`. `candidates` key = dimension (`"genre"|"region"|"rating"|"platform"`), value = the candidate values **as strings** (parsed per-dim inside).
  - [ ] Define dimension constants (e.g. `dimGenre = "genre"`, `dimRegion`, `dimRating`, `dimPlatform`).
  - [ ] For each `(dim, value)`: clone `base`, **add** the facet per AC2 semantics — genre/platform `append` (atoi the value); region `set Region` (string); rating `set VoteAverageGte` (parseFloat). For platform, ensure `WatchRegion` is set (default `base.Region` → `"TW"`) since TMDb requires `watch_region` with `with_watch_providers`.
  - [ ] **Normalize each cloned param set (AC9, AR-F2/F3):** `SortBy=""`, `Page=1`, `Language=<pinned>` (use `base.Language` if set, else a default e.g. `"zh-TW"`).
  - [ ] **Reuse the existing cached path (Q2=A):** call `s.DiscoverMovies(gctx, p)` + `s.DiscoverTVShows(gctx, p)` (the existing `:374`/`:411` methods — they cache at `CacheTypeTMDb` + `TrendingDiscoverCacheTTL`); count = `movies.TotalResults + tv.TotalResults`. **NO new cache namespace** in this story (AR-F4/F5/F8 deferred → see Follow-ups).
  - [ ] **Fan-out:** `g, gctx := errgroup.WithContext(ctx)` wrapped by `context.WithTimeout(ctx, ~800ms)`; `g.SetLimit(N)` (N≈4). Each `(dim,value)` runs in `g.Go(func() error { ... })`: check `gctx.Err()`, compute the movie+tv sum, write into a **mutex-guarded** `map[string]map[string]int`, and **always `return nil`** (swallow per-facet errors → fail-soft, AC6). After `g.Wait()`, any requested `(dim,value)` NOT in the map → omit it and set `Partial=true` (AC5/AC6). Do NOT use errgroup's error to abort the whole fan-out.
  - [ ] `0` is a real (dead-end) count — keep it in the response (the FE dims-but-keeps-selectable). Do not treat `0` as "missing".
- [ ] **Task 3 — Service passthrough (AC: #1)**
  - [ ] File: `apps/api/internal/services/tmdb_service.go` — add `func (s *TMDbService) DiscoverFacetCounts(ctx, base, candidates) (*tmdb.FacetCounts, error) { return s.cacheService.DiscoverFacetCounts(ctx, base, candidates) }`.
  - [ ] Extend `TMDbServiceInterface` in `apps/api/internal/handlers/tmdb_handler.go:14-28` with the new method.
  - [ ] **Note:** counts use the **raw** `CacheService.DiscoverMovies` (no `ContentFilterService` applied), so counts are TMDb-reported `total_results` and may slightly exceed the content-filtered grid (tech-spec Decision #7 — counts are approximate; acceptable).
- [ ] **Task 4 — Handler + route (AC: #1, #7, #8)**
  - [ ] File: `apps/api/internal/handlers/tmdb_handler.go`.
  - [ ] `func (h *TMDbHandler) DiscoverFacetCounts(c *gin.Context)`: `base, err := parseDiscoverParams(c)` (reuse `:549-592` — gives validation + `*TMDbError` for free, AC7); on err → `handleValidationError(...)`. Parse candidate CSVs: `genre_values`, `region_values`, `rating_values`, `platform_values` (use `tmdb.ParseIntCSV` for genre/platform-ish where helpful, but keep them as `[]string` in the `candidates` map; region stays string). Build `candidates map[string][]string` (omit empty dims — AC8). Call `h.service.DiscoverFacetCounts(c.Request.Context(), base, candidates)`; on err → `handleTMDbError(...)`; else `SuccessResponse(c, result)`.
  - [ ] Register route after `:616` (after `discover.GET("/tv", h.DiscoverTVShows)`): `discover.GET("/facet-counts", h.DiscoverFacetCounts)` (group var is `discover`, `:613`).
  - [ ] Swagger annotations (mirror `DiscoverMovies` `:269` block); document the `*_values` params + the `FacetCounts` response. **No new `@Failure` code** beyond the existing `TMDB_INVALID_*` / `TMDB_UNSUPPORTED_SORT`.
- [ ] **Task 5 — BE tests (AC: all)**
  - [ ] Files: `apps/api/internal/tmdb/cache_test.go`, `apps/api/internal/handlers/tmdb_handler_test.go`.
  - [ ] cache_test.go (table-driven, reuse `MockCacheRepository` `:16-73` + `MockFallbackClient` call-counters): counts correctness; **movie+tv sum**; **contextual add semantics per dim** (append vs set, AC2); **cache reuse — no extra fallback-client call within TTL** (AC3, assert call counters); **normalization** — assert the params handed to the client have `SortBy=""`/`Page=1`/pinned `Language` (AC9); **`SetLimit` concurrency bound** (AC4 — e.g. a counting/blocking mock asserts ≤N in-flight); **partial on timeout** (AC5); **one-facet-fail fail-soft** (AC6, inject an error for one value → it's omitted, others present, `Partial=true`); **candidate-only** (AC8 — unsupplied dim absent); **`0` kept** (dead-end count present).
  - [ ] handler test (reuse `MockTMDbService` `:19-53` + `setupTMDbRouter`): candidate-CSV parse → `DiscoverFacetCountsCalls` capture; **validation passthrough** (`year_gte>year_lte` → `TMDB_INVALID_YEAR_RANGE`, AC7); `ApiResponse<FacetCounts>` shape; empty-candidates → empty `counts`.
  - [ ] `go mod tidy` (errgroup `SetLimit` needs `golang.org/x/sync` ≥ v0.1.0 — errgroup is already imported in `internal/subtitle/engine.go`, so the module is present; verify `SetLimit` resolves). Run `cd apps/api && go test ./internal/tmdb/ ./internal/handlers/ -run FacetCount -v`, then `pnpm lint:all` (go vet + staticcheck + eslint + prettier, Rule 12).

## Dev Notes

### Ratified design decisions (Party Mode 2026-06-24 — Winston/John/Murat/Bob)

- **Q1 = A (FE supplies candidates; BE is stateless).** The curated facet inventory (5 regions 台/韓/日/美/英, 4 rating thresholds, 3 platforms, 18 genres) is a **frontend product-curation artifact** — TMDb has no endpoint that enumerates "the 5 regions we show". Mirroring it in the BE = two editorial copies = guaranteed drift (Rule 24 class). So the request carries `{base filter}` + per-dimension candidate value CSVs; the BE counts exactly those. Future inventory changes (6th region, 4th platform) become **FE-only** changes — no BE redeploy (a real win on single-NAS). This refines the tech-spec's `dims []string` signature to `candidates map[string][]string`.
- **Q2 = A (reuse `DiscoverMovies/TVShows` as-is).** After normalizing `SortBy=""`/`Page=1`/explicit `Language` (AR-F2/F3, **done now**), reuse the existing cached discover methods — zero new caching code, and the `infra-cache-entries-expiry-sweep` prerequisite already sweeps `type="tmdb"` so growth is bounded. **Consequence (accepted for v1):** count entries share `type="tmdb"` + 1h TTL and **zeros are cached for ≤1h**. AR-F4 (distinct `type="tmdb_facet"`), AR-F8 (dedicated `FacetCountCacheTTL`), AR-F5 (skip-caching-zeros) all **require a divergent count-cache path** and are **deferred** → `ux3-facet-count-cache-refinement` (Rule-24 ③ backlog). Harm of a stale `0` is bounded: the FE keeps 0-facets **selectable** (a soft "dead-end" hint, not a block), and it self-corrects within the 1h TTL.
- **Synergy myth correction (Winston):** AR-F2 normalization makes count cache keys **distinct from the grid's** (the grid uses real `sort`/`page`). Counts do **NOT** ride the grid's warm cache — cache reuse is **count-to-count**. AC3 is worded accordingly.

### Prerequisite & consumer (dependency chain)

- **PREREQUISITE — `infra-cache-entries-expiry-sweep` (must land first).** This endpoint is a **write-amplifier** on `cache_entries` (many distinct facet-combo keys). The scheduled `ClearExpired` sweep is the bound on that growth. Do not ship facet-counts before the sweep is merged. (That story is `ready-for-dev`.)
- **CONSUMER — `ux3-discover-facet-aggregation-fe` (acks this contract).** The FE story builds `FacetCountChip` + the `useDiscoverFacetCounts` hook and MUST `confirmed against [@contract-v1]` the request (base + `*_values`) / response (`{counts:{dim:{value:int}}, partial}`) shape. The FE `discoverFilters.ts` enumeration helper (its Task 8) is the candidate source for the `*_values` lists.

### Verified code anchors (current line numbers — verbatim signatures)

- **`internal/tmdb/cache.go`**: `CacheService` struct `:34-43` + `NewCacheService` `:80-94`; `CacheServiceInterface` `:45-77` (add `DiscoverFacetCounts`); cached `DiscoverMovies` `:374-409` / `DiscoverTVShows` `:411-435` (both: `cacheKey := discoverCacheKey(...)` → `cache.Get` → on miss `client.Discover*WithFallback` → `cache.Set(ctx, key, value, CacheTypeTMDb, TrendingDiscoverCacheTTL)`); `discoverCacheKey` `:437-453` (**hashes** GenreIDs, YearGte/Lte, Region, VoteAverageGte/Lte, WatchProviders, WatchRegion, Language, SortBy, Page — **incl. SortBy + Page**, which is exactly why AR-F2 normalization matters); TTL consts `:14-26` (`CacheTypeTMDb="tmdb"`, `DefaultCacheTTL=24h`, `TrendingDiscoverCacheTTL=1h`); `cache.Set` signature = `Set(ctx, key, value, cacheType, ttl)`.
- **`internal/tmdb/types.go`**: `DiscoverParams` `:254-266` (`GenreIDs []int; YearGte/YearLte int; Region string; VoteAverageGte/Lte float64; WatchProviders []int; WatchRegion/Language/SortBy string; Page int`); `SearchResultMovies` `:75-81` / `SearchResultTVShows` `:83-89` (`TotalResults int \`json:"total_results"\``).
- **`internal/handlers/tmdb_handler.go`**: `TMDbServiceInterface` `:14-28`; `parseDiscoverParams` `:549-592` (returns `(DiscoverParams, error)` — `*TMDbError` on year/vote-range + unsupported-sort; **reuse for AC7**); `DiscoverMovies` handler `:269-281` (parse → `handleValidationError` → `service.DiscoverMovies` → `handleTMDbError` → `SuccessResponse`); `RegisterRoutes` discover group var `discover` `:613`, `DiscoverTVShows` registered `:616` (**add new route after**).
- **`internal/services/tmdb_service.go`**: `TMDbService` struct `:60-64` (`cacheService tmdb.CacheServiceInterface`); `NewTMDbService` `:70-110`; existing passthrough `DiscoverMovies` `:366-380` (note it applies `ContentFilterService` — the count path does NOT).
- **`internal/tmdb/errors.go`**: `TMDB_*` consts `:10-33` (incl. `ErrCodeInvalidYearRange`, `ErrCodeInvalidVoteRange`, `ErrCodeUnsupportedSort`); `TMDbError` type `:44+`; `NewInvalidYearRangeError()` `:152-159` (**reused — no new code**).
- **`internal/tmdb/client.go`**: `Client.limiter *rate.Limiter` `:95-101` (UNEXPORTED, one per Client, **40 req/10s burst 40**); `limiter.Wait(ctx)` first line of `doRequest` `:140-144`. The fan-out flows through this — it cannot exceed the limit, only serialize (cache is checked before the limiter, Rule 27 ②).
- **errgroup pattern**: imported in `internal/subtitle/engine.go:11` (`var g errgroup.Group; g.Go(func() error {...}); g.Wait()`). **`SetLimit` not yet used in the codebase** — this story introduces it (needs `golang.org/x/sync` ≥ v0.1.0).
- **Test mocks**: `MockTMDbService` `tmdb_handler_test.go:19-53` (Response/Error fields + call-capture slices like `DiscoverMoviesCalls []tmdb.DiscoverParams`); `MockCacheRepository` `cache_test.go:16-73` (`data map`, `setCalled`/`getCalled` counters, `lastSetTTL`); cache-reuse assertion precedent = `TestCacheService_GetTrendingMovies_CacheMissThenHit` `cache_test.go:754-779` (asserts upstream call-counter unchanged after a hit).

### Rule compliance

- Rule 1 (backend → `/apps/api`), Rule 4 (Handler→Service→Repository — handler calls `TMDbService`, never the cache directly), Rule 7 (**no new error code**; reuse `TMDB_INVALID_*`), Rule 11 (interface in services/cache package), Rule 9 (co-located `_test.go`), Rule 13 (per-facet errors swallowed+logged at the boundary, never propagated as page failure), Rule 14 (rate limiter reused, not recreated), Rule 18 (snake_case JSON; FE `fetchApi` does the case transform), Rule 20 (`[@contract-v1]` stamped on AC1/AC8/AC9), Rule 27 (Pillar ① rate limit via the existing limiter + `SetLimit`, ② cache checked before limiter, ③ fail-soft degrade, ④ reuse `TMDB_*`, ⑤ existing key mgmt).
- **Cross-stack split check:** 5 tasks, all backend; 0 frontend → single story, **no split** (FE is the separate `-fe` story).

### Project Structure Notes

- All edits under `apps/api/internal/{tmdb,services,handlers}` + co-located tests. No migration, no new error code, no `main.go` change (the endpoint hangs off the already-wired `TMDbHandler.RegisterRoutes`).

### Time-dependent visual coverage

- **N/A — no `apps/web/src/components/**/*.{ts,tsx}` touched.** Backend-only story (Go endpoint + service + tests); renders no UI, captures no visual baselines.

### References

- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md`] — full spec: Overview, Technical Decisions #1–#13, Tasks 1–5 (BE), AC1–AC7, Architecture Review AR-F1…F9, Investigation Findings (Step 2 anchors), response shape.
- [Source: tech-spec §"Architecture Review (applied — AR-F#)"] — AR-F2 (normalize sort/page), AR-F3 (pin Language), AR-F4/F5/F8 (deferred → `ux3-facet-count-cache-refinement`).
- [Source: `project-context.md` Rule 27 (External Integration Standard — Five Pillars), Rule 7 (error codes), Rule 20 (AC contract versioning), Rule 13/14, §8 SSE n/a] — governing rules.
- [Source: `apps/api/internal/tmdb/cache.go`, `types.go`, `client.go`, `errors.go`; `internal/handlers/tmdb_handler.go`; `internal/services/tmdb_service.go`] — anchors above (verified current line numbers, 2026-06-24).
- Party Mode ratification (2026-06-24): Q1=A (FE candidates), Q2=A (reuse-as-is) + two tracked follow-ups (this story's AC10 + `ux3-facet-count-cache-refinement`).

## Dev Agent Record

### Agent Model Used

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

<!-- MUST record the AC10 finding here: does DiscoverMoviesWithFallback short-circuit on explicit Language? What path did the count probes ultimately use? If scope was expanded per Rule 24, name the added sub-task. -->

### Discovery Triage

<!-- Rule 24 (project-context.md). -->

- **Did this story discover any work outside its current scope?** **YES — two items, both pre-triaged at story-creation (Party-Mode ratified 2026-06-24):**
  - **③ — AR-F4/F5/F8 count-cache fidelity (deferred refinement).** Reusing `DiscoverMovies/TVShows` as-is (Q2=A) means count entries share `type="tmdb"` + 1h TTL and cache zeros. A divergent count-cache path (distinct `type="tmdb_facet"`, dedicated `FacetCountCacheTTL`, skip-caching-`0`) is real but **non-blocking** (bounded harm — 0-facets stay selectable; prereq sweep bounds growth). Filed as **`ux3-facet-count-cache-refinement`** (backlog), bidirectional link. Lane ③.
  - **① — AR-F3 fallback-language-bypass verification (absorbed in-scope).** Whether `DiscoverMoviesWithFallback` short-circuits on explicit `Language` must be verified to implement AC9 correctly. Absorbed into THIS story as **AC10 + Task 2** (it is in-scope implementation work, tracked by an AC — lane ①, not a deferral). If verification fails (fallback can't issue a single call per probe), AC10 mandates a Rule-24 scope expansion at that moment (route probes through the raw `Client`).
- Reference: `project-context.md` Rule 24; Rule 20 (`[@contract-v1]`); origin: tech-spec AR-F# + Party-Mode design ratification (2026-06-24).

### File List

_(to be filled by dev agent)_

- `apps/api/internal/tmdb/types.go` (MODIFIED — `FacetCounts` type)
- `apps/api/internal/tmdb/cache.go` (MODIFIED — `DiscoverFacetCounts` + interface)
- `apps/api/internal/services/tmdb_service.go` (MODIFIED — passthrough)
- `apps/api/internal/handlers/tmdb_handler.go` (MODIFIED — handler + route + interface)
- `apps/api/internal/tmdb/cache_test.go` (MODIFIED — service tests)
- `apps/api/internal/handlers/tmdb_handler_test.go` (MODIFIED — handler tests)
