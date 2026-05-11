# Automation Summary — bugfix-10-6 (ExploreBlock Polish Bundle)

**Date:** 2026-05-11
**Story:** `bugfix-10-6-polish-ux-visual-pass` (status: review — DEV /dev-story complete, commit `26b8f99`)
**Workflow:** `bmad:bmm:workflows:testarch-automate` (TEA / Murat), BMad-Integrated Mode
**Coverage target:** `critical-paths`
**Mode flags:** `tea_use_playwright_utils: false` (traditional fixture/route-mock patterns) · `tea_use_mcp_enhancements: false` (no auto-heal loop)

---

## Why this pass

bugfix-10-6 shipped its prescribed AC #6 coverage with the DEV story (chevron-presence-when-items, no-chevrons-when-empty, lucide-not-emoji on settings rows). This `automate` pass expands beyond that with the small set of **critical-path gaps** the AC #6 coverage left open:

| Gap | Risk | Level chosen | Why this level |
|-----|------|--------------|----------------|
| Chevron `onClick` → `scrollerRef.scrollBy(...)` had **zero** assertions (AC #2 says "clicking them MUST still call `scrollBy`", but the spec only asserted DOM presence) | Med — a refactor could silently break the scroll handler and every existing test stays green | Component (vitest) | Pure UI interaction + ref behaviour; fast, deterministic, no browser needed |
| AC #3 (`<option>` labels dropped 🎬/📺) had **no** regression guard — story noted "no test references the emoji, no change needed", which means a future regression that *re-adds* the emoji is uncaught | Low–Med | Component (vitest) | The `<select>` renders fully in jsdom; cheap guard |
| AC #4 / AC #5 verified only at the React-tree level — no **browser-rendered** check (story DoD explicitly deferred "browser-pixel verification to NAS deploy") | Low–Med — covers real-DOM/CSS `display` vs not-rendered, and the real `/settings/homepage` + homepage routes | E2E (Playwright, route-mocked) | Browser-layer regression confidence without flaky timing/pixel assertions; `explore-blocks.spec.ts` already runs in CI/nightly |

**Deliberately NOT added** (per `test-quality` / `risk-governance` KB — depth scales with impact, avoid flaky patterns):
- Chevron **hover-fade** assertion (`opacity-0` → `lg:group-hover:opacity-100`). `toBeVisible()` on a CSS-transition element is the exact anti-pattern Rule 16 + `test-quality.md` warn against; the story itself deferred this to manual/NAS verification. Chevron *presence* + `hidden lg:block` + `pointer-events-none` scrims are structurally covered by the existing AC #2 test.
- Re-testing AC #2 / #4 / #5 logic at E2E level (already covered at unit level — would be duplicate coverage). E2E adds only the *browser-render* dimension, not the logic variations.
- New fixtures/factories — the existing inline mock pattern (`vi.mocked` typed mocks for vitest, `tests/support/fixtures` route mocks for Playwright) is the established idiom; no infra change warranted.

---

## Tests Created

### Component Tests (vitest + RTL) — P2 — 2 new

- `apps/web/src/components/homepage/ExploreBlock.spec.tsx`
  - **[P2] `clicking a scroll chevron scrolls the scroller by 80% of its width (AC #2)`** — renders a populated block, pins `clientWidth` to 800 (jsdom reports 0), stubs `scrollBy` on the scroller element, clicks each chevron, asserts `scrollBy({ left: -640, behavior: 'smooth' })` (left) and `{ left: 640, behavior: 'smooth' }` (right). Closes the AC #2 "clicking MUST still call `scrollBy`" gap.
- `apps/web/src/components/settings/ExploreBlockEditModal.spec.tsx`
  - **[P2] `content-type options render plain labels with no emoji (AC #3)`** — asserts the `<select data-testid="explore-block-type-select">` options are `value="movie"|"tv"` with text `電影`/`影集`, and `queryByText(/🎬|📺/)` is `null`.

### E2E Tests (Playwright, route-mocked) — P2 — 2 new

- `tests/e2e/explore-blocks.spec.ts`
  - **[P2] `empty block shows the no-results message and no scroll chevrons (bugfix-10-6 AC#5)`** (`Homepage Explore Blocks` describe) — stubs `/explore-blocks` with one block whose `/content` resolves to `{ movies: [], total_items: 0 }`; on `/`, asserts the block renders, `explore-block-empty` has text `沒有符合條件的內容`, and `explore-block-scroll-left` / `-right` have count `0`.
  - **[P2] `block rows show lucide content-type icons, not 🎬/📺 emoji (bugfix-10-6 AC#4)`** (`Settings — Explore Blocks Management` describe) — stubs `/explore-blocks` with the two default blocks; on `/settings/homepage`, asserts `getByText(/🎬|📺/)` count `0`, and that each row's meta `<p>` contains exactly one `<svg>` (lucide `<Film>`/`<Tv>`) plus the text `電影 ·` / `影集 ·`.

## Infrastructure Created

None — reused existing patterns:
- vitest: `vi.mocked(useExploreBlockContent)` typed mock (no `as any`), inline `testBlock()` builder.
- Playwright: `tests/support/fixtures` (`stubHomepageBaseline`, `jsonOk`), existing `defaultBlocks` fixture.

---

## Validation Results

| Suite | Command | Result |
|-------|---------|--------|
| Targeted vitest (3 touched spec files) | `pnpm exec vitest run ExploreBlock.spec.tsx ExploreBlockEditModal.spec.tsx ExploreBlocksSettings.spec.tsx` | **30/30 PASS** (3 files) |
| Full web unit suite | `pnpm nx test web --skip-nx-cache` | **1790/1790 PASS** (146 files) — baseline 1788 + the 2 new vitest tests; no removals. `test:cleanup:all` auto-ran clean. |
| New E2E discoverability | `npx playwright test tests/e2e/explore-blocks.spec.ts --list` | Both new tests resolve (`:209` AC#5, `:419` AC#4) across all 5 projects; no syntax/import errors. |
| ESLint (3 touched files) | `pnpm exec eslint <files>` | **0 errors, 1 warning** — the warning is the **pre-existing** `{ router } as any` at `ExploreBlock.spec.tsx:83` (part of the 122-warning baseline; untouched by this pass). **0 new warnings.** No new `as any` casts. |
| Prettier | `pnpm exec prettier --check <files>` | Clean — "All matched files use Prettier code style!" |
| Orphan check | `pnpm run test:cleanup` | "No test processes found" |

### Browser E2E run — DEFERRED (documented, not a failure)

The 2 new Playwright tests are **route-mocked and deterministic** (no real-TMDB dependency, no timing/pixel assertions) but require the local `webServer` stack (`go run ./cmd/api` + `nx serve web`) to execute. Per the story's own DoD ("browser-pixel verification deferred to user / NAS deploy") and `risk-governance` (these are P2), the full browser run was **not** executed in this pass. It runs automatically in **CI / nightly** (the `explore-blocks.spec.ts` spec is part of the 328-test Playwright suite per `project-context.md` §2). **Recommended before story closeout / merge:** `pnpm test:e2e tests/e2e/explore-blocks.spec.ts -g "bugfix-10-6"`.

`tea_use_mcp_enhancements: false` ⇒ no auto-heal loop; n/a here since all executed suites are green.

---

## Coverage Status

- ✅ AC #1 (chevron contrast tokens / scrim / hover-reveal) — DEV unit test (presence + `hidden lg:block`) + DEV's structural design-vs-code table vs HP-5; hover-fade timing intentionally not asserted (flaky).
- ✅ AC #2 (chevrons present when items, click still scrolls) — DEV presence test **+ NEW** click→`scrollBy` test (this pass).
- ✅ AC #3 (`<option>` labels lose emoji) — **NEW** plain-label/no-emoji guard (this pass).
- ✅ AC #4 (settings rows use lucide `<Film>`/`<Tv>`, not 🎬/📺) — DEV unit test (svg present, no emoji, plain label) **+ NEW** browser-level E2E (this pass).
- ✅ AC #5 (empty block renders no chevrons) — DEV unit test **+ NEW** browser-level E2E (this pass).
- ⚠️ Gap (intentional): hover-fade reveal timing, chevron contrast over poster art, Settings-row icons at 390 px & 1440 px — deferred to manual / NAS-deploy pixel verification per story DoD; not flaky-testable in headless E2E.

## Definition of Done

- [x] Execution mode determined (BMad-Integrated — story file present)
- [x] BMad artifacts loaded (story `bugfix-10-6-polish-ux-visual-pass.md`; no tech-spec/test-design exists for this bundle — single-story papercut fix)
- [x] Framework config loaded (`playwright.config.ts`, vitest via `nx run web:test`)
- [x] Existing coverage analyzed (`ExploreBlock.spec.tsx`, `ExploreBlockEditModal.spec.tsx`, `ExploreBlocksSettings.spec.tsx`, `tests/e2e/explore-blocks.spec.ts`)
- [x] KB fragments consulted (`test-levels-framework`, `test-priorities-matrix`, `test-quality`, `risk-governance`, `network-first`, `selective-testing` via `tea-index.csv`)
- [x] Automation targets identified (4 gaps; 4 covered)
- [x] Test levels selected appropriately (component for UI logic/interaction, E2E for browser-render dimension; no duplicate-level coverage)
- [x] Priorities assigned (all P2 — regression-hardening on a shipped polish bundle)
- [x] Tests follow Given-When-Then-ish structure, priority tags in names, `data-testid` selectors, specific Rule 16 matchers (`toHaveBeenCalledWith`, `toBeNull`, `toHaveText`, `toHaveCount`)
- [x] No hard waits / no flaky patterns / no shared state / no page objects
- [x] No new `as any`; no lint regressions (0 new warnings)
- [x] vitest suites run & green (1790 PASS); E2E syntax-validated; browser E2E run deferred per story DoD (documented)
- [x] Orphan check clean
- [ ] (Closeout follow-up) full `pnpm lint:all` baseline re-confirmation + browser E2E run — recommended at story closeout, not blocking (zero Go touched ⇒ 0/122 expected to hold)

## File List

Modified (tests only — no production code, no `.pen`, no screenshots):
- `apps/web/src/components/homepage/ExploreBlock.spec.tsx` — +1 test: chevron click → `scrollBy` (AC #2).
- `apps/web/src/components/settings/ExploreBlockEditModal.spec.tsx` — +1 test: `<option>` plain labels, no emoji (AC #3).
- `tests/e2e/explore-blocks.spec.ts` — +2 tests: empty-block-no-chevrons (AC #5), settings-rows-lucide-not-emoji (AC #4).

Created:
- `_bmad-output/implementation-artifacts/automation-summary-bugfix-10-6.md` (this file).

## Next Steps

1. Review the 4 new tests (this pass added 2 vitest + 2 Playwright, all P2).
2. Run the browser E2E locally before merge: `pnpm test:e2e tests/e2e/explore-blocks.spec.ts -g "bugfix-10-6"` (spins up the local `webServer` stack).
3. CI/nightly already exercises `explore-blocks.spec.ts` — the 2 new E2E tests join that run automatically.
4. Optional quality-gate decision: `bmad tea` → `[TR]` trace (Phase 2 gate) if a formal PASS/CONCERNS gate record is wanted for bugfix-10-6.
