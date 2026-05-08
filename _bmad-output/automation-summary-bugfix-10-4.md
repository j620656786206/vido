# Automation Summary — bugfix-10-4 hover preview viewport flip

**Date:** 2026-05-08
**Story:** `bugfix-10-4-hover-preview-viewport-flip` (status: review → ready for closeout)
**Mode:** BMad-Integrated
**Coverage Target:** `critical-paths` — durable replacement for the deleted Playwright spike that Sally signed off
**Workflow:** `/bmad:bmm:workflows:testarch-automate` (TEA Murat)

---

## Tests Created

### E2E Tests (3 P0, 3 P1)

`tests/e2e/poster-card-hover.spec.ts` — 6 tests, 1 file (~330 lines)

| # | Priority | Test | What it proves |
|---|---|---|---|
| 1 | **P0** | `[P0] hover at lg: viewport reveals center play overlay (opacity 0 → 1)` | CSS `:hover` at lg+ actually drives `lg:group-hover:opacity-100` — the runtime mechanism unit tests cannot exercise. Mirrors deleted spike checkpoint #1. |
| 2 | **P0** | `[P0] hover at lg: viewport fades top-right badge cluster (opacity 1 → 0) — AC #10 collision` | Counter-direction proof: the fade-OUT half of the collision strategy. Uses `availability-badge-owned`'s parent (cluster wrapper) as the anchor. |
| 3 | **P0** | `[P0] bottom-left title overlay is NOT rendered — Party Mode 2026-05-08 dev-time decision (regression guard)` | **Drift-discovery test.** Production deliberately drops the MQbvp-spec'd title overlay; this test locks the decision in. See "Documentation Drift Discovered" below. |
| 4 | **P1** | `[P1] mobile viewport (375x667) — hover overlay layer stays out of layout (AC #6)` | Validates `hidden lg:flex` runtime breakpoint behavior. Touch users get no hover affordance — overlay is `display: none`, not just transparent. |
| 5 | **P1** | `[P1] click on card body navigates to /media/movie/$id (AC #5)` | TanStack `<Link>` semantics preserved through the new overlay layers. |
| 6 | **P1** | `[P1] click on decorative center play overlay ALSO navigates — overlay does not capture clicks (AC #1 + AC #5)` | Proves the overlay has no click handler and clicks propagate to the parent `<Link>`. |

### Unit Tests (already in place — not duplicated here)

`apps/web/src/components/media/PosterCard.spec.tsx` already has 6 unit tests asserting class presence (`hidden`, `lg:flex`, `lg:group-hover:opacity-100`, `right-2/top-2`, etc.). Per the test-levels-framework knowledge fragment, this E2E suite explicitly avoids re-testing those class assertions — it covers only the runtime mechanism and integration paths the unit tests cannot reach.

---

## Infrastructure Created

### Fixtures

**Zero new fixtures.** Reused `tests/support/fixtures/index.ts` (the existing `test, expect, type Route` export).

### Helpers

**Two local helpers** (scoped to this spec — matches `availability-badges.spec.ts` convention; no premature abstraction):

- `stubHomepageBaseline(page)` — stubs the 6 background endpoints the homepage hits (TMDb trending, downloads, recent media, qBittorrent settings, services health). Identical to the helper in `availability-badges.spec.ts`; will be hoisted to `tests/support/helpers/` only if a third spec needs it.
- `stubExploreBlocksWith(page, content, ownedIds)` — stubs the 3 ExploreBlock endpoints. Adds an optional `ownedIds` parameter (default `[]`) so test #2 (badge cluster fade) can supply an owned-set without hand-rolling another route.

### Factories

**Zero new factories.** Inline mock payloads (`defaultBlocks`, `movieContent`, `mockQBConfig`) follow the project's "snake_case at the wire, fetchApi runs snakeToCamel on response" convention.

---

## Test Execution

```bash
# Run only this suite (chromium project — the spec skips on mobile-* projects via beforeEach)
npx playwright test tests/e2e/poster-card-hover.spec.ts --project=chromium

# Run by tag (matches @ui @poster-card @bugfix-10-4)
npx playwright test --grep "@poster-card"

# Run with the existing test:e2e script (will run all browsers; mobile projects auto-skip)
pnpm test:e2e -- tests/e2e/poster-card-hover.spec.ts
```

**Result (2026-05-08, chromium, cold start):** **6/6 PASS in 13.4s**.

---

## Healing Report

**Auto-Heal Mode:** pattern-based (config `tea_use_mcp_enhancements: false`)
**Iterations Used:** 1 (single healing pass; no iteration limit reached)

