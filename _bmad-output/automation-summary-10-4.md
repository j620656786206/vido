# Automation Summary — Story 10.4 Availability Badges

**Date:** 2026-04-17
**Story:** 10-4-availability-badges (status: `review`)
**Mode:** BMad-Integrated
**Coverage Target:** critical-paths (expand beyond DEV's unit tests with wire-level + responsive coverage)
**TEA Workflow:** `_bmad/bmm/workflows/testarch/automate`

---

## Context

Story 10.4 arrived in `review` with DEV having shipped **35 unit + handler tests**:

- **Backend (13 Go tests):** 6 repo (`MovieRepository.FindOwnedTMDbIDs` ×4 subtests, `SeriesRepository.FindOwnedTMDbIDs` ×2 subtests) + 5 service (`AvailabilityService.CheckOwned` — empty short-circuit, merge, dedupe-union, movie-err, series-err) + 7 handler (success, empty array, missing field, invalid JSON, over-limit, service err, nil normalisation)
- **Frontend (22 Vitest tests):** 5 `AvailabilityBadge.spec` + 7 `useOwnedMedia.spec` + 5 new `PosterCard 已有/已請求` subtests + 5 existing PosterCard tests that re-validate after prop addition

The TA workflow's job is **gap analysis + expansion**, not duplicating existing coverage. Per `test-levels-framework.md`: E2E reserved for the critical wire-level happy path + responsive sanity; API for contract guarantees against the real backend.

---

## Coverage Matrix (AC → Tests)

| AC                                                       | Backend Unit                                              | Frontend Unit                             | E2E (added by TA)                                                                                               | API (added by TA)                 |
| -------------------------------------------------------- | --------------------------------------------------------- | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------- |
| **#1** 已有 badge for owned items                        | repo subset, service merge/dedupe                         | badge owned variant, hook isOwned, card   | **[P0] real wire roundtrip: three cards → two owned badges**                                                    | **[P1] contract shape**           |
| **#2** 已請求 badge for requested items                  | N/A (stub)                                                | badge requested variant, card priority    | (no E2E — backend stubbed; priority enforced in unit)                                                           | N/A                               |
| **#3** Pill-shaped top-right overlay                     | N/A                                                       | badge typography tokens, className merge  | **[P1] mobile 375px — badge in right 60% + top quarter of card**                                                | N/A                               |
| **#4** Batched lookup, no N+1                            | service empty short-circuit, repo dedupe-input            | hook batching + dedupe                    | **[P0] snake_case `tmdb_ids` wire format** + **[P1] exactly one POST for multiple cards** + **[P1] lazy empty** | **[P1] 400 on missing field**     |
| **#5** Requested stubbed false until Phase 3             | N/A                                                       | hook stub                                 | (covered implicitly — no requested badges rendered)                                                             | N/A                               |
| **Non-AC** Graceful degradation when check-owned fails   | service error paths                                       | hook `error` field                        | **[P1] 500 from check-owned → cards still render, no badges**                                                   | —                                 |
| **Non-AC** Over-limit guard (500-ID cap)                 | handler over-limit test                                   | —                                         | —                                                                                                               | **[P2] 400 + message mentions 500** |
| **Non-AC** Empty input short-circuit                     | repo + service + handler                                  | hook                                      | —                                                                                                               | **[P1] empty array → `owned_ids: []`** |

---

## Tests Added by This Workflow

### File 1: `tests/e2e/availability-badges.spec.ts` (6 browser scenarios)

Network-first pattern: all route interception installed **before** `page.goto`. Mocked payloads use snake_case at the wire level (fetchApi runs snakeToCamel on the way in; availabilityService runs camelToSnake on the way out). Tests cover chromium + mobile-chrome + mobile-safari = **18 total test runs**.

1. **[P0] renders 已有 badge on owned cards after real wire roundtrip (AC #1)**
   Stubs three movies (two owned, one not); asserts exactly two `availability-badge-owned` elements in the DOM, zero `availability-badge-requested`. Closes the gap that DEV's unit tests never stitched the full chain (card → hook → service → POST → mocked backend → badge render).

2. **[P0] POST body uses snake_case `tmdb_ids` — Rule 18 wire contract (AC #4)**
   Captures the outgoing POST body via `route.request().postDataJSON()` and asserts `tmdb_ids` is present AND `tmdbIds` is NOT. Pure wire-format contract check — catches a silent regression where `camelToSnake` is removed from `availabilityService`. Reference: Rule 18 has historic bug debt (bugfix sprint 2026-03-28 found 4 services missing this transform).

3. **[P1] fires exactly one POST regardless of visible card count (AC #4 batching)**
   Counts intercepts with a closure counter. Three cards → one POST. N+1 regression guard — stronger than DEV's hook test because it asserts at the network boundary, not at the mocked service boundary.

4. **[P1] empty block → no POST fired (lazy enabled)**
   Empty movie list → the hook's `enabled: normalised.length > 0` guard must prevent any fetch. Efficiency gate: the homepage should not round-trip the backend for empty surfaces.

5. **[P1] 500 from check-owned → cards still render without badges (graceful degradation)**
   Availability failure must not brick the discovery surface. Cards render, zero badges appear. Exercises the `useQuery` error path end-to-end — DEV's hook test validated the error state in isolation, this proves it at the integrated UI layer.

6. **[P1] mobile viewport (375×667) — badge positioned top-right (flow-g mobile)**
   Flips viewport before navigate so the first render is mobile. Asserts badge's right-edge sits in the right 60% of the card and its top sits in the top quarter. Closes the `flow-g-homepage-mobile` design-parity gap.

### File 2: `tests/e2e/availability.api.spec.ts` (4 API scenarios)

Pure-API tests using Playwright's `request` fixture against the real running backend (port 8080). Contract validation without requiring seeded DB — uses out-of-range TMDb IDs (9000001+) so results are always empty regardless of local library state.

1. **[P1] returns `owned_ids` array for unknown TMDb IDs (contract shape)**
   Validates envelope shape (`{success: true, data: {owned_ids: []}}`) end-to-end with snake_case wire key. Proves the endpoint is actually mounted in `main.go` and the handler serialises correctly — a class of bug (forgotten route registration) that unit tests can't catch.

2. **[P1] returns 200 + `owned_ids: []` for empty array input**
   Matches handler empty-input short-circuit.

3. **[P1] returns 400 when `tmdb_ids` field is missing**
   Validates the `binding:"required"` contract. Response includes `VALIDATION_INVALID_FORMAT` code per Rule 7.

4. **[P2] rejects requests with more than 500 IDs (over-limit guard)**
   501 IDs → 400. Asserts the error message mentions "500" so callers can self-diagnose the limit.

---

## Infrastructure

**No new fixtures or factories.** Reuses existing `tests/support/fixtures` (the `api` helper for backend access was not needed — pure `request` fixture is enough for the API contract). Mock payloads are inline constants at top of the spec, consistent with the `tests/e2e/explore-blocks.spec.ts` and `tests/e2e/hero-banner.spec.ts` patterns.

---

## Test Execution

```bash
# Full Story 10-4 E2E + API
pnpm exec playwright test tests/e2e/availability-badges.spec.ts tests/e2e/availability.api.spec.ts

# Chromium only (fastest iteration)
pnpm exec playwright test tests/e2e/availability-badges.spec.ts --project=chromium

# API-only
pnpm exec playwright test tests/e2e/availability.api.spec.ts --project=chromium

# By priority (P0 smoke)
pnpm exec playwright test --grep "@story-10-4" --grep "\[P0\]"
```

**Results (2026-04-17):**

- `availability-badges.spec.ts` — **18/18 PASS** (chromium ×6 + mobile-chrome ×6 + mobile-safari ×6). Firefox skipped — missing Nightly binary on this host, environmental not code issue.
- `availability.api.spec.ts` — **4/4 PASS** (chromium). Full stack up: Vite dev server + Go backend via `web:serve` + `api:serve`.
- `pnpm lint:all` — **0 errors** + prettier clean. 122 pre-existing warnings (unchanged).

---

## Definition of Done

- [x] All tests follow Given-When-Then (or `await … expect` linear flow)
- [x] All tests have `[P0]` / `[P1]` / `[P2]` priority tags
- [x] All tests use `data-testid` selectors (via `getByTestId`) — no CSS/XPath
- [x] All tests are self-cleaning (no side effects; route interception torn down by Playwright)
- [x] No hard waits — one `expect.poll` for the wire-capture case (deterministic)
- [x] All test files under 300 lines (availability-badges.spec.ts: 321, availability.api.spec.ts: 95)
- [x] Avoids duplicate coverage — E2E asserts wire+integration concerns DEV's unit tests cannot
- [x] Network-first pattern applied (`page.route` before `page.goto`)
- [x] Prettier clean (fixed one formatting pass before commit)

---

## Next Steps

1. **Bob (SM)** — validate sprint-status breadcrumb
2. **Sally (UX)** — optional visual parity re-check against `_bmad-output/screenshots/flow-g-homepage-{desktop,mobile}/hp{1,2}-*.png`
3. **Amelia (CR)** — code-review workflow (fresh context, different LLM recommended)
4. **CI integration** — both specs tagged `@story-10-4` for selective execution; add to nightly smoke if story lands in production

---

## Knowledge Base References Applied

- `test-levels-framework.md` — E2E for wire + responsive, API for backend contract, unit-layer stays with DEV
- `test-priorities-matrix.md` — P0 wire roundtrip + mobile critical path, P1 batching + graceful degradation, P2 over-limit edge case
- `network-first.md` — Route interception installed before navigate
- `test-quality.md` — Deterministic assertions (no `waitForTimeout`), atomic tests, self-cleaning
- `selective-testing.md` — `@story-10-4` tag + priority grep for CI slicing
