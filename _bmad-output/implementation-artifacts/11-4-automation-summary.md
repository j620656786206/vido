# Automation Summary — Story 11-4 Saved Filter Presets

**Date:** 2026-06-04
**Story:** 11-4-saved-filter-presets (P2-015)
**Mode:** BMad-Integrated
**Coverage Target:** critical-paths (gap-fill — story already shipped with dev-authored tests)
**Author:** Murat (TEA `*automate`)

---

## Pre-Existing Coverage (from dev-story — NOT duplicated)

| Level | Count | What |
|-------|-------|------|
| Unit — Go (mocked) | 21 | repository CRUD, service validate/limit/delete, handler routing/validation/limit/404 |
| Component — vitest (mocked hooks) | 21 | SavePresetDialog (6), PresetChips (7), FilterChipBar save-button (4 of 8), + existing |
| E2E — Playwright (mocked `/filter-presets`) | 3 | save→persist-on-reload (P0), apply preset (P0), right-click delete (P1) |

## Coverage Gap Analysis (risk-based)

The dev suite is strong but every layer that touches `/filter-presets` was **mocked** — the handler unit test swaps a mock service into a fresh gin router, and the UI E2E intercepts the endpoints entirely. Two genuine, non-duplicative gaps remained:

| Gap | Risk (prob × impact) | Level chosen | Rationale |
|-----|----------------------|--------------|-----------|
| **Live backend wire contract** | MED × HIGH | **API (P1)** | No test exercised real HTTP→service→repo→SQLite, `main.go` route registration (Rule 15 precedent class), or migration 023. A broken route / un-applied migration would ship green. |
| **caseTransform JSON-string boundary** | LOW × HIGH | **Unit (P2)** | The "filters as opaque string" design decision was only covered indirectly. If `camelToSnake`/`snakeToCamel` ever mangled the string's inner `year_gte` keys, presets silently corrupt. |

**Deliberately NOT added (avoid-duplicate-coverage):**
- Max-20 limit at API level — covered at service + handler unit level, and creating 20 rows on a shared dev DB is destructive.
- Re-testing save/apply/delete journeys at API or component level — already covered by the dev E2E + component specs.

## Tests Created

### API Tests (P1-P2) — `tests/e2e/saved-filter-presets.api.spec.ts` (5 tests)
- **[P1]** POST persists + GET returns it with `filters` intact as a JSON string (AC #1, #2, #5) — locks the wire round-trip + real SQLite persistence
- **[P1]** DELETE removes a persisted preset (AC #4)
- **[P1]** POST empty name → 400 `FILTER_PRESET_VALIDATION_FAILED` (AC #1)
- **[P1]** POST malformed filters JSON → 400 (AC #1)
- **[P2]** DELETE non-existent → 404 `FILTER_PRESET_NOT_FOUND`

*Non-destructive + self-cleaning:* tracks created ids, deletes only those in `afterEach` (verified in run log: create → delete). Hits the live backend (port 8080); needs no TMDB key.

### Unit Tests (P2) — `apps/web/src/services/filterPresetService.spec.ts` (4 tests)
- **[P2]** `getAll` — maps `sort_order`→`sortOrder` but leaves `filters` string (inner `year_gte`) untouched
- **[P2]** `create` — outbound body keeps `filters` as the exact JSON string; inbound response preserved
- **[P2]** `create` — throws backend error message on non-ok (e.g. 409 limit)
- **[P2]** `remove` — DELETEs by id

## Test Execution

```bash
# New unit spec (frontend service boundary)
npx vitest run "filterPresetService"            # 4 pass

# New live-backend API spec (auto-starts go backend on :8080)
npx playwright test tests/e2e/saved-filter-presets.api.spec.ts --project=chromium   # 5 pass

# Pre-existing UI journey E2E
npx playwright test tests/e2e/saved-filter-presets.spec.ts --project=chromium       # 3 pass
```

**Result:** 5 API + 4 unit = **9 new tests, all green.** ESLint + prettier clean. Cleanup verified (no orphaned processes).

## Coverage Status (post-TA)

| AC | Covered by |
|----|-----------|
| #1 save with name + dialog | component (SavePresetDialog) + UI E2E + **API (validation)** |
| #2 presets as chips | component (PresetChips) + UI E2E + **API (GET round-trip)** |
| #3 click → restore filters | component + UI E2E |
| #4 delete after confirm | component + UI E2E + **API (DELETE + 404)** |
| #5 DB persistence (not localStorage) | UI E2E (reload) + **API (real SQLite GET after POST)** |

✅ All ACs now covered at ≥2 levels including the **real backend stack**. Test pyramid is healthy: unit-heavy, API mid, E2E thin (critical paths only).

## Definition of Done
- [x] Given-When-Then format, priority tags ([P1]/[P2])
- [x] data-testid / API contract assertions (no fragile selectors)
- [x] Self-cleaning (API spec tracks + deletes created rows; unit spec stubs fetch)
- [x] Deterministic — no hard waits, no conditional flow
- [x] Files lean (API 124 lines, unit 123 lines)
- [x] ESLint + prettier clean; no duplicate coverage with dev suite

## Next Steps
1. Review the 2 new spec files with the team
2. The `@api` spec runs in the existing Playwright CI shards (tagged `@regression`)
3. Optional: TEA `*trace` to produce a formal requirements-to-tests matrix + gate decision
4. Recommend running `*nfr-assess` only if preset volume/perf becomes a concern (currently capped at 20 — low risk)

**Knowledge applied:** test-levels-framework (API vs E2E vs unit selection), test-priorities-matrix (P1/P2), test-quality (deterministic, self-cleaning), data-factories (inline non-destructive seeding).
