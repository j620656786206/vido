# Story: AC Contract Drift Check in dev-story Workflow

Status: done

## Story

As a Scrum Master (Bob) and Dev Agent (Amelia) working stories that touch **existing** acceptance criteria across previously-completed story files,
I want `dev-story/instructions.xml` to enforce a mandatory "AC Contract Drift Check" whenever a new story modifies behavior specified in a prior story's AC,
so that cross-story AC contract drift is caught at implementation time (not escaping until adversarial CR, as happened with Story 10-4 → 10-5).

## Acceptance Criteria

1. Given a developer runs `/bmad:bmm:workflows:dev-story` on a story whose implementation changes the observable behavior of a prior story's acceptance criterion, when the dev-story workflow loads the story (Step 2), then it executes a mandatory "AC Contract Drift Check" action that requires the dev agent to grep `_bmad-output/implementation-artifacts/*.md` for cross-references to the modified AC text or contract
2. Given the drift check finds ≥1 hit in a previous story file that references the AC-in-question, when the dev agent proceeds with implementation, then the dev agent MUST document the drift in Dev Agent Record → Completion Notes using a prefix like `🔗 AC Drift: {prior-story} AC #N — {old contract} → {new contract}`, AND update the current story's File List to include the drifted prior story path as a reference marker
3. Given the drift check finds no hits, when the dev agent proceeds, then the dev agent MUST still record `🔗 AC Drift: NONE (checked: {grep-pattern-used})` in Completion Notes so the check is auditable and cannot be silently skipped
4. Given the dev-story workflow modification lands, when `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` is inspected, then the new action is tagged `critical="MANDATORY"`, cites Epic 10 Retro AI-2 + Pattern #2 (10-4 → 10-5 drift) as the precedent, and includes a concrete example of what a drift finding looks like (so future dev agents don't treat the check as a checkbox ceremony)
5. Given this is a pure workflow-docs change, when `pnpm lint:all` + `pnpm nx test api` + `pnpm nx test web` run post-change, then ALL pass with zero regressions (this story modifies no Go or TS code)
6. Given the sprint-status.yaml entry `retro-10-AI2-ac-contract-drift-check`, when this story completes the dev-story → code-review cycle, then the status transitions `backlog → ready-for-dev → in-progress → review → done` as per workflow, and a note captures the final placement (step number + line range) of the new check

## Tasks / Subtasks

