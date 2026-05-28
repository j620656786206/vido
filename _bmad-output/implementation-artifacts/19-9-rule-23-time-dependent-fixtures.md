# Story 19.9: Rule 23 — Time-Dependent Component Fixture Stability (Visual-Harness Class-Level Hardening)

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- This story is the epic-19 post-capstone hardening: bundles (a) immediate fix for the 19-8 PR #8 visual-regression blocker (`components/library-recently-added/default` baseline stale), (b) author Rule 23, (c) ESLint rule `local/time-dependent-fixture-stability` enforce, (d) audit + migrate all components/ files that read `Date.now()` / `new Date()`, (e) SM /create-story template extension per Sally. Born of party-mode consensus 2026-05-26 (Murat + Sally + Winston): the symptom is one stale fixture, the class is "any component reading the wall clock without an injectable clock = visual-baseline time bomb". -->
<!-- @contract-v1 on AC #1–#5 (Rule 23 spec / ESLint rule API / audit-doc shape / clock-mock pattern / dual-state baseline convention — future Rule 22 retros + downstream visual-baseline stories may grep). -->
<!-- 🔗 AC Drift: NONE — this story does not modify any prior story's observable AC behavior; it introduces a new Rule + new ESLint rule + new test pattern; the library-recently-added fixture change is a NEW state addition, not a contract-shape change to any prior 19-4/19-4b AC (those ACs talk about platform-suffix + baseline path convention, both preserved). · 📎 Contract Stamps: NEW (v1×5; consumes 19-3 [@contract-v3] ESLint-rule plugin pattern + 19-4 [@contract-v1] AC #1 visual project + 19-4b [@contract-v1] AC #5 platform-suffix — see Dev Notes acks) · 🔒 Rule 7: N/A (no Go errors; pure FE/test/lint-rule/docs) · 🎨 UX: Sally + Murat dual-classifier (Sally owns the "what state matters visually" call per component; Murat owns the "what's testable + worth a baseline" risk call; Amelia executes). -->
<!-- markers-block-end -->

## Story

As the **stewards of the visual-regression harness** (Sally for the design-state coverage decision, Murat for the test-architecture decision, Amelia for the implementation, Bob for the SM template + sprint propagation),

I want a one-pass class-level hardening that (a) introduces project-context.md **Rule 23: Time-Dependent Component Fixture Stability** + a custom ESLint rule `local/time-dependent-fixture-stability` to enforce it, (b) audits every `apps/web/src/components/**/*.{ts,tsx}` file for the time-bomb pattern (wall-clock reads against fixed-date fixtures), (c) introduces a Playwright clock-mock helper and migrates `library-recently-added` (the current 19-8 PR-blocker) to dual-state baselines (`recent` + `stale`), (d) migrates or documents exemptions for every other time-bomb candidate the audit surfaces, and (e) extends the SM `/create-story` template so future stories MUST list ≥2 fixture state baseline paths when their component reads the wall clock,

**so that** the 19-8 PR #8 visual-regression check goes green (unblocking merge → epic-19 capstone closes), **AND** the same class of bug — silently re-occurring every time a wall-clock-window boundary crosses — is permanently inoculated against, with the inoculation enforced in CI (ESLint rule) instead of relying on human review discipline (which already missed it through 19-4, 19-4b, AND 19-8's full sweep).

## Acceptance Criteria

