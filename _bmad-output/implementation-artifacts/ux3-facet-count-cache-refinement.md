# Story (UX3): Facet-Count Cache Fidelity Refinement (AR-F4 / F5 / F8)

Status: backlog

<!-- Intentionally kept `backlog` (NOT promoted to ready-for-dev): hard-blocked by
     ux3-discover-facet-aggregation-be (must merge first — this refactors its code) AND
     conditional ("do when stale-0 dimming proves annoying"). The story is content-complete
     and pickup-ready the moment the BE lands and the refinement is judged warranted. -->


<!-- Story key: ux3-facet-count-cache-refinement.
     Rule-24 ③ follow-up deferred from ux3-discover-facet-aggregation-be (Party Mode Q2=A, 2026-06-24).
     CONDITIONAL + HARD-BLOCKED — see "Blocking Prerequisite". Backend-only. -->

## ⚠️ Blocking Prerequisite & Conditionality (read FIRST)

- **HARD-BLOCKED by `ux3-discover-facet-aggregation-be` (must be MERGED first).** This story **refactors** the `CacheService.DiscoverFacetCounts` method that the BE story creates. There is nothing to refine until that ships. Sequence strictly after it.
- **CONDITIONAL (do-when-needed).** The BE story (Q2=A) intentionally ships the simpler **reuse-`DiscoverMovies/TVShows`-as-is** path. The harm it accepts is bounded (a transient/wrong-locale `0` dims a facet for ≤1h, and 0-facets stay **selectable**). This refinement is worth doing **when stale-`0` dimming proves annoying in practice**, or when count-cache freshness/eviction needs to be tuned independently of the grid. It is **not** required for the facet-counts feature to function.

## Story

