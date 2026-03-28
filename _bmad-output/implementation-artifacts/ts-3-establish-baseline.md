# Story: Establish Baseline Test Results

Status: backlog

## Story

As a QA engineer,
I want a baseline test run,
so that future regressions can be detected.

## Acceptance Criteria

1. All TCs executed against NAS at http://192.168.50.52:8088
2. Results recorded in test_results.json
3. Failures categorized as: known-bug / tc-issue / new-bug

## Dependencies

- ts-2 (test plan and .py files must exist)

## Tasks / Subtasks

- [ ] Task 1: Execute all TCs (AC: #1)
  - [ ] 1.1 Run all generated `.py` test files against NAS endpoint
  - [ ] 1.2 Capture pass/fail results for each TC

- [ ] Task 2: Record results (AC: #2)
  - [ ] 2.1 Save execution results to test_results.json
  - [ ] 2.2 Record execution timestamp and environment info

- [ ] Task 3: Categorize failures (AC: #3)
  - [ ] 3.1 Review each failure and classify as known-bug, tc-issue, or new-bug
  - [ ] 3.2 Annotate known-bug failures with references to bugfix stories
  - [ ] 3.3 File tc-issue items for test plan correction

- [ ] Task 4: Create baseline report
  - [ ] 4.1 Summarize pass/fail/skip counts
  - [ ] 4.2 Document known-bug annotations for future comparison
