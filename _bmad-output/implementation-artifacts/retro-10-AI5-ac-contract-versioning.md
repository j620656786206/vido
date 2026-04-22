# Story: AC Contract Versioning (retro-10-AI5)

Status: ready-for-dev

## Story

As a Dev Agent (Amelia) / CR reviewer reading a story that references an AC from another story as a contract baseline,
I want each cross-story-referenced AC to carry a `[@contract-vN]` version stamp + mandatory Change Log on bump + downstream acknowledgement,
so that cross-story AC drift (Pattern #2 from Epic 10 retro) is caught at story-authoring time via grep — not reactively at retro time via forensics.

## Acceptance Criteria

1. Given a story author writing an AC that defines a contract (endpoint shape, request count, path, response payload, etc.), when the AC lands for the first time AND the AC is expected to be referenced by downstream stories, then the AC text MAY carry a `[@contract-v1]` prefix in the format `AC #N [@contract-v1]: Given/When/Then…`. The stamp is MAY not MUST — only cross-story-referenced ACs need versioning; ACs that remain self-contained stay unstamped and are implicitly `v0`.

2. Given an AC already stamped `[@contract-vN]`, when the author changes the contract shape or semantics (not wording), then (a) the stamp MUST bump to `[@contract-v(N+1)]` AND (b) the same story's Change Log MUST carry an entry formatted exactly: `| {Date} | [@contract-vN→v(N+1)] AC #N: {what changed, what breaks downstream} |`. A bump without a matching Change Log entry is a MEDIUM CR finding per the adversarial code-review pattern.

3. Given a downstream story references an upstream AC as a precondition (e.g., "per Story X-Y AC #N"), when the downstream story is authored, then Dev Notes MUST include a line: `confirmed against [@contract-vN] (Story X-Y AC #N)` — a non-silent acknowledgement of which upstream version was validated. A missing acknowledgement when an upstream reference exists is a HIGH CR finding.

4. Given the grep helper `grep -nE '\[@contract-v[0-9]+\]' <story_file>`, when run against any story file, then it lists every stamped AC with line numbers. This helper is documented in `project-context.md` Rule 20 (new rule) and usable by both DEV and CR workflows.

5. Given `dev-story/instructions.xml` Step 2 already runs an AC Drift Check (retro-10-AI2 action) that greps for cross-story references, when this story lands, then Step 2 MUST additionally run `grep -nE '\[@contract-v[0-9]+\]'` across the story file + all referenced upstream stories AND record a `📎 Contract Stamps: FOUND | NONE | N/A` line in Completion Notes — matching the three-state audit pattern established by retro-10-AI2 and retro-10-AI4.

6. Given the retrofit strategy is forward-only, when a new story would reference a historical (unstamped, implicit-v0) AC, then the author of the NEW story MUST either (a) stamp the historical AC `[@contract-v1]` in-place as part of the new story's scope AND add the historical story file to File List, OR (b) defer and file a follow-up story. Historical ACs are implicitly frozen at `v0` and MUST NOT be bumped without a new story owning the change.

## Tasks / Subtasks

- [ ] Task 1: Extend `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` Step 2 with Contract Stamp grep pass (AC #4, #5)
  - [ ] 1.1 Locate the existing AC Drift Check MANDATORY action block at instructions.xml Step 2 (lines 148-201 per retro-10-AI2 final placement). Append a new MANDATORY action "Contract Stamp Check" AFTER the existing drift check action (keep both actions independent for clarity — mirrors retro-10-AI3 Rule 7 pattern with main action + sibling binding action).
  - [ ] 1.2 New action runs `grep -nE '\[@contract-v[0-9]+\]' {{story_path}}` AND (if upstream story references are cited in Dev Notes) across each referenced upstream story file. Records in Completion Notes: `📎 Contract Stamps: FOUND ({count} across {n} files) | NONE | N/A (no stamped ACs in scope)`.
  - [ ] 1.3 Add sibling MANDATORY binding action that sets `{{contract_stamps_result}}` variable explicitly (matches retro-10-AI2 H1 pattern + retro-10-AI3 final fix). Ensures the Step 2 output renders a concrete value, not an unresolved placeholder.
  - [ ] 1.4 xmllint PASS verification on instructions.xml (matches retro-10-AI2/AI-3 regression gate).

- [ ] Task 2: Add new Rule 20 "AC Contract Versioning" to `project-context.md` (AC #1, #2, #3, #6)
  - [ ] 2.1 Insert new Rule 20 section AFTER the current final rule (Rule 19 "Package Dependency Boundaries" at line 538) and its closing code fence. DO NOT renumber existing rules; Rule 19 stays put.
  - [ ] 2.2 Draft Rule 20 following the shape of existing Rules (✅ / ❌ bullets, short code block, precedent line). Content:
    ```
    AC Contract Versioning:
      ✅ Cross-story-referenced ACs MAY carry `[@contract-v1]` prefix.
         Format: `AC #N [@contract-v1]: Given/When/Then...`
      ✅ When changing a stamped AC's contract shape/semantics, bump
         `[@contract-vN]` → `[@contract-v(N+1)]` AND add Change Log entry:
         `| {Date} | [@contract-vN→v(N+1)] AC #N: {what changed} |`
      ✅ Downstream stories referencing a stamped AC MUST record in Dev Notes:
         `confirmed against [@contract-vN] (Story X-Y AC #N)`.
      ✅ Historical unstamped ACs are implicitly `v0` (frozen); stamp only
         when newly referenced by a forward story (forward-only retrofit).
      ❌ Bumping a stamp without a Change Log entry or without notifying
         downstream consumers via the ack rule.
      📌 Precedent (Epic 10 Retro AI-5, spike 2026-04-22): Pattern #2 from
         Epic 10 retro — cross-story AC drift recurred 3 times across 3 epics.
         retro-10-AI2 AC Drift Check caught story-ID references; this rule
         closes the contract-shape gap. Spike doc:
         `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`.
    ```
  - [ ] 2.3 Update "Last Updated" header at line 7 with Rule 20 addition citation (pattern: same as retro-10-AI4 Rule 15 extension note).

- [ ] Task 3: Add Contract Stamp checklist item to `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` (AC #2, #3)
  - [ ] 3.1 Insert new item under `## ✅ Implementation Completion` AFTER the "HTTP Route ↔ Client Method Sync" item (currently at checklist.md:40-76 post-retro-10-AI4-CR) and BEFORE `## 🧪 Testing & Quality Assurance`. Placement rationale: same scope-completeness group as retro-10-AI4's item.
  - [ ] 3.2 Draft item content (pattern: same structure as retro-10-AI4's item — label, procedure, "Why this exists" block):
    ```markdown
    - [ ] **AC Contract Versioning (Epic 10 Retro AI-5):** If any AC in this
          story carries `[@contract-vN]` stamp OR references an upstream stamped
          AC, run the contract stamp audit before marking the story complete.

          Verification procedure:
            1. Grep: `grep -nE '\[@contract-v[0-9]+\]' {story_file}` — list
               every stamped AC in this story.
            2. For EACH upstream AC referenced in Dev Notes (e.g.,
               "per Story X-Y AC #N"), grep the upstream story file for the
               AC's current stamp. Record in Dev Notes: `confirmed against
               [@contract-vN] (Story X-Y AC #N)`.
            3. If any stamped AC in this story has BUMPED its version (v1→v2),
               verify a matching Change Log entry exists in the same story:
               `| {Date} | [@contract-vN→v(N+1)] AC #N: {what changed} |`.
            4. Record result in Dev Agent Record → Completion Notes:
               `📎 Contract Stamps: FOUND ({count} across {n} files) | NONE | N/A`.

          Why this exists: Epic 10 retro Pattern #2 — cross-story AC drift
          recurred 3 times across 3 epics. retro-10-AI2 AC Drift Check caught
          story-ID references; this check closes the contract-shape gap at
          story-completion time, not at CR/retro time.
    ```

- [ ] Task 4: Full regression gate (AC #5 parity with retro-10-AI2/AI-3/AI-4 precedent)
  - [ ] 4.1 `pnpm lint:all` — expected 0 errors (docs/XML-only changes; not processed by ESLint/Prettier/go-vet/staticcheck per repo's lint scope).
  - [ ] 4.2 `pnpm nx test api` — expected PASS (zero Go code change).
  - [ ] 4.3 `pnpm nx test web` — expected PASS (zero frontend code change).
  - [ ] 4.4 Update `sprint-status.yaml` entry `retro-10-AI5-cross-story-contract-versioning`: `ready-for-dev → in-progress → review → done`. Final comment records line ranges: instructions.xml Step 2 Contract Stamp action block, project-context.md Rule 20 line range, checklist.md new item line range.

## Dev Notes

### Spike Output Reference (mandatory read before starting)

`_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md` — BMAD Party Mode session 2026-04-22 output from Winston (Architect) + Mary (Analyst) + Bob (SM). Contains:

- 3-option trade-off matrix (header stamp / test-snapshot tag / defer)
- Drift-type coverage mapped against Epic 8-10 retro history
- Resolution of 3 decision points (format, bump protocol, retrofit strategy)
- Empirical rationale for MAY-not-MUST, forward-only strategy, and manual-bump + Change Log

DEV: read the spike doc BEFORE reading this story's Dev Notes. The spike is the "why"; this story is the "what".

### Root Cause

Epic 10 retro (2026-04-20) Pattern #2: **cross-story AC drift**. Recurrent pattern across Epic 8-10:

- **Story 10-4 path rename** (Epic 10): `/movies/check-owned → /media/check-owned` mid-implementation.
- **Epic 8 TD3/TD4**: story claims done with Change Log empty, shape drifted silently.
- **retro-10-AI4 meta-irony** (just landed): the anti-precedent-drift story itself cited a wrong precedent path. CR caught it, but only because CR ran adversarially.

retro-10-AI2 AC Drift Check (dev-story Step 2) closed the **story-ID reference** gap. Missing layer: **contract shape/semantics** gap. AI-5 closes it.

### Why MAY not MUST (AC #1)

Empirical baseline from Epic 8-10: ~15% of ACs are actually cross-referenced by downstream stories. Enforcing MUST on 100% of ACs forces ~85% unnecessary stamps and dilutes signal. MAY + author judgment + grep helper = lower noise, higher signal.

### Why Manual Bump (AC #2)

Auto-bump would require structural semantic diff of ACs — impossible with natural-language ACs. Manual bump is a **declaration of intent**: the author asserts "I am changing the contract", not "I am tweaking wording". The Change Log entry is the accountability trail.

### Why Forward-Only Retrofit (AC #6)

Retrofit cost estimated at **250+ manual stamps** across Epic 1-10 historical stories for zero immediate value. Historical ACs are frozen by definition — they aren't being referenced by new stories today unless by a specific new-story author's deliberate choice. Forward-only = pay cost only where benefit exists.

### Placement Rationale (Rule 20 new rule, not Rule 15 sub-section)

Rule 15 is "Pre-commit Self-verification" — checklist-style checks before marking task complete (wiring, DB, Swagger, HTTP Route ↔ Client Method Sync). AC Contract Versioning is a **story-authoring protocol**, not a pre-commit check. Different concern → new rule. Rule 20 is next available (current max is Rule 19 "Package Dependency Boundaries" at line 538).

### Cross-Stack Split Check (Agreement 5, Epic 8 Retro + Epic 9c Retro AI-1 enforced)

- **Backend task count:** 0 (no Go code, no handlers, no services, no DB).
- **Frontend task count:** 0 (no React, no frontend code).
- **Docs/workflow task count:** 4 (all tasks modify Markdown + XML + YAML).
- **Threshold:** both counts ≤3 → single story, no split required. ✅

### Precedent Stories (shape + pattern to mirror)

- `retro-10-AI2-ac-contract-drift-check.md` — sibling AC Drift Check at dev-story Step 2. Contract Stamp Check MUST be placed IMMEDIATELY AFTER this action to keep the drift-defense stack local. Pattern: main MANDATORY action + sibling binding action.
- `retro-10-AI4-http-route-client-method-gap.md` — sibling Rule 15 extension precedent. Pattern: checklist item with label + verification procedure + "Why this exists" block + precedent citation. Use this shape for Task 3.2's checklist item.
- `retro-10-AI3-rule7-wire-format-cr-check.md` — sibling MANDATORY action with sibling binding pattern. Final placement: instructions.xml lines 96-188 (main action) + 190-194 (sibling binding) + line 228 (findings summary). Use this shape for Task 1.1/1.3.

### Grep Patterns (for DEV to use during implementation)

```bash
# List every stamped AC in a story file
grep -nE '\[@contract-v[0-9]+\]' story.md

# Find all stamped ACs across all implementation-artifacts
grep -rnE '\[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/

# Find stories that acknowledge upstream contracts
grep -rnE 'confirmed against \[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/
```

### Out of Scope

- **Automation of contract-hash computation (Opt 2 rejected in spike).** Too fragile for natural-language ACs. Revisit if Opt 1 proves insufficient after 2 forward epics.
- **Cross-system contract versioning (API consumers, SDK bindings, frontend-backend wire formats).** This story targets internal story-to-story AC references ONLY. Rule 7 (Wire Format) and retro-10-AI3 handle backend error-code wire contracts separately.
- **Retrofit of Epic 1-10 historical ACs.** Explicitly out per AC #6 and spike DP3.
- **Touching code-review/instructions.xml.** CR's adversarial pass will naturally inherit the grep helper via shared project-context.md Rule 20. No dedicated CR-side action needed.

### Risk Assessment

- **False-positive risk:** ZERO. Opt 1 is pure prefix convention; grep is deterministic.
- **False-negative risk:** LOW-MEDIUM. Author forgets to stamp → downstream CR (retro-10-AI4 pattern) will still catch the shape mismatch. Belt-and-suspenders defense.
- **Regression risk:** ZERO. Docs/XML/YAML-only changes; pattern mirrors 3 prior retro-10-AI* stories that landed clean.

## References

- [Source: `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`] — this story's full rationale + options matrix + decision points
- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#challenges`] — Pattern #2 (cross-story AC drift) source motivation
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI2-ac-contract-drift-check.md`] — sibling at dev-story Step 2; insertion point for Task 1's new action
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI4-http-route-client-method-gap.md`] — sibling Rule extension + checklist pattern; shape to mirror for Task 2/Task 3
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI3-rule7-wire-format-cr-check.md`] — sibling MANDATORY action + sibling binding pattern; shape to mirror for Task 1.1/1.3
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] — `retro-10-AI5-cross-story-contract-versioning: backlog` (transitioning to `ready-for-dev` as this story file is saved)
- [Source: `project-context.md` current state] — Rule 19 "Package Dependency Boundaries" at line 538; Rule 20 is next available

## Dev Agent Record

### Agent Model Used

_Pending — to be filled by Dev Agent Amelia on `/bmad:bmm:agents:dev` → `/bmad:bmm:workflows:dev-story` invocation._

### Debug Log References

_Pending._

### Completion Notes List

_Pending._

### File List

_Pending. Expected: `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` (Task 1), `project-context.md` (Task 2), `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` (Task 3), `_bmad-output/implementation-artifacts/sprint-status.yaml` (Task 4), this story file._

### Change Log

| Date       | Change                                                                                                                                                                                                                                                                                            |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-22 | Story created by SM Bob via `/bmad:bmm:workflows:create-story` from spike `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`. 6 ACs, 4 tasks, 11 subtasks. Cross-stack split check: 0 BE + 0 FE + 4 docs tasks → single story (within ≤3 threshold). Status: ready-for-dev. |
