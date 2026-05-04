---
title: 'CI Smoke Gate for nx serve api'
slug: 'ci-smoke-gate-nx-serve'
created: '2026-04-27'
status: 'ready-for-dev'
stepsCompleted: [1, 2, 3, 4]
tech_stack:
  - 'GitHub Actions (ubuntu-latest runner)'
  - 'pnpm 9 via pnpm/action-setup@v4'
  - 'Node via actions/setup-node@v4 (.nvmrc + cache: pnpm)'
  - 'Go 1.25 via actions/setup-go@v5 (cache-dependency-path: apps/api/go.sum)'
  - 'nx task runner (pnpm exec nx serve api → GIN_MODE=debug go run ./cmd/api, cwd: apps/api)'
  - 'SQLite (auto-created under VIDO_DATA_DIR; migrations applied on startup)'
  - 'curl (pre-installed on ubuntu-latest)'
  - 'bash (for-loop retry + trap-based cleanup)'
files_to_modify:
  - '.github/workflows/test.yml (single file; net-new job `serve-smoke`)'
code_patterns:
  - 'CI job with `needs: [lint, test-unit]` (parallel to `build`, fails earliest)'
  - 'Background process pattern: `pnpm exec nx serve api &` + PID capture + `trap EXIT` cleanup'
  - 'Bash retry loop `for i in {1..30}; do curl ... && break; sleep 1; done` (mirrors test.yml:259-265 E2E pattern verbatim)'
  - '`mkdir -p ${{ github.workspace }}/vido-data` before backend start (matches E2E job line 253)'
  - 'Job timeout: `timeout-minutes: 5` (server start ≤30s + curl loop ≤30s + cleanup buffer)'
test_patterns:
  - 'HTTP 200 assertion on `GET /api/v1/explore-blocks` via curl exit-code'
  - 'Fail-fast non-zero exit on smoke failure (timeout or non-200)'
  - 'Server-side: explore-blocks handler has zero TMDB dependency → no TMDB_API_KEY secret needed'
  - 'Regression-test-the-gate strategy: post-merge, push a branch with deliberately-broken sibling .go file → expect serve-smoke to FAIL'
---

# Tech-Spec: CI Smoke Gate for nx serve api

**Created:** 2026-04-27
**Sprint-Status Entry:** `retro-10-AI6-ci-smoke-gate-nx-serve` (priority MEDIUM, backlog → ready-for-dev via this spec)
**Origin:** Discovered 2026-04-20 during retro-10-CP1 prep. The `nx serve api` target was silently broken from 2026-03-26 (commit `9d0590c feat: unified single Docker image for NAS deployment` added `apps/api/cmd/api/static.go` into `package main`) until 2026-04-20 (commit `2287563 fix(api): nx serve uses package path so static.go compiles` switched the nx target from `go run ./cmd/api/main.go` single-file mode to `go run ./cmd/api` package mode). Neither `go test ./...` nor `go build ./...` caught the regression — both compile the whole package. The bug was only visible at dev runtime, and no CI gate exercised dev runtime.

## Overview

### Problem Statement

The `nx serve api` target can ship broken to `main` without any existing CI gate catching it. This has already happened once (25-day silent window, 2026-03-26 → 2026-04-20). The class-of-bug is: **a new sibling `.go` file is added to `apps/api/cmd/api/` and referenced by `main.go`, but the nx target invocation mode (single-file vs package) skips it at compile time.** Adjacent classes this gate also catches: broken migrations on startup, unregistered route regressions, broken DB initialization, nx target config rot, and `main.go` wiring gaps (Rule 15 adjacent).

### Solution

Add a new CI job `serve-smoke` to `.github/workflows/test.yml` with `needs: [lint, test-unit]` that:

1. Starts `pnpm exec nx serve api` as a background process (PID captured)
2. Polls `curl http://localhost:8080/api/v1/explore-blocks` in a retry loop (fixed-backoff, Playwright globalSetup parity)
3. Asserts HTTP 200 within a bounded timeout
4. Tears down the background process (kill PID, cleanup via `trap`)
5. Exits non-zero on any step failure → CI fails

