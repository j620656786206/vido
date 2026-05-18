# Story 19.8: Comprehensive Design-Implementation Drift Sweep (Design-Drift Audit — Phase 2 Application)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- This story is the CAPSTONE of epic-19: applies the 19-3 ESLint rule + 19-4/19-4b visual harness to do the full component-vs-`.pen` sweep, files material-drift bugfix-N stories, and upgrades the 94 <screen-section — pending epic-19-8 mapping> placeholders to canonical Rule-21 forms. -->
<!-- @contract-v1 on AC #1–#5 (scope/methodology/audit-doc/bugfix-spawn/screen-section-upgrade contracts that future Rule 22 retros + bugfix-N tracking will rely on). -->
<!-- 🔗 AC Drift: N/A (capstone — applies established rules, no AC contradicts any prior story) · 📎 Contract Stamps: NEW (v1×5; consumes 19-3 [@contract-v2] Implements-marker grammar + 19-4 [@contract-v1] AC #1-#5 + 19-4b [@contract-v1] AC #5 platform-suffix — see Dev Notes acks) · 🔒 Rule 7: N/A (no Go errors; pure FE source headers + audit docs) · 🎨 UX: Sally is the primary classifier — material/minor/exact judgement is hers; this story IS the Sally-led design-drift sweep -->
<!-- markers-block-end -->

## Story

As the maintainers of Vido's design-implementation contract (Sally for design, Amelia for code, Bob for stories),
I want a one-pass comprehensive sweep that compares every `apps/web/src/components/**/*.tsx` file against its corresponding `ux-design.pen` node — classifying each as exact-match / minor-drift / material-drift per Rule 22's pixel-diff thresholds — and produces (a) a durable audit doc `_bmad-output/audit/drift-sweep-2026-05.md`, (b) one or more `bugfix-N` stories for the material-drift findings, and (c) upgraded Rule-21 headers on the 94 components currently carrying the `<screen-section — pending epic-19-8 mapping>` placeholder,
so that the epic-19 hypothesis ("the `HoverPreviewCard.tsx` ↔ `Component/PosterCardHover` drift that hid for months is systemic, not isolated" — Party Mode 2026-05-08) gets a definitive empirical answer, the design contract becomes fully traceable end-to-end (every component file → real `.pen` node OR documented exception), and the user gets a prioritised list of which drift bugs to actually fix.

## Acceptance Criteria

1. [@contract-v1] **Scope: every `apps/web/src/components/**/*.tsx` file in scope is examined.** The sweep covers exactly the same 131 files the 19-3 ESLint rule covers (12 Category-A real-pen-mapped + 25 Category-B `<utility — no .pen counterpart>` + 94 Category-C `<screen-section — pending epic-19-8 mapping>` per `_bmad-output/audit/drift-19-3-2026-05.md` — verify counts at sweep-start, account for any additions/removals since 19-4b closed). Each file gets a row in the AC #3 audit doc with: file path, current Rule-21 marker, sweep finding (exact / minor / material / N/A-utility), `.pen` node ID lookup result, classification rationale (one-line note from Sally), and follow-up disposition (no-op / minor-log-only / bundled-bugfix-N / standalone-bugfix-N / utility-confirmed / screen-section-upgraded-to-{node}). Tests / hooks / services / stores under `components/` (per Rule 21 exemptions) are excluded as before; index.ts barrels excluded as before.

2. [@contract-v1] **Classification methodology follows Rule 22 step 3 verbatim:**
   - **exact-match:** pixel diff < 0.5 % against the `.pen` node OR the current rendered baseline visually matches the design intent per Sally's expert eye (visual-baseline-19-4 is the rendered set; `pencil.app` MCP `get_screenshot` is the `.pen` side). Action: row in audit doc, no follow-up.
   - **minor drift:** 0.5–5 % diff (typography / spacing / colour micro-shifts). Action: row in audit doc with one-line rationale; bundled into a `bugfix-N-polish-ux-visual-pass-2` story (mirrors `bugfix-10-6-polish-ux-visual-pass` precedent) ONLY if ≥ 3 minor drifts share a theme — if < 3 or no theme, log only and revisit in next epic's Rule 22 retro.
   - **material drift:** > 5 % diff OR structural change (different layout, missing elements, wrong component composition). Action: row in audit doc + dedicated `bugfix-N-{slug}` story (one per material finding, no bundling — material drifts get individual triage). The originating story (`bugfix-10-4-hover-preview-viewport-flip`) is the model: a single component's drift, isolated story, full investigation.
   - **N/A — utility-confirmed:** Category-B files (`<utility — no .pen counterpart>`) — Sally validates each still has zero `.pen` counterpart. If a file moved Category-B → Category-A in the months since 19-3 (e.g. an originally-utility component grew enough UI surface to warrant a Reusable Component), upgrade its marker. Action: row in audit doc with disposition `utility-confirmed` or `re-classified-to-{category}`.
   - **screen-section-upgraded-to-{node}:** Category-C files (`<screen-section — pending epic-19-8 mapping>`) — Sally maps each to a `.pen` Screen Frame node ID via Pencil MCP `get_editor_state` (every Screen is listed there); Amelia rewrites the header per AC #5. Action: row in audit doc + the header upgrade (in-place edit).

3. [@contract-v1] **Audit doc location and shape: `_bmad-output/audit/drift-sweep-2026-05.md`.** Top-level structure (preserve heading levels on writes — the doc is a durable artifact future Rule 22 retros may grep):
   - `## Header` — sweep date, sweep agent identities (Sally + Amelia), total files examined, total drifts by classification, top-line conclusion (one sentence: "drift is systemic" / "drift is isolated to {N} components" / "drift is non-existent — design and implementation aligned").
   - `## Methodology` — Rule 22 thresholds verbatim, sample policy (this is a FULL sweep so the "≥5 sample" doesn't apply — explicit override), diff-calculation tooling (Playwright `toHaveScreenshot` for code-side rendering + Pencil MCP `get_screenshot` for design-side; pixel-diff via a sweep-time helper script if visual inspection alone is insufficient — Sally's call per component).
   - `## Sweep findings table` — single Markdown table, one row per file (131 rows), columns: `File`, `Current marker (post-sweep)`, `.pen Node`, `Classification`, `Rationale`, `Disposition`. Sortable by classification (group by `material` first, then `minor`, then `screen-section-upgraded`, then `utility-confirmed`, then `exact`).
   - `## Material drift findings — bugfix story index` — sub-table listing each material finding's slug + the spawned bugfix-N story file path + Sally's prioritisation rank (1 = fix first; N = fix last).
   - `## Minor drift findings — bundling decisions` — for each minor-drift theme cluster, list members + decide bundled-into-`bugfix-N-polish-ux-visual-pass-2` vs log-only.
   - `## Screen-section mapping resolution` — for each of the 94 Category-C files, the new pen-node mapping decision + the form of the upgrade (canonical `// Implements:` header OR `// Design ref:` soft comment OR re-classified to utility).
   - `## Audit-trail markers` — sweep-date + sweep-agent IDs + git SHA at sweep-time + the 19-4b queue rendering snapshot path (`tests/visual/components.visual.spec.ts-snapshots/`).

4. [@contract-v1] **Bugfix-N story spawn rules.** Each material drift becomes a dedicated story; minor-drift bundles (≥3 themed) become a single polish story. Naming and numbering:
   - Slug pattern: `bugfix-N-{kebab-component-name-or-theme}` where `N` is the next available post-Epic 10 bugfix number per existing sequence (current state as of 2026-05-18: latest `bugfix-19-4b-1` is mid-epic-numbered; for 19-8-spawned material-drift stories use post-Epic 10 numbers — first available is `bugfix-10-7` if the polish-pass convention holds, but **DEV verifies via `ls _bmad-output/implementation-artifacts/bugfix-10-*.md` + `ls bugfix-19-*.md` at sweep-time** and picks the next free integer in the appropriate series).
   - Each spawned bugfix story uses the existing `bugfix-10-4-hover-preview-viewport-flip.md` template structure (Status / Story / ACs / Tasks / Dev Notes / References / Change Log).
   - The bugfix story's Dev Notes MUST link back to this story (`19-8-comprehensive-component-sweep.md`) AND to the row in the audit doc that recorded its finding.
   - Sprint-status entries for each new bugfix story are added by THIS workflow (not deferred to a follow-up) — under `epic-19` if Sally judges the fix is part of the 19-8 closure; under separate post-Epic-19 buckets if Sally judges they're "stand-alone bugs surfaced by this sweep". Default: file under epic-19 unless Sally explicitly carves out.
   - The "no-bundling" rule for material drifts comes from the bugfix-10-4 precedent: each material drift gets full investigation context (root cause, regression test, design-side review) — bundling dilutes that. The "bundle minor drifts" rule comes from bugfix-10-6's success: small visual nits cluster well into one polish pass.

5. [@contract-v1] **Screen-section header upgrade.** Every Category-C file (`<screen-section — pending epic-19-8 mapping>`) MUST receive ONE of three upgraded markers per the Rule 21 L692 spec:
   - **Real `// Implements: Component/{Name} ({pen-node-id})`** — when Sally maps the section to a designed Reusable Component (the section was always reusable but wasn't carved out in `.pen` yet OR was missed during 19-3 mapping). Apply when the screen-section is clearly a designed widget that could be reused elsewhere.
   - **Soft `// Design ref: ux-design.pen Screen {ScreenName} ({nodeId})`** — when the section is part of a designed Screen Frame but is NOT a reusable widget (e.g. a one-off header layout inside a specific screen). The soft form is documented in Rule 21 L688 as the Phase-2 upgrade target. ESLint rule `local/implements-pen-node-id` MUST accept this form (verify during implementation — if the rule doesn't, file as a sub-finding requiring a quick rule-grammar bump under 19-3 [@contract-v3], handled within this story's scope or carved out per Sally's call).
   - **Re-classified to utility `// Implements: <utility — no .pen counterpart>`** — when Sally determines the file is genuinely utility-shaped despite the 19-3 backfill having placeholder-marked it as screen-section. This is a correction, not a primary outcome — expect few of these.
   
   The upgrade is a per-file edit. Total: 94 files. Mechanical when Sally's mapping decision is clear; cognitively-loaded when the mapping is ambiguous (e.g. a section that overlaps two screens, or a section the design predates). The decision goes in the audit doc's `## Screen-section mapping resolution` block (AC #3) before the code edit lands.

6. **Find-Sally-touchpoints.** The sweep is Sally-led on classification, Amelia-led on tooling + edits. Workflow:
   1. Amelia produces a worklist (the 131 files + their current markers + their committed baselines + the `.pen` node mapping from `_bmad-output/audit/drift-19-3-2026-05.md` + Pencil MCP `get_editor_state` output for screen frames).
   2. Sally walks the worklist in component-folder order (smallest concern: alphabetic; biggest concern: cluster by design-domain — `media/`, `library/`, `search/`, `downloads/`, etc.) and classifies each.
   3. For minor/material findings Sally provides a one-line rationale + Amelia files the bugfix story OR adds the row to the bundling table.
   4. For Category-C files Sally provides the screen-frame node ID OR the "no specific frame, soft ref to {ScreenName}" decision; Amelia executes the header upgrade.
   5. For Category-B files Sally confirms or re-classifies.
   6. Audit doc is filled in incrementally — Amelia commits per-section progress (e.g. one commit per components subdir) so a long-running sweep doesn't risk losing work.
   
   This pairing is the same pattern as the 19-4b Task 4 Sally-close-gate that succeeded: Sally reviews, Amelia codes, both sign off.

7. **No baseline regeneration unless the upgrade demands it.** This story does NOT touch `tests/visual/components.visual.spec.ts-snapshots/` — committed baselines remain the rendered source of truth for the sweep. If a screen-section header upgrade reveals the fixture needs a `routePath` / `seedQueries` change to render the section properly (19-4b's GalleryFixture extensions), the fixture change + baseline rebless is a NEW STORY (sub-finding of this sweep), NOT in-scope here. This boundary prevents 19-8 ballooning into a 19-4c bulk-fill round.

8. **No production code edits beyond the 94 Rule-21 header upgrades.** Modified files: `apps/web/src/components/**/*.tsx` (94 files maximum, header line ONLY); `_bmad-output/audit/drift-sweep-2026-05.md` (created); `_bmad-output/audit/drift-19-3-2026-05.md` (updated — the Category A/B/C tally is corrected if the sweep re-classifies any file); `_bmad-output/audit/visual-baseline-19-4.md` (the rendered-baseline status table gets the post-sweep classification column added per AC #2 if it isn't already there); `project-context.md` (Rule 21 + Rule 22 + Last Updated entries — see AC #10); `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-8 status + N new bugfix-N entries); N new `bugfix-N-*.md` files under `_bmad-output/implementation-artifacts/`. NO edits under `apps/api/`, `tests/visual/components.visual.spec.ts` (the spec is unchanged — it auto-discovers via DOM), `playwright.config.ts`, `package.json`. `ux-design.pen` is **READ ONLY** — Pencil MCP `get_editor_state` + `get_screenshot` are read operations; no `set_variables` / `batch_design` / `replace_all_matching_properties` calls in this story (those would modify `.pen` and trigger the CLAUDE.md screenshot-export workflow — out of scope here).

9. **Regression + framework hygiene.** `pnpm lint:all` 0 errors at close (warnings ≤ 122 baseline). The `local/implements-pen-node-id` ESLint rule (19-3) MUST stay green across the 94 header upgrades — if Sally's mapping decisions produce a marker form the rule doesn't accept, EITHER (a) reshape the decision into an accepted form, OR (b) file as a sub-finding requiring a Rule 21 grammar bump (the `// Design ref:` soft form was added to Rule 21 L688 in advance for exactly this case; verify the ESLint rule recognises it). `pnpm nx test web` + `pnpm nx test api` pass. `pnpm test:e2e --list` count unchanged. `pnpm run test:visual` green (no baseline change — AC #7). `pnpm run test:cleanup` no orphans. `ux-design.pen` untouched in modification-sense (read via MCP only).

10. **`project-context.md` updates** at sweep close:
    - **Rule 21:** the `<screen-section — pending epic-19-8 mapping>` placeholder description (L688–L692) gets updated past-tense — the sweep is done; the placeholder is no longer expected to appear in `components/`. Either: (i) keep the placeholder definition (for future first-pass backfills if a new file lands), but add a sentence "As of story 19-8 (2026-05-{day}), no `components/` file should still carry the pending placeholder — the sweep upgraded all 94 such files; see `_bmad-output/audit/drift-sweep-2026-05.md`." OR (ii) remove the placeholder entirely if Bob+Sally judge it's a once-and-done backfill aid not worth keeping in the rule's grammar. **Recommended: option (i)** — the placeholder remains a documented form for future re-backfill scenarios; the past-tense closure points readers at the sweep doc for context.
    - **Rule 22:** the tooling line gets extended to mention the audit-doc + the sweep is the precedent application (Rule 22 step 1's "sample-pick policy" notes that this story is the override case — full sweep, not ≥5 sample). Add a Last Updated entry summarising 19-8's outcome: total drifts found by classification.
    - **Last Updated header:** one entry, mirror the format of 19-5/19-6/19-7 entries.

## Tasks / Subtasks

- [ ] Task 1: Build the worklist (AC: #1, #6)
  - [ ] Amelia: regenerate the in-scope file list — `find apps/web/src/components -type f \( -name '*.tsx' -o -name '*.ts' \) -not -name '*.spec.*' -not -name 'index.ts' | sort` → should be 131 files; cross-check against `_bmad-output/audit/drift-19-3-2026-05.md` and reconcile any additions/removals since 19-3 closed.
  - [ ] For each file, capture: (a) current Rule-21 marker (`head -3 {file} | grep 'Implements:'`); (b) committed baseline path (`tests/visual/components.visual.spec.ts-snapshots/components/{gallery-id}/{state}-visual-darwin.png` if rendered; absent if utility-only); (c) `.pen` node mapping if Category-A (already in the drift-19-3 doc).
  - [ ] Amelia: open Pencil MCP via `get_editor_state` → enumerate all Screen Frames (their names + node IDs) → produce a CSV/table for Sally to map against during Task 3.
  - [ ] Commit the worklist as a draft section of `_bmad-output/audit/drift-sweep-2026-05.md` (the `## Sweep findings table` skeleton with 131 rows, classification column blank). This is the visible artifact Sally annotates during Task 3.

- [ ] Task 2: Tooling helper (AC: #2, #6)
  - [ ] If pixel-diff numbers are needed for borderline classifications (Sally requests on a per-file basis), use `npx playwright test --project=visual --update-snapshots --grep "{gallery-id}"` to regen a particular component's baseline, then `git diff` the PNGs via `git show :{path}` vs the working-tree to surface byte counts — note this is approximate (Playwright's `maxDiffPixelRatio 0.001` is the pass/fail gate; for the 0.5 % / 5 % bands, a manual visual comparison + Sally's judgement is the primary tool).
  - [ ] Pencil MCP `get_screenshot` for each Category-A node → save under `_bmad-output/audit/drift-sweep-2026-05-pencil-snapshots/` (gitignored; this is sweep-time scratch, not durable). Compare side-by-side with the corresponding `tests/visual/...-snapshots/.../default-visual-darwin.png`.
  - [ ] Optional: a tiny script `scripts/sweep-diff.sh {component-id}` that takes a gallery-id and prints the rendered baseline path + the .pen node + opens both in `open` for side-by-side review. Optional because manual eyeballing per component is sustainable for 131 files; the script is convenience.

- [ ] Task 3: Sally's classification pass (AC: #1, #2, #5, #6)
  - [ ] Sally walks the worklist by `components/` subdir (`media/`, `library/`, `search/`, `downloads/`, `shell/`, `setup/`, `health/`, `homepage/`, `dashboard/`, `metadata-editor/`, `parse/`, `degradation/`, `retry/`, `subtitle/`, `manual-search/`, `settings/`, `scanner/`, `ui/` — verify the actual subdir set during Task 1). For each file:
    - Eyeball baseline + `.pen` design side-by-side.
    - Classify exact / minor / material.
    - For Category-C: pick a Screen Frame node OR mark "soft ref to {ScreenName}" OR re-classify utility.
    - For Category-B: confirm utility OR re-classify.
    - For minor/material: provide a one-line rationale for the audit doc.
  - [ ] Amelia annotates the audit doc table as Sally calls each one. Commit per subdir (e.g. `docs(19-8): media/ subdir sweep — 12 files, 1 material, 3 minor`) to keep churn manageable.
  - [ ] After the full sweep walk, tally totals: exact / minor / material / utility-confirmed / re-classified / screen-section-upgraded. Write the top-line conclusion in the audit doc header.

- [ ] Task 4: Spawn bugfix-N stories (AC: #4)
  - [ ] For each material drift (count from Task 3): create a `bugfix-N-{slug}.md` story under `_bmad-output/implementation-artifacts/` using the `bugfix-10-4-hover-preview-viewport-flip.md` template. Cross-link to this story + the audit doc row. Add a sprint-status entry under `epic-19` (default; carve-out only if Sally explicitly judges stand-alone).
  - [ ] For minor-drift bundles (count from Task 3): if ≥ 3 share a theme, file ONE `bugfix-N-polish-ux-visual-pass-2.md` (the "-2" because `bugfix-10-6-polish-ux-visual-pass.md` is the precedent). List bundle members in its AC #1.
  - [ ] Bob's role: review the spawn pattern (one-per-material, bundle-minor) before any of the spawned stories transition past `backlog` — the spawn IS the create-story job for the children. Amelia executes; Bob signs off.
  - [ ] Confirm post-spawn: the audit doc's `## Material drift findings — bugfix story index` sub-table has a row per spawned story.

- [ ] Task 5: Header upgrades (AC: #5, #8, #9)
  - [ ] For each Category-C file, apply the header upgrade per Sally's Task 3 decision. The change is ONE LINE per file (the `// Implements:` comment). 94 mechanical edits — batch by subdir, commit per subdir.
  - [ ] For each re-classified file (Category-C → utility, or vice versa), apply the corresponding marker. Update `_bmad-output/audit/drift-19-3-2026-05.md` Category tables to reflect the corrections.
  - [ ] Run `pnpm exec eslint apps/web/src/components/ --no-fix` to verify the `local/implements-pen-node-id` rule stays green across all 94 upgrades. Iterate if any upgrade trips the rule — Sally re-picks a form, Amelia re-applies.
  - [ ] Per-subdir commit message: `docs(19-8): {subdir}/ Rule-21 header upgrade — {N} files (X→component, Y→screen-frame, Z→utility)`.

- [ ] Task 6: Documentation closure (AC: #3, #10)
  - [ ] `_bmad-output/audit/drift-sweep-2026-05.md` — full doc populated (header / methodology / findings table / material-bugfix-index / minor-bundling decisions / screen-section mapping / audit-trail markers).
  - [ ] `_bmad-output/audit/visual-baseline-19-4.md` — add the post-sweep classification column to its rendered-baseline table OR append a new section `## 19-8 sweep classification (2026-05-{day})` that maps each row to its drift-sweep classification. Single source of truth for "what state is each baseline in post-sweep".
  - [ ] `project-context.md` — Rule 21 placeholder description gets the past-tense closure sentence (AC #10 option (i)); Rule 22 tooling line gets the audit-doc precedent mention; Last Updated header gets a 19-8 entry. Mirror format of 19-4 / 19-4b / 19-5 / 19-6 / 19-7 entries.

- [ ] Task 7: Close-out regression (AC: #8, #9)
  - [ ] `pnpm lint:all` → 0 errors, warnings ≤ 122 (the header-upgrade edits should NOT change warnings; if they do, investigate).
  - [ ] `pnpm exec eslint apps/web/src/components/ --no-fix` → 0 errors (the 19-3 rule's exact scope).
  - [ ] `pnpm nx test web` + `pnpm nx test api` pass (header-only edits don't break tests — confirm).
  - [ ] `pnpm test:e2e --list` → 1663 unchanged.
  - [ ] `pnpm run test:visual` green (no baseline change per AC #7).
  - [ ] `pnpm run test:cleanup` no orphans.
  - [ ] `ux-design.pen` untouched (read-only via MCP).
  - [ ] Sprint-status entry: `ready-for-dev` → `in-progress` → `review` (the in-progress phase will span several commits — sub-section completions per Task 3 / Task 5).

## Dev Notes

### Why this story exists / where it sits in epic-19

- **This is the capstone of epic-19.** Stories 19-1 + 19-2 added the Rules (21 + 22). Stories 19-3 + 19-4 + 19-4b built the tooling (ESLint enforcement + visual harness + 262 baselines). Story 19-5 + 19-6 + 19-7 wired CI (PR-blocking visual diff + monthly TestSprite cron + month-end quota alert). **19-8 APPLIES the rules + tooling to the actual product** — the empirical answer to "how widespread is design-implementation drift in Vido?" The originating Party Mode 2026-05-08 ruling hypothesised systemic drift; 19-8 produces the data.
- **Three deliverables, one story:**
  1. **Drift sweep audit doc** (`_bmad-output/audit/drift-sweep-2026-05.md`) — the durable record of the state of the system on sweep-day.
  2. **Spawned bugfix-N stories** — actionable triage queue for material drifts.
  3. **94 Rule-21 header upgrades** — completes the design-traceability chain at the screen-section level (the Phase-2 follow-through to 19-3's Phase-1 backfill).
- **Why NOT split into sub-stories** (à la 19-4 → 19-4b): the three deliverables are tightly coupled — Task 3's classification produces both the audit row AND the screen-section mapping decision AND (for material findings) the bugfix-story-spawn input. Splitting would mean walking the 131-file list three times. Single story keeps the walk to one pass. **Caveat:** if Sally + Amelia at implementation find the volume unwieldy, the established precedent (19-4 Party Mode 2026-05-12 re-cut) allows in-flight scope re-negotiation — file as a 19-8b for whichever deliverable is least ready when the other two land cleanly. Document the call in Completion Notes.
- **Sample-policy override:** Rule 22 step 1 says "≥5 components per epic retro". 19-8 is the FULL sweep, not a Rule-22-retro instance — the policy doesn't apply. Note this explicitly in the audit doc Methodology section so future Rule 22 retros don't mistakenly cite 19-8 as the "≥5 sample" precedent.
- **Why now and not later:** the harness (19-4 + 19-4b) is fully built — 262 committed `-darwin` baselines, 123 fixtures, Sally already approved them. The ESLint rule (19-3) is enforced. The CI gate (19-5) is ready-for-dev. The sweep can finally happen without "but the tooling isn't ready" excuses. Delaying past Epic 19 means the screen-section placeholder lingers in `components/` indefinitely, eroding the rule's "every file is mapped" guarantee.

### Architecture / constraints — read before implementing

- **Pure FE source headers + audit docs.** 0 Go, 0 logic changes, 0 tests authored, 0 baseline regeneration, 0 fixture changes, 0 `.pen` modifications. The 94 source edits are header-line ONLY (one comment per file). Cross-stack split check: backend tasks = 0, frontend logic tasks = 0 (header-only edits are not logic). Single story is correct.
- **Volume is real but mechanical** — 131 files to walk, 94 to edit, N material findings to spawn into stories. The walk itself is the cognitively heavy part (Sally's classification judgements); the edits are repetitive. Per-subdir commits keep diffs manageable for review.
- **Pencil MCP read-only contract** — `get_editor_state` (lists Screens + Reusable Components), `get_screenshot` (renders a node to PNG for visual comparison), `batch_get` (reads multiple nodes), `snapshot_layout` (layout snapshot). **DO NOT call** `set_variables`, `batch_design`, `replace_all_matching_properties`, `export_nodes` — those modify `.pen`, which triggers the CLAUDE.md screenshot-export workflow (a separate concern). If a sweep finding reveals a design needs a change (`.pen` needs a new Reusable Component carved out, or a node needs renaming), that's a NEW DESIGN STORY (out of scope here).
- **Per-subdir commit cadence** — `media/` → `library/` → `search/` → ... per the `components/` subdir set. Each commit body lists: files touched, drift counts by classification, screen-section upgrade counts. This gives Bob a per-commit audit trail (the SM can spot-check Sally's classifications without reviewing 131-file mega-diffs).
- **The screen-section mapping problem** — 94 files currently say `<screen-section — pending epic-19-8 mapping>`. Sally's job is to pick, for each, ONE of:
  - (a) A specific Screen Frame node ID → `// Design ref: ux-design.pen Screen {ScreenName} ({nodeId})` soft comment (Rule 21 L688).
  - (b) A Reusable Component node ID (if the section is actually a designed reusable) → `// Implements: Component/{Name} ({pen-node-id})` canonical form.
  - (c) Re-classify to utility → `// Implements: <utility — no .pen counterpart>`.
  
   Sally's judgement is the load-bearing decision. The mapping table goes in the audit doc's `## Screen-section mapping resolution` section BEFORE the code edit (audit-doc-first, code-edit-second — this preserves the decision rationale even if the edit needs to be revised later).
- **ESLint rule (19-3) accepts the soft `// Design ref:` form?** Verify during Task 5. The rule's flat-config (`apps/web/src/eslint-rules/implements-pen-node-id.js`) was updated in 19-3 to accept the four canonical forms: `Component/X (id)`, multi-component `+`-joined, `<utility …>`, `<screen-section — pending epic-19-8 mapping>`. The Phase-2 `// Design ref:` form is described in Rule 21 L688 as an upgrade target but **the ESLint rule's regex may not yet recognise it**. Sub-finding handling: if the rule errors on the soft form, file an inline patch to the rule's regex (or extend the accepted-marker list) as a sub-finding of 19-8 — handled in this story's scope if it's a one-line regex edit; carved out to 19-3-followup if the change is larger. The rule's source file is `apps/web/src/eslint-rules/implements-pen-node-id.js`; the wiring is in `eslint.config.mjs`.
- **Bugfix-N numbering** — verify the next free number at implementation time. State as of 2026-05-18:
  - `bugfix-10-1` through `bugfix-10-7` (the polish pass) — most under epic-10.
  - `bugfix-19-4b-1-gallery-max-update-depth-warnings.md` — already under epic-19 series.
  - Default for 19-8 spawns: use `bugfix-19-8-{slug}` numbering for material findings under epic-19; OR if a finding clearly belongs to a different epic (a `media/` component drift is more naturally "epic-10 polish"), use `bugfix-10-N` numbering. **Sally + Bob decide naming at spawn-time** per finding.
- **Audit doc filename: `drift-sweep-2026-05.md`** — the date is the sweep MONTH. If the sweep slips to June, rename to `drift-sweep-2026-06.md`. The audit doc is a one-shot artifact (vs. the periodic `drift-{epic}-{YYYY-MM}.md` files Rule 22 retros produce). Future Rule 22 retros may compare against this baseline ("since 19-8, drift rate is N material per epic" — trend analysis).
- **Sally's prioritisation rank** (AC #3 `## Material drift findings — bugfix story index` sub-table) — material drifts are 1..N ordered by Sally's judgement of "fix this first because it's most user-visible / most-likely-to-cause-real-bug / breaks the most other things". This is INPUT to the user's sprint planning, not enforced order — the user can ignore the rank if priority shifts.
- **No new dependencies, no version bumps.** All tooling (`pencil` MCP, `playwright`, `eslint`) is already pinned. No `package.json` change.

### Project Structure Notes

- **New files:** `_bmad-output/audit/drift-sweep-2026-05.md` (the primary deliverable); N × `_bmad-output/implementation-artifacts/bugfix-{N}-*.md` (one per material drift + at most one polish-bundle); this story file.
- **Modified files:** `apps/web/src/components/**/*.tsx` (94 files max, header LINE ONLY); `_bmad-output/audit/drift-19-3-2026-05.md` (Category tally corrections if any re-classifications happen); `_bmad-output/audit/visual-baseline-19-4.md` (post-sweep classification column or appended section); `project-context.md` (Rule 21 placeholder closure + Rule 22 tooling + Last Updated); `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-8 status + N new bugfix entries).
- **Read-only (via MCP):** `ux-design.pen` (Pencil MCP `get_editor_state` + `get_screenshot`).
- **Read-only (via filesystem):** `tests/visual/components.visual.spec.ts-snapshots/**` (the 262 committed baselines — sweep input, not sweep output).
- **Sweep-time scratch (gitignored):** `_bmad-output/audit/drift-sweep-2026-05-pencil-snapshots/*.png` if Task 2 uses Pencil-side screenshots for side-by-side comparison.
- **Out of scope:** any baseline regeneration (would be a 19-4c); any `.pen` modification (would trigger CLAUDE.md export workflow); any new fixture or `apps/web/src/routes/test/-gallery.fixtures.tsx` change (would be a 19-4c); any change to `playwright.config.ts` or `tests/visual/components.visual.spec.ts` (the spec auto-discovers); any TestSprite-side test-case work (19-6/19-7's domain); any Epic 20+ planning (this is the epic-19 closeout, not the next-epic kickoff).

### Testing standards (project-context.md)

- **No new test code in this story.** The deliverable is documentation + header-line edits + spawned bugfix stories.
- **Header-edit validation:** the `local/implements-pen-node-id` ESLint rule is the test for the 94 edits (per-subdir `eslint apps/web/src/components/{subdir}/` runs as part of Task 5).
- **Rule 12 lint gate:** `pnpm lint:all` 0 errors / ≤122 warnings at close. Header-only edits don't change warnings.
- **Rule 16 assertion quality:** N/A (no test assertions).
- **Rule 13 error handling:** N/A (no logic).
- **`pnpm run test:cleanup`:** post-Task-7 verification per the standard discipline.

### Rule 21 / Rule 22 / Rule 20 linkage

- **Rule 21:** this story is the Phase-2 application — every `components/**/*.tsx` file gets a CANONICAL marker (no more pending-placeholders). The Phase-1 ESLint enforcement (19-3) continues to gate any new file; 19-8 cleans up the Phase-1 backfill's 94 placeholder rows.
- **Rule 22:** this story is the FULL-SWEEP precedent — Rule 22's normal mode is "≥5 sample per epic retro"; 19-8 is the explicit override (full sweep, capstone application). The audit doc explicitly notes the override so future retros don't mis-cite. The tooling line gets a post-sweep update.
- **Rule 20 (AC Contract Versioning):** stamps `[@contract-v1]` on AC #1–#5. Downstream consumers: future Rule 22 retros + the spawned bugfix-N stories' own ACs may cross-reference 19-8's classifications. **Upstream consumed:**
  - confirmed against 19-3 `[@contract-v2]` (Implements: marker grammar — the 4-form acceptance list this story extends/respects);
  - confirmed against 19-4 `[@contract-v1]` AC #1 (the `visual` Playwright project exists), AC #5 (baseline path convention — the sweep reads from this path);
  - confirmed against 19-4b `[@contract-v1]` AC #5 (platform-suffix decision — sweep reads `-darwin` baselines; `-linux` set is 19-5's CI bootstrap and not consumed here).
  No upstream bump — this story consumes the rules + tooling without modifying their contracts.
- **Rule 7 (Error Codes):** N/A — no Go errors.

### Latest tech information

- **Pencil MCP** — already in the stack; tools `get_editor_state`, `get_screenshot`, `batch_get`, `snapshot_layout` are read-safe. Pencil MCP's HTTP mode (port 9876 per CLAUDE.md screenshot-export script) is the same one this story reads. NO `set_variables` / `batch_design` / `replace_all_matching_properties` / `export_nodes` calls — those are writes.
- **Playwright `--update-snapshots`** — used in 19-4b for baseline regeneration; ONLY invoked here if Task 2's borderline-classification helper needs a re-render of a single component (rare; manual eye + Pencil screenshot is the primary method).
- **`eslint apps/web/src/components/`** — narrow-scope lint, faster than `pnpm lint:all`, used iteratively during Task 5 to verify each per-subdir commit's headers.

### References

- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:529] — 19-8 charter: UX Sally + DEV Amelia, AFTER 19-3 ESLint lands, full sweep, findings tracked in `_bmad-output/audit/drift-sweep-2026-05.md`, material drift → bugfix-N story, minor drifts bundled per bugfix-10-6 precedent.
- [Source: _bmad-output/audit/drift-19-3-2026-05.md] — Category A/B/C tables this story extends/corrects.
- [Source: _bmad-output/audit/visual-baseline-19-4.md] — rendered-baseline status table this story extends with post-sweep classification.
- [Source: _bmad-output/audit/drift-bugfix-10-4-2026-05.md] — the originating bugfix finding (the Party Mode 2026-05-08 hypothesis source).
- [Source: tests/visual/components.visual.spec.ts-snapshots/**] — the 262 committed `-darwin` baselines (Sally + Amelia 19-4b Task 4 verification source).
- [Source: project-context.md#Rule-21-Component-to-Design-Node-Traceability] — the four-form marker grammar this story applies; the L688 `// Design ref:` soft form this story upgrades placeholders to.
- [Source: project-context.md#Rule-22-Epic-Retro-Design-Drift-Audit] — the classification thresholds (0.5 % / 5 %), the sample-pick policy this story overrides.
- [Source: project-context.md#Rule-20-AC-Contract-Versioning] — the stamp + ack format.
- [Source: _bmad-output/implementation-artifacts/19-3-eslint-pen-node-id-rule.md] — the ESLint rule this story's header upgrades must keep green (the rule's regex / accepted-marker list is in `apps/web/src/eslint-rules/implements-pen-node-id.js`).
- [Source: _bmad-output/implementation-artifacts/19-4-playwright-visual-snapshot-baseline.md] — the harness this story uses; baseline path conventions.
- [Source: _bmad-output/implementation-artifacts/19-4b-visual-baseline-bulk-fill.md] — the bulk fill that brought the harness to 122/123/262 coverage; the Sally-led close-gate pattern Task 3 mirrors.
- [Source: _bmad-output/implementation-artifacts/bugfix-10-4-hover-preview-viewport-flip.md] — the template for spawned material-drift bugfix-N stories.
- [Source: _bmad-output/implementation-artifacts/bugfix-10-6-polish-ux-visual-pass.md] — the precedent for minor-drift bundling.
- [Source: CLAUDE.md "UX Design Screenshots Workflow"] — read-only Pencil MCP usage is safe; modification triggers the export-screenshots workflow (out of scope here).
- [Source: apps/web/src/eslint-rules/implements-pen-node-id.js] — the rule whose grammar this story's header upgrades respect (and possibly extend, per Dev Notes sub-finding).

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-18 | SM Bob /create-story (YOLO) — story drafted ready-for-dev. Pure documentation + Rule-21 header-line edits + bugfix-story spawning; 0 Go / 0 frontend logic / 0 tests authored → single story (cross-stack split N/A; backend tasks = 0, frontend logic tasks = 0; header-only edits are not logic). 10 ACs (#1–#5 stamped `[@contract-v1]`), 7 tasks. Sally + Amelia dual-agent ownership: Sally classifies / decides screen-section mappings; Amelia tooling / edits / files spawned bugfix stories; Bob signs off the spawn pattern. Key SM decisions in Dev Notes: (a) **Single story, not 3 sub-stories** — three deliverables (audit doc + spawned bugfix-N + 94 header upgrades) are tightly coupled to Task 3's per-file classification walk; splitting would mean walking 131 files three times; 19-4→19-4b re-cut precedent allows in-flight split if scope balloons (Completion Notes follow-up). (b) **Sample-policy override** — Rule 22's "≥5 sample per epic retro" is NOT applied; this is the full-sweep capstone; audit doc Methodology section MUST note the override explicitly so future retros don't mis-cite. (c) **Material vs minor classification action policy** — material gets dedicated `bugfix-N-{slug}` story (no bundling — preserves bugfix-10-4 investigation depth); minor bundles into one `bugfix-N-polish-ux-visual-pass-2.md` ONLY if ≥3 share theme, else log-only. (d) **Three Rule-21 marker forms for the 94 screen-section files** — real `// Implements: Component/{Name} ({nodeId})` (designed reusable), soft `// Design ref: ux-design.pen Screen {ScreenName} ({nodeId})` (one-off section in a screen frame), or re-classified `<utility …>`; Sally picks; ESLint rule (19-3) must stay green — if soft form trips the rule, sub-finding inline-patches the rule's regex. (e) **Pencil MCP read-only** — `get_editor_state` / `get_screenshot` / `batch_get` allowed; `set_variables` / `batch_design` / `replace_all_matching_properties` / `export_nodes` FORBIDDEN (would trigger CLAUDE.md export workflow). (f) **Per-subdir commit cadence** — 18 subdirs walked separately, each its own commit; Bob reviews per-commit not per-mega-diff. (g) **Bugfix-N numbering** — Sally+Bob decide naming per finding (bugfix-19-8-{slug} for clear epic-19 belonging; bugfix-10-N for a `media/` drift naturally under epic-10). (h) **No baseline regeneration, no fixture changes, no `.pen` modification** — if any of those are needed, file as 19-4c (boundary preserves story scope). Consumes upstream contracts per Rule 20 forward-only retrofit: confirmed against 19-3 [@contract-v2] (Implements marker grammar), 19-4 [@contract-v1] AC #1 + AC #5 (visual project + baseline path), 19-4b [@contract-v1] AC #5 (platform-suffix). No upstream bump (consumes, doesn't modify). 🔒 Rule 7: N/A (no Go). 🎨 UX: Sally IS the primary classifier — material/minor/exact judgement is hers; the story is the Sally-led sweep. Depends on 19-3 (done) + 19-4 (done) + 19-4b (done); pairs nicely with 19-5/19-6/19-7 (ready-for-dev) but doesn't block them. Closes epic-19 when done. |
| 2026-05-18 | [@contract-v0→v1] AC #1–#5 stamped on creation — what's defined: scope (every `apps/web/src/components/**/*.tsx` file in 19-3's 131-file in-scope set; same exemptions: tests/hooks/services/stores/index.ts excluded) (AC #1); classification methodology (Rule 22 step-3 thresholds verbatim: <0.5%=exact, 0.5-5%=minor, >5%-or-structural=material; per-classification disposition; sample-policy explicit override since this is full-sweep) (AC #2); audit doc shape (`_bmad-output/audit/drift-sweep-2026-05.md` with Header/Methodology/findings-table/material-bugfix-index/minor-bundling/screen-section-mapping/audit-trail sections — table column set + row ordering policy) (AC #3); bugfix-N spawn rules (one-per-material, ≥3-themed-minor-bundle into one polish-pass-2 story, bugfix-10-4 template, cross-link to this story + audit row, sprint-status entry under epic-19 default) (AC #4); screen-section header upgrade (94 files; three forms — real Implements/Component, soft Design-ref/Screen, re-classified utility; ESLint rule stays green; mapping goes in audit doc before edit) (AC #5). What breaks downstream: future Rule 22 retros depend on AC #2's classification taxonomy + AC #3's audit-doc shape for trend analysis (silently changing taxonomy breaks "drift rate per epic" comparison); spawned bugfix-N stories depend on AC #4's spawn convention for their own context (template + cross-link); the Phase-2 `// Design ref:` form's ESLint acceptance is a downstream consumer of the 19-3 rule's grammar (if 19-3 bumps to [@contract-v3] to formally accept the soft form, that's a separate ack chain). Upstream consumed: confirmed against [@contract-v2] (Story 19-3 — Implements: marker grammar, four accepted forms, the soft Design-ref form's Phase-2 addition), confirmed against [@contract-v1] (Story 19-4 AC #1 — the `visual` Playwright project that produces the baselines this story reads), confirmed against [@contract-v1] (Story 19-4 AC #5 — baseline path convention `tests/visual/components.visual.spec.ts-snapshots/components/{id}/{state}-visual-{platform}.png`), confirmed against [@contract-v1] (Story 19-4b AC #5 — platform-suffix `-darwin`/`-linux` decision; this story consumes `-darwin` only). No upstream contract bumps — this story is a pure capstone application. |
