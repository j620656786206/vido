# Automation Summary — Story 10.3 Custom Explore Blocks

**Date:** 2026-04-16
**Story:** 10-3-custom-explore-blocks (status: `review`)
**Mode:** BMad-Integrated
**Coverage Target:** critical-paths (expand beyond ATDD with edge cases + negative paths)
**TEA Workflow:** `_bmad/bmm/workflows/testarch/automate`

---

## Context

Story 10.3 was already implemented with comprehensive coverage at the unit/integration/E2E layers:

- **Backend (47 tests):** 13 repository + 17 service + 17 handler tests
- **Frontend (22 component tests):** 7 ExploreBlock + 4 ExploreBlocksList + 7 ExploreBlocksSettings + 4 ExploreBlockEditModal
- **E2E (9 scenarios baseline):** homepage render / settings CRUD entry points / API contract

The TA workflow's job here is **gap analysis + expansion**, not duplicating existing coverage. Per `test-levels-framework.md`: E2E reserved for critical happy paths, API/Component for variations — principle observed.

---

## Coverage Matrix (AC → Tests)

| AC                                     | Backend Unit/Integration                                                  | Component          | E2E (pre-TA)                                      | E2E (added by TA)                                     |
| -------------------------------------- | ------------------------------------------------------------------------- | ------------------ | ------------------------------------------------- | ----------------------------------------------------- |
| **#1** Horizontal rows + section title | repo GetAll ordering, service ReorderBlocks                               | ExploreBlock ×7    | renders blocks w/ title, poster click → detail    | —                                                     |
| **#2** Create block (all fields)       | repo Create, service CreateBlock + validation × 3, handler Create×3       | EditModal ×4       | modal opens with all fields                       | **create flow submits POST w/ snake_case**            |
| **#3** Reorder                         | repo Reorder + rollback + empty, handler Reorder×3 + route collision test | Settings reorder×2 | reorder **up** arrow                              | **reorder down arrow**                                |
| **#4** Edit / delete without reload    | service UpdateBlock + cache invalidation, handler Update/Delete×4         | Settings delete×1  | delete confirmation                               | **edit round-trip PUT**, **homepage reflects delete** |
| **#5** Fresh install default blocks    | service SeedDefaultsIfEmpty + Idempotent                                  | —                  | (implicit in default-blocks render)               | —                                                     |
| **#6** TMDB discover integration       | service GetBlockContent (Movie/TV/Cap/Cache/Error/NotFound)               | —                  | content endpoint envelope shape                   | —                                                     |

**Non-AC architectural guards**:

- Route ordering regression (`/reorder` vs `/:id`) — handler unit test
- Cache invalidation fan-out — service test
- Content-filter integration (far-future + low-quality) — service test
- CHECK constraint on `content_type IN ('movie', 'tv')` — repository test

---

## Tests Added by This Workflow

**File:** `tests/e2e/explore-blocks.spec.ts` (4 additions → total 13 scenarios)

### Settings — Explore Blocks Management

1. **[P0] edit modal round-trip submits PUT with updated payload (AC4)**
   Opens edit modal for existing block, changes name, clicks save, asserts `PUT /api/v1/explore-blocks/:id` fires with updated `{ name }` field and modal closes. Fills the gap between component-level mutation test and end-to-end SPA behaviour.

