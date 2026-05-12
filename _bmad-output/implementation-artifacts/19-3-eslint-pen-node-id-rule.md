# Story 19.3: ESLint Rule — Enforce Component-to-Design Node Traceability (Rule 21 Phase 2)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a frontend maintainer,
I want a custom ESLint rule that fails the build when a file under `apps/web/src/components/` lacks (or malforms) its `// Implements: Component/{Name} ({pen-node-id})` header,
so that design-implementation drift cannot silently re-enter the codebase the way `HoverPreviewCard.tsx` did (bugfix-10-4 root cause) — turning project-context.md Rule 21 from a manual SM-template gate into a CI-enforced invariant.

## Acceptance Criteria

1. [@contract-v2] A custom ESLint rule (`implements-pen-node-id` under a local plugin, e.g. `local/implements-pen-node-id`) exists and is registered in `eslint.config.mjs` for files matching `apps/web/src/components/**/*.{ts,tsx}`. The rule reports a lint **error** (not warning) when a component file's leading comment block does **not** contain a line matching one of:
   - `// Implements: Component/{Name} ({nodeId})` — `{Name}` is a non-empty identifier-ish token (allow letters, digits, `-`), `{nodeId}` is a non-empty token of letters/digits. Multiple components on one line joined by ` + ` are allowed (precedent: `PosterCard.tsx` → `// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)`).
   - `// Implements: <utility — no .pen counterpart>` (pure layout/utility component exemption)
   - `// Implements: <route-only>` (one-off route-level wrapper exemption)
   - `// Implements: <screen-section — pending epic-19-8 mapping>` (Phase-2 placeholder — component renders a section of a designed *screen frame*, not a Reusable Component; canonical mapping tracked by epic-19-8. Em-dash or hyphen accepted; epic number generalised as `epic-{N}-{M}`. Added per Sally+Amelia+Bob Party Mode 2026-05-12 — see Change Log [@contract-v1→v2].)