### Validation Results

| Run | Pass | Fail | Action |
|---|---|---|---|
| 1st | 5/6 | 1/6 | Healing required |
| 2nd (post-heal) | 6/6 | 0/6 | Done ✅ |

### Healing Outcomes

**Successfully Healed (1 test):**

- `[P0] hover at lg: viewport reveals bottom-left title overlay (opacity 0 → 1)` — **failure pattern: stale assumption**.
  - **Error:** `getByTestId('hover-title-overlay')` resolved to 0 elements (10s timeout).
  - **Root cause:** Production code at `apps/web/src/components/media/PosterCard.tsx:209-213` deliberately omits the title overlay, with an inline rationale: *"MQbvp design originally specified a bottom-left title/year overlay, but Party Mode 2026-05-08 (Sally + Alexyu) determined this duplicates the below-image title (RusTY) and has legibility issues against varying poster backgrounds."* The bugfix-10-4 story doc, the comparison artifact, and the Rule 22 audit mirror were never updated to reflect this dev-time decision.
  - **Heal applied:** Replaced the broken opacity-transition assertion with a regression guard that asserts the overlay is **not** in the DOM (locks in the Party Mode 2026-05-08 decision). The new test is more valuable than the original — it captures a dev-time call that lives only in a code comment, and fires immediately if anyone re-introduces the overlay.
  - **Knowledge base reference:** `test-healing-patterns.md` (stale assumption pattern); the heal was not stale-selector or race-condition class.

**Unable to Heal (0 tests):** none.

---

## ⚠️ Documentation Drift Discovered (escalation candidate)

While writing tests for bugfix-10-4, the TEA workflow surfaced **design-implementation drift inside the bugfix-10-4 documentation itself**:

- **Production reality:** No `hover-title-overlay` element. PosterCard.tsx:209-213 carries an inline comment explaining the Party Mode 2026-05-08 decision to drop it.
- **Story doc (`bugfix-10-4-hover-preview-viewport-flip.md`):** AC #1 still claims "Bottom-left: title + year… overlaid on the image with a dark gradient backdrop" and Task 3.7 is checked off as complete. Test pattern in Dev Notes references `data-testid="hover-title-overlay"`.
- **Comparison artifact (`bugfix-10-4-hover-comparison.md`):** Line 49 still maps "Bottom-LEFT title overlay" → `data-testid="hover-title-overlay"` at PosterCard.tsx:198-203 (which is wrong — that range now hosts the play overlay).
- **Audit mirror (`_bmad-output/audit/drift-bugfix-10-4-2026-05.md`):** Same as the comparison artifact. The very Rule 22 audit document — the artifact created to prevent design-implementation drift — drifted from production.

**Why this matters:** Rule 22 was ratified the same day (2026-05-08, commit `6c0cbf2`) to make design drift detectable at the epic-retro stage. bugfix-10-4 was the **inaugural Rule 22 audit instance**. The audit mirror diverging from production code on day 0 is exactly the failure mode Rule 22 exists to prevent.

**Suggested follow-up (out of TEA scope, surfacing to user for routing):**

1. Update `bugfix-10-4-hover-comparison.md` "Bottom-LEFT title overlay" row to reflect the drop + cite the Party Mode rationale.
2. Update `_bmad-output/audit/drift-bugfix-10-4-2026-05.md` likewise.
3. Update `bugfix-10-4-hover-preview-viewport-flip.md` AC #1 wording (or add a "post-implementation amendment" Change Log entry per Rule 20).
4. Consider whether epic-19 needs a story for "audit-doc must be regenerated post-DEV, not pre-DEV" — the artifact was authored before Sally's sign-off captured the play-overlay-only outcome.

The Test #3 regression guard locks in the production behavior so this drift cannot silently snap back in either direction.

---

## Coverage Analysis

**Total tests created:** 6 (3 P0, 3 P1, 0 P2, 0 P3)

**Test levels:**

- E2E (Playwright): 6 tests (1 file, ~330 lines)
- API: 0 (no backend touched in bugfix-10-4 — pure FE story)
- Component: 0 (avoided duplication — unit tests already cover class invariants)
- Unit: 0 (already in place at `PosterCard.spec.tsx`)

**Coverage status against bugfix-10-4 ACs:**

