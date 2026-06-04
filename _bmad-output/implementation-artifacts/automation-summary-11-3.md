# Automation Summary — Story 11-3: Instant Search with zh-TW Priority

**Date:** 2026-06-04
**Story:** 11-3-instant-search-zh-tw-priority
**Mode:** BMad-Integrated
**Coverage Target:** critical-paths
**Test Architect:** Murat (TEA `*automate`)

---

## Strategy: avoid-duplicate-coverage

Story 11-3 already shipped strong **unit / component / integration** coverage from the
dev-story implementation. The `*automate` pass adds the **missing E2E layer** only — the
cross-layer, real-browser journey that lower levels cannot exercise. No behavior is
re-tested at a higher level than necessary (test-levels framework).

| Behavior | Lowest sufficient level | Where covered | E2E added? |
|----------|------------------------|---------------|------------|
| Dual-language merge + zh-TW boost + dedup | Unit (Go) | `search_service_test.go` | ❌ (no dup) |
| Graceful per-category degradation | Unit (Go) | `search_service_test.go` | ❌ |
| `SearchPeople` client call | Unit (Go) | `people_test.go` | ❌ |
| `/api/v1/search` handler 400/500/200 | Integration (Go) | `search_handler_test.go` | ❌ |
| Debounce timing, <2-char gating, keyboard index math, clear | Component | `InstantSearchBar.spec.tsx` | ❌ |
| Section render, dept label, active highlight, empty/loading | Component | `SearchSuggestions.spec.tsx` | ❌ |
| `unifiedSearch` URL + camelCase boundary | Unit | `tmdb.spec.ts` | ❌ |
| **Toolbar → debounce → API → dropdown → navigation** | **E2E** | **`instant-search.spec.ts`** | ✅ NEW |

---

## Tests Created

### E2E Tests — `tests/e2e/instant-search.spec.ts` (5 tests, hermetic, ~180 lines)

`/api/v1/search` is network-mocked (`page.route`), so the suite is deterministic and
needs **no `TMDB_API_KEY`**.

**Desktop (`Instant Search — Desktop`)**
- **[P0]** typing ≥2 chars opens a dropdown with 電影 / 影集 / 人物 sections; asserts the
  request carried `q` + `page=1` (AC #1). Includes the <2-char no-dropdown gate.
- **[P1]** clicking a suggestion navigates to `/media/movie/1` (AC #4)
- **[P1]** Enter with no highlight opens `/search?q=…` full-results page (AC #6 bridge)
- **[P1]** ArrowDown×2 + Enter opens the highlighted result `/media/tv/2` (AC #1, #4)

**Mobile (`Instant Search — Mobile`, viewport 390×844)**
- **[P2]** search toggle opens the full-screen overlay; typing renders live suggestions
  inside it (AC #5)

---

## Infrastructure

- No new fixtures/factories required — reused existing `tests/support/fixtures` (`test`,
  `expect`). Mock data is inline (snake_case wire shape, mirrors `discover-filters.spec.ts`).

---

## Execution & Validation (Step 5)

```bash
npx playwright test tests/e2e/instant-search.spec.ts --project=chromium
# → 5 passed (17.6s)
```

- ✅ **5/5 passed** on first run — no healing required.
- ✅ Orphaned-process cleanup ran automatically at session end (4 killed) — process-safety OK.
- ⚠️ Backend logs show TMDb errors for *other* shell endpoints (empty `TMDB_API_KEY` env) —
  unrelated to this suite; the mocked `/api/v1/search` never reaches the backend.
- ✅ `eslint` clean, `prettier --check` clean on the new spec.

---

## Coverage Status

- ✅ AC #1 (categorized suggestions within debounce) — E2E P0 + component
- ✅ AC #2 / #3 (zh-TW boost, dual-language merge) — Go unit (correctly kept at unit level)
- ✅ AC #4 (click → media detail) — E2E P1
- ✅ AC #5 (mobile full-width search view) — E2E P2
- ✅ AC #6 (backward compat / legacy `/search`) — E2E P1 bridge + existing `search.spec.ts`

**No coverage gaps identified.**

---

## Definition of Done

- [x] E2E tests follow Given-When-Then with `[P0]`/`[P1]`/`[P2]` priority tags
- [x] `data-testid` selectors (`instant-search-input`, `search-suggestions`,
      `search-suggestion-item`, `mobile-search-overlay`, `mobile-search-toggle`)
- [x] Network-first: routes intercepted before navigation
- [x] No hard waits, no conditional flow, deterministic assertions (web-first `expect`)
- [x] Hermetic (no external API dependency), self-contained
- [x] File under 300 lines
- [x] Validated by execution: 5/5 passing

---

## Next Steps

1. Run the suite in CI (sharded Playwright) alongside the rest of `tests/e2e/`.
2. Optional: quality-gate decision via TEA `*trace` (requirements→tests matrix).
3. Burn-in the new spec (10×) if flakiness is a concern before merge.
