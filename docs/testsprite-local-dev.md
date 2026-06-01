<!-- Bilingual: English (this file) · 繁體中文 → testsprite-local-dev.zh-TW.md -->

# Running TestSprite Locally for Test Development

> **Status (2026-06-01):** The automated monthly gate
> (`.github/workflows/testsprite-monthly.yml`) is **deferred** — its `schedule:`
> cron is commented out. While the local journey-test suite is being grown, run
> TestSprite **locally** instead. The workflow file is retained and can still be
> triggered manually (`workflow_dispatch`); see its header `DEFERRED` note for
> the revival conditions. Carry-forward tracked as
> `retro-19-OPS1-wire-testsprite-secrets` in `sprint-status.yaml`.

## Why local-first right now

A manual `workflow_dispatch` run on 2026-06-01 proved the
`TESTSPRITE_API_KEY` secret works, but surfaced two blockers that make the
automated monthly cron net-negative today:

1. **Runner-local endpoint = environment noise.** With no stable
   `TESTSPRITE_TARGET_URL`, the workflow spins up a throwaway Vido on the runner
   (API :8080 + Vite :4200). 15 of 24 cases came back `error` (not `fail`) —
   i.e. cases that never produced a verdict, not real product drift. The run
   also took ~43 minutes.
2. **Commit-back blocked.** The workflow commits queue + generated `TC*.py`
   files back to `main`, but branch protection blocks `github-actions[bot]`
   → `git push failed after 3 attempts`.

Local development sidesteps both: you control the environment and run only the
test(s) you're working on.

## Prerequisites

- A TestSprite API key. The CI value mirrors
  `testsprite_tests/tmp/config.json` → `executionArgs.envs.API_KEY` (or
  re-issue from the TestSprite dashboard).
- The Vido app running locally (the target under test).

## Start the app under test

```bash
pnpm nx serve api      # API on :8080
pnpm nx serve web      # Vite dev server on :4200
```

## Option 1 — MCP server (recommended for interactive dev)

The repo already has the **TestSprite MCP server** configured. Inside an agent
session you can drive it directly to scope tests to a single page/journey:

- `testsprite_bootstrap` — initialize for the project
- `testsprite_generate_frontend_test_plan` — produce a plan
- `testsprite_generate_code_and_execute` — generate `.py` cases and run them

This is the fastest path for red-green-refactor on one journey at a time.

## Option 2 — CLI (reproduces the CI invocation)

```bash
# All settings (testIds, target URL, etc.) are read from
# testsprite_tests/tmp/config.json
API_KEY=<your-testsprite-key> \
  npx --yes "@testsprite/testsprite-mcp@0.0.37" generateCodeAndExecute
```

- The CLI reads `process.env.API_KEY` (or `TSMCP_API_KEY` as fallback) — **not**
  `TESTSPRITE_API_KEY`.
- Generated `.py` cases land in `testsprite_tests/`.
- To run a subset, narrow `executionArgs.testIds` in
  `testsprite_tests/tmp/config.json` — no need to run all 24 cases (~43 min)
  every iteration.

## Reviving the monthly gate later

Uncomment the two `schedule:` lines in `testsprite-monthly.yml` once:

1. the local test suite is stable;
2. a stable `TESTSPRITE_TARGET_URL` is set (replaces runner-local, cuts the
   error rate);
3. bot-push-to-`main` is solved (push a side branch / open a PR, or add
   `github-actions[bot]` to the branch-protection bypass list).

## See also

- Workflow: `.github/workflows/testsprite-monthly.yml`
- Story: `_bmad-output/implementation-artifacts/19-6-github-actions-testsprite-monthly.md`
- Queue: `_bmad-output/audit/testsprite-queue.yaml`
