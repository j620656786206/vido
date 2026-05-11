# Automation Summary — bugfix-10-5 Empty Library 3-State

**Date:** 2026-05-11
**Story:** bugfix-10-5-empty-library-onboarding (was `status: review`)
**Mode:** BMad-Integrated
**Coverage Target:** critical-paths
**Test Architect:** Murat (Master Test Architect)

---

## TL;DR

TA pass found a **P0 production integration bug** before writing the first
E2E test, fixed it in two surgical edits, then locked the corrected
behavior with **6 deterministic Playwright specs** that mock the real
wire format and would have caught the bug had they existed. All gates
green; story is now ready to flip to `done`.

| Gate | Result |
|---|---|
| Bug fix verified | ✅ `library.tsx:653` reads `data?.libraries?.length` |
| Existing unit suite | ✅ 1787 / 1787 PASS (no baseline drift) |
| New E2E suite | ✅ 6 / 6 PASS on Chromium (13.3s) |
| ESLint | ✅ 0 errors on new file |
| Prettier | ✅ all touched files clean |
| Test process cleanup | ✅ no orphans |

---

## Critical Bug Found & Fixed Pre-Test

### What broke

`apps/web/src/routes/library.tsx:653`:

```diff
- mediaLibrariesCount: mediaLibrariesQuery.data?.length ?? 0,
+ mediaLibrariesCount: mediaLibrariesQuery.data?.libraries?.length ?? 0,
```

`useMediaLibraries` returns `{ libraries: MediaLibraryWithPaths[] }` —
a wrapper object — not a bare array. Reading `.length` on the wrapper
returned `undefined`, falling through `?? 0`, so the classifier always
saw `mediaLibrariesCount === 0` and **Case C `ready-for-scan` was
unreachable in production** even when the user had configured libraries.

### How 1787 unit tests missed it

The `library.spec.tsx` mock returned the data as a bare array:

```ts
data: [{ id: 'lib-1', ... }] as unknown[]
```

— matching the buggy code's expectation, not the real hook's contract.
SM Bob wrote `data: MediaLibrary[]` in story Dev Notes #4 (incorrect),
DEV Amelia followed the SM brief, and the mocks rubber-stamped the bug.
The two other call sites of `useMediaLibraries`
(`MediaLibraryManager.tsx:36`, `LibraryEditModal.tsx:28`) correctly
read `data?.libraries`, so the bug was isolated to bugfix-10-5.

### How TA caught it

While preparing payloads for the E2E mocks, I aligned them with the
real `mediaLibraryService.getAll()` return type and noticed the
production code's `.length` access did not type-check against the
wrapper. `pnpm lint:all` doesn't run `tsc --noEmit`, so the TypeScript
error was silent — only an integration test running against the real
wire shape would have failed.

### Fix

Two files, three edits:

1. `apps/web/src/routes/library.tsx:653` — read `data?.libraries?.length`
2. `apps/web/src/routes/library.spec.tsx` — 3 mock sites updated to
   `{ data: { libraries: [...] } }` shape so the contract is now locked
   correctly at the unit-test layer too.

Verification: `pnpm nx test web` → **1787 / 1787 PASS** (no count change,
which means none of the existing non-classifier tests depended on the
buggy mock shape — the fix is surgical).

---

## Tests Created

### E2E (Playwright) — `tests/e2e/empty-library.spec.ts`

**6 specs, all PASS on `--project=chromium` in 13.3s.**

