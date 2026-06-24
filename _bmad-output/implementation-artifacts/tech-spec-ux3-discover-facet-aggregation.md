---
title: 'Discover Contextual Facet Counts'
slug: 'ux3-discover-facet-aggregation'
created: '2026-06-24'
status: 'ready-for-dev'
stepsCompleted: [1, 2, 3, 4]
tech_stack:
  ['Go', 'Gin', 'golang.org/x/sync (errgroup w/ SetLimit)', 'golang.org/x/time/rate', 'React', 'TanStack Query', 'TanStack Router']
files_to_modify:
  [
    'apps/api/internal/tmdb/cache.go (+DiscoverFacetCounts + interface)',
    'apps/api/internal/tmdb/types.go (+FacetCounts response type)',
    'apps/api/internal/services/tmdb_service.go (+passthrough)',
    'apps/api/internal/handlers/tmdb_handler.go (+handler, +route L616, +interface)',
    'apps/api/internal/tmdb/cache_test.go',
    'apps/api/internal/handlers/tmdb_handler_test.go',
    'apps/web/src/services/tmdb.ts (+discoverFacetCounts)',
    'apps/web/src/hooks/useDiscoverFacetCounts.ts (NEW)',
    'apps/web/src/lib/discoverFilters.ts (+facet enumeration helper)',
    'apps/web/src/components/search/FilterPanel.tsx (per-chip count + dimmed-0)',
    'apps/web/src/components/search/DiscoverFilterRail.tsx (wire counts hook)',
    'apps/api/internal/services/cache_sweep_scheduler.go (NEW — AR-F1 prerequisite infra story) + cmd/api/main.go wiring',
    'ux-design.pen (design prerequisite — ux-designer)',
  ]
code_patterns:
  [
    'cached discover path (CacheService.DiscoverMovies/TVShows → discoverCacheKey, 1h TTL); cache checked BEFORE limiter (Rule 27 ②)',
    'shared rate limiter, UNEXPORTED, one per Client (40req/10s, burst 40); limiter.Wait first line of doRequest',
    'handler: parseDiscoverParams → service → SuccessResponse; TMDb errors via *TMDbError + handleTMDbError',
    'TMDB_ error code constants (Rule 7); reuse existing, no new code for facet-counts',
    'shell-gated v2 FE (useShellVersion); chips keyed by genre.id/region.code/rating value/platform.id',
  ]
test_patterns:
  [
    'Go: table-driven tests + MockTMDbService (call-capture slices) + testify assert/require; setupTMDbRouter helper',
    'FE: Vitest + Testing Library, mock the hook, assert chip render/dimmed/fallback',
  ]
---

# Tech-Spec: Discover Contextual Facet Counts

**Created:** 2026-06-24

> Spike basis: `_bmad-output/planning-artifacts/ux3-discover-facet-aggregation-spike.md`
> Backlog origin: `ux3-discover-facet-aggregation-be` (Rule-24 ③ deferral from ux3-3-2).

## Overview

### Problem Statement

The Discover v2 rail (`DiscoverFilterRail`, shipped in ux3-3-2) shows only a single live total
`符合 N 部`. When a user filters across 類型 / 地區 / 評分 / 平台, they cannot see how many results each
remaining facet would yield, and can pick a dead-end (0-result) filter with no warning. TMDb `/discover`
returns only `total_results` (no facet aggregation), so a per-facet count must be computed by the backend —
one `/discover` call per facet value.

**Search space note (sizes BOTH the value and the cost):** Discover navigates the **TMDb-scale catalog**
(hundreds of thousands of titles), NOT the user's private 媒體庫. So guided exact contextual counts have
**high value** here (drilling through a vast space is exactly where dead-end prevention + "how many remain"
help most) — AND the large space means **more distinct facet combinations → lower cache hit rate, more cold
fan-outs, more `partial` responses**, which raises the importance of AR-F1 (unbounded growth), AR-F2 (key
normalization) and AR-F8 (dedicated TTL). This is why the decision is full exact counts, not grey-out-only.

### Solution

