# Epic 9-T: TestSprite Journey Test Integration

## Goal
Integrate TestSprite journey-level tests against the deployed NAS app (http://192.168.50.52:8088) to fill the gap between Unit tests and Playwright E2E tests.

## Context
- TestSprite MCP already installed (Free plan, 150 credits/month)
- NAS has running app with real media data
- Existing 50 TCs are outdated — will regenerate from scratch
- Runs parallel to bugfix sprint (separate worktree)

## Stories
- ts-1: Update TestSprite config for NAS target
- ts-2: Regenerate test plan and .py files
- ts-3: Establish baseline
- ts-4: Document manual trigger workflow
- ts-5: Update project documentation

## Dependencies
- Requires NAS app to be accessible at http://192.168.50.52:8088
- No dependency on bugfix sprint completion (runs in parallel)

## Acceptance Criteria
1. TestSprite config points to NAS URL with production mode
2. New test plan generated based on actual app state
3. Baseline results recorded with known-bug annotations
4. Manual trigger workflow documented
