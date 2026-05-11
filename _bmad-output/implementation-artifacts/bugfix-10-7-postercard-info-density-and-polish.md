# Story bugfix-10-7: PosterCard Info-Density & Polish — Hover-Lazy Runtime/Episode Line, Badge-Cluster Scale, lucide Star

Status: ready-for-dev

<!-- Created 2026-05-11 by SM Bob /create-story (YOLO). Sally UX delivery committed 58ba519 (Screen PC-1 spec, frame XlFIq). Bundles the 4 PosterCard polish items deferred from bugfix-10-4's Sally sign-off + Party Mode 2026-05-08. Cross-stack split check: BE = 0 tasks (Rule 15 verified — TMDb detail routes already wired), FE = 6 tasks → single story, NOT split (the sprint-status original "may split to 10-7a/b if BE>3" is moot — BE=0). -->

## Story

**As a** Vido user scanning a wall of poster cards (homepage explore blocks, search results, library grid),
**I want** each card to show the running time (movies) or season + episode count (series) next to the year below the poster — fetched lazily only when I hover that card so it costs nothing — plus the small visual papercuts fixed (the kebab-overlapped badge cluster shrinks as it fades, and the rating star is a crisp lucide SVG instead of a system emoji),
**so that** I can judge "is this worth opening" at a glance without clicking through, and the card feels polished and intentional rather than half-finished — the 4 items Alexyu flagged during the bugfix-10-4 sign-off ritual.

## Context & Why This Is Bundled

These are 4 polish items, all on `apps/web/src/components/media/PosterCard.tsx`, surfaced during `bugfix-10-4`'s Sally sign-off and the Party Mode design discussion (2026-05-08, Sally + Bob + Winston + Amelia + Murat). They share **one UX design pass** (Sally, committed `58ba519` — Screen **PC-1** in `ux-design.pen`, frame `XlFIq`, screenshot `_bmad-output/screenshots/flow-b-hover-detail-desktop/pc1-postercard-info-density-polish-bugfix-10-7.png`) and should ship in one DEV commit + one code review.

**This is a pure-frontend story — ZERO Go edits, ZERO migrations, ZERO swagger.** AC1's runtime/episode data is fetched **lazy-on-hover** from the TMDb-detail endpoints **that already exist and are already wired** (`GET /api/v1/tmdb/movies/:id`, `GET /api/v1/tmdb/tv/:id` — see Pre-flight #6 / Rule 15 below), via the **existing** `useMovieDetails` / `useTVShowDetails` hooks (`apps/web/src/hooks/useMediaDetails.ts`) — no new endpoint, no backend cache-merge. Cross-stack split check: BE = 0 tasks, FE = 6 tasks → single story (the `>3 each side` split threshold is not met). The sprint-status note's original "AC1 needs BE / may split to 10-7a + 10-7b" assumed a backend cache-merge strategy; Alexyu's design decision (2026-05-11) chose lazy-fetch-on-hover instead → no BE, no split.

## Acceptance Criteria

