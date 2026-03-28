# Story: Regenerate Test Plan and .py Files

Status: done

## Story

As a QA engineer,
I want fresh test cases generated from the actual app state,
so that tests reflect reality.

## Acceptance Criteria

1. New test plan JSON generated via TestSprite MCP
2. `.py` test files generated for all new TCs
3. Test plan covers all implemented features (Epics 1-8 scope)

## Dependencies

- ts-1 (config must point to NAS first)

## Tasks / Subtasks

- [x] Task 1: Bootstrap TestSprite (AC: #1)
  - [x] 1.1 Run `testsprite_bootstrap` to initialize fresh state

- [x] Task 2: Generate code summary (AC: #3)
  - [x] 2.1 Run `generate_code_summary` against current codebase

- [x] Task 3: Generate standardized PRD (AC: #3)
  - [x] 3.1 Run `generate_standardized_prd` from existing PRD artifacts

- [x] Task 4: Generate frontend test plan (AC: #1, #3)
  - [x] 4.1 Run `generate_frontend_test_plan` to produce test plan JSON
  - [x] 4.2 Review test plan for completeness against implemented features

- [x] Task 5: Generate executable test code (AC: #2)
  - [x] 5.1 Run `generate_code_and_execute` to produce `.py` files for all TCs
  - [x] 5.2 Verify all `.py` files are syntactically valid
