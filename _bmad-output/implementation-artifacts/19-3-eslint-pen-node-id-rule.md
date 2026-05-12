# Story 19.3: ESLint Rule — Enforce Component-to-Design Node Traceability (Rule 21 Phase 2)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a frontend maintainer,
I want a custom ESLint rule that fails the build when a file under `apps/web/src/components/` lacks (or malforms) its `// Implements: Component/{Name} ({pen-node-id})` header,
so that design-implementation drift cannot silently re-enter the codebase the way `HoverPreviewCard.tsx` did (bugfix-10-4 root cause) — turning project-context.md Rule 21 from a manual SM-template gate into a CI-enforced invariant.

## Acceptance Criteria

1. [@contract-v1] A custom ESLint rule (`implements-pen-node-id` under a local plugin, e.g. `local/implements-pen-node-id`) exists and is registered in `eslint.config.mjs` for files matching `apps/web/src/components/**/*.{ts,tsx}`. The rule reports a lint **error** (not warning) when a component file's leading comment block does **not** contain a line matching one of:
   - `// Implements: Component/{Name} ({nodeId})` — `{Name}` is a non-empty identifier-ish token (allow letters, digits, `-`), `{nodeId}` is a non-empty token of letters/digits. Multiple components on one line joined by ` + ` are allowed (precedent: `PosterCard.tsx` → `// Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp)`).
   - `// Implements: <utility — no .pen counterpart>` (pure layout/utility component exemption)
   - `// Implements: <route-only>` (one-off route-level wrapper exemption)
2. [@contract-v1] The "leading comment block" is defined as: comments appearing before the first non-comment, non-whitespace token of the file (i.e. before the first `import`/`export`/statement). A malformed marker anywhere else in the file does **not** satisfy the rule. The rule message names the file and links Rule 21 (`project-context.md` Rule 21) and tells the author how to obtain the node ID (query `ux-design.pen` via Pencil MCP `get_editor_state` → "Reusable Components").
3. [@contract-v1] The rule does **not** apply to (and reports nothing for): `*.spec.tsx` / `*.spec.ts` / `*.test.tsx` files, files under `apps/web/src/hooks/`, `apps/web/src/services/`, `apps/web/src/stores/`, `apps/web/src/utils/`, and `apps/web/src/routes/` (route files are exempt per Rule 21 — only `components/` files that render designed UI are in scope). An `index.ts` barrel re-export file under `components/` is also exempt (no rendered UI). Scoping is done via ESLint flat-config `files`/`ignores` for the rule's config object, not inside the rule.
4. The rule auto-fix is **not** provided (node IDs cannot be invented — author must look them up). The rule MAY provide a suggestion message only.
5. All ~127 existing files under `apps/web/src/components/**/*.tsx` (excluding spec files) carry a valid Rule 21 header after this story — either a real `// Implements: Component/{Name} ({nodeId})` line backfilled from the `.pen` node-ID mapping, or a deliberate exemption annotation (`<utility — no .pen counterpart>` / `<route-only>`). The mapping is sourced by querying `ux-design.pen` via Pencil MCP `get_editor_state` (its "Reusable Components" listing gives `{Name} → {nodeId}`). Where a `components/` file has no `.pen` counterpart, annotate the exemption — do not leave it bare. Already-headed files (`PosterCard.tsx`, `EmptyNoQBT.tsx`, `EmptyNoFolder.tsx`, `EmptyReadyForScan.tsx`) are left as-is if already valid.
6. `pnpm lint:all` is **green** at story close (0 errors). The new rule must not introduce errors that weren't backfilled in AC #5. If any `components/` file genuinely cannot be mapped to a `.pen` node and isn't a utility/route wrapper, that is a **material drift finding** — record it in `_bmad-output/audit/drift-19-3-2026-05.md` and add it to the epic-19-8 sweep backlog (do NOT slap a fake exemption on it to make lint pass).
7. The rule has its own unit test (Vitest or ESLint `RuleTester`) co-located with the rule file, covering: (a) valid `Component/X (id)` header passes, (b) valid multi-component `A (id) + B (id)` header passes, (c) both exemption forms pass, (d) missing header → error, (e) malformed header (`// Implements: PosterCard` with no `Component/` prefix or no `(id)`) → error, (f) a valid marker that appears only *after* an `import` → still error (not in leading block), (g) a spec/hook/route file is not flagged (verify via the flat-config wiring or by testing the rule with the file path it would receive — whichever the chosen test harness supports).
8. `eslint.config.mjs` change is minimal and additive: register the local plugin + add one config object scoping the rule to `apps/web/src/components/**` with the documented `ignores`. No change to existing rule severities. The four-tool `pnpm lint:all` order (go vet → staticcheck → eslint → prettier) is unaffected; `eslint .` still covers `apps/web/`, `libs/shared-types/`, `tests/`.
9. `project-context.md` Rule 21 "Enforcement" block is updated: Phase 2 line changes from "(story 19-3) custom ESLint rule flags missing or malformed headers" (future tense) to a present-tense statement naming the rule id and where it lives. No other Rule 21 text changes.

## Tasks / Subtasks