1. **PosterCard below-image metadata line shows year + runtime/episode-count, lazy-fetched on hover** (`apps/web/src/components/media/PosterCard.tsx`):
   - The `mt-2` block at the bottom of the card currently renders `<h3>{title}</h3>` then `{year && <p className="text-xs text-[var(--text-secondary)]">{year}</p>}` (PosterCard.tsx:225-231). Change the `<p>` to render a composed metadata string: **`{year} · {extra}`** where `extra` is the running time (movies) or `{seasons} 季 {episodes} 集` (series).
   - **Lazy fetch on hover** — the runtime / season-count is NOT in the TMDb discover/search/trending list payloads (that's the N+1 problem); it needs a detail call. Trigger that call **only when the user hovers the card**, with a **~200 ms hover-intent debounce** so a mouse sweeping across a grid doesn't fire a burst of detail requests:
     - On `onMouseEnter` of the `<Link>`: `setTimeout(() => setHoverIntent(true), 200)`, the timer id held in a `useRef`. On `onMouseLeave`: `clearTimeout` the pending timer (do NOT reset `hoverIntent` once it's `true` — the data is loaded, keep showing it; this also avoids re-fetch flicker). Clean the timer up on unmount (`useEffect(() => () => clearTimeout(ref.current), [])` — Rule 14).
     - The **visual** hover effects (image `scale-105`, the center play overlay, the kebab, the badge-cluster fade) MUST stay **instant** — they are CSS `:hover` / `lg:group-hover:` and are NOT gated by `hoverIntent`. Only the *data fetch* is debounced.
   - **Reuse the existing hooks — do NOT write a new fetch.** `useMovieDetails(id: number)` and `useTVShowDetails(id: number)` in `apps/web/src/hooks/useMediaDetails.ts` already wrap `tmdbService.getMovieDetails` / `getTVShowDetails` with TanStack Query, keyed by `detailKeys.movie(id)` / `detailKeys.tv(id)`, `staleTime: 10 * 60 * 1000`, `enabled: id > 0`. The detail page already uses them ⇒ the cache is shared (Rule 5). Gate the fetch by passing `id = 0` until you actually want it — the hooks' built-in `enabled: id > 0` then does the gating, **no change to `useMediaDetails.ts` needed**:
     ```tsx
     const tmdbId = /^\d+$/.test(id) ? Number(id) : 0;       // numeric id ⇒ TMDb item; UUID ⇒ owned-library item ⇒ 0
     const fetchId = hoverIntent ? tmdbId : 0;                // 0 until hover intent ⇒ hooks stay disabled
     const movieQ = useMovieDetails(type === 'movie' ? fetchId : 0);
     const tvQ    = useTVShowDetails(type === 'tv'   ? fetchId : 0);
     ```
     (Calling both hooks unconditionally is the React-correct way; the non-matching one always has `id = 0` ⇒ disabled. If DEV prefers, a thin `usePosterMeta(type, tmdbId, enabled)` wrapper hook in `useMediaDetails.ts` is acceptable — but it MUST reuse `detailKeys` so the cache stays shared, and existing `useMovieDetails`/`useTVShowDetails` callers must not change behaviour.)
   - **Owned-library cards are unaffected.** PosterCard is also rendered by `RecentlyAdded.tsx` / `LibraryGrid.tsx` with local-DB UUID `id`s — `/^\d+$/.test(uuid)` is `false` ⇒ `tmdbId = 0` ⇒ no fetch ⇒ those cards keep showing year-only, exactly as today. (Enriching owned cards from `useLocalMovieDetails`/`useLocalSeriesDetails` is **out of scope** — keep the diff tight; note it in Dev Notes.) Touch devices never fire `onMouseEnter` ⇒ naturally year-only there too.
   - **Format helpers (pure functions, Rule 16 ⇒ unit-tested — see AC #5).** Add to a new `apps/web/src/lib/formatMedia.ts` (or append to `apps/web/src/lib/timeFormat.ts` if DEV prefers — keep the co-located `.spec.ts`):
     - `formatRuntime(minutes?: number | null): string` — `minutes` falsy / `<= 0` → `''`; `minutes < 60` → `${minutes} 分鐘`; else `h = Math.floor(minutes/60)`, `m = minutes%60`, return `${h} 小時 ${m} 分` — **but drop the ` ${m} 分` when `m === 0`** → `${h} 小時`. (So `139` → `2 小時 19 分`, `120` → `2 小時`, `60` → `1 小時`, `47` → `47 分鐘`, `0`/`undefined` → `''`.)
     - `formatSeriesCount(seasons?: number | null, episodes?: number | null): string` — `seasons` falsy / `<= 0` → `''`; `episodes` falsy / `<= 0` → `${seasons} 季`; else `${seasons} 季 ${episodes} 集`.
     - `formatPosterMeta(year: number | null, extra: string): string` — `year && extra` → `${year} · ${extra}`; `year` only → `${year}`; `extra` only → `extra`; neither → `''`.
   - **Render.** `extra = type === 'movie' ? formatRuntime(movieQ.data?.runtime) : formatSeriesCount(tvQ.data?.numberOfSeasons, tvQ.data?.numberOfEpisodes)`; `metaLine = formatPosterMeta(year, extra)`; then `{metaLine && <p className="truncate text-xs text-[var(--text-secondary)]">{metaLine}</p>}`. The `<p>` keeps `truncate` (single line — same width as the card; the line just gets longer after the fetch resolves, MUST NOT push the card layout). `metadata` line does **not** get `<HighlightText>` (title still does). An optional `transition-opacity duration-200` on the `<p>` for a micro-fade when the line updates is allowed but **not required**.
   - **Types** (`apps/web/src/types/tmdb.ts`): `MovieDetails.runtime?: number` (minutes), `TVShowDetails.numberOfSeasons?: number`, `TVShowDetails.numberOfEpisodes?: number` — these already exist (the detail page consumes them); DEV confirms before using, does NOT redefine. (Backend returns `runtime` / `number_of_seasons` / `number_of_episodes`; `fetchApi`'s `snakeToCamel` → the camelCase names — Rule 18, already handled by `tmdb.ts`.)

2. **`[@contract-v1]` PosterCard top-right badge cluster fade-out adds `scale-95`** (`apps/web/src/components/media/PosterCard.tsx:154`):
   - The badge-cluster wrapper `<div>` is currently `className="absolute right-2 top-2 flex items-center gap-1 transition-opacity duration-300 lg:group-hover:opacity-0"`. Change to add a kinetic shrink as it fades (so it reads as "receding behind the kebab" rather than just dissolving): `transition-opacity` → `transition-all`, add `origin-top-right` (scale anchored at the badge's visual corner) and `lg:group-hover:scale-95`. `duration-300` / `ease-out` (implicit) **unchanged** — stays in sync with the image-wrapper's `lg:group-hover:scale-105 transition-all duration-300`.
   - Resulting className: `absolute right-2 top-2 flex origin-top-right items-center gap-1 transition-all duration-300 lg:group-hover:scale-95 lg:group-hover:opacity-0`.
   - The kebab (`opacity-0 ... lg:group-hover:opacity-100`, PosterCard.tsx:188) and the center play overlay (PosterCard.tsx:196-207) are **unchanged**. (Optionally the kebab could mirror this with `scale-95 → scale-100` on enter — **not required**; this AC only mandates the badge-cluster *exit* gets a `scale-95`.)
   - ⚠️ **bugfix-10-4 CR H2 cascade trap (still applies)**: don't add a `group-hover:opacity-*` rule on a *child* of this `<div>` that conflicts with the parent's `lg:group-hover:opacity-0` — the `metadataSource` child badge (PosterCard.tsx:170-174) was already de-conflicted in bugfix-10-4 CR H2 (it inherits the parent fade now); keep it that way, don't re-add child-level `opacity-*` gating.
   - Stamp rationale: this is the "the kebab-overlapped badge cluster recedes (opacity+scale) on hover, not just dissolves" invariant — future PosterCard redesigns that drop the scale must bump to `[@contract-v2]` + a Change Log row. PosterCard.tsx is implicit `v0` for this region (no prior stamp on the badge cluster) → forward-only retrofit per Rule 20.

3. **`[@contract-v1]` PosterCard rating badge uses lucide `<Star>` SVG, not the `⭐` emoji** (`apps/web/src/components/media/PosterCard.tsx:216-222`):
   - The rating badge currently renders `<span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-[var(--warning)]">⭐ {voteAverage.toFixed(1)}</span>`. Replace the `⭐ ` glyph with a lucide `<Star>`:
     ```tsx
     <span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-[var(--warning)]">
       <Star className="h-3 w-3 fill-[var(--warning)] text-[var(--warning)]" aria-hidden="true" />
       {voteAverage.toFixed(1)}
     </span>
     ```
     (`fill-[var(--warning)] text-[var(--warning)]` ⇒ a solid gold star matching the visual weight of `⭐` and the `.pen` PC-1 / `MQbvp` star treatment. DEV may use outline-only `text-[var(--warning)]` instead if it matches the `.pen` better on closer inspection — DEV+CR judgment, but DO NOT introduce a new color literal.)
   - Add `Star` to the existing import: `import { MoreHorizontal, Check, Play, Star } from 'lucide-react';` (PosterCard.tsx:6).
   - The `<span>` container classes, the `voteAverage !== undefined && voteAverage > 0` render guard, and the `absolute bottom-2 right-2 z-20` positioning are **unchanged**.
   - **Out of scope** (do NOT touch): `PosterCardMenu.tsx`, `MediaDetailPanel.tsx`, search-result cards, `RecentMediaPanel.tsx`, notification toasts — any other `⭐` in the app. If a future "full emoji sweep" wants to migrate those, that's a separate story (filed loosely in the sprint-status OUT-OF-SCOPE list, mirroring bugfix-10-6's pattern).
   - Stamp rationale: this is the canonical "no `⭐` emoji on the PosterCard rating — use lucide `<Star>` for cross-OS rendering consistency" invariant (parallels bugfix-10-6 AC #4's no-decorative-emoji-use-lucide invariant for admin chrome). Forward-only retrofit per Rule 20.

4. **PosterCard selection checkbox stays mode-gated — NO code change** (`apps/web/src/components/media/PosterCard.tsx:138-150`):
   - The `selection-checkbox` overlay is currently rendered only when `selectable` is `true` (`{selectable && (...)}`, PosterCard.tsx:138). **This stays exactly as is** — the design decision (Alexyu 2026-05-11, recorded on Screen PC-1 section `o91Cq`) is to **keep it mode-gated**, NOT reveal it on hover. Rationale: the hover state already carries the center play overlay + top-right kebab + bottom-right rating; adding a top-left checkbox on every hover is clutter for a feature most users won't use per-card; batch-select is an explicit mode (Story 5-7's original design); a hover-reveal would also need to resolve click ambiguity (checkbox-region `stopPropagation` vs `<Link>` navigation), the 4-element hover-state priority order, and the no-hover-on-touch fallback — none of which is in scope.
   - This AC is purely a **decision record** so it isn't re-litigated. DEV writes **zero code** for it. (If a future story revives "hover-reveal quick-select", it must design (a) the click-ambiguity, (b) the hover-element ordering, (c) the touch entry point, (d) the stacking relationship with AC2's badge-cluster fade — that's a fresh story, not this one.)

5. **Test coverage** (one new spec file + extend the existing PosterCard spec):
   - **`apps/web/src/lib/formatMedia.spec.ts`** (NEW — co-located with `formatMedia.ts`, Rule 9) — pure-function tests with the boundary cases:
     - `formatRuntime`: `0` → `''`, `undefined` → `''`, `null` → `''`, `59` → `'59 分鐘'`, `60` → `'1 小時'`, `120` → `'2 小時'`, `125` → `'2 小時 5 分'`, `139` → `'2 小時 19 分'`.
     - `formatSeriesCount`: `(0, …)` → `''`, `(undefined, …)` → `''`, `(1, undefined)` → `'1 季'`, `(1, 0)` → `'1 季'`, `(4, 34)` → `'4 季 34 集'`.
     - `formatPosterMeta`: `(2022, '2 小時 19 分')` → `'2022 · 2 小時 19 分'`, `(2022, '')` → `'2022'`, `(null, '4 季')` → `'4 季'`, `(null, '')` → `''`.
     - Rule 16 matchers (`toBe` for strings).
   - **`apps/web/src/components/media/PosterCard.spec.tsx`** (EXTEND — the file already has the router mock + `QueryClientProvider` wrapper from bugfix-10-1):
     - **AC1 — default state**: render a `type="movie"` PosterCard with `releaseDate="2022-..."`, no hover ⇒ the metadata `<p>` shows just `2022` (no runtime). (`useMovieDetails`/`useTVShowDetails` are disabled with `id=0` ⇒ `data` is `undefined` ⇒ `extra` is `''` ⇒ `formatPosterMeta(2022,'')` = `'2022'`.) Assert `screen.getByText('2022')` is in the document. (If an existing test already asserts the year, keep it green — the no-hover output is unchanged.)
     - **AC1 — after hover (movie)**: mock `useMovieDetails` to return resolved `{ data: { runtime: 139, ... } }` (typed-mock: `vi.mock('../../hooks/useMediaDetails', …)` returning `Partial<ReturnType<typeof useMovieDetails>>` — **no `as any`**, bugfix-10-2 CR M3 pattern), `vi.useFakeTimers()`, `fireEvent.mouseEnter` on the card, `vi.advanceTimersByTime(200)`, then `await screen.findByText('2022 · 2 小時 19 分')`. (Or, if the fake-timers + async-query combo is fiddly, mock the hook to return resolved data AND test that after `mouseEnter` + `advanceTimersByTime(200)` the `fetchId` becomes non-zero / the line updates — DEV picks the cleanest RTL incantation; the assertion MUST be `findByText`/`toBeInTheDocument`, never `toBeTruthy`.)
     - **AC1 — after hover (tv)**: same with `type="tv"`, mock `useTVShowDetails` → `{ data: { numberOfSeasons: 4, numberOfEpisodes: 34 } }` ⇒ assert `2022 · 4 季 34 集` (or whatever year the fixture uses).
     - **AC1 — UUID id (owned card) never fetches**: render with `id="0ce73c75-a742-..."` (a UUID), hover, advance 200 ms ⇒ the metadata line stays year-only AND the mocked `useMovieDetails` was called with `id = 0` (or simply: `tmdbId` derivation yields `0` ⇒ no fetch). A lighter version: just assert the line stays `2022` after hover for a UUID id.
     - **AC3 — rating uses lucide `<Star>`, not `⭐`**: render with `voteAverage={8.4}` ⇒ `expect(container.querySelector('svg')).not.toBeNull()` (lucide `<Star>` renders inline `<svg>` — loose assertion, don't couple to lucide internals), AND `expect(screen.queryByText(/⭐/)).toBeNull()`, AND `screen.getByText('8.4')` is in the document.
     - **AC2 — badge-cluster fade has scale (optional)**: a className-presence assertion (`expect(badgeClusterDiv).toHaveClass('lg:group-hover:scale-95')`) is acceptable but **not required** — CSS hover transitions aren't really unit-testable (Rule 16: don't use `toBeVisible`/`toBeTruthy` on hover state); the browser smoke / E2E covers the visual. If DEV adds it, fine; if not, fine.
   - **Anti-pattern guard**: ZERO new `as any` casts. Mock the detail hooks via `vi.mock` returning a `Partial<ReturnType<typeof useX>>`-shaped object (bugfix-10-2 CR M3 precedent — same pattern used in bugfix-10-6's specs).

6. **Design-traceability (Rule 21)**: Sally's design contract is **Screen PC-1 "PosterCard Info-Density & Polish (bugfix-10-7)"** in `ux-design.pen` (frame `XlFIq`, committed `58ba519`) — section `jWJgb` (`secA`) = AC1 info-density Before/After + 5-step spec, `jn1If` (`secB`) = AC2 badge-cluster scale-95, `R9PWRe` (`secC`) = AC3 ⭐→`<Star>` Before/After, `o91Cq` (`secD`) = AC4 keep-gated decision, `BZrw4` (`footer`) = SM/DEV handoff notes. `PosterCard.tsx` already has the Rule 21 header `// Implements: Component/PosterCardHover (MQbvp)` (bugfix-10-4 inaugural). Because this story also touches the **default (non-hover) state** — the below-image metadata line, which is part of `Component/PosterCard` (`RusTY`) — **DEV updates the header to reference both nodes** and adds the PC-1 screen ref:
   ```tsx
   // Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)
   // Design ref: ux-design.pen Screen PC-1 (XlFIq) — bugfix-10-7 info-density & polish
   // Source: ux-design.pen (Pencil app)
   ```
   (`RusTY` / `MQbvp` themselves were intentionally **not** edited by Sally — the post-hover loaded metadata line is a *fetched-state* enhancement, not the default state; Screen PC-1 is the contract for this story. So no `.pen` drift to chase between `RusTY` and the code's no-hover output.)

7. **AC Drift / Rule 20**: This story introduces NEW `[@contract-v1]` stamps on **AC #2** (badge-cluster fade adds `scale-95` — "recede, don't just dissolve" invariant) and **AC #3** (PosterCard rating uses lucide `<Star>`, not `⭐` — no-emoji-use-lucide invariant). Upstream: `PosterCard.tsx`'s hover-overlay layout (center play overlay + kebab top-RIGHT + rating bottom-RIGHT + top-right badge cluster) is `bugfix-10-4` AC #1 `[@contract-v1]` (`bugfix-10-4-hover-preview-viewport-flip.md`) — **DEV records in Dev Notes: `confirmed against [@contract-v1] (bugfix-10-4 AC #1)` — the hover-overlay positions/structure are PRESERVED; this story only (a) adds `scale-95` to the badge-cluster's fade-out animation (AC2) and (b) swaps the rating's `⭐` glyph for lucide `<Star>` (AC3); neither changes the contract's layout shape ⇒ no v2 bump of bugfix-10-4 AC #1 needed.** The new metadata `<p>` (AC1) is below the image, outside the hover-overlay layout contract entirely. No downstream story consumes these new stamps yet. Change Log MUST record the `[@contract-v0→v1]` rows. DEV greps `\[@contract-v[0-9]+\]` across `_bmad-output/implementation-artifacts/` and `confirmed against \[@contract-v[0-9]+\]` per Rule 20 before claiming completion.

8. **Regression gates** (Definition of Done):
   - `pnpm nx test web` PASS — baseline from the bugfix-10-6 closeout was **1790** tests (1788 DEV + 2 TEA-pass unit tests; verify the current count with a fresh run before claiming a delta) + the new `formatMedia.spec.ts` cases + the new `PosterCard.spec.tsx` assertions. No removals expected (the no-hover year line is unchanged ⇒ existing PosterCard tests stay green).
   - `pnpm nx test api` PASS (no Go changes; run anyway per Epic 9 Retro AI-1 mandatory-gate rule — if `TestScannerService_SSEBroadcast_ScanCancelled` flakes on the full-suite run, retry once and reference `preexisting-fail-scanner-sse-scan-cancelled-flake` in `sprint-status.yaml`; do NOT file a new entry).
   - `pnpm lint:all` → **0 errors / 122 warnings** — matches the bugfix-10-6 closeout baseline EXACTLY. ZERO new warnings: no new `as any`; the new `useEffect` cleanup has `[]` deps (no `react-hooks/exhaustive-deps` warning); `useRef`/`useState` for hover intent are camelCase. `prettier --check` (or `pnpm format:check`) clean on every touched file — run `pnpm exec prettier --write <files>` first (`feedback_format_before_commit` — subagent edits skip Prettier).
   - `pnpm run test:cleanup` verified — no orphaned vitest workers (Epic 9c retro lesson; never run test suites with `run_in_background`).
   - **No `.pen` edits** in this story — Sally's `58ba519` locked the PC-1 contract. DEV does NOT run `scripts/export-pen-screenshots.py`. (The `// Design ref: …` header comment in AC #6 is a source-code comment, not a `.pen` edit.)
   - **Rule 15 self-check** (record the result in Completion Notes): `grep -n "tmdbHandler.RegisterRoutes" apps/api/cmd/api/main.go` → confirms `main.go:526`; `grep -n "GetMovieDetails\|GetTVShowDetails" apps/api/internal/handlers/tmdb_handler.go` → `GET /api/v1/tmdb/movies/:id` (`:118`), `GET /api/v1/tmdb/tv/:id` (`:157`). **Both routes are already wired — DEV does NOT add any backend route. If (and only if) a fresh grep contradicts this Pre-flight, expand scope (new task + AC) before continuing — do not silently add it.**
   - Manual smoke (Task 6 substitute, CLI precedent): `pnpm nx serve web` against any backend → on the homepage / search results, hover a poster card and confirm (after ~200 ms) the line under the title becomes e.g. `2022 · 2 小時 19 分` (movie) or `2022 · 4 季 34 集` (series), and that the top-right type/badge cluster shrinks slightly as it fades, and the rating chip shows a crisp gold star SVG. Browser-pixel verification against the PC-1 screenshot deferred to user / NAS deploy. **Per bugfix-10-2/10-5/10-6 CLI precedent**, the deterministic vitest assertions (AC #5) substitute for the browser smoke since a CLI agent can't drive Chrome DevTools. Optional cheap insurance: `pnpm exec tsc -p apps/web/tsconfig.app.json --noEmit` — there are ~17 pre-existing errors (RecentMediaPanel / HeroBanner / Empty* / ScanProgress* / downloads / `media/$type.$id.tsx` route-type & arg-count) per bugfix-10-6 CR; **any NEW tsc error in `PosterCard.tsx` / `formatMedia.ts` / the spec files is yours — zero new is the bar.**

## Tasks / Subtasks

- [ ] **Task 1 — `formatMedia.ts` pure-function helpers** (AC: #1, #5)
  - [ ] 1.1 Create `apps/web/src/lib/formatMedia.ts` exporting `formatRuntime(minutes?: number | null): string`, `formatSeriesCount(seasons?: number | null, episodes?: number | null): string`, `formatPosterMeta(year: number | null, extra: string): string` per the exact rules in AC #1. (Or append to `apps/web/src/lib/timeFormat.ts` — DEV's call; keep the co-located `.spec.ts` either way.)
  - [ ] 1.2 Create `apps/web/src/lib/formatMedia.spec.ts` with the boundary cases in AC #5 (59/60/120/125/139 min, 1 season, 0 episodes, null runtime, `formatPosterMeta` all 4 branches). Rule 16 matchers (`toBe`).

- [ ] **Task 2 — PosterCard AC1: hover-intent lazy-fetch + metadata line** (AC: #1)
  - [ ] 2.1 Add hover-intent state: `const [hoverIntent, setHoverIntent] = useState(false)`; `const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)`. `handleMouseEnter`: if a timer is already pending, do nothing; else `hoverTimerRef.current = setTimeout(() => setHoverIntent(true), 200)`. `handleMouseLeave`: `clearTimeout` the pending timer + null the ref; do **not** reset `hoverIntent`. `useEffect(() => () => { if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current); }, [])` (Rule 14 cleanup).
  - [ ] 2.2 Wire `onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}` onto the `<Link>` (PosterCard.tsx:71-83) — alongside the existing `onClick={handleCardClick}`.
  - [ ] 2.3 Derive `tmdbId = /^\d+$/.test(id) ? Number(id) : 0` and `fetchId = hoverIntent ? tmdbId : 0`; call `const movieQ = useMovieDetails(type === 'movie' ? fetchId : 0)` and `const tvQ = useTVShowDetails(type === 'tv' ? fetchId : 0)` (import from `../../hooks/useMediaDetails`). **Do not modify `useMediaDetails.ts`** (the `enabled: id > 0` built-in does the gating). If DEV instead adds a `usePosterMeta` wrapper there, it MUST reuse `detailKeys` and not change existing callers' behaviour.
  - [ ] 2.4 Compute `extra = type === 'movie' ? formatRuntime(movieQ.data?.runtime) : formatSeriesCount(tvQ.data?.numberOfSeasons, tvQ.data?.numberOfEpisodes)` and `metaLine = formatPosterMeta(year, extra)`. Replace `{year && <p className="text-xs text-[var(--text-secondary)]">{year}</p>}` (PosterCard.tsx:230) with `{metaLine && <p className="truncate text-xs text-[var(--text-secondary)]">{metaLine}</p>}`. (Optional `transition-opacity duration-200` on the `<p>` — not required.)
  - [ ] 2.5 Confirm `MovieDetails.runtime?: number` / `TVShowDetails.numberOfSeasons?: number` / `TVShowDetails.numberOfEpisodes?: number` exist in `apps/web/src/types/tmdb.ts` (they do — the detail page consumes them). Do NOT redefine; if a name differs, use the actual one and note it.

- [ ] **Task 3 — PosterCard AC2: badge-cluster fade adds `scale-95`** (AC: #2)
  - [ ] 3.1 Edit the badge-cluster wrapper `<div>` at PosterCard.tsx:154: `transition-opacity` → `transition-all`; add `origin-top-right` and `lg:group-hover:scale-95`. Keep `absolute right-2 top-2 flex items-center gap-1`, `duration-300`, and `lg:group-hover:opacity-0`. Don't add child-level `group-hover:opacity-*` (bugfix-10-4 CR H2 trap — `metadataSource` child stays inheriting the parent fade).

- [ ] **Task 4 — PosterCard AC3: rating ⭐ → lucide `<Star>`** (AC: #3)
  - [ ] 4.1 Add `Star` to the lucide import: `import { MoreHorizontal, Check, Play, Star } from 'lucide-react';` (PosterCard.tsx:6).
  - [ ] 4.2 Replace `⭐ {voteAverage.toFixed(1)}` (PosterCard.tsx:219) with `<Star className="h-3 w-3 fill-[var(--warning)] text-[var(--warning)]" aria-hidden="true" />{voteAverage.toFixed(1)}` (mind whitespace — the `<span>`'s `flex gap-1` handles the spacing, so no literal space needed between icon and number). `<span>` container classes + `voteAverage > 0` guard + positioning unchanged.
  - [ ] 4.3 Update the Rule 21 header (PosterCard.tsx:1-2) to `// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)` + `// Design ref: ux-design.pen Screen PC-1 (XlFIq) — bugfix-10-7 info-density & polish` (keep the `// Source: ux-design.pen (Pencil app)` line). [AC #6]
  - [ ] 4.4 (AC4 — selection checkbox) NO change. Confirm PosterCard.tsx:138-150 is untouched.

- [ ] **Task 5 — PosterCard tests** (AC: #5)
  - [ ] 5.1 Extend `apps/web/src/components/media/PosterCard.spec.tsx`: (a) default-state movie ⇒ metadata `<p>` shows `{year}` only; (b) after `mouseEnter` + 200 ms + resolved `useMovieDetails` mock (`{ data: { runtime: 139 } }`) ⇒ `2022 · 2 小時 19 分`; (c) tv variant ⇒ `2022 · 4 季 34 集`; (d) UUID `id` ⇒ stays year-only after hover (no fetch); (e) AC3 — `container.querySelector('svg')` not null, `queryByText(/⭐/)` is null, `getByText('8.4')` present. Optional (f) AC2 className check. Mock the detail hooks via `vi.mock(..., () => ({ useMovieDetails: () => (… as Partial<ReturnType<typeof useMovieDetails>>), useTVShowDetails: () => (…) }))` — **no `as any`**. Rule 16 matchers throughout.
  - [ ] 5.2 `pnpm nx test web` PASS — confirm the count = post-bugfix-10-6 baseline (run a fresh `pnpm nx test web`; bugfix-10-6 closeout reported **1790**) + the new `formatMedia.spec.ts` cases + the new `PosterCard.spec.tsx` assertions; no removals.

- [ ] **Task 6 — Regression gates + closeout** (AC: #8)
  - [ ] 6.1 `pnpm nx test web` PASS. `pnpm nx test api` PASS (no Go changes; if the SSE-broadcast-scan-cancelled test flakes on the full run, retry once + reference the existing `preexisting-fail-scanner-sse-scan-cancelled-flake` entry — no new entry).
  - [ ] 6.2 `pnpm lint:all` → **0 errors / 122 warnings** (matches bugfix-10-6 closeout — ZERO new). `pnpm exec prettier --write` on all touched files first, then `prettier --check .` clean.
  - [ ] 6.3 `pnpm run test:cleanup` → no orphaned vitest workers.
  - [ ] 6.4 Rule 15 self-check (record in Completion Notes): grep `tmdbHandler.RegisterRoutes` in `apps/api/cmd/api/main.go` + `GetMovieDetails`/`GetTVShowDetails` in `apps/api/internal/handlers/tmdb_handler.go` — confirm both TMDb-detail routes are already wired (Pre-flight #6); ZERO backend edits in this story.
  - [ ] 6.5 `git status` shows only the expected files: `apps/web/src/components/media/PosterCard.tsx`, `apps/web/src/lib/formatMedia.ts` (new), `apps/web/src/lib/formatMedia.spec.ts` (new), `apps/web/src/components/media/PosterCard.spec.tsx`, `_bmad-output/implementation-artifacts/sprint-status.yaml`, this story file. **No `.pen` changes, no `_bmad-output/screenshots/` changes, `scripts/export-pen-screenshots.py` not run.** (`.claude/github-star-reminder.txt` may already be dirty at session start — leave it.)
  - [ ] 6.6 Manual browser smoke substituted by AC #5 deterministic vitest assertions (CLI agent can't drive Chrome DevTools — bugfix-10-2/10-5/10-6 precedent). Browser-pixel verification of the hover-fade timing / scale-95 kinetics / star fill at 390 & 1440 recommended on NAS deploy. Bonus: `pnpm exec tsc -p apps/web/tsconfig.app.json --noEmit` — ~17 pre-existing errors, none should be in `PosterCard.tsx` / `formatMedia.ts` / the touched spec files ⇒ zero new tsc errors.

## Dev Notes

### Pre-flight confirmed by SM (sanity-check before edits — but still open the files to confirm line numbers haven't drifted; bugfix-10-5 CR lesson)

1. **`PosterCard.tsx` current shape** (`apps/web/src/components/media/PosterCard.tsx`, ~234 lines):
   - L1-2: Rule 21 header `// Implements: Component/PosterCardHover (MQbvp)` + `// Source: ux-design.pen (Pencil app)`. (Task 4.3 updates this.)
   - L4-9: imports — `useState` from `react`; `Link` from `@tanstack/react-router`; `import { MoreHorizontal, Check, Play } from 'lucide-react'` (L6 — Task 4.1 appends `Star`); `cn`, `getImageUrl`/`getImageSrcSet`/`getImageSizes`, `HighlightText`, `AvailabilityBadge`. (Task 2 adds `useRef`, `useEffect` to the `react` import, and `useMovieDetails`/`useTVShowDetails` from `../../hooks/useMediaDetails`.)
   - L12-33: `PosterCardProps` — `id: string`, `type: 'movie' | 'tv'`, `title`, `posterPath`, `releaseDate?`, `voteAverage?`, `metadataSource?`, `isNew?`, `isOwned?`, `isRequested?`, `highlightQuery?`, `onMenuClick?`, `selectable?`, `selected?`, `onSelect?` (+ a few unused: `originalTitle?`, `overview?`, `genreIds?`). **No prop changes needed** — `id` + `type` + `releaseDate` are all we need; the runtime/season-count comes from the lazy fetch keyed on `id`.
   - L52-53: `const [imageLoaded, setImageLoaded] = useState(false)`; `const [imageError, setImageError] = useState(false)`. (Task 2.1 adds `hoverIntent` + `hoverTimerRef` here.)
   - L55: `const year = releaseDate ? new Date(releaseDate).getFullYear() : null;` — keep; Task 2.3-2.4 adds the `tmdbId`/`fetchId`/`movieQ`/`tvQ`/`extra`/`metaLine` derivations right after.
   - L63-69: `handleCardClick` (selection-mode click handler) — unchanged.
   - L71-83: `<Link to="/media/$type/$id" params={{type, id}} data-testid="poster-card" onClick={handleCardClick} className={cn(...)}>` — Task 2.2 adds `onMouseEnter`/`onMouseLeave`.
   - L85-102: the image-wrapper `<div>` with `clipPath: 'inset(0 round 0.5rem)'`, `transition-all duration-300 transform-gpu`, `!selectable && 'lg:group-hover:scale-105 lg:group-hover:shadow-2xl'`, `active:scale-[0.98]`, selection ring. **Unchanged.**
   - L137-150: `{selectable && (<div data-testid="selection-checkbox" className="absolute left-2 top-2 ...">...)}` — **AC4: UNCHANGED.**
   - L152-178: the top-right badge cluster `<div className="absolute right-2 top-2 flex items-center gap-1 transition-opacity duration-300 lg:group-hover:opacity-0">` containing `AvailabilityBadge` (owned/requested), `new-badge`, `metadataSource` badge (L170-174 — child de-conflicted in bugfix-10-4 CR H2; inherits the parent fade), and the `電影`/`影集` type chip. **Task 3.1 edits the wrapper `<div>` className only.**
   - L180-194: the kebab `<button data-testid="poster-menu-button" className="absolute right-2 top-2 z-20 ... opacity-0 transition-opacity duration-300 ... lg:group-hover:opacity-100">`. **Unchanged.**
   - L196-207: the center play overlay `<div data-testid="hover-play-overlay" aria-hidden="true" className="absolute inset-0 z-10 hidden ... opacity-0 transition-opacity duration-300 lg:flex lg:group-hover:opacity-100">`. **Unchanged.**
   - L209-213: the comment block about MQbvp's bottom-left title overlay being omitted — DEV may update the trailing line (`...deferred to feature-X-postercard-info-density`) to point at this story (`bugfix-10-7`) since this story IS that follow-up; minor, optional.
   - L215-222: the rating badge `{voteAverage !== undefined && voteAverage > 0 && (<div className="absolute bottom-2 right-2 z-20"><span className="flex items-center gap-1 rounded bg-black/70 px-2 py-0.5 text-xs text-[var(--warning)]">⭐ {voteAverage.toFixed(1)}</span></div>)}` — **Task 4.2 swaps the `⭐ ` for `<Star ... />`.**
   - L225-231: `<div className="mt-2"><h3 className="truncate text-sm font-medium text-white"><HighlightText text={title} query={highlightQuery} /></h3>{year && <p className="text-xs text-[var(--text-secondary)]">{year}</p>}</div>` — **Task 2.4 changes the `<p>` to `{metaLine && <p className="truncate text-xs text-[var(--text-secondary)]">{metaLine}</p>}`.**

2. **The existing detail hooks** (`apps/web/src/hooks/useMediaDetails.ts`) — REUSE these, don't reinvent:
   - `detailKeys = { all: ['details'], movie: (id:number) => [...all,'movie',id], tv: (id:number) => [...all,'tv',id], movieCredits, tvCredits, localMovie, localSeries }`.
   - `useMovieDetails(id: number)` → `useQuery<MovieDetails>({ queryKey: detailKeys.movie(id), queryFn: () => tmdbService.getMovieDetails(id), staleTime: 10*60*1000, enabled: id > 0 })`.
   - `useTVShowDetails(id: number)` → `useQuery<TVShowDetails>({ queryKey: detailKeys.tv(id), queryFn: () => tmdbService.getTVShowDetails(id), staleTime: 10*60*1000, enabled: id > 0 })`.
   - The detail page (`media/$type.$id.tsx` / its detail-view component, bugfix-10-1) already calls these for TMDb-id items ⇒ **shared cache** — a card the user hovered, then clicked through to detail, doesn't re-fetch. Passing `id = 0` ⇒ `enabled: 0 > 0` = `false` ⇒ the query is created-but-idle (`['details','movie',0]` is never executed, no cache pollution). That's the gating mechanism — no edit to `useMediaDetails.ts`.

3. **The TMDb service** (`apps/web/src/services/tmdb.ts`) — `tmdbService.getMovieDetails(movieId: number): Promise<MovieDetails>` → `fetchApi<MovieDetails>('/tmdb/movies/${movieId}')`; `getTVShowDetails(tvId: number): Promise<TVShowDetails>` → `fetchApi<TVShowDetails>('/tmdb/tv/${tvId}')`. `fetchApi` already does `snakeToCamel` on the response (Rule 18). `MovieDetails` / `TVShowDetails` types are in `apps/web/src/types/tmdb.ts`; `MovieDetails.runtime?: number`, `TVShowDetails.numberOfSeasons?: number`, `TVShowDetails.numberOfEpisodes?: number` (confirmed against `libs/shared-types` shapes — the API returns `runtime` / `number_of_seasons` / `number_of_episodes`).

4. **PosterCard consumers** (who renders it, what `id` they pass):
   - `apps/web/src/components/homepage/ExploreBlock.tsx` — TMDb discover/trending items ⇒ **numeric `id`** ✓ (gets the metadata line on hover).
   - `apps/web/src/components/media/MediaGrid.tsx` — TMDb search results ⇒ **numeric `id`** ✓.
   - `apps/web/src/components/library/RecentlyAdded.tsx`, `apps/web/src/components/library/LibraryGrid.tsx` — owned-library items ⇒ **UUID `id`** ⇒ `tmdbId = 0` ⇒ no fetch ⇒ year-only (unchanged). **Out of scope to enrich these from `useLocalMovieDetails`/`useLocalSeriesDetails` — keep the diff tight.**
   - `apps/web/src/routes/media/$type.$id.tsx` — if it renders PosterCards (e.g. related items), same numeric-vs-UUID rule applies; no special handling needed.
   - **Do NOT change any of these consumer files** — the change is entirely inside `PosterCard.tsx` (+ the new `formatMedia.ts`).

5. **Test baselines** (post-bugfix-10-6 closeout): `pnpm nx test web` ≈ **1790** tests PASS; `pnpm lint:all` = **0 errors / 122 warnings**. Confirm the actual current count with a fresh `pnpm nx test web` before claiming a delta. Any deviation beyond the new test count from AC #5 (and any byte-for-byte lint baseline drift) is a regression you introduced — fix it, don't paper over it.

6. **Backend Rule 15 — VERIFIED, no scope expansion needed**:
   - `GET /api/v1/tmdb/movies/:id` → `tmdbHandler.GetMovieDetails` — `apps/api/internal/handlers/tmdb_handler.go:118` (`// GetMovieDetails handles GET /api/v1/tmdb/movies/:id`).
   - `GET /api/v1/tmdb/tv/:id` → `tmdbHandler.GetTVShowDetails` — `apps/api/internal/handlers/tmdb_handler.go:157` (`// GetTVShowDetails handles GET /api/v1/tmdb/tv/:id`).
   - Both registered: `tmdbHandler.RegisterRoutes(apiV1)` — `apps/api/cmd/api/main.go:526`. `tmdbHandler := handlers.NewTMDbHandler(tmdbService)` at `main.go:452`.
   - **Routes are wired. DEV does NOT add any backend route, service method, migration, or swagger annotation. If a fresh grep contradicts this, expand scope (new task + AC) before continuing — do not silently add.** (This is exactly the bugfix-10-4 Rule-15 lesson — a client method existing ≠ the server route wired; here both the client method `tmdbService.getMovieDetails`/`getTVShowDetails` AND the server routes exist and are wired, confirmed.)

7. **No backend changes. No `.pen` changes. No new files except `formatMedia.ts` + `formatMedia.spec.ts`.** Everything else is edits to `PosterCard.tsx` + `PosterCard.spec.tsx`.

### Sally's design contract (Screen PC-1)

Committed `58ba519` (2026-05-11). Screen PC-1 "PosterCard Info-Density & Polish (bugfix-10-7)" in `ux-design.pen` — frame `XlFIq` (at `x:2670 y:20949`, directly below HP-5 / bugfix-10-6), screenshot `_bmad-output/screenshots/flow-b-hover-detail-desktop/pc1-postercard-info-density-polish-bugfix-10-7.png`. Sections:

| PC-1 section | Node | AC | What it shows |
|---|---|---|---|
| A — info-density | `jWJgb` (`secA`) | #1 | Before mini-card (`媽的多重宇宙` / `2022`) vs After mini-cards (movie `媽的多重宇宙` / `2022 · 2 小時 19 分`; series `怪奇物語` / `2016 · 4 季 34 集`) + a 5-step spec box (hover-intent ~200 ms; `useQuery` on the detail endpoint, `enabled:hoverIntent`, `staleTime:24h`; the zh-TW format rules; the `{year} · {extra}` composition rules; `<HighlightText>` only on title) |
| B — hover scale | `jn1If` (`secB`) | #2 | Spec only (it's a transition — no static state): badge-cluster `transition-opacity` → `transition-all` + `lg:group-hover:scale-95` + `origin-top-right`, `duration-300` synced with image `scale-105` |
| C — rating star | `R9PWRe` (`secC`) | #3 | Before badge chip (`⭐ 8.4` emoji) vs After badge chip (lucide `<Star fill-warning>` + `8.4`) + spec (`import { Star } from 'lucide-react'`; `<Star className='h-3 w-3 fill-[var(--warning)] text-[var(--warning)]' aria-hidden />`; container classes unchanged) |
| D — checkbox decision | `o91Cq` (`secD`) | #4 | Decision record: keep mode-gated (Netflix hover-reveal rejected — clutter, click-ambiguity, no-hover-on-touch); zero code |
| footer | `BZrw4` | — | SM/DEV handoff notes (single FE story / Rule 21 header update / Rule 22 retro / Rule 16 helper tests / out-of-scope list) |

`RusTY` (`Component/PosterCard`) and `MQbvp` (`Component/PosterCardHover`) were intentionally **not** edited — the fetched metadata line is a *loaded-state* enhancement, not the default state, and PC-1 is the contract for this story. DEV verifies code-vs-PC-1 at dev-story Step 9 (structural comparison table — browser-pixel deferred to NAS deploy per the bugfix-10-2/10-5/10-6 CLI precedent).

### Cross-cutting Rule compliance checklist

- **Rule 4 (Layered Architecture)**: N/A — pure UI.
- **Rule 5 (TanStack Query for server state)**: ✅ — the runtime/season-count is server state, fetched via the existing TanStack-Query hooks (`useMovieDetails`/`useTVShowDetails`), cache shared with the detail page via `detailKeys`. `hoverIntent` is *UI* state (`useState`) — that's allowed (TanStack Query is for server state only).
- **Rule 6 (Naming)**: ✅ — components stay `PascalCase.tsx`, the new util `formatMedia.ts`/`formatRuntime`/`formatSeriesCount`/`formatPosterMeta` camelCase, `hoverIntent`/`hoverTimerRef`/`tmdbId`/`fetchId` camelCase.
- **Rule 9 (Test co-location)**: ✅ — `formatMedia.spec.ts` next to `formatMedia.ts`; `PosterCard.spec.tsx` already next to `PosterCard.tsx`.
- **Rule 11 (Interface Location)**: N/A — no new interfaces.
- **Rule 12 (lint:all baseline)**: ✅ Task 6.2 enforces 0/122.
- **Rule 13 (Error Handling Completeness)**: N/A — `useQuery` handles its own error/loading states; the metadata line just doesn't render when `data` is undefined/errored (graceful degradation to year-only).
- **Rule 14 (Resource Lifecycle)**: ✅ — the `setTimeout` hover-intent timer is cleared on `onMouseLeave` AND on unmount (`useEffect` cleanup with `[]` deps). No `ResizeObserver`/listeners added.
- **Rule 15 (Pre-commit self-verification — HTTP Route ↔ Client Method Sync)**: ✅ — Pre-flight #6 + Task 6.4: the TMDb-detail routes (`GET /api/v1/tmdb/movies/:id`, `GET /api/v1/tmdb/tv/:id`) AND the client methods (`tmdbService.getMovieDetails`/`getTVShowDetails`) both exist and are wired (`tmdbHandler.RegisterRoutes(apiV1)`, `main.go:526`). No new API surface in this story. (Wiring/DB/Swagger N/A — no backend change.)
- **Rule 16 (Test Assertion Quality)**: ✅ AC #5 mandates `toBe`/`toBeInTheDocument`/`toBeNull`/`findByText` — never `toBeTruthy`/`toBeVisible` on hover state.
- **Rule 18 (API Boundary Case Transformation)**: ✅ — `tmdbService.getMovieDetails`/`getTVShowDetails` already `snakeToCamel` the response; no new service code, so no new boundary-transform to add.
- **Rule 19 (Package Boundaries)**: N/A — pure frontend.
- **Rule 20 (AC Contract Versioning)**: ✅ — AC #2, #3 stamped `[@contract-v1]` (forward-only `v0→v1` retrofit; Change Log carries the rows). DEV records `confirmed against [@contract-v1] (bugfix-10-4 AC #1)` in Completion Notes (the hover-overlay layout contract is preserved; AC2/AC3 are animation/glyph tweaks within it, no v2 bump). No downstream consumer of the new stamps yet.
- **Rule 21 (Component-to-Design Node Traceability)**: ✅ — AC #6 / Task 4.3: header → `// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)` + `// Design ref: ux-design.pen Screen PC-1 (XlFIq)`. PC-1's sub-sections are *screen frames* (not Reusable Components), so a strict `// Implements: Component/X (id)` for them N/A — the screen-ref comment is the right form.
- **Rule 22 (Epic Retro Design-Drift Audit)**: deferred to the Epic 10 retro (fires at retro time, not story closeout). PosterCard has a **hover-state change** (AC2) ⇒ it's a **mandatory-include** in that retro's drift sample (per Rule 22's sample-pick policy: "always include any component with hover/focus state changes").

### Previous story intelligence

- **bugfix-10-6 (ExploreBlock polish bundle)** — just completed (DEV+TEA+CR). Same shape as this story (a small UX-pass-then-bundle). Lessons carried forward: (1) the **CR M1/M2 pattern** — keep the File List truthful, list *every* file touched (including any TEA-pass files if a TEA `/testarch-automate` runs after dev-story). (2) bugfix-10-6 CR added `duration-300` to ExploreBlock's hover overlay "to match PosterCard's hover overlay" — so PosterCard's `duration-300` is the house standard; AC2 keeps it. (3) bugfix-10-6's `<option>` emoji-strip had thin coverage (CR L3) — for *this* story, the AC3 `⭐`→`<Star>` swap IS covered by the new PosterCard.spec.tsx assertion (AC #5e).
- **bugfix-10-4 (hover-preview-viewport-flip)** — the origin of these 4 deferred items. Key lessons: (a) the **`group-hover` opacity cascade trap** (CR H2): `PosterCard.tsx:171` (the `metadataSource` child badge) had `opacity-0 lg:group-hover:opacity-100` nested under the parent's `lg:group-hover:opacity-0` ⇒ effective `0` in both states; the fix made the child inherit the parent fade. **AC2 must not regress this** — when you add `transition-all`/`scale-95` to the parent, don't re-introduce child-level `opacity-*` gating. (b) The **Chromium GPU `clipPath` workaround** (PosterCard.tsx:86-89) — leave it; don't touch the image-wrapper. (c) bugfix-10-4 AC #1 is `[@contract-v1]` for the hover-overlay layout (center play + kebab top-RIGHT + rating bottom-RIGHT + top-right badge cluster) — this story preserves those positions; ACK it in Dev Notes. (d) bugfix-10-4 CR H1 was File-List-lying — don't.
- **bugfix-10-1 (PosterCard TMDb-ID 404)** — established the TMDb-id-vs-UUID distinction on the media-detail route, and the PosterCard.spec.tsx router-mock setup (the `QueryClientProvider` + mocked `<Link>` you'll reuse for the new tests). The `useMovieDetails`/`useTVShowDetails` "progressive enhancement" hooks (used by the TMDb-backed detail view) are the ones this story reuses.
- **bugfix-10-2 (qbt-downloads-http-status)** — established the `Partial<ReturnType<typeof useX>>` typed-mock pattern (vs `as any`) — AC #5 requires it for mocking the detail hooks.
- **bugfix-10-3 (skeleton-flicker)** — `## 🧪 Known dev-mode artifacts` in `project-context.md`: React 18 `<StrictMode>` double-mounts components in dev (`apps/web/src/main.tsx:11`) — so `useEffect`-driven side effects (like the hover-intent timer cleanup) get exercised twice in dev. Your `useEffect(() => () => clearTimeout(ref.current), [])` cleanup must be idempotent (it is — `clearTimeout(null)` is a no-op). For any manual smoke, `pnpm nx run web:preview` (prod build, StrictMode stripped) is the truthful target — but the unit tests are the gate here.

### Project Structure Notes

**Modified (production):**
- `apps/web/src/components/media/PosterCard.tsx` — Rule 21 header update; `useRef`/`useEffect`/`useMovieDetails`/`useTVShowDetails` imports + `Star` from lucide; hover-intent state + `onMouseEnter`/`onMouseLeave` on the `<Link>`; `tmdbId`/`fetchId`/`movieQ`/`tvQ`/`extra`/`metaLine` derivations; the `mt-2` `<p>` → composed metadata line (AC1); badge-cluster `<div>` className → `transition-all` + `origin-top-right` + `lg:group-hover:scale-95` (AC2); rating `<span>` `⭐` → `<Star>` (AC3). Selection checkbox (AC4) untouched.

**Created (production):**
- `apps/web/src/lib/formatMedia.ts` — `formatRuntime` / `formatSeriesCount` / `formatPosterMeta` pure helpers (AC1). (Or DEV may append to `apps/web/src/lib/timeFormat.ts` instead — then no new file, and `formatMedia.spec.ts` becomes additions to `timeFormat.spec.ts`/a new spec; DEV's call.)

**Created (tests):**
- `apps/web/src/lib/formatMedia.spec.ts` — boundary-case tests for the 3 helpers (AC5).

**Modified (tests):**
- `apps/web/src/components/media/PosterCard.spec.tsx` — default-state year-only; after-hover movie line; after-hover tv line; UUID-id no-fetch; AC3 lucide-svg / no-`⭐`; (optional) AC2 className check (AC5).

**Modified (process / tracking):**
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `bugfix-10-7-postercard-info-density-and-polish`: `backlog` → `ready-for-dev` (this SM pass) → `in-progress` → `review` (DEV) → `done` (CR).
- `_bmad-output/implementation-artifacts/bugfix-10-7-postercard-info-density-and-polish.md` — this story file (DEV record, Change Log rows, CR follow-ups).

**Untouched (DO NOT TOUCH — out of scope):**
- All backend code (`apps/api/**`), all migrations, swagger — Rule 15 verified, TMDb-detail routes already wired.
- `ux-design.pen` (Sally's `58ba519` / Screen PC-1 `XlFIq` is the locked contract). `_bmad-output/screenshots/**`. `scripts/export-pen-screenshots.py` (not run).
- `apps/web/src/hooks/useMediaDetails.ts` — REUSE as-is (`enabled: id > 0` is the gating mechanism; pass `id=0` until hover). Only touch it if DEV chooses the optional `usePosterMeta` wrapper, and even then existing callers must not change behaviour.
- `apps/web/src/services/tmdb.ts` — `getMovieDetails`/`getTVShowDetails` already there; no change.
- `apps/web/src/components/media/MediaDetailPanel.tsx` (`:122` shows `{runtime} 分鐘` — the detail panel's own format; `MediaDetailPanel.spec.tsx:166` asserts `'120 分鐘'`; **changing it would break that test and is out of scope** — the poster card's `2 小時` vs the detail panel's `120 分鐘` is an intentional divergence: the card is space-constrained, the panel isn't).
- `PosterCardMenu.tsx`, search-result cards, `RecentMediaPanel.tsx`, notification toasts — other `⭐`/emoji contexts; not in this story (filed loosely under the "full emoji sweep" idea in the sprint-status note).
- `RecentlyAdded.tsx` / `LibraryGrid.tsx` / `MediaGrid.tsx` / `ExploreBlock.tsx` / `media/$type.$id.tsx` — PosterCard *consumers*; the change is entirely inside `PosterCard.tsx`, none of these need editing.
- `PosterCardSkeleton.tsx` — loading skeleton, unrelated.

### References

- Sally's UX pass + design contract: `_bmad-output/implementation-artifacts/sprint-status.yaml` (bugfix-10-7 entry, post-commit `58ba519`); `ux-design.pen` Screen PC-1 (`XlFIq`); screenshot `_bmad-output/screenshots/flow-b-hover-detail-desktop/pc1-postercard-info-density-polish-bugfix-10-7.png`. Commit `58ba519` `feat(ux): bugfix-10-7 — PosterCard info-density & polish design pass (PC-1 spec screen)`.
- Code under change: `apps/web/src/components/media/PosterCard.tsx:1-234` (header L1-2, imports L4-9, state L52-53, year derivation L55, `<Link>` L71-83, badge cluster L152-178, rating L215-222, metadata `<p>` L225-231, selection checkbox L137-150 unchanged).
- Reused infra: `apps/web/src/hooks/useMediaDetails.ts:7-71` (`detailKeys`, `useMovieDetails`, `useTVShowDetails`); `apps/web/src/services/tmdb.ts:57-70` (`getMovieDetails`, `getTVShowDetails`); `apps/web/src/types/tmdb.ts` (`MovieDetails.runtime`, `TVShowDetails.numberOfSeasons`/`numberOfEpisodes`).
- Backend route confirmation (Rule 15): `apps/api/internal/handlers/tmdb_handler.go:118` (`GET /api/v1/tmdb/movies/:id` → `GetMovieDetails`), `:157` (`GET /api/v1/tmdb/tv/:id` → `GetTVShowDetails`); `apps/api/cmd/api/main.go:452` (`tmdbHandler := handlers.NewTMDbHandler(tmdbService)`), `:526` (`tmdbHandler.RegisterRoutes(apiV1)`).
- `group-hover` cascade trap: `apps/web/src/components/media/PosterCard.tsx` (bugfix-10-4 CR H2 fix on `:171`); story `bugfix-10-4-hover-preview-viewport-flip.md` AC #1 `[@contract-v1]`.
- Typed-mock pattern: bugfix-10-2 CR M3 — `Partial<ReturnType<typeof useX>>` (also used in bugfix-10-6's specs).
- Existing format idioms: `apps/web/src/lib/timeFormat.ts` (relative time `${n} 分鐘前` / `${n} 小時前`); `apps/web/src/components/media/MediaDetailPanel.tsx:122` (`{runtime} 分鐘` — detail panel, out of scope).
- Project rules: `project-context.md` Rule 5, Rule 9, Rule 14, Rule 15, Rule 16, Rule 18, Rule 20, Rule 21, Rule 22; `## 🧪 Known dev-mode artifacts` (StrictMode double-mount); §4 caching (TMDb 24h backend; the FE hooks use a 10-min staleTime — reuse as-is).
- Test/lint baselines: bugfix-10-6 closeout — `nx test web` ≈ 1790 tests, `lint:all` 0 errors / 122 warnings (`sprint-status.yaml` bugfix-10-6 entry).

## Change Log

| Date | Change |
|---|---|
| 2026-05-11 | [@contract-v0→v1] AC #2 (new): PosterCard top-right badge cluster fade-out adds `lg:group-hover:scale-95` + `origin-top-right` + `transition-all` (was `transition-opacity` only) — the cluster *recedes* (opacity + scale) as the kebab takes over, instead of just dissolving; `duration-300` synced with the image `scale-105`. What breaks downstream: nothing today (no test asserts the badge-cluster transition classes); future PosterCard redesigns that drop the `scale-95` must bump to `[@contract-v2]` + add a Change Log row. PosterCard.tsx was implicit `v0` for this region (no prior stamp) → forward-only retrofit per Rule 20. |
| 2026-05-11 | [@contract-v0→v1] AC #3 (new): PosterCard rating badge uses lucide `<Star className='h-3 w-3 fill-[var(--warning)] text-[var(--warning)]' aria-hidden />` instead of the `⭐` emoji — cross-OS rendering consistency, matching the `.pen` PC-1 / `MQbvp` star treatment. What breaks downstream: nothing today (no test asserts the `⭐` glyph on PosterCard; the new PosterCard.spec.tsx assertion is the going-forward guard); a future "full emoji sweep" that migrates the other `⭐` sites (PosterCardMenu / detail / search / toasts) should reference this as the canonical pattern. PosterCard.tsx implicit `v0` → forward-only retrofit per Rule 20. |
| 2026-05-11 | SM Bob /create-story (YOLO): story file created from Sally's PC-1 design contract (`58ba519`) + the bugfix-10-7 sprint-status note + project-context.md. 8 ACs (2 stamped `[@contract-v1]` — AC #2, #3; AC #4 is a zero-code decision record), 6 tasks all FE (0 BE — Rule 15 verified, TMDb-detail routes already wired), single story (cross-stack split check: BE=0, not met). `bugfix-10-7-postercard-info-density-and-polish`: `backlog` → `ready-for-dev`. → DEV /dev-story next (use a different LLM than this SM session per workflow tip; run /code-review after with a third). |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
