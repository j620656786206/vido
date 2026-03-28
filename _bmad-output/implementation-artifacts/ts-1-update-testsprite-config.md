# Story: Update TestSprite Config for NAS Target

Status: ready-for-dev

## Story

As a QA engineer,
I want TestSprite config pointing to the NAS,
so that journey tests run against the deployed app.

## Acceptance Criteria

1. `localEndpoint` in config.json set to `http://192.168.50.52:8088`
2. `serverMode` set to `production`
3. Old `.py` test files (TC009, TC010, TC012 etc.) cleaned up
4. `test_results.json` cleared of stale data

## Tasks / Subtasks

- [ ] Task 1: Update config.json (AC: #1, #2)
  - [ ] 1.1 Set `localEndpoint` to `http://192.168.50.52:8088`
  - [ ] 1.2 Set `serverMode` to `production`
  - [ ] 1.3 Verify config is valid JSON

- [ ] Task 2: Delete old .py test files (AC: #3)
  - [ ] 2.1 Remove deprecated TC009/TC010/TC012 `.py` files
  - [ ] 2.2 Remove any other outdated `.py` test files from previous generations

- [ ] Task 3: Clear stale test results (AC: #4)
  - [ ] 3.1 Clear or reset `test_results.json`
  - [ ] 3.2 Verify clean state for fresh test generation
