# Story: Rule 7 Wire Format Auto-Check in Code Review Workflow

Status: ready-for-dev

## Story

As a Code Reviewer (Amelia / Murat) performing adversarial `/bmad:bmm:workflows:code-review` on Go backend stories,
I want `code-review/instructions.xml` to enforce a mandatory "Rule 7 Wire Format Check" that greps for error code constants missing a known source prefix,
so that wire-format violations (e.g., `INVALID_YEAR_RANGE` without `TMDB_`) are caught automatically in CR rather than escaping to a wire-contract re-commit, as happened with Story 10-1a.

## Acceptance Criteria

1. Given a code reviewer runs `/bmad:bmm:workflows:code-review` on a story that touched Go backend files under `apps/api/internal/**`, when Step 3 (Execute adversarial review) runs, then the workflow executes a mandatory "Rule 7 Wire Format Check" action that greps the files in the review scope for error code string constants (pattern: `Err[A-Z]\w* = "[A-Z][A-Z0-9_]*"` or `ErrCode\w+ = "..."`) and flags any whose right-hand-side string does NOT start with a known Rule 7 source prefix
2. Given the wire-format grep identifies a violation (e.g., `ErrCodeInvalidYearRange = "INVALID_YEAR_RANGE"` without `TMDB_`), when the CR agent categorizes findings, then this finding is flagged as `HIGH` severity with the explicit annotation "auto-fixable" — so when the user picks option `[1] Fix them automatically` at Step 4, the CR agent's fix is: (a) prefix the code with the correct source (e.g., `"INVALID_YEAR_RANGE"` → `"TMDB_INVALID_YEAR_RANGE"`), (b) update ALL call sites (test assertions, Swagger annotations, any hard-coded response-body expectations), (c) update the story's File List + Completion Notes
3. Given the grep finds no violations, when the CR agent proceeds to Step 4, then the CR agent MUST still record in the review findings summary: `🔒 Rule 7 Wire Format: PASS (N error codes checked, all prefixed correctly)` — so the check is auditable and cannot be silently skipped (identical pattern to retro-10-AI2's drift-check audit rule)
4. Given the check lists the authoritative Rule 7 source prefixes inline (so CR doesn't need to cross-read project-context.md), when a new source prefix is added to Rule 7 in the future, then the check text includes a "last synced with project-context.md Rule 7 on YYYY-MM-DD" header so drift between the two files is visible on inspection — reviewer updates the date when extending the list
5. Given the check triggers only for files under `apps/api/internal/**` (Go backend error codes), when the review scope is pure frontend (apps/web/) or docs-only, then the check self-skips with a recorded `🔒 Rule 7 Wire Format: N/A (no Go error-code files in scope)` — no false alarm on non-Go changes
6. Given the `code-review/instructions.xml` modification lands, when `pnpm lint:all` + `pnpm nx test api` + `pnpm nx test web` run post-change, then ALL pass with zero regressions (this story modifies zero Go or TS source code; only the XML workflow file)
7. Given the sprint-status.yaml entry `retro-10-AI3-rule7-wire-format-cr-check`, when this story completes dev-story → code-review, then the status transitions `backlog → ready-for-dev → in-progress → review → done`, and a completion note captures the final placement (step number + line range) inside `instructions.xml`

## Tasks / Subtasks

- [ ] Task 1: Add "Rule 7 Wire Format Check" action to `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` (AC: #1, #2, #3, #4, #5)
  - [ ] 1.1 Placement decision: insert a new action block in Step 3 "Execute adversarial review" AFTER the "Story File Integrity Audit" action (lines ~88–94) and BEFORE the "Code Quality Deep Dive" action (lines ~96–103). Rationale: Rule 7 check is a **contract-level** project-wide gate, not a per-file inspection — placing it ahead of the per-file deep dive means violations are caught systematically, not by happenstance while reviewing a specific file. Keeps Step 3's logical progression: git-vs-story → ACs → tasks → story hygiene → **wire contract** → per-file code quality → min-issues retry.
  - [ ] 1.2 Draft the new action XML block with this shape:
    ```xml
    <!-- Rule 7 Wire Format Check (Epic 10 Retro AI-3) -->
    <action critical="MANDATORY">RULE 7 WIRE FORMAT CHECK:
      Rule 7 of project-context.md mandates that every error code string constant
      in the Go backend follow the format `{SOURCE}_{ERROR_TYPE}` where {SOURCE}
      is a registered prefix. This check catches violations that ship as
      wire-contract drift (what the frontend / API consumers see as error.code
      in ApiResponse<T>.error).

      Scope filter — run the check ONLY if the review file list contains at least
      one file under `apps/api/internal/**`. If pure-frontend or docs-only review,
      record: "🔒 Rule 7 Wire Format: N/A (no Go error-code files in scope)" and
      skip to the next action.

      Known source prefixes (last synced with project-context.md Rule 7 on 2026-04-20):
        TMDB_, AI_, DB_, VALIDATION_, SUBTITLE_, PLUGIN_, SCANNER_, SSE_, LIBRARY_

      Grep procedure (scoped to in-review Go files under apps/api/internal/**):
        1. Run: grep -rnE 'Err[A-Z]\w* *= *"[A-Z][A-Z0-9_]*"' <scope>
           (catches: ErrCodeFoo = "FOO_BAR", ErrFoo = "FOO", etc.)
        2. Extract the quoted string from each hit (e.g., "INVALID_YEAR_RANGE").
        3. For each string, check: does it start with one of the known prefixes
           above followed by `_`?
        4. If NO → VIOLATION. Flag as HIGH severity, tagged "auto-fixable".
        5. If string starts with an UNKNOWN prefix (e.g., a new subsystem prefix
           not yet in Rule 7) → escalate to user: "prefix '{X}_' not in Rule 7.
           Extend Rule 7 and re-run, or rename to use an existing prefix?"

      Auto-fix procedure (when user picks [1] in Step 4):
        1. Determine correct prefix from file path:
           - apps/api/internal/tmdb/** → TMDB_
           - apps/api/internal/subtitle/** → SUBTITLE_
           - apps/api/internal/scanner/** → SCANNER_
           - apps/api/internal/sse/** → SSE_
           - apps/api/internal/plugins/** → PLUGIN_
           - apps/api/internal/ai/** → AI_
           - apps/api/internal/repository/** (DB errors) → DB_
           - apps/api/internal/handlers/** or services/** (input validation) → VALIDATION_
           - apps/api/internal/library/** or services/library_service.go → LIBRARY_
           - Other → ask user
        2. Rename the string literal: "INVALID_YEAR_RANGE" → "TMDB_INVALID_YEAR_RANGE".
        3. Grep the repo for the OLD string (literal, case-sensitive) and update
           ALL call sites: test assertions (_test.go), Swagger annotations
           (@Failure 400 responses in handler comments), any hard-coded
           response-body matchers in apps/web/tests/ (unlikely but possible).
        4. Update the story's File List with every touched file.
        5. Add Completion Notes entry: "🔒 Rule 7 Wire Format Fix: {OLD} → {NEW}
           (N call sites updated)"

      Pass recording (when no violations found):
        Record in the review findings summary BEFORE Step 4 categorization:
        "🔒 Rule 7 Wire Format: PASS (N error codes checked, all prefixed
        correctly with known sources)"
        This is MANDATORY — silence is not an option.

      Precedent (Epic 10, Story 10-1a H1):
        `ErrCodeInvalidYearRange = "INVALID_YEAR_RANGE"` shipped without the
        mandatory `TMDB_` prefix even though the code lives in tmdb/errors.go.
        Caught by adversarial human CR, but at the cost of a wire-contract
        re-commit: the test `TestNewInvalidYearRangeError` and response-body
        expectation had to be resynced. This check exists to surface that class
        of violation systematically in Step 3, not accidentally during per-file
        review.
    </action>
    ```
  - [ ] 1.3 Verify the inserted action does not break Step 3's existing sub-structure. Step 3 currently contains 5 named audits (git/story, AC, task, story-file-integrity, code-quality-deep-dive) plus a retry gate (`total_issues_found lt 3`). The new check inserts as a 6th audit between "story file integrity" and "code quality deep dive". Confirm all `<action>` and `<check>` tags remain balanced by reading lines 59–117 end-to-end after edit.
  - [ ] 1.4 Cross-reference the new action's text from the step-level critical block if useful. Specifically, do NOT add a new `<critical>` at the top of Step 3 — the existing `<critical>VALIDATE EVERY CLAIM - Check git reality vs story claims</critical>` already covers the meta-intent. Adding a second critical would be noise.
  - [ ] 1.5 Update the findings summary output block in Step 4 (currently lines 124–147) to include the wire-format line under the review summary header:
    ```
    **Story:** {{story_file}}
    **Git vs Story Discrepancies:** {{git_discrepancy_count}} found
    **🔒 Rule 7 Wire Format:** {{wire_format_check_result}}  <!-- new line -->
    **Issues Found:** {{high_count}} High, {{medium_count}} Medium, {{low_count}} Low
    ```
    Where `{{wire_format_check_result}}` is one of: "PASS (N codes checked)", "FAIL (N violations — auto-fixable HIGH)", or "N/A (no Go error-code files in scope)".

- [ ] Task 2: Verify zero code regressions (AC: #6)
  - [ ] 2.1 Run `pnpm lint:all` from repo root — PASS expected (the edited file is `.xml` under `_bmad/`, excluded from Prettier and ESLint; Go vet/staticcheck unaffected)
  - [ ] 2.2 Run `pnpm nx test api` — PASS expected (zero Go code change)
  - [ ] 2.3 Run `pnpm nx test web` — PASS expected (zero frontend code change)
  - [ ] 2.4 Re-read the modified `instructions.xml` from Step 1 through Step 5 to confirm:
    - Step numbering (1–5) preserved
    - XML well-formed (no dangling tags, balanced `<check>` / `</check>`)
    - The new action sits AFTER `<!-- Story File Integrity Audit -->` and BEFORE `<!-- Code Quality Deep Dive -->` comments
    - The Step 4 `<output>` block still compiles (all `{{var}}` placeholders present)

- [ ] Task 3: Documentation discoverability — light touch (AC: #4)
  - [ ] 3.1 Do NOT add a new Rule to project-context.md. Rationale: Rule 7 already defines the contract; this story just enforces it at CR time. Adding a sibling "Rule 7-CR" would fragment the Rule. Precedent: retro-9-AI1 (Full Regression Gate) and retro-10-AI2 (AC Drift Check) both live in workflow XML only, not in project-context.md.
  - [ ] 3.2 Verify the Epic 10 retro document at `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md` already cites AI-3 Pattern #3 with enough detail. No edit needed — retro is frozen.
  - [ ] 3.3 Add a cross-reference line in the new action's header comment: `<!-- Last synced with project-context.md Rule 7 on 2026-04-20 -->`. When Rule 7 gains a new prefix (e.g., a future DVR_ for plugins), the reviewer who adds the prefix should also update this file and bump the date.

- [ ] Task 4: Update sprint-status.yaml (AC: #7)
  - [ ] 4.1 Mark `retro-10-AI3-rule7-wire-format-cr-check: ready-for-dev` at story creation time (handled by `/create-story` workflow — this step).
  - [ ] 4.2 Transitions during execution:
    - `/dev-story` start: `ready-for-dev → in-progress`
    - `/dev-story` finish: `in-progress → review`
    - `/code-review` pass: `review → done` with completion-note recording the final line range inside `instructions.xml` (e.g., "inserted Step 3 audit at lines 96–158")
  - [ ] 4.3 Preserve sprint-status.yaml comment structure and the Agreement 4 tracking rule.

## Dev Notes

### Root Cause

Epic 10 retro (2026-04-20) Pattern #3: Story 10-1a's follow-up work added `ErrCodeInvalidYearRange = "INVALID_YEAR_RANGE"` in `apps/api/internal/tmdb/errors.go`. The code-review on 10-1a caught this as a HIGH finding (H1) — but only via adversarial self-review. The fix required re-syncing:

- The Go constant itself (1 file)
- The unit test `TestNewInvalidYearRangeError` that asserted on the exact string
- Swagger annotations on both Discover handlers (400 response documentation)
- An additional helper `handleValidationError` whose log message referenced the error

Had CR workflow included an automatic grep for error-code strings missing a Rule 7 prefix, this violation would have surfaced in Step 3's contract audit rather than as a H1 finding in Step 4 — catching it at a cheaper point in the review. The pattern is: **Rule 7 is a wire contract, not a style rule**; it affects what the frontend sees in `ApiResponse<T>.error.code`. Violations are auto-fixable mechanically (grep + string-replace + call-site sync), which is exactly the work profile CR auto-fix (option `[1]` in Step 4) is designed for.

### Why code-review (not dev-story)?

An alternative placement is `dev-story/instructions.xml` — catch the violation at implementation time, before CR ever runs. That's tempting for defense-in-depth, but has two weaknesses:

1. **dev-story runs on the author**. The author just wrote the code; they're psychologically bound to the name they chose. A Rule 7 check at dev-story time would feel like style-nit fighting, not contract enforcement.
2. **CR is the adversarial gate**. Rule 7 violations ARE adversarial findings — the author can't see them; only a fresh reviewer (different LLM recommended per dev-story Step 11) will catch them. Placing the check in CR aligns with the gate's purpose.

Future extension: if this check catches violations consistently (say, >3 hits across epics), promote it to a Go boundaries-test under `apps/api/internal/` (mirroring retro-9-AI5's `TestServicesMustNotImportSubtitle`). That's explicitly OUT of scope for this story.

### Why MANDATORY + "record PASS even when no violations"?

Same rationale as retro-10-AI2 (AC Drift Check): **silent skip is the default failure mode of optional checks**. The retro-9-AI1 Full Regression Gate solved this by requiring every story's Completion Notes to cite the gate's execution. This story uses the identical pattern: **the CR agent MUST record a wire-format check result in the review findings summary, even if the result is "PASS" or "N/A"**. Benefits:

- Future retros can grep `🔒 Rule 7 Wire Format:` across review records to audit effectiveness
- User reviewing the CR findings knows the check actually ran
- Agent cannot claim "no Go files in scope" silently — they have to record "N/A (no Go error-code files in scope)" explicitly

### Why grep instead of a Go test?

A Go `TestErrorCodesUseKnownPrefixes` in `apps/api/internal/` would be more authoritative — it would run on every `go test` and block merges. But:

1. **Scope**: Such a test needs to traverse AST to find string-literal constants, filter to error-code patterns, and validate prefixes. It's roughly the same complexity as retro-9-AI5's `boundaries_test.go` — doable, but not a 15-minute change.
2. **Coverage**: A grep in CR catches the common case (`ErrCodeFoo = "FOO"`). A Go test would catch that plus dynamic construction (`fmt.Sprintf("%s_FOO", prefix)`), which is both more work and more brittle.
3. **Scope of THIS retro action**: AI-3's written success criterion is "code-review workflow grep for error code constants without source prefix; auto-fixable HIGH". The retro explicitly chose CR-workflow grep. A Go test is a separate, heavier intervention that belongs in a future retro if CR-grep proves insufficient.

Therefore: implement the CR-grep now; promote to Go test only if needed.

### Rule 7 source-prefix list (synced 2026-04-20)

From `project-context.md` lines 281–294 (Rule 7: Error Codes System):

```
TMDB_, AI_, DB_, VALIDATION_, SUBTITLE_, PLUGIN_, SCANNER_, SSE_, LIBRARY_
```

Example valid codes: `TMDB_TIMEOUT`, `AI_QUOTA_EXCEEDED`, `DB_NOT_FOUND`, `VALIDATION_REQUIRED_FIELD`, `SUBTITLE_DOWNLOAD_FAILED`, `PLUGIN_HEALTH_CHECK_FAILED`, `SCANNER_PERMISSION_DENIED`, `SSE_CONNECTION_FAILED`, `LIBRARY_DUPLICATE_PATH`.

Example invalid: `INVALID_YEAR_RANGE` (missing prefix — this was 10-1a H1), `INTERNAL_ERROR` (too generic), `NOT_FOUND` (no source).

### File placement (concrete)

Current Step 3 action sequence in `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml`:

```xml
<step n="3" goal="Execute adversarial review">
    <critical>VALIDATE EVERY CLAIM - Check git reality vs story claims</critical>

    <!-- Git vs Story Discrepancies -->            ← ~line 63
    <action>...</action>

    <!-- Use combined file list -->                 ← ~line 69
    <action>...</action>

    <!-- AC Validation -->                          ← ~line 72
    <action>...</action>

    <!-- Task Completion Audit -->                  ← ~line 80
    <action>...</action>

    <!-- Story File Integrity Audit -->             ← ~line 88
    <action>...</action>

    <!-- ⬇ INSERT NEW ACTION HERE -->

    <!-- Code Quality Deep Dive -->                 ← ~line 96
    <action>...</action>

    <check if="total_issues_found lt 3">            ← ~line 105
      ...
    </check>
  </step>
```

### Precedent patterns (Epic 8 / 9 / 9c / 10 retros)

Established pattern for SM workflow-improvement retros:

- **retro-8-P2** (story-splitting-rule): inline `<action>` in `create-story/instructions.xml` Step 5
- **retro-9-AI1** (full-regression-gate): inline `<action critical="MANDATORY">` in `dev-story/instructions.xml` Step 7
- **retro-9c-AI2** (fix-or-file): inline `<action critical="MANDATORY" if="...">` in `dev-story/instructions.xml` Step 7
- **retro-10-AI2** (ac-contract-drift-check): inline `<action critical="MANDATORY">` in `dev-story/instructions.xml` Step 2 (currently in review)

This story is the same pattern, applied to `code-review/instructions.xml` Step 3. Confidence: HIGH.

### Out of Scope

- Promoting the check to a Go test (`TestErrorCodesUseKnownPrefixes` under `apps/api/internal/`). Deferred — implement only if CR-grep proves insufficient.
- Frontend error-code checking (apps/web/ error handling). The wire contract flows Go → frontend; source of truth is Go. Frontend is consumer-only.
- Retroactively scanning all Epic 1–10 stories for latent Rule 7 violations. Not this story.
- Automatically suggesting a prefix from the file path (the auto-fix procedure in Task 1.2 section 2 hints at it, but the reviewer remains in the loop for ambiguous cases like `handlers/` which could be either VALIDATION_ or a domain-specific prefix).
- Adding Rule 7 check to any other workflow (dev-story, create-story, testarch-*). Only code-review this iteration.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#challenges`] Pattern #3 — Wire format Rule 7 violation (10-1a H1)
- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#action-items`] AI-3 row (SM, MEDIUM priority)
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] `retro-10-AI3-rule7-wire-format-cr-check: backlog` entry + Agreement 4 tracking rule
- [Source: `project-context.md#rule-7-error-codes-system`] The authoritative source prefix list (TMDB_, AI_, DB_, VALIDATION_, SUBTITLE_, PLUGIN_, SCANNER_, SSE_, LIBRARY_)
- [Source: `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml`] Step 3, target placement between "Story File Integrity Audit" and "Code Quality Deep Dive" blocks
- [Source: `apps/api/internal/tmdb/errors.go`] The 10-1a precedent: `ErrCodeInvalidYearRange = "TMDB_INVALID_YEAR_RANGE"` (post-fix); pre-fix value was `"INVALID_YEAR_RANGE"`
- [Precedent: `retro-9-AI1-full-regression-gate`] Same inline-MANDATORY-action pattern, different workflow (dev-story vs code-review)
- [Precedent: `retro-10-AI2-ac-contract-drift-check`] Same audit-always-record-result pattern

## Dev Agent Record

### Agent Model Used

(populated by dev-story workflow)

### Debug Log References

### Completion Notes List

### File List

(populated by dev-story workflow — expected single file: `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml`)

### Change Log

(populated by dev-story workflow)
