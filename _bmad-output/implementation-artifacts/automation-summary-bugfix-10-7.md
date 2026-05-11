# Automation Summary — bugfix-10-7 (PosterCard Info-Density & Polish)

**Date:** 2026-05-11
**Story:** `bugfix-10-7-postercard-info-density-and-polish` (status: review — DEV /dev-story complete, commit `7e9c3d2`)
**Workflow:** `bmad:bmm:workflows:testarch-automate` (TEA / Murat), BMad-Integrated Mode
**Coverage target:** `critical-paths`
**Mode flags:** `tea_use_playwright_utils: false` (traditional fixture/route-mock patterns) · `tea_use_mcp_enhancements: false` (no auto-heal loop)

---

## Why this pass

The DEV story already shipped strong AC #5 coverage: `apps/web/src/lib/formatMedia.spec.ts` (17 boundary cases — exhaustive for the 3 pure helpers) + `apps/web/src/components/media/PosterCard.spec.tsx` +7 (default year-only, after-hover movie/tv lines, UUID-id no-fetch, no-metadata-line, AC3 lucide-svg/no-⭐/8.4, AC2 className). DEV also added two regression-driven `vi.mock` lines to `MediaGrid.spec.tsx` / `SearchResults.spec.tsx` (PosterCard now uses `useQuery`).

This `automate` pass expands beyond that with the small set of **critical-path gaps** the AC #5 coverage left open — all on the AC #1 hover-intent debounce timing logic (the only non-trivial logic in the story) plus the AC #2 browser-only `scale-95` behaviour:

| Gap | Risk | Level chosen | Why this level |
|-----|------|--------------|----------------|
| Hover-intent **cancel path** — `onMouseLeave` before the ~200 ms debounce fires must `clearTimeout` the pending timer and never fetch. DEV's tests only cover the happy path (`enter → 200 ms → line resolves`). | Med — a refactor of `handleMouseLeave` (drop the `clearTimeout`, or reset `hoverIntent` instead) would silently fire bursts of detail requests on a grid sweep, and every existing test stays green | Component (vitest, fake timers) | Pure debounce timing; fast, deterministic, no browser; the timing-debugging KB pattern (fake timers + `act`) |
| Hover-intent **no-flicker** — once the line has resolved, `onMouseLeave` must NOT reset `hoverIntent` (AC #1 says "do NOT reset `hoverIntent` once it's `true` — keep showing the data, avoid re-fetch flicker"), and a re-enter must be idempotent. **Zero** assertions on this in the DEV story. | Med — a regression that resets `hoverIntent` on leave makes the metadata line flicker year-only ↔ composed on every hover churn | Component (vitest, fake timers) | Same as above — UI state transition, no browser needed |
| Hover-intent **Rule 14 cleanup** — unmount with a pending hover-intent timer must `clearTimeout` it (`useEffect(() => () => clearTimeout(...), [])`). Not explicitly asserted; only implicitly exercised by RTL's per-test auto-cleanup. | Low–Med — a missing/broken cleanup leaks a timer that fires `setHoverIntent` after unmount (project-context.md Rule 14 — Resource Lifecycle) | Component (vitest, `setTimeout`/`clearTimeout` spies) | White-box but the only way to deterministically prove the cleanup ran |
| AC #2 — the **browser-rendered** `scale-95` transform on hover. The DEV unit test asserts the *className* (`lg:group-hover:scale-95`); only a real browser can prove CSS `:hover` actually drives the transform. Story DoD explicitly deferred browser-pixel verification to NAS deploy. | Low–Med — a regression that drops `scale-95` (or `transition-all` reverts to `transition-opacity`) passes the className test if it's incomplete; the kinetic "recede" is a `[@contract-v1]` invariant | E2E (Playwright, route-mocked) | Browser-layer regression confidence without flaky pixel/timing assertions; `poster-card-hover.spec.ts` already runs in CI/nightly; reuses the exact `badgeCluster = ownedBadge.locator('xpath=..')` anchor from the sibling `[P0]` opacity test |

**Deliberately NOT added** (per `test-quality` / `risk-governance` / `test-priorities-matrix` KB — depth scales with impact, avoid duplicate coverage & flaky patterns):

- More `formatMedia.ts` boundary tests — the DEV `formatMedia.spec.ts` already covers `0`/`null`/`undefined`/negative/`47`/`59`/`60`/`120`/`125`/`139` for `formatRuntime`, `(0,…)`/`(undefined,…)`/`(null,…)`/`(1,undefined)`/`(1,0)`/`(4,34)` for `formatSeriesCount`, and all 4 branches of `formatPosterMeta`. Adding e.g. `(1,1)` or `formatRuntime(1)` is marginal duplication of already-exhaustive boundary coverage.
- A **browser** test for AC #3 (lucide `<Star>` not `⭐`) — unlike bugfix-10-6's `<option>` no-emoji case (which had *zero* coverage and `<option>` content is awkward to assert), the DEV story already has a vitest test (`svg in .absolute.bottom-2.right-2` + `queryByText(/⭐/)` null + `getByText('8.4')`). lucide rendering an inline `<svg>` is well-established; a browser duplicate adds no logic dimension.
- The **hover-fade timing/easing** (`duration-300` / `ease-out` over 300 ms) as a precise E2E assertion — that's the exact flaky-timing anti-pattern the KB warns against; `toHaveCSS` polls until the transform *settles* at `matrix(0.95,…)`, which is the right deterministic level. Browser-pixel timing verification stays deferred to NAS deploy per the story DoD.
- New fixtures/factories — the existing patterns are the established idiom and need no change: vitest `vi.mock('../../hooks/useMediaDetails')` + the typed-mock helpers `movieResult()`/`tvResult()` (double-cast through `Partial<ReturnType<typeof useX>>`, zero `as any` — bugfix-10-2 CR M3); Playwright `tests/support/fixtures` route mocks (`stubHomepageBaseline`, `stubExploreBlocksWith`, `jsonOk`).
- Re-testing AC #1/#2/#3 *logic* at E2E level — already covered at the unit level; E2E adds only the browser-render dimension (the `scale-95` transform), not logic variations. Avoiding duplicate coverage.

---

## Tests Created

### Component Tests (vitest + RTL) — P2 — 3 new

All in `apps/web/src/components/media/PosterCard.spec.tsx` under a new describe block `Hover-intent debounce edge cases (bugfix-10-7 AC #1 — TEA regression guards)`. All use `vi.useFakeTimers()` + `act(() => vi.advanceTimersByTime(...))` (deterministic, no hard waits), restore real timers at the end, and reuse the existing `mockUseMovieDetails.mockImplementation((id) => movieResult(id > 0 ? 139 : undefined))` typed-mock pattern that simulates the hooks' built-in `enabled: id > 0` gating.

- **[P2] `mouseLeave before the ~200 ms debounce fires ⇒ timer cancelled, no detail fetch, line stays year-only`** — `mouseEnter`, advance 100 ms (not yet), `mouseLeave` (cancels), advance 300 ms (well past). Asserts: `getByText('2022')` present, `queryByText(/小時/)` absent, and `mockUseMovieDetails.mock.calls.every(([id]) => id === 0)` is `true` (the hook was never asked for a real id ⇒ no network call would have happened).
- **[P2] `once the line has resolved, mouseLeave then re-enter does NOT flicker back to year-only`** — `mouseEnter` → advance 200 ms → asserts `getByText('2022 · 2 小時 19 分')`. Then `mouseLeave` → asserts the composed line is *still* there AND `queryByText('2022')` (bare year, exact) is `null` (the line stuck, didn't revert). Then `mouseEnter` again → advance 200 ms → asserts the composed line is still there (re-entry is idempotent: re-arms the now-null timer, `setHoverIntent(true)` is a React no-op when already `true`).
- **[P2] `unmount with a pending hover-intent timer clears it (Rule 14 — no leaked timer)`** — spies on `globalThis.setTimeout` / `clearTimeout`; `mouseEnter` arms the 200 ms timer (the only `setTimeout` call triggered by `mouseEnter`, no re-render ⇒ last spied call is ours); captures that timer id; `unmount()`; asserts `clearTimeout` was called *with that exact id* (the `useEffect` cleanup ran). Restores the spies + real timers.

### E2E Tests (Playwright, route-mocked) — P2 — 1 new

- `tests/e2e/poster-card-hover.spec.ts` — **[P2] `hover at lg: viewport shrinks the top-right badge cluster (scale-95) as it fades — bugfix-10-7 AC #2 kinetic recede`** (added inside the existing `PosterCard Hover @ui @poster-card @bugfix-10-4` describe, directly after the sibling `[P0] … fades top-right badge cluster (opacity 1 → 0)` test). Same fixture: `stubHomepageBaseline(page)` + `stubExploreBlocksWith(page, movieContent, [603])` (movie 603 owned so `AvailabilityBadge` anchors the cluster). Same anchor: `badgeCluster = ownedBadge.locator('xpath=..')`. BEFORE hover: `toHaveCSS('transform', 'none')`. AFTER `card.hover()`: `toHaveCSS('transform', 'matrix(0.95, 0, 0, 0.95, 0, 0)')` — `toHaveCSS` polls up to `expect.timeout` (10 s per the file's design notes) so the 300 ms `transition-all` settles. Proves the CSS `:hover` actually drives the `lg:group-hover:scale-95` transform, not just that the class string is present.

## Infrastructure Created

None — reused existing patterns:
- vitest: `vi.mock('../../hooks/useMediaDetails')` + the `movieResult()`/`tvResult()` typed-mock helpers (added by the DEV story; zero `as any`), `vi.useFakeTimers()` + `act()`.
- Playwright: `tests/support/fixtures` (`stubHomepageBaseline`, `stubExploreBlocksWith`, `jsonOk`), the existing `defaultBlocks` / `movieContent` fixtures, and the `ownedBadge.locator('xpath=..')` cluster anchor from the bugfix-10-4 opacity test.

---

## Validation Results

| Suite | Command | Result |
|-------|---------|--------|
| Targeted vitest | `pnpm exec vitest run apps/web/src/components/media/PosterCard.spec.tsx` | **61/61 PASS** (1 file) — 58 (post-DEV) + 3 new |
| Full web unit suite | `pnpm exec vitest run` (= `pnpm nx test web`) | **1817/1817 PASS** (147 files) — baseline 1814 (post-DEV: 1790 + 17 formatMedia + 7 PosterCard) + the 3 new vitest tests; no removals. `cleanup-test-processes.sh --all` ran clean. |
| New E2E discoverability | `pnpm exec playwright test tests/e2e/poster-card-hover.spec.ts --list` | New `[P2] … scale-95` test resolves at `poster-card-hover.spec.ts:251:7` across all 5 projects (chromium/firefox/mobile-chrome/mobile-safari/…); no syntax/import errors. **Browser E2E run deferred** per the story's own DoD (`poster-card-hover.spec.ts` runs in CI/nightly; a CLI agent can't drive Chrome for this pass). |
| ESLint (touched files) | `pnpm exec eslint apps/web/src/components/media/PosterCard.spec.tsx tests/e2e/poster-card-hover.spec.ts` | **0 errors, 0 warnings.** No new `as any`. |
| `lint:all` (full workspace) | `pnpm lint:all` | exit 0 — `nx run-many -t lint` + `eslint .` + `prettier --check .` all pass; eslint 0 errors / 122 warnings (the established baseline; this pass adds 0 new warnings). |
| Prettier | `pnpm exec prettier --check apps/web/src/components/media/PosterCard.spec.tsx tests/e2e/poster-card-hover.spec.ts` | Clean — "All matched files use Prettier code style!" |
| Orphan check | `pnpm run test:cleanup` | "No test processes found" |

**No production code changed. No `.pen` changes. No new files except this summary.** `git status` (excluding the pre-existing `.claude/github-star-reminder.txt`): `apps/web/src/components/media/PosterCard.spec.tsx`, `tests/e2e/poster-card-hover.spec.ts` (+ this summary + the story-file note + the sprint-status note).

---

## Coverage Status

- ✅ AC #1 (info-density / lazy-on-hover): happy path (DEV) + **cancel path / no-flicker / Rule-14 cleanup edge cases (this pass)** + the pure-helper boundary matrix (DEV) — comprehensive.
- ✅ AC #2 (`scale-95` recede): className (DEV unit) + **browser-rendered transform on hover (this pass E2E)**.
- ✅ AC #3 (lucide `<Star>`): svg-in-rating-chip + no-`⭐` + `8.4` (DEV unit) — adequate; browser duplicate deliberately skipped.
- ✅ AC #4 (selection checkbox stays mode-gated): zero-code decision record; the existing `Selection Mode (Story 5-7)` describe in `PosterCard.spec.tsx` already guards `selectable`-gating.
- ⚠️ Deferred to NAS deploy / CI-nightly (per the story DoD, not this pass): browser-pixel verification of the hover-fade *timing/easing* at 390 px & 1440 px, and the star *fill colour* render. The E2E `scale-95` transform check is the deterministic regression guard; pixel-exact is out of scope (would be flaky).

## Definition of Done

- [x] Execution mode determined — BMad-Integrated (story `bugfix-10-7-postercard-info-density-and-polish`)
- [x] Framework loaded — vitest + RTL (`apps/web`), Playwright (`tests/e2e/`, `playwright.config.ts`)
- [x] Existing coverage analyzed — DEV's AC #5 tests reviewed; gaps identified (cancel path, no-flicker, Rule-14 cleanup, browser `scale-95`)
- [x] KB consulted — `tea-index.csv`; applied `test-priorities-matrix` (P2 = UI-polish edge cases), `test-quality` (deterministic, atomic, no hard waits, length limits), `test-levels-framework` (component vs E2E choice — debounce logic → component, CSS `:hover` transform → E2E), `timing-debugging` (fake timers + `act` for the debounce)
- [x] Test levels selected appropriately; **duplicate coverage avoided** (E2E adds only the browser-render dimension)
- [x] Priorities assigned — all P2 (matches the bugfix-10-6 TEA-pass precedent)
- [x] No new fixtures/factories needed — reused existing patterns
- [x] Component tests: fake timers + `act`, no hard waits, atomic, explicit assertions (`toBeInTheDocument` / `toHaveBeenCalledWith` / `.every(...)` — no `toBeTruthy`/`toBeVisible` on CSS state, Rule 16)
- [x] E2E test: route-mocked (network-first — every `page.route` before `page.goto`), `data-testid` selectors, `toHaveCSS` (polls — no hard wait), Given-When-Then structure
- [x] All tests deterministic; no flaky patterns; test files under size limits
- [x] Quality standards enforced — zero new `as any`, zero new lint warnings, prettier clean
- [x] Full regression suite run (`pnpm nx test web` 1817/1817), E2E discoverability checked, orphan check clean
- [x] Automation summary created (this file)
- [ ] Browser E2E run — **deferred per the story DoD** (`poster-card-hover.spec.ts` runs in CI/nightly; not run in this CLI pass)

## Next Steps

1. (Story owner) Run `/code-review` on bugfix-10-7 with a different LLM than DEV/TEA — this TEA pass's tests are part of what CR reviews.
2. CI/nightly will run the new `[P2] scale-95` E2E test against the deployed app; on NAS deploy, eyeball the hover-fade timing/easing + star fill at 390 px & 1440 px (the bits not deterministically testable).
3. Monitor the burn-in loop for flakiness on the new E2E test (the `toHaveCSS('transform', ...)` poll could be sensitive to transition timing on a slow CI runner — if it flakes, raise `expect.timeout` for that assertion or settle the transition explicitly).

## Knowledge Base References Applied

- `test-priorities-matrix.md` — P0–P3 classification (P2: secondary features / UI polish / config / "can defer edge cases" — fits the hover-intent edge cases + the cosmetic `scale-95` browser guard; matches the bugfix-10-6 TEA-pass tagging).
- `test-levels-framework.md` — Component vs E2E selection: debounce/state-transition logic → component (fast, deterministic, no browser); CSS `:hover`-driven transform → E2E (only a browser can prove it).
- `test-quality.md` — deterministic, isolated, atomic, explicit assertions, no hard waits, file-size limits; restore fake timers; reuse fixtures over new infra.
- `timing-debugging.md` — fake timers + `act(() => vi.advanceTimersByTime(...))` as the deterministic pattern for debounce/timeout logic (vs `waitForTimeout`).
- `network-first.md` — every `page.route()` installed before `page.goto()` in the E2E test (reused from the existing `poster-card-hover.spec.ts` fixtures).
- `risk-governance.md` — "depth scales with impact": bugfix-10-7 is a low-risk polish bundle ⇒ a focused 4-test gap-closing pass, not a broad new suite; explicitly enumerated what was deliberately NOT added.
- `selective-testing.md` — `[P2]` tags in test names enable `--grep '@P2'` selective execution; the E2E test joins the existing `@ui @poster-card @bugfix-10-4` tag set.