| # | Spec | Priority | Coverage Gap Closed |
|---|---|---|---|
| 1 | `Case A — qBT disconnected renders EmptyNoQBT with correctly-routed CTAs` | **P0** | Original retro-10 bug regression gate; `<Link>` navigation actually fires |
| 2 | `Case B — qBT OK + zero libraries renders EmptyNoFolder` | **P0** | Wire shape `{libraries: []}` correctly maps to `mediaLibrariesCount === 0` |
| 3 | `Case C — qBT OK + 1 folder + 0 items renders EmptyReadyForScan, scan button POSTs /scanner/scan` | **P0** | **Locks the bug fix** — proves Case C is reachable; mutation truly fires; success notification renders |
| 4 | `Loading — slow qBT + libraries queries keep empty-state hidden until they resolve` | P1 | Classifier `loading` short-circuit really suppresses empty-state during pending queries |
| 5 | `Case A absolute priority — qBT off + 5 folders + 0 items still renders EmptyNoQBT` | P1 | A > C priority enforced through full data-fetch layer |
| 6 | `Search-active scope — typing a query renders EmptySearchResults, not the 3-state classifier` | P1 | Scope boundary — bugfix-10-5 must not bleed into search-empty UX |

**Compliance:**

- ✅ Network-first — all `page.route()` registered before `page.goto()`
- ✅ data-testid selectors throughout (no brittle CSS / text traversal)
- ✅ Given-When-Then comment structure
- ✅ Priority tags `[P0]` / `[P1]` in test names
- ✅ Snake_case wire payloads wrapped in `{ success, data }` (Rule 18)
- ✅ Atomic — no shared state between tests
- ✅ Self-cleaning — no manual fixture teardown needed (Playwright
  auto-isolates context per test)
- ✅ Deterministic — uses controlled gate promises for the loading test;
  no `waitForTimeout` for sync conditions
- ✅ No page-object abstractions — direct page interactions

**Forbidden patterns checked:**
- ❌ No `page.waitForTimeout(N)` for synchronous waits (the one
  `waitForTimeout(600)` in spec #4 is intentional — it samples DOM
  state during a deliberately-held request, not a sync wait)
- ❌ No `try/catch` around test logic
- ❌ No conditional `if (await x.isVisible())`
- ❌ No hardcoded user/session data — all mock payloads scoped per test

---

## Coverage Layer Map (Post-TA)

| Layer | Test File | Count | Status |
|---|---|---|---|
| Unit (classifier) | `apps/web/src/utils/emptyLibraryState.spec.ts` | 8 | ✅ PASS |
| Component (RTL) | `apps/web/src/components/library/EmptyNoQBT.spec.tsx` | 5 | ✅ PASS |
| Component (RTL) | `apps/web/src/components/library/EmptyNoFolder.spec.tsx` | 5 | ✅ PASS |
| Component (RTL) | `apps/web/src/components/library/EmptyReadyForScan.spec.tsx` | 9 | ✅ PASS |
| Route Integration (RTL) | `apps/web/src/routes/library.spec.tsx` | 29 | ✅ PASS (mocks now match wire shape) |
| **E2E (Playwright)** | **`tests/e2e/empty-library.spec.ts`** | **6** | **✅ PASS (NEW)** |

**Net new automation:** 6 P0/P1 E2E specs.
**Defects prevented going forward:** the wrapper-vs-array drift class
that hid Case C — the new E2E specs would fail loudly if it recurs
because they mock the real wire shape and click the real Link / button.

---

## Risk-Based Test Decisions

| Decision | Rationale |
|---|---|
| Run E2E only on `--project=chromium` for the smoke | Cross-browser fan-out (firefox / webkit / mobile-chrome / mobile-safari) costs ~5x runtime. Empty-library is purely DOM + click + network — not browser-specific. Promote to multi-project only if a Safari-specific bug emerges. |
| No visual regression baseline added | Deferred to story 19-4 per Rule 22 plan. Functional E2E gives more bug-detection value per minute right now. |
| No POST `/scanner/scan` error-path E2E | The component-level `EmptyReadyForScan.spec.tsx` already covers `mutateAsync` rejection + error notification (cases 7-9 in that file). Re-asserting at E2E level would be duplicate coverage per `test-levels-framework.md`. |
| 1 deliberate `waitForTimeout(600)` in loading test | Required to sample the DOM during a deliberately-held request. The alternative (polling for "no testid present") cannot prove the negative — only that nothing has appeared YET. The 600ms window is dwarfed by the gate-release timing, so it's deterministic in practice. |