2. [@contract-v2] The "leading comment block" is defined as: comments appearing before the first non-comment, non-whitespace token of the file (i.e. before the first `import`/`export`/statement). A malformed marker anywhere else in the file does **not** satisfy the rule. Both `//` line comments and `/* */`/JSDoc block comments are scanned (a leading `*` per line is stripped). The rule message names the accepted forms (incl. the `<screen-section …>` placeholder) and tells the author how to obtain the node ID (query `ux-design.pen` via Pencil MCP `get_editor_state` → "Reusable Components").
3. [@contract-v1] The rule does **not** apply to (and reports nothing for): `*.spec.tsx` / `*.spec.ts` / `*.test.tsx` files, files under `apps/web/src/hooks/`, `apps/web/src/services/`, `apps/web/src/stores/`, `apps/web/src/utils/`, and `apps/web/src/routes/` (route files are exempt per Rule 21 — only `components/` files that render designed UI are in scope). An `index.ts` barrel re-export file under `components/` is also exempt (no rendered UI). Scoping is done via ESLint flat-config `files`/`ignores` for the rule's config object, not inside the rule.
4. The rule auto-fix is **not** provided (node IDs cannot be invented — author must look them up). The rule MAY provide a suggestion message only.
5. All existing files under `apps/web/src/components/` that the rule scopes (`*.{ts,tsx}` minus `*.spec`/`*.test`/`index.ts`) carry a valid Rule 21 header after this story — either a real `// Implements: Component/{Name} ({nodeId})` line backfilled from the `.pen` node-ID mapping, a deliberate exemption (`<utility — no .pen counterpart>` / `<route-only>`), or the `<screen-section — pending epic-19-8 mapping>` placeholder (for components rendering a section of a designed screen frame — see AC #1). The mapping is sourced by querying `ux-design.pen` via Pencil MCP `get_editor_state` (its "Reusable Components" listing gives `{Name} → {nodeId}`). Nothing left bare. Already-headed files (`PosterCard.tsx`, `EmptyNoQBT.tsx`, `EmptyNoFolder.tsx`, `EmptyReadyForScan.tsx`) are left as-is if already valid.
6. `pnpm lint:all` is **green** at story close (0 errors). The new rule must not introduce errors that weren't backfilled in AC #5. No `components/` file gets a *false* exemption to silence lint: genuine utilities → `<utility>`, components with a designed screen-frame section but no Reusable Component node → `<screen-section …>` (honest — states the design exists, mapping pending), un-mappable-and-not-utility/screen-section would be a **material drift finding** in `_bmad-output/audit/drift-19-3-2026-05.md` (none this story).
7. The rule has its own unit test (Vitest or ESLint `RuleTester`) co-located with the rule file, covering: (a) valid `Component/X (id)` header passes, (b) valid multi-component `A (id) + B (id)` header passes, (c) all three exemption/placeholder forms pass — `<utility …>` (em-dash + hyphen), `<route-only>`, `<screen-section — pending epic-19-8 mapping>` (em-dash + hyphen), (d) missing header → error, (e) malformed headers → error (`// Implements: PosterCard` with no `Component/` prefix; `Component/X` with no `(id)`; empty `(id)`; bare `<screen-section>` without the `pending epic-N-M mapping` clause; arbitrary `<some other reason>`), (f) a valid marker that appears only *after* an `import` → still error (not in leading block), (g) a spec/hook/route file is not flagged (verified via assertions on the `eslint.config.mjs` `files`/`ignores` wiring).
8. `eslint.config.mjs` change is minimal and additive: register the local plugin + add one config object scoping the rule to `apps/web/src/components/**` with the documented `ignores`. No change to existing rule severities. The four-tool `pnpm lint:all` order (go vet → staticcheck → eslint → prettier) is unaffected; `eslint .` still covers `apps/web/`, `libs/shared-types/`, `tests/`.
9. `project-context.md` Rule 21 is updated: (a) "Enforcement" Phase 2 line → present-tense statement naming the rule id (`local/implements-pen-node-id`) and where it lives; (b) the accepted-marker grammar block gains the multi-component ` + ` form and the `<screen-section — pending epic-19-8 mapping>` placeholder (authorised by the Sally+Amelia+Bob Party Mode 2026-05-12 ruling — see Change Log). The `Last Updated` header line gets a story-19-3 entry. No other Rule 21 prose changes.

## Tasks / Subtasks

- [x] Task 1: Author the custom ESLint rule (AC: #1, #2, #3, #4)
  - [x] Created `apps/web/src/eslint-rules/implements-pen-node-id.js` (CJS — repo `type` is commonjs; `eslint.config.mjs` default-imports it as `module.exports`; lives under `src/` so `pnpm nx test web`'s vitest can co-locate the spec)
  - [x] `meta.type = 'problem'`, no `fixable`; visits `Program`, takes comments with `range[1] <= firstStatement.range[0]` (all comments if body is empty), normalises `//` and `/* */`/JSDoc comment lines (strips leading `*`), regex-matches the 4 accepted forms (`Component/X (id)` ` + `-joined multi; `<utility — no .pen counterpart>`; `<route-only>`; `<screen-section — pending epic-{N}-{M} mapping>` — em-dash/hyphen tolerated on the two `< … — … >` forms) — the 4th form added per the 2026-05-12 Party Mode ruling
  - [x] Reports on `Program` at line 1 with `messageId: 'missing'` — message names the accepted forms + Pencil MCP `get_editor_state` lookup hint
  - [x] `module.exports = { rules: { 'implements-pen-node-id': rule } }`
- [x] Task 2: Wire the rule into `eslint.config.mjs` (AC: #1, #3, #8)
  - [x] Added `import localRules from './apps/web/src/eslint-rules/implements-pen-node-id.js'` + ONE config object: `{ files: ['apps/web/src/components/**/*.{ts,tsx}'], ignores: ['apps/web/src/components/**/*.spec.{ts,tsx}', 'apps/web/src/components/**/*.test.{ts,tsx}', 'apps/web/src/components/**/index.ts'], plugins: { local: localRules }, rules: { 'local/implements-pen-node-id': 'error' } }`. Confirmed hooks/services/stores/utils/routes are outside `components/**` ⇒ no extra ignores needed.
  - [x] Placed after the JS-files config block, immediately before the `prettier` block (prettier stays last)
  - [x] `npx eslint apps/web/src/components` → 127 files flagged (= 131 candidates − 4 already-headed) — this was the AC #5 worklist
- [x] Task 3: Build the `.pen` node-ID mapping (AC: #5)
  - [x] Queried `ux-design.pen` via Pencil MCP `get_editor_state` (read-only) → 399 Reusable Components; the app-specific set is the unprefixed `Component/*` nodes (`RusTY`/`otvKh`/`YDPhc`/`L1NP6`/`TboA7`/`j98G4`/`6MxLT`/`MQbvp`/`jD7gF`/`955EZ`/`4EHFN`/`Wd9AL`/`L9m19`/`9iTW3`/`f84BM`/`cUjyv`/`fSKuT`/`U3SGxG`/`mfKgm`)
  - [x] Classified all 131 candidates: Category A = 12 mapped to Reusable Components (4 pre-existing + 8 new), Category B = 25 genuine pure-utility/layout/helper/`.ts`-module exemptions (`<utility — no .pen counterpart>`), Category C = 94 screen-section feature components → `<screen-section — pending epic-19-8 mapping>` (canonical screen-frame mapping deferred to epic-19-8 — NOT drift). No file given a false exemption (AC #6).
  - [x] Recorded the full classification + per-file mapping table + 19-8 worklist in `_bmad-output/audit/drift-19-3-2026-05.md`
- [x] Task 4: Backfill headers across `apps/web/src/components/` (AC: #5, #6)
  - [x] `scripts/backfill-rule21-headers.mjs` (idempotent) prepended `// Implements: Component/X (id)\n// Source: ux-design.pen (Pencil app)` to the 8 Category-A files; the 4 pre-existing-headed files left untouched. Initial pass put `<utility — no .pen counterpart>` on the remaining 119; after the 2026-05-12 Party Mode ruling, 94 of those (Category C) were flipped to `<screen-section — pending epic-19-8 mapping>` (the script's `CATEGORY_B` set / B-vs-C split recorded which stays which). _(CR follow-up 2026-05-12: the one-shot script was removed once the backfill landed — the durable A/B/C mapping record lives in `_bmad-output/audit/drift-19-3-2026-05.md`.)_
  - [x] Only `apps/web/src/components/**` `.tsx`/non-`index.ts` `.ts` files touched; no `*.spec.tsx`, nothing outside `components/`
  - [x] No file is un-mappable-and-mislabelled → zero material-drift findings filed (recorded as such in the audit doc; the Category-C deferral mirrors the bugfix-10-6 precedent)
- [x] Task 5: Unit-test the rule (AC: #7)
  - [x] `apps/web/src/eslint-rules/implements-pen-node-id.spec.ts` — ESLint `RuleTester` (uses vitest's ambient `describe`/`it`) + a `describe` block asserting the `eslint.config.mjs` wiring (rule registered at `error`, scoped to `components/**/*.{ts,tsx}`, ignores spec/test/index.ts) — covers AC #7 (a) valid single, (b) valid multi, (c) all exemption/placeholder forms (`<utility …>` em-dash + hyphen, `<route-only>`, `<screen-section — pending epic-19-8 mapping>` em-dash + hyphen) + JSDoc form + below-other-comments, (d) missing, (e) 5× malformed (no `Component/`; no `(id)`; empty `(id)`; bare `<screen-section>`; arbitrary `<…>`), (f) marker-after-import → still error, (g) out-of-scope handled via the config-wiring assertions
  - [x] Runs under `pnpm nx test web` (22 tests, all pass — 9 valid + 8 invalid `RuleTester` cases + 5 `eslint.config.mjs` wiring assertions; vitest `include` covers `src/**/*.spec.ts`. Was 19 at dev-story close; CR follow-up 2026-05-12 added the hyphenated-node-id invalid case and rewrote the wiring block to inspect the resolved flat-config object instead of substring-matching the file text.)
- [x] Task 6: Update `project-context.md` Rule 21 Enforcement block (AC: #9)
  - [x] Phase 2 bullet rewritten to present tense ("LIVE since story 19-3"), names `local/implements-pen-node-id` + file path + that it runs inside `eslint .` ⇒ `pnpm lint:all` ⇒ CI
  - [x] (post-Party-Mode) grammar block gained the multi-component ` + ` form + the `<screen-section — pending epic-19-8 mapping>` placeholder (with the "don't use this for components with genuinely no design" caveat); `Last Updated` header got a story-19-3 entry
- [x] Task 7: Full regression + close (AC: #6, #8)
  - [x] `pnpm lint:all` → 0 errors / 122 warnings (matches bugfix-10-7 closeout baseline exactly; new rule adds 0 errors), prettier `--check .` clean
  - [x] `pnpm nx test web` → 148 files / 1840 tests pass (includes the new rule's 22 tests; 1837 at dev-story close, +3 from the CR follow-up)
  - [x] `pnpm nx test api` → pass (full regression gate, Epic 9 Retro AI-1)
  - [x] `pnpm run test:cleanup` → no orphaned processes
  - [x] `ux-design.pen` not modified (read-only via MCP) — `scripts/export-pen-screenshots.py` NOT run; CLAUDE.md screenshot workflow does not trigger

## Dev Notes

### Why this story exists

bugfix-10-4 root cause (Party Mode 2026-05-08): `HoverPreviewCard.tsx` was independently invented and diverged from the `.pen` `Component/PosterCardHover` (node `MQbvp`) for months because nothing linked code back to the design contract. Rule 21 (added to `project-context.md` L654 in 19-1, commit `6c0cbf2`) made the `// Implements:` header mandatory; Phase 1 enforcement is the SM `/create-story` template gate. This story is **Phase 2**: machine enforcement so the gate can't be bypassed by hand-written files.

### Architecture / constraints

- **All frontend** — 0 Go, 0 migrations, 0 swagger, 0 backend tests. Per the create-story cross-stack split check, single story is correct (backend task count = 0, so the ">3 each side" split threshold is not met).
- **ESLint flat config** — repo uses `eslint.config.mjs` (flat config, ESM). Custom rules register as a plugin object in a config entry's `plugins` map. There is currently NO local-rules infrastructure — this story creates it. Keep it minimal: one rule file + one config object.
- **`pnpm lint:all`** runs `eslint .` from repo root as step 3 of 4 (`go vet` → `staticcheck@2026.1` → `eslint .` → `prettier --check .`), mirroring CI's `lint` job (`.github/workflows/test.yml`). The new rule runs inside that `eslint .`. (Rule 12 in `project-context.md`.)
- **`.pen` access** — `ux-design.pen` is encrypted; access ONLY via Pencil MCP tools (`get_editor_state` lists reusable components with node IDs). NEVER `Read`/`Grep` the `.pen` file. Pencil.app must be running for MCP calls to work. (CLAUDE.md "Key Paths" + Pencil MCP server instructions.)
- **Rule 21 marker grammar** (authoritative — `project-context.md` Rule 21):
  - `// Implements: Component/{Name} ({pen-node-id})`
  - exemptions: `// Implements: <utility — no .pen counterpart>` / `// Implements: <route-only>`
  - tests/hooks/services/stores: exempt, no annotation required
  - in-repo precedent: `apps/web/src/components/media/PosterCard.tsx:1` (`// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)`), `apps/web/src/components/library/EmptyNoFolder.tsx:1` (`// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)`)
- **Don't fake exemptions** — AC #6 is explicit: a `components/` file with no `.pen` node that isn't a utility/route wrapper is a real drift finding for epic-19-8, not a lint-silencing opportunity. This is the whole point of the rule.

### Dependencies / sequencing (from `sprint-status.yaml` epic-19 block)

- Depends on **19-1** (done — Rule 21 in `project-context.md`).
- **19-4** (Playwright visual baselines) depends on this story — it needs the pen-node mapping table produced in Task 3 (`drift-19-3-2026-05.md`).
- **19-8** (comprehensive component sweep) depends on this rule being live.
- Estimate: ~3 days (Amelia), per `sprint-status.yaml:522`.

### Rule 22 / audit linkage

This story produces `_bmad-output/audit/drift-19-3-2026-05.md` (the file→node mapping + any material-drift findings). That doc is the durable record 19-4 and 19-8 build on, and feeds the epic-19 retro's Rule 22 audit.

### Testing standards

- Frontend tests: Vitest + RTL, co-located (`*.spec.tsx`). For an ESLint rule, ESLint's built-in `RuleTester` is the idiomatic choice and exercises the real AST visitor — prefer it. Co-locate the test next to the rule file.
- Run via `pnpm nx test web` (root or `apps/web`). After ANY test run: `pnpm run test:cleanup` (CLAUDE.md "Test Process Cleanup").
- Lint gate: `pnpm lint:all` must be 0 errors at close; warnings should match the prior closeout baseline (bugfix-10-7).
- Assertion quality: Rule 16 — `toBeInTheDocument` / `toEqual` / `toThrow`, not `toBeTruthy`.

### Project Structure Notes

- New file: `eslint-local-rules/implements-pen-node-id.js` (+ co-located test) — pick the exact dir name during implementation; keep ESM. Update `eslint.config.mjs` import path accordingly.
- Touched: `eslint.config.mjs` (one config object + one import), `project-context.md` (Rule 21 Enforcement bullet only), ~120ish files under `apps/web/src/components/` (one header line each), `_bmad-output/audit/drift-19-3-2026-05.md` (new).
- Out of scope: Playwright visual baselines (19-4), GitHub Actions visual regression (19-5), TestSprite cron (19-6/19-7), the full component-vs-`.pen` sweep (19-8), any actual drift *fixes* (those become bugfix-N stories per Rule 22).

### References

- [Source: project-context.md#Rule-21-Component-to-Design-Node-Traceability] — marker grammar, exemptions, Phase 1/Phase 2 enforcement plan
- [Source: project-context.md#Rule-12-Code-Quality-Checks-CI-based] — `pnpm lint:all` tool order + scope
- [Source: project-context.md#Rule-22-Epic-Retro-Design-Drift-Audit] — audit doc convention `_bmad-output/audit/drift-{epic}-{YYYY-MM}.md`
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:506-528] — epic-19 dependency notes + 19-3 line (`19-3-eslint-pen-node-id-rule: backlog`)
- [Source: eslint.config.mjs] — current flat-config structure (where to insert the new config object: after the TS block, before `prettier`)
- [Source: apps/web/src/components/media/PosterCard.tsx:1] — multi-component header precedent
- [Source: apps/web/src/components/library/EmptyNoFolder.tsx:1] — single-component header precedent
- [Source: CLAUDE.md] — `.pen` MCP-only access; screenshot workflow only triggers on `.pen` *modification* (not the case here)
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:490] — bugfix-10-4 root-cause narrative (why Rule 21 exists)

## Dev Agent Record

### Agent Model Used

claude-opus-4-7[1m] (Amelia / dev-story workflow)

### Debug Log References

- `npx eslint apps/web/src/components` after wiring → 127 errors (the AC #5 worklist), 0 after backfill
- `pnpm nx test web` → 148 files / 1840 tests pass (1837 at dev-story close; +3 CR follow-up)
- `pnpm nx test api` → pass
- `pnpm lint:all` → 0 errors / 122 warnings (matches bugfix-10-7 baseline) + prettier clean
- (CR follow-up) `npx eslint apps/web/src/components` after tightening the `{nodeId}` regex → still 0 errors (no backfilled header broke)

### Completion Notes List

- **🔗 AC Drift: NONE** (checked: `grep -rn 'Implements: Component'|'pen-node'|'implements-pen-node'` across `_bmad-output/implementation-artifacts/*.md` — hits are PosterCard (bugfix-10-4), the EmptyLibrary trio (bugfix-10-5), and the ExploreBlock soft-ref note (bugfix-10-6); all REUSE not DRIFT. This story adds comment-only `// Implements:` headers + a new lint rule — it changes no external/runtime contract of any prior story's component.)
- **📎 Contract Stamps: FOUND** — this story carries `[@contract-v2]` on AC #1 + AC #2 (the 2026-05-12 Party Mode bump that added the 4th `<screen-section …>` marker form + the block-comment-scan note — Change Log carries the `[@contract-v1→v2]` row with both "what changed" + "what breaks downstream"; the original `[@contract-v0→v1]` row is also present) and `[@contract-v1]` on AC #3 (unchanged — scope/ignore set untouched). Upstream Story 19-1 is pre-Rule-20 (done via Party Mode, no story file) → grep returns 0 stamps → implicit v0, ack requirement skipped per Rule 20 v0-fallback. No downstream story consumes these stamps yet.
- **🔒 Rule 7 Wire Format: N/A** — pure frontend story, no Go error codes touched.
- **🎨 UX Verification: N/A** — the only changes to component files are leading `// Implements:` comment headers; zero rendered-output change, so no design-screenshot comparison applies. `ux-design.pen` was read via Pencil MCP (`get_editor_state`) but **not modified** → `scripts/export-pen-screenshots.py` not run, CLAUDE.md screenshot workflow does not trigger.
- **Header backfill classification** (full table in `_bmad-output/audit/drift-19-3-2026-05.md`): Category A (12 files) = real `// Implements: Component/X (id)` mapped to a `.pen` Reusable Component (4 pre-existing — PosterCard, EmptyNoQBT/NoFolder/ReadyForScan; 8 new — `ui/Button.tsx`→ButtonPrimary+ButtonSecondary, `search/SearchBar.tsx`→SearchInput, `search/MediaTypeTabs.tsx`+`shell/TabNavigation.tsx`→TabActive+TabInactive, `library/FilterChips.tsx`→FilterChip, `library/SortSelector.tsx`→SortDropdown, `metadata-editor/GenreSelector.tsx`→GenreTag, `media/TechBadge.tsx`→4×TechBadge). Category B (25 files) = genuine pure-utility/layout/helper/`.ts`-module exemptions → `// Implements: <utility — no .pen counterpart>`. Category C (94 files) = screen-section feature components → `// Implements: <screen-section — pending epic-19-8 mapping>` (the 4th marker form, added per the 2026-05-12 Party Mode ruling — Sally flagged that `<utility>` was inaccurate for these since the design *does* exist as a screen-frame section; same gray zone bugfix-10-6 hit with ExploreBlock). Canonical screen-frame mapping is tracked as a single batch item for epic-19-8, **not** 94 individual drift findings. **No file given a false exemption** (AC #6) — nothing is both un-mappable and mislabelled.
- **No new dependencies** — the ESLint rule is plain JS using only the `eslint` API already present; `RuleTester` imported from `eslint` (existing dep).
- **Pre-existing test failures:** none detected — `pnpm nx test web` and `pnpm nx test api` both fully green.

### File List

**New:**
- `apps/web/src/eslint-rules/implements-pen-node-id.js` — the custom ESLint rule (CJS plugin); 4 accepted marker forms incl. the `<screen-section — pending epic-{N}-{M} mapping>` placeholder
- `apps/web/src/eslint-rules/implements-pen-node-id.spec.ts` — RuleTester unit tests (19) + config-wiring assertions
- `_bmad-output/audit/drift-19-3-2026-05.md` — Rule 21 backfill audit + file→`.pen`-node mapping table + epic-19-8 worklist (Category C grep) — also the durable A/B/C classification record (the one-shot backfill script was removed after the headers landed; see Removed below)

**Removed (CR follow-up 2026-05-12):**
- `scripts/backfill-rule21-headers.mjs` — the idempotent one-shot backfill script; dead code once the headers were committed. Its A/B/C mapping is preserved in `_bmad-output/audit/drift-19-3-2026-05.md`.

**Modified:**
- `eslint.config.mjs` — import the local plugin + one new config object scoping `local/implements-pen-node-id` to `apps/web/src/components/**/*.{ts,tsx}`
- `project-context.md` — Rule 21 "Enforcement" Phase-2 bullet → present tense + names the rule; grammar block + `Last Updated` header updated (multi-component ` + ` form + `<screen-section …>` placeholder per Party Mode 2026-05-12)
- `_bmad-output/implementation-artifacts/19-3-eslint-pen-node-id-rule.md` — this story file (tasks [x], Dev Agent Record, File List, Change Log, Status)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 19-3 status `ready-for-dev` → `in-progress` → `review`
- `apps/web/src/components/ui/Button.tsx` — `// Implements: Component/ButtonPrimary (otvKh) + Component/ButtonSecondary (YDPhc)` header
- `apps/web/src/components/search/SearchBar.tsx` — `// Implements: Component/SearchInput (6MxLT)` header
- `apps/web/src/components/search/MediaTypeTabs.tsx` — `// Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4)` header
- `apps/web/src/components/shell/TabNavigation.tsx` — `// Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4)` header
- `apps/web/src/components/library/FilterChips.tsx` — `// Implements: Component/FilterChip (jD7gF)` header
- `apps/web/src/components/library/SortSelector.tsx` — `// Implements: Component/SortDropdown (955EZ)` header
- `apps/web/src/components/metadata-editor/GenreSelector.tsx` — `// Implements: Component/GenreTag (L1NP6)` header
- `apps/web/src/components/media/TechBadge.tsx` — `// Implements: Component/TechBadge-Video (L9m19) + Component/TechBadge-Audio (9iTW3) + Component/TechBadge-Subtitle (f84BM) + Component/TechBadge-HDR (cUjyv)` header
- *(+119 other files under `apps/web/src/components/` — leading `// Implements:` header line only, no logic changes: 25 carry `<utility — no .pen counterpart>` (Category B), 94 carry `<screen-section — pending epic-19-8 mapping>` (Category C). Full lists in `_bmad-output/audit/drift-19-3-2026-05.md`. The 4 pre-existing-headed files — `media/PosterCard.tsx`, `library/Empty{NoQBT,NoFolder,ReadyForScan}.tsx` — were NOT touched.)*

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-12 | CR (code-review follow-up, 6 LOW findings auto-fixed): (1) `implements-pen-node-id.js` — tightened the `{nodeId}` capture from `[A-Za-z0-9-]+` to `[A-Za-z0-9]+` so a hyphen is only legal inside `{Name}` (e.g. `TechBadge-Video`), matching AC #1's "letters/digits" wording; re-ran `eslint apps/web/src/components` → still 0 errors (no real node ID contains `-`). (2) Rule spec gained an invalid case for a hyphenated node id. (3) Rewrote the `eslint.config.mjs`-wiring `describe` block to dynamic-`import()` the resolved flat-config array and assert the rule's config object semantically (`files`/`ignores`/`rules`/`plugins`) instead of substring-matching the raw file text — rule spec now 22 tests (was 19). (4) Removed `scripts/backfill-rule21-headers.mjs` (dead one-shot script; A/B/C mapping preserved in `drift-19-3-2026-05.md`). (5)/(6) Synced the test-count figures across this story (rule spec 22, `nx test web` 148 files / 1840 tests) — earlier draft said "16"/"19" and "1834" inconsistently; the 94 Category-C `<screen-section …>` deferral and the JSDoc-`*`-stripping regex were reviewed and left as-is (by-design / no real-world gap). Regression after fixes: `pnpm nx test web` 148 files / 1840 tests PASS, `eslint apps/web/src/components` 0 errors. No contract change (rule grammar is *narrower*, not wider — already-valid headers all still pass). |
| 2026-05-12 | [@contract-v1→v2] AC #1 + AC #2 — what changed: added a 4th accepted marker form `// Implements: <screen-section — pending epic-{N}-{M} mapping>` (em-dash/hyphen tolerated) to the ESLint rule + Rule 21 grammar, for components that render a section of a designed *screen frame* (not a Reusable Component); also documented that `/* */`/JSDoc block comments are scanned. Origin: Sally + Amelia + Bob Party Mode 2026-05-12 — Sally flagged that the dev-story pass's `<utility — no .pen counterpart>` placeholder on ~94 screen-section components was inaccurate (the design exists, just as a screen-frame section). 94 Category-C files flipped from `<utility>` → `<screen-section …>`; rule message + project-context.md Rule 21 grammar list + `Last Updated` header + audit doc updated; rule spec gained the screen-section valid/invalid cases (19 tests). What breaks downstream: any future story adding a screen-section component under `apps/web/src/components/` may now use the `<screen-section …>` placeholder (and epic-19-8 will upgrade all 94 to canonical `// Design ref:` / `// Implements:` headers); downstream consumers must accept the 4-form grammar. No upstream stamp to ack (19-1 pre-Rule-20). |
| 2026-05-12 | DEV (dev-story): implemented `local/implements-pen-node-id` ESLint rule + wired into `eslint.config.mjs` + backfilled Rule 21 `// Implements:` headers across all 131 `apps/web/src/components/` candidate files (8 newly mapped to `.pen` Reusable Components, 119 exemption headers, 4 pre-existing left as-is) + RuleTester unit tests (16, all pass) + audit doc `drift-19-3-2026-05.md` + `project-context.md` Rule 21 Phase-2 bullet → present tense. Regression: `pnpm nx test web` 148 files/1834 tests PASS, `pnpm nx test api` PASS, `pnpm lint:all` 0 errors/122 warnings (matches bugfix-10-7 baseline), prettier clean, no orphaned test procs. Status `ready-for-dev` → `in-progress` → `review`. |
| 2026-05-12 | [@contract-v0→v1] AC #1, #2, #3 stamped on creation — what changed: defined the ESLint rule id (`local/implements-pen-node-id`), the accepted marker grammar (`Component/{Name} ({nodeId})` + ` + `-joined multi + two exemption forms), the "leading comment block" definition, and the scope/ignore set; what breaks downstream: 19-4 must consume the `drift-19-3-2026-05.md` mapping produced here, and any future story adding `components/` files inherits this lint error — header is now mandatory at CI time, not just SM-template time. Upstream 19-1 is pre-Rule-20 (no story file, done via Party Mode) → implicit v0, ack-skipped per Rule 20 forward-only retrofit. |