The smoke target `/api/v1/explore-blocks` was chosen deliberately: it exercises the full startup path (SQLite DB + migrations + handler registration + routing) — the exact pathway a sibling-file compile skip could break.

### Scope

**In Scope:**

- New job `serve-smoke` in `.github/workflows/test.yml`
  - `needs: [lint, test-unit]` (fails early, independent of frontend `build`)
  - Steps: checkout, setup Go, setup Node + pnpm, install deps, start nx serve in background, retry-curl, assert 200, cleanup
- Background process lifecycle with `trap` for guaranteed cleanup (success or failure path)
- Fixed-backoff retry loop (~30 iterations × 1s) matching Playwright globalSetup pattern documented in `project-context.md` §Test Process Cleanup
- Fail-fast behavior: any non-200 response or timeout → non-zero exit
- Trust migrations to auto-run on `nx serve` startup (no explicit DB bootstrap in CI; default SQLite path under CI runner temp dir)

**Out of Scope:**

- Additional smoke targets (auth, TMDb, other endpoints) — this gate is specifically for file-mode-regression class + startup-breaking regressions
- Reconfiguring the `nx serve` target itself (already fixed in commit 2287563)
- Playwright E2E test orchestration (separate `test-e2e-sharded` job, unaffected)
- Docker image smoke (separate `docker.yml` workflow, out of scope)
- Runtime monitoring / APM / uptime SLOs
- Frontend `nx serve web` smoke (out of scope; only backend nx target regressions are in the class-of-bug)

## Context for Development

### Codebase Patterns

- **CI workflow style** (`.github/workflows/test.yml`): existing 5 jobs follow `lint → test-unit → build → test-e2e-sharded → merge-test-results` chain. Each job uses: `runs-on: ubuntu-latest`, `timeout-minutes: <bounded>`, step-scoped `name:` fields, cached `pnpm/action-setup@v4` (v9 pin) + `actions/setup-node@v4` (with `.nvmrc` + `cache: 'pnpm'`) + `actions/setup-go@v5` (with `cache-dependency-path: apps/api/go.sum`).
- **Existing curl retry-loop precedent** (`test.yml:259-265` in `test-e2e-sharded` job, "Start Go backend" step):
  ```yaml
  - name: Start Go backend
    run: |
      ./bin/api &
      echo "Waiting for backend to be ready..."
      for i in {1..30}; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
          echo "Backend is ready!"
          break
        fi
        sleep 1
      done
    env:
      VIDO_DATA_DIR: ${{ github.workspace }}/vido-data
      VIDO_PORT: 8080
      TMDB_API_KEY: ${{ secrets.TMDB_API_KEY }}
  ```
  This is the canonical pattern. `serve-smoke` will mirror this, with two critical deviations: (a) replace `./bin/api &` with `pnpm exec nx serve api &` to exercise `go run` package-mode invocation (the bug-class-specific path), (b) probe `/api/v1/explore-blocks` instead of `/health` for broader gate coverage.
- **Built binary vs nx serve invocation gap** (the bug class, restated):
  - `go build -o bin/api ./cmd/api` (used in `build` job): always compiles the whole `cmd/api` package → all sibling files included → ALWAYS catches compile errors.
  - `go run ./cmd/api` (used in nx serve, post-2287563): same package-mode → all sibling files included.
  - `go run ./cmd/api/main.go` (single-file mode, REGRESSED state from 2026-03-26→2026-04-20): only `main.go` compiled → sibling files like `static.go` SKIPPED → at runtime, references to skipped-file symbols cause compile error visible only when `nx serve` is run.
  - **Therefore**: only `pnpm exec nx serve api` exercises the regression-prone path. `bin/api`-based smokes (existing E2E) cannot detect this class.
