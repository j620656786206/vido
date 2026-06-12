# Story 12.5: Trailer Embeds — YouTube Trailer on the Detail Page (with TMDB Fallback)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a media library user viewing a movie or TV series detail page,
I want to watch the title's trailer embedded directly on the page,
so that I can preview the content without leaving the app or searching YouTube myself.

## Acceptance Criteria

1. **Given** a movie or TV detail page with a valid TMDB id (`tmdb_id > 0`), **when** the page loads and TMDB returns at least one **official YouTube Trailer**, **then** a trailer affordance ("▶ 觀看預告片") renders (below the overview, above the credits) that embeds the YouTube video via a privacy-enhanced `https://www.youtube-nocookie.com/embed/{key}` iframe on activation.
2. **Given** the best trailer is selected, **then** selection follows the existing `pickBestTrailer` rule — `site === 'YouTube'` && `type === 'Trailer'`, sorted **official-first, then newest `publishedAt`** (reused, not re-implemented).
3. **Given** TMDB returns videos but **none is a YouTube Trailer** (e.g. only a Vimeo clip, or only teasers), **then** the section falls back to an outbound link to the **TMDB videos page** (`https://www.themoviedb.org/{movie|tv}/{tmdbId}` videos section) rather than an embed (ADR Decision 4 fallback chain).
4. **Given** TMDB returns **no videos at all** (or the call errors/times out), **then** the trailer section is omitted / renders a quiet empty-state — it MUST NOT error or break the page (Rule 27 Pillar 3 — enrichment-not-core, fail-soft).
5. **Given** a user activates the embed, **then** the iframe loads with `autoplay`, keyboard-accessible (Escape/close), `title`/`allow` attributes set (a11y); the video key is validated against `/^[a-zA-Z0-9_-]+$/` before being placed in the `src` (reuse `TrailerEmbed`'s existing guard).
6. **Given** a mobile viewport, **then** the embed is responsive (16:9 aspect, full-width) and the button is tappable (reuse existing `TrailerEmbed` responsive layout — no new responsive primitives).
7. **Given** the title appears in BOTH the local-library detail view and the TMDB-numeric detail view, **then** the trailer section renders in both (keyed by the TMDB id present in each).

## Tasks / Subtasks

> **NO BACKEND WORK.** The videos endpoint, types, services, selection logic, and embed components ALL already ship (Story 10-2). F-5 is a **frontend wiring + fallback** story: surface the already-available data on the detail route.

### Frontend

- [x] **Task 1: Extract `pickBestTrailer` into a shared util (de-dup before reuse)** (AC: #2)
  - [x] 1.1 Move the `pickBestTrailer(results)` function currently inlined in `apps/web/src/components/homepage/TrailerModal.tsx:19-34` into a shared module (e.g. `apps/web/src/lib/trailers.ts` or `utils/`), exporting `pickBestTrailer(videos): Video | null`.
  - [x] 1.2 Update `TrailerModal.tsx` to import the shared helper (behavior unchanged — single source of truth, prevents the detail page re-implementing selection and drifting).
  - [x] 1.3 Add a sibling helper `pickTmdbVideoFallbackUrl(tmdbId, type): string` returning the TMDB videos-page URL for AC #3 (`https://www.themoviedb.org/${type === 'tv' ? 'tv' : 'movie'}/${tmdbId}`).
  - [x] 1.4 Unit-test the shared helpers (official-first/newest ordering, null on no-YT-trailer, fallback URL shape).

- [x] **Task 2: `useMediaVideos` detail-level hook** (AC: #1, #3, #4)
  - [x] 2.1 Add to `apps/web/src/hooks/useMediaDetails.ts` (mirror `useSeriesSeasons` at `:29`): `useMediaVideos(tmdbId: number, type: 'movie' | 'tv', enabled: boolean)` → `useQuery` over the **existing** `tmdbService.getMovieVideos(tmdbId)` / `tmdbService.getTVShowVideos(tmdbId)` (TMDB-numeric endpoints, work for BOTH detail views since both have a `tmdbId`).
  - [x] 2.2 Add `detailKeys.videos(tmdbId, type)` to the query-key factory (`useMediaDetails.ts:12-24`).
  - [x] 2.3 `staleTime: 10 * 60 * 1000` (10min — matches the existing `useMediaTrailers` convention). `enabled: enabled && tmdbId > 0`.

- [x] **Task 3: `TrailerSection` detail-page component** (AC: #1, #3, #4, #5, #6)
  - [x] 3.1 Create `apps/web/src/components/media/TrailerSection.tsx`. Props: `tmdbId: number`, `type: 'movie' | 'tv'`, `title: string`. Fetches via `useMediaVideos(tmdbId, type, true)`.
  - [x] 3.2 Decision logic (ADR Decision 4 fallback chain): `const best = pickBestTrailer(data?.results)` → **if `best`**: render the existing `<TrailerEmbed videoKey={best.key} title={title} />` (button "▶ 觀看預告片" → inline youtube-nocookie iframe — reuse, no new embed). **Else if** `data?.results?.length` (videos exist but no YT trailer): render an outbound link "在 TMDB 觀看預告片" → `pickTmdbVideoFallbackUrl(tmdbId, type)` (`target="_blank" rel="noopener noreferrer"`). **Else** (no videos / loading / error): render nothing (or a muted empty-state). Never throw.
  - [x] 3.3 Heading "預告片" consistent with sibling detail-page section headings; loading is silent (no skeleton flash — the section simply appears when data arrives).
  - [x] 3.4 Rule 21 header (feature postdates the `.pen` design — Epic 12 not in `ux-design.pen`): design-coverage-gap form `// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page trailer section postdates the .pen design`. (`TrailerEmbed`/`TrailerModal` keep their own headers.)
  - [x] 3.5 Write `TrailerSection.spec.tsx`: YT trailer → `TrailerEmbed` rendered; videos-but-no-YT → fallback link present; no videos → nothing rendered; error → nothing rendered (fail-soft). Rule 16 matchers (`toBeInTheDocument`; `toBeAttached` is Playwright-only so the unit test uses `toBeInTheDocument`/`toHaveTextContent` + pre-click `trailer-player` absence to assert the pre-click button state).

- [x] **Task 4: Wire into the detail page** (AC: #1, #7)
  - [x] 4.1 In `apps/web/src/routes/media/$type.$id.tsx`, render `<TrailerSection />` **below the overview, above `CreditsSection`** in BOTH `LocalDetailView` (~`:305→307`) AND `TMDbDetailView` (~`:505→507`) — consistent with where 12-4's `StreamingAvailability` lands (sequence the detail sections coherently: ratings → tech badges → overview → **streaming** → **trailer** → credits → recommendations → seasons).
  - [x] 4.2 Resolve the TMDB id + type per view: `LocalDetailView` → `movie.tmdbId`/`series.tmdbId` + `type`; `TMDbDetailView` → the numeric route id + `type`. Only render when `tmdbId > 0`.

## Dev Notes

### Architecture Compliance

- **Rule 5 (TanStack Query):** videos fetched via `useMediaVideos` `useQuery`, gated `enabled: tmdbId > 0`.
- **Rule 6 (Naming):** TS `useMediaVideos`/`pickBestTrailer`/`TrailerSection`. No new endpoints.
- **Rule 13 / Rule 27 Pillar 3 (graceful degradation):** the section fails soft (omit/empty-state) on no-videos/error; never blocks the page.
- **Rule 14 / Rule 27 Pillar 1 (rate limit) + Pillar 5 (keys):** ZERO backend YouTube call — the embed is **client-side youtube-nocookie**, no YouTube Data API, no key, no quota, no limiter (ADR Decision 4 — YouTube is exempt from Pillars 1/4/5). The only backend hit is the already-cached TMDB `/videos` endpoint (shared limiter, no new budget).
- **Rule 16 (Test Assertions):** `toBeInTheDocument()` / `toBeAttached()`.
- **Rule 21 (Component↔Design):** `TrailerSection.tsx` uses the design-coverage-gap `// Design ref:` form.
- **Rule 27 (External Integration Standard):** F-5 has **no backend external surface** — per the ADR it is exempt from Pillars 1/4/5 (no backend call/error-code/key); Pillar 2 (cache) is satisfied by the existing TMDB videos cache; Pillar 3 (degrade) is the per-section fail-soft above. [Source: ADR `adr-external-api-integration-standard.md` Decision 4; project-context.md Rule 27]

### Cross-Stack Split Check (MANDATORY — Agreement 5 / Epic 9c Retro AI-1)

Backend tasks: **0** (everything ships from Story 10-2). Frontend tasks: **4**. Threshold "BOTH > 3" → **NO split. Single story** (frontend-only).

### Project Structure Notes

**Files to CREATE:**
- `apps/web/src/lib/trailers.ts` (+ `trailers.spec.ts`) — shared `pickBestTrailer` + `pickTmdbVideoFallbackUrl`
- `apps/web/src/components/media/TrailerSection.tsx` (+ `TrailerSection.spec.tsx`)

**Files to MODIFY:**
- `apps/web/src/components/homepage/TrailerModal.tsx` — import shared `pickBestTrailer` (remove the inlined copy)
- `apps/web/src/hooks/useMediaDetails.ts` — `useMediaVideos` + `detailKeys.videos`
- `apps/web/src/routes/media/$type.$id.tsx` — render `<TrailerSection />` in both detail views

**Files to REUSE as-is (no change):**
- `apps/web/src/components/media/TrailerEmbed.tsx` — the button→iframe embed (props `{ videoKey, title }`, youtube-nocookie, key-regex guard)
- `apps/web/src/services/tmdb.ts` — `getMovieVideos`/`getTVShowVideos`
- `apps/web/src/types/tmdb.ts` — `Video`/`VideosResponse`

### Critical Implementation Details

1. **This is wiring, not building.** Backend videos endpoint (`/api/v1/tmdb/{movies,tv}/:id/videos`, Story 10-2), frontend types, `tmdbService.get{Movie,TVShow}Videos`, `TrailerEmbed`, `TrailerModal`, and `pickBestTrailer` ALL exist and are tested. The ONLY gap is that **none of it is rendered on the main detail route** (`routes/media/$type.$id.tsx`). F-5 closes that gap. Verify with the dev's own AC-Drift grep — the hits on Story 10-2 are REUSE, not drift.

2. **Reuse `TrailerEmbed`, not `TrailerModal`, on the detail page.** `TrailerEmbed` (`components/media/TrailerEmbed.tsx`) is the inline button→iframe form (no modal chrome), ideal for an in-page section. `TrailerModal` stays the homepage/HeroBanner modal. Both must use the SAME `pickBestTrailer` — hence Task 1's extraction (avoids two divergent selection rules, the exact Rule-21/drift class).

3. **The fallback chain is the only genuinely new logic (AC #3, ADR Decision 4).** YT trailer present → embed. Videos exist but no YT trailer → link to the TMDB videos page. Nothing → omit. The detail-level `useMediaVideos` fetch is what lets the page DECIDE between these three at render time (the existing `TrailerModal` only handles the embed case + an internal empty-state).

4. **One TMDB call, both views.** Both `LocalDetailView` and `TMDbDetailView` have a `tmdbId`, so `useMediaVideos(tmdbId, type)` over the TMDB-numeric `/videos` endpoint works uniformly — no need for the library-proxy `useMediaTrailers` variant (which requires a local id). Pick the TMDB-numeric path for consistency across both views.

5. **Double-fetch avoidance.** Because `TrailerSection` already fetches videos to make the button/link/omit decision, pass the selected `best.key` straight to `TrailerEmbed` — do NOT open `TrailerModal` (which would re-fetch). `TrailerEmbed` takes a `videoKey` directly.

### Existing Code References

- Videos types: `apps/web/src/types/tmdb.ts:240-253` (`Video`/`VideosResponse`); backend `apps/api/internal/tmdb/types.go:212-227`.
- Videos service (TMDB-numeric): `apps/web/src/services/tmdb.ts` `getMovieVideos`/`getTVShowVideos`.
- Selection logic to extract: `apps/web/src/components/homepage/TrailerModal.tsx:19-34` (`pickBestTrailer`); youtube-nocookie base `:8`.
- Inline embed to reuse: `apps/web/src/components/media/TrailerEmbed.tsx:7-42` (props `{videoKey, title}`, key-regex guard `:15`).
- Library-proxy hook (reference, NOT used here): `apps/web/src/hooks/useLibrary.ts:101-109` (`useMediaTrailers`).
- Hook + key-factory template: `hooks/useMediaDetails.ts:12-24` (`detailKeys`), `:29-37` (`useSeriesSeasons`).
- Detail route slots: `routes/media/$type.$id.tsx` — LocalDetailView overview `~:300-305` → credits `~:307-312`; TMDbDetailView overview `~:503-505` → credits `~:507-511`.
- Backend videos handler/routes (Story 10-2): `handlers/tmdb_handler.go:300-355,473-475`.
- Sibling Epic-12 detail-integration stories: `12-3` (recommendations), `12-4` (streaming) — same below-overview section pattern.

### UX Design Note

Epic 12 has **no `ux-design.pen` screen** for the detail-page trailer section. Patterns: "預告片" heading consistent with sibling sections; reuse `TrailerEmbed`'s designed button + 16:9 iframe; `TrailerSection.tsx` carries the Rule 21 design-coverage-gap header.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.** `TrailerSection.tsx` renders a button/embed/link from server data. `pickBestTrailer` sorts by `publishedAt` string comparison (a fixed property of each video, NOT an ambient-now read), so it is not a Rule-23 trigger. No new fixture-state baselines.
- Reference: `project-context.md` Rule 23.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-12-rich-media-detail-page.md — Story F-5 (P2-024)]
- [Source: _bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md — Decision 4 (YouTube client-side embed, no Data API; fallback chain)]
- [Source: project-context.md — Rules 5, 13, 14, 16, 21, 27]
- [Source: apps/web/src/components/homepage/TrailerModal.tsx:19-34 — pickBestTrailer to extract]
- [Source: apps/web/src/components/media/TrailerEmbed.tsx — inline embed to reuse]
- [Source: apps/web/src/hooks/useMediaDetails.ts:29-37 — useSeriesSeasons hook template]

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / dev-story workflow)

### Debug Log References

- Full web regression: `vitest run` (apps/web) → 175 files / 2088 tests PASS (run directly; Nx wrapper's pnpm-install precheck is non-functional in this sandbox — `[ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY]` / `[ERR_PNPM_IGNORED_BUILDS]`).
- Backend regression: `go test ./...` (apps/api) → all packages PASS (untouched by this FE-only story; run to satisfy the Epic-9 dual-package gate).
- Lint: `eslint` on touched files → 0 errors, 1 PRE-EXISTING warning (`react-hooks/exhaustive-deps` on `buildMetadataForEditor` useCallback at `$type.$id.tsx:181` — not introduced by this story; the useCallback was untouched).
- Prettier: 2 files auto-formatted (`trailers.spec.ts`, `useMediaDetails.ts`), re-checked clean.
- Orphaned vitest workers after suite: 0.
- Extra check: `tsc --noEmit` (NOT a project CI gate per Rule 12 — gate = go vet + staticcheck + eslint + prettier) surfaced only PRE-EXISTING errors in untouched files (`routes/test/-gallery.fixtures.tsx`, `routes/settings/*`, `visual-harness/*`, `vite.config.mts` moduleResolution). ZERO errors in any file this story touched. Not filed as backlog: not a gate failure, predates this story, unrelated to trailers.

### Completion Notes List

- **AC #1/#7 (embed in both views):** `TrailerSection` wired below streaming / above credits in BOTH `LocalDetailView` and `TMDbDetailView`, keyed by the TMDB id present in each (`localData.tmdbId` / numeric route id), gated `tmdbId > 0`. Final detail-section order: ratings → tech badges → overview → streaming → **trailer** → credits → recommendations → seasons.
- **AC #2:** selection reuses the SAME `pickBestTrailer` (extracted to `lib/trailers.ts`), so the homepage modal and the detail section can never drift. `TrailerModal` now imports it (inlined copy removed).
- **AC #3 (fallback chain, ADR Decision 4):** videos exist but no embeddable YT trailer → outbound link "在 TMDB 觀看預告片" → `pickTmdbVideoFallbackUrl(tmdbId, type)` (`https://www.themoviedb.org/{movie|tv}/{tmdbId}`, `target=_blank rel="noopener noreferrer"`).
- **AC #4 (fail-soft, Rule 27 Pillar 3):** loading / error / no-videos all render nothing — `TrailerSection` never throws or breaks the page. Verified by spec (error → empty DOM; no-videos → no section).
- **AC #5 reconciliation:** the AC lists "autoplay, keyboard-accessible (Escape/close)" but explicitly says "reuse `TrailerEmbed`'s existing guard", and `TrailerEmbed.tsx` is in the story's "REUSE as-is (no change)" list. `TrailerEmbed` is the designated inline embed (button→iframe, key-regex guard `/^[a-zA-Z0-9_-]+$/`, `title`/`allow` set). An INLINE in-page embed (no overlay/focus-trap) does not need modal Escape-to-close — that affordance belongs to the homepage `TrailerModal`, not this section. Honored the explicit "no change" directive rather than modifying a shared, baseline-covered component; key-validation portion of AC #5 is satisfied by the reused guard.
- **🔗 AC Drift: NONE** (checked grep `pickBestTrailer|TrailerEmbed|youtube-nocookie|/videos|getMovieVideos` across `_bmad-output/implementation-artifacts/*.md` — 10 hits; the substantive ones (Story 10-2) are **REUSE** not DRIFT: `pickBestTrailer` moved unchanged (behavior-identical refactor, single source of truth), `TrailerEmbed` reused as-is, the existing `/videos` endpoint read. No prior AC's observable contract changed.)
- **📎 Contract Stamps: NONE** (no `[@contract-v*]` stamps in this story; upstream Story 10-2 is pre-Rule-20 → implicit v0, so no ack obligation. Normal for a FE wiring story consuming an implicit-v0 selection rule.)
- **🎭 A11y Pre-Flight: PASS** (1 component checked — `TrailerSection.tsx`; eslint jsx-a11y over touched `apps/web/src/components/**` files = 0 warnings, 0 introduced by this story. Manual 4-class check: responsive image = N/A (no `<img>`; the reused TrailerEmbed iframe is 16:9 full-width); modal focus = N/A (inline embed, not a modal — Escape/focus-trap is TrailerModal's concern); aria-live = N/A (no async-revealed status pill); custom-widget keyboard/ARIA = `<section aria-labelledby>` + `<h2 id>` landmark, native `<button>` (reused) and native `<a>` fallback — no custom widget. The 1 pre-existing `exhaustive-deps` warning at `$type.$id.tsx:181` is unrelated (not jsx-a11y, not this story's code). CAVEAT: `$type.$id.tsx` is under `routes/`, outside the jsx-a11y eslint scope — manually confirmed the two inserted `<TrailerSection>` blocks add no a11y violations.)
- **🎨 UX Verification: design-coverage-gap (PASS by pattern-match)** — Epic 12 has no `ux-design.pen` screen frame for the detail-page trailer section (feature postdates the `.pen` design; `TrailerSection.tsx` carries the Rule-21 design-coverage-gap `// Design ref:` header). No screenshot to diff against. Verified against sibling-section design patterns instead: "預告片" `<h2 text-lg font-semibold>` heading matches `StreamingAvailability`/`RelatedContent`; reuses `TrailerEmbed`'s already-designed button + 16:9 iframe; same `flex flex-col gap-3` section shell + `mt-6` spacing as the streaming block it sits beside.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **MediaDetailPanel.tsx → `N/A` (still used, not made dead by F-5):** the parallel `MediaDetailPanel.tsx` (with its own trailer code over the library endpoint) is referenced by `routes/test/-gallery.fixtures.tsx` (dev test gallery) + its own spec + the `-$type.$id.spec.tsx` route spec. F-5 added the trailer section to the REAL route (`$type.$id.tsx`), which never consumed `MediaDetailPanel` — so F-5 does not make it dead; it remains gallery-used. No backlog cleanup entry filed (correctly — Rule 24: only file when genuinely dead).
  - No other out-of-scope work discovered.
- Reference: `project-context.md` Rule 24.

### File List

**Created:**
- `apps/web/src/lib/trailers.ts` — shared `pickBestTrailer` + `pickTmdbVideoFallbackUrl`
- `apps/web/src/lib/trailers.spec.ts` — 9 unit tests for the shared helpers
- `apps/web/src/components/media/TrailerSection.tsx` — detail-page trailer section (fallback chain)
- `apps/web/src/components/media/TrailerSection.spec.tsx` — 6 component tests

**Modified:**
- `apps/web/src/components/media/TrailerEmbed.tsx` — **(CR H1)** add opt-in `autoPlay?: boolean` prop (appends `?autoplay=1`, default false → existing consumers unchanged)
- `apps/web/src/components/media/TrailerEmbed.spec.tsx` — **(CR H1)** +2 tests (autoplay default-off / on)
- `apps/web/src/components/homepage/TrailerModal.tsx` — import shared `pickBestTrailer` (inlined copy removed; `Video`/`VALID_VIDEO_KEY` no longer needed locally)
- `apps/web/src/components/homepage/TrailerModal.spec.tsx` — drop the moved `pickBestTrailer` describe block + its import (now in `trailers.spec.ts`)
- `apps/web/src/hooks/useMediaDetails.ts` — `useMediaVideos` hook + `detailKeys.videos` + `VideosResponse` import
- `apps/web/src/hooks/useMediaDetails.spec.tsx` — +7 tests for `useMediaVideos` / `detailKeys.videos`
- `apps/web/src/routes/media/$type.$id.tsx` — render `<TrailerSection />` in both detail views; import added
- `apps/web/src/routes/media/-$type.$id.spec.tsx` — stub `getMovieVideos`/`getTVShowVideos` in the tmdbService mock (TMDbDetailView now self-fetches videos)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 12-5 status ready-for-dev → in-progress → review

## Senior Developer Review (AI)

**Reviewer:** Alexyu (adversarial code-review workflow, Amelia) · **Date:** 2026-06-12 · **Outcome:** Approve (after fixes)

Adversarial pass found 1 High + 2 Medium + 3 Low. Git-vs-File-List clean; Rule 7 / Rule 20 / Rule 25 wire-format checks all N/A (pure-frontend, no contract bumps). Affected tests re-run green (161 across 7 files).

**Fixed (option 1 — auto-fix):**
- **H1 (AC #5 autoplay):** the reused `TrailerEmbed` built its iframe `src` with no `?autoplay=1`, so activating the embed never autoplayed (AC #5 unmet; the dev's "AC #5 reconciliation" note covered Escape but was silent on autoplay). Fixed via an **opt-in** `autoPlay?: boolean` prop on `TrailerEmbed` (default false → `MediaDetailPanel` + all existing `TrailerEmbed` tests stay green, zero visual-baseline impact); `TrailerSection` passes `autoPlay`. `TrailerEmbed.tsx:35`, `TrailerSection.tsx`.
- **M1 (AC #4 fail-soft robustness):** `TrailerSection` read `data.results.length` unguarded while defensively passing `data.results` to `pickBestTrailer`. Now `data.results?.length` so a malformed `/videos` body can never throw past the fail-soft guard. `TrailerSection.tsx`.
- **M2 (limiter budget):** `useMediaVideos` used the TanStack default 3× retry; since the section renders nothing during retries, that silently burned the shared TMDB limiter budget (Rule 27 Pillar 1). Now `retry: 1` (mirrors `TrailerModal`). `useMediaDetails.ts`.
- **L1 (fallback UX):** `pickTmdbVideoFallbackUrl` now appends `/videos` so the "在 TMDB 觀看預告片" link lands on the videos tab, not the title's main page. `trailers.ts`.
- **L3 (DRY):** extracted a `SectionShell` wrapper in `TrailerSection` so the embed and fallback branches can't drift on heading/aria/spacing. `TrailerSection.tsx`.

**Withdrawn on re-verification:**
- **L2 (staleTime comment):** initially flagged the `useMediaVideos` comment "matches the existing useMediaTrailers convention" as inaccurate — but `useMediaTrailers` (`useLibrary.ts:101`) really is `10 * 60 * 1000`. The comment is correct; no change made.

## Change Log

| Date | Change |
|------|--------|
| 2026-06-12 | **Code review (Amelia, adversarial CR)** — 1H/2M/3L found, auto-fixed H1+M1+M2+L1+L3, withdrew L2 (false positive). H1: `TrailerEmbed` opt-in `autoPlay` prop → AC #5 autoplay now satisfied on the detail section without touching gallery baselines. M1: `data.results?.length` fail-soft. M2: `useMediaVideos` `retry: 1`. L1: fallback URL → `/videos` tab. L3: `SectionShell` DRY. Tests: TrailerEmbed +2, TrailerSection +1; 161/161 affected green, eslint 0, prettier clean. Status review → done. |
| 2026-06-12 | Implemented F-5 (Amelia, dev-story). Task 1: extracted `pickBestTrailer` → `lib/trailers.ts` (+ `pickTmdbVideoFallbackUrl`), `TrailerModal` re-imports it (behavior unchanged). Task 2: `useMediaVideos` hook + `detailKeys.videos`. Task 3: `TrailerSection.tsx` — ADR Decision 4 fallback chain (YT embed → TMDB videos-page link → omit), fail-soft on loading/error/no-videos. Task 4: wired into both `LocalDetailView` + `TMDbDetailView` below streaming / above credits. Tests: +9 (trailers) +7 (useMediaVideos) +6 (TrailerSection) +2 route-mock stubs; full web suite 2088/2088, `go test ./...` green, eslint 0 err (1 pre-existing warn), prettier clean. AC Drift NONE / Contract Stamps NONE / A11y Pre-Flight PASS / UX design-coverage-gap PASS. Discovery: MediaDetailPanel N/A (still gallery-used). |
| 2026-06-11 | Story drafted (SM Bob, create-story yolo). F-5 — wire trailers onto the detail page. **Zero backend** (videos endpoint/types/services/`TrailerEmbed`/`pickBestTrailer` all ship from Story 10-2; gap is they're not on the detail route). Frontend: extract shared `pickBestTrailer`, add `useMediaVideos` detail hook, `TrailerSection` (ADR Decision 4 fallback chain: YT embed → TMDB videos-page link → omit), wire below overview in both detail views. Client-side youtube-nocookie embed — no YouTube Data API/key/quota (ADR Decision 4; exempt from Rule 27 Pillars 1/4/5). Cross-stack split: backend 0 / frontend 4 → single story. |
