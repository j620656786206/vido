# Story: Document Manual Trigger Workflow

Status: done

## Story

As a developer,
I want a documented workflow for re-running TestSprite after deploys,
so that I can verify regressions efficiently.

## Acceptance Criteria

1. Workflow documented with step-by-step instructions
2. Includes steps for triggering test execution
3. Includes steps for comparing results against baseline
4. Includes steps for updating baseline when expected changes occur

## Dependencies

- ts-3 (baseline must exist to compare against)

## Tasks / Subtasks

- [x] Task 1: Write workflow documentation (AC: #1, #2, #3, #4)
  - [x] 1.1 Document pre-requisites (NAS accessible, TestSprite MCP available)
  - [x] 1.2 Document trigger steps (which MCP commands to run)
  - [x] 1.3 Document result comparison process (diff against baseline)
  - [x] 1.4 Document baseline update process (when to update, how to annotate)

- [x] Task 2: Add to project context (AC: #1)
  - [x] 2.1 Add TestSprite workflow reference to project-context.md
