# Tech Spike — `ux3-discover-facet-aggregation-be`

> **Status:** spike (pre-spec) · **Author:** dev (Amelia) · **Date:** 2026-06-24
> **Feeds:** `quick-spec` → SM `create-story`. NOT a story. No code written.
> **Origin:** Rule-24 ③ deferral from ux3-3-2 (shipped a single total `符合 N 部`; per-facet counts deferred).

## Problem

Show a **per-facet result count** next to every rail chip (`動作 340` / `Netflix 540` …) instead of the
single live total. TMDb `/discover` returns only `total_results` — **no facet aggregation** (confirmed vs
`apps/api/internal/tmdb/types.go`). So a count per facet value = **one `/discover` call per facet value**.

Facet inventory (from `apps/web/src/lib/discoverFilters.ts`): 18 genres + 5 regions + 3 platforms + 4 ratings
= **30 categorical values** (year is a numeric range → no per-value count). ×2 media types = **~60 sub-queries**
for one full recompute.

## Key findings (these change the approach — the backlog's "N×2 will blow the rate limit" is largely mitigated)

1. **A facet count is already a cached quantity.** `CacheService.DiscoverMovies/TVShows`
   (`internal/tmdb/cache.go`) caches the FULL `SearchResult*` — which includes `TotalResults` — keyed by
   `discoverCacheKey()` over **every** filter dimension, at **1h TTL** (`TrendingDiscoverCacheTTL`). A
   per-facet count is just `DiscoverMovies(params + thisFacet).TotalResults`. → **no new cache needed**; reuse
   the existing discover path. Adjacent facet recomputes share most sub-queries → high warm-cache hit rate.
2. **Fan-out is already rate-safe.** `internal/tmdb/client.go` fronts every call with
   `golang.org/x/time/rate` (40 req / 10s = 250ms spacing, `limiter.Wait(ctx)`), and the cache is checked
   **before** the limiter (Rule 27 Pillar 2). So a server-side fan-out **cannot** exceed the TMDb limit — it
   just serializes. `golang.org/x/sync` (errgroup) is in `go.mod` for bounded concurrency.
3. **The real cost is COLD first-call latency, not rate-limit violation.** All-cold worst case ≈ 60 × 250ms ≈
   **15s**. Warm (post-pre-warm or repeat) ≈ instant. This is the one decision the spec must make.

## The semantic lever (biggest cost+product decision — decide FIRST)

| Semantic | Meaning | Cost | Pre-warmable? |
|---|---|---|---|
| **Baseline** | count for this facet **alone**, ignoring other active filters (`動作` = all action titles) | Fixed ~60 calls, changes ~hourly | ✅ fully (cron, 1h) |
| **Contextual** | count if this facet is **added to the current selection** (live, recomputes every toggle) | ~60 calls **per filter change** | ❌ only via warm cache reuse |

Baseline is cheap, stable, fully pre-warmable, and arguably "good enough" for orientation. Contextual is the
expensive, live version the original design implied. **Recommend: ship Baseline first; treat Contextual as a
follow-up** — this is the cost/UX knob the product owner should set.

## Endpoint draft (for the spec to refine)

```
GET /api/v1/tmdb/discover/facet-counts?<current discover filters>&dimensions=genre,region,platform,rating
→ 200 ApiResponse<{
    counts: { genre: {28: 340, 16: 512, ...}, region: {TW: 88, ...}, platform: {...}, rating: {...} },
    basis: "baseline" | "contextual",
    partial: bool        // true if some counts are still warming (strategy B)
  }>
```
- New `CacheService.DiscoverFacetCounts(ctx, params, dims)` orchestrates the fan-out via errgroup; each
  sub-call goes through the existing cached `DiscoverMovies/TVShows`. Error code prefix `TMDB_` (Rule 7).
- Frontend (revival): re-enable the `.pen` `FacetCountChip` + per-chip Mono counts; gate on the endpoint;
  fall back to today's single total when `partial`/unavailable.

## Cold-latency strategies (pick in spec)

- **A — Lazy / per-dimension:** only fan out the dimension the user expands or hovers. Cuts cold cost to
  ~one section (e.g. 18 genres ×2 = 36). Simplest; counts appear on interaction. **Good default.**
- **B — Progressive backfill:** endpoint returns cached counts immediately + `partial:true`; a background job
  warms the rest; frontend polls / SSE-streams the fill-in. Best UX, most complex (job + transport).
- **C — Scheduled pre-warm:** a worker computes the **baseline** ~60 counts hourly into the existing cache;
  endpoint then almost always hits cache. Cheap, simple, pairs naturally with Baseline semantics. **Recommend
  C for baseline + A for any contextual/deep combos.**

## Open decisions for `quick-spec` / SM

1. **Baseline vs Contextual** counts (the semantic lever above) — product call.
2. **Cold-latency strategy** A / B / C (or C+A hybrid).
3. **Worth-it gate:** single total already shipped; is the info-density gain worth a BE endpoint now?
4. **Story shape:** likely a cross-stack story OR a 2-story split — BE `facet-counts` endpoint + FE
   `FacetCountChip` revival. Backend effort > 3 subtasks ⇒ probably **split**.

## Suggested next step

`quick-spec` to lock decisions 1–2, then SM `create-story`. If C+A+Baseline is chosen, this is a small,
low-risk BE addition (reuses cache + limiter + errgroup) plus a contained FE revival.
