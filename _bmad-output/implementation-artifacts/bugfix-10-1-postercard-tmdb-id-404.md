# Story: Bugfix 10.1 — PosterCard TMDb-ID 404 Regression

Status: review

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

- [x] Task 1: Route-loader ID-kind detection (AC: #1, #2, #8)
  - [x] 1.1 Edit `apps/web/src/routes/media/$type.$id.tsx` Route loader: add `classifyId(id: string): 'local-uuid' | 'tmdb-numeric'` (numeric-and-positive → `tmdb-numeric`; else → `local-uuid`)
  - [x] 1.2 Loader return type: `{ type: ValidMediaType; id: string; idKind: 'local-uuid' | 'tmdb-numeric' }`
  - [x] 1.3 Keep existing `notFound()` for empty-string `id` AND invalid `type`; do NOT add a notFound branch for numeric IDs (they are now valid)
  - [x] 1.4 Co-located unit test: 10 table-driven cases at `apps/web/src/routes/media/-$type.$id.spec.tsx` (note hyphen-prefix to exclude from routing per existing convention; expanded beyond story draft to include UUID v4, decimal-trap, negative-trap)

- [x] Task 2: TMDb-numeric branch in `MediaDetailRoute` component (AC: #3, #4, #5)
  - [x] 2.1 `MediaDetailRoute` reads `{ type, id, idKind }` from `Route.useLoaderData()`; the function now branches at the top: `idKind === 'tmdb-numeric'` → `<TMDbDetailView />`, else `<LocalDetailView />`. Refactor split the original body into two sibling components so the TMDb path never touches local-DB hooks.
  - [x] 2.2 `TMDbDetailView` calls `useMovieDetails(isMovie ? tmdbId : 0)` / `useTVShowDetails(!isMovie ? tmdbId : 0)`. The hooks already gate themselves with `enabled: id > 0` (`useMediaDetails.ts:53,66`), so the disabled branch issues zero network calls.
  - [x] 2.3 `LocalDetailView` is a verbatim move of the prior `MediaDetailRoute` body — zero behavioral diff for the UUID path. `useNavigate` re-instantiated inside the new function (no shared closure issues).
  - [x] 2.4 `TMDbDetailView` is exported (for unit tests) and reuses the `hasMetadata === true` branch's Tailwind classes — `relative min-h-screen bg-[var(--bg-primary)]`, `max-w-5xl px-4 py-6`, `flex flex-col gap-8 md:flex-row`, `aspect-[2/3]` poster fallback. Title/meta/overview/CreditsSection identical structure.
  - [x] 2.5 `if (detailsQuery.isError || !data) return <NotFoundComponent />;` — same UX as the local error path.
  - [x] 2.6 No `MetadataEditorDialog`, no `Pencil` Edit button, no `metadataSource` badge, no `TechBadgeGroup` in the TMDb branch — these only make sense for local rows. AC-aligned regression test `editor button is NOT in DOM` confirms.
  - [x] 2.7 Container div carries `data-testid="tmdb-detail-view"`; targeted by 9 of the 11 new TMDb-branch tests.

- [x] Task 3: Owned-state indicator (AC: #6)
  - [x] 3.1 `const ownership = useOwnedMedia([tmdbId])` invoked unconditionally inside `TMDbDetailView` (single-element array; hook normalises + dedupes per `useOwnedMedia.ts:16`).
  - [x] 3.2 Badge rendered top-right of the title row: `<span data-testid="tmdb-detail-owned-badge" class="… bg-emerald-900/30 …">📁 已在媒體庫</span>`. Emerald palette mirrors `DownloadPanel.ConnectionStatusBadge` (`apps/web/src/components/dashboard/DownloadPanel.tsx:120`).
  - [x] 3.3 Inline code comment near the `useOwnedMedia` call cites Story 10-4 + the deferred `GET /api/v1/movies/by-tmdb/:tmdbId` follow-up.
  - [x] 3.4 `showOwnedBadge = !ownership.isLoading && ownership.isOwned(tmdbId)` — badge only renders post-load AND when truly owned. Loading-state regression test verifies absence.

- [x] Task 4: PosterCard call-site audit (AC: #7)
  - [x] 4.1 Audit table at end of Dev Agent Record below — 5 entries × 2 categories (UUID-correct vs TMDb-fixed-transitively). Re-verified at HEAD (file/line numbers stable since story bootstrap): LibraryGrid.tsx:173/279, RecentlyAdded.tsx:82, MediaGrid.tsx:61/77/103/117, ExploreBlock.tsx:138.
  - [x] 4.2 Call Site Coverage paragraph appended to Dev Notes (see "Call Site Coverage" subsection below).

- [x] Task 5: Test coverage (AC: #9)
  - [x] 5.1 Existing spec `-$type.$id.spec.tsx` extended with 2 new top-level `describe` blocks: `classifyId` (10 cases) + `TMDbDetailView` (11 cases).
  - [x] 5.2 `vi.mock('@tanstack/react-router', ...)` + `vi.mock('../../hooks/useOwnedMedia', ...)` added at the top of the file (auto-hoisted). `tmdbService.*` mocks reuse the existing `vi.mock('../../services/tmdb', ...)` block.
  - [x] 5.3 Movie + TV variants: `tmdbService.getMovieDetails` / `getTVShowDetails` toHaveBeenCalledWith(numericId). Local-DB hooks are not imported by `TMDbDetailView`, so the negative assertion is structural rather than runtime.
  - [x] 5.4 `mockRejectedValueOnce(new Error('TMDB_TIMEOUT'))` for both movie and tv → asserts `<NotFoundComponent />` renders ("404", "找不到該媒體內容", and `tmdb-detail-view` absent).
  - [x] 5.5 `ownedTrue` fixture (`isOwned(12345) === true`) + assertion that `tmdb-detail-owned-badge` is present and contains "已在媒體庫".
  - [x] 5.6 `screen.queryByRole('button', { name: /編輯|edit/i })` and `queryByTestId('edit-metadata-button')` both `not.toBeInTheDocument()`.
  - [x] 5.7 `ExploreBlock.spec.tsx` smoke added: `expect(card).toHaveAttribute('href', '/media/movie/83533')` — confirms the regression locus emits a TMDb-numeric URL that the route classifier picks up. Used `as ReturnType<typeof useExploreBlockContent>` instead of `as any` to avoid +1 lint warning.
  - [x] 5.8 `pnpm nx test web` 1760/1760 PASS; `pnpm nx test api` PASS on retry (`TestScannerService_SSEBroadcast_ScanCancelled` flaked once — pre-existing race condition, filed `preexisting-fail-scanner-sse-scan-cancelled-flake` in sprint-status per Epic 9c retro AI-2). `pnpm lint:all` 0 errors / 129 warnings (baseline restored after type-cast cleanup).

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

### Call Site Coverage

The fix is **centralized at the route loader** (`classifyId` + `idKind` + `MediaDetailRoute` early-return), not at any of the five PosterCard call sites. This means:

1. UUID-emitting call sites (`LibraryGrid` × 2, `RecentlyAdded`) keep the existing local-DB flow with zero diff.
2. TMDb-numeric-emitting call sites (`MediaGrid` × 4, `ExploreBlock`) automatically resolve to the TMDb branch the moment the loader sees a numeric `id` — no change needed at the call site.
3. Future PosterCard call sites added by Epic 11+ inherit the same routing behavior for free.

The single decision boundary is `classifyId(id)`, so any future regression around poster routing has exactly one diagnostic surface. This also incidentally repairs `/search` results (`MediaGrid`), which were 404-ing for the same root cause but weren't called out explicitly in the user-reported bug.

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

claude-opus-4-7[1m] (Amelia, BMAD dev agent)

### Debug Log References

- Pre-change route file lint warning at `routes/media/$type.$id.tsx:117:6` (useCallback exhaustive-deps on `localMovie`/`localSeries`) — pre-existing, shifted to line 146:6 after my refactor. Not introduced by this story.
- One backend test flake on full-suite run: `TestScannerService_SSEBroadcast_ScanCancelled` (apps/api/internal/services). Passed in isolation and on `--skip-nx-cache` rerun. Filed as `preexisting-fail-scanner-sse-scan-cancelled-flake` in sprint-status per Epic 9c retro AI-2.

### Completion Notes List

- **🔗 AC Drift:** FOUND — bugfix-1-media-detail-route-refactor AC #2 (UUID-only) ⇒ bugfix-10-1 AC #1/#2 [@contract-v1] (UUID OR TMDb-numeric). Documented in AC #8 + Change Log + Dev Notes Rule 20 subsection. Grep audit (`media/\$type/\$id` + `MediaDetailRoute` + `classifyId`) covered Epic 12 stories 12-1/12-2 (future, no deployed impact yet) and Story 2-3/2-4 (search/detail page predecessors).
- **📎 Contract Stamps:** FOUND (2 stamped ACs in this story v1; upstream bugfix-1 pre-Rule-20 / implicit v0; ack line "confirmed against [@contract-v0] (bugfix-1-media-detail-route-refactor AC #2)" embedded in AC #8).
- **🎯 Solution shape:** Single-point-of-fix at the route loader. Backend untouched. Frontend hooks/services untouched. Only `routes/media/$type.$id.tsx` adds new logic; only test files gain new assertions.
- **📁 Owned-state badge:** Story 10-4 read-through only; bi-directional redirect (TMDb → local UUID) deferred — needs `GET /api/v1/movies/by-tmdb/:tmdbId` endpoint not yet shipped.
- **🧪 Test counts:** 39 in `-$type.$id.spec.tsx` (28 pre-existing + 10 classifyId + 11 TMDbDetailView = +21 new) + 8 in `ExploreBlock.spec.tsx` (7 pre-existing + 1 smoke = +1 new). Web suite total: 1760 PASS (was ~1738 baseline).
- **⚙️ Regression gate:** `pnpm nx test web` 1760/1760 PASS; `pnpm nx test api` PASS (after one retry — flake filed). `pnpm lint:all` 0 errors / 129 warnings (matches baseline; no net new warnings).
- **🎨 UX Verification:** PASS — TMDb branch reuses the `hasMetadata === true` JSX subtree's exact Tailwind classes (`max-w-5xl`, `flex flex-col gap-8 md:flex-row`, `aspect-[2/3]` poster, emerald palette for owned badge mirrors `DownloadPanel.ConnectionStatusBadge` at `apps/web/src/components/dashboard/DownloadPanel.tsx:120-134`). No new design screen — visual identity is inherited from the shipped local detail page (Flow B). Editor button + tech-badge group + metadata-source pill intentionally absent from TMDb branch (no local row to drive them). Manual smoke deferred to NAS deploy per project convention.

### Call Site Audit (filled during Task 4)

| File | Line | ID source | Kind | Status |
|------|------|-----------|------|--------|
| `apps/web/src/components/library/LibraryGrid.tsx` | 173 | `m.id` from `LibraryItem.movie/series` | local UUID | unchanged — already correct |
| `apps/web/src/components/library/LibraryGrid.tsx` | 279 | `m.id` (virtualized variant) | local UUID | unchanged — already correct |
| `apps/web/src/components/library/RecentlyAdded.tsx` | 82 | `m.id` from `LibraryItem.movie/series` | local UUID | unchanged — already correct |
| `apps/web/src/components/media/MediaGrid.tsx` | 61, 77, 103, 117 | `movie.id` / `show.id` from TMDb search | TMDb numeric | unchanged code; resolves via route classifier |
| `apps/web/src/components/homepage/ExploreBlock.tsx` | 138 | `String(item.id)` from TMDb trending/discover | TMDb numeric | unchanged code; resolves via route classifier — **regression locus** |

### File List

**Modified (3):**
- `apps/web/src/routes/media/$type.$id.tsx` — `classifyId` export + `idKind` loader field; `MediaDetailRoute` branches on `idKind`; new `LocalDetailView` (verbatim move of original body) + new `TMDbDetailView` (exported for tests).
- `apps/web/src/routes/media/-$type.$id.spec.tsx` — `vi.mock('@tanstack/react-router', ...)` + `vi.mock('../../hooks/useOwnedMedia', ...)` at top; new `classifyId` describe (10 cases) + new `TMDbDetailView` describe (11 cases). Existing 28 tests untouched.
- `apps/web/src/components/homepage/ExploreBlock.spec.tsx` — 1 new test: poster card link encodes TMDb numeric id (smoke for the regression locus).

**Modified (sprint tracking):**
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `bugfix-10-1-postercard-tmdb-id-404`: `ready-for-dev` → `in-progress` → `review`. Added `preexisting-fail-scanner-sse-scan-cancelled-flake: backlog` per Epic 9c retro AI-2.
- `_bmad-output/implementation-artifacts/bugfix-10-1-postercard-tmdb-id-404.md` — this file (task checkboxes + Dev Agent Record + Status).

**AC drift reference (read-only, no code change):**
- `_bmad-output/implementation-artifacts/bugfix-1-media-detail-route-refactor.md` — implicit `[@contract-v0]` upstream per Rule 20.

**Untouched (audited, confirmed correct):**
- `apps/web/src/components/library/LibraryGrid.tsx`, `apps/web/src/components/library/RecentlyAdded.tsx`, `apps/web/src/components/media/PosterCard.tsx`, `apps/web/src/components/media/MediaGrid.tsx`, `apps/web/src/components/homepage/ExploreBlock.tsx`, `apps/web/src/hooks/useMediaDetails.ts`, `apps/web/src/hooks/useOwnedMedia.ts`, `apps/web/src/services/tmdb.ts`. All backend.