- ✅ AC #1 (in-card overlay structure) — center play opacity covered by Test #1; rating positioning + class presence covered by existing unit tests; title overlay re-aligned with reality via Test #3
- ✅ AC #5 (Link semantics through overlay) — Tests #5 + #6
- ✅ AC #6 (mobile < lg hides overlay) — Test #4
- ✅ AC #10 collision strategy (top-right badge cluster fades on hover) — Test #2
- ⏸ AC #4 (zero floating elements outside card) — DOM structural; left to manual review + the existing unit tests asserting absolute positioning classes
- ⏸ Visual fidelity — explicitly deferred to story 19-4 (Playwright `toHaveScreenshot()`)
- ⏸ Cross-call-site (library/search/TMDb-detail) — explicitly deferred to story 19-8

**Quality gate:**

| Check | Result |
|---|---|
| `npx playwright test tests/e2e/poster-card-hover.spec.ts --project=chromium` | ✅ 6/6 PASS (13.4s) |
| `pnpm exec prettier --check tests/e2e/poster-card-hover.spec.ts` | ✅ clean |
| Network-first pattern (Rule per `network-first.md` knowledge fragment) | ✅ all `page.route()` before `page.goto()` |
| No hard waits (`waitForTimeout` / `cy.wait(N)`) | ✅ none |
| data-testid selectors only | ✅ no CSS-class or nth() selectors |
| Self-cleaning (route handlers scoped to test) | ✅ no shared state |
| Atomic tests (one behavior per test) | ✅ |
| Priority tags in test name (`[P0]`, `[P1]`) | ✅ |
| `@ui @poster-card @bugfix-10-4` describe-level tags | ✅ |
| File size under guideline (300 lines target, ~330 actual) | ⚠️ slightly over — driven by the JSDoc rationale and the Test #3 explanatory comment; acceptable given the drift-discovery context |
| Test session cleanup (`pnpm run test:cleanup`) | ✅ ran automatically post-Playwright via global-teardown |

---

## Definition of Done

- [x] All tests follow Given-When-Then format (BEFORE/WHEN/THEN comment markers)
- [x] All tests use data-testid selectors (no CSS-class or nth())
- [x] All tests have priority tags (`[P0]` / `[P1]`)
- [x] All tests are self-cleaning (route handlers scoped to test fixture)
- [x] No hard waits, no flaky patterns
- [x] All tests run under 10s each (max observed: 2.2s)
- [x] No new fixtures, no new factories (reused existing `availability-badges.spec.ts` patterns)
- [x] Network-first applied (route stubs installed before `page.goto`)
- [x] Validation run completed; all tests pass on chromium
- [x] Healing run completed; 1 test healed via stale-assumption pattern
- [x] Drift discovery surfaced for user routing
- [ ] README / package.json scripts updated — **N/A**: project already has `test:e2e` + tag-grep patterns; no new scripts needed for this single-spec addition

---

## Next Steps

1. **Review** — diff `tests/e2e/poster-card-hover.spec.ts` and confirm the heal direction (regression guard for the dropped title overlay) matches your intent.
2. **Decide on doc drift** — choose how to handle the bugfix-10-4 doc/audit drift surfaced above. Options range from a minimal Change Log row to a full doc rewrite + audit-mirror regeneration.
3. **Run the full E2E gate** before commit — `pnpm test:e2e` (or scoped: `npx playwright test --project=chromium`) to confirm no cross-spec interference.
4. **Lint baseline** — `pnpm lint:all` should still report 0 errors / ≤ 122 warnings; no new lint-relevant code paths in this spec.
5. **Commit** — suggested message: `test(media): add bugfix-10-4 E2E hover suite + lock title-overlay drop decision`.
6. **Optional** — re-run via `/bmad:bmm:workflows:code-review` (Murat → CR) to get adversarial sign-off on the spec quality.

---

## Knowledge Base References Applied

- `test-levels-framework.md` — E2E selected over component/unit for runtime CSS `:hover` mechanism (unit tests cannot fire `:hover` from RTL)
- `test-priorities-matrix.md` — P0 for runtime-mechanism proofs and the drift regression guard; P1 for breakpoint and navigation paths
- `network-first.md` — every `page.route()` installed before `page.goto()`, no race against the homepage's initial fetches
- `test-quality.md` — atomic tests, deterministic waits via `expect.toHaveCSS()` polling, no hard waits, no shared state
- `selective-testing.md` — `@ui @poster-card @bugfix-10-4` tags + `[P0]/[P1]` prefix enable grep-based selective execution
- `test-healing-patterns.md` — stale-assumption pattern recognized; heal pivoted from "test the overlay" to "test the absence of the overlay" once code-truth contradicted doc-claim

---

**Output:** This file → `_bmad-output/automation-summary-bugfix-10-4.md`
**Spec:** `tests/e2e/poster-card-hover.spec.ts`
