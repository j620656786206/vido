# Automation Summary — Story 11-2 Persistent Filter Chip UI

**Date:** 2026-06-03
**Mode:** BMad-Integrated (story 11-2)
**Coverage Target:** critical-paths
**Author:** Murat (TEA `*automate`)
**Config:** `tea_use_playwright_utils: false` (traditional patterns) · `tea_use_mcp_enhancements: false` (no auto-heal loop)

---

## Coverage Strategy (Risk-Based)

The dev-story workflow already shipped **39 unit/component tests** for 11-2. Per the
avoid-duplicate-coverage principle, this expansion does **not** repeat component-level
logic. It adds the **integration/E2E layer** that lower levels structurally cannot
exercise:

- The URL ⇄ filter-state ⇄ discover API ⇄ chip-bar ⇄ results-grid round trip
- **Browser back/forward preserving filter state via URL (AC #4)** — unit tests mock
  `useNavigate`, so true history behaviour is only observable end-to-end
- Mobile filter bottom-sheet apply flow against a real DOM/viewport (AC #6)

### Test Level Rationale

| Behaviour | Level chosen | Why not lower |
|-----------|--------------|---------------|
| Chip render/remove logic, panel toggles, sheet draft/apply | Unit/Component (existing) | Pure UI logic — already covered, fast |
| URL↔backend param mapping, sort mapping | Unit (existing `discoverFilters.spec`) | Pure function |
| Filter → chip → URL → re-query integration | **E2E** | Crosses router + query + service + DOM |
| Back-button filter restoration (AC #4) | **E2E only** | Requires real browser history |
| Mobile bottom sheet apply | **E2E** | Viewport-gated (`lg:hidden`) + real overlay |

---

## Tests Created

**File:** `tests/e2e/discover-filters.spec.ts` (6 tests, hermetic — discover API mocked, no `TMDB_API_KEY` needed)

### E2E (P0)
- `[P0]` selecting a genre adds a chip, updates the URL, and re-queries with the filter (AC #1, #2, #5)
- `[P0]` browser back restores the previous filter state from the URL (AC #4)

### E2E (P1)
- `[P1]` removing a chip drops its URL param (AC #2)
- `[P1]` 清除全部 removes all chips at once when ≥2 filters active (AC #3)
- `[P1]` mobile bottom sheet: open → select genre → 套用篩選 updates chips + URL (AC #6)

### E2E (P2)
- `[P2]` a deep link renders the chips for the URL filter state (AC #4)

**Result:** `6 passed` (chromium), ~12s. No orphaned processes (global-teardown verified).

---

## 🐞 Bug Found & Fixed (E2E-surfaced)

**Severity: HIGH — shipped in the dev-story commit, would have reached production.**

A single-value deep link such as `/discover?genre=16` (or `?platform=8`) **silently dropped
the filter**. Root cause: TanStack Router's default search parser JSON-parses a lone numeric
query value (`16`) into the **number** `16`, but the route's `validateSearch` guarded with
`typeof search.genre === 'string'` and therefore discarded it. Multi-value `genre=16,28`
stayed a string and worked, which is why the unit tests (which pass string inputs) missed it.

**Fix (lane ① expand-scope-in-place — in code this story shipped):**
- `apps/web/src/routes/discover.tsx` — new `toCsvString()` coerces number→string for `genre`/`platform` in `validateSearch`.
- `apps/web/src/lib/discoverFilters.ts` — `parseCsvInts` now defensively `String()`-coerces its input.

Regression guard: `[P2] a deep link renders the chips` fails without the fix, passes with it.

> Note: the app emits the round-trip-safe encoded form `genre=%2216%22` when a filter is
> applied via the UI (TanStack quotes a string value to preserve its type); a clean
> `genre=16` deep link also round-trips via the coercion above. Both are functionally
> equivalent — the E2E asserts on either encoding.

---

## Infrastructure

No new fixtures/factories required — reused the existing `tests/support/fixtures`
(`test`, `expect`, `Route`) and the established `page.route(...jsonOk...)` mocking pattern
from `tests/e2e/explore-blocks.spec.ts`.

---

## Coverage Status

- ✅ AC #1 (chips at top, results respect filters) — E2E P0
- ✅ AC #2 (remove individual chip) — E2E P1 + unit
- ✅ AC #3 (清除全部 at ≥2) — E2E P1 + unit
- ✅ AC #4 (URL persistence / back button) — E2E P0 + P2
- ✅ AC #5 (panel selection → chips + results) — E2E P0 + component
- ✅ AC #6 (mobile bottom sheet) — E2E P1 + component

**Identified gaps (intentionally deferred, non-blocking):**
- Visual-regression baselines for the new `components/search/Filter*` components (Rule 22 cadence — belongs to the epic-11 retro design-drift audit, not this story).
- Platform-filter (`watch_providers`) end-to-end against a live backend — the discover mock covers the param-propagation contract; a live-TMDb smoke would duplicate the 11-1 API spec.

---

## Definition of Done

- [x] E2E tests follow Given-When-Then with `[P0]/[P1]/[P2]` priority tags
- [x] data-testid selectors (no brittle CSS/text-only selectors)
- [x] No hard waits — `expect`/`waitForURL`/`expect.poll` auto-retry only
- [x] Hermetic & deterministic (discover API mocked; no external TMDb dependency)
- [x] Self-contained (no shared state between tests; per-test route stubs)
- [x] Avoids duplicate coverage (component logic left to existing unit specs)
- [x] All 6 E2E pass; unit suite still green (20/20 discover specs); ESLint + Prettier clean

## Run

```bash
npx playwright test tests/e2e/discover-filters.spec.ts --project=chromium
# by priority
npx playwright test tests/e2e/discover-filters.spec.ts --project=chromium -g "\[P0\]"
```

## Next Steps

1. `discover-filters.spec.ts` is auto-discovered by the CI E2E shard (lives in `tests/e2e/`).
2. Quality gate (`*trace` / `*nfr`) optional before merge.
3. Address the deferred visual-baseline gap in the epic-11 retro (Rule 22).