As the **Vido backend cache layer**,
I want **facet-count sub-queries cached on a dedicated path — distinct cache `type`, dedicated TTL, and zero-results not cached (or short-TTL'd)**,
so that **count freshness/eviction can be tuned independently of the result grid, and a transient or wrong-locale `0` never wrongly dims a facet chip for a full hour.**

## Background — what the BE story deferred and why

The BE story `ux3-discover-facet-aggregation-be` computes each facet count by reusing `CacheService.DiscoverMovies` / `DiscoverTVShows` **as-is** (Party Mode Q2=A). Those methods hardcode:

- cache `type = CacheTypeTMDb = "tmdb"` (shared with the result grid), and
- `TTL = TrendingDiscoverCacheTTL = 1h` (`cache.go:14-26`), and
- they **cache every response, including `total_results == 0`**.

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

- [ ] **Task 1 — Constants (AC: #1, #2)**
  - [ ] File: `apps/api/internal/tmdb/cache.go` (`:14-26` const block). Add `FacetCountCacheTTL = 1 * time.Hour` (with a comment that it is independently tunable from `TrendingDiscoverCacheTTL`) and `CacheTypeTMDbFacet = "tmdb_facet"`.
- [ ] **Task 2 — Dedicated cached count path (AC: #3, #4, #5)**
  - [ ] File: `apps/api/internal/tmdb/cache.go`. Add a `facetCountCacheKey(kind string, p DiscoverParams) string` mirroring `discoverCacheKey` (`:437-453`) but with a **distinct namespace** (e.g. `tmdb:facetcount/{kind}:...`) so count entries never collide with grid (`tmdb:discover/...`) entries. Inputs are the already-normalized params (sort=''/page=1/pinned Language — done upstream in `DiscoverFacetCounts`).
  - [ ] Add `func (s *CacheService) discoverCountCached(ctx, kind string, p DiscoverParams) (int, error)`: `Get(facetCountCacheKey)` → on hit unmarshal the **int** and return; on miss call the same client path the BE story uses (`DiscoverMoviesWithFallback` / `DiscoverTVShowsWithFallback`, single call per probe per the BE AC10 resolution), read `TotalResults`; **if `TotalResults == 0` → return it WITHOUT caching** (AC3); else `cache.Set(key, strconv(count), CacheTypeTMDbFacet, FacetCountCacheTTL)` (store the int, AC5) and return.
- [ ] **Task 3 — Refactor `DiscoverFacetCounts` to use the dedicated path (AC: #6)**
  - [ ] File: `apps/api/internal/tmdb/cache.go`. Replace the `s.DiscoverMovies(...)` + `s.DiscoverTVShows(...)` calls inside the fan-out with `s.discoverCountCached(gctx, "movie", p)` + `s.discoverCountCached(gctx, "tv", p)`; sum. Keep everything else (errgroup `SetLimit`, time-budget, partial, fail-soft, contextual add semantics) **identical** — this is a cache-path swap, not a behavior change.
- [ ] **Task 4 — Tests (AC: all)**
  - [ ] File: `apps/api/internal/tmdb/cache_test.go` (extend the BE story's facet-count tests). New/updated cases: **dedicated TTL** (assert `lastSetTTL == FacetCountCacheTTL`, independent of trending); **distinct type** (assert the `Set` `cacheType == "tmdb_facet"`); **zero-skip** (a sub-query returning `TotalResults==0` → `setCalled` NOT incremented for that key, and a re-probe re-calls the client); **non-zero count-to-count reuse** (second identical probe → no new client call); **int stored** (cached value parses back to the int); **behavior parity** (re-run the BE-story fan-out assertions — counts correctness, partial, fail-soft — green against the refactored path).
  - [ ] Optional: a `ClearByType(ctx, "tmdb_facet")` test asserting it evicts count entries but leaves `type="tmdb"` grid entries.
  - [ ] `cd apps/api && go test ./internal/tmdb/ -run FacetCount -v`; `pnpm lint:all` (Rule 12).

## Dev Notes

### Why a divergent path is mandatory (the core rationale)

All three fixes are **impossible** on the BE story's reuse path: `DiscoverMovies/TVShows` hardcode `type`, TTL, and cache-everything, and they are **shared with the result grid** — changing them to skip-zeros or retag would corrupt the grid's caching (the grid legitimately caches 0-result discovers at `type="tmdb"`/1h). So count fidelity REQUIRES a dedicated count cache method (`discoverCountCached`), which is exactly the path the BE story deliberately deferred to keep its prerequisite-gated scope minimal.

### Anchors (current — verify against the AS-MERGED BE story, line numbers will have shifted)

- `apps/api/internal/tmdb/cache.go`: const block `:14-26` (`CacheTypeTMDb="tmdb"`, `DefaultCacheTTL=24h`, `TrendingDiscoverCacheTTL=1h`) — add the two new consts here; `discoverCacheKey` `:437-453` (mirror for `facetCountCacheKey`); `cache.Set(ctx, key, value, cacheType, ttl)` signature; **`DiscoverFacetCounts`** (added by the BE story — refactor its fan-out body). The BE story's `cache.go` edits land first; re-locate these before editing.
- `apps/api/internal/repository/cache_repository.go`: `ClearByType(ctx, cacheType) (int64, error)` already exists (`CacheRepositoryInterface`) — AC2 enables it for `"tmdb_facet"`; no repo change needed.
- `apps/api/internal/repository/cache_repository.go`: `idx_cache_entries_type` index (migration `004`) already backs `ClearByType` efficiently.

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

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES — one item, pre-noted:**
  - **③ — true purge-isolation needs a separate cache table (Hard, deferred).** The distinct `type="tmdb_facet"` gives targeted `ClearByType` eviction, but the **manual full purge** is still a wholesale `clearTable("cache_entries")` (tech-spec AR-F4 note). Genuine isolation (purge counts without touching the grid via the manual path, or independent size budgets) needs a **separate table** — Hard, out of scope, remains deferred. If/when this is needed, file a new `infra-facet-count-separate-table` backlog at that point (not now — YAGNI, no current consumer).
- Reference: `project-context.md` Rule 24; origin: tech-spec AR-F4 deep-fix note + this story's scope boundary.

### File List

_(to be filled by dev agent)_

- `apps/api/internal/tmdb/cache.go` (MODIFIED — consts + `facetCountCacheKey` + `discoverCountCached` + refactor `DiscoverFacetCounts`)
- `apps/api/internal/tmdb/cache_test.go` (MODIFIED — fidelity tests)