Add a backend **facet-counts** endpoint that, for the current filter selection, computes the **contextual**
result count for every facet value (count if that facet were *added* to the current selection) by fanning out
**cached** `/discover` calls through the existing rate limiter + 1h cache + errgroup, returning ONE response.
The frontend revives the `.pen` `FacetCountChip` to show each chip's contextual count in JetBrains Mono,
**greys out 0-result facets** (dead-end prevention), fills counts in **progressively** (cached instant,
uncached shimmer) on a **debounced** recompute when the committed filter settles, and falls back to today's
single total when the endpoint is unavailable/partial.

### Scope

**In Scope:**

- **BE:** `GET /api/v1/tmdb/discover/facet-counts` + `CacheService.DiscoverFacetCounts(ctx, params, dims)` —
  errgroup fan-out where each facet count = `(base params + that facet).TotalResults`, summed over movie+tv,
  reusing the existing cached `DiscoverMovies/TVShows`. `TMDB_*` error code prefix (Rule 7).
- **FE:** revive per-chip Mono counts in `DiscoverFilterRail`'s `FilterPanel`; **dim (but keep selectable)**
  0-count facets; progressive subtle fill (no per-chip spinner); debounced recompute on committed-filter
  change; fall back to the single total.
- **Tests:** BE service + handler (counts correctness, fan-out, cache reuse, error fail-soft);
  FE chip rendering, 0-grey-out, shimmer, fallback.

**Out of Scope:**

- **Baseline** (filter-independent) counts — contextual only.
- **Pre-warm worker / scheduled job** (the "C" strategy) — useless for contextual (combos are user-specific);
  rely on the existing 1h cache warming naturally; revisit only if cold latency proves painful.
- Per-value counts for the **year-range numeric** facet (a range has no single-value count).
- **SSE** transport — use request/poll or synchronous-with-budget; mechanism decided in Step 2.
- The single-total footer stays unchanged (it is the fallback / summary).
- **Mobile `FilterBottomSheet`** — facet-counts are DESKTOP-rail only; mobile keeps the single draft total
  (a batch sheet that already live-counts the draft). Avoids per-facet fan-out on small screens / slow nets
  (pre-mortem F5).

## Context for Development

### Codebase Patterns

- **A facet count is already a cached quantity.** `CacheService.DiscoverMovies/TVShows`
  (`internal/tmdb/cache.go`) cache the full `SearchResult*` (incl. `TotalResults`) keyed by
  `discoverCacheKey()` over every filter dimension at `TrendingDiscoverCacheTTL` (1h). The new service reuses
  this path — no new cache. Adjacent recomputes share most sub-queries → high warm-cache hit rate.
- **Fan-out is rate-safe by construction.** `internal/tmdb/client.go` fronts every call with
  `golang.org/x/time/rate` (40 req / 10s, burst 40), and the cache is checked **before** the limiter
  (Rule 27 Pillar 2). A server-side fan-out cannot exceed the limit — it only serializes.
- `golang.org/x/sync` (errgroup) is available for bounded concurrency.
- FE: shell-gated v2 (`useShellVersion`); the rail hosts the shared `FilterPanel`; `.pen` `FacetCountChip`
  (Component Library `filter-controls-v2`) is the design for the per-chip count.

### Files to Reference

| File | Purpose |
| ---- | ------- |
| `apps/api/internal/tmdb/cache.go` | `DiscoverMovies/TVShows` cached path, `discoverCacheKey`, 1h TTL — reuse for each facet sub-query; add `DiscoverFacetCounts` here |
| `apps/api/internal/tmdb/client.go` | global rate limiter (40/10s, burst 40) — fan-out flows through it |
| `apps/api/internal/handlers/tmdb_handler.go` | `DiscoverMovies/TVShows` handlers + `parseDiscoverParams` — mirror for the new endpoint |
| `apps/api/internal/tmdb/types.go` | `DiscoverParams`, `SearchResultMovies/TVShows` (`TotalResults`) |
| `apps/api/internal/tmdb/errors.go` | `TMDB_*` error codes (Rule 7) |
| `apps/web/src/lib/discoverFilters.ts` | facet inventory (18 genres/5 regions/3 platforms/4 ratings), `buildDiscoverParams` |
| `apps/web/src/components/search/FilterPanel.tsx` | the chips that gain per-facet counts |
| `apps/web/src/components/search/DiscoverFilterRail.tsx` | hosts the rail; wires the counts hook |
| `apps/web/src/hooks/useDiscoverResults.ts` | pattern for the FE count-fetching hook |
| `ux-design.pen` `FacetCountChip` | design for per-chip Mono count (filter-controls-v2) |

