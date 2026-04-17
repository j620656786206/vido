# Automation Summary — Story 10-5: Homepage Layout Engine & Responsive Design

**Date:** 2026-04-17
**Agent:** Murat (Master Test Architect) — BMM `/testarch-automate` workflow
**Story File:** `_bmad-output/implementation-artifacts/10-5-homepage-layout-responsive.md`
**Mode:** BMad-Integrated (story available, ACs traced)
**Coverage Target:** critical-paths

---

## Scope

DEV Amelia closed Story 10-5 with **17 new Vitest unit tests** (1738/1738 suite green) covering:

- `ExploreBlockSkeleton.spec.tsx` (3) — placeholder count, aria-hidden
- `useInViewport.spec.ts` (4) — IntersectionObserver callback, `once` latch, SSR fallback
- `routes/index.spec.tsx` (4) — section DOM order, `flex flex-col gap-6 md:gap-8` classes, `hideWhenEmpty` prop threading, loader prefetch call
- `RecentMediaPanel.spec.tsx` (2) — `hideWhenEmpty` empty + loading-no-flash
- `DownloadPanel.spec.tsx` (3) — `hideWhenEmpty` disconnected + empty-connected + loading-no-flash

**Gaps DEV unit tests cannot cover** (reason jsdom/stubs don't suffice):

| # | Gap | Why Vitest Can't Cover It |
|---|-----|---------------------------|
| 1 | AC #1 real-router DOM order | jsdom test mocks `HeroBanner` / `ExploreBlocksList` / panels as stubs — verifies the JSX skeleton but not the full composition of AppShell + route loader + real children |
| 2 | AC #3 hero pixel heights | `h-[250px]` / `md:h-[400px]` are Tailwind arbitrary classes — only a real browser proves they compile to 250/400px at viewport 390/1440 |
| 3 | AC #3 mobile horizontal scroll | `scrollWidth > clientWidth` is a real-DOM measurement, not a class token |
| 4 | AC #2 lazy-load **network** proof | jsdom's `IntersectionObserver` is an inert stub — the third block's content endpoint never fires regardless. A real browser lets us prove `b3` and `b4` stay at zero fetches until scrolled |
| 5 | AC #2 route-loader prefetch on Link hover | Requires the real router + AppShell + a real `<Link to="/">`; can't be exercised in a component spec |
| 6 | AC #4 per-block skeleton during **inflight** request | Unit tests mock `isLoading: true` — the deferred-fetch handoff (skeleton → poster cards) is only observable when real network timing is present |
| 7 | AC #5 panels truly absent from live DOM | Unit tests stub the panels themselves; proves only the prop threading — not that the real panels return `null` after the real `useQuery` resolves empty |
| 8 | Legacy `dashboard.spec.ts` assertions still referenced the removed `dashboard-layout` / `dashboard-grid` testids + expected empty-state placeholders that are now hidden by `hideWhenEmpty` | 10-5 refactor structurally broke 4 pre-existing E2E tests — would have surfaced as failing runs on the next CI pass |

---

## Tests Created / Updated

### New file: `tests/e2e/homepage-layout.spec.ts` (6 new tests)

| Priority | Test | AC | Closes gap # |
|---------|------|----|--------------|
| **[P0]** | `AC #1 — Hero → Explore → Recent → Downloads render in that order` | 1 | 1 |
| **[P0]** | `AC #3 — hero is 250px tall at mobile (390×844 iPhone)` | 3 | 2 |
| **[P0]** | `AC #3 — hero is 400px tall at desktop (1440×900)` | 3 | 2 |
| **[P1]** | `AC #3 — explore block scroller is horizontally scrollable on mobile` | 3 | 3 |
| **[P0]** | `AC #2 — below-the-fold block content is NOT fetched until scrolled into view` | 2 | 4 |
| **[P1]** | `AC #2 — route loader prefetches trending hero BEFORE navigation when a Link to "/" is hovered` | 2 | 5 |
| **[P0]** | `AC #4 — per-block skeleton renders while each block content is inflight, then swaps to cards` | 4 | 6 |
| **[P0]** | `AC #5 — when trending/blocks/recent/downloads all return empty, only HeroBanner-less homepage-root survives` | 5 | 7 |
| **[P1]** | `AC #5 — populated Downloads + empty Recent renders only DownloadPanel` | 5 | 7 |

Total: **9 new tests** (6 P0, 3 P1). All use network-first interception — zero dependency on live TMDb / backend.

### Updated file: `tests/e2e/dashboard.spec.ts` (4 broken assertions fixed / rewritten)

| Test (original) | Fix |
|-----------------|-----|
| `[P1] should display dashboard with download panel and recent media (AC1)` | Renamed to "should display homepage with…"; `dashboard-layout` testid → `homepage-root` |
| `[P1] should show disconnected state when qBittorrent not configured (AC3)` | Renamed to "Story 10-5 AC #5 — disconnected qBittorrent hides Download panel on homepage"; flipped assertion from "shows 未連線 banner" to "panel has count 0". Explicit comment notes the disconnected UI itself is still covered by `DownloadPanel.spec.tsx` unit. |
| `[P1] should stack panels vertically on mobile (AC4)` | Renamed to "Story 10-5 — panels stack vertically on homepage flex column"; replaces `dashboard-grid` grid-template-columns check with `homepage-root` computed-style (`display: flex`, `flex-direction: column`, `rowGap: 24px`) — verifies the new `flex-col gap-6` actually compiles, not just its class tokens. |
| `[P2] should show empty states when connected but no data (AC1, AC3)` | Renamed to "Story 10-5 AC #5 — homepage hides both panels entirely when connected but empty"; flipped assertion from "empty placeholders visible" to "panels and placeholders have count 0". Downloads payload shape also corrected (was returning `data: []` which doesn't match the `items: [...], totalItems` wrapper the hook expects). |

---

## Test Coverage Plan Coverage

Every AC has at least one P0 or P1 E2E run **after** DEV's Vitest layer:

- **AC #1 (section order)** — 1 P0 E2E (real-router DOM) + 1 updated P1 E2E (populated render check) + 1 Vitest (testid order)
- **AC #2 (LCP < 2s)** — 1 P0 E2E (lazy-load network proof) + 1 P1 E2E (route-loader prefetch on hover) + 1 Vitest (loader mock called)
- **AC #3 (responsive mobile)** — 2 P0 E2E (real hero pixel height @ 390 / 1440) + 1 P1 E2E (horizontal scroll proof) + Vitest class tokens
- **AC #4 (per-block skeleton)** — 1 P0 E2E (deferred route; skeleton→cards transition) + Vitest (skeleton renders when isLoading)
- **AC #5 (empty-section hide)** — 1 P0 E2E (all empty, only homepage-root survives) + 1 P1 E2E (one populated, one empty) + 1 updated P1 E2E (panels count=0) + Vitest (hideWhenEmpty prop wiring & per-panel empty/loading branches)

No duplicate coverage — each level tests a different axis:

- Vitest: prop contracts + class tokens + mock returns
- E2E: real router + real network timing + real Tailwind CSS output + real browser viewport measurement

---

## Infrastructure

**No new fixtures or factories needed.** Reused:

- `tests/support/fixtures/index.ts` — existing `test`/`expect` composed fixtures
- Existing route-interception pattern (`page.route` + `jsonOk` / `jsonRaw` helpers inline) matching the house style from `hero-banner.spec.ts`, `explore-blocks.spec.ts`, `availability-badges.spec.ts`
- Existing `stubHomepageBaseline` pattern — inlined a module-private version in `homepage-layout.spec.ts` tuned for 4 blocks so the lazy-load test has enough content off-screen

---

## Validation

### Static

- **`pnpm lint:all`** → 0 errors, 129 pre-existing warnings (no new ones introduced)
- **`pnpm exec prettier --check .`** → all files formatted
- **`pnpm nx test web --run`** → **1738 / 1738 PASS** (full regression, still green after dashboard.spec.ts edits — the changed tests are Playwright-only so they don't run under vitest, but the shared `apps/web` units confirm no upstream breakage)

### Runtime (to be executed in CI / on dev server)

The 9 new homepage-layout tests and 4 updated dashboard tests should execute via:

```bash
# All homepage + dashboard tests
npx playwright test tests/e2e/homepage-layout.spec.ts tests/e2e/dashboard.spec.ts

# Only Story 10-5 additions
npx playwright test --grep "@story-10-5"

# Only P0
npx playwright test tests/e2e/homepage-layout.spec.ts --grep "\[P0\]"
```

Execution requires:

1. `nx serve web` running (Vite dev server on :4200)
2. `nx serve api` or Docker backend running (Go API on :8080)
3. QB / TMDb backends can be OFF — the tests intercept every call

### Known edge case (documented in-test)

The `AC #2 lazy-load` test asserts `hits.b3 === 0` initially and `hits.b4 <= 1`. The upper bound on `b4` is intentionally loose because `rootMargin: '400px'` may pull `b4` into the intersection zone at the tester's block height. If this proves false on real devices, the assertion can be tightened to `hits.b4 === 0` and the mocked `blockContent` can be padded with more cards to push `b4` further down.

---

## Definition of Done

- [x] All new tests follow Given-When-Then via `// GIVEN/WHEN/THEN` comments or via described setup blocks
- [x] All new tests tagged with priority (`[P0]` / `[P1]`) in the test name
- [x] All new tests use `data-testid` selectors (hero-banner, explore-blocks-list, recent-media-panel, download-panel, explore-block-b{1..4}, explore-block-scroller, explore-block-skeleton, homepage-root, explore-block-b3)
- [x] Network-first intercept applied to **every** test — no live TMDb / backend traffic
- [x] No hard waits (`waitForTimeout` / `sleep`); only `expect.poll`, `toBeVisible`, `waitForLoadState('networkidle')`, `scrollIntoViewIfNeeded`
- [x] No try/catch around test logic
- [x] No page object classes — direct Playwright APIs throughout
- [x] Self-cleaning — no test mutates shared state; each gets its own `page` + `route` handlers
- [x] Deterministic — release-on-command pattern for skeleton test (`new Promise` gate) instead of racing timers
- [x] Test file under 420 lines (homepage-layout.spec.ts final ≈ 418)
- [x] Automation summary written

---

## File List

**New:**
- `tests/e2e/homepage-layout.spec.ts` — 9 new E2E tests (6 P0, 3 P1)

**Modified:**
- `tests/e2e/dashboard.spec.ts` — 4 tests updated for Story 10-5 structural refactor (2 renamed + assertion flips, 2 rewritten)
- `_bmad-output/automation-summary-10-5.md` — this document

---

## Next Steps

1. **Commit** these changes as `test(e2e): Story 10-5 TA expansion — …` per recent commit style (see `353e99f test(e2e): Story 10-4 TA expansion — 22 new E2E/API runs`).
2. **Run E2E on NAS dev server** — proves the lazy-load, hero-height, and prefetch assertions on real Chromium/Webkit/Firefox (3 projects).
3. **Handoff to CR (Amelia)** — fresh context, different LLM recommended per dev workflow Step 11. CR should validate AC fulfillment, architecture compliance (Rules 1–19), and check that the E2E assertions truly close the jsdom gaps listed above.
4. **Optional**: after CR passes, consider trimming `dashboard.spec.ts` further — several remaining tests (`should display downloads in compact view`, `should display recent media with titles`) overlap with newer `homepage-layout.spec.ts` scenarios but stay green so they aren't dead weight yet.

---

## Knowledge Base References Applied

- `test-levels-framework.md` — E2E chosen over component tests for real-router DOM order, real-browser pixel measurement, and real-network lazy-load proof
- `test-priorities-matrix.md` — P0 for AC-critical real-browser gaps; P1 for secondary guarantees
- `network-first.md` — `page.route(...)` intercept before `page.goto()` on every test
- `fixture-architecture.md` — reused existing `test`/`expect` from `tests/support/fixtures/index.ts`; no new fixtures needed
- `selective-testing.md` — `@homepage @story-10-5 @ui` tags for grep-based selection
- `test-quality.md` — deterministic releases via promise gates, no hard waits, one logical assertion per test
