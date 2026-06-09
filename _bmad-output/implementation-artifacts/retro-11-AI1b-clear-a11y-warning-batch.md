# Story: Clear the existing component a11y warning batch + ratchet jsx-a11y warn → error

Status: ready-for-dev

**⛔ Depends on: `retro-11-AI1` (backend gate)** — `eslint-plugin-jsx-a11y` must be installed and wired at `warn` (AI1 prong 1) BEFORE this story can run. The warning batch this story clears does not exist until AI1 lands. sprint-status entry is `blocked` until AI1 is `done`; flip to `ready-for-dev` then.

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Dev Agent (Amelia) finishing the Epic 11 Retro AI-1 a11y enforcement push,
I want to clear every `jsx-a11y/*` warning that `eslint-plugin-jsx-a11y` surfaces under `apps/web/src/components/**` and then ratchet those rules from `warn` to `error`,
so that the automation backstop AI1 installed actually blocks future a11y regressions (a `warn`-level rule the build ignores would let the exact 11-2 / 11-3 class slip again).

## Acceptance Criteria

1. **Precondition verified:** `retro-11-AI1` is `done` — `eslint-plugin-jsx-a11y` is installed and the `warn`-level scoped flat-config block is live in `eslint.config.mjs`. If AI1 is not yet done, this story HALTS (it cannot surface the batch). The first Dev Agent Record note records the AI1 commit/state confirmed.

2. **Authoritative inventory captured (NOT grep):** Running the real lint (`pnpm lint:all`, or `pnpm exec eslint "apps/web/src/components/**/*.{ts,tsx}"`) produces the definitive list of `jsx-a11y/*` warnings. The Dev Agent Record records, before any fix, the full inventory grouped by rule id → file → line → count. This is the source of truth; the grep recon in Dev Notes is only a starting estimate.

3. **Batch cleared to zero:** Every `jsx-a11y/*` warning under `apps/web/src/components/**` is resolved with the correct accessibility remedy (not suppressed wholesale). Per category:
   - `label-has-associated-control` → associate the `<label>` with its control via `htmlFor`+`id` (or nest the control), preserving the rendered layout and Chinese label text.
   - `no-autofocus` → see AC #4 nuance: for genuine modal/dialog focus-on-open, convert the `autoFocus` prop to a `ref` + `useEffect(() => el.focus(), [])` (the a11y-accepted pattern that ALSO satisfies AI1's modal-focus-management item), NOT a blind deletion that would regress dialog focus. A non-dialog `autoFocus` is removed.
   - `click-events-have-key-events` / `no-static-element-interactions` (e.g. `PosterCardMenu.tsx`) → make the interactive element a real `<button>`, or add `role` + `tabIndex` + keyboard handler (Enter/Space), matching the Story 10-2 / bugfix-10-6 keyboard-affordance precedents.
   - Any other surfaced rule → fixed per that rule's standard remedy; a suppression (`// eslint-disable-next-line jsx-a11y/...`) is allowed ONLY with an inline justification comment AND is listed explicitly in the Dev Agent Record (count + reason per suppression).

4. **`no-autofocus` on dialogs does not regress modal focus (cross-link to AI1):** Any dialog that legitimately takes focus on open (`SavePresetDialog`, `ManualSearchDialog`, `InstantSearchBar`, `AppShell`, and any others the inventory surfaces) keeps its focus-on-open behavior via the ref+effect pattern. This is verified to still satisfy the AI1 modal-focus-management pre-flight item — the two stories must not work against each other.

5. **Ratchet warn → error (the closing move AI1 deferred):** Once the inventory is at zero, the jsx-a11y rules in `eslint.config.mjs` are bumped from `warn` to `error` (drop the `jsxA11yWarn` remap; apply the recommended ruleset at its native `error` severity, or an explicit `error` map). The AI1 wiring spec (`apps/web/src/eslint-rules/jsx-a11y-config.spec.ts`) is updated to assert the rules are registered at `error` (was `warn`). After the bump, `pnpm lint:all` still reports **0 errors** (proving the batch is truly clear).

6. **Zero behavior/visual regression:** `pnpm lint:all` (0 errors, 0 jsx-a11y warnings), `pnpm nx test api`, `pnpm nx test web` (incl. the updated wiring spec) all pass; `pnpm run test:cleanup` shows no orphaned workers. If any a11y fix changes rendered DOM/pixels, the affected Rule 22 visual baselines are reblessed through the established `-darwin`/`-linux` flow with a Sally/UX re-engagement note; if no pixels change, that is stated explicitly. sprint-status transitions `blocked → ready-for-dev → in-progress → review → done`.

## Tasks / Subtasks

