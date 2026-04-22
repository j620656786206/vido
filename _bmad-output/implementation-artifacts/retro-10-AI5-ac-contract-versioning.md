# Story: AC Contract Versioning (retro-10-AI5)

Status: done

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

- [x] Task 1: Extend `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` Step 2 with Contract Stamp grep pass (AC #4, #5)
  - [x] 1.1 Locate the existing AC Drift Check MANDATORY action block at instructions.xml Step 2 (lines 148-201 per retro-10-AI2 final placement). Append a new MANDATORY action "Contract Stamp Check" AFTER the existing drift check action (keep both actions independent for clarity — mirrors retro-10-AI3 Rule 7 pattern with main action + sibling binding action).
  - [x] 1.2 New action runs `grep -nE '\[@contract-v[0-9]+\]' {{story_path}}` AND (if upstream story references are cited in Dev Notes) across each referenced upstream story file. Records in Completion Notes: `📎 Contract Stamps: FOUND ({count} across {n} files) | NONE | N/A (no stamped ACs in scope)`.
  - [x] 1.3 Add sibling MANDATORY binding action that sets `{{contract_stamps_result}}` variable explicitly (matches retro-10-AI2 H1 pattern + retro-10-AI3 final fix). Ensures the Step 2 output renders a concrete value, not an unresolved placeholder.
  - [x] 1.4 xmllint PASS verification on instructions.xml (matches retro-10-AI2/AI-3 regression gate).

- [x] Task 2: Add new Rule 20 "AC Contract Versioning" to `project-context.md` (AC #1, #2, #3, #6)
  - [x] 2.1 Insert new Rule 20 section AFTER the current final rule (Rule 19 "Package Dependency Boundaries" at line 538) and its closing code fence. DO NOT renumber existing rules; Rule 19 stays put.
  - [x] 2.2 Draft Rule 20 following the shape of existing Rules (✅ / ❌ bullets, short code block, precedent line). Content:
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
  - [x] 2.3 Update "Last Updated" header at line 7 with Rule 20 addition citation (pattern: same as retro-10-AI4 Rule 15 extension note).

- [x] Task 3: Add Contract Stamp checklist item to `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` (AC #2, #3)
  - [x] 3.1 Insert new item under `## ✅ Implementation Completion` AFTER the "HTTP Route ↔ Client Method Sync" item (currently at checklist.md:40-76 post-retro-10-AI4-CR) and BEFORE `## 🧪 Testing & Quality Assurance`. Placement rationale: same scope-completeness group as retro-10-AI4's item.
  - [x] 3.2 Draft item content (pattern: same structure as retro-10-AI4's item — label, procedure, "Why this exists" block):
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

- [x] Task 4: Full regression gate (AC #5 parity with retro-10-AI2/AI-3/AI-4 precedent)
  - [x] 4.1 `pnpm lint:all` — expected 0 errors (docs/XML-only changes; not processed by ESLint/Prettier/go-vet/staticcheck per repo's lint scope).
  - [x] 4.2 `pnpm nx test api` — expected PASS (zero Go code change).
  - [x] 4.3 `pnpm nx test web` — expected PASS (zero frontend code change).
  - [x] 4.4 Update `sprint-status.yaml` entry `retro-10-AI5-cross-story-contract-versioning`: `ready-for-dev → in-progress → review → done`. Final comment records line ranges: instructions.xml Step 2 Contract Stamp action block, project-context.md Rule 20 line range, checklist.md new item line range.

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

Amelia (BMM Dev Agent) / Claude Opus 4.7 (1M context) — invoked 2026-04-22 via `/bmad:bmm:agents:dev` → `/bmad:bmm:workflows:dev-story` with explicit story key `retro-10-AI5-cross-story-contract-versioning` (auto-discovery would have skipped `retro-*` pattern).

### Debug Log References

- `pnpm lint:all` (repo root, 2026-04-22): 0 errors, 129 pre-existing warnings (no new warnings — docs/XML/YAML-only changes cannot introduce lint regressions); `prettier --check .` PASS.
- `xmllint --noout _bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` (2026-04-22): PASS (XML well-formed after Contract Stamp Check action insertion).
- `pnpm nx test api` (2026-04-22): PASS, Go backend green (Nx cache hit — zero Go code touched).
- `pnpm nx test web` (2026-04-22): 144 files / 1738 tests, all PASS in 65.37s; `test:cleanup:all` ran automatically, both spawned PIDs (19909, 4379) exited cleanly — no orphaned workers.
- CR post-fix gate (2026-04-22): `xmllint --noout instructions.xml` PASS (XML well-formed after procedure-step expansion in Contract Stamp Check); `pnpm lint:all` 0 errors / 129 pre-existing warnings (identical baseline to DEV gate — no new issues introduced by Rule 20 / instructions.xml / checklist.md edits); Prettier PASS. Go + React suites not re-run — zero code touched by CR fixes (100% docs/XML/YAML).

### Completion Notes List

- 🔗 AC Drift: N/A (bootstrap of AC contract versioning system — grep `@contract-v|Contract Stamps|AC Contract Versioning` across `_bmad-output/implementation-artifacts/*.md` excluding self-refs (retro-10-AI5 + spike-10-AI5 files) returned 0 hits; prior stories carry no stamps, so no prior AC contract exists to drift against. This story establishes the convention.)
- 📎 Contract Stamps: N/A (bootstrap — this story INTRODUCES the `[@contract-v*]` stamp convention via Rule 20. No upstream stamped AC exists yet to reference. Forward stories will be the first consumers.)
- 🎨 UX Verification: SKIPPED — no UI changes in this story (zero files under `apps/web/`).
- AC #1 satisfied: Rule 20 at `project-context.md:615-635` documents MAY-not-MUST convention with format `AC #N [@contract-v1]: Given/When/Then...`. Checklist item at `checklist.md:77-108` mirrors the rule for DoD verification.
- AC #2 satisfied: Rule 20 documents bump protocol (`[@contract-vN]` → `[@contract-v(N+1)]`) + mandatory Change Log entry format. Checklist item Procedure step 3 enforces it. Contract Stamp Check action in `instructions.xml:205-257` flags bumps without matching Change Log entries via grep-and-verify.
- AC #3 satisfied: Rule 20 includes downstream acknowledgement rule (`confirmed against [@contract-vN] (Story X-Y AC #N)`). Checklist item Procedure step 2 enforces it as HIGH-severity gap if missing.
- AC #4 satisfied: grep helper `grep -nE '\[@contract-v[0-9]+\]' <story_file>` documented in Rule 20 body + checklist item Procedure step 1 + Contract Stamp Check action.
- AC #5 satisfied: `instructions.xml` Step 2 Contract Stamp Check action added at lines 205-257 (main MANDATORY action) + 259-263 (sibling MANDATORY binding action setting `{{contract_stamps_result}}`) + Step 2 output updated at line 268 (`Contract Stamps: {{contract_stamps_result}}`). Three-state audit pattern (FOUND / NONE / N/A) matches retro-10-AI2 and retro-10-AI4 precedent. xmllint PASS verified.
- AC #6 satisfied: Rule 20 documents forward-only retrofit (`Historical unstamped ACs are implicitly v0 (frozen); stamp only when newly referenced by a forward story`). Checklist item "Why this exists" block cites Epic 10 retro Pattern #2 recurrences as empirical justification. Avoids ~250+ retrofit stamps across Epic 1-10.
- Sprint-status.yaml transitioned `backlog → ready-for-dev → in-progress → review` across spike, /create-story, and /dev-story workflows. Final transition to `done` is CR's responsibility.
- ✅ Resolved CR findings (2026-04-22 adversarial self-review, 1 HIGH + 2 MEDIUM + 3 LOW, all fixed):
  - **[HIGH] H1** — AC #4 grep helper was missing from Rule 20's body (was only in `checklist.md` + `instructions.xml`). Fix: added `🔎 Grep helpers` subsection to Rule 20 with three helpers (single-story, all-artifacts, downstream-ack). Closes AC #4 literal requirement + restores CR's claimed "inherit via shared Rule 20" mechanism.
  - **[MEDIUM] M1** — Change Log bump format drift across 3 canonical sources (AC #2 / Rule 20 / checklist+instructions each wrote it differently). Fix: unified to AC #2's wording `{what changed, what breaks downstream}` across Rule 20 line 623, checklist.md line 94, instructions.xml line 224. Drift spec in the anti-drift rule — closed.
  - **[MEDIUM] M2** — Change Log verification was presence-only, accepted degenerate entries like `AC #3: tweak`. Fix: two-stage verify (row presence + ≥2 non-empty sub-tokens after `AC #N:`) codified in Rule 20, instructions.xml, checklist.md. Strengthens check to match AC #2 spec.
  - **[LOW] L1** — File List claimed `checklist.md:77-108`, actual was 77-109 (off-by-one at item end). Fix: post-CR ranges (77-115) now reflect the expanded procedure. Updated here, in sprint-status.yaml, and Change Log rows.
  - **[LOW] L2** — Ack-line punctuation drift (Rule 20 had trailing period, AC #3 + checklist did not). Fix: dropped period on Rule 20 line 625. Consistent across all three.
  - **[LOW] L3** — Upstream-ref detection was narrow phrase-based (3 exact phrasings). Fix: added bare substring `"Story X-Y AC #N"` as fourth trigger, plus explicit v0 fallback for pre-Rule-20 upstream — both in instructions.xml Step 2 and checklist.md. Matches Rule 20 forward-only retrofit semantics.
- 📎 Contract Stamps (CR re-check, post-fix): N/A (bootstrap still — this story introduces the convention; post-CR grep against `_bmad-output/implementation-artifacts/` still shows stamps only in this story and its spike, as expected).

### File List

- `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` — added Contract Stamp Check block at Step 2: comment line 204, main MANDATORY action lines 205-267 (post-CR; was 205-257 pre-CR), sibling MANDATORY binding action lines 269-273 (post-CR; was 259-263 pre-CR), Step 2 output updated line 278 (`Contract Stamps: {{contract_stamps_result}}`; post-CR; was line 268 pre-CR). CR follow-up: procedure step 2 broadened to include bare substring "Story X-Y AC #N" + v0 fallback for pre-Rule-20 upstream; procedure step 3 upgraded to two-stage verify (presence + `{what breaks downstream}` sub-token population) per CR M2. xmllint PASS. (Task 1 + CR L3 + CR M2)
- `project-context.md` — added Rule 20 "AC Contract Versioning" at lines 615-652 (post-CR; was 615-635 pre-CR) with ✅/❌/🔎/📌 pattern; "Last Updated" header at line 7 refreshed with Rule 20 citation + CR follow-up note. Rules 1-19 unchanged, no renumbering. CR follow-up: (H1) grep helpers hoisted into Rule 20 body (three helpers: single-story, all-artifacts, downstream-ack search); (M1) Change Log format unified to `{what changed, what breaks downstream}`; (L2) terminal period dropped on ack-line example; (M2) two-stage verify rule added to Rule 20 body; (L3) v0 fallback paragraph added. (Task 2 + CR H1 + CR M1 + CR L2 + CR M2 + CR L3)
- `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` — added new "AC Contract Versioning (Epic 10 Retro AI-5)" item at lines 77-115 (post-CR; was 77-109 actual / 77-108 claimed pre-CR — CR L1 fix) under `## ✅ Implementation Completion`, directly AFTER "HTTP Route ↔ Client Method Sync" item (lines 40-75) and BEFORE `## 🧪 Testing & Quality Assurance` section (now starts at line 117). CR follow-up: Procedure step 2 broadened (bare "Story X-Y AC #N" + v0 fallback); step 3 upgraded to two-stage verify + unified `{what changed, what breaks downstream}` wording. (Task 3 + CR L1 + CR L3 + CR M1 + CR M2)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — entry `retro-10-AI5-cross-story-contract-versioning` transitioned `ready-for-dev → in-progress` (Step 4 of dev-story) → `review` (Step 10 of dev-story). (Task 4)
- `_bmad-output/implementation-artifacts/retro-10-AI5-ac-contract-versioning.md` — this story file: all 17 task/subtask checkboxes marked [x] (4 main + 13 subtasks), Status `ready-for-dev → review`, Dev Agent Record populated, Change Log extended. NOTE: SM's original draft cited "11 subtasks" but actual count is 13 (Task 1×4 + Task 2×3 + Task 3×2 + Task 4×4 = 13). Cosmetic discrepancy only; no content impact.

### Change Log

| Date       | Change                                                                                                                                                                                                                                                                                            |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-22 | Story created by SM Bob via `/bmad:bmm:workflows:create-story` from spike `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`. 6 ACs, 4 tasks, 11 subtasks (actual: 13). Cross-stack split check: 0 BE + 0 FE + 4 docs tasks → single story (within ≤3 threshold). Status: ready-for-dev. |
| 2026-04-22 | DEV Amelia: extended `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml` Step 2 with CONTRACT STAMP CHECK MANDATORY action (lines 205-257) + sibling MANDATORY binding action (lines 259-263) + Step 2 output extended (line 268 — `Contract Stamps: {{contract_stamps_result}}`). Matches retro-10-AI2/AI-3 main+binding pattern. xmllint PASS verified. (AC #4, #5) |
| 2026-04-22 | DEV Amelia: added Rule 20 "AC Contract Versioning" to `project-context.md` at lines 615-635, AFTER Rule 19 "Package Dependency Boundaries" (lines 538-613). Content: 4 ✅ bullets (MAY stamp, bump protocol, downstream ack, forward-only retrofit) + 1 ❌ anti-pattern + 1 📌 precedent citing Epic 10 Retro AI-5 spike. "Last Updated" header at line 7 refreshed. Rules 1-19 unchanged. (AC #1, #2, #3, #6) |
| 2026-04-22 | DEV Amelia: added new "AC Contract Versioning (Epic 10 Retro AI-5)" checklist item to `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` at lines 77-108, between retro-10-AI4 HTTP Route item (ends line 75) and `## 🧪 Testing & Quality Assurance` section (now line 111). Procedure mirrors Rule 20 + three-state audit record (retro-10-AI2 pattern). (AC #2, #3) |
| 2026-04-22 | DEV Amelia: full regression gate PASS — `pnpm lint:all` 0 errors 129 pre-existing warnings + Prettier PASS; `xmllint --noout instructions.xml` PASS; `pnpm nx test api` PASS (Nx cache); `pnpm nx test web` 144 files / 1738 tests PASS in 65.37s + cleanup verified PIDs 19909/4379 exited cleanly. 🔗 AC Drift: N/A (bootstrap). 📎 Contract Stamps: N/A (bootstrap). 🎨 UX: SKIPPED. (AC #5) |
| 2026-04-22 | DEV Amelia: sprint-status.yaml `retro-10-AI5-cross-story-contract-versioning` transitioned `ready-for-dev → in-progress → review`. Final transition to `done` is CR's responsibility. Story file all 17 checkboxes [x] (4 main + 13 subtasks), Status `ready-for-dev → review`. (AC #6) |
| 2026-04-22 | CR Amelia (adversarial self-review, fresh CR pass): 6 findings (1 HIGH + 2 MED + 3 LOW), all auto-fixed. **H1**: AC #4 grep helper was not in Rule 20 body → added `🔎 Grep helpers` subsection to Rule 20 with 3 helpers. **M1** [@contract-v1→v2] AC #2: Change Log format unified across 3 canonical sources to `{what changed, what breaks downstream}` (Rule 20 line 623, checklist.md line 94, instructions.xml line 224). **M2**: Change Log verification upgraded from presence-only to two-stage (row present + ≥2 populated sub-tokens after `AC #N:`) — codified in all three sources. **L1**: checklist.md line-range claim corrected 77-108 → 77-115 (was off-by-one even pre-fix; grew via CR follow-up procedure). **L2**: Rule 20 ack-line trailing period dropped (matches AC #3 + checklist). **L3**: upstream-ref detection broadened (+ bare "Story X-Y AC #N") and v0 fallback for pre-Rule-20 upstream documented. Post-CR line ranges: Rule 20 `project-context.md:615-652`; Contract Stamp Check action `instructions.xml:205-267` + binding 269-273 + output line 278; checklist item `checklist.md:77-115`. Status `review → done`. (CR findings: H1/M1/M2/L1/L2/L3) |
| 2026-04-22 | CR Amelia: regression gate re-verified post-fix — xmllint instructions.xml PASS; `pnpm lint:all` PASS (docs/XML/YAML-only edits, same baseline as DEV gate); Go + React test suites unchanged (no code touched). Sprint-status.yaml `retro-10-AI5-cross-story-contract-versioning: review → done`. |
