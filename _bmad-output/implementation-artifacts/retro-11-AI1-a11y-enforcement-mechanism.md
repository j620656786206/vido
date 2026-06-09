# Story: A11y Enforcement Mechanism — jsx-a11y Lint + MANDATORY dev-story Pre-Flight

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Scrum Master (Bob) + Dev Agent (Amelia) responsible for keeping accessibility from slipping to code-review,
I want (1) `eslint-plugin-jsx-a11y` installed and scoped to `apps/web/src/components/**`, and (2) the a11y pre-flight promoted from a **passive** `checklist.md` `[ ]` to a **MANDATORY** `dev-story` action with a recorded `{{a11y_preflight_result}}` line,
so that a11y issues are caught by automation + an enforced workflow gate (the same enforcement shape as AC-Drift / Rule-7 / Contract-Stamps that demonstrably **stuck** through Epic 11) instead of recurring at CR (11-2 bottom sheet `aria-modal` with no keyboard, 11-3 combobox `aria-activedescendant`).

## Acceptance Criteria

1. `eslint-plugin-jsx-a11y` is installed as a root `devDependency` (pinned `^6.10.2`, the current stable) and appears in `package.json` alongside the other `eslint-plugin-*` entries. `pnpm install` resolves cleanly under the pnpm workspace.

2. A NEW flat-config block in `eslint.config.mjs` enables the `jsx-a11y` **recommended** ruleset, scoped to `apps/web/src/components/**/*.{ts,tsx}` with the SAME `ignores` list as the existing Rule 21 / Rule 23 blocks (`**/*.spec.{ts,tsx}`, `**/*.test.{ts,tsx}`, `**/index.ts` under `components/`). The block registers the plugin under the `jsx-a11y` namespace and is placed BEFORE the final `prettier` config object.

3. The jsx-a11y rules are enabled at **`warn`** severity (NOT `error`). Rationale (see Dev Notes): this preserves the established `pnpm lint:all` **0-errors** gate while surfacing the existing component a11y-violation batch as warnings for `retro-11-AI1b` to clear. A future bump to `error` is explicitly out of scope (it is the closing act of AI1b, not this story).

4. A wiring spec (mirroring `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` → `describe('eslint.config.mjs wiring ...')`) asserts, by importing the resolved flat config: (a) exactly ONE config object enables jsx-a11y rules; (b) it is scoped to `['apps/web/src/components/**/*.{ts,tsx}']`; (c) it carries the same `ignores` triple; (d) the `jsx-a11y` plugin object is registered; (e) the rules are registered at `warn`. A scope refactor cannot silently widen/narrow coverage without failing this spec.

5. The a11y pre-flight is **removed** from `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` as a passive `[ ]` section and **re-homed** as an `<action critical="MANDATORY">` in `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml`. The `checklist.md` section is replaced by a short **pointer** to the enforced action (single source of truth — no duplicated, drift-prone copy of the 4 items in two places).

6. The new MANDATORY action: (a) gates on "story touches `apps/web/`" and is explicitly skippable only when 100% backend; (b) enumerates the 4 Epic-10-Retro-AI-1 a11y items WITH their concrete Epic-10 story CR precedents (so it reads as a real failure history, not a ceremony checkbox); (c) requires running jsx-a11y lint on the touched components (`pnpm lint:all`, or a scoped `eslint` on the changed `components/**` files) and reading the result; (d) is followed by a sibling binding `<action critical="MANDATORY">` that sets `{{a11y_preflight_result}}` to exactly one of `PASS ({n} components checked, {w} jsx-a11y warnings)` | `FOUND — see Completion Notes` | `N/A (100% backend — no apps/web/ files touched)`, recorded in Dev Agent Record → Completion Notes as `🎭 A11y Pre-Flight: {result}` — mirroring the `{{drift_check_result}}` / `{{contract_stamps_result}}` binding-action pattern proven in retro-10-AI2 / retro-10-AI5.

7. This is a tooling + workflow-docs change with one small TS config + spec touch. `pnpm lint:all` (0 errors), `pnpm nx test api`, and `pnpm nx test web` all pass with zero regressions. sprint-status transitions `backlog → ready-for-dev → in-progress → review → done`. The `retro-11-AI1b` boundary is preserved: clearing the surfaced warning batch is NOT done here.

## Tasks / Subtasks

