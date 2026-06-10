# Story: Clear the existing component a11y warning batch + ratchet jsx-a11y warn → error

Status: done

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

- [x] **Task 1: Confirm the AI1 gate + capture the authoritative inventory** (AC: #1, #2)
  - [x] 1.1 Verify `retro-11-AI1` is `done` in sprint-status and `eslint-plugin-jsx-a11y` resolves (`pnpm ls eslint-plugin-jsx-a11y`). If not done → HALT and record the blocker.
  - [x] 1.2 Run `pnpm exec eslint "apps/web/src/components/**/*.{ts,tsx}"` (or `pnpm lint:all` and filter `jsx-a11y/`). Capture the FULL warning list.
  - [x] 1.3 Record the inventory in Dev Agent Record grouped by `rule id → file:line → count`, plus a per-rule total. This is the work-list for Task 2 and the AC #2 artifact.

- [x] **Task 2: Clear the batch, category by category** (AC: #3, #4)
  - [x] 2.1 `label-has-associated-control`: for each flagged `<label>` (recon points at `settings/`, `setup/`, `metadata-editor/`, `library/SettingsGearDropdown.tsx` clusters), add `htmlFor` + a matching control `id` (or nest the control). Preserve Tailwind classes + Chinese copy. Add/extend the component's spec to assert the association (`getByLabelText` works).
  - [x] 2.2 `no-autofocus`: convert dialog `autoFocus` to ref+`useEffect` focus (keeps focus-on-open, satisfies AI1 modal-focus item); remove non-dialog `autoFocus`. Files from recon: `shell/AppShell.tsx`, `search/SavePresetDialog.tsx`, `search/InstantSearchBar.tsx`, `manual-search/ManualSearchDialog.tsx` (confirm against Task 1 inventory — recon, not authoritative). *(All 4 confirmed dialog/search-overlay contexts → all converted to ref+effect, none deleted; `InstantSearchBar` prop renamed `autoFocus` → `focusOnMount`.)*
  - [x] 2.3 `click-events-have-key-events` / `no-static-element-interactions`: fix `library/PosterCardMenu.tsx` (recon: the one static-element `onClick`) → real `<button>` or `role`+`tabIndex`+keyboard handler. Add keyboard-activation test. *(Inventory surfaced 14+5 across 14 files — dialog-backdrop class fixed via dedicated `aria-hidden` backdrop layer; genuinely interactive elements (`MediaFileCard`/`MediaFileRow`, `PosterUploader` dropzone) got `role="button"`+`tabIndex`+Enter/Space handlers + tests.)*
  - [x] 2.4 Any other rule the Task 1 inventory surfaced: fix per standard remedy. For each unavoidable suppression, add an inline justification + list it in Dev Agent Record. *(Suppressions added: **0**.)*
  - [x] 2.5 Re-run the scoped lint after each category; drive the jsx-a11y warning count to **0**. *(0 jsx-a11y findings under the ratcheted config — see Completion Notes for the Alexyu-ruled measurement basis.)*

- [x] **Task 3: Ratchet warn → error** (AC: #5)
  - [x] 3.1 In `eslint.config.mjs`, change the AI1 jsx-a11y block from the `jsxA11yWarn` remap to native `error` severity (apply `jsxA11y.flatConfigs.recommended` directly, or an explicit `error` map). Remove the now-unused `jsxA11yWarn` helper if nothing else uses it. *(Applied `jsxA11y.flatConfigs.recommended.rules` natively — AC #5's first option; the `Object.fromEntries` warn remap removed.)*
  - [x] 3.2 Update `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts`: the severity assertion flips `warn` → `error`; scope/ignores/plugin assertions unchanged. *(Asserts `alt-text` at `'error'`, every rule at native `error`/`off`, none at `warn`.)*
  - [x] 3.3 Run `pnpm lint:all` — confirm **0 errors** (a non-zero count means the batch was not fully cleared; return to Task 2). *(0 errors / 123 warnings — all non-a11y; see Completion Notes.)*

- [x] **Task 4: Regression + visual verification** (AC: #6)
  - [x] 4.1 `pnpm lint:all` (0 errors, 0 jsx-a11y warnings), `pnpm nx test api`, `pnpm nx test web` (incl. updated wiring spec + any new association/keyboard tests), `pnpm run test:cleanup`. *(web 2021/2021 PASS incl. flipped wiring spec 5/5 + 17 new a11y tests; api PASS; no orphaned processes.)*
  - [x] 4.2 Visual check: determine whether any fix altered rendered DOM/pixels (htmlFor/id are non-visual; keyboard handlers + role are non-visual; a NEW visible focus ring or a changed element tag could shift pixels). If pixels change → rebless the affected Rule 22 baselines (`-darwin` + `-linux` per the 19-5 CI flow) with a Sally/UX re-engagement note. If not → record `Visual: no rendered-output change`. *(**Visual: no rendered-output change** — `pnpm run test:visual` 1 passed against all committed `-darwin` baselines, 0 pixel diffs; element-tag swaps kept identical classNames; backdrop layers are transparent.)*
  - [x] 4.3 `pnpm exec prettier --check` on all touched files. *(Clean after `--write` on 5 reflowed files; full `prettier --check .` passes inside lint:all.)*

- [x] **Task 5: sprint-status transitions** (AC: #6)
  - [x] 5.1 Entry stays `blocked` until `retro-11-AI1` is `done`; on AI1 close, flip `blocked → ready-for-dev`. *(Done by AI1 close on 2026-06-09.)*
  - [x] 5.2 `ready-for-dev → in-progress` on `/dev-story` start; `in-progress → review` on completion (record the final jsx-a11y warning count = 0 + the warn→error bump confirmation).
  - [x] 5.3 `review → done` on `/code-review` pass. *(2026-06-10 adversarial CR PASS: 0H/3M/3L, all M + L1 fixed in-review — see Change Log.)*

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

claude-fable-5 (Fable 5) — DEV Amelia /dev-story 2026-06-10

### Debug Log References

- `pnpm ls eslint-plugin-jsx-a11y` → `eslint-plugin-jsx-a11y 6.10.2` (root devDependencies; AC #1 gate confirmed; sprint-status L472 `retro-11-AI1: done`).
- AC #2 authoritative inventory (warn-remap config, `pnpm exec eslint "apps/web/src/components/**/*.{ts,tsx}"`, 2026-06-10): **146 jsx-a11y warnings** — exact match to AI1 CR's +146 breakdown: 46 `label-has-for` / 37 `control-has-associated-label` / 24 `label-has-associated-control` / 16 `no-noninteractive-element-interactions` / 14 `click-events-have-key-events` / 5 `no-static-element-interactions` / 4 `no-autofocus`. Full `rule → file:line` listing in the AC #2 inventory table below.
- **Measurement run under the AC #5 end-state config** (native `jsxA11y.flatConfigs.recommended.rules`, temporary edit, reverted): **56 errors** — 24 `label-has-associated-control` / 14 `click-events-have-key-events` / 9 `no-noninteractive-element-interactions` (7 of 16 exempted by recommended's handler options) / 5 `no-static-element-interactions` / 4 `no-autofocus`. The 90-warning delta = 46 `label-has-for` + 37 `control-has-associated-label` (both natively `off` in recommended — the AI1 `Object.keys(...).map(r => [r,'warn'])` remap accidentally enabled them and dropped all rule options) + 7 option-exempted `no-noninteractive`.

#### AC #2 inventory (146 warnings, grouped rule → file:line)

| Rule (count) | Files:lines |
|---|---|
| `label-has-for` (46) | DownloadList:42; SettingsGearDropdown:80,103,120; CastEditor:44; GenreSelector:52; MetadataEditorDialog:249,272,291,316,340,359,402,421; PosterUploader:115; SavePresetDialog:112; BackupScheduleConfig:119,133,151; ExploreBlockEditModal:259; LibraryCard:117; LibraryEditModal:110,124,140; MediaLibraryManager:41; MetadataExport:54; QBittorrentForm:88,106,124,142; ScannerSettings:136,162; ApiKeysStep:19,38,61; MediaFolderStep:13; MediaLibrarySetupStep:66,81; QBittorrentStep:13,31,49; WelcomeStep:19; BatchSubtitleDialog:86,99; SubtitleSearchDialog:223,237 |
| `control-has-associated-label` (37) | DownloadPanel:160; QuickSearchBar:154; DownloadItem:35; LibraryTable:120; ManualSearchDialog:216; CastEditor:63; MetadataEditorDialog:379; PosterUploader:162,200; SavePresetDialog:118; BackupScheduleConfig:93; ExploreBlockEditModal:149,172,184,194,221; LibraryCard:118; LibraryEditModal:113,167; LogFilters:86; MetadataExport:63; QBittorrentForm:94,112,130,148; ApiKeysStep:25,67; MediaFolderStep:19; MediaLibrarySetupStep:69; QBittorrentStep:19,37,55; BatchSubtitleDialog:87,100; SubtitleSearchDialog:196,224,239 |
| `label-has-associated-control` (24) | SettingsGearDropdown:80,103,120; MetadataEditorDialog:249,272,291,316,340,359,402,421; PosterUploader:115; BackupScheduleConfig:119,133,151; LibraryEditModal:110,124,140; MediaLibraryManager:41; MetadataExport:54; ScannerSettings:162; MediaLibrarySetupStep:66,81; SubtitleSearchDialog:237 |
| `no-noninteractive-element-interactions` (16) | HeroBanner:61,195; TrailerModal:108; BatchConfirmDialog:60; ManualSearchDialog:162; SearchResultCard:60; MediaDetailPanel:114,159; PosterCard:161; MetadataEditorDialog:203; ScanProgressCard:113; PresetChips:111; SavePresetDialog:68; BatchSubtitleDialog:50; SubtitleSearchDialog:165; SidePanel:54 |
| `click-events-have-key-events` (14) | TrailerModal:108; BatchConfirmDialog:60; PosterCardMenu:103; ManualSearchDialog:162,170; MetadataEditorDialog:203; PosterUploader:148; MediaFileCard:52,127; PresetChips:111; SavePresetDialog:68; BatchSubtitleDialog:50; SubtitleSearchDialog:165; SidePanel:54 |
| `no-static-element-interactions` (5) | PosterCardMenu:103; ManualSearchDialog:170; PosterUploader:148; MediaFileCard:52,127 |
| `no-autofocus` (4) | ManualSearchDialog:221; InstantSearchBar:143; SavePresetDialog:128; AppShell:86 |

### Completion Notes List

- **🔗 AC Drift: FOUND (sanctioned hand-off) — `retro-11-AI1` AC #3/#4(e): jsx-a11y severity `warn` → `error`.** AI1 AC #3 explicitly states "A future bump to `error` is explicitly out of scope (it is the closing act of AI1b)" — the drift is the contract's own planned completion, executed here. Checked via `grep -ln "jsx-a11y" _bmad-output/implementation-artifacts/*.md` (3 hits: this story, AI1, the epic-11 retro — AI1's is the only prior AC; autofocus grep hits in 19-4b/19-8/bugfix-19-4b-1 are Sally observations, not ACs, and focus-on-open behavior is preserved → REUSE not DRIFT).
- **📎 Contract Stamps: NONE** (`grep -E '\[@contract-v[0-9]+\]'` → 0 hits in this story file and in retro-11-AI1's — both pre-Rule-20 process stories defining no wire contracts; no upstream stamped ACs consumed).
- **🔒 Rule 7 Wire Format: N/A** (pure frontend + lint config; no Go error codes, no JSON wire surface).
- **🎭 A11y Pre-Flight: PASS (29 components checked, 0 jsx-a11y warnings on touched files, 0 introduced by this story).** Per the 4 recurring classes: (1) responsive image sizing — no TMDb `<img>` added/modified; (2) modal focus management — focus-on-open *preserved* via ref+effect in all 4 former-autoFocus dialogs, Escape-dismiss *added* to `PosterCardMenu` (menu + mobile bottom sheet) and `PresetChips` delete-confirm (11-2 CR M2 class); (3) aria-live — no new async-revealed status content; progressbars gained `aria-label`; (4) keyboard+ARIA for custom widgets — dropzone/cards keyboardized (Enter/Space), switches named (`aria-label`/`aria-labelledby`), button groups labelled via `role="group"`+`aria-labelledby`. Lazy-load contract accuracy: N/A (no IntersectionObserver/pagination touched).
- **Batch disposition (146 warn-remap warnings → 0):** 56 native-recommended errors fixed with correct remedies (24 `label-has-associated-control` htmlFor/id or group-label; 14+5 click/static via backdrop-layer or role+tabIndex+keyboard; 9 `no-noninteractive` same; 4 `no-autofocus` → ref+effect). 90 warn-remap-only artifacts (46 deprecated `label-has-for` + 37 natively-off `control-has-associated-label` + 7 option-exempted `no-noninteractive`) eliminated by the Task 3 native-config ratchet — config correction per the Alexyu ruling, NOT suppression. Within the 37, the genuine accessible-name gaps were ALSO fixed per the ruling: aria-labels on `QuickSearchBar`/`LogFilters`/`SubtitleSearchDialog`/`ManualSearchDialog`/`PosterUploader` inputs, `DownloadPanel`/`DownloadItem` progressbars, `LibraryTable` selection-column `th`, `BackupScheduleConfig`+`SubtitleSearchDialog` switches, `LibraryEditModal` close/remove-path buttons, `MetadataEditorDialog` cast-remove buttons, `MetadataExport` option labels; htmlFor/id for `CastEditor` (useId), `ExploreBlockEditModal` Field (useId+cloneElement), `MediaLibrarySetupStep` (indexed ids). Remaining were verified false positives (controls already associated via htmlFor+id or label-nesting: `QBittorrentForm`×4, `ApiKeysStep`×2, `QBittorrentStep`×3, `MediaFolderStep`, `SavePresetDialog`, `BatchSubtitleDialog`×2, `LibraryCard`, `SubtitleSearchDialog` checkbox).
- **eslint-disable suppressions added: 0.**
- **Pre-existing micro-bug fixed in passing (`PosterUploader`):** programmatic `fileInput.click()` bubbled back to the dropzone `onClick`, double-triggering the picker — `stopPropagation` at the input (caught by the new keyboard test's spy count).
- **lint:all warnings ledger:** 268 (AI1 close) → **123** = 122 pre-AI1 baseline + 1 new `@typescript-eslint/no-explicit-any` from the added `BackupScheduleConfig.spec` test following that file's established `as any` mock pattern. 0 errors; 0 jsx-a11y.
- **⚖️ AC #3 measurement-basis ruling (Alexyu, 2026-06-10, Rule 24 lane ①):** the AI1 warn-remap accidentally enabled 2 rules recommended ships as `off` (deprecated `label-has-for` — whose default `{every: [nesting, id]}` is unsatisfiable via `htmlFor` alone — and `control-has-associated-label`) and dropped all rule options. Literal zero under the warn-remap is impossible without mass DOM restructuring. Ruling: **AC #3 "zero" is measured against the AC #5 end-state config (native recommended @ error) → fix the 56 native errors with correct remedies, PLUS add `aria-label` to the genuine unlabeled-icon-button subset of the 37 `control-has-associated-label` hits** (real a11y wins, even though that rule stays natively `off` post-ratchet). The remaining warn-remap-only artifacts disappear with the remap itself in Task 3 — config correction, not suppression.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **One discovery, triaged lane ① (expand-scope-in-place):** the AI1 warn-remap implementation artifact — `Object.keys(recommended.rules).map(r => [r,'warn'])` accidentally enabled 2 natively-`off` rules and dropped all rule options, making the literal AC #3 "zero under warn-remap" unsatisfiable (deprecated `label-has-for` demands nesting+id simultaneously). Surfaced at Task 1, escalated to Alexyu 2026-06-10 → ruling recorded above (measure AC #3 against the AC #5 native end-state + fix genuine icon-button/aria-label gaps among the off-rule hits). Absorbed into this story per Rule 24 lane ① — the tracking artifact is the ruling note + this entry (the fix IS this story's Task 2/3 work; no separate sprint-status entry needed since no work leaves this story).
  - Also absorbed under lane ① (same surface, tiny): `PosterUploader` double-`click()` bubbling micro-bug; Escape-dismiss for `PosterCardMenu`/`PresetChips` (pre-flight class-4 gap on touched dialogs).
  - **Two lane ③ entries formalized at /code-review 2026-06-10 (CR M3 — they were prose-only "Deferred (noted for PR body)" in the /ship self-review row, the exact Rule 24 banned pattern):** `disc-2026-06-shared-dialog-dismiss-layer` (L7 — extract the 7×-duplicated aria-hidden dismiss-backdrop pattern into a shared layer/hook) and `disc-2026-06-component-id-useid-hygiene` (L9 — migrate hardcoded htmlFor/id literals to `useId`, matching CastEditor/ExploreBlockEditModal). Both filed `backlog` P3 in sprint-status.yaml with bidirectional links back to this story.
- Reference: `project-context.md` Rule 24.

### File List

**Config + wiring spec (the ratchet):**

- `eslint.config.mjs` (jsx-a11y block: warn remap → native `jsxA11y.flatConfigs.recommended.rules`)
- `apps/web/src/eslint-rules/jsx-a11y-config.spec.ts` (severity assertion warn → error/off-native)

**Components (a11y remedies):**

- `apps/web/src/components/dashboard/DownloadPanel.tsx`
- `apps/web/src/components/dashboard/QuickSearchBar.tsx`
- `apps/web/src/components/downloads/DownloadItem.tsx`
- `apps/web/src/components/homepage/TrailerModal.tsx`
- `apps/web/src/components/library/BatchConfirmDialog.tsx`
- `apps/web/src/components/library/LibraryTable.tsx`
- `apps/web/src/components/library/PosterCardMenu.tsx`
- `apps/web/src/components/library/SettingsGearDropdown.tsx`
- `apps/web/src/components/manual-search/ManualSearchDialog.tsx`
- `apps/web/src/components/metadata-editor/CastEditor.tsx`
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.tsx`
- `apps/web/src/components/metadata-editor/PosterUploader.tsx`
- `apps/web/src/components/parse/MediaFileCard.tsx`
- `apps/web/src/components/search/InstantSearchBar.tsx`
- `apps/web/src/components/search/PresetChips.tsx`
- `apps/web/src/components/search/SavePresetDialog.tsx`
- `apps/web/src/components/settings/BackupScheduleConfig.tsx`
- `apps/web/src/components/settings/ExploreBlockEditModal.tsx`
- `apps/web/src/components/settings/LibraryEditModal.tsx`
- `apps/web/src/components/settings/LogFilters.tsx`
- `apps/web/src/components/settings/MediaLibraryManager.tsx`
- `apps/web/src/components/settings/MetadataExport.tsx`
- `apps/web/src/components/settings/ScannerSettings.tsx`
- `apps/web/src/components/setup/MediaLibrarySetupStep.tsx`
- `apps/web/src/components/shell/AppShell.tsx`
- `apps/web/src/components/subtitle/BatchSubtitleDialog.tsx`
- `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx`
- `apps/web/src/components/ui/SidePanel.tsx`

**Specs (extended):**

- `apps/web/src/components/homepage/TrailerModal.spec.tsx`
- `apps/web/src/components/library/PosterCardMenu.spec.tsx`
- `apps/web/src/components/library/SettingsGearDropdown.spec.tsx`
- `apps/web/src/components/manual-search/ManualSearchDialog.spec.tsx`
- `apps/web/src/components/metadata-editor/MetadataEditorDialog.spec.tsx`
- `apps/web/src/components/metadata-editor/PosterUploader.spec.tsx`
- `apps/web/src/components/parse/MediaFileCard.spec.tsx`
- `apps/web/src/components/search/InstantSearchBar.spec.tsx`
- `apps/web/src/components/search/PresetChips.spec.tsx`
- `apps/web/src/components/search/SavePresetDialog.spec.tsx`
- `apps/web/src/components/settings/BackupScheduleConfig.spec.tsx`
- `apps/web/src/components/settings/ExploreBlockEditModal.spec.tsx` *(was missing from this list — CR 2026-06-10 M1)*
- `apps/web/src/components/subtitle/SubtitleSearchDialog.spec.tsx`
- `apps/web/src/components/ui/SidePanel.spec.tsx`

**Specs (new):**

- `apps/web/src/components/settings/LibraryEditModal.spec.tsx`
- `apps/web/src/components/setup/MediaLibrarySetupStep.spec.tsx`

**Tracking:**

- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/retro-11-AI1b-clear-a11y-warning-batch.md` (this file)
- `_bmad-output/implementation-artifacts/retro-11-AI1-a11y-enforcement-mechanism.md` (AC drift reference — see Completion Notes; file itself unmodified)

## Change Log

| Date | Change | Author |
| ---- | ------ | ------ |
| 2026-06-10 | /code-review adversarial CR (post-merge vs 24431d4) PASS → done. All 6 ACs verified with live evidence (scoped eslint 0 errors / 0 jsx-a11y; wiring spec 6/6; 16 touched spec files 200/200; Rule 7/20/25 all N/A). Findings 0H/3M/3L, fixed in-review: M1 File List was missing ExploreBlockEditModal.spec.tsx (added); M2 PosterCardMenu Escape handler called onClose inside the setShowConfirm updater — impure updater double-invoked under StrictMode (main.tsx wraps the app) → rewritten to read showConfirm via effect deps; M3 Rule 24 violation — /ship-deferred L7/L9 existed only as PR-body prose → formalized as lane ③ backlog entries disc-2026-06-shared-dialog-dismiss-layer + disc-2026-06-component-id-useid-hygiene (bidirectional links recorded in Discovery Triage); L1 MediaFileRow keyboard spec now also asserts role/tabindex (parity with Card variant). L2/L3 folded into the two new backlog entries. Re-verified post-fix: PosterCardMenu + MediaFileCard specs green, scoped lint 0 errors, prettier clean. | Amelia (DEV /code-review) |
| 2026-06-10 | /ship adversarial self-review (clean-context subagent, 12 findings: 1H/6M/5L) — ALL in-scope findings fixed: H1 繁體轉換 text-click forwarding restored (label htmlFor → button id, + forwarding test); M4 MediaFileCard/Row role/tabIndex now conditional on onClick (no actionless tab-stops); M3 PosterUploader 海報圖片 reverted to span heading (dangling htmlFor in URL mode + surprise label-click→picker), file input aria-label'd; M5 LibraryEditModal 新增路徑 Plus button aria-label + test; M6 wiring spec now pins rules toEqual(recommended.rules) (anti-shrinkage) + header comment rot fixed; M7 PosterCardMenu Escape disarms delete-confirm before closing (+ test); M2 MetadataEditorDialog no-backdrop-dismiss documented as deliberate (edit form, was dead code before); L8 MetadataExport aria-label keeps description in accname; L11 BatchSubtitleDialog comment corrected (Escape exists); L10 ExploreBlockEditModal Field association test added; L12 implementation-detail assertion swapped. Deferred (noted for PR body): L9 hardcoded ids in singleton components (useId churn), L7 global shared dismiss-layer utility (cross-cutting). Re-verified: lint:all 0 errors, 0 jsx-a11y, web 2025/2025, api PASS, test:visual 0 diffs, cleanup clean. | Amelia (DEV /ship) |
| 2026-06-10 | Tasks 2–5 complete (DEV Amelia /dev-story): 56 native-recommended violations fixed across 28 components (0 suppressions) — htmlFor/id+group labels, dedicated aria-hidden backdrop layers, role/tabIndex/Enter-Space keyboardization, autoFocus→ref+effect (focus-on-open preserved; InstantSearchBar prop renamed focusOnMount); icon-button/progressbar/switch aria-labels per Alexyu ruling; Escape-dismiss added to PosterCardMenu + PresetChips delete-confirm. Ratchet: eslint.config.mjs → native recommended @ error (off rules stay off, options restored); wiring spec flipped to error assertion. Gates: lint:all 0 errors (123 warns = 122 baseline +1 spec-pattern `as any`), web 2021/2021 (incl. 17 new a11y tests + 2 new spec files), api PASS, test:visual 1 passed / 0 pixel diffs, test:cleanup clean. 🎭 A11y Pre-Flight PASS. Status → review (5.3 deferred to /code-review per AI1 precedent). | Amelia (DEV) |
| 2026-06-10 | Task 1 complete (DEV Amelia /dev-story): AI1 gate confirmed (jsx-a11y 6.10.2, AI1 done); authoritative 146-warning inventory captured (exact match to AI1 CR breakdown); end-state measurement run → 56 native-recommended errors; Alexyu ruling: clear-to-zero basis = native recommended @ error + icon-button aria-labels (Rule 24 lane ①, recorded in Completion Notes). | Amelia (DEV) |
| 2026-06-09 | Story created (SM Bob /create-story, YOLO) — set to `blocked` (depends on retro-11-AI1). Clears the jsx-a11y `warn` batch under apps/web/src/components/** (recon: ~label-has-associated-control cluster + 4 no-autofocus + 1 static-onClick PosterCardMenu; authoritative list = Task 1 eslint run) and ratchets jsx-a11y warn→error + flips the AI1 wiring spec assertion. Cross-stack: BE=0 → single story. | Bob (SM) |