1. [@contract-v1] **Rule 23 authored in `project-context.md`** — full rule text inserted under `## 📋 MANDATORY Rules` after Rule 22 (L723 region), titled **"Rule 23: Time-Dependent Component Fixture Stability"**. Rule body MUST specify: (a) trigger — any file under `apps/web/src/components/**` whose source body contains an unwrapped `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()` call (or imports a utility that does, transitively — judgment call documented per file in AC #3 audit doc); (b) requirement — the component must either inject `clock` (or `now`) as a prop/context so callers can deterministically control time, OR its gallery fixture(s) in `apps/web/src/routes/test/-gallery.fixtures.tsx` MUST be paired with a Playwright `page.clock.setFixedTime(...)` call in `tests/visual/components.visual.spec.ts` (per AC #4 helper); (c) coverage — components with time-dependent state branching MUST be visually baselined in ≥2 fixture states covering both branches (the 19-8 PR root cause: `library-recently-added` only baselined `isWithin7Days = true`, losing the `false` branch); (d) exemption — Hooks/services that never render are exempt (only `components/**` UI surfaces in scope); (e) precedent citation — party-mode 2026-05-26 (Murat + Sally + Winston) + bugfix-19-8-time-bomb-fixtures origin (19-8 PR #8 CI failure 2026-05-26). Last-Updated header gets a 19-9 entry mirroring the 19-5/19-6/19-7/19-8 entry format. Rule 22's tooling block gets one cross-reference sentence noting Rule 23 is the time-dimension counterpart.

2. [@contract-v1] **Custom ESLint rule `local/time-dependent-fixture-stability` exists and is wired in `eslint.config.mjs`** — new file `apps/web/src/eslint-rules/time-dependent-fixture-stability.js` (mirror the structural shape of `apps/web/src/eslint-rules/implements-pen-node-id.js` — JSDoc-style header, `meta.type: 'problem'`, `messages` map, single `create(context)` AST visitor). Rule behaviour: when the file is under `apps/web/src/components/**` (scoping done in `eslint.config.mjs` flat-config `files`/`ignores`, NOT in rule logic — same pattern as 19-3 rule), and the file's AST contains a `MemberExpression` matching `Date.now` (or `NewExpression` with `Date` callee and zero args) anywhere in source body, error with message ID `time-bomb-detected` UNLESS the file's leading-comment header carries one of these accepted forms: (i) `// Clock-mocked: gallery fixture {fixture-id} uses page.clock.setFixedTime` — declares the component is harness-protected via dual baselines (the canonical form), (ii) `// Clock-injected: component accepts `clock` prop; no fixture-side mock needed` — declares the component itself dodges the time-bomb via DI, (iii) `// Time-bomb-exempt: <one-line rationale>` — explicit acknowledged exemption (e.g. a debug-only display that doesn't affect visual baselines). Rule spec file `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` MUST have ≥12 tests covering: each accepted form valid case, each form mis-placed (after first statement) invalid case, `Date.now()` without any header invalid, `new Date(...)` constructor invalid, no Date use → no header needed valid, scoping (file outside `components/**` → rule no-ops), comment-form syntax variants (trailing whitespace, mixed case in marker keyword). Rule is exported via the same plugin entry as `implements-pen-node-id` (19-3 plugin pattern reused).

3. [@contract-v1] **Audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`** — created in this story, structure:
   - `## Header` — scan date, scan agent (Amelia, party-mode-blessed), total `components/**/*.{ts,tsx}` files scanned, time-bomb candidates found, classification breakdown, top-line conclusion.
   - `## Methodology` — grep pattern used (`grep -rln -E 'Date\.now\(\)|new Date\(' apps/web/src/components/ --include='*.tsx' --include='*.ts' --exclude='*.spec.*'`), how each hit was triaged (pure render-affecting wall-clock read vs. one-off timing utility vs. test helper), Rule 23 conformance check.
   - `## Time-bomb candidates table` — one row per candidate file, columns: `File`, `Wall-clock call site`, `Visual-state impact (Yes/No/N/A)`, `Existing fixture state`, `Rule 23 disposition` (clock-mocked / clock-injected / time-bomb-exempt / migrated-in-this-story / pre-existing-safe-via-fixed-date).
   - `## Migration log` — for each migrated component, the before/after fixture diff snippet + the new baseline paths.
   - `## Exemption ledger` — for each exemption, the rationale + a reviewer (Sally OR Murat) initials.
   - `## Cross-link to 19-3 audit` — note that this audit complements the 19-3 ESLint rule + 19-8 sweep; together they cover the *spatial* (design-vs-code) and *temporal* (clock-window) dimensions of visual-baseline correctness.

4. [@contract-v1] **Playwright clock-mock helper + pattern documented** — introduce a small helper in `tests/visual/clock-mock.ts` (one named export, e.g. `withFixedClock(page, isoTimestamp)` that wraps `page.clock.install({ time: ... })` OR `page.addInitScript(now => { Date.now = () => now; }, time)` — Amelia picks based on Playwright version capability check at impl time; document the choice rationale in Dev Agent Record). `tests/visual/components.visual.spec.ts` MUST be extended so a fixture row may declare `clockTime: '2026-05-15T00:00:00Z'` (per fixture in `-gallery.fixtures.tsx`), and when present the spec calls the helper before `toHaveScreenshot()`. Fixtures without `clockTime` continue unchanged (backward-compatible — all 122 existing fixtures unaffected). The helper API + the fixture-row extension shape are the contract surface stamped here; downstream Rule 23 migrations consume this API.

5. [@contract-v1] **`library-recently-added` dual-state baselines committed** — `apps/web/src/routes/test/-gallery.fixtures.tsx` gets the `library-recently-added` fixture (currently single row at L~2256) split into TWO state rows: (i) `library-recently-added/recent` — `createdAt` derived OR `clockTime: '2026-05-15T00:00:00Z'` paired with existing `createdAt: '2026-05-12T08:00:00Z'` (so `isWithin7Days` = true → green "新增" badge visible); (ii) `library-recently-added/stale` — same fixture data BUT `clockTime: '2026-05-30T00:00:00Z'` (so `isWithin7Days` = false → no badge). Each state gets its own `-darwin` AND `-linux` baseline under `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{recent,stale}/`. The current stale `default` baseline is removed (it represented neither state cleanly). `pnpm run test:visual` MUST go green on both platforms after this AC lands. The component file `apps/web/src/components/library/RecentlyAdded.tsx` gets the AC #2 Rule 23 header `// Clock-mocked: gallery fixture library-recently-added uses page.clock.setFixedTime` placed BEFORE the existing `// Design ref:` header (both headers coexist — they're orthogonal rules).

6. **Audited components — migrate or exempt** — for every time-bomb candidate the AC #3 audit surfaces (estimated 4–5 files based on initial scan: `library/RecentlyAdded.tsx`, `retry/CountdownTimer.tsx`, `retry/RetryNotifications.tsx`, `parse/useParseProgress.ts`, `metadata-editor/MetadataEditorDialog.tsx` — verify exact set at impl time), produce ONE of three outcomes per file:
   - **migrated** — fixture pair + clock-mock + dual baselines (per AC #4/#5 pattern);
   - **clock-injected** — component refactored to take `clock` prop, fixture provides a stable injected clock, single baseline OK;
   - **time-bomb-exempt** — documented in audit doc with reviewer initials (Sally for visual-state calls, Murat for test-architecture calls).
   
   Files that are *hooks/services* (e.g. `useParseProgress.ts`) and never directly rendered are out of scope — Rule 23 is component-only. Pre-existing safe patterns (e.g. `CountdownTimer`'s fixed-date `nextAttemptAt: '2020-...'` trick documented in 19-4b) get classification `pre-existing-safe-via-fixed-date` in the audit and a Rule 23 header added retroactively for ESLint conformance; whether to upgrade them to dual-state baselines is a Sally + Murat per-component call (default: leave the working pattern alone; upgrade only when there's actual coverage value).

7. **SM `/create-story` template extended** — `_bmad/bmm/workflows/4-implementation/create-story/template.md` OR the SM agent's create-story guidance (`_bmad/bmm/agents/sm.md` activation step OR the instructions.xml — whichever is the canonical SM author-time prompt; investigate at impl time and pick the lowest-friction insertion point) gets a NEW Dev Notes sub-section template: **"Time-dependent visual coverage"** with prompts: (a) "Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()`?", (b) "If yes — list the fixture state baseline paths the dev MUST capture (≥2 per Rule 23 AC #1d).", (c) "If no — explicitly state 'N/A — no wall-clock-reading components touched'." This is the Sally request from party-mode 2026-05-26 — turn "evaluator's discretionary checklist" into "story-template enforced field". Existing in-flight stories are NOT retrofitted (Rule 20 forward-only); the field applies to stories created AFTER 19-9 closes.

8. **Regression + framework hygiene** — at close: `pnpm lint:all` 0 errors / ≤ baseline warnings (baseline at 19-8 close = 122; this story should not add warnings — new rule does its own errors-only enforcement). The new `local/time-dependent-fixture-stability` rule MUST be green across all in-scope `components/**` files after Task 5 migrations + exemptions land. `pnpm nx test web` + `pnpm nx test api` pass. `pnpm test:e2e --list` count unchanged. `pnpm run test:visual` GREEN on both `-darwin` and `-linux` (no pre-existing failures permitted at close — the `library-recently-added/default` flake that motivated this story MUST be resolved by AC #5; the existing backlog entry `preexisting-fail-visual-recently-added-stale-baseline` is removed from sprint-status.yaml). `pnpm run test:cleanup` no orphans. `ux-design.pen` `git status` clean (read-only Pencil MCP if used — this story shouldn't need design-side reads, but if Sally requests one for a state-judgment call, MUST stay read-only).

9. **Scope boundaries** — modified files: `project-context.md` (Rule 23 + Rule 22 cross-ref + Last Updated); `apps/web/src/eslint-rules/time-dependent-fixture-stability.js` (NEW); `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` (NEW); `apps/web/src/eslint-rules/index.js` OR equivalent plugin entry (EXTENDED to export the new rule); `eslint.config.mjs` (EXTENDED to wire the new rule with `apps/web/src/components/**` scoping); `tests/visual/clock-mock.ts` (NEW); `tests/visual/components.visual.spec.ts` (EXTENDED for `clockTime` fixture-row support); `apps/web/src/routes/test/-gallery.fixtures.tsx` (EXTENDED — `library-recently-added` split + N other fixtures per Task 5); `apps/web/src/components/**/*.tsx` (Rule 23 headers added per Task 5, N files — header LINE only; ZERO logic edits unless a component opts for `clock`-injection refactor in which case the diff stays minimal); `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{recent,stale}/*.png` (NEW — both platforms); `_bmad-output/audit/time-bomb-fixtures-2026-05.md` (NEW); `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-9 status + remove the `preexisting-fail-visual-recently-added-stale-baseline` backlog entry per AC #8). NO edits under `apps/api/`, `playwright.config.ts` (the helper goes under `tests/visual/`, not the config), `package.json` (no new deps — `page.clock` is built-in to recent Playwright, `addInitScript` is fallback). `ux-design.pen` **untouched** (read-only iff Sally needs a state-call reference).

10. **PR #8 unblocking** — once 19-9 is merged to main, the 19-8 PR's `Visual Regression / PR` check passes when rebased on top of main (because the `library-recently-added/default` baseline that was stale is now replaced by clock-mocked `recent` + `stale` baselines, and the spec walks them per AC #4 extension). Verify by: (a) merge 19-9 → main; (b) on PR #8 branch run `git rebase origin/main`; (c) push; (d) confirm `gh pr checks 8` shows Visual Regression PR ✅. The rebase + push are 19-8's owner's call (not this story's responsibility), but this AC enforces that 19-9's design MUST make that rebase mechanical (no AC drift in 19-8). If for any reason 19-9 lands but PR #8 still fails for an unrelated visual reason, that's a new finding — file separately.

## Tasks / Subtasks

- [x] Task 1: Author Rule 23 in `project-context.md` (AC: #1)
  - [x] Insert Rule 23 body under `## 📋 MANDATORY Rules` after Rule 22 (L723 region), full text per AC #1 a-e.
  - [x] Add one cross-reference sentence to Rule 22's tooling block linking to Rule 23 as the time-dimension counterpart.
  - [x] Add Last-Updated header entry (2026-05-26): one paragraph mirroring 19-5/19-6/19-7/19-8 format — origin (party-mode 2026-05-26 + 19-8 PR #8 CI failure), what landed (Rule 23 + ESLint rule + audit doc + dual-state baseline pattern), upstream consumed (19-3 v3, 19-4 v1, 19-4b v1), what's stamped (5×v1).
  - [x] Commit message: `docs(19-9): Rule 23 — time-dependent fixture stability [@contract-v1]`.

- [x] Task 2: Time-bomb scan + audit doc (AC: #3)
  - [x] Run the AC #3 Methodology grep: `grep -rln -E 'Date\.now\(\)|new Date\(' apps/web/src/components/ --include='*.tsx' --include='*.ts' --exclude='*.spec.*'`. Expected ~5 candidates per initial scan (RecentlyAdded, CountdownTimer, RetryNotifications, useParseProgress, MetadataEditorDialog — verify).
  - [x] For each candidate, classify per AC #6: visual-state impact (Yes/No/N/A), existing fixture state, disposition (migrated / clock-injected / exempt / pre-existing-safe).
  - [x] Hooks/services that never render → mark out-of-scope (Rule 23 is component-only); record in audit doc `## Exemption ledger` with Murat's initials (test-arch call).
  - [x] Write `_bmad-output/audit/time-bomb-fixtures-2026-05.md` per AC #3 structure. Commit message: `docs(19-9): time-bomb fixtures audit doc`.

- [x] Task 3: Playwright clock-mock helper + spec extension (AC: #4)
  - [x] Check Playwright version in `package.json` — if ≥ 1.45 (`page.clock` API GA), use `withFixedClock(page, iso)` wrapping `page.clock.install({ time: new Date(iso) })`; else fallback to `page.addInitScript` overriding `Date.now`. Document the choice in Dev Agent Record.
  - [x] Create `tests/visual/clock-mock.ts` with the helper + JSDoc explaining when/why to call it.
  - [x] Extend `tests/visual/components.visual.spec.ts` to read an optional `clockTime` field per fixture and call the helper before `toHaveScreenshot()`. Backward-compatible: fixtures without `clockTime` skip the helper call.
  - [x] Add a one-fixture smoke-test (any new fixture row with `clockTime`) to confirm the wiring works before Task 5's bulk migration.
  - [x] Commit message: `feat(19-9): Playwright clock-mock helper + spec extension`.

- [x] Task 4: ESLint rule `local/time-dependent-fixture-stability` + spec + wire (AC: #2)
  - [x] Create `apps/web/src/eslint-rules/time-dependent-fixture-stability.js` — JSDoc header + `meta.type: 'problem'` + `messages.time-bomb-detected` + AST visitor (MemberExpression `Date.now`, NewExpression `Date()`). Mirror 19-3 `implements-pen-node-id.js` structural style.
  - [x] Create `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` — ≥12 tests per AC #2 enumeration.
  - [x] Wire the rule in `eslint.config.mjs` — extend the existing local-rules plugin block OR add a new one if architecture demands; scope to `apps/web/src/components/**/*.{ts,tsx}` excluding `*.spec.*` + `index.ts` barrels (same scoping as 19-3 rule for consistency).
  - [x] Verify `pnpm exec eslint apps/web/src/components/` → fails on candidates without Rule 23 header (drives Task 5).
  - [x] Commit message: `feat(19-9): ESLint rule local/time-dependent-fixture-stability [@contract-v1]`.

- [x] Task 5: Migrate `library-recently-added` + remaining candidates (AC: #5, #6)
  - [x] Split `library-recently-added` fixture into `recent` + `stale` state rows in `-gallery.fixtures.tsx` (use `clockTime` field per AC #4 extension; `recent` uses 2026-05-15 (3 days after createdAt = within window); `stale` uses 2026-05-30 (18 days after createdAt = outside window)).
  - [x] Capture both baselines: `pnpm run test:visual --update-snapshots` (locally — `-darwin`) and pushed to PR (CI will produce `-linux`). Remove the old `default` baseline PNG.
  - [x] Add Rule 23 header `// Clock-mocked: gallery fixture library-recently-added uses page.clock.setFixedTime` to `apps/web/src/components/library/RecentlyAdded.tsx` (above existing `// Design ref:` line; both coexist).
  - [x] For each other Task 2 audit candidate: apply migrated / clock-injected / exempt outcome per AC #6. Each gets a Rule 23 header added. Each migrated component gets fixture pair + dual baselines (per AC #5 pattern). Each exemption gets audit-doc rationale + reviewer initials.
  - [x] Per-component commits: `feat(19-9): migrate {component-name} to Rule 23 dual-state baselines` OR `docs(19-9): exempt {component-name} per Rule 23 [{Sally|Murat}]`.

- [x] Task 6: SM template extension + close-out regression (AC: #7, #8, #9, #10)
  - [x] Identify canonical SM `/create-story` author-time prompt insertion point (template.md vs instructions.xml vs sm.md). Add the "Time-dependent visual coverage" Dev Notes sub-section template per AC #7.
  - [x] Smoke-test: confirm the template addition renders sensibly by mentally walking through "if Bob ran /create-story for a new component-touching story tomorrow, would the prompt appear?".
  - [x] Run regression: `pnpm lint:all` (0 errors) · `pnpm nx test web` · `pnpm nx test api` · `pnpm test:e2e --list` (count unchanged) · `pnpm run test:visual` (BOTH platforms GREEN — this is the AC #8 gate, since the failing baseline that drove 19-9's creation MUST be resolved by AC #5) · `pnpm run test:cleanup` (no orphans).
  - [x] Remove backlog entry `preexisting-fail-visual-recently-added-stale-baseline` from sprint-status.yaml (per AC #8 — that flake is the very thing this story fixed).
  - [x] Update 19-9 sprint-status: ready-for-dev → in-progress (Task 1 start) → review (Task 6 close).
  - [x] Confirm AC #10 path: document in Dev Agent Record the exact rebase command 19-8's owner needs to run; do NOT execute the rebase from this story (boundary).
  - [x] Commit message: `chore(19-9): close-out — regression + sprint-status hygiene + AC #10 handoff`.

## Dev Notes

### Why this story exists / how it relates to epic-19

- **This is the epic-19 post-capstone class-level fix.** 19-8 was the capstone (full sweep of design-vs-code drift across 131 files; outcome: 0 material drift, hypothesis disproven). 19-9 is the *follow-on* hardening: the same 19-8 PR surfaced a CI failure (`library-recently-added` stale baseline) that turned out NOT to be a design-vs-code drift (which is what 19-8 hunted) but a *time-vs-code drift* — same fixture, different clock. Per party-mode 2026-05-26 (Murat + Sally + Winston with Alexyu): treating this as one fixture rebless is treating the symptom; the class is "any component reading the wall clock without an injectable clock = visual-baseline time bomb". Rule 23 makes it enforceable.
- **The 19-8 PR (#8) is currently blocked on this** — the `Visual Regression / PR` check fails on `library-recently-added/default` baseline. The backlog entry `preexisting-fail-visual-recently-added-stale-baseline` was filed per Epic 9c retro AI-2 option 2 routing it to 19-4c. Party-mode upgraded the scope from 19-4c (treat-symptom: fixture date stabilization) to 19-9 (treat-class: Rule + ESLint + audit + helper). Renamed accordingly.
- **Three lessons compounded** to make this rule necessary:
  1. **19-4b CountdownTimer** taught half the lesson — fixed `nextAttemptAt: '2020-...'` to escape the moving-window problem. We learned "pin the date" but didn't generalize to "any wall-clock read in a component needs harness protection".
  2. **19-4/19-4b baseline capture** only baselined ONE state per fixture. Sally's party-mode admission: "I only saw one state when I approved; I didn't ask 'what does the OTHER branch look like?'" The harness was coverage-thin.
  3. **19-8 sweep** examined 131 files for design-vs-code drift and found zero — BUT didn't grep for `Date.now()` (it wasn't asking that question). The PR's CI failure revealed the time-dimension gap.
- **Decision: bundle 5 deliverables in one story, not 5 sub-stories.** Reasons: (a) the deliverables are tightly coupled — Task 2's audit feeds Task 5's migrations, Task 3's helper feeds Task 4's rule, Task 4's rule drives the per-file Task 5 migrations; (b) split would require running the scan three times; (c) party-mode consensus was unambiguous about the single-story framing — Winston specifically called out "治標+治本綁同一故事". Caveat: if Task 5's migration count blows past ~8 components, in-flight re-cut to 19-9b for the remaining migrations is precedent-blessed (19-4 → 19-4b re-cut pattern).

### Architecture / constraints — read before implementing

- **Pure FE + test + lint-rule + docs.** 0 Go code. 0 logic edits to components except optional `clock`-injection refactors (which should stay minimal — header + signature change + propagate `clock` prop one level deep). Cross-stack split check: backend tasks = 0, frontend logic tasks = 1 (clock-mock helper) + 1 (ESLint rule) = 2 → well under the >3 threshold. Single story is correct.
- **Playwright clock API** — newer Playwright (≥1.45) ships `page.clock.install({ time, shouldAdvance: false })` which is the cleanest API. If our pinned version is older, `page.addInitScript(time => { const orig = Date.now; Date.now = () => time; }, isoString.getTime())` works. Amelia: check `package.json` → `@playwright/test` version → pick path. Document in Dev Agent Record. Either way, helper lives at `tests/visual/clock-mock.ts` with one exported function.
- **ESLint rule structural template** — mirror `apps/web/src/eslint-rules/implements-pen-node-id.js` (19-3 rule). Same plugin pattern, same flat-config wiring style, same spec-file approach (`*.spec.ts` co-located, run via `pnpm nx test web`). Scoping for `components/**` lives in `eslint.config.mjs`, NOT in rule body. The rule's detection logic is AST-based (not regex on source text — robust against comments/strings containing "Date.now" literally).
- **The accepted Rule 23 marker forms** (AC #2) are designed to be additive to existing Rule 21 markers — they coexist (different rules, different lines, no conflict). Order convention: Rule 23 marker FIRST (Clock-mocked / Clock-injected / Time-bomb-exempt), then Rule 21 marker (Implements / Design ref / utility / screen-section). Two-line header. Rule 21's ESLint rule (`local/implements-pen-node-id`, 19-3) sees Rule 23 lines as "extra leading comments" and ignores them (per its "leading comment" definition).
- **Fixture-row `clockTime` extension** — backward-compatible. `tests/visual/components.visual.spec.ts` reads each fixture row; if `clockTime` field is present, call `withFixedClock(page, clockTime)` before screenshot; else skip. All 122 existing fixtures unaffected unless their migration target adds the field.
- **Dual-state baseline naming** — `tests/visual/components.visual.spec.ts-snapshots/components/{gallery-id}/{state}-visual-{platform}.png`. The `{state}` segment becomes a real path component (currently it's `default` or `hover`/`focus`/`open` per the 19-4 4-state set). New states like `recent` / `stale` slot into the same naming scheme — the spec walks them by listing the fixture rows.
- **Baseline-bootstrap for `-linux`** — same as 19-4b's CI bootstrap: first push to main triggers the Visual Regression `Main` job which auto-bootstraps any missing `-linux` baselines via a `requires-manual-review` PR (Sally re-engagement gate per 19-4b Task 5 ruling). Don't try to fake `-linux` baselines locally — let CI do it. AC #5 calls for both platforms; Amelia commits `-darwin` locally, CI fills `-linux` post-merge.
- **The `preexisting-fail-visual-recently-added-stale-baseline` backlog entry** must be removed from sprint-status.yaml ONLY after AC #5 lands and Task 6 regression confirms green. Removing it pre-fix would leave a worse trace ("we knew about it and forgot it").
- **Audit doc lifetime** — `time-bomb-fixtures-2026-05.md` is durable (like `drift-19-3-2026-05.md` and `drift-sweep-2026-05.md` from 19-3 / 19-8). Future Rule 22 retros may grep for new time-bomb candidates added since 19-9; the ESLint rule prevents them from landing but the audit doc is the historical "we explicitly thought about this" record.
- **No new dependencies.** Playwright clock API is built-in; ESLint rule plugin pattern is already in use; no `package.json` change.

### Project Structure Notes

- **New files:**
  - `_bmad-output/audit/time-bomb-fixtures-2026-05.md` (audit doc, AC #3)
  - `apps/web/src/eslint-rules/time-dependent-fixture-stability.js` (rule, AC #2)
  - `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` (rule spec, AC #2)
  - `tests/visual/clock-mock.ts` (helper, AC #4)
  - `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{recent,stale}/*.png` (new baselines, AC #5)
- **Modified files:**
  - `project-context.md` (Rule 23 + Rule 22 cross-ref + Last Updated, AC #1)
  - `eslint.config.mjs` (wire new rule, AC #2)
  - `apps/web/src/eslint-rules/index.js` OR plugin entry (export new rule, AC #2)
  - `tests/visual/components.visual.spec.ts` (`clockTime` fixture-row support, AC #4)
  - `apps/web/src/routes/test/-gallery.fixtures.tsx` (fixture splits per Task 5)
  - `apps/web/src/components/**/*.tsx` (Rule 23 headers per Task 5, ~4-5 files; ZERO logic edits unless component opts for clock-injection)
  - `_bmad/bmm/workflows/4-implementation/create-story/template.md` OR equivalent (Time-dependent visual coverage section, AC #7)
  - `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-9 status + remove pre-existing-fail entry per AC #8)
- **Read-only:** `ux-design.pen` (Pencil MCP read-only iff Sally requests a state-call reference — likely not needed; this story is harness-side, not design-side).
- **Out of scope:** any `apps/api/` change (Go is untouched); any `playwright.config.ts` change (helper is a separate file under `tests/visual/`); any `package.json` change (no new deps); any baseline regen for fixtures NOT in the Task 2 audit candidate set (the 19-4b queue stays committed source-of-truth); any retroactive 19-4c-style bulk fixture-date stabilization (the ESLint rule catches new instances; retroactive cleanup is in-scope ONLY for AC #6 candidates, NOT for any fixture that happens to read a date).

### Testing standards (project-context.md)

- **Rule 9 (test co-location):** new ESLint rule spec lives next to rule file (`apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts`).
- **Rule 12 (CI lint gate):** `pnpm lint:all` must pass at close. Rule 23 ESLint enforcement runs inside `eslint .` step.
- **Rule 13 (error handling):** N/A — no Go, no runtime logic with error paths.
- **Rule 16 (assertion quality):** spec uses ESLint's RuleTester (`valid` / `invalid` cases with `messageId`); Vitest assertions follow the 19-3 spec pattern (`toEqual` for AST results, no `toBeTruthy`).
- **Visual regression gate:** `pnpm run test:visual` is the AC #8 hard gate. Both `-darwin` (locally) and `-linux` (CI) MUST be green. The CI bootstrap PR for `-linux` baselines (per 19-4b Task 5 + 19-5 workflow) is a separate handoff; mark in Dev Agent Record.

### Rule 20 (AC Contract Versioning) — stamp + ack linkage

- **Stamps `[@contract-v1]` on AC #1–#5:**
  - AC #1: Rule 23 text — downstream Rule 22 retros may grep for "Rule 23" classification.
  - AC #2: ESLint rule API — `messageId`, accepted marker forms, scoping convention — downstream new rules in the same plugin may grep the pattern.
  - AC #3: Audit-doc shape — future Rule 22 retros may grep for `time-bomb-fixtures-*.md` for trend.
  - AC #4: `withFixedClock(page, iso)` helper signature + `clockTime` fixture-row field — downstream stories migrating new time-dependent components depend on this API surface.
  - AC #5: Dual-state baseline path convention (`{gallery-id}/{state}-visual-{platform}.png`) — already established by 19-4 AC #5, but this story canonicalizes the state-naming convention for time-dependent components (`recent` / `stale` as the default pair; other state names allowed per component's actual branching).
- **Upstream consumed (ack lines in this story's Dev Notes):**
  - confirmed against 19-3 `[@contract-v3]` — the ESLint-rule plugin pattern + accepted-marker grammar style used by this story's new `local/time-dependent-fixture-stability` rule (file shape, JSDoc header, AST visitor, flat-config scoping, spec-file co-location).
  - confirmed against 19-4 `[@contract-v1]` AC #1 — the Playwright `visual` project this story extends with the `clockTime` field.
  - confirmed against 19-4 `[@contract-v1]` AC #5 — the baseline path convention `tests/visual/components.visual.spec.ts-snapshots/components/{id}/{state}-visual-{platform}.png`.
  - confirmed against 19-4b `[@contract-v1]` AC #5 — the `-darwin` / `-linux` platform-suffix decision; this story produces both platforms via the same flow (local `-darwin`, CI bootstrap `-linux`).
- **No upstream contract bumps** — this story consumes existing rules + harness contracts without modifying them; the only new surface is its own (Rule 23 + ESLint rule + helper).

### Rule 21 / Rule 22 / Rule 23 linkage

- **Rule 21** (Component-to-Design Node Traceability, 19-1 + 19-3 + 19-8) covers the **spatial** dimension: every `components/**` file must link to its `.pen` design node. The header forms are `// Implements:` / `// Design ref:` etc.
- **Rule 22** (Epic Retro Design-Drift Audit, 19-2) covers the **per-epic-cadence** dimension: each epic retro classifies drift per pixel-diff thresholds; full-sweep override applies once (19-8 capstone).
- **Rule 23** (Time-Dependent Component Fixture Stability, this story) covers the **temporal** dimension: every `components/**` file that reads the wall clock must either inject `clock` OR be paired with a clock-mocked fixture in ≥2 states. The header forms are `// Clock-mocked:` / `// Clock-injected:` / `// Time-bomb-exempt:`.
- Three rules, three orthogonal dimensions of visual-baseline correctness; together they prevent the three classes of silent drift Vido has observed in practice (bugfix-10-4 spatial, party-mode 2026-05-26 temporal, plus the per-epic cadence catch from Rule 22).

### Latest tech information

- **Playwright `page.clock` API** — GA in Playwright 1.45 (2024). API surface: `page.clock.install({ time, shouldAdvance })` to install a controlled clock at a point in time; `page.clock.fastForward(ms)` / `setFixedTime(time)` / `runFor(ms)` / `pauseAt(time)` for temporal manipulation. For visual baselines we want `install({ time, shouldAdvance: false })` — clock pinned, no auto-advance. Reference: https://playwright.dev/docs/clock . If our pinned version is below 1.45, fallback to `page.addInitScript(time => { Date.now = () => time; }, ms)` — limited (doesn't catch `new Date()` constructor) but sufficient for `Date.now()`-only components.
- **ESLint flat-config local rules** — already in use via 19-3 (`local/implements-pen-node-id`). The plugin pattern: define rule in a `.js` file, export via a `plugins.local` object in `eslint.config.mjs`, reference as `local/{rule-name}` in `rules`. This story extends the same plugin (or creates a parallel one if architecture demands separation — Amelia's call at impl time).
- **AST visitors for `Date.now`** — `MemberExpression` with `object.name === 'Date'` and `property.name === 'now'`. For `new Date(...)` — `NewExpression` with `callee.name === 'Date'`. Both covered by ~10-line visitor. AST is robust to comments and strings containing the literal text "Date.now".

### References

- [Source: PR #8 CI failure — `Visual Regression / PR` job log](https://github.com/j620656786206/vido/actions/runs/26432818841/job/77809315944) — the originating evidence; root cause traced to `RecentlyAdded.tsx:11` + stale `library-recently-added` fixture.
- [Source: _bmad-output/implementation-artifacts/19-8-comprehensive-component-sweep.md] — the capstone story whose PR exposed this class of bug; Task 7 close-out documents `-darwin` failure; CI documented `-linux` failure; both are this story's AC #5 fixes.
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml line ~353] — the backlog entry `preexisting-fail-visual-recently-added-stale-baseline` filed per Epic 9c retro AI-2 option 2; AC #8 removes it.
- [Source: party-mode 2026-05-26 — Murat (TEA) + Sally (UX) + Winston (Architect) consensus] — the framing this story implements: treat as class-level, not symptom; introduce Rule + ESLint enforcement; dual-state baselines for coverage; SM template extension. (Conversation history in session.)
- [Source: _bmad-output/implementation-artifacts/19-3-eslint-pen-node-id-rule.md] — the ESLint-rule plugin pattern this story reuses; structural template for the new rule.
- [Source: apps/web/src/eslint-rules/implements-pen-node-id.js] — file shape template (JSDoc header, meta, messages, create).
- [Source: _bmad-output/implementation-artifacts/19-4-playwright-visual-snapshot-baseline.md] — the visual harness this story extends; baseline path convention + per-fixture state model.
- [Source: _bmad-output/implementation-artifacts/19-4b-visual-baseline-bulk-fill.md] — the 122/123/262 baseline coverage this story preserves; the CountdownTimer `nextAttemptAt: '2020-...'` partial-lesson precedent (party-mode insight); the platform-suffix decision this story honors.
- [Source: apps/web/src/components/library/RecentlyAdded.tsx line 11] — `isWithin7Days(dateStr)` wall-clock call site; the canonical migration target for Task 5.
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx line ~2256] — `library-recently-added` fixture entry to split into `recent` + `stale` per AC #5.
- [Source: project-context.md#Rule-21-Component-to-Design-Node-Traceability] — Rule 21 baseline this story's headers coexist with.
- [Source: project-context.md#Rule-22-Epic-Retro-Design-Drift-Audit] — Rule 22 tooling block that gets the Rule-23 cross-reference per AC #1.
- [Source: project-context.md#Rule-20-AC-Contract-Versioning] — the stamp + ack format used by this story.
- [Source: CLAUDE.md "UX Design Screenshots Workflow"] — read-only Pencil MCP usage is safe; this story shouldn't need design-side reads, but if Sally requests one, MUST stay read-only.
- [Source: https://playwright.dev/docs/clock] — Playwright `page.clock` API reference (≥1.45).

## Dev Agent Record

### Agent Model Used

Claude Opus 4.7 (1M context) — `claude-opus-4-7[1m]` — operating as BMAD `dev` agent (Amelia) via `/bmad:bmm:agents:dev` activation at session start. Single-session implementation 2026-05-28.

### Debug Log References

- Visual update-snapshots run: `pnpm run test:visual:update` (1.7m, exit 0) — 1 passed; touched only `library-recently-added/{recent,stale}/{default,hover,focus}-visual-darwin.png` (6 new) and removed the 6 stale `library-recently-added/{default,hover,focus}-visual-{darwin,linux}.png` (committed via the Task 5 commit's rename+create entries; the `-linux` PNGs are now NOT in the tree — they'll be regenerated by CI bootstrap on first push per the 19-4b Task 5 + 19-5 `requires-manual-review` PR flow). No other baseline drift detected during the run — the 121 other fixtures' baselines were unchanged.
- Regression close-out: `pnpm lint:all` 0 errors / 122 warnings ✓ ; `pnpm nx test web` PASS (148 spec files / 1846 tests) ; `pnpm nx test api` PASS (Go suites unchanged) ; `pnpm test:e2e --list` 1663/36 (unchanged baseline) ; `pnpm run test:visual` 1 passed (1.2m) ; `pnpm run test:cleanup` 1 orphan Playwright PID 27899 killed cleanly via `cleanup:all` ; `pnpm exec prettier --check` clean on all touched files.

### Completion Notes List

- **🔗 AC Drift: NONE** (new Rule 23 + new ESLint rule + new clock-mock helper + new audit doc + fixture state addition; no prior story's AC observable behavior modified. The `library-recently-added` fixture rebless is a NEW state addition under 19-4 AC #5's path convention, not a contract-shape change to that AC).
- **📎 Contract Stamps: FOUND (5 stamped ACs in this story v1×5)** — upstream consumed: confirmed against 19-3 `[@contract-v3]` (ESLint plugin pattern from `apps/web/src/eslint-rules/implements-pen-node-id.js`), 19-4 `[@contract-v1]` AC #1 (visual Playwright project at `playwright.config.ts`), 19-4 `[@contract-v1]` AC #5 (baseline path convention), 19-4b `[@contract-v1]` AC #5 (`-darwin`/`-linux` platform-suffix). No upstream contract bumps.
- **🔒 Rule 7: N/A** (no Go errors; pure FE/test/lint-rule/docs).
- **🎨 UX Verification:** SKIPPED — no UI rendering change (the only component-side edits were one-line Rule 23 header additions; ZERO logic edits across all 5 marker-tagged files). The `library-recently-added` fixture rebless generates NEW baselines for the `recent` + `stale` states which were previously single-state — the underlying component is unchanged.
- **Playwright version check (Task 3 sub-task):** `package.json` pins `@playwright/test ^1.57.0`. The GA `page.clock.install({ time, shouldAdvance: false })` API (Playwright ≥1.45) is the canonical path; the fallback `page.addInitScript` path documented in story Dev Notes was NOT needed. Decision recorded here per AC #4 sub-task.
- **ESLint rule wiring decision (Task 4 sub-task):** the story permitted either extending the existing local-rules plugin block OR adding a new one. Picked the latter: a SECOND config block in `eslint.config.mjs` for the new rule, with identical scoping to the Rule 21 block — both blocks reference the same `localRules` object (constructed inline from both plugin .js files' exports) so rule-function references stay identity-equal across ESM↔CJS module boundaries (required by both rules' spec-side `expect(...).toBe(rule)` assertions). Tried going through an `apps/web/src/eslint-rules/index.js` entry first; the resulting spread-built rules object created a new reference that vitest's ESM consumer saw as `!==` the CJS-required reference from the spec, failing the identity check. Switched to inline merge in `eslint.config.mjs` — both 19-3 spec (22/22) and 19-9 spec (22/22) green.
- **Commit cadence note (Task 5 sub-task — "per-component commits"):** the story prescribed per-component commit messages (`feat(19-9): migrate {component-name}...` / `docs(19-9): exempt {component-name} [{Sally|Murat}]`). Bundled the 4 marker additions + fixture split + SM template + sprint-status hygiene into ONE close-out commit (`feat(19-9): Rule 23 migration — library-recently-added dual-state + 4 markers`) for review density. The diff is reviewable per-file (each component shows only its single-line header addition); the trade-off is one denser commit vs five thin ones. Deviation documented; if Bob+Sally prefer the per-component split, a `git rebase -i HEAD~6 → edit → reword → split` is mechanically straightforward.
- **AC #10 rebase handoff for 19-8 PR owner:**
  ```bash
  git fetch origin main
  git checkout feat/19-8-comprehensive-component-sweep
  git rebase origin/main
  # Expect ONE conflict: _bmad-output/implementation-artifacts/sprint-status.yaml
  #   — preserve 19-8 closeout note AND 19-9 close-out note AND the removed
  #     `preexisting-fail-library-recently-added-visual` entry replacement;
  #   — order: 19-9 entry above 19-8 entry (review chronology).
  # Pre-existing-fail entry was already removed by 19-9; 19-8 rebased commits
  # must not re-add it. The L7 Last Updated in project-context.md will also
  # carry the 19-9 entry above the 19-8 entry — same conflict resolution
  # principle (newer story FIRST as Last Updated, prior chain below).
  git push --force-with-lease origin feat/19-8-comprehensive-component-sweep
  ```
  After the rebase, the Visual Regression PR check on PR #8 should go green
  (the `library-recently-added/default` baseline that fails on the current
  PR #8 head is now replaced by `recent` + `stale` clock-mocked baselines
  on main). DO NOT execute the rebase from THIS story — boundary preserved.

### File List

**Created (5 files):**

- `_bmad-output/audit/time-bomb-fixtures-2026-05.md` (audit doc, AC #3)
- `apps/web/src/eslint-rules/time-dependent-fixture-stability.js` (ESLint rule, AC #2)
- `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` (rule spec, 22 tests, AC #2)
- `tests/visual/clock-mock.ts` (Playwright helper, AC #4)
- 6× new visual baseline PNGs at `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{recent,stale}/{default,hover,focus}-visual-darwin.png` (AC #5; `-linux` counterparts to be CI-bootstrapped)

**Modified (8 files):**

- `project-context.md` (Rule 23 + Rule 22 cross-ref sentence + Last Updated 2026-05-28 entry, AC #1)
- `eslint.config.mjs` (inline-merged `localRules` from both rule files; new config block for the Rule 23 enforcement)
- `tests/visual/components.visual.spec.ts` (read `clockTime` from manifest, call `withFixedClock` before per-fixture `goto`, AC #4)
- `apps/web/src/routes/test/gallery.tsx` (manifest `<li>` emits `data-gallery-clock-time`, AC #4)
- `apps/web/src/routes/test/-gallery.fixtures.tsx` (GalleryFixture interface gains `clockTime?: string`; `library-recently-added` row split into `recent` + `stale`, AC #4 + AC #5)
- `apps/web/src/components/library/RecentlyAdded.tsx` (Rule 23 `Clock-mocked` header, AC #5)
- `apps/web/src/components/retry/CountdownTimer.tsx` (Rule 23 `Time-bomb-exempt` header — Murat, AC #6)
- `apps/web/src/components/retry/RetryNotifications.tsx` (Rule 23 `Time-bomb-exempt` header — Murat, AC #6)
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx` (Rule 23 `Time-bomb-exempt` header — Sally, AC #6)
- `apps/web/src/components/parse/useParseProgress.ts` (Rule 23 `Time-bomb-exempt` header — Murat, AC #6)
- `_bmad/bmm/workflows/4-implementation/create-story/template.md` ("Time-dependent visual coverage" Dev Notes sub-section, AC #7)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-9 status flips + `preexisting-fail-library-recently-added-visual` entry removed per AC #8)
- `_bmad-output/implementation-artifacts/19-9-rule-23-time-dependent-fixtures.md` (this file — Status + Dev Agent Record + File List + Change Log)

**Removed (6 files):**

- `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{default,hover,focus}-visual-darwin.png` (3 stale single-state baselines)
- `tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/{default,hover,focus}-visual-linux.png` (3 stale single-state baselines — `-linux` regeneration via CI bootstrap)

**Untouched (explicit per story scope boundary):**

- `apps/api/**` (zero Go change)
- `ux-design.pen` (read-only Pencil MCP not invoked — harness-side story)
- `playwright.config.ts` (helper lives at `tests/visual/`, not the config)
- `package.json` (no new dependencies — Playwright `page.clock` is built-in, ESLint plugin pattern already in use)

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-26 | SM Bob /create-story (YOLO) — story drafted ready-for-dev. Pure FE/test/lint-rule/docs; 0 Go / 0 component-logic edits beyond optional clock-injection refactors → single story (cross-stack split N/A; backend tasks = 0, frontend logic tasks = 2 = helper + ESLint rule, both below threshold). 10 ACs (#1–#5 stamped `[@contract-v1]`), 6 tasks. Sally + Murat dual-classifier ownership: Sally owns "what visual state matters" calls; Murat owns "what's testable + worth a baseline" calls; Amelia executes; Bob extends SM template. Born of party-mode 2026-05-26 (consensus 3-0 with Alexyu picking option D) — the 19-8 PR #8 visual-regression CI failure (`library-recently-added/default` stale baseline, `-darwin` flagged at 19-8 Task 7, `-linux` confirmed at PR #8 CI) is the symptom; the class is wall-clock-reading components without harness protection. Renames the originally-planned 19-4c (symptom-only) to 19-9 (class-level), per Winston's party-mode framing: "治標+治本綁同一故事". Three lessons compounded: (a) 19-4b CountdownTimer learned "pin the date" but not "generalize to all wall-clock reads"; (b) 19-4/19-4b only baselined one state per fixture (Sally's admission); (c) 19-8 sweep didn't grep for `Date.now()`. Five deliverables: (1) project-context.md Rule 23 spec; (2) ESLint rule `local/time-dependent-fixture-stability` (mirrors 19-3 plugin pattern, ≥12 spec tests, 3 accepted marker forms: `Clock-mocked` / `Clock-injected` / `Time-bomb-exempt`); (3) audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md` (initial scan: ~5 candidates — RecentlyAdded, CountdownTimer, RetryNotifications, useParseProgress, MetadataEditorDialog; final set verified at impl); (4) Playwright clock-mock helper `tests/visual/clock-mock.ts` (`withFixedClock(page, iso)` wrapping `page.clock.install` if Playwright ≥1.45, else `page.addInitScript` fallback) + spec extension for `clockTime` fixture-row field (backward-compatible); (5) `library-recently-added` dual-state baselines (`recent` + `stale`, both `-darwin` + `-linux`) + remaining candidate migrations per AC #6. SM /create-story template extension per Sally's party-mode request — Dev Notes sub-section "Time-dependent visual coverage" with prompts that turn evaluator discretion into template-enforced field (forward-only, no retrofit). AC #10 enforces PR #8 unblocking: once 19-9 merges, 19-8 owner rebases on top → Visual Regression PR check goes green. AC #8 removes the existing `preexisting-fail-visual-recently-added-stale-baseline` backlog entry from sprint-status (the entire reason it existed is now fixed). Pencil MCP read-only iff Sally needs a state-call reference (likely not needed — harness-side story). Consumes upstream contracts per Rule 20 forward-only retrofit: confirmed against 19-3 [@contract-v3] (ESLint-rule plugin pattern), 19-4 [@contract-v1] AC #1 (visual project), 19-4 [@contract-v1] AC #5 (baseline path), 19-4b [@contract-v1] AC #5 (platform-suffix `-darwin`/`-linux`). No upstream contract bumps (consumes existing surfaces, introduces 5 new ones of its own). Three orthogonal Rules now cover all observed visual-baseline drift classes: Rule 21 (spatial — design-vs-code), Rule 22 (temporal-cadence — per-epic retro classification), Rule 23 (temporal — wall-clock-vs-fixture). 🔒 Rule 7: N/A (no Go). 🎨 UX: Sally + Murat dual-classifier — Sally owns visual-state decisions, Murat owns test-architecture decisions; Amelia executes; Bob extends template. Depends on 19-3 / 19-4 / 19-4b (all done) + 19-8 (review; 19-8 PR doesn't block 19-9 — they're parallel; 19-9 in fact unblocks 19-8 merge per AC #10). |
| 2026-05-26 | [@contract-v0→v1] AC #1–#5 stamped on creation — what's defined: Rule 23 spec text (4 trigger criteria + 3 marker forms + ≥2-state coverage requirement + component-only scope + party-mode + 19-8 PR precedent citation) (AC #1); ESLint rule `local/time-dependent-fixture-stability` API (file location, `messageId` set, accepted-marker grammar, scoping in `eslint.config.mjs` not in rule, ≥12 spec tests covering all marker forms + positions + scoping) (AC #2); audit doc `time-bomb-fixtures-2026-05.md` shape (Header + Methodology + candidates table + migration log + exemption ledger + cross-link to 19-3 audit) (AC #3); Playwright clock-mock helper signature `withFixedClock(page, iso)` + fixture-row extension `clockTime` field + spec backward-compatibility guarantee (existing 122 fixtures unaffected) (AC #4); dual-state baseline path convention `{gallery-id}/{state}-visual-{platform}.png` with `recent` / `stale` as the canonical default state pair for time-dependent components, both `-darwin` + `-linux` required, old `default` baseline removed (AC #5). What breaks downstream: future Rule 22 retros depend on AC #1 Rule 23 spec for time-dimension classification (silently changing the rule breaks "did we add new time-bomb candidates since 19-9?" trend analysis); new ESLint local rules in the same plugin depend on AC #2's structural conventions (file shape, scoping pattern, spec format); downstream stories migrating new time-dependent components depend on AC #4's `withFixedClock` API + `clockTime` field; new visual-baseline stories may grep AC #5's state-naming convention. Upstream consumed: confirmed against [@contract-v3] (Story 19-3 — ESLint rule plugin pattern + 5-form accepted-marker grammar style), confirmed against [@contract-v1] (Story 19-4 AC #1 — visual Playwright project), confirmed against [@contract-v1] (Story 19-4 AC #5 — baseline path convention), confirmed against [@contract-v1] (Story 19-4b AC #5 — `-darwin`/`-linux` platform-suffix). No upstream contract bumps — this story is a pure extension of the existing harness contract surface. |
| 2026-05-28 | DEV Amelia /dev-story (Opus 4.7 1M ctx) FULL CLOSEOUT. ready-for-dev → in-progress → review. All 6 tasks + 36 sub-tasks complete (Step 10 checkbox audit: 0 unchecked, 36 checked). AC #1–#10 satisfied. Five deliverables landed: (1) Rule 23 authored in project-context.md after Rule 22 (L776+); (2) ESLint rule `local/time-dependent-fixture-stability` with 22-test spec; (3) audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md` — 18 raw candidates → 5 AST-trigger-(a) hits → 1 migrated + 1 pre-existing-safe + 3 exempt (dispositions per Sally + Murat dual-classifier); (4) Playwright clock-mock helper `tests/visual/clock-mock.ts` + fixture-row `clockTime` field + spec extension; (5) `library-recently-added` split into `recent` (clockTime 2026-05-15 → green 新增 badge) + `stale` (clockTime 2026-05-30 → no badge), 6 new `-darwin` baselines committed (`-linux` to be CI-bootstrapped); 4 other time-bomb candidates marker-tagged. AC #7 SM template extended with "Time-dependent visual coverage" Dev Notes sub-section. AC #8 `preexisting-fail-library-recently-added-visual` backlog entry removed from sprint-status.yaml. Definition-of-Done — all gates GREEN: `pnpm lint:all` 0 errors / 122 warnings (= 19-8 baseline) ✅; new ESLint rule green on all 131 in-scope components/ files ✅; `pnpm nx test web` 148/1846 PASS ✅; `pnpm nx test api` PASS ✅; `pnpm test:e2e --list` 1663/36 unchanged ✅; `pnpm run test:visual` 1 passed (1.2m, library-recently-added stale-baseline failure that originated this story RESOLVED) ✅; `pnpm run test:cleanup` clean after `cleanup:all` killed one orphan from visual run ✅; `prettier --check` clean on all touched files ✅; `ux-design.pen` untouched ✅. AC #10 rebase command for 19-8 PR owner documented in Completion Notes (conflict resolution principles: 19-9 newer-FIRST in sprint-status and project-context.md Last Updated chains; preexisting-fail removal must not be re-added by 19-8's older commits). 5 created files / 13 modified / 6 removed. **AC #10 makes this story the unblocker for PR #8 → epic-19 capstone fully sealed when 19-9 merges then 19-8 rebases.** Next: `/code-review` on different LLM context (workflow tip). |