- **`/health` smoking gun cross-reference** (`apps/api/cmd/api/static.go:94`): `static.go` itself contains `if path == "/health"` logic. Single-file mode skipping `static.go` would either (a) prevent server compile entirely, or (b) silently change `/health` behavior. The user's chosen `/api/v1/explore-blocks` smoke target catches case (a) — server-won't-start — equally well, while also exercising migrations + handler registration + DB query path.
- **Migration auto-run on startup** (`apps/api/cmd/api/main.go:74-108`): `nx serve api` on a clean checkout will create SQLite DB file under `VIDO_DATA_DIR` and apply all registered migrations before the server binds the port. No explicit CI bootstrap step needed.
- **Route registration target** (`apps/api/cmd/api/main.go:536`): `exploreBlocksHandler.RegisterRoutes(apiV1)` exposes `GET /api/v1/explore-blocks` (Story 10.3). Handler at `apps/api/internal/handlers/explore_blocks_handler.go` has **zero TMDB dependency** (verified by grep) → CI smoke does NOT need `TMDB_API_KEY` secret.
- **Background-process cleanup pattern**: Playwright `globalSetup`/`globalTeardown` (documented in `project-context.md` §Test Process Cleanup) is the in-house reference for tracking spawned servers + PID-based cleanup. For a single CI step, simpler `bash trap` on EXIT/ERR is sufficient and idiomatic.

### Files to Reference