- [ ] **Task 1: Confirm the AI1 gate + capture the authoritative inventory** (AC: #1, #2)
  - [ ] 1.1 Verify `retro-11-AI1` is `done` in sprint-status and `eslint-plugin-jsx-a11y` resolves (`pnpm ls eslint-plugin-jsx-a11y`). If not done → HALT and record the blocker.
  - [ ] 1.2 Run `pnpm exec eslint "apps/web/src/components/**/*.{ts,tsx}"` (or `pnpm lint:all` and filter `jsx-a11y/`). Capture the FULL warning list.
  - [ ] 1.3 Record the inventory in Dev Agent Record grouped by `rule id → file:line → count`, plus a per-rule total. This is the work-list for Task 2 and the AC #2 artifact.

- [ ] **Task 2: Clear the batch, category by category** (AC: #3, #4)
  - [ ] 2.1 `label-has-associated-control`: for each flagged `<label>` (recon points at `settings/`, `setup/`, `metadata-editor/`, `library/SettingsGearDropdown.tsx` clusters), add `htmlFor` + a matching control `id` (or nest the control). Preserve Tailwind classes + Chinese copy. Add/extend the component's spec to assert the association (`getByLabelText` works).
  - [ ] 2.2 `no-autofocus`: convert dialog `autoFocus` to ref+`useEffect` focus (keeps focus-on-open, satisfies AI1 modal-focus item); remove non-dialog `autoFocus`. Files from recon: `shell/AppShell.tsx`, `search/SavePresetDialog.tsx`, `search/InstantSearchBar.tsx`, `manual-search/ManualSearchDialog.tsx` (confirm against Task 1 inventory — recon, not authoritative).
  - [ ] 2.3 `click-events-have-key-events` / `no-static-element-interactions`: fix `library/PosterCardMenu.tsx` (recon: the one static-element `onClick`) → real `<button>` or `role`+`tabIndex`+keyboard handler. Add keyboard-activation test.
  - [ ] 2.4 Any other rule the Task 1 inventory surfaced: fix per standard remedy. For each unavoidable suppression, add an inline justification + list it in Dev Agent Record.
  - [ ] 2.5 Re-run the scoped lint after each category; drive the jsx-a11y warning count to **0**.

- [ ] **Task 3: Ratchet warn → error** (AC: #5)
  - [ ] 3.1 In `eslint.config.mjs`, change the AI1 jsx-a11y block from the `jsxA11yWarn` remap to native `error` severity (apply `jsxA11y.flatConfigs.recommended` directly, or an explicit `error` map). Remove the now-unused `jsxA11yWarn` helper if nothing else uses it.
  - [ ] 3.2 Update `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts`: the severity assertion flips `warn` → `error`; scope/ignores/plugin assertions unchanged.
  - [ ] 3.3 Run `pnpm lint:all` — confirm **0 errors** (a non-zero count means the batch was not fully cleared; return to Task 2).

- [ ] **Task 4: Regression + visual verification** (AC: #6)
  - [ ] 4.1 `pnpm lint:all` (0 errors, 0 jsx-a11y warnings), `pnpm nx test api`, `pnpm nx test web` (incl. updated wiring spec + any new association/keyboard tests), `pnpm run test:cleanup`.
  - [ ] 4.2 Visual check: determine whether any fix altered rendered DOM/pixels (htmlFor/id are non-visual; keyboard handlers + role are non-visual; a NEW visible focus ring or a changed element tag could shift pixels). If pixels change → rebless the affected Rule 22 baselines (`-darwin` + `-linux` per the 19-5 CI flow) with a Sally/UX re-engagement note. If not → record `Visual: no rendered-output change`.
  - [ ] 4.3 `pnpm exec prettier --check` on all touched files.

- [ ] **Task 5: sprint-status transitions** (AC: #6)
  - [ ] 5.1 Entry stays `blocked` until `retro-11-AI1` is `done`; on AI1 close, flip `blocked → ready-for-dev`.
  - [ ] 5.2 `ready-for-dev → in-progress` on `/dev-story` start; `in-progress → review` on completion (record the final jsx-a11y warning count = 0 + the warn→error bump confirmation).
  - [ ] 5.3 `review → done` on `/code-review` pass.

## Dev Notes

### Why this is a separate story from AI1 (and why blocked)

AI1 (a) installs jsx-a11y at `warn` so the existing batch surfaces WITHOUT breaking the `lint:all` 0-errors gate, and (b) wires the MANDATORY dev-story a11y pre-flight. AI1 deliberately does NOT fix the surfaced warnings or bump to `error` — that is this story. The seam (install+enforce vs. clean-up+ratchet) is exactly where the retro drew AI1 vs AI1b. This story is hard-blocked: with no jsx-a11y installed there are no warnings to clear, so Task 1's inventory is empty/impossible until AI1 lands.

### The warn → error ratchet is the whole point

A `warn`-level rule the CI build does not fail on is functionally the same passive non-enforcement that let a11y slip to CR in 11-2/11-3 (the retro's root cause). Clearing the batch is necessary but not sufficient — without the `error` bump, the next violation just adds a warning nobody blocks on. AC #5 (the bump) is the load-bearing AC; AC #3 (clear the batch) exists to make the bump possible.

### Recon estimate (APPROXIMATE — grep, not AST; Task 1 is authoritative)

Read-only grep over `apps/web/src/components/**` (135 component files) at story-creation time:

| Likely rule | recon signal | est. |
|---|---|---|
| `label-has-associated-control` | `<label>` with no same-line `htmlFor`, not obviously wrapping a control | ~20–40 (noisy — some wrap their control and are FINE) |
| `no-autofocus` | `autoFocus` prop | 4 files: `AppShell.tsx`, `SavePresetDialog.tsx`, `InstantSearchBar.tsx`, `ManualSearchDialog.tsx` |
| `click-events-have-key-events` / `no-static-element-interactions` | `onClick` on `div/span/li/td` | 1 file: `library/PosterCardMenu.tsx` |
| `img` alt / anchor-is-valid / mouse-events-have-key-events / positive-tabindex | — | 0 hits (clean) |

⚠️ jsx-a11y is AST-based; grep both over- and under-counts (e.g. a `<label>` that wraps its `<input>` passes `label-has-associated-control` despite no `htmlFor`). Treat the table as a scoping sketch, not the contract. The real batch is whatever Task 1's eslint run prints.

### `no-autofocus` is a trap — do NOT blind-delete on dialogs

Most `autoFocus` hits here are dialogs/search inputs that SHOULD take focus on open — that is good modal a11y and is precisely what AI1's modal-focus-management pre-flight item asks for. `jsx-a11y/no-autofocus` flags the *prop* (it can disorient SR users in non-dialog contexts), but the accepted fix for a legitimate dialog is to focus programmatically via `ref` + `useEffect`, keeping the behavior. Deleting the focus outright would REGRESS the very a11y AI1 enforces — AC #4 guards this. Confirm each case against whether the element lives in a dialog/modal.

### Project Structure Notes

- Touches many files under `apps/web/src/components/**` (labels, dialogs, `PosterCardMenu`), plus `eslint.config.mjs` + `jsx-a11y-config.spec.ts` (the AI1 artifacts) for the ratchet.
- Cross-stack split check: backend tasks = **0** (no Go). FE work is broad but single-category (clear a11y warnings) + the config ratchet. BE=0 so the >3-each-side split threshold cannot be met → **single story**.
- No `.pen` edits. Screenshots only if Task 4.2 finds a pixel change → rebless via the standard Rule 22 flow.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched`. The a11y fixes (htmlFor/id, ref-focus, role/keyboard handlers) add no clock reads. If a touched component already reads the clock for unrelated reasons, this story does not change that path.
- Reference: `project-context.md` Rule 23.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#challenges`] Pattern #1 — a11y caught at CR not pre-flight (11-2 bottom sheet, 11-3 combobox)
- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#action-items`] retro-11-AI1b row (DEV/QD, MED — sub-item of AI1)
- [Source: `_bmad-output/implementation-artifacts/retro-11-AI1-a11y-enforcement-mechanism.md`] the gating story — jsx-a11y `warn` install + the deferred warn→error ratchet (AC #3 Dev Notes "warn→error is AI1b's closing move") + the `jsx-a11y-config.spec.ts` wiring spec this story updates
- [Source: `apps/web/src/components/library/PosterCardMenu.tsx`] the static-element `onClick` recon hit
- [Source: `apps/web/src/components/search/SavePresetDialog.tsx`, `manual-search/ManualSearchDialog.tsx`, `search/InstantSearchBar.tsx`, `shell/AppShell.tsx`] the `autoFocus` recon hits (verify vs Task 1 inventory)
- [Precedent: Story 10-2 H2/M1 + bugfix-10-6 M1] keyboard-affordance + focus remedies for the static-element / focus-ring fixes
- [Precedent: `retro-11-AI1`] sibling enforcement story — this one is its clean-up + ratchet half

### Out of Scope

- Widening jsx-a11y beyond `apps/web/src/components/**` (hooks/services/routes, `apps/web/src` root) — AI1 scoped it to components/; widening is a separate future decision.
- a11y improvements NOT flagged by the jsx-a11y recommended ruleset (e.g. color-contrast, which jsx-a11y cannot check statically) — out of scope; a dedicated audit if ever wanted.
- The MANDATORY dev-story a11y pre-flight action itself — that is AI1 prong 2, already authored.

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - Expected default: `N/A — no out-of-scope work discovered`. If Task 1's eslint inventory surfaces a violation class that needs a non-trivial refactor beyond a per-element a11y fix (e.g. a shared interactive primitive used in dozens of places), do NOT balloon this story — file it per Rule 24 lane ③ (`backlog`/`bugfix-N` entry, bidirectional link) and keep this story to the mechanical batch + the ratchet. If a fix is genuinely blocking the warn→error bump, use lane ② (spawn-blocking-story) instead.
- Reference: `project-context.md` Rule 24.

### File List

## Change Log

| Date | Change | Author |
| ---- | ------ | ------ |
| 2026-06-09 | Story created (SM Bob /create-story, YOLO) — set to `blocked` (depends on retro-11-AI1). Clears the jsx-a11y `warn` batch under apps/web/src/components/** (recon: ~label-has-associated-control cluster + 4 no-autofocus + 1 static-onClick PosterCardMenu; authoritative list = Task 1 eslint run) and ratchets jsx-a11y warn→error + flips the AI1 wiring spec assertion. Cross-stack: BE=0 → single story. | Bob (SM) |
