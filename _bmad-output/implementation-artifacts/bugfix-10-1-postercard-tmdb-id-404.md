# Story: Bugfix 10.1 — PosterCard TMDb-ID 404 Regression

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a user landing on the Vido homepage (Epic 10),
I want clicking any poster card (trending, discover, recommended, recently-added) to open a working detail page,
so that I can browse content without hitting 404 dead-ends or seeing console error spam.

## Acceptance Criteria

1. [@contract-v1] Given a click on a `PosterCard` whose `id` is a TMDb numeric ID (e.g. `83533`, `76479`, `687163`, `1523145`), when the route `/media/$type/$id` loads, then the route MUST NOT call `GET /api/v1/movies/:id` or `GET /api/v1/series/:id` (which expect internal UUIDs) and MUST NOT render the existing 404 NotFoundComponent.
2. [@contract-v1] Given the route loader at `/media/$type/$id`, when it receives `id`, then it MUST classify the ID into one of `'local-uuid' | 'tmdb-numeric'` using a deterministic rule: a string matching `^\d+$` with `parseInt(id) > 0` is `'tmdb-numeric'`; everything else is treated as `'local-uuid'` (existing path). The classification MUST be exposed to the component as a typed `idKind` field on the loader return value.
3. Given `idKind === 'tmdb-numeric'`, when the detail page mounts, then it MUST fetch from `GET /api/v1/tmdb/movies/{id}` (movie) or `GET /api/v1/tmdb/tv/{id}` (tv) via the existing `useMovieDetails` / `useTVShowDetails` hooks, AND fetch credits via `useMovieCredits` / `useTVShowCredits` for progressive enhancement.
4. Given `idKind === 'tmdb-numeric'` and the TMDb fetch succeeds, when the page renders, then it MUST display: poster, title (zh-TW), original title, release/first-air year, vote average, genres, overview, top 6 cast members, director (movies only). Visual layout matches the existing `hasMetadata === true` branch of `MediaDetailRoute` (max-w-5xl, backdrop, two-column).
5. Given `idKind === 'tmdb-numeric'` and the TMDb fetch fails (network error, 404, 502), when the page handles error, then it MUST render the existing `NotFoundComponent` (same UX as before — no broken half-rendered state).
6. Given `idKind === 'tmdb-numeric'` and the user owns the title (via `useOwnedMedia([tmdbId])` returning `isOwned(tmdbId) === true`), when the page renders, then it MUST display a non-intrusive "📁 已在媒體庫" hint badge near the title. Clicking is NOT yet rerouted to the local-UUID page (full bi-directional redirect deferred — needs a `GET /api/v1/movies/by-tmdb/:tmdbId` endpoint that does not yet exist; tracked in Dev Notes).
7. Given the audit of all `PosterCard` call sites, when the audit is documented in the Dev Agent Record, then ALL four call sites (`LibraryGrid`, `RecentlyAdded`, `MediaGrid`, `ExploreBlock`) MUST be classified as either "emits local UUID (correct)" or "emits TMDb numeric ID (handled by ID-kind detection)" with file path + line number. No call site may be silently broken or omitted.
8. Given the existing `bugfix-1-media-detail-route-refactor` AC #2 ("The route `/media/$type/$id` accepts UUID string IDs (not just numeric)"), when this story changes route-loader behavior, then Dev Notes MUST acknowledge `confirmed against [@contract-v0] (bugfix-1-media-detail-route-refactor AC #2)` to satisfy Rule 20 forward-only retrofit AND record in the Change Log: contract widened from "UUID-only" to "UUID OR TMDb-numeric".
9. Given regression coverage, when the full test suite runs, then `nx test web` (existing 1700+ tests) MUST stay green, AND new tests MUST cover: (a) route loader returns correct `idKind` for both ID forms, (b) TMDb-numeric branch renders via TMDb API mock, (c) TMDb-numeric branch surfaces 404 on TMDb error, (d) owned-item indicator appears when `isOwned` returns true, (e) `ExploreBlock` poster click navigates to `/media/movie/{numericTmdbId}` and resolves (smoke).

## Tasks / Subtasks