### Technical Decisions

1. **Semantic: Contextual** (count if facet added to current selection), not baseline — baseline counts
   mislead once filters stack; Discover is a drill-down surface where dead-end prevention is the value.
2. **Latency: progressive fill-in + debounce, instant-feel preserving.** All rail sections are visible at
   once, so cached counts show instantly and uncached ones fill in subtly (NO per-chip spinners). On a toggle,
   recompute only the counts in dimensions OTHER than the just-toggled one — keep already-known counts stable
   so the rail never reads as "everything is loading" (protects the ux3-3-2 instant-rail identity, pre-mortem
   F1). Recompute only after the committed filter settles (debounce ~350ms, matching the year-input debounce).
   Optional lazy/per-dimension via IntersectionObserver as a later optimization.
3. **No TMDb quota risk.** TMDb free API has **no daily/monthly request quota** — it is rate-limited only
   (~40–50 req/s per IP; verified against TMDb developer docs). Our local limiter (40 req/10s, burst 40) is
   ~10× more conservative than TMDb's real ceiling, so cold fan-out latency (~5s for a 60-call cold combo
   today) is governed by **our** config, not TMDb. Headroom exists to give facet-counts a dedicated/relaxed
   rate budget if needed. The real constraint is **shared-limiter contention** (facet fan-out briefly
   starving other in-app TMDb calls) — mitigated by debounce, warm cache, on-demand compute, and bounded
   fan-out.
4. **Fail-soft + fallback.** If facet-counts is unavailable or returns partial, the rail silently falls back
   to the existing single total `符合 N 部` — the page never hard-fails.
5. **Story shape: BE / FE split** (recommended to SM) — BE endpoint + service is > 3 subtasks; FE is the
   `FacetCountChip` revival.
6. **Rate-limit isolation (pre-mortem F2).** The facet-counts fan-out MUST use a separate capped / lower-
   priority rate budget so it never drains the shared limiter and starves interactive TMDb calls
   (detail / search / homepage). `golang.org/x/time/rate` is a single bucket → mechanism resolved to
   `errgroup.SetLimit(N)` (a second limiter is the escalation). The FE re-poll for `partial` MUST use
   backoff + a conservative N (AR-F7) or it re-issues the same fan-out and re-drains the shared bucket.
7. **Count accuracy (pre-mortem F3).** Counts are TMDb-reported `total_results` (approximate; page-capped at
   ~10k). Derive every facet sub-query from the SAME `buildDiscoverParams` as the grid so a count and the grid
   it predicts stay consistent; present as exact but tolerate small drift.
8. **Dead-end facets stay selectable (pre-mortem F6).** A 0-count facet is DIMMED, not hard-disabled — the
   user may want to SWITCH to it (replace another filter), so it must remain clickable.
9. **Cache-key normalization for counts (AR-F2).** `total_results` is identical across `sort` and `page`, but
   `discoverCacheKey` includes both — reusing `buildDiscoverParams` as-is fragments the cache (changing sort
   misses the whole rail). Count sub-queries MUST set `SortBy=""` and `Page=1` before calling Discover so all
   sorts/pages share one entry.
10. **Pin an explicit language for count probes (AR-F3).** An empty `Language` makes
    `DiscoverMoviesWithFallback` loop the language chain (multiple `/discover` calls per probe) AND makes
    `total_results` locale-dependent. Count probes set an explicit `Language` (skip the loop); counts are
    documented as per-locale.
