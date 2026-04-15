# Automation Summary — Story 10-2 Hero Banner Carousel

**Date:** 2026-04-15
**Story:** `_bmad-output/implementation-artifacts/10-2-hero-banner-component.md` (status: review)
**Mode:** BMad-Integrated (TA workflow — YOLO)
**Coverage target:** critical-paths (P0 + P1)

---

## Context

Story 10-2 delivered the homepage hero banner carousel + TrailerModal. Development produced 36 unit tests (HeroBanner.spec.tsx, TrailerModal.spec.tsx, useTrending.spec.ts) and 7 Go tests (tmdb_service / tmdb_handler for the new `/videos` routes). **There was zero browser-level or API-contract coverage before this TA run.**

All 6 acceptance criteria now have automated coverage at the appropriate test level; the UI suite runs fully offline via network mocking, and the API contract suite targets the live backend when `TMDB_API_KEY` is configured.

---

## Tests Created

### E2E (UI) — `tests/e2e/hero-banner.spec.ts` (14 tests, 524 lines)

Fully mocked via `page.route()` — no live TMDb calls. Deterministic auto-rotation via `page.clock` (virtual time).

| # | AC | Priority | Test |
|---|----|----------|------|
| 1 | 1 | P0 | display banner with title, year, rating, overview when trending data available |
| 2 | 5 | P0 | hide banner section when trending API returns empty |
| 3 | 5 | P0 | hide banner section when trending API fails |
| 4 | — | P1 | filter out items missing `backdrop_path` (render guard) |
| 5 | 2 | P1 | auto-rotate to next slide after 8s via virtual clock |
| 6 | 2 | P1 | navigate to a specific slide via dot indicator |
| 7 | 3 | P1 | navigate to media detail page when 查看詳情 link is clicked |
| 8 | 4 | P1 | render banner with mobile-sized backdrop and truncated overview (375×812) |
| 9 | 6 | P0 | open trailer modal with YouTube iframe on play button click |
| 10 | 6 | P0 | close modal on Escape key |
| 11 | 6 | P1 | close modal on backdrop click |
| 12 | 6 | P1 | close modal on close button click |
| 13 | 6 | P1 | show empty state when no embeddable trailer exists |
| 14 | 6 | P1 | show empty state when videos endpoint fails |

### API Contract — `tests/e2e/tmdb-videos.api.spec.ts` (5 tests, 127 lines)

Live-backend smoke (skipped when `TMDB_API_KEY` absent, matching `tmdb-trending-discover.api.spec.ts`).

| # | Priority | Test |
|---|----------|------|
| 1 | P1 | `GET /tmdb/movies/:id/videos` returns `ApiResponse<VideosResponse>` with snake_case fields (Rule 18) |
| 2 | P1 | `GET /tmdb/tv/:id/videos` returns `ApiResponse<VideosResponse>` |
| 3 | P2 | rejects invalid id (0) with 400 |
| 4 | P2 | rejects non-numeric id with 4xx |
| 5 | P2 | returns 404 or empty results for unknown id (envelope still valid) |

---

## Validation Results

**Local run (chromium project):**

```
Running 14 tests using 4 workers
  14 passed (23.2s)
```

All 14 UI tests pass on chromium. Auto-rotation test (`[P1] ... via virtual clock`) completes in 9.9s despite asserting an 8s rotation interval — `page.clock.fastForward(8000)` advances virtual time without real wall-clock wait.

**API spec:** lists cleanly (5 tests). Skips locally because `.env` does not contain `TMDB_API_KEY`; runs in CI where the key is injected.

**Other projects:** tests discoverable on firefox, mobile-chrome, mobile-safari via Playwright's project matrix (56 total test runs when the full matrix executes). Not run locally — CI's 4-shard matrix handles those.

---

## Coverage Matrix (Acceptance Criteria → Tests)