| File | Purpose |
| ---- | ------- |
| `.github/workflows/test.yml` | The file being modified; new `serve-smoke` job will be added between `test-unit` and `build` (parallel to `build`, both depending on `[lint, test-unit]`) |
| `.github/workflows/test.yml:255-282` | Canonical curl retry-loop pattern (E2E job's "Start Go backend" step) — to be mirrored verbatim with the two deviations noted above |
| `apps/api/project.json:16-22` | nx `serve` target ground truth: `GIN_MODE=debug go run ./cmd/api`, `cwd: "apps/api"` |
| `apps/api/cmd/api/main.go:74-108` | Migration auto-run sequence on startup |
| `apps/api/cmd/api/main.go:510` | `/health` route registration (NOT chosen as smoke target) |
| `apps/api/cmd/api/main.go:536` | `/api/v1/explore-blocks` route registration (CHOSEN smoke target — Story 10.3) |
| `apps/api/cmd/api/static.go` | The sibling file that surfaced the class-of-bug (commit `9d0590c`); `static.go:94` references `/health` |
| `apps/api/internal/handlers/explore_blocks_handler.go` | Smoke target handler — zero TMDB deps confirmed via grep |
| `project-context.md` Rule 15 (HTTP Route ↔ Client Method Sync) | Adjacent regression class this gate also catches |
| `project-context.md` §Test Process Cleanup | Background-process lifecycle pattern (Playwright globalSetup parity) |

### Technical Decisions

Confirmed with user during Step 1, refined by Step 2 investigation:

1. **Job placement (Q1=b)** ✓: New independent job `serve-smoke` with `needs: [lint, test-unit]`. Runs parallel to `build`. Fails earliest when applicable, saves CI minutes vs. waiting for frontend build.
2. **Server-ready detection (Q2=b)** ✓: Bash `for i in {1..30}; do curl ... break; sleep 1; done` retry loop. Mirrors `test.yml:259-265` E2E job pattern verbatim — zero new dependencies, idiomatic to this codebase.
3. **DB bootstrap (Q3=a)** ✓ (validated by Step 2 main.go:74-108 read): Trust auto-migrations on startup. CI step `mkdir -p ${{ github.workspace }}/vido-data` creates the SQLite parent directory (matches E2E job line 253). Migrations apply idempotently before port bind.
4. **Smoke command (Step 2 derived, NOT confirmed in Step 1)**: MUST use `pnpm exec nx serve api &`, NOT `./bin/api &`. Built binary uses `go build` (package mode), which would never catch the regression-prone `go run main.go` single-file mode. Only `nx serve` invocation exercises the bug-class path.
5. **Smoke target endpoint (Step 2 derived, validates user's backlog choice)**: `GET /api/v1/explore-blocks` — broader gate than `/health` (exercises migrations + handler registration + DB query). `/health` is already covered by E2E job; this gate's value-add is the broader API-stack probe via `go run` invocation. Handler has zero external secret dependencies (no TMDB key needed).
6. **No `TMDB_API_KEY` secret needed (Step 2 derived)**: explore-blocks handler is self-contained DB query. Reduces CI secret surface area; `serve-smoke` job can skip the `TMDB_API_KEY` env line that E2E uses.
7. **Cleanup pattern (Step 2 derived)**: `trap "kill $SERVER_PID 2>/dev/null || true" EXIT` on the run-block. Guarantees PID kill on success, failure, or timeout exit. Idiomatic bash, no extra tool.
8. **Job timeout (Step 2 derived)**: `timeout-minutes: 5`. Worst-case: 60s server-start + 60s curl loop + cleanup buffer = under 5min cap. Diverges from `lint`'s 10min cap downward (smoke is faster).
9. **Retry iterations (Party Mode — Murat)**: **60 iterations × 1s sleep**, not 30. Rationale: GitHub Actions `ubuntu-latest` runners observed to IO-stall 5-8s during cold-start migrations + slog setup; 30 may be too tight for p99. 30 extra seconds only spent on failure path.
10. **Failure-path log dump (Party Mode — Murat + Amelia)**: Background server stdout+stderr captured to `/tmp/nx-serve.log`. On smoke timeout or non-200, dump `tail -100 /tmp/nx-serve.log` to job log so panic stacks / migration errors are visible to the PR author. Without this, smoke failures are debugging hell.
11. **Strict HTTP 200 check (Party Mode — Amelia)**: Use `curl -s -o /dev/null -w "%{http_code}"` and string-compare to `200`, NOT `curl -s ... > /dev/null && break` exit-code check. Rationale: AC explicitly states "expects 200" — a 500/404 would also produce a non-zero curl exit but should still fail the gate (server up but route broken).
12. **Defense-in-depth framing (Party Mode — Winston)**: This gate is the **runtime/post-commit** layer of three. Layer 1: pre-commit checklist (Rule 15 in `project-context.md`). Layer 2: build-time (`go build`/`go test` already passing). Layer 3 (this gate): runtime smoke catches what 1-2 miss. The three layers do NOT replace each other.
13. **Adjacent lint-level reinforcement (Party Mode — Winston, OUT OF SCOPE)**: A static check `grep "go run ./cmd/api[^/]" apps/api/project.json` could catch `serve` target single-file regression at lint time, earlier than this runtime gate. Documented as adjacent future enhancement; deliberately NOT bundled into this spec to keep scope tight.

## Implementation Plan

### Tasks

Single-PR scope: Tasks 1–3. Post-merge follow-up: Tasks 4–5 (gate-the-gate validation + Change Log update).

- [ ] **Task 1: Add `serve-smoke` job to `.github/workflows/test.yml`**
  - File: `.github/workflows/test.yml`
  - Action: Insert new job after `test-unit` (line ~158), before `build` (line ~163). Use this exact skeleton (party-mode-validated):
    ```yaml
    serve-smoke:
      name: Serve Smoke Gate (nx serve api)
      needs: [lint, test-unit]
      runs-on: ubuntu-latest
      timeout-minutes: 5
      steps:
        - name: Checkout code
          uses: actions/checkout@v4
        - name: Install pnpm
          uses: pnpm/action-setup@v4
          with:
            version: 9
        - name: Setup Node.js
          uses: actions/setup-node@v4
          with:
            node-version-file: '.nvmrc'
            cache: 'pnpm'
        - name: Install Node dependencies
          run: pnpm install --frozen-lockfile
        - name: Setup Go
          uses: actions/setup-go@v5
          with:
            go-version: ${{ env.GO_VERSION }}
            cache-dependency-path: apps/api/go.sum
        - name: Download Go dependencies
          working-directory: apps/api
          run: go mod download
        - name: Create data directory
          run: mkdir -p ${{ github.workspace }}/vido-data
        - name: Start nx serve api (background) and probe smoke endpoint
          run: |
            pnpm exec nx serve api > /tmp/nx-serve.log 2>&1 &
            SERVER_PID=$!
            trap "kill $SERVER_PID 2>/dev/null || true" EXIT

            echo "Started nx serve api (PID=$SERVER_PID); probing /api/v1/explore-blocks..."
            for i in {1..60}; do
              code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/explore-blocks || echo "000")
              if [ "$code" = "200" ]; then
                echo "✓ Smoke passed on iteration $i (HTTP $code)"
                exit 0
              fi
              sleep 1
            done

            echo "::error::Smoke timeout after 60s. Last HTTP code: $code"
            echo "===== nx serve api log (tail -100) ====="
            tail -100 /tmp/nx-serve.log || echo "(no log captured)"
            exit 1
          env:
            VIDO_DATA_DIR: ${{ github.workspace }}/vido-data
            VIDO_PORT: 8080
    ```
  - Notes:
    - `needs: [lint, test-unit]` runs parallel to `build`, fails earliest. Validated by party-mode (Bob, Murat).
    - `pnpm exec nx serve api` (NOT `./bin/api`) — exercises `go run ./cmd/api` package-mode invocation, which is the regression-prone path. Built binary always uses `go build` and would never catch the bug class.
    - `trap "kill ..." EXIT` is set IMMEDIATELY after `$!` capture to prevent PID-leak race (Amelia's catch).
    - Strict `[ "$code" = "200" ]` check, NOT exit-code-based — catches 500/404 server-up-but-broken (Amelia's catch).
    - `tail -100 /tmp/nx-serve.log` dump on timeout exposes panic stacks / migration errors to PR author (Murat's catch).
    - `TMDB_API_KEY` deliberately omitted — explore-blocks handler verified TMDb-free (Step 2 grep).
    - 60 iterations × 1s sleep, not 30 — accommodates `ubuntu-latest` IO stalls during cold-start migrations (Murat's tightening).

- [ ] **Task 2: Local pre-push validation**
  - File: (none — local sanity check)
  - Action: Run the equivalent commands locally to catch any bash/YAML typos before pushing:
    ```bash
    rm -rf vido-data && mkdir -p vido-data
    VIDO_DATA_DIR=$(pwd)/vido-data VIDO_PORT=8080 pnpm exec nx serve api > /tmp/nx-serve.log 2>&1 &
    SERVER_PID=$!
    sleep 5
    curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/v1/explore-blocks  # expect 200
    kill $SERVER_PID
    ```
    Confirm output is `200`. If non-200, abort and debug locally before pushing.
  - Notes: This catches typos / env var mismatches without burning a CI run.

- [ ] **Task 3: Push branch + verify happy-path CI run**
  - File: (none — observation step)
  - Action: Push the branch with the new job, open PR, observe `serve-smoke` step runs green within 5 minutes.
  - Notes: This is the AC1 acceptance evidence. Capture the CI run URL for Change Log.

- [ ] **Task 4 (post-merge): Gate-the-gate sabotage validation** ← MUST execute, see Testing Strategy
  - File: temporary branch only
  - Action: After Task 1's PR merges, create branch `chore/verify-serve-smoke-gate-fails-as-expected`, add `apps/api/cmd/api/_smoke_validator.go` with `package main\n\nfunc init() { undefinedFunction() }`, push, observe `serve-smoke` FAILS with the compile error visible in the log dump. Delete branch without merging.
  - Notes: Without this, the gate is theatre. Murat's Party Mode insistence — PROVE the gate has teeth.

- [ ] **Task 5 (post-merge): Document validation result**
  - Files: `_bmad-output/implementation-artifacts/sprint-status.yaml` (entry `retro-10-AI6-ci-smoke-gate-nx-serve` → done), this spec's Change Log
  - Action: Append validation evidence — happy-path CI run URL (Task 3), sabotage CI run URL (Task 4 expected-FAIL), commit SHA of the merged Task 1 PR.
  - Notes: Closes audit trail; matches followup-* story precedent format.

### Acceptance Criteria

- [ ] **AC1 (happy path)**: Given a PR that introduces no compile errors and does not break startup, when CI runs, then `serve-smoke` job exits 0 with the line `✓ Smoke passed on iteration N (HTTP 200)` in the job log within 5 minutes wall-time.

- [ ] **AC2 (class-of-bug — sibling-file compile)**: Given a `.go` file added to `apps/api/cmd/api/` declaring `package main` (with or without `main.go` referencing its symbols), when the file contains a compile error, then `serve-smoke` MUST exit non-zero AND the compile error MUST be visible in the `tail -100 /tmp/nx-serve.log` dump in the job log. (Validated by Task 4 sabotage branch.)

- [ ] **AC3 (cleanup)**: Given any failure path — timeout, non-200, server panic, compile error — when the job exits, then no `nx serve` / `go run` background process is leaked; the `trap "kill $SERVER_PID 2>/dev/null || true" EXIT` MUST fire on every exit path. (Verified by GitHub Actions runner ephemerality + the explicit trap pattern.)

- [ ] **AC4 (performance — measurement-based)**: Given nominal CI conditions, when `serve-smoke` runs across the first 10 successful runs post-merge, then p95 wall-time MUST be < 90 seconds. If p95 > 120 seconds, investigate and tune (e.g., reduce iteration count, profile migration startup). Initial p50 expected ≈ 30-45 seconds.

- [ ] **AC5 (scope independence)**: Given a `serve-smoke` failure, when the workflow proceeds, then `test-e2e-sharded` and `merge-test-results` MUST still run (they depend on `build`, not `serve-smoke`). The new gate is additive, NOT on the E2E critical path.

- [ ] **AC6 (dev parity)**: Given a clean dev checkout, when a developer runs `mkdir -p vido-data && VIDO_DATA_DIR=$(pwd)/vido-data pnpm exec nx serve api &; sleep 5; curl http://localhost:8080/api/v1/explore-blocks`, then the response is HTTP 200 with valid JSON body. (Task 2's local validation step.)

- [ ] **AC7 (HTTP code strictness)**: Given `nx serve api` running but `/api/v1/explore-blocks` returning a non-200 (e.g., 500 from broken handler logic, 404 from missing route), when `serve-smoke` probes, then MUST exit non-zero. The `[ "$code" = "200" ]` string check (NOT curl exit-code-only) MUST be used.

- [ ] **AC8 (gate-the-gate — proof of teeth)**: Given the merge of Task 1, when Task 4's sabotage branch is pushed, then `serve-smoke` MUST fail with the compile error visible in the log dump. The CI run URL of the sabotage failure MUST be recorded in this spec's Change Log + sprint-status.yaml entry. Without this evidence, AC2 is unverified theatre.

## Additional Context

### Dependencies

**GitHub Actions** (all pinned, all already used by other jobs in this workflow):
- `actions/checkout@v4`
- `pnpm/action-setup@v4` with `version: 9`
- `actions/setup-node@v4` with `node-version-file: '.nvmrc'`, `cache: 'pnpm'`
- `actions/setup-go@v5` with `go-version: ${{ env.GO_VERSION }}` (1.25), `cache-dependency-path: apps/api/go.sum`

**Pre-installed on `ubuntu-latest` runner**:
- `curl` (system binary)
- `bash` (default shell for `run:` blocks)

**Existing nx target** (no changes required to nx config):
- `pnpm exec nx serve api` → `apps/api/project.json:16-22` `serve` target → `GIN_MODE=debug go run ./cmd/api` from `apps/api/` cwd, post-commit `2287563` package-mode fix.

**Repo state assumptions**:
- `.nvmrc` exists and pins a Node version compatible with pnpm 9 + nx (already verified — used by all existing CI jobs).
- `apps/api/go.sum` exists and is committed (already verified).
- No GitHub Actions `secrets` required (`TMDB_API_KEY` deliberately omitted; explore-blocks handler is TMDb-free).

**Adjacent (NOT a dependency, but a related potential follow-up — Winston's Party Mode insight)**:
- Lint-level static check `grep "go run ./cmd/api[^/]" apps/api/project.json && exit 1` could catch `serve` target single-file regression earlier than this runtime gate. Out of scope for this spec; if the team wants it later, file a new follow-up story.

### Testing Strategy

- **Primary test (acceptance)**: The CI job itself is the test. Push a PR → `serve-smoke` runs → green = pass.
- **Gate-the-gate regression test (Party Mode — Murat, MUST be executed)**: After merging the new job, perform a deliberate sabotage validation:
  1. Create branch `chore/verify-serve-smoke-gate-fails-as-expected`
  2. Add a sibling file `apps/api/cmd/api/_smoke_validator.go` with intentional compile error: `package main\n\nfunc init() { undefinedFunction() }`
  3. Push branch → confirm `serve-smoke` job FAILS with the compile error visible in `nx-serve.log` dump
  4. Delete branch without merging
  5. Document the validation result (commit SHA + CI run URL) in this spec's Change Log
  
  This proves the gate has teeth. Without this validation, the gate is theatre.
- **Local dev parity**: Document in CLAUDE.md (or this spec's followup story note) that running `pnpm exec nx serve api &` locally and `curl http://localhost:8080/api/v1/explore-blocks` mirrors what CI does. Use this as the local pre-PR sanity check.
- **No Go/Vitest unit tests required**: this is a CI config change, not a code change. Playwright E2E is not the right level (too heavyweight + already covers the built-binary path via `test-e2e-sharded` job).

### Notes

- **25-day silent window**: 2026-03-26 (9d0590c, static.go added) → 2026-04-20 (2287563, nx target fixed). During this window, anyone running `nx serve api` on their machine would hit the compile error. CI never surfaced it because no job used `nx serve`.
- **Why `/api/v1/explore-blocks` and not `/api/v1/health/services`**: The health endpoint is comprehensive but may defer to plugin probes and have its own failure modes. `/api/v1/explore-blocks` is a simple GET returning JSON from DB — exercises migrations + routing + handler wiring but not plugin infrastructure. Closer to a "the server is alive and can talk to DB" smoke signal.
- **Port 8080 conflict**: GitHub Actions `ubuntu-latest` runners are ephemeral per-job; no port conflict expected (verified in Step 2).
- **This gate does NOT replace Rule 15**: Rule 15 (HTTP Route ↔ Client Method Sync) is a pre-commit checklist. This CI gate is a post-commit safety net for the subset of Rule 15 issues that manifest as server-won't-start failures.
- **Adjacent regressions this gate also catches** (Party Mode — Murat's broader risk frame):
  - Migration breaks startup (e.g., new migration with syntax error)
  - Plugin Manager init failure (qBittorrent / Sonarr / Radarr init panic at startup)
  - Route registration regression (e.g., handler exists but `RegisterRoutes` not called from `main.go` — Rule 15 missed)
  - Static asset embed broken (`static.go` itself or its asset path)
  - Database open failure (e.g., schema corruption from migration)
  
  This widens the gate's value-add beyond just the sibling-file class.
- **Defense-in-depth (Party Mode — Winston)**: This gate composes with existing layers:
  - **Layer 1 (pre-commit)**: `project-context.md` Rule 15 checklist — manual, human-in-the-loop, catches 70% of issues before commit.
  - **Layer 2 (build-time)**: `go vet` + `staticcheck` + `go build ./...` + `go test ./...` — already in `lint` and `test-unit` jobs, catches compile errors that affect package mode.
  - **Layer 3 (this gate, runtime)**: catches what L1+L2 miss — invocation-mode-specific compile errors, migration runtime failures, plugin runtime panics. Adds ~60s p95 to CI per PR.
- **Sequencing decision (Party Mode 2026-04-27)**: Three priority experts (PM John, SM Bob, TEA Murat) reached 3-0 consensus to ship this gate **now** rather than defer to a later sprint. Rationale: (a) Step 1+2 sunk cost still in working memory makes resume-later more expensive; (b) cmd/api/ has 0 commits in last 4 weeks → low expected-value-per-defer; (c) quality gates accumulate entropy when deferred. User confirmed.