11. **Don't cache zero-count probes / give them a short TTL (AR-F5).** A transient or wrong-locale `0` cached
    the full 1h would wrongly DIM a facet (undermines Decision #8). Skip caching a `total_results==0`, or
    cache it with a much shorter TTL.
12. **Dedicated TTL constant + cache type for counts (AR-F8/AR-F4).** Introduce `FacetCountCacheTTL` (may start
    = 1h) instead of borrowing `TrendingDiscoverCacheTTL`, and tag count entries with a distinct cache `type`
    (e.g. `"tmdb_facet"`), so count freshness/eviction can be tuned without touching the grid. NOTE: targeted
    eviction by type alone is limited — the manual purge runs a wholesale `clearTable("cache_entries")`; true
    isolation needs a separate table (Hard, deferred).
13. **(prerequisite, AR-F1) `cache_entries` has no scheduled expiry sweep** — the fan-out is a write-amplifier
    that makes this latent gap urgent. See the prerequisite task + "Architecture Review" design notes below.

### Investigation Findings (Step 2 — anchor points)

**BE anchors (file:line):**

- `internal/tmdb/types.go:254-266` — `DiscoverParams{ GenreIDs []int; YearGte/YearLte int; Region string;
  VoteAverageGte/Lte float64; WatchProviders []int; WatchRegion/Language/SortBy string; Page int }`.
  `SearchResultMovies/TVShows.TotalResults int` (`:80` / `:88`) — the count we sum.
- `internal/handlers/tmdb_handler.go` — `parseDiscoverParams` `:549-592` (query → `DiscoverParams`, validates
  ranges → `*TMDbError`); `DiscoverMovies` handler `:269-281`; `RegisterRoutes` `:595-635` (add the new route
  after `:616`); `TMDbServiceInterface` `:16-28` (extend with `DiscoverFacetCounts`).
- `internal/services/tmdb_service.go:70-110` `NewTMDbService` wires `Client` + `CacheService` (`:61-65`
  struct) — add a `DiscoverFacetCounts` passthrough.
- `internal/tmdb/errors.go:10-33` `TMDB_*` constants + `NewInvalidYearRangeError` `:152-159` pattern —
  **NO new error code** (reuse parse validation; fan-out failures fail-soft per Rule 27 ③).
- `internal/tmdb/client.go:96-102` — `Client.limiter` is UNEXPORTED, one per `Client` (`:124-126`).

**Resolved decisions:**

- **ADR-2 transport = synchronous-with-time-budget (CONFIRMED feasible).** Handler is synchronous
  (parse → service → `SuccessResponse`). `DiscoverFacetCounts` runs the fan-out under a ~800ms
  `context.WithTimeout` + `errgroup`; returns counts resolved within budget + `partial:true` for the rest;
  FE re-polls (TanStack Query refetch) to fill the tail. Cached facets (1h) resolve instantly, so steady
  state is rarely partial.
- **#6 rate isolation = `errgroup.SetLimit(N)` at the service layer** (x/sync available) bounding facet
  fan-out concurrency (e.g. N=4) — non-invasive, since the limiter is unexported/one-per-Client. Escalation
  if shared-bucket contention proves real: add a 2nd `rate.Limiter` field on `Client` (additive). N tuned in
  implementation.

**Response shape** (keys mirror the FE chip `data-testid` ids — genre `genre.id`, region `region.code`,
rating value, platform `platform.id`):

```json
{ "counts": { "genre": { "28": 340 }, "region": { "TW": 88 }, "rating": { "8": 95 }, "platform": { "8": 540 } }, "partial": false }
```

**FE anchors:** `discoverFilters.ts:54-78` facet inventory (18 genres / 5 regions / 3 platforms / 4 ratings);
`FilterPanel.tsx` chip maps (`:147` / `:169` / `:219` / `:240`) gain a per-chip Mono count + dimmed-0; new
`useDiscoverFacetCounts` hook mirrors `useDiscoverResults`; `FacetCountChip` is `.pen`-only today (build in
the FE story).

## Implementation Plan

### Tasks

Ordered by dependency (design → BE contract → FE). SM may split at the `— FE story —` divider.

- [ ] **Task P (PREREQUISITE — separate infra story, AR-F1):** Wire a scheduled `cache_entries` expiry sweep.
  - Files: a new sweep service (mirror `internal/services/backup_scheduler.go`) calling
    `CacheRepository.ClearExpired`; wire it in `cmd/api/main.go`.
  - Guardrails: see "Architecture Review → AR-F1 design notes" in Additional Context (lifecycle pattern,
    NEVER VACUUM-on-ticker, swallow+log errors, ~30–60 min interval). Benefits the whole app; MUST land
    before facet-counts ships (the fan-out is the amplifier).

- [ ] **Task 0 (design prerequisite — ux-designer, NOT this dev flow):** Update `.pen` I1-D-v2 (`fxCVk`) —
      re-add `FacetCountChip` per-chip Mono counts with (i) computing / progressive-fill and (ii)
      dead-end-dimmed-but-selectable states, desktop rail only; regen `i1-d.png` per CLAUDE.md.

_— BE story —_

- [ ] **Task 1: Add the `FacetCounts` response type.**
  - File: `apps/api/internal/tmdb/types.go`
  - Action: `type FacetCounts struct { Counts map[string]map[string]int \`json:"counts"\`; Partial bool \`json:"partial"\` }`
  - Notes: outer key = dimension (`genre|region|rating|platform`), inner key = facet id as string (mirrors FE chip ids).
- [ ] **Task 2: Add `DiscoverFacetCounts` to `CacheService`.**
  - File: `apps/api/internal/tmdb/cache.go` (+ `CacheServiceInterface`)
  - Action: `DiscoverFacetCounts(ctx, base DiscoverParams, dims []string) (*FacetCounts, error)`. For each requested
    dimension, enumerate its facet values; for each value, clone `base` and **add** that facet (multi-select
    genre/platform → append; single-select region/rating → set), then call the existing cached
    `DiscoverMovies` + `DiscoverTVShows` and sum `TotalResults`. Run the fan-out in an `errgroup` with
    `g.SetLimit(N)` (N≈4) under a `context.WithTimeout(ctx, ~800ms)`. Counts resolved within budget are
    returned; any not resolved → `Partial=true` (omit that key). Reuses `discoverCacheKey` via the existing
    methods — NO new cache.
  - Notes: contextual semantics; `0` is a real (dead-end) count, kept. A single facet sub-query error is
    swallowed (fail-soft, Rule 27 ③) — that facet is omitted, the rest return.
  - **Count-param normalization (AR-F2/F3/F5/F8/F4):** before each sub-query set `SortBy=""`, `Page=1`, and an
    explicit `Language` (skip the fallback loop); cache with `FacetCountCacheTTL` + a distinct cache `type`
    (`"tmdb_facet"`); do NOT cache a `total_results==0` (or use a short TTL) so a transient 0 doesn't dim a
    facet for an hour.
- [ ] **Task 3: Service passthrough.**
  - File: `apps/api/internal/services/tmdb_service.go` (+ struct method); also extend
    `TMDbServiceInterface` in `tmdb_handler.go:16-28`.
  - Action: `DiscoverFacetCounts(ctx, params, dims)` → `s.cacheService.DiscoverFacetCounts(...)`.
- [ ] **Task 4: Handler + route.**
  - File: `apps/api/internal/handlers/tmdb_handler.go`
  - Action: `DiscoverFacetCounts(c *gin.Context)` — `parseDiscoverParams(c)` (reuse) + parse `dimensions` CSV
    (default all 4); call service; `SuccessResponse(c, result)`; errors via `handleTMDbError` /
    `handleValidationError`. Register `discover.GET("/facet-counts", h.DiscoverFacetCounts)` after `:616`.
    Swagger annotations. **No new error code** (reuse parse validation; fan-out fail-soft → `partial`).
- [ ] **Task 5: BE tests.**
  - Files: `apps/api/internal/tmdb/cache_test.go`, `apps/api/internal/handlers/tmdb_handler_test.go`
  - Action: table-driven + `MockTMDbService` — counts correctness, movie+tv sum, **cache reuse (no extra
    TMDb call within 1h)**, `SetLimit` concurrency bound, **partial on timeout**, **one-facet-fail fail-soft**,
    `dimensions` filter, validation-error passthrough, ApiResponse shape.

_— FE story (after `.pen` Task 0) —_

- [ ] **Task 6: Service call.** File: `apps/web/src/services/tmdb.ts` — `discoverFacetCounts(params, dims)` → endpoint.
- [ ] **Task 7: Hook (NEW).** File: `apps/web/src/hooks/useDiscoverFacetCounts.ts` — TanStack Query; debounced
      on committed filter (~350ms); refetch/poll while `partial` **with backoff + conservative concurrency
      (AR-F7)** so re-polls don't re-drain the shared limiter; gated off when shell≠v2 or endpoint unavailable;
      mirrors `useDiscoverResults`.
- [ ] **Task 8: Enumeration helper.** File: `apps/web/src/lib/discoverFilters.ts` — helper returning the
      `{dim, value}` set to request + the shared "add facet to params" mapping (so counts and chips agree).
- [ ] **Task 9: Per-chip count UI.** File: `apps/web/src/components/search/FilterPanel.tsx` — render a per-chip
      Mono count; `0` → **dimmed but still clickable** (not disabled); subtle fill (no spinner); counts only
      on the desktop-rail path (prop-gated; mobile sheet unchanged).
- [ ] **Task 10: Wire the rail.** File: `apps/web/src/components/search/DiscoverFilterRail.tsx` — call
      `useDiscoverFacetCounts`; pass counts to `FilterPanel`; recompute only non-toggled dimensions
      (instant-feel); fall back to the single total when unavailable/all-partial.
- [ ] **Task 11: FE tests.** Files: `FilterPanel.spec.tsx`, `useDiscoverFacetCounts.spec.tsx` — count render,
      dimmed-0-still-selectable, debounce, partial re-poll, disabled-when-not-v2, fallback-to-total.

### Acceptance Criteria

- [ ] **AC1 (happy, BE):** Given a filter selection and `dimensions=genre,region,rating,platform`, when
      `GET /api/v1/tmdb/discover/facet-counts` is called, then it returns `ApiResponse<FacetCounts>` where
      `counts[dim][id]` = summed movie+tv `total_results` for (base params + that facet).
- [ ] **AC2 (contextual):** Given 動作(28) is already selected, when facet-counts is requested, then
      `counts.region.KR` reflects (動作 ∩ 韓國), NOT 韓國-alone.
- [ ] **AC3 (cache reuse):** Given the same facet combo was requested within the 1h TTL, when requested again,
      then NO new TMDb call is made (served from the existing discover cache) — asserted via mock call-capture.
- [ ] **AC4 (rate isolation):** Given a fan-out, when it runs, then concurrent facet-counts TMDb calls never
      exceed N (`errgroup.SetLimit`), so interactive calls (detail/search/home) are not starved.
- [ ] **AC5 (time-budget partial):** Given some sub-queries are cold and exceed ~800ms, when the endpoint
      responds, then it returns the resolved counts with `partial:true` and never blocks beyond the budget.
- [ ] **AC6 (fail-soft):** Given one facet sub-query errors (e.g. `TMDB_TIMEOUT`), when computing, then that
      facet is omitted and the rest still return (Rule 27 ③ — never fail the page).
- [ ] **AC7 (validation passthrough):** Given `year_gte > year_lte`, when facet-counts is requested, then it
      returns the existing `TMDB_INVALID_YEAR_RANGE` error (no new error code).
- [ ] **AC8 (FE display):** Given the v2 shell and counts available, when the rail renders, then each chip
      shows its contextual count in JetBrains Mono.
- [ ] **AC9 (FE dead-end):** Given a facet's count is 0, when rendered, then the chip is DIMMED but still
      clickable (the user can switch to it).
- [ ] **AC10 (FE instant-feel):** Given a chip is toggled, when counts recompute, then only non-toggled
      dimensions update, already-known counts stay stable, there are no per-chip spinners, and the recompute is
      debounced (~350ms).
- [ ] **AC11 (FE fallback):** Given facet-counts is unavailable or all-partial, when the rail renders, then it
      falls back to the single total `符合 N 部` (no hard fail).
- [ ] **AC12 (scope/mobile):** Given the mobile `FilterBottomSheet`, when rendered, then it does NOT request
      per-facet counts (keeps the single draft total).

## Additional Context

### Dependencies

- Backend: existing `tmdb.CacheService`, `tmdb.Client` (rate limiter), `golang.org/x/sync/errgroup`.
- Frontend: existing `DiscoverFilterRail` / `FilterPanel` / `discoverFilters.ts`; `.pen` `FacetCountChip`.
- **Design (.pen) — PREREQUISITE for the FE story.** I1-D-v2 (`fxCVk`) currently shows the SINGLE total —
  ux3-3-2 decision 2改 REMOVED the per-facet count nodes. Reviving per-facet counts needs a ux-designer /
  Pencil pass to re-add `FacetCountChip` with its NEW states: (i) computing / progressive fill-in, (ii)
  dead-end-dimmed-but-selectable 0. Desktop rail only. Regenerate `i1-d.png` per the CLAUDE.md screenshots
  workflow.

### Testing Strategy

- BE: table tests for `DiscoverFacetCounts` (count correctness, movie+tv sum, cache reuse, one-facet-fails
  fail-soft) + handler test (params parse, `TMDB_*` error, ApiResponse shape).
- FE: Vitest for per-chip count render, dimmed-but-selectable 0, subtle fill, debounce, fallback-to-total.

### Risks & Mitigations (pre-mortem)

| Risk | Mitigation |
| ---- | ---------- |
| F1 — shimmer-on-every-chip makes the rail feel laggy (undercuts the instant-rail identity) | recompute only non-toggled dimensions; keep known counts stable; subtle fill, no spinners (Tech Decision #2) |
| F2 — facet fan-out drains the shared limiter and starves interactive TMDb calls (detail / search / homepage) | dedicated capped / low-priority rate lane for facet-counts (Tech Decision #6) |
| F3 — counts disagree with the grid (race / TMDb approximation + 10k page cap) | share `buildDiscoverParams` with the grid; tolerate small drift; treat as approximate (Tech Decision #7) |
| F4 — cold-start fan-out spike (fresh deploy / cache expiry) | debounce + lazy (compute visible / expanded dims) + bounded concurrency |
| F5 — mobile sheet does per-facet fan-out on small screen / slow net | facet-counts are DESKTOP-rail only; mobile keeps the single draft total (Out of Scope) |
| F6 — greyed-out 0 facets trap users who want to switch | dimmed but still selectable (Tech Decision #8) |

### Architecture Review (applied — AR-F#)

A fresh-context architect review of the cache reuse surfaced 9 findings (verified against code). Actionable
ones are folded into Technical Decisions / Tasks above. **`AR-F#` is distinct from the pre-mortem `F#` in the
Risks table above.**

| ID | Sev | Finding | Resolution |
| -- | --- | ------- | ---------- |
| AR-F1 | Critical | `cache_entries` has no scheduled expiry sweep (`ClearExpired` has no prod caller); fan-out is a write-amplifier → unbounded SQLite growth | **Prerequisite infra task** (Task P) — see design notes below |
| AR-F2 | High | count cache fragmented by `sort`/`page` (both irrelevant to a count) | normalize `SortBy=""`,`Page=1` (Decision #9, Task 2) |
| AR-F3 | High | empty `Language` fans out the fallback chain + makes the count locale-dependent | pin explicit `Language` (Decision #10, Task 2) |
| AR-F4 | Med | counts share `type="tmdb"`; manual purge is wholesale `clearTable("cache_entries")` | distinct cache type; true isolation = separate table (Hard, deferred) (Decision #12) |
| AR-F5 | Med | a transient/wrong `0` cached 1h wrongly dims a facet | don't cache zero / short TTL (Decision #11) |
| AR-F6 | Med | storage: 20-item JSON blob cached to read one int | known trade-off (reuse + grid synergy); compounded by AR-F1, mitigated by the sweep |
| AR-F7 | Med | FE re-poll re-drains the shared limiter | re-poll backoff + conservative N (Decision #6, Task 7) |
| AR-F8 | Low | TTL borrows trending's constant ("trending freshness" ≠ "count freshness") | dedicated `FacetCountCacheTTL` (Decision #12) |
| AR-F9 | Low | SQLite single-node cache assumption | explicit constraint; multi-instance = Hard, out of scope for single-NAS |

**AR-F1 — `cache_entries` expiry-sweep design notes** (for the Task P infra story; verified against code):

- **Safe by construction.** `ClearExpired` deletes `WHERE expires_at <= datetime('now')`
  (`cache_repository.go:124`) — the exact complement of the read filter `expires_at > datetime('now')`
  (`:33`). It only removes rows reads already treat as misses; it can never delete a live-served hit and
  introduces no timezone divergence (both sides use `datetime('now')`). Index-backed by
  `idx_cache_entries_expires_at`. Scoped to `cache_entries` only — AI/douban/wikipedia/image caches are
  separate tables.
- **Side effect 1 — write-lock contention** (mainly the FIRST sweep on a bloated table). WAL: readers don't
  block, but a large DELETE holds the single writer lock → concurrent cache `Set` may hit `busy_timeout` →
  transient cache-write misses (logged warnings, **non-fatal** — re-fetched next time). Mitigate: low
  interval; the table stays small after the first run.
- **Side effect 2 — disk is NOT reclaimed.** `auto_vacuum` is not set (default NONE) → DELETE frees pages for
  reuse but does NOT shrink the `.db` file. The sweep keeps ROW COUNT (query perf) healthy, not FILE SIZE —
  acceptable for an ongoing cache (stabilizes at the working-set high-water mark).
- **Implementation guardrails:**
  1. Mirror `backup_scheduler.go`: `Start(ctx)` + `time.NewTicker` + `defer ticker.Stop()` +
     `select { case <-ctx.Done(): return; case <-ticker.C: ... }` + `Stop()`. No goroutine leak.
  2. **Never** put `VACUUM` on the ticker (whole-DB rewrite + exclusive lock + ~DB-size temp space). If disk
     reclamation is ever needed, make it a rare manual/admin action.
  3. Swallow + log `ClearExpired` errors; never panic the goroutine.
  4. Interval ~30–60 min (aligned with the 1h TTL); ideally settings-configurable (mirror backup/scan schedulers).
  5. (latent, pre-existing, NOT caused by the sweep) verify how `modernc.org/sqlite` serializes `time.Now()`
     (local) vs `datetime('now')` (UTC) — works today (hits happen) so it's consistent; the sweep stays
     consistent with reads regardless.

### Notes

- **Transport decision (ADR-2 — DECIDED in Step 2: option (a)):** progressive fill-in mechanism —
  (a) **synchronous-with-time-budget**: endpoint computes up to ~800ms, returns ready counts + `partial:true`,
  FE re-polls for the rest ← **recommended**;
  (b) return-cached-only + async warm, FE re-polls;
  (c) blocking until all (5–15s) → rejected; (d) SSE → out of scope.
- **Phasing is an optional DELIVERY-SEQUENCING choice, NOT a value cut (Party Mode, premise-corrected
  2026-06-24).** End state = **full exact contextual counts** (this spec, unchanged). The earlier
  "grey-out-only MVP" idea assumed Discover browses the small private library; corrected — Discover navigates
  the **TMDb-scale catalog**, where exact counts genuinely help (see Problem Statement). Note the BE endpoint
  is the SAME either way (grey-out = `total_results > 0`), so a phase-1 only defers the FE progressive-fill /
  `partial` / re-poll layer + the exact-number `.pen` design. SM may still ship grey-out first as an
  increment toward exact counts if delivery risk warrants — but the target does not change.