- [ ] Task 1: Author the custom ESLint rule (AC: #1, #2, #3, #4)
  - [ ] Create the rule file (suggest `eslint-local-rules/implements-pen-node-id.js` at repo root, or `tools/eslint-rules/` — pick one, keep it ESM to match `eslint.config.mjs`)
  - [ ] Implement as a `meta.type = 'problem'` rule with no `fixable`; use `Program` node + `sourceCode.getCommentsBefore` / leading-comment inspection to find the leading comment block; regex-match the three accepted forms (anchor on the comment text, tolerate leading `*`/whitespace if a block comment is used, but a `// ` line comment is the documented norm)
  - [ ] Report on `Program` (loc: first line) when no accepted marker is found in the leading block; message cites Rule 21 + Pencil MCP `get_editor_state` lookup hint
  - [ ] Export a plugin object `{ rules: { 'implements-pen-node-id': rule } }` for flat-config registration
- [ ] Task 2: Wire the rule into `eslint.config.mjs` (AC: #1, #3, #8)
  - [ ] Import the local plugin; add ONE new config object: `{ files: ['apps/web/src/components/**/*.{ts,tsx}'], ignores: ['**/*.spec.{ts,tsx}', '**/*.test.{ts,tsx}', 'apps/web/src/components/**/index.ts'], plugins: { local: localPlugin }, rules: { 'local/implements-pen-node-id': 'error' } }` — note hooks/services/stores/utils/routes are already outside `components/**` so no explicit ignore needed for them; double-check that's true and only ignore what's actually inside `components/`
  - [ ] Place the new config object AFTER the TS config block and BEFORE the `prettier` config (which must stay last)
  - [ ] Run `pnpm exec eslint apps/web/src/components --max-warnings=9999` to see the full list of files the rule now flags (this is the AC #5 worklist)
- [ ] Task 3: Build the `.pen` node-ID mapping (AC: #5)
  - [ ] Query `ux-design.pen` via Pencil MCP `get_editor_state` → capture the "Reusable Components" list (`{Name} → {nodeId}`). The `.pen` file is encrypted — Pencil MCP tools ONLY, never `Read`/`Grep` on it (CLAUDE.md). Pencil.app must be running.
  - [ ] For each flagged file from Task 2, decide: real component node (use the mapping), `<utility — no .pen counterpart>`, or `<route-only>`. When unsure whether a `.pen` node exists, prefer searching the editor state over guessing.
  - [ ] Record the mapping you used in `_bmad-output/audit/drift-19-3-2026-05.md` (file → node-id-or-exemption) so 19-4 (Playwright baselines) and 19-8 (full sweep) can reuse it
- [ ] Task 4: Backfill headers across `apps/web/src/components/` (AC: #5, #6)
  - [ ] Add the `// Implements: Component/{Name} ({nodeId})` (or exemption) line as the FIRST line of each flagged file, above existing imports — match the in-repo precedent (`PosterCard.tsx`, `EmptyNoFolder.tsx`); a `// Source: ux-design.pen (Pencil app)` second line is optional, keep it only where it adds value
  - [ ] Do NOT touch `*.spec.tsx` files or anything outside `components/`
  - [ ] Any file that genuinely can't be mapped and isn't a utility/route wrapper → log as a material-drift finding in `_bmad-output/audit/drift-19-3-2026-05.md` + add to the epic-19-8 sweep backlog; do NOT add a fake exemption to silence lint (AC #6)
- [ ] Task 5: Unit-test the rule (AC: #7)
  - [ ] Co-locate the test with the rule file; use ESLint `RuleTester` (preferred — it exercises the actual AST path) or Vitest if `RuleTester` is awkward under this setup
  - [ ] Cover all 7 cases in AC #7 (valid single, valid multi, both exemptions, missing, malformed, marker-after-import, out-of-scope-file-not-flagged)
  - [ ] Ensure the test runs under `pnpm nx test web` (or document how it's run if it lives outside the web project)
- [ ] Task 6: Update `project-context.md` Rule 21 Enforcement block (AC: #9)
  - [ ] Change the Phase 2 bullet to present tense, naming the rule id (`local/implements-pen-node-id`) and file path; leave all other Rule 21 prose untouched
- [ ] Task 7: Full regression + close (AC: #6, #8)
  - [ ] `pnpm lint:all` → 0 errors (warnings count should match the bugfix-10-7 closeout baseline; the new rule must not add errors)
  - [ ] `pnpm nx test web` → all pass (new rule test included)
  - [ ] `pnpm exec prettier --check .` clean (or `--write` the touched files)
  - [ ] `pnpm run test:cleanup` → no orphans
  - [ ] No `.pen` file modification, no screenshot regeneration (this story only READS the `.pen` via MCP — CLAUDE.md screenshot workflow does NOT trigger)

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-12 | [@contract-v0→v1] AC #1, #2, #3 stamped on creation — what changed: defined the ESLint rule id (`local/implements-pen-node-id`), the accepted marker grammar (`Component/{Name} ({nodeId})` + ` + `-joined multi + two exemption forms), the "leading comment block" definition, and the scope/ignore set; what breaks downstream: 19-4 must consume the `drift-19-3-2026-05.md` mapping produced here, and any future story adding `components/` files inherits this lint error — header is now mandatory at CI time, not just SM-template time. Upstream 19-1 is pre-Rule-20 (no story file, done via Party Mode) → implicit v0, ack-skipped per Rule 20 forward-only retrofit. |