- [x] **Task 1 (Prong 1 — DEV/ESLint): Install + wire `eslint-plugin-jsx-a11y`** (AC: #1, #2, #3)
  - [x] 1.1 Add `eslint-plugin-jsx-a11y` (`^6.10.2`) to root `package.json` `devDependencies` (next to `eslint-plugin-react` / `eslint-plugin-react-hooks`); run `pnpm install` and confirm the lockfile updates cleanly.
  - [x] 1.2 In `eslint.config.mjs`, add the import: `import jsxA11y from 'eslint-plugin-jsx-a11y';` (top, with the other plugin imports).
  - [x] 1.3 Build a `warn`-severity rule map from the recommended ruleset rather than hand-listing rules, so the set stays current with the plugin:
    ```js
    // jsx-a11y recommended ships at 'error'; remap to 'warn' so the existing
    // lint:all 0-errors gate stays green while the existing component batch
    // surfaces as warnings for retro-11-AI1b to clear. Bump to 'error' is AI1b's
    // closing step, NOT this story.
    const jsxA11yWarn = Object.fromEntries(
      Object.keys(jsxA11y.flatConfigs.recommended.rules).map((r) => [r, 'warn'])
    );
    ```
  - [x] 1.4 Add a NEW flat-config object immediately AFTER the Rule 23 block and BEFORE the `prettier` config (mirror the Rule 21 / Rule 23 block shape exactly):
    ```js
    // Epic 11 Retro AI-1 — eslint-plugin-jsx-a11y enforcement. Scope mirrors the
    // Rule 21/23 component blocks. WARN (not error) preserves the 0-errors gate;
    // the surfaced batch is cleared by retro-11-AI1b.
    {
      files: ['apps/web/src/components/**/*.{ts,tsx}'],
      ignores: [
        'apps/web/src/components/**/*.spec.{ts,tsx}',
        'apps/web/src/components/**/*.test.{ts,tsx}',
        'apps/web/src/components/**/index.ts',
      ],
      plugins: { 'jsx-a11y': jsxA11y },
      rules: jsxA11yWarn,
    },
    ```
  - [x] 1.5 Sanity-run `pnpm lint:all` — confirm **0 errors** still (warnings count may rise; that is expected and is AI1b's input). Record the new warnings delta in Dev Agent Record.

- [x] **Task 2 (Prong 1 — DEV/ESLint): Wiring spec** (AC: #4)
  - [x] 2.1 Add `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts` (or extend an existing config-wiring spec). Import the resolved flat config exactly as `time-dependent-fixture-stability.spec.ts` does (`resolve(__dirname, '../../../../eslint.config.mjs')` → dynamic `import`).
  - [x] 2.2 Assert: exactly one config object has a `rules` map containing a `jsx-a11y/*` key (`flatConfig.filter((c) => c.rules && Object.keys(c.rules).some((k) => k.startsWith('jsx-a11y/')))` → `toHaveLength(1)`).
  - [x] 2.3 Assert that block's `files` equals `['apps/web/src/components/**/*.{ts,tsx}']` and `ignores` equals the spec/test/index triple.
  - [x] 2.4 Assert the `jsx-a11y` plugin object is registered in `plugins` and that a spot-checked recommended rule (e.g. `jsx-a11y/alt-text`) is present at `'warn'`.
  - [x] 2.5 Run `pnpm nx test web` filtered to this spec; confirm green.

- [x] **Task 3 (Prong 2 — SM/workflow): Promote a11y pre-flight to MANDATORY action** (AC: #5, #6)
  - [x] 3.1 In `dev-story/instructions.xml`, add the a11y MANDATORY action to **Step 7 (Run validations and tests)**, immediately after the FULL REGRESSION GATE action — Step 7 is where lint runs (so jsx-a11y warnings surface) and is the established home for validation gates (retro-9-AI1 regression gate + retro-9c-AI2 fix-or-file both live at Step 7). Draft shape:
    ```xml
    <!-- Epic 11 Retro AI-1 (a11y kept slipping to CR: 11-2 bottom sheet, 11-3 combobox).
         Promotes the Epic-10-Retro-AI-1 a11y pre-flight from a passive checklist.md [ ]
         to a MANDATORY recorded gate — mirrors the AC-Drift / Contract-Stamp pattern that
         demonstrably stuck through Epic 11. Pairs with eslint-plugin-jsx-a11y (warn). -->
    <action critical="MANDATORY" if="story touches any file under apps/web/">A11Y PRE-FLIGHT (Epic 11 Retro AI-1):
      Skip ONLY if the story is 100% backend (no apps/web/ files). Otherwise you MUST,
      before marking the story review-ready:
        1. Run jsx-a11y lint over the touched components (pnpm lint:all, or a scoped
           eslint on the changed apps/web/src/components/** files). Read the jsx-a11y
           warnings for files THIS story added or modified; resolve any introduced by
           this story's own new/changed code (pre-existing batch belongs to retro-11-AI1b,
           do NOT expand scope to clear it — note its count only).
        2. Manually confirm the 4 recurring a11y classes for any component this story
           touches (each maps to a real Epic-10 CR finding — concrete, not abstract):
           - Responsive image sizing: TMDb <img> uses srcSet + sizes, sub-original
             baseline in src (Story 10-2 CR H1).
           - Modal focus management: aria-modal components trap focus, move focus on
             open, restore on close; inactive carousel slides inert (Story 10-2 H2/M1;
             repeat class = 11-2 bottom sheet had aria-modal w/ no keyboard).
           - aria-live on async-revealed content: status pills/badges carry role=status
             + aria-live=polite (Story 10-4 L1).
           - Keyboard + ARIA semantics for custom widgets: comboboxes expose
             aria-activedescendant + listbox-only options (11-3 CR), bottom sheets close
             on Escape + take initial focus (11-2 CR M2).
        3. Lazy-load contract accuracy: any IntersectionObserver/pagination request-count
           is described accurately in BOTH the AC text and the request-site comment
           (Story 10-5 H1) — pairs with the retro-10-AI2 AC drift check.
    </action>
    ```
  - [x] 3.2 Add the sibling binding action so the result renders as a concrete, auditable value (mirror the `{{drift_check_result}}` binding at lines 199–202):
    ```xml
    <action critical="MANDATORY">Set {{a11y_preflight_result}} to exactly one of:
      "PASS ({n} components checked, {w} jsx-a11y warnings on touched files, 0 introduced
       by this story)" | "FOUND — see Completion Notes" | "N/A (100% backend — no apps/web/
      files touched)". Record it in Dev Agent Record → Completion Notes as
      "🎭 A11y Pre-Flight: {{a11y_preflight_result}}". This binding MUST execute so the
      result is auditable (a future retro can grep "🎭 A11y Pre-Flight:" across stories).</action>
    ```
  - [x] 3.3 In `dev-story/checklist.md`, REMOVE the body of the `## 🎭 Frontend Performance + Accessibility Pre-Flight (Epic 10 Retro AI-1)` section (the 4 `[ ]` items + verification steps) and replace it with a one-line pointer:
    > **Now enforced as a MANDATORY gate in `dev-story/instructions.xml` Step 7 (Epic 11 Retro AI-1).** Result recorded as `🎭 A11y Pre-Flight:` in Completion Notes. Do not re-add passive checkboxes here — single source of truth lives in the workflow action.
  - [x] 3.4 Read the modified `instructions.xml` end-to-end: confirm Step 1–11 numbering is preserved, the new actions nest INSIDE Step 7 (do not close it early), and the XML is well-formed (`xmllint --noout` PASS). Confirm `checklist.md` still parses (frontmatter + section headers intact).

- [x] **Task 4: Verification — zero regressions** (AC: #7)
  - [x] 4.1 `pnpm lint:all` — 0 errors (warnings may increase from jsx-a11y; record the delta vs the last-known baseline of 122 warnings).
  - [x] 4.2 `pnpm nx test api` — all Go packages PASS (zero Go change expected).
  - [x] 4.3 `pnpm nx test web` — all PASS incl. the new wiring spec; run `pnpm run test:cleanup` to confirm no orphaned workers.
  - [x] 4.4 `pnpm prettier --check` on the touched `.ts`/`.mjs`/`.md` files.

- [x] **Task 5: sprint-status transitions** (AC: #7)
  - [x] 5.1 `retro-11-AI1-a11y-enforcement-mechanism: ready-for-dev` at story creation (this `/create-story` step).
  - [x] 5.2 `ready-for-dev → in-progress` on `/dev-story` start; `in-progress → review` on completion.
  - [ ] 5.3 `review → done` on `/code-review` pass; append a completion note recording the final `instructions.xml` Step 7 line range + the new lint:all warnings count, and re-confirm the AI1b hand-off (warning batch NOT cleared here).

## Dev Notes

### Root Cause (from Epic 11 Retro, 2026-06-09)

Epic 10's retro committed 5 action items. The three that became a **mandatory workflow action with a recorded line** (AC Drift Check, Rule 7 wire-format, Contract Stamps) or an **automated check** all stuck and were visibly applied in every Epic 11 story. The ONE that stayed a **passive `[ ]` in `checklist.md`** — Epic-10-Retro-AI-1's frontend perf + a11y pre-flight — did NOT stick: a11y issues still surfaced at CR in 11-2 (bottom sheet declared `role="dialog" aria-modal="true"` with no keyboard affordances) and 11-3 (combobox needed `aria-activedescendant` + listbox-only options). **Enforcement mechanism predicts stickiness.** This story converts the a11y pre-flight to the enforcement shape that worked AND adds the automation backstop (`eslint-plugin-jsx-a11y`) that was previously absent.

### Why `warn`, not `error` (AC #3)

The established gate is `pnpm lint:all` → **0 errors / ~122 warnings**, and every prior retro story treats "0 errors" as the pass condition. jsx-a11y's recommended ruleset ships at `error`; enabling it as-is would fail `lint:all` the moment any existing `components/**` file violates a rule — which the retro explicitly anticipates (that's the entire reason `retro-11-AI1b` exists: "clear the batch of existing component a11y **warnings**"). The word *warnings* in AI1b is the tell. So: enable at `warn` here (gate stays green, batch becomes visible), let AI1b clear the batch, and let the `warn → error` ratchet be AI1b's closing move. This keeps THIS story self-contained and non-blocking, exactly like retro-10-AI2 was.

### Placement decision — Step 7, not Step 2 (AC #6)

The AC Drift / Contract Stamp checks live in Step 2 because they are **pre-implementation analysis** (reconcile the new AC against prior contracts before writing code). The a11y gate is **post-implementation validation** — it needs the implemented components to lint and the rendered widgets to verify. Step 7 ("Run validations and tests") is the established home for validation gates: retro-9-AI1's FULL REGRESSION GATE and retro-9c-AI2's fix-or-file both landed there, and jsx-a11y warnings surface precisely when lint runs in Step 7. The recorded-line auditability (the actual mechanism the retro asked us to "mirror from the AC Drift Check") is preserved via the sibling binding action + the `🎭 A11y Pre-Flight:` Completion-Notes prefix.

### Single source of truth (AC #5)

We deliberately do NOT keep the 4 a11y items in BOTH `checklist.md` and `instructions.xml` — that would be a drift trap (the exact failure class retro-10-AI2 exists to prevent). The enforced copy lives in `instructions.xml`; `checklist.md` keeps a one-line pointer so a reader scanning the DoD still discovers it but cannot mistake the passive checkbox for the gate.

### Why one story (cross-stack split check)

Backend tasks: **0** (no Go). Frontend/code tasks: the eslint block + wiring spec (Tasks 1–2). Workflow-docs tasks: Task 3. Verification/tracking: Tasks 4–5. Neither side exceeds 3 code tasks → **single story, no split**. The two prongs are tightly coupled (the MANDATORY action references the jsx-a11y lint that prong 1 enables), and `retro-11-AI1b` (clear the warning batch) is already a separate tracked backlog entry — so the natural seam is AI1 (install + enforce) vs AI1b (clean up), which the retro already drew.

### Project Structure Notes

- `eslint.config.mjs` is a single root flat-config (ESLint 9). New scoped block sits between the Rule 23 block (`apps/web/src/eslint-rules/time-dependent-fixture-stability.js`) and the trailing `prettier` object. No change to the Rule 21 / Rule 23 blocks.
- Wiring spec lives under `apps/web/src/eslint-rules/` next to the two existing config-wiring specs — the directory already owns "assert the flat config is scoped correctly" tests.
- Workflow files under `_bmad/bmm/workflows/4-implementation/dev-story/` are not linted (XML/MD), so prong 2 cannot regress `lint:all`.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — explicitly `N/A — no wall-clock-reading components touched`. This story changes only `eslint.config.mjs` (tooling config), a wiring `.spec.ts`, and two BMAD workflow docs. It adds/edits zero React components.
- Reference: `project-context.md` Rule 23.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#challenges`] Pattern #1 — a11y keeps being caught at CR not pre-flight (11-2, 11-3)
- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#action-items`] retro-11-AI1 row (DEV ESLint + SM workflow, HIGH) + the Epic-10-follow-through table (enforcement-mechanism → stickiness)
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] `retro-11-AI1-a11y-enforcement-mechanism: backlog` (this story) + `retro-11-AI1b-clear-a11y-warning-batch: backlog` (the explicit out-of-scope follow-up)
- [Source: `eslint.config.mjs`] Rule 21 (lines 212–225) + Rule 23 (lines 233–246) scoped-block precedent for the new jsx-a11y block
- [Source: `apps/web/src/eslint-rules/time-dependent-fixture-stability.spec.ts` lines 130–170] flat-config wiring-spec pattern to mirror
- [Source: `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` lines 133–152] the passive `## 🎭 Frontend Performance + Accessibility Pre-Flight` section being re-homed
- [Source: `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` Step 7 + Step 2 lines 199–202] the FULL REGRESSION GATE target placement + the `{{drift_check_result}}` binding-action pattern to mirror
- [Precedent: `retro-10-AI2-ac-contract-drift-check`] near-identical workflow-enforcement story (MANDATORY action + recorded auditable line + binding action)

### Out of Scope

- **Clearing the existing component a11y warning batch** → `retro-11-AI1b` (separate tracked entry). This story only SURFACES the batch; it must not expand scope to fix it.
- **Bumping jsx-a11y `warn → error`** → the closing move of AI1b, once the batch is at zero.
- **Applying jsx-a11y outside `components/**`** (hooks/services/routes/`apps/web/src` root) → the retro scoped it to `apps/web/src/components/**`; widening is a future decision, not this story.
- **Retroactively a11y-auditing Epic 1–11 components** → not this story.

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia / Dev Agent, /dev-story)

### Debug Log References

- `pnpm lint` after wiring jsx-a11y: `268 problems (0 errors, 268 warnings)` — 0-errors gate preserved.
- jsx-a11y warning delta: **+146** (baseline ~122 → 268). Breakdown: 46 `label-has-for`, 37 `control-has-associated-label`, 24 `label-has-associated-control`, 16 `no-noninteractive-element-interactions`, 14 `click-events-have-key-events`, 5 `no-static-element-interactions`, 4 `no-autofocus`.
- `pnpm nx test web`: 164 files / 2001 tests passed (incl. new `jsx-a11y-config.spec.ts`).
- `pnpm nx test api`: PASS (no Go change).
- `xmllint --noout` on `instructions.xml`: PASS; steps 1–11 intact; a11y actions at lines 408 + 432 nest inside Step 7 (394) before Step 8 (453).

### Completion Notes List

- **Prong 1 (DEV/ESLint):** Added `eslint-plugin-jsx-a11y@^6.10.2` (current stable) to root `devDependencies`; `pnpm install` resolved cleanly. New scoped flat-config block in `eslint.config.mjs` (after Rule 23, before `prettier`) enables the **recommended** ruleset remapped to `warn` via `Object.fromEntries(Object.keys(jsxA11y.flatConfigs.recommended.rules).map((r) => [r, 'warn']))`. Scope + ignores triple mirror the Rule 21/23 blocks exactly. The 0-errors `lint:all` gate is preserved; the existing component a11y batch (146 warnings) now surfaces for retro-11-AI1b — **NOT cleared here** (AI1b boundary preserved).
- **Wiring spec:** `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts` imports the resolved flat config and asserts (a) exactly one block enables `jsx-a11y/*`, (b) `files` scope, (c) `ignores` triple, (d) plugin registered, (e) every registered jsx-a11y rule (spot-check `alt-text`) at `warn`. The `warn` assertion is the load-bearing line AI1b flips to `error`.
- **Prong 2 (SM/workflow):** Promoted the Epic-10-Retro-AI-1 a11y pre-flight from a passive `checklist.md` `[ ]` to an `<action critical="MANDATORY" if="story touches any file under apps/web/">` in `instructions.xml` **Step 7** (immediately after the FULL REGRESSION GATE, lines 408–431), plus a sibling binding `<action critical="MANDATORY">` (lines 432–437) setting `{{a11y_preflight_result}}` — mirroring the `{{drift_check_result}}`/`{{contract_stamps_result}}` pattern. `checklist.md` section body replaced with a one-line pointer (single source of truth; no drift-prone duplicate). `<img>` in the action text encoded as `&lt;img&gt;` to keep the XML well-formed.
- 🔗 **AC Drift: N/A** (workflow-docs + tooling change; re-homes the Epic-10-AI-1 a11y pre-flight from passive `checklist.md` to a MANDATORY `instructions.xml` action — an AC-mandated promotion, not a silent behavior drift of any shipped codebase AC. Grep `'Frontend Performance + Accessibility Pre-Flight|a11y pre-flight|jsx-a11y'` across `_bmad-output/implementation-artifacts/*.md` + dev-story workflow files returned only the retro, this story, AI1b, and the section being re-homed).
- 📎 **Contract Stamps: NONE** (no `[@contract-v*]` stamps in this story or upstream refs — normal for a tooling/workflow story; `grep -nE '\[@contract-v[0-9]+\]'` on the story file → 0 hits).
- 🎭 **A11y Pre-Flight: N/A** (this story adds/edits zero React components — only `eslint.config.mjs` tooling config, a wiring `.spec.ts`, and two BMAD workflow docs; no `apps/web/src/components/**` component touched. The new gate it installs applies to *future* frontend stories, not itself.)
- 🔌 **Route Sync: N/A** (no backend route touched — zero Go change).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story (tooling config + wiring spec + workflow docs; zero React components added or modified).
- **Checkbox audit (Step 10):** all task checkboxes 1.1–5.2 marked `[x]`. Task **5.3** (`review → done` on `/code-review` pass) is intentionally **deferred** — it transitions at the next workflow (`/code-review`), not at dev-story completion; the dev-story workflow terminates at `review` status. Justified deferral, not an incomplete task.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - Expected default: `N/A — no out-of-scope work discovered`. The one known adjacency (the existing a11y warning batch surfaced by enabling jsx-a11y) is NOT a new discovery — it is already tracked as `retro-11-AI1b-clear-a11y-warning-batch` in sprint-status.yaml (bidirectional link: this story surfaces it; AI1b clears it). DEV: if jsx-a11y surfaces a violation class NOT covered by AI1b, file it per Rule 24 lane ③ at discovery time.
- Reference: `project-context.md` Rule 24.

### File List

- `package.json` (modified — added `eslint-plugin-jsx-a11y@^6.10.2` devDependency)
- `pnpm-lock.yaml` (modified — lockfile updated by `pnpm install`)
- `eslint.config.mjs` (modified — import + new scoped jsx-a11y `warn` flat-config block)
- `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts` (new — flat-config wiring spec)
- `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` (modified — MANDATORY a11y pre-flight action + binding action in Step 7)
- `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` (modified — a11y section body replaced with one-line pointer)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified — status transitions)
- `_bmad-output/implementation-artifacts/retro-11-AI1-a11y-enforcement-mechanism.md` (this story file — task checkboxes, Dev Agent Record, Change Log, Status)

## Change Log

| Date | Change | Author |
| ---- | ------ | ------ |
| 2026-06-09 | Story created (SM Bob /create-story, YOLO) — backlog → ready-for-dev. Two prongs: (1) install + scope eslint-plugin-jsx-a11y at warn; (2) promote a11y pre-flight from checklist.md [ ] to MANDATORY dev-story Step 7 action with {{a11y_preflight_result}} binding. AI1b boundary preserved. | Bob (SM) |
| 2026-06-09 | Task 1 — installed eslint-plugin-jsx-a11y ^6.10.2; added import + scoped warn-severity recommended-ruleset block to eslint.config.mjs (same files/ignores as Rule 21/23). lint:all stays 0 errors; +146 jsx-a11y warnings surfaced for AI1b. | Amelia (Dev) |
| 2026-06-09 | Task 2 — added jsx-a11y-config.spec.ts wiring spec (5 assertions: single block, files scope, ignores triple, plugin registered, all rules at warn). nx test web green (2001 tests). | Amelia (Dev) |
| 2026-06-09 | Task 3 — promoted a11y pre-flight to MANDATORY action in instructions.xml Step 7 (lines 408–431) + sibling {{a11y_preflight_result}} binding (432–437); replaced checklist.md section body with one-line pointer. xmllint PASS, steps 1–11 intact. | Amelia (Dev) |
| 2026-06-09 | Task 4 — verification: lint:all 0 errors, nx test api PASS, nx test web 2001 PASS, prettier --check clean, test:cleanup no orphans. Task 5 — sprint-status ready-for-dev → in-progress → review. 5.3 (review → done) deferred to /code-review. | Amelia (Dev) |
