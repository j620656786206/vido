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

### File List