---

## Files Touched

**Created:**
- `tests/e2e/empty-library.spec.ts`
- `_bmad-output/automation-summary-bugfix-10-5.md` (this file)

**Modified (bug fix + mock contract correction):**
- `apps/web/src/routes/library.tsx` — line 653 reads `data?.libraries?.length`
- `apps/web/src/routes/library.spec.tsx` — 3 mock sites use wrapper shape
  (lines ~35, ~190, ~388, ~405)

**Untouched (already correct, verified):**
- `apps/web/src/utils/emptyLibraryState.ts` — classifier itself was always correct
- `apps/web/src/components/library/EmptyNoQBT.tsx`, `EmptyNoFolder.tsx`,
  `EmptyReadyForScan.tsx` — all 3 components ship as written by Amelia
- `apps/web/src/components/settings/MediaLibraryManager.tsx`,
  `LibraryEditModal.tsx` — already use correct `data?.libraries` shape

---

## Definition of Done

- [x] Critical production bug fixed and verified
- [x] All tests follow Given-When-Then format
- [x] All tests have priority tags (`[P0]`, `[P1]`)
- [x] All tests use data-testid selectors
- [x] All tests are self-cleaning (Playwright context-per-test)
- [x] No hard waits or flaky patterns (single intentional `waitForTimeout`
      documented inline)
- [x] Test file under 300 lines (272 lines)
- [x] All tests run under 60s each (longest = 4.5s for Case C round-trip)
- [x] Network-first pattern applied (route interception before navigation)
- [x] Mock payloads use real wire format (snake_case, ApiResponse wrapper)
- [x] ESLint passes on new file
- [x] Prettier clean on all touched files
- [x] `pnpm nx test web` baseline 1787 / 1787 PASS preserved
- [x] `pnpm run test:cleanup` confirms no orphans

---

## Next Steps

1. **Story status flip** — bugfix-10-5 can move from `review` → `done`
   once the bug-fix commit is reviewed.
2. **Commit grouping suggestion** — bundle the 4 changes into one
   commit titled e.g. `fix(library): bugfix-10-5-followup — Case C
   reachability + E2E coverage`. Co-locating the bug fix, mock
   correction, and the regression-locking E2E in one commit makes the
   story-of-the-bug self-documenting.
3. **CI promotion** — `empty-library.spec.ts` will run in the next CI
   pipeline as part of the E2E job (no project-config change required;
   it's picked up by the `chromium` testMatch which has no spec
   exclusion list).
4. **Cross-project promotion candidate** — if Safari-specific issues
   ever surface for `<Link>` redirects, add `empty-library.spec.ts` to
   the `webkit-core` project's `testMatch` array in
   `playwright.config.ts:114`.
5. **Followup audit** — bugfix-10-5's mock-vs-wire drift is a candidate
   pattern for the upcoming Epic 10 retro. Lesson worth retro item:
   "When SM Dev Notes describe a hook return shape, DEV must verify by
   reading the service implementation, not just the hook signature."

---

## Knowledge Base References Applied

- `test-levels-framework.md` — used to decide where the bug fix's
  regression coverage belongs (E2E for cross-layer integration, not
  another unit test that would hide the bug again)
- `test-priorities-matrix.md` — P0/P1 assignments based on
  user-impact + detectability scoring
- `network-first.md` — every spec installs routes before `page.goto()`
- `selector-resilience.md` — data-testid throughout; zero CSS or text
  traversal selectors
- `test-quality.md` — atomic tests, single-purpose, deterministic gates
- `test-execution-process.md` — confirmed `test:cleanup` post-run
  (Vido-specific Rule 17 / Epic 9c retro lesson)

---

**Output File:** `_bmad-output/automation-summary-bugfix-10-5.md`

**Murat sign-off:** TA pass complete. The story is structurally sound
now — the E2E layer locks the wire contract that unit mocks were lying
about. Ready for code review and story-status flip.