- [ ] Task 1: Route-loader ID-kind detection (AC: #1, #2, #8)
  - [ ] 1.1 Edit `apps/web/src/routes/media/$type.$id.tsx` Route loader: add `classifyId(id: string): 'local-uuid' | 'tmdb-numeric'` (numeric-and-positive → `tmdb-numeric`; else → `local-uuid`)
  - [ ] 1.2 Loader return type: `{ type: ValidMediaType; id: string; idKind: 'local-uuid' | 'tmdb-numeric' }`
  - [ ] 1.3 Keep existing `notFound()` for empty-string `id` AND invalid `type`; do NOT add a notFound branch for numeric IDs (they are now valid)
  - [ ] 1.4 Co-located unit test: `apps/web/src/routes/media/$type.$id.spec.tsx` — table-driven `classifyId` cases: `'movie-uuid-abc'` → local-uuid; `'83533'` → tmdb-numeric; `'0'` → local-uuid (numeric but non-positive — falls through to existing local handler, which will 404 via the empty/invalid branch); `''` → notFound; `'abc-123'` → local-uuid

- [ ] Task 2: TMDb-numeric branch in `MediaDetailRoute` component (AC: #3, #4, #5)
  - [ ] 2.1 In `MediaDetailRoute`, read `idKind` from loader: `const { type, id, idKind } = Route.useLoaderData();`
  - [ ] 2.2 When `idKind === 'tmdb-numeric'`: call `useMovieDetails(parseInt(id, 10))` or `useTVShowDetails(parseInt(id, 10))` (DO NOT call local-DB hooks — guard with `enabled: idKind === 'tmdb-numeric' && tmdbId > 0`)
  - [ ] 2.3 When `idKind === 'local-uuid'`: existing `useLocalMovieDetails(id)` / `useLocalSeriesDetails(id)` flow unchanged
  - [ ] 2.4 Render branch: extract a small `<TMDbDetailView>` sub-component (or inline conditional) reusing the same JSX structure as the existing `hasMetadata` branch — backdrop, poster, title, meta-line, overview, credits. Source data shape is `MovieDetails` / `TVShowDetails` (already returned by `useMovieDetails`/`useTVShowDetails` in `useMediaDetails.ts`).
  - [ ] 2.5 On TMDb fetch error → `return <NotFoundComponent />` (same as existing local-error path)
  - [ ] 2.6 Editor button (`MetadataEditorDialog`) MUST be hidden when `idKind === 'tmdb-numeric'` (no local row to edit)
  - [ ] 2.7 Add `data-testid="tmdb-detail-view"` to the TMDb branch container so the regression test can target it

- [ ] Task 3: Owned-state indicator (AC: #6)
  - [ ] 3.1 In TMDb-numeric branch, call `useOwnedMedia([tmdbId])` (single-element array; the hook normalises and dedupes)
  - [ ] 3.2 If `isOwned(tmdbId) === true` → render badge `<span data-testid="tmdb-detail-owned-badge">📁 已在媒體庫</span>` near the title (top-right of meta-line, non-blocking)
  - [ ] 3.3 Document in code comment near the badge: "Story 10-4 ownership read-through; bi-directional redirect deferred — needs GET /api/v1/movies/by-tmdb/:tmdbId (out of scope for bugfix-10-1)"
  - [ ] 3.4 Do NOT render the badge while `useOwnedMedia.isLoading === true` (prevents a flicker)

- [ ] Task 4: PosterCard call-site audit (AC: #7)
  - [ ] 4.1 Verify each call site explicitly. Update Dev Agent Record's "Call Site Audit" table with file:line + ID source + classification. Required entries:
    - `apps/web/src/components/library/LibraryGrid.tsx:173` — `m.id` (UUID) — correct, no change
    - `apps/web/src/components/library/LibraryGrid.tsx:279` — `m.id` (UUID, virtualized variant) — correct, no change
    - `apps/web/src/components/library/RecentlyAdded.tsx:82` — `m.id` (UUID) — correct, no change
    - `apps/web/src/components/media/MediaGrid.tsx:61, 77, 103, 117` — `movie.id` / `show.id` (TMDb numeric, used by `/search` route) — now resolves via Task 1+2 (search clicks were ALSO 404-ing before this fix; this story incidentally repairs them)
    - `apps/web/src/components/homepage/ExploreBlock.tsx:138` — `String(item.id)` (TMDb numeric, **the regression locus**) — now resolves via Task 1+2
  - [ ] 4.2 Add a single-paragraph "Call Site Coverage" section to Dev Notes confirming the fix is centralized at the route level (not at each call site) — this is the simplest and least-error-prone shape.

- [ ] Task 5: Test coverage (AC: #9)
  - [ ] 5.1 Update `apps/web/src/routes/media/-$type.$id.spec.tsx` (note leading hyphen — TanStack Router excludes hyphen-prefixed files from routing; this is the existing test file). Add tests for `idKind` classification AND for the new TMDb-numeric render path.
  - [ ] 5.2 Mock setup: re-use existing `tmdbService.getMovieDetails` / `getTVShowDetails` mock pattern from `useMediaDetails.spec` if present; otherwise stub with `vi.mock('../../services/tmdb', ...)`.
  - [ ] 5.3 Test: TMDb-numeric URL renders title from TMDb API mock, NOT from local DB (assert `useLocalMovieDetails` mock was NOT called)
  - [ ] 5.4 Test: TMDb-numeric URL with TMDb error → renders `NotFoundComponent`
  - [ ] 5.5 Test: owned-item indicator appears when `useOwnedMedia` mock returns `isOwned(tmdbId) === true`
  - [ ] 5.6 Test: editor button hidden in TMDb-numeric branch
  - [ ] 5.7 Add `apps/web/src/components/homepage/ExploreBlock.spec.tsx` smoke: clicking a poster sets `Link.params.id = String(item.id)` and routes to `/media/$type/$id` (mock router). Don't deep-render the detail page in this spec — keep the unit boundary.
  - [ ] 5.8 Run full regression: `nx test web` MUST stay green. Run `pnpm lint:all` (Rule 12) before marking complete.

## Dev Notes

### Root Cause Analysis

Epic 10 introduced a homepage with `HeroBanner`, `ExploreBlock`, and recommendation rows that surface TMDb-sourced items (trending + discover) — content the user does NOT yet own. The `ExploreBlock` renders these via `<PosterCard id={String(item.id)} ... />` where `item.id` is the TMDb numeric ID (e.g. 83533).

`PosterCard` (`components/media/PosterCard.tsx:74-77`) builds a `<Link to="/media/$type/$id" params={{ type, id }}>`. The route loader at `routes/media/$type.$id.tsx:28-43` accepts any non-empty string. The component then calls `useLocalMovieDetails(id)` → `GET /api/v1/movies/:id` (apps/api/internal/handlers/movie_handler.go:90, 286), which does a DB lookup keyed on internal UUID. TMDb numeric IDs never match → repository returns `sql.ErrNoRows` → handler returns 404 → frontend renders `NotFoundComponent`.

**Console evidence (user-reported):** four 404s on initial homepage load — `83533`, `76479`, `687163`, `1523145` — all matching `/api/v1/movies/{id}` and `/api/v1/series/{id}` requests.

### Why this is an Epic 10 regression (not a pre-existing bug)

`bugfix-1-media-detail-route-refactor` (done 2026-03-28) intentionally narrowed the route contract from "tmdbId or UUID" to "UUID only" because at that time the only entry point was `LibraryGrid`, where every item already had a local row. Story 10-3 (`ExploreBlock`) introduced a NEW entry point (homepage) that surfaces TMDb-only items. The bugfix-1 contract no longer covers the new entry point — that is the regression.

### Solution Strategy — Decision Record

The sprint-status comment offered three options:

| Option | Approach | Verdict |
|--------|----------|---------|
| A | Route branching: emit `/media/$type/tmdb/$tmdbId` for TMDb-sourced items | **Rejected** — requires changes at every PosterCard call site, two route loaders to keep in sync, and breaks the URL stability that bugfix-1 established. |
| B | TMDb-backed detail page (single route, branch in component) | **Selected** — backend endpoints (`GET /api/v1/tmdb/movies/:id`, `GET /api/v1/tmdb/tv/:id`) and frontend hooks (`useMovieDetails`, `useTVShowDetails`) already exist. Single-point-of-fix at the route loader. Owned-state badge added as a stepping-stone toward the future redirect. |
| C | Disable click for un-owned items | **Rejected** — UX regression: removes the natural "tap to learn more" entry-point that homepage browsing depends on. Story 10-4 ownership badges show *which* items the user owns, but the click affordance is part of the discovery flow. |

**Selected: Option B.** Centralizing the fix at the route loader means `MediaGrid` (search results — `movie.id` is TMDb numeric there too) is repaired as a side effect, with no code change at the call site. This satisfies AC #7's audit requirement.

### Out-of-Scope (deferred)

The full UX would route an owned item's TMDb-ID click directly to its local-UUID page (so the user sees their own file/transcoding state, not the canonical TMDb metadata). That requires a new backend endpoint:

```
GET /api/v1/movies/by-tmdb/:tmdbId    → 200 {id: <uuid>} or 404
GET /api/v1/series/by-tmdb/:tmdbId    → 200 {id: <uuid>} or 404
```

This story explicitly leaves that for a follow-up (would be a cross-stack story per Rule 19/20 cross-stack-split — see retro-9c-AI2). For now AC #6 just shows an "已在媒體庫" hint so the user knows they can still find it under `/library`.

### Architecture Compliance

- **Rule 1 (Single Backend):** No backend changes; all changes in `apps/web/`.
- **Rule 4 (Layered Architecture):** Frontend route → component → hook → service — unchanged.
- **Rule 5 (TanStack Query):** Reuses existing `useMovieDetails` / `useTVShowDetails` query hooks. No Zustand for server state.
- **Rule 7 (Error Codes):** No new codes. TMDb errors propagate as before through `tmdbService.fetchApi`.
- **Rule 11 (Interface Location):** No interface changes.
- **Rule 12 (CI Lint):** `pnpm lint:all` MUST pass before commit.
- **Rule 13 (Error Handling):** TMDb fetch error → render `NotFoundComponent` (existing pattern). No swallowed errors.
- **Rule 16 (Assertions):** Use `toBeInTheDocument` for DOM presence; `toHaveTextContent` for title-mock assertions. Avoid `toBeTruthy`.
- **Rule 18 (Case Transformation):** TMDb response already passes through `snakeToCamel` in `tmdbService.fetchApi` (`apps/web/src/services/tmdb.ts:37`).
- **Rule 19 (Package Boundaries):** No backend changes, no concern.
- **Rule 20 (AC Contract Versioning):** This story stamps AC #1, #2 as `[@contract-v1]` because they redefine the route-loader contract that downstream stories may reference. AC #8 retrofits the upstream `bugfix-1` AC #2 as implicit `v0` (Rule 20 forward-only retrofit) and acknowledges it. Change Log entry below.

### Cross-Stack Split Check

- Backend tasks: 0
- Frontend tasks: 5 (Task 1–5)
- Verdict: **Single story** (no split required — backend tasks are zero, frontend is 5 ≤ 3-each-side rule does NOT trigger because BE side is empty).

### Key Files

| File | Change |
|------|--------|
| `apps/web/src/routes/media/$type.$id.tsx` | Add `classifyId` + `idKind` field; branch render in `MediaDetailRoute` |
| `apps/web/src/routes/media/-$type.$id.spec.tsx` | Add classification + TMDb-branch tests (existing test file uses leading-hyphen exclusion) |
| `apps/web/src/components/homepage/ExploreBlock.spec.tsx` | New: smoke test for poster→detail navigation |
| `apps/web/src/hooks/useMediaDetails.ts` | NO change — existing hooks (`useMovieDetails`, `useTVShowDetails`, `useMovieCredits`, `useTVShowCredits`) are reused |
| `apps/web/src/services/tmdb.ts` | NO change — `getMovieDetails`/`getTVShowDetails` already exist |
| `apps/web/src/hooks/useOwnedMedia.ts` | NO change — read-only consumer in Task 3 |
| `apps/web/src/components/media/PosterCard.tsx` | NO change — already accepts `id: string` (bugfix-1) |
| `apps/web/src/components/library/LibraryGrid.tsx` | NO change — emits UUID (audited) |
| `apps/web/src/components/library/RecentlyAdded.tsx` | NO change — emits UUID (audited) |
| `apps/web/src/components/media/MediaGrid.tsx` | NO change — emits TMDb numeric ID, fixed transitively via route classification |
| `apps/web/src/components/homepage/ExploreBlock.tsx` | NO change — emits TMDb numeric ID, fixed transitively |

### API Endpoints (already exist — no backend work)

- `GET /api/v1/movies/:id` (UUID lookup) — `apps/api/internal/handlers/movie_handler.go:90`
- `GET /api/v1/series/:id` (UUID lookup) — `apps/api/internal/handlers/series_handler.go:94`
- `GET /api/v1/tmdb/movies/:id` (TMDb int lookup) — `apps/api/internal/handlers/tmdb_handler.go:131`
- `GET /api/v1/tmdb/tv/:id` (TMDb int lookup) — `apps/api/internal/handlers/tmdb_handler.go:170`

### Data Shape Comparison

| Field | `LibraryMovie` (local) | `MovieDetails` (TMDb) | TMDb branch render |
|------|------------------------|------------------------|---------------------|
| `id` | UUID string | TMDb int | use `parseInt(id)` |
| `tmdbId` | optional int | always = id | derived |
| `title` | filename or TMDb title | TMDb title (zh-TW first) | `data.title` |
| `originalTitle` | optional | always present | `data.originalTitle` |
| `releaseDate` | optional | always present | `data.releaseDate` |
| `posterPath` | optional | always present | `data.posterPath` |
| `voteAverage` | optional | always present | `data.voteAverage` |
| `genres` | string[] | `Genre[]` ({id, name}) | map to names |
| `overview` | optional | always present | `data.overview` |
| `metadataSource` | "tmdb" / "ai" / "manual" | n/a | omit badge |
| `videoCodec` etc. | present (Story 8) | absent | omit `TechBadgeGroup` |
| `parseStatus` | required | n/a | omit fallback UI |

The TMDb branch is **simpler** than the local branch — no fallback UI, no tech badges, no editor.

### Project Structure Notes

- All code in `/apps/web/src/` (Rule 1).
- Test files co-located (Rule 9).
- `data-testid` naming follows existing convention (e.g. `tmdb-detail-view`, `tmdb-detail-owned-badge`).

### References

- [Source: project-context.md Rule 7 — Error Codes System (no new codes needed)]
- [Source: project-context.md Rule 18 — API Boundary Case Transformation]
- [Source: project-context.md Rule 20 — AC Contract Versioning]
- [Source: bugfix-1-media-detail-route-refactor.md AC #2 — original UUID-only route contract]
- [Source: 10-1-tmdb-trending-discover-api.md — Epic 10 backend (already shipped)]
- [Source: apps/api/internal/handlers/tmdb_handler.go:131,170 — existing TMDb detail endpoints]
- [Source: apps/web/src/hooks/useMediaDetails.ts:48,61 — existing TMDb hooks]
- [Source: apps/web/src/hooks/useOwnedMedia.ts — Story 10-4 ownership]

### Change Log

| Date       | Change |
|------------|--------|
| 2026-05-04 | [@contract-v0→v1] AC #1, #2: Route loader contract widened from "UUID-only" (bugfix-1 AC #2 implicit v0) to "UUID OR positive-integer-string"; downstream effect — any future caller passing a numeric string MUST expect TMDb-detail rendering (no longer a 404). Confirmed against [@contract-v0] (bugfix-1-media-detail-route-refactor AC #2). |

## Dev Agent Record

### Agent Model Used

(to be filled by dev-story)

### Debug Log References

(to be filled by dev-story)

### Completion Notes List

(to be filled by dev-story — must include "🔗 AC Drift: NONE|FOUND|N/A" per retro-10-AI2)

### Call Site Audit (filled during Task 4)

| File | Line | ID source | Kind | Status |
|------|------|-----------|------|--------|
| `apps/web/src/components/library/LibraryGrid.tsx` | 173 | `m.id` from `LibraryItem.movie/series` | local UUID | unchanged — already correct |
| `apps/web/src/components/library/LibraryGrid.tsx` | 279 | `m.id` (virtualized variant) | local UUID | unchanged — already correct |
| `apps/web/src/components/library/RecentlyAdded.tsx` | 82 | `m.id` from `LibraryItem.movie/series` | local UUID | unchanged — already correct |
| `apps/web/src/components/media/MediaGrid.tsx` | 61, 77, 103, 117 | `movie.id` / `show.id` from TMDb search | TMDb numeric | unchanged code; resolves via route classifier |
| `apps/web/src/components/homepage/ExploreBlock.tsx` | 138 | `String(item.id)` from TMDb trending/discover | TMDb numeric | unchanged code; resolves via route classifier — **regression locus** |

### File List

(to be filled by dev-story)
