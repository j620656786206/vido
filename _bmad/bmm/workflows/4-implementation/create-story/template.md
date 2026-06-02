# Story {{epic_num}}.{{story_num}}: {{story_title}}

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a {{role}},
I want {{action}},
so that {{benefit}}.

## Acceptance Criteria

1. [Add acceptance criteria from epics/PRD]

## Tasks / Subtasks

- [ ] Task 1 (AC: #)
  - [ ] Subtask 1.1
- [ ] Task 2 (AC: #)
  - [ ] Subtask 2.1

## Dev Notes

- Relevant architecture patterns and constraints
- Source tree components to touch
- Testing standards summary

### Project Structure Notes

- Alignment with unified project structure (paths, modules, naming)
- Detected conflicts or variances (with rationale)

### Time-dependent visual coverage

<!-- Added by story 19-9 (Rule 23 — project-context.md). Forward-only: applies to stories
     CREATED after 19-9 closes; in-flight stories are NOT retrofitted.
     Delete this sub-section ONLY if you are POSITIVE every component the story touches
     is wall-clock-independent. -->

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - If **YES**: list the fixture state baseline paths the dev MUST capture (≥2 per Rule 23 AC #1d — typically `recent` + `stale`, but component-specific state names are allowed when branching is naturally named differently, e.g. `overdue` / `on-time`). Reference the AC #4 helper `withFixedClock(page, iso)` and the `clockTime` fixture-row field. Pair the component with one of the three accepted Rule 23 marker forms (`Clock-mocked` / `Clock-injected` / `Time-bomb-exempt`).
  - If **NO**: explicitly state `N/A — no wall-clock-reading components touched`.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`; canonical migration precedent: story 19-9 AC #5 (`library-recently-added` → `recent` + `stale`).

### References

- Cite all technical details with source paths and sections, e.g. [Source: docs/<file>.md#Section]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

<!-- Added by retro-19-P1 (Rule 24 — project-context.md, Discovery Triage). Forward-only:
     applies to stories CREATED after retro-19-P1 closes; in-flight stories are NOT retrofitted.
     RULE 24 BAN — "mentioned in prose but not in sprint-status": any out-of-scope finding
     surfaced during this story (here, in Dev Notes, a PR description, a TODO comment, or
     ANYWHERE in narrative) MUST be triaged into ONE lane below AND recorded with its tracked
     sprint-status.yaml entry ID (lane ② / ③) or absorbed-AC # (lane ①) BEFORE this story is
     marked done. A prose-only mention with no matching entry is a banned deferred-discovery
     time-bomb and bounces the story at review. -->

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - If **YES**: list each discovery on its own row, classified into EXACTLY ONE lane:
    - **① expand-scope-in-place** — absorbed into THIS story → cite the added AC # / sub-task that now tracks it (silently fixing it without an AC is NOT this lane — it is an untracked change).
    - **② spawn-blocking-story** — blocks this story's correct completion → cite the new `sprint-status.yaml` story ID; this story is marked `blocked` (or `blocked-by: {id}`) and does NOT close until ② resolves.
    - **③ backlog-with-carry-forward-link** — real but out-of-scope AND non-blocking → cite the `backlog` / `bugfix-N` `sprint-status.yaml` entry ID filed AT DISCOVERY TIME (bidirectional: the entry names this story; this row names the entry).
- Reference: `project-context.md` Rule 24; origin retro-19-P1 (Alexyu pain point B — the find→fix gap spanned a whole story: 19-7/19-8 → 19-9 → bugfix-19-9). Generalizes retro-8-P1 ("ALL retro action items become tracked entries") from end-of-epic retros to the moment of mid-story discovery.

### File List