- [x] Task 1: Add "AC Contract Drift Check" action to `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` (AC: #1, #2, #3, #4)
  - [x] 1.1 Decide placement: insert into Step 2 "Load project context and story information" AFTER the existing `<action>Load comprehensive context from story file's Dev Notes section</action>` line — this places the check AFTER story context is loaded but BEFORE any implementation decisions in Step 5 (optimal: dev agent has full AC text in hand, hasn't yet written code)
  - [x] 1.2 Draft the new action XML block (~25–35 lines) with the following shape:
    ```xml
    <action critical="MANDATORY">AC CONTRACT DRIFT CHECK (Epic 10 Retro AI-2):
      If this story's implementation modifies the observable behavior of an acceptance
      criterion already shipped in a prior story, you MUST grep across previous story
      files for cross-references BEFORE writing code.

      Trigger conditions (run the check if ANY is true):
        (a) A task description says "changes behavior of {feature X}" where X was
            delivered in a previous story.
        (b) An AC in THIS story rephrases or contradicts an AC from a PRIOR story
            (e.g., "single POST" → "≤N POSTs").
        (c) A task touches a file already listed in a previous story's File List
            in a way that alters its external contract (signatures, wire format,
            timing guarantees, batch semantics).

      Check procedure:
        1. Identify the candidate prior story file(s) in _bmad-output/implementation-artifacts/.
        2. grep -rn "{AC-relevant keyword(s)}" _bmad-output/implementation-artifacts/*.md
           Use keywords from the AC wording: e.g., "single POST", "batch", "check-owned",
           "aria-modal", "focus trap", etc. Start broad, narrow if too many hits.
        3. For each hit, read the surrounding AC and decide: is my change a DRIFT
           (changes behavior the prior AC specified) or a REUSE (prior AC still holds)?

      Documentation rule:
        - If drift found: add to Dev Agent Record → Completion Notes:
            "🔗 AC Drift: {prior-story-key} AC #N — {old contract} → {new contract}"
          AND append the prior story file path to this story's File List with the
          annotation "(AC drift reference — see Completion Notes)".
        - If NO drift found: still record in Completion Notes:
            "🔗 AC Drift: NONE (checked: '{grep-pattern-used}' across
            _bmad-output/implementation-artifacts/*.md — N hits, all REUSE not DRIFT)"
          The check is MANDATORY — silence is not an option.

      Precedent (Epic 10, Story 10-4 → 10-5):
        Story 10-4 AC #4 stated "single POST batch for all visible TMDb IDs".
        Story 10-5 added IntersectionObserver lazy-load, which silently changed the
        contract to "≤N POSTs (one per lazy-revealed batch)". The drift escaped
        internal review and was only caught by adversarial CR (10-5 CR H1).
        This check exists to catch that class of drift BEFORE CR.
    </action>
    ```
  - [x] 1.3 Place the new action BETWEEN the existing Step 2 actions — specifically after `<action>Use enhanced story context to inform implementation decisions and approaches</action>` (currently the last action before the `<output>` tag in Step 2). The new action must execute before Step 5 (implementation) so the dev agent has already done the drift analysis by the time they write code.
  - [x] 1.4 Update the Step 2 `<output>` block to mention drift-check status:
    ```
    ✅ **Context Loaded**
      Story and project context available for implementation
      AC Drift Check: {{drift_check_result}}  <!-- new line -->
    ```
    Where `{{drift_check_result}}` is one of: "NONE", "FOUND — see Completion Notes", "N/A (new feature, no prior AC to drift from)".
  - [x] 1.5 Add a brief cross-reference comment at the top of the new action citing both Pattern #2 from the Epic 10 retro and Agreement 4 (retro action items become tracked entries) so the provenance is clear when a future SM reads the file.

- [x] Task 2: Verify no regression in existing dev-story behavior (AC: #5)
  - [x] 2.1 Run `pnpm lint:all` from repo root — PASS expected (docs-only change; `.xml` files are not linted by ESLint/Prettier/go vet/staticcheck)
  - [x] 2.2 Run `pnpm nx test api` — PASS expected (zero code change)
  - [x] 2.3 Run `pnpm nx test web` — PASS expected (zero code change)
  - [x] 2.4 Read the modified `instructions.xml` end-to-end to confirm:
    - Steps 1–11 numbering is preserved (no accidental renumbering)
    - All existing `<critical>` / `<action>` / `<check>` tags remain intact
    - The new action is nested INSIDE Step 2 (not accidentally closing Step 2 early)
    - XML is well-formed (no dangling tags, no `&` or `<` in content without escaping)

- [x] Task 3: Documentation discoverability — light touch (AC: #4)
  - [x] 3.1 Do NOT add Rule 20 to project-context.md. Rationale: project-context.md Rules govern code conventions (logging, layering, naming); workflow process rules belong in the workflow file itself. This is deliberate consistency with retro-9-AI1 (FULL REGRESSION GATE was added to dev-story/instructions.xml, NOT to project-context.md).
  - [x] 3.2 Verify the Epic 10 retro document `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md` already cites AI-2 with enough detail that a future reader can find this story. No edit needed to the retro — it's frozen.

- [x] Task 4: Update sprint-status.yaml (AC: #6)
  - [x] 4.1 Mark `retro-10-AI2-ac-contract-drift-check: ready-for-dev` at story creation time (this file's generation step, handled by `/create-story` workflow).
  - [x] 4.2 On `/dev-story` invocation, transition `ready-for-dev → in-progress`.
  - [x] 4.3 On `/dev-story` completion, transition `in-progress → review`.
  - [x] 4.4 On `/code-review` pass, transition `review → done` with a completion note in the sprint-status.yaml comment recording the final placement line range inside `instructions.xml` (e.g., "added Step 2 action at lines 147-180").

## Dev Notes

### Root Cause

Epic 10 retro (2026-04-20) Pattern #2: Story 10-4 shipped AC #4 stating "single POST batch for all visible TMDb IDs". Story 10-5 later added IntersectionObserver lazy-load, which silently changed the contract to "≤N POSTs (one per lazy-revealed batch)". This drift was only caught by adversarial code review on 10-5 (H1 finding: mislabeled comment + missing stability gate). Without adversarial CR, the drift would have shipped.

The root cause is workflow-level: **the dev-story workflow has no checkpoint that forces the dev agent to reconcile a new story's behavior against prior stories' AC text before implementing**. The dev agent reads the current story's Dev Notes, sees "use IntersectionObserver", implements it, and never grep's the prior story file to notice that "single POST" was a contract, not a detail.

### Why dev-story (not create-story)?

An alternative placement for this check is `create-story/instructions.xml` — force the SM to do the grep when writing the new story. That would also work, but has two disadvantages:

1. **SMs author stories in batch** (a whole sprint at a time), and the SM doesn't yet know which AC semantics will conflict — that only becomes concrete when the DEV agent is actually implementing and picking library patterns. The check in create-story would produce noise (false positives) without the implementation context.
2. **dev-story runs once per story at implementation time**, so the grep is targeted and cheap. Any drift found can be immediately reconciled with the AC author (SM) via `*correct-course` or by amending the AC text inline before Step 5 implementation begins.

Therefore the check belongs in `dev-story/instructions.xml`, at Step 2, before any implementation begins.

### Why MANDATORY + "record NONE when no drift"?

The default failure mode of optional checks is silent skipping. Retro-9-AI1 (FULL REGRESSION GATE) solved this by making the gate MANDATORY and forcing the dev agent to cite its execution in every story's Completion Notes. This story uses the identical pattern: **the dev agent MUST record a drift-check result in Completion Notes, even if the result is "NONE"**. That way:

- A future retro can grep "🔗 AC Drift:" across story files and audit how often drift is being caught.
- A CR agent can verify the check was actually run (the prefix is unambiguous).
- The dev agent cannot claim "I didn't need to check" — they have to either record NONE with the grep pattern used, or record a finding.

### File placement (concrete)

Current Step 2 (from `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml`, as of 2026-04-20):

```xml
<step n="2" goal="Load project context and story information">
    <critical>Load all available context to inform implementation</critical>

    <action>Load {project_context} for coding standards and project-wide patterns (if exists)</action>
    <action>Parse sections: Story, Acceptance Criteria, Tasks/Subtasks, Dev Notes, Dev Agent Record, File List, Change Log, Status</action>
    <action>Load comprehensive context from story file's Dev Notes section</action>
    <action>Extract developer guidance from Dev Notes: architecture requirements, previous learnings, technical specifications</action>
    <action>Use enhanced story context to inform implementation decisions and approaches</action>
    <output>✅ **Context Loaded**
      Story and project context available for implementation
    </output>
  </step>
```

New action inserts between the last `<action>` and the `<output>` block. Step 2 remains a single coherent unit (no numbering shift downstream).

### Precedent patterns (Epic 8 / 9 / 9c retros)

This story follows the established pattern for SM workflow-improvement retros:

- **Retro-8-P2** (story-splitting-rule): added inline `<action>` check to `create-story/instructions.xml` Step 5. Cited in commit + sprint-status entry.
- **Retro-9-AI1** (full-regression-gate): added inline `<action critical="MANDATORY">` to `dev-story/instructions.xml` Step 7. Identical shape to this story's target.
- **Retro-9c-AI2** (fix-or-file): added inline `<action critical="MANDATORY" if="...">` to `dev-story/instructions.xml` Step 7. Also identical shape.

Confidence: HIGH. The pattern is well-trodden, the file is well-structured, the drift-check text is self-contained.

### Out of Scope

- Automating the grep (e.g., a Go test that enforces AC cross-references). Rationale: the grep is cheap but requires human judgment ("is this DRIFT or REUSE?"). Automation would produce too many false positives without a structured AC annotation format. That's the scope of the deferred `retro-10-AI5-cross-story-contract-versioning` spike.
- Retroactively scanning all Epic 1–10 stories for latent drift. Not this story.
- Updating the Scrum Master (SM) agent's `create-story` workflow. Future iteration if needed; not now.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#challenges`] Pattern #2 — 10-4 → 10-5 AC contract drift
- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#action-items`] AI-2 row (SM, MEDIUM priority)
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] `retro-10-AI2-ac-contract-drift-check: backlog` entry + tracking rule (Agreement 4)
- [Source: `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml`] Step 2, target placement for new action
- [Precedent: `retro-9-AI1-full-regression-gate`] — identical pattern for adding a MANDATORY action to dev-story Step 7
- [Precedent: `retro-9c-AI2-fix-or-file-test-failures`] — same pattern for conditional MANDATORY action

## Dev Agent Record

### Agent Model Used

Claude Opus 4.7 (1M context) — `claude-opus-4-7[1m]`

### Debug Log References

- `pnpm lint:all` → 0 errors, 129 pre-existing warnings (no-explicit-any + react-hooks/exhaustive-deps — baseline, unrelated to this story)
- `pnpm nx test api` → all Go packages PASS
- `pnpm nx test web` → 144 test files / 1738 tests PASS, auto-cleanup ran (2 PIDs killed)
- Grep verification on `epic-10-retro-2026-04-20.md` → AI-2 cited at lines 44, 83, 129, 187 (row + Pattern #2 + action item + summary)
- Structural verification on `dev-story/instructions.xml` → `<step n="1">` through `<step n="11">` numbering intact post-edit (Grep confirmed line positions: step 2 @ 139, step 3 @ 195, step 10 @ 437, step 11 @ 499)

### Completion Notes List

- **🔗 AC Drift: N/A** (bootstrap of the drift check itself — grep `dev-story/instructions.xml` across `_bmad-output/implementation-artifacts/retro-9*.md` + `retro-10*.md` found 1 hit in `retro-9-AI5-package-dependency-boundaries.md` but it references unrelated Step/subject matter; retro-9-AI1 + retro-9c-AI2 both modified dev-story Step 7 (not Step 2) per sprint-status comments, so no contract collision with this story's Step 2 insertion. All subsequent stories must record NONE or FOUND per the new rule.) **[CR fix M2]**
- **✅ AC #1 satisfied**: MANDATORY action inserted at Step 2 of `dev-story/instructions.xml` (lines 148–187). Check runs when the dev agent has full AC text in hand but has NOT yet written code (optimal placement, per story Dev Notes rationale).
- **✅ AC #2 satisfied**: Documentation rule embedded — `🔗 AC Drift: {prior-story-key} AC #N — {old contract} → {new contract}` format prescribed; File List annotation `(AC drift reference — see Completion Notes)` also prescribed.
- **✅ AC #3 satisfied**: `NONE` path explicit — dev agent MUST record grep pattern used + hit count + classification, so silence is impossible.
- **✅ AC #4 satisfied**: action tagged `critical="MANDATORY"`, cites "Epic 10 Retro AI-2" in tag title + "Pattern #2 (Story 10-4 → 10-5)" in precedent block, includes full concrete drift example ("single POST batch" → "≤N POSTs (one per lazy-revealed batch)", IntersectionObserver trigger, caught only by adversarial CR 10-5 H1). Precedent is narrated, not just linked — future dev agents see the story, not just a ceremony checkbox.
- **✅ AC #5 satisfied**: docs-only change (modified `.xml` file); zero Go/TS code touched. All three gates green (lint + api test + web test). XML integrity re-verified via Read + Grep.
- **✅ AC #6 status transitions executed**:
  - `ready-for-dev` → `in-progress` (Step 4 of dev-story, Amelia)
  - `in-progress` → `review` (Step 10 of dev-story, Amelia)
  - `review` → `done` deferred to `/code-review` workflow (per Task 4.4 spec)
  - Final placement line range `148–187` recorded in sprint-status.yaml comment now (the "completion note" part of 4.4); the status-value flip awaits CR.
- **🎨 UX Verification: SKIPPED** — no UI changes in this story (pure workflow-docs).
- **📦 Task 3.1 decision upheld**: `project-context.md` deliberately NOT modified. Rationale: project-context.md Rules govern code conventions (logging, layering, naming); workflow process rules belong inside the workflow file itself — consistent with retro-9-AI1 precedent (FULL REGRESSION GATE added to `dev-story/instructions.xml`, not to project-context.md).
- **📦 Task 3.2 verified**: `epic-10-retro-2026-04-20.md` already cites AI-2 adequately (row, Pattern #2 narrative, action item, summary line); no retro edit required (retro docs are frozen once shipped).

### File List

- `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` (modified) — inserted `<!-- Epic 10 Retro AI-2 ... Agreement 4 -->` comment + `<action critical="MANDATORY">` AC Contract Drift Check block (~45 lines) between last pre-existing Step 2 `<action>` and `<output>` (lines 148–196 post-CR); added sibling `<action critical="MANDATORY">` that binds `{{drift_check_result}}` (line 198–201); updated Step 2 `<output>` block to include `AC Drift Check: {{drift_check_result}}` line
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified) — `retro-10-AI2-ac-contract-drift-check` status: `ready-for-dev` → `in-progress` → `review` → `done` (post-CR); comment appended with Amelia dev-story note + final placement line range + CR fix summary
- `_bmad-output/implementation-artifacts/retro-10-AI2-ac-contract-drift-check.md` (self, modified) — all task checkboxes marked `[x]`, Dev Agent Record populated, Status `ready-for-dev` → `review` → `done`
- `_bmad-output/implementation-artifacts/retro-9-AI1-full-regression-gate.md` (AC drift reference — see Completion Notes) **[CR fix L1]** — precedent for MANDATORY-action pattern at Step 7; confirmed by grep that it modifies Step 7 only, no Step 2 collision
- `_bmad-output/implementation-artifacts/retro-9c-AI2-fix-or-file-test-failures.md` (AC drift reference — see Completion Notes) **[CR fix L1]** — precedent for conditional MANDATORY-action pattern at Step 7; same non-collision verified

### Change Log

| Date       | Change                                                                                                                                                                                                                                                                                                                                                                             | Author |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 2026-04-20 | **Task 1** — inserted AC Contract Drift Check as MANDATORY action in `dev-story/instructions.xml` Step 2 (lines 148–187); added cross-reference comment citing Pattern #2 + Agreement 4; updated Step 2 `<output>` block to include `AC Drift Check` status line                                                                                                                | Amelia |
| 2026-04-20 | **Task 2** — verification green: `pnpm lint:all` (0 errors, 129 pre-existing warnings), `pnpm nx test api` (all PASS), `pnpm nx test web` (144 files / 1738 tests PASS with auto-cleanup); XML well-formed, Step 1–11 numbering intact                                                                                                                                             | Amelia |
| 2026-04-20 | **Task 3** — documentation discoverability: confirmed `project-context.md` deliberately unchanged (workflow-level concerns stay in workflow files; consistent with retro-9-AI1 precedent); confirmed `epic-10-retro-2026-04-20.md` already cites AI-2 at lines 44/83/129/187 — no retro edit needed                                                                                 | Amelia |
| 2026-04-20 | **Task 4** — sprint-status transitions executed: `ready-for-dev` (SM Bob, create-story) → `in-progress` (Amelia, dev-story start, Step 4) → `review` (Amelia, dev-story complete, Step 10). Line-range note `148–187` recorded in sprint-status comment. Task 4.4 status-value flip (`review → done`) deferred to `/code-review` workflow per task spec                              | Amelia |
| 2026-04-20 | **CR fixes** (Amelia /code-review adversarial self-review) — 7 findings (1 HIGH / 4 MED / 2 LOW) all fixed: **H1** `{{drift_check_result}}` now bound by sibling `<action critical="MANDATORY">` (lines 198–201) so Step 2 output renders concrete value, not unresolved placeholder; **M1** Documentation rule expanded from binary (FOUND/NONE) to ternary (FOUND/NONE/N/A) — N/A branch now has defined semantics with mandatory reason; **M2** bootstrap Completion Note now cites actual grep scope (retro-9*.md + retro-10*.md) + classification reasoning, no longer silent-skip; **M3** trigger condition (a) reworded from literal `"changes behavior of {feature X}"` to semantic `"changes the observable behavior... any phrasing"`; **M4** Precedent block now links `epic-10-retro-2026-04-20.md` section "Challenges → Pattern #2"; **L1** File List now annotates retro-9-AI1 + retro-9c-AI2 as `(AC drift reference — see Completion Notes)` — self-application of the new rule; **L2** keyword example list now abstraction-consistent (`"batch semantics"` + `"lazy-load"` replace the route-path example `"check-owned"`). XML well-formedness re-verified via `xmllint --noout` PASS. Status transitioned `review → done`. Final placement post-CR: `instructions.xml` lines 148–201 (original MANDATORY block + comment + binding action + output). | Amelia (CR) |