2. **[P1] create flow submits POST with snake_case payload (AC2)**
   Fills form, clicks save, asserts `POST /api/v1/explore-blocks` receives `{ name, content_type, max_items }` (not camelCase). Proves Rule 18 (API boundary case transformation) end-to-end — critical because Rule 18 was a recurring bug source (see Rule 18's 2026-03-28 audit note).

3. **[P1] reorder down arrow swaps adjacent blocks (AC3)**
   Complements the existing up-arrow test. Down arrow is a distinct click path on a distinct DOM node and could diverge.

### Homepage reflects settings changes (new `describe` block)

4. **[P0] deleting a block in settings removes it from homepage without reload (AC4)**
   Full cross-route SPA journey: `/` → `/settings/homepage` → delete → back to `/`. The stub serves a two-block payload initially, then a one-block payload after DELETE fires, proving TanStack Query's `exploreBlockKeys.all` invalidation propagates. This is the only E2E that actually validates the AC's "without page reload" guarantee end-to-end.

**All 4 additions passed on first run.** Full spec: **13/13 passed in 14.6s** on chromium (`npx playwright test tests/e2e/explore-blocks.spec.ts --project=chromium`).

---

## Gaps Considered But NOT Filled

These were deliberately skipped — adding them would duplicate existing coverage without raising confidence:

| Considered                                         | Skipped because                                                                                           |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Isolated `models.ExploreBlock` validation test     | Exhaustively covered via service-level `TestExploreBlockService_CreateBlock_ValidationErrors` table test  |
| Backend integration test for SQLite schema         | `explore_block_repository_test.go` runs against real `:memory:` SQLite with the same schema as production |
| E2E for `max_items` boundary (41 → HTTP 400)       | Service validation table covers it; UI `<input type="number" max={40}>` enforces client-side              |
| E2E for seed-defaults on fresh install             | `TestExploreBlockService_SeedDefaultsIfEmpty` + `_Idempotent` + main.go wiring — integration-tested       |
| E2E for cache 1-hour TTL expiration                | `TestExploreBlockService_GetBlockContent_Caches` covers; TTL verification is repository-level concern     |
| Unit tests for `buildSeeMoreTarget` (ExploreBlock) | Returns a static `/search` placeholder pending Epic 11 — not yet stabilized behaviour                     |

---

## Infrastructure Changes

**None needed.** The existing test infrastructure was sufficient:

- Fixtures: `tests/support/fixtures/index.ts` (re-used)
- Factories: not needed — inline JSON mocks are clearer for API stubs
- Helpers: `jsonOk`, `stubHomepageBaseline` already existed in the file

Per workflow principles: don't create infrastructure without a second caller — no over-engineering.

---

## Quality Standards Audit

All 13 scenarios verified:

- [x] Given-When-Then structure (implicit — Playwright arrange/act/assert)
- [x] Priority tags `[P0]` / `[P1]` in every test name
- [x] `data-testid` selectors exclusively (no CSS classes, no XPath)
- [x] Route interception **before** `page.goto()` (network-first, `network-first.md`)
- [x] No `waitForTimeout()` / no hard waits
- [x] No conditional flow (`if (await el.isVisible())`)
- [x] Deterministic — same input → same result (stubbed network)
- [x] Self-cleaning (no DB writes; stubs auto-dispose per page context)
- [x] Under 500 lines (current: ~485 lines with 13 scenarios — acceptable)

---

## Running the Tests

```bash
# Run the full story 10-3 E2E suite
npx playwright test tests/e2e/explore-blocks.spec.ts --project=chromium

# Run by priority (after CI pipeline adds grep flags to playwright.config)
npx playwright test --grep "@P0"
npx playwright test --grep "@P0|@P1"

# Run full backend coverage
cd apps/api && go test ./internal/... -run ExploreBlock -v

# Run frontend component coverage
npx vitest 'explore' --reporter=verbose
```

**Tag-based selection available now:** `--grep "@explore-blocks"` or `--grep "@story-10-3"`.

---

## Definition of Done

- [x] All ACs (1–6) covered at ≥ 1 test level
- [x] Critical paths (P0) covered by E2E
- [x] Variations + edge cases covered at API/component level
- [x] Error paths tested (404, 400, service errors, TMDb failure, cache failure)
- [x] Route-ordering regression guarded
- [x] Cache-invalidation regression guarded
- [x] API boundary (Rule 18 snake_case) verified end-to-end (new)
- [x] SPA cross-route cache invalidation verified end-to-end (new)
- [x] All tests deterministic, no flaky patterns
- [x] 13/13 passing locally on chromium
- [x] Prettier formatted

---

## Next Steps

1. **Merge these additions** — CI already runs `tests/e2e/explore-blocks.spec.ts` in the sharded Playwright matrix, so no workflow changes needed.
2. **Track in burn-in loop** — the 4 new tests should pass 10× consecutive runs (`ci-burn-in.md`). If any flake surfaces, the SPA cross-route test (#4) is the highest-risk candidate; consider tightening with an explicit `waitForResponse` on the second `/explore-blocks` fetch.
3. **Future enhancement:** when Epic 11 finalizes `/search` with filter scaffolding, add an E2E for "查看更多" routing with block filters applied.

---

## Knowledge Base References Applied

- `test-levels-framework.md` — E2E reserved for cross-route SPA journeys; API/component for variations
- `test-priorities-matrix.md` — P0 for Rule 18 snake_case + cross-route delete (compliance + AC proof); P1 for paired variants
- `network-first.md` — all stubs registered before `page.goto()`
- `test-quality.md` — deterministic stubs, data-testid selectors, Given-When-Then structure
- `selective-testing.md` — tags `@ui @api @explore-blocks @story-10-3` enable targeted execution

---

**Output file:** `_bmad-output/automation-summary-10-3.md`
**Modified file:** `tests/e2e/explore-blocks.spec.ts` (+4 scenarios)
**Validation:** 13/13 E2E passed in 14.6s (chromium, 4 workers)
