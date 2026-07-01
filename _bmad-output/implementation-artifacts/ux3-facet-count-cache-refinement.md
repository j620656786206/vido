# Story (UX3): Facet-Count Cache Fidelity Refinement (AR-F4 / F5 / F8)

Status: done

<!-- Promoted backlog → ready-for-dev 2026-06-30 (SM create-story re-run, *yolo). The hard
     block is CLEARED: ux3-discover-facet-aggregation-be merged (#100) and -fe merged (#101).
     Anchors below re-verified against the AS-MERGED apps/api/internal/tmdb/cache.go on
     2026-06-30 (line numbers updated). The "do-when-stale-0-annoys" conditionality is treated
     as satisfied by the explicit pickup; if you only wanted an anchor refresh, bounce the
     status back to backlog — the body is correct either way. -->


<!-- Story key: ux3-facet-count-cache-refinement.
     Rule-24 ③ follow-up deferred from ux3-discover-facet-aggregation-be (Party Mode Q2=A, 2026-06-24).
     Backend-only. Hard block CLEARED (see "Prerequisite" below). -->

## ✅ Prerequisite (CLEARED) & Conditionality

- **PREREQUISITE CLEARED — `ux3-discover-facet-aggregation-be` is MERGED (#100).** This story **refactors** the `CacheService.DiscoverFacetCounts` method that the BE story created (now live at `apps/api/internal/tmdb/cache.go:533`). The consumer `ux3-discover-facet-aggregation-fe` is also merged (#101). All anchors below were re-verified against the merged tree on 2026-06-30.
- **CONDITIONAL (do-when-needed) — treated as triggered.** The BE story (Q2=A) intentionally shipped the simpler **reuse-`DiscoverMovies/TVShows`-as-is** path. The harm it accepts is bounded (a transient/wrong-locale `0` dims a facet for ≤1h, and 0-facets stay **selectable**). This refinement is worth doing **when stale-`0` dimming proves annoying in practice**, or when count-cache freshness/eviction needs to be tuned independently of the grid. It is **not** required for the facet-counts feature to function — if the stale-`0` symptom has not actually been observed, this can be returned to backlog.

## Story

As the **Vido backend cache layer**,
I want **facet-count sub-queries cached on a dedicated path — distinct cache `type`, dedicated TTL, and zero-results not cached (or short-TTL'd)**,
so that **count freshness/eviction can be tuned independently of the result grid, and a transient or wrong-locale `0` never wrongly dims a facet chip for a full hour.**

## Background — what the BE story deferred and why

The BE story `ux3-discover-facet-aggregation-be` (as merged, #100) computes each facet count inside `DiscoverFacetCounts` (`cache.go:533`) by calling `s.DiscoverMovies(gctx, probe.param)` (`cache.go:588`) + `s.DiscoverTVShows(gctx, probe.param)` (`cache.go:594`) and summing `movies.TotalResults + tv.TotalResults` (`cache.go:604`) — i.e. it reuses `CacheService.DiscoverMovies` / `DiscoverTVShows` **as-is** (Party Mode Q2=A). Those two methods (`cache.go:425` / `:456`) hardcode:

- cache `type = CacheTypeTMDb = "tmdb"` (shared with the result grid), and
- `TTL = TrendingDiscoverCacheTTL = 1h` (const block `cache.go:17-29`), and
- they **cache every response, including `total_results == 0`** (`cache.go:447-451` / `:478-482` — `Set` is called unconditionally after a successful fetch).

The architect review flagged three fidelity gaps that all require **diverging from the reuse path** (so they were deferred, not done in the BE story):

| ID | Sev | Gap | This story's fix |
| -- | --- | --- | ---------------- |
| **AR-F8** | Low | counts borrow trending's TTL ("trending freshness" ≠ "count freshness") | dedicated `FacetCountCacheTTL` const, independently tunable |
| **AR-F4** | Med | counts share `type="tmdb"` with the grid → no targeted eviction | distinct `type="tmdb_facet"` → enables `ClearByType("tmdb_facet")` |
| **AR-F5** | Med | a transient/wrong-locale `0` cached the full 1h wrongly dims a facet | **do NOT cache `total_results==0`** (or cache with a short TTL) |

## Acceptance Criteria

1. **AC1 (AR-F8 — dedicated TTL):** A new `FacetCountCacheTTL` constant governs facet-count cache entries (may start `= 1h`), tunable **independently** of `TrendingDiscoverCacheTTL`. Changing the trending TTL does not change count freshness and vice-versa.
2. **AC2 (AR-F4 — distinct cache type):** Facet-count cache entries are tagged `type = "tmdb_facet"` (new `CacheTypeTMDbFacet` const), distinct from the grid's `"tmdb"`. This enables a targeted `CacheRepository.ClearByType(ctx, "tmdb_facet")` eviction of only the count entries without touching grid entries.
3. **AC3 (AR-F5 — zero not cached):** When a facet sub-query returns `total_results == 0`, it is **NOT written to the cache** (or written with a much shorter TTL). A subsequent identical probe re-fetches rather than serving a stale `0` for the full TTL — so a transient/wrong-locale `0` cannot dim a facet chip for an hour.
4. **AC4 (count-to-count reuse preserved):** The BE-story invariant still holds — given the same **normalized** (sort=''/page=1/pinned-Language) facet combo with a **non-zero** count requested within `FacetCountCacheTTL`, NO new TMDb call is made (served from the dedicated count cache). Asserted via mock call-capture.
5. **AC5 (storage — incidentally AR-F6):** The dedicated count cache stores the **integer count** (small), not the full `SearchResult*` JSON blob, reducing per-entry size for the count workload.
6. **AC6 (behavior parity):** `DiscoverFacetCounts`'s response (`counts` / `partial` semantics, contextual values, fail-soft, time-budget) is **unchanged** from the BE story EXCEPT the zero-skip (AC3) — all existing BE-story ACs (AC1/AC2/AC4/AC5/AC6/AC7) still pass against the refactored path.
7. **AC7 (deep isolation stays deferred — documented):** This story does the distinct-`type` tag + `ClearByType` targeted eviction (the shallow AR-F4 fix). **True purge-isolation** (the manual purge runs a wholesale `clearTable("cache_entries")`) still requires a **separate table** — that is Hard and remains **out of scope / deferred** (tech-spec AR-F4 note). The code/comment records this boundary.

## Tasks / Subtasks

- [x] **Task 1 — Constants (AC: #1, #2)**
  - [x] File: `apps/api/internal/tmdb/cache.go`. Add `CacheTypeTMDbFacet = "tmdb_facet"` next to `CacheTypeTMDb` (first const block, `:17-29`) and `FacetCountCacheTTL = 1 * time.Hour` with a comment that it is independently tunable from `TrendingDiscoverCacheTTL`. **Recommendation:** put `FacetCountCacheTTL` in the **second (facet) const block** (`:31-70`, which already holds `facetCountBudget`/`facetCountConcurrency`/`maxFacetProbes`/`defaultFacetCountLanguage`) so all facet-count knobs sit together; cross-reference `TrendingDiscoverCacheTTL` in the comment.
- [x] **Task 2 — Dedicated cached count path (AC: #3, #4, #5)**
  - [x] File: `apps/api/internal/tmdb/cache.go`. Add `facetCountCacheKey(kind string, p DiscoverParams) string` mirroring `discoverCacheKey` (**now at `:491-502`**, format `tmdb:discover/%s:g=…:sort=%s:p=%d`) but with a **distinct namespace** (e.g. `tmdb:facetcount/{kind}:…`) so count entries never collide with grid (`tmdb:discover/…`) entries. Inputs are the already-normalized params — `normalizeFacetParams` (`:675`) is applied upstream at `:561` BEFORE the probe is built, so `discoverCountCached` receives sort=''/page=1/pinned-Language params; do **not** re-normalize.
  - [x] Add `func (s *CacheService) discoverCountCached(ctx context.Context, kind string, p DiscoverParams) (int, error)`: `s.cache.Get(ctx, facetCountCacheKey(kind, p))` → on hit `strconv.Atoi(cached.Value)` and return the **int**; on miss call the **same client method the grid path uses** — `s.client.DiscoverMoviesWithFallback(ctx, p)` for `kind=="movie"` (see `:438`) / `s.client.DiscoverTVShowsWithFallback(ctx, p)` for `kind=="tv"` (see `:469`), single call per probe (pinned Language short-circuits the fallback chain per `normalizeFacetParams` comment / BE AC10) — read `TotalResults`; **if `TotalResults == 0` → return 0 WITHOUT caching** (AC3); else `s.cache.Set(ctx, key, strconv.Itoa(count), CacheTypeTMDbFacet, FacetCountCacheTTL)` (store the **int string**, AC5) and return.
  - [x] ⚠️ **Intentional grid-cache bypass:** going through `s.client.*WithFallback` (not `s.DiscoverMovies`/`DiscoverTVShows`) is required — the grid methods always `Set` at `type="tmdb"`/1h including zeros, which is exactly what AC2/AC3 must diverge from. Consequence: a facet probe no longer warms the grid blob cache. This is **negligible** — facet probes normalize `sort=''`/`page=1`, while the grid uses the user's real sort+page, so the two key-spaces almost never overlapped anyway. Note this in a code comment.
- [x] **Task 3 — Refactor `DiscoverFacetCounts` to use the dedicated path (AC: #6)**
  - [x] File: `apps/api/internal/tmdb/cache.go`, inside the `g.Go` closure (`:583-607`). Replace `movies, err := s.DiscoverMovies(gctx, probe.param)` (`:588`) and `tv, err := s.DiscoverTVShows(gctx, probe.param)` (`:594`) with `mc, err := s.discoverCountCached(gctx, "movie", probe.param)` + `tc, err := s.discoverCountCached(gctx, "tv", probe.param)`, and the sum at `:604` becomes `counts[probe.dim][probe.value] = mc + tc`. Keep the per-sub-query fail-soft (`return nil` on err, `:592`/`:598`), the `gctx.Err()` budget check (`:585`), errgroup `SetLimit`, partial computation (`:613-618`) **byte-identical** — this is a cache-path swap, not a behavior change. Zero-skip is **per sub-query**: if movie=5 and tv=0, movie's 5 is cached and tv's 0 is not; the summed `5` is still returned and KEPT in the response (AC6 parity preserved).
- [x] **Task 4 — Tests (AC: all) — extend `apps/api/internal/tmdb/cache_test.go`**
  - [x] ⚠️ **MUST-FIX existing test — `TestCacheService_DiscoverFacetCounts_CacheReuse` (`:1186`).** Line **`:1202`** currently asserts `repo.lastSetTTL == TrendingDiscoverCacheTTL` with comment `"counts ride the 1h discover cache path (Q2=A)"` — this **flips** after the refactor: update it to `FacetCountCacheTTL` and rewrite the comment (counts now ride the dedicated facet cache, not the trending path). The reuse mechanics (`fb.DiscoverMoviesCalled == 1` warm, `:1200`/`:1208`) still hold because the mock returns non-zero (340/60), so the int IS cached and reused — keep those, they now prove AC4 through the new path.
  - [x] **Distinct-type assertion (AC2):** `MockCacheRepository` (test file) currently captures `lastSetTTL` but likely **not** `lastSetType` — add a `lastSetType` capture in its `Set` if absent, then assert it equals `"tmdb_facet"` (`CacheTypeTMDbFacet`) on a count `Set`.
  - [x] **NEW zero-not-cached test (AC3)** — model on `TestCacheService_DiscoverFacetCounts_ZeroKept` (`:1352`, both sub-queries return `TotalResults: 0`): assert (a) the response still KEEPS the `0` (parity, unchanged from `_ZeroKept`), AND (b) the cache `Set` was NOT called for that key, AND (c) a SECOND identical `DiscoverFacetCounts` RE-CALLS upstream (`fb.DiscoverMoviesCalled == 2`) — proving the transient `0` is not pinned for the TTL. (`_ZeroKept` itself stays green — it asserts only the response, not caching.)
  - [x] **Int-stored (AC5):** assert the cached value for a non-zero probe parses back to the int (`strconv.Atoi(repo.lastSetValue)` → the count), not a JSON `SearchResult*` blob.
  - [x] **Behavior parity (AC6):** the rest of the BE facet-count suite (`_MovieTVSumPerValue` `:1031`, `_AddSemantics` `:1059`, `_Normalization` `:1147`, `_ConcurrencyBound` `:1215`, `_PartialOnTimeout` `:1265`, `_FailSoft` `:1301`, `_CandidateOnly` `:1329`, `_UnparseableSkipped` `:1372`, `_EmptyCandidates` `:1392`, `_DuplicateCandidateDeduped` `:1408`) MUST stay green against the refactored path — run the whole `FacetCount` filter, not just new cases.
  - [x] Optional (NOT duplicated — covered by existing coverage): `ClearByType`'s type-isolated eviction (evicts one `type`, leaves others intact) is already proven by `TestCacheRepository_ClearByType` (`apps/api/internal/repository/cache_repository_test.go:398`). That logic is **type-agnostic** — the `type` string is just a parameter — so tagging count entries `"tmdb_facet"` (asserted here via `repo.lastSetType` in `_CacheReuse`) transitively enables targeted `ClearByType(ctx, "tmdb_facet")` without a near-duplicate test. Decision recorded in Completion Notes.
  - [x] Run: `cd apps/api && go test ./internal/tmdb/ -run FacetCount -v`; then `pnpm lint:all` (Rule 12).

## Dev Notes

### Why a divergent path is mandatory (the core rationale)

All three fixes are **impossible** on the BE story's reuse path: `DiscoverMovies/TVShows` hardcode `type`, TTL, and cache-everything, and they are **shared with the result grid** — changing them to skip-zeros or retag would corrupt the grid's caching (the grid legitimately caches 0-result discovers at `type="tmdb"`/1h). So count fidelity REQUIRES a dedicated count cache method (`discoverCountCached`), which is exactly the path the BE story deliberately deferred to keep its prerequisite-gated scope minimal.

### Anchors (RE-VERIFIED against the AS-MERGED BE story #100 on 2026-06-30)

All line numbers below were confirmed against the current `apps/api/internal/tmdb/cache.go` after #100/#101 merged (the original draft's numbers were pre-merge estimates and have been corrected):

- `apps/api/internal/tmdb/cache.go`:
  - First const block **`:17-29`** (`CacheTypeTMDb="tmdb"` `:19`, `DefaultCacheTTL=24h` `:22`, `TrendingDiscoverCacheTTL=1h` `:28`) — add `CacheTypeTMDbFacet` here.
  - Second (facet) const block **`:31-70`** (`DimGenre` `:37`, `facetCountBudget=800ms` `:47`, `facetCountConcurrency=4` `:52`, `maxFacetProbes=64` `:60`, `defaultFacetCountLanguage="zh-TW"` `:70`) — recommended home for `FacetCountCacheTTL`.
  - `DiscoverMovies` **`:425`** / `DiscoverTVShows` **`:456`** — the grid methods we BYPASS; they call `s.client.DiscoverMoviesWithFallback` (`:438`) / `DiscoverTVShowsWithFallback` (`:469`) — the client methods `discoverCountCached` calls directly.
  - `discoverCacheKey` **`:491-502`** (`tmdb:discover/%s:…`) — mirror for `facetCountCacheKey` with a `tmdb:facetcount/%s:…` namespace.
  - `DiscoverFacetCounts` **`:533`**; fan-out closure **`:583-607`** (movie call `:588`, tv call `:594`, sum `:604`); partial calc `:613-618` — refactor target.
  - `applyFacet` `:628`, `normalizeFacetParams` `:675` (called at `:561`, pre-normalizes probe params) — DO NOT touch; they guarantee `discoverCountCached` receives normalized params.
  - `cache.Set(ctx, key, value string, cacheType string, ttl time.Duration)` signature confirmed at `:448`; `cache.Get(ctx, key)` returns `(cached, err)` with `cached.Value string` confirmed at `:428-431`.
- `apps/api/internal/tmdb/types.go`: `FacetCounts` struct **`:275`** (response shape — unchanged by this story).
- `apps/api/internal/repository/cache_repository.go`: `ClearByType(ctx, cacheType) (int64, error)` exists at **`:144`** (and `ClearExpired` at `:123`) — AC2 enables targeted eviction for `"tmdb_facet"`; **no repo change needed**.
- `idx_cache_entries_type` index (migration `004`) already backs `ClearByType` efficiently.

### Interaction with the other cache stories

- `infra-cache-entries-expiry-sweep` (the scheduled `ClearExpired`) already covers ALL `cache_entries` rows regardless of `type`, so count entries are swept whether tagged `"tmdb"` or `"tmdb_facet"` — this refinement does NOT change the growth-bound story; it adds **targeted** eviction (`ClearByType`) on top of the time-based sweep.
- The distinct `"tmdb_facet"` type also improves **observability** (count vs grid cache rows are now distinguishable in any cache-stats query).

### Rule compliance

- Rule 1 (backend `/apps/api`), Rule 4/11 (CacheService method, interface in package), Rule 7 (no new error code), Rule 9 (co-located tests), Rule 12 (lint:all), Rule 14 (reuse the existing client/limiter — `discoverCountCached` calls the SAME client path, no new client), Rule 27 ② (cache still checked before the limiter).
- **Cross-stack split check:** 4 tasks, all backend; 0 frontend → single story, **no split**.
- The FE story is **transparent to this change** — the `[@contract-v1]` response shape is unchanged; only the BE's internal caching differs. No FE re-ack needed.

### Project Structure Notes

- All edits in `apps/api/internal/tmdb/cache.go` + `cache_test.go`. No migration (reuses the existing `cache_entries` schema + `type` column + `idx_cache_entries_type`), no new error code, no handler/route change, no contract change.

### Time-dependent visual coverage

- **N/A — no `apps/web/src/components` touched.** Backend cache-internals only.

### References

- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md`] — Architecture Review AR-F4 / AR-F5 / AR-F6 / AR-F8; Technical Decisions #11 (don't cache zero) / #12 (dedicated TTL + distinct type; note true isolation = separate table, Hard, deferred).
- [Source: `_bmad-output/implementation-artifacts/ux3-discover-facet-aggregation-be.md`] — the `DiscoverFacetCounts` reuse path this story refactors; the BE Discovery-Triage entry that filed this follow-up; AC3/AC9 (normalization) + AC10 (fallback-bypass) this story inherits.
- [Source: `apps/api/internal/tmdb/cache.go`, `apps/api/internal/repository/cache_repository.go`] — anchors above.
- [Source: `project-context.md` Rule 1/4/7/9/12/14/27] — governing rules.

## Dev Agent Record

### Agent Model Used

Amelia (BMM Dev Agent) — claude-opus-4-8[1m]

### Debug Log References

- `cd apps/api && go test ./internal/tmdb/ -run FacetCount -v` → all FacetCount cases PASS (incl. new `_ZeroNotCached` + updated `_CacheReuse`).
- `pnpm nx test api` → all Go packages PASS. `pnpm nx test web` → 2251 tests / 202 files PASS (no regression from a backend-only change).
- `pnpm nx lint api` (go vet + staticcheck@2026.1) → clean. `pnpm lint:all` → 0 errors / 123 warnings (all pre-existing `apps/web/**`, none in touched files). `gofmt -l` on touched Go files → empty. `prettier --check .` → all files conform.

### Completion Notes List

Implemented the AR-F4/F5/F8 facet-count cache fidelity refinements as a DIVERGENT count-cache path, leaving the result-grid path (`DiscoverMovies`/`DiscoverTVShows`) byte-unchanged.

- **AC1 (AR-F8 — dedicated TTL):** added `FacetCountCacheTTL = 1 * time.Hour` in the facet const block (`cache.go`), independently tunable from `TrendingDiscoverCacheTTL` (comment cross-references it). `_CacheReuse` now asserts `repo.lastSetTTL == FacetCountCacheTTL`.
- **AC2 (AR-F4 — distinct type):** added `CacheTypeTMDbFacet = "tmdb_facet"` in the first const block; `discoverCountCached` tags every count `Set` with it. `_CacheReuse` asserts `repo.lastSetType == CacheTypeTMDbFacet`. Targeted `ClearByType(ctx, "tmdb_facet")` eviction is thereby enabled (mechanism proven by existing `TestCacheRepository_ClearByType`).
- **AC3 (AR-F5 — zero short-TTL'd, per CR M1):** `discoverCountCached` caches a `TotalResults == 0` on the SHORT `FacetCountZeroTTL` (1 min) rather than the full `FacetCountCacheTTL` — this is the AC3 "**or written with a much shorter TTL**" branch, chosen after CR M1 flagged that never-caching genuine-0 facets re-hits TMDb (2 calls) on every debounced probe. `TestCacheService_DiscoverFacetCounts_ZeroShortTTL` proves (a) the 0 is kept in the response, (b) the 0 is cached with `FacetCountZeroTTL` + tagged `tmdb_facet` + stored as `"0"`, (c) a second identical probe within the TTL is served from cache (no re-fetch). Expiry after the short TTL is repository-enforced (covered by the `CacheRepository` TTL tests).
- **AC4 (count-to-count reuse preserved):** a warm NON-ZERO probe serves from the dedicated count cache with no new TMDb call (`_CacheReuse`, `DiscoverMoviesCalled` stays 1).
- **AC5 (int stored — AR-F6):** stores `strconv.Itoa(count)`, not a `SearchResult*` JSON blob. `_CacheReuse` asserts the stored value parses via `strconv.Atoi`.
- **AC6 (behavior parity):** `DiscoverFacetCounts` fan-out swapped to `discoverCountCached` per sub-query; budget/errgroup/partial/fail-soft logic byte-identical. Zero-skip is per sub-query (movie=5, tv=0 → 5 kept, only tv's 0 uncached). All 13 pre-existing FacetCount cases stay green (incl. `_ZeroKept`, `_PartialOnTimeout`, `_FailSoft`, `_ConcurrencyBound`, `_Normalization`).
- **AC7 (deep isolation deferred):** recorded the AR-F4 boundary in the `CacheTypeTMDbFacet` doc comment — targeted `ClearByType` only; true purge-isolation (wholesale `clearTable`) still needs a separate table (Hard, deferred; see Discovery Triage ③).
- **Intentional grid-cache bypass** documented in `discoverCountCached` — calling `s.client.*WithFallback` (not the grid methods) is required so counts diverge on type/TTL/zero; a facet probe no longer warms the grid blob (negligible: the sort=""/page=1 count key-space almost never overlapped the grid's real sort+page).
- **ClearByType (optional test) — NOT duplicated:** type-isolated eviction is already proven type-agnostically by `TestCacheRepository_ClearByType` (`cache_repository_test.go:398`); tagging counts `"tmdb_facet"` (asserted via `lastSetType`) transitively enables `ClearByType(ctx, "tmdb_facet")`. No near-duplicate test added.
- **🔗 AC Drift: NONE** (checked `'tmdb_facet|FacetCountCacheTTL|CacheTypeTMDbFacet'` across `_bmad-output/implementation-artifacts/*.md` — hits are the tech-spec + BE-story #100 DEFERRAL notes that explicitly PLANNED this refinement; the `[@contract-v1]` `FacetCounts` wire shape is unchanged and BE AC6 parity is preserved. The only in-family assertion that flips — `_CacheReuse:1202` TTL — was pre-declared as a MUST-FIX in Task 4 and updated).
- **📎 Contract Stamps: NONE** (this story defines no `[@contract-v*]` stamps and does NOT bump the upstream `[@contract-v1]`; the response shape is unchanged — internal caching only, so no downstream re-ack / Change-Log bump row is required).
- **🎭 A11y Pre-Flight: N/A** (100% backend — no `apps/web/` files touched).
- **🎨 UX Verification: SKIPPED** — no UI changes in this story (backend cache internals only; no `apps/web/src/components` touched, no design screenshot in scope).
- **🔌 Route Sync: N/A** (no backend route touched — internal `CacheService` refactor; the existing `GET /api/v1/tmdb/discover/facet-counts` route from #100 is unchanged. `discoverCountCached` calls the external TMDb `*WithFallback` client, out of the in-scope route-sync check).
- **🔒 Security (Epic 9 Retro AI-6):** Bounded I/O — stores a compact int string, reads a single `TotalResults` int (no unbounded reads introduced). Context lifecycle — `discoverCountCached` accepts and honors `ctx` (passed to `cache.Get/Set` + client, respects budget cancellation). Error wrapping — returns the raw client error (matching the grid path's style); it is swallowed+logged at the fan-out boundary, never leaked to the API response.

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES — one item, pre-noted:**
  - **③ — true purge-isolation needs a separate cache table (Hard, deferred).** The distinct `type="tmdb_facet"` gives targeted `ClearByType` eviction, but the **manual full purge** is still a wholesale `clearTable("cache_entries")` (tech-spec AR-F4 note). Genuine isolation (purge counts without touching the grid via the manual path, or independent size budgets) needs a **separate table** — Hard, out of scope, remains deferred. If/when this is needed, file a new `infra-facet-count-separate-table` backlog at that point (not now — YAGNI, no current consumer).
- Reference: `project-context.md` Rule 24; origin: tech-spec AR-F4 deep-fix note + this story's scope boundary.

### File List

- `apps/api/internal/tmdb/cache.go` (MODIFIED — `CacheTypeTMDbFacet` + `FacetCountCacheTTL` + `FacetCountZeroTTL` (CR M1) consts; new `facetCountCacheKey` + `discoverCountCached`; `DiscoverFacetCounts` fan-out swapped to the dedicated count path; method/bypass doc comments refreshed)
- `apps/api/internal/tmdb/cache_test.go` (MODIFIED — `MockCacheRepository` captures `lastSetType`/`lastSetValue`; `_CacheReuse` flipped to `FacetCountCacheTTL` + AC2/AC5(`==60`) assertions; new `_ZeroShortTTL` (CR M1) + `TestFacetCountCacheKey_DistinctNamespace` (CR M2); `strings` import added)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED — `ux3-facet-count-cache-refinement` status → in-progress → review → done)

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-07-01 | Implemented all 4 tasks (Amelia dev-story). Added dedicated facet-count cache path: `CacheTypeTMDbFacet="tmdb_facet"` (AC2) + independently-tunable `FacetCountCacheTTL` (AC1) consts; `facetCountCacheKey` (distinct `tmdb:facetcount/…` namespace) + `discoverCountCached` (stores compact int per AC5, short-TTL's `total_results==0` per AC3, calls the same `*WithFallback` client as the grid to preserve count-to-count reuse per AC4, intentional grid-cache bypass documented); refactored `DiscoverFacetCounts` fan-out to use it with byte-identical budget/errgroup/partial/fail-soft (AC6 parity). Tests: `MockCacheRepository` now captures type+value; `_CacheReuse` TTL assertion flipped `TrendingDiscoverCacheTTL→FacetCountCacheTTL` + AC2/AC5 assertions added; new zero + key-namespace tests. AR-F4 deep purge-isolation (separate table) stays deferred per AC7. Regression: `nx test api` green, `nx test web` 2251 green, go vet + staticcheck + eslint(0 err) + prettier clean. AC Drift NONE; Contract Stamps NONE; A11y N/A (backend); Route Sync N/A. |
| 2026-07-01 | Addressed code-review findings — 4 items resolved (2 Med, 2 Low), 0 High. **M1:** switched `total_results==0` from never-cache to SHORT-TTL cache via new `FacetCountZeroTTL=1min` const (AC3 "or short TTL" branch) so genuine-0 facets stop re-hitting TMDb on every debounced probe while transient-0 still self-corrects fast; reworked `_ZeroNotCached`→`_ZeroShortTTL`. **M2:** added `TestFacetCountCacheKey_DistinctNamespace` guarding the AR-F4 no-collision guarantee at the KEY level (not just `type`). **L1:** tightened `_CacheReuse` AC5 assertion `Greater(n,0)`→`Equal(60,n)`. **L2:** corrected the grid-cache-bypass comment (key-spaces overlap only at default-sort landing, not "almost never"). Regression re-run: `nx test api` green, `nx lint api` (vet+staticcheck) green, gofmt clean; web unchanged (no `apps/web/` files touched). |

## Senior Developer Review (AI)

**Reviewer:** Amelia (BMM adversarial code-review workflow) · **Date:** 2026-07-01 · **Outcome:** ✅ Approve (all findings resolved inline)

**Mandatory checks:** 🔒 Rule 7 Wire Format: PASS (0 error-code constants in scoped Go files) · 🔒 Rule 20 Contract Bump: N/A (no `[@contract-vN→v(N+1)]` bump) · 🔒 Rule 25 Mega-line: N/A (no `project-context.md` "Last Updated:" change) · Git vs File List: 0 discrepancies · AC1–AC7 all IMPLEMENTED · all tasks `[x]` genuinely done (no false claims).

**Findings & resolutions (0 High / 2 Med / 2 Low — all fixed, user chose auto-fix):**

- [x] **[Med] M1 — genuine-0 facets never cached → repeated 2×TMDb calls per debounced probe** (`cache.go` `discoverCountCached`). Never-caching a `0` (AC-compliant) meant a legitimately-empty facet re-fetched movie+tv on every facet-counts request, pressuring the 40-req/10s rate budget + 800ms fan-out budget (could force `Partial`). **Fix:** AC3's "or short TTL" branch — new `FacetCountZeroTTL=1min` caches `0` briefly (self-corrects fast, but coalesces rapid re-probes). Test `_ZeroShortTTL`.
- [x] **[Med] M2 — no regression test guarded the distinct cache-KEY namespace** (`cache_test.go`). Only the `type` was asserted; the AR-F4 no-collision guarantee rests on the `tmdb:facetcount/` vs `tmdb:discover/` key namespace, which a future refactor could silently break (blob/int cross-clobber). **Fix:** `TestFacetCountCacheKey_DistinctNamespace`.
- [x] **[Low] L1 — `_CacheReuse` AC5 assertion too weak** (`assert.Greater(n,0)` didn't lock the stored magnitude). **Fix:** `assert.Equal(t, 60, n)`.
- [x] **[Low] L2 — grid-cache-bypass comment overstated non-overlap** ("almost never overlapped"): the default-sort landing (grid `SortBy=""`/page=1) DOES overlap the normalized count key-space. **Fix:** comment corrected to state the overlap is limited to that one case with a one-time-warm cost.

No CRITICAL/HIGH issues: no task marked `[x]` was unimplemented, every AC has code + test evidence, no security/injection/unbounded-I/O concerns, tests are real assertions (not placeholders).