| AC | Description | Unit | E2E | API | Notes |
|----|-------------|------|-----|-----|-------|
| 1 | Banner displays backdrop/title/year/rating/overview | ✅ 14 | ✅ 1 | — | Unit covers props/layout; E2E covers full page mount + data flow |
| 2 | Auto-rotate every 8s with crossfade | ✅ 5 | ✅ 2 | — | E2E uses `page.clock` for deterministic time travel |
| 3 | Click → media detail navigation | ✅ 1 | ✅ 1 | — | E2E exercises TanStack Router navigation + URL assertion |
| 4 | Mobile compact layout | ✅ 2 | ✅ 1 | — | E2E asserts height via `boundingBox()` and `line-clamp-2` class |
| 5 | Graceful hide on empty/error | ✅ 3 | ✅ 2 | — | E2E covers both empty array and 500 error paths |
| 6 | Trailer modal with YouTube embed | ✅ 16 | ✅ 6 | ✅ 2 | Modal open, Escape/backdrop/close-button/empty-state/error |

**Total new tests this run:** 19 (14 E2E + 5 API contract)
**Pre-existing unit tests retained:** 36 (14 + 16 + 6)
**Story-10-2 total coverage:** 55 tests across 3 levels

---

## Design Decisions & Trade-offs

### Why mocked UI tests instead of live backend?

Story 10-2 is a UI feature; the contract it depends on (trending + videos endpoints) is owned by Story 10-1 + the Go tests added in this story. Re-exercising TMDb from the browser buys nothing but flake. The mocked suite asserts **component behaviour** (rotation, navigation, modal lifecycle, graceful degradation). Contract enforcement lives in the API spec + Go test suite.

### Why `page.clock` for auto-rotation?

AC#2 specifies "auto-rotate every 8 seconds". A real `waitForTimeout(8000)` would:
- Add 8s × N-projects × N-shards to CI wallclock;
- Flake on slow workers (not actually deterministic).

`page.clock.install()` before `goto()` + `fastForward(8000)` advances virtual time deterministically in <50ms. This pattern is the knowledge-base recommendation (timing-debugging.md).

### Why separate E2E and API specs?

Journey-level: TestSprite covers production deploy smoke (per project-context.md §2). Feature-level E2E (this suite) should not double-book API contract verification — that belongs in the `.api.spec.ts` siblings.

### Deliberately NOT tested at E2E

- **Pixel-perfect backdrop image rendering** — image URL is asserted, actual TMDb CDN delivery is out of scope.
- **YouTube iframe autoplay** — asserted via `src` attribute match (contains `autoplay=1`); cannot interact with third-party iframe content from Playwright.
- **Hover pause** — covered at unit level (HeroBanner.spec.tsx verifies `isPaused` state logic). Hover-based timing in E2E is flake-prone.
- **Crossfade opacity transition** — per user's memory note (MEMORY.md): do not assert CSS hover transitions with `toBeVisible()`; the active-slide data attribute is the correct hook.

---

## Files

**Created:**
- `tests/e2e/hero-banner.spec.ts` (524 lines, 14 tests)
- `tests/e2e/tmdb-videos.api.spec.ts` (127 lines, 5 tests)
- `_bmad-output/automation-summary-10-2.md` (this file)

**No modifications** to existing test infrastructure — `tests/support/fixtures/index.ts` already provides the `{ test, expect }` helpers used here.

---

## Follow-ups (out of scope for this run)

- **Playwright trace capture on auto-rotation failure:** consider `trace: 'on'` for the clock-based test if it ever flakes on firefox — virtual time can behave differently under that engine.
- **Hero banner + SSE interaction:** if `/events` ever pushes trending refresh events (future story), add a test that the active slide index resets appropriately.
- **Visual regression:** Percy/Playwright-screenshots for the hero banner not added; mark as P3 unless design system drift becomes a recurring issue.

---

## Knowledge Base References

- `test-levels-framework.md` — E2E for user-facing journey, API for contract, Unit for logic (no duplicate coverage)
- `test-priorities-matrix.md` — P0 for graceful-hide + display (data integrity & core UX), P1 for rotation/nav/modal variants
- `network-first.md` — all `page.route()` stubs registered before `page.goto()`
- `timing-debugging.md` — `page.clock` virtual time replaces `waitForTimeout`
- `test-quality.md` — Given-When-Then structure, explicit waits, no shared state, atomic assertions
