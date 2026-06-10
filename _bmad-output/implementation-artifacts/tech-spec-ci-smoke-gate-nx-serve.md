---
title: 'CI Smoke Gate for nx serve api'
slug: 'ci-smoke-gate-nx-serve'
created: '2026-04-27'
status: 'done'
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
7. **Cleanup pattern (Step 2 derived, CR-hardened 2026-06-10)**: `set -m` + `trap "kill -- -$SERVER_PID 2>/dev/null || kill $SERVER_PID 2>/dev/null || true" EXIT` on the run-block. `set -m` makes the backgrounded job a process-group leader so the negative-PID `kill` signals the ENTIRE tree (pnpm → nx → go run → compiled binary), not just the top pnpm PID. Original CR finding H1: a plain `kill $!` leaks the grandchild server because `go run` does not forward SIGTERM to its exec'd binary. (In CI the ephemeral runner reclaims everything regardless; the group-kill makes local/self-hosted runs correct too and makes AC3's guarantee real rather than incidental.)
8. **Job timeout (Step 2 derived)**: `timeout-minutes: 5`. Worst-case: 60s server-start + 60s curl loop + cleanup buffer = under 5min cap. Diverges from `lint`'s 10min cap downward (smoke is faster).
9. **Retry iterations (Party Mode — Murat)**: **60 iterations × 1s sleep**, not 30. Rationale: GitHub Actions `ubuntu-latest` runners observed to IO-stall 5-8s during cold-start migrations + slog setup; 30 may be too tight for p99. 30 extra seconds only spent on failure path.
10. **Failure-path log dump (Party Mode — Murat + Amelia)**: Background server stdout+stderr captured to `/tmp/nx-serve.log`. On smoke timeout or non-200, dump `tail -100 /tmp/nx-serve.log` to job log so panic stacks / migration errors are visible to the PR author. Without this, smoke failures are debugging hell.
11. **Strict HTTP 200 check (Party Mode — Amelia)**: Use `curl -s -o /dev/null -w "%{http_code}"` and string-compare to `200`, NOT `curl -s ... > /dev/null && break` exit-code check. Rationale: AC explicitly states "expects 200" — a 500/404 would also produce a non-zero curl exit but should still fail the gate (server up but route broken).
12. **Defense-in-depth framing (Party Mode — Winston)**: This gate is the **runtime/post-commit** layer of three. Layer 1: pre-commit checklist (Rule 15 in `project-context.md`). Layer 2: build-time (`go build`/`go test` already passing). Layer 3 (this gate): runtime smoke catches what 1-2 miss. The three layers do NOT replace each other.
13. **Adjacent lint-level reinforcement (Party Mode — Winston, OUT OF SCOPE)**: A static check `grep "go run ./cmd/api[^/]" apps/api/project.json` could catch `serve` target single-file regression at lint time, earlier than this runtime gate. Documented as adjacent future enhancement; deliberately NOT bundled into this spec to keep scope tight.

## Implementation Plan

### Tasks

Single-PR scope: Tasks 1–3. Post-merge follow-up: Tasks 4–5 (gate-the-gate validation + Change Log update).

- [x] **Task 1: Add `serve-smoke` job to `.github/workflows/test.yml`**
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
        - name: Warm Go build cache
          working-directory: apps/api
          run: go build -o /dev/null ./cmd/api
        - name: Create data directory
          run: mkdir -p ${{ github.workspace }}/vido-data
        - name: Start nx serve api (background) and probe smoke endpoint
          run: |
            set -m
            pnpm exec nx serve api > /tmp/nx-serve.log 2>&1 &
            SERVER_PID=$!
            trap "kill -- -$SERVER_PID 2>/dev/null || kill $SERVER_PID 2>/dev/null || true" EXIT

            echo "Started nx serve api (PGID=$SERVER_PID); probing /api/v1/explore-blocks..."
            for i in {1..60}; do
              code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/explore-blocks || true)
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
    - `set -m` makes the backgrounded job its own process-group leader (PGID == `$!`), so `trap "kill -- -$SERVER_PID"` signals the WHOLE tree (pnpm → nx → go run → compiled binary). A plain `kill $!` hits only pnpm and leaks the grandchild server, because `go run` does not forward SIGTERM to the binary it exec's. Trap is armed IMMEDIATELY after `$!` capture to prevent a PID-leak race (CR H1 fix, 2026-06-10).
    - `Warm Go build cache` step (`go build -o /dev/null ./cmd/api`) runs BEFORE the timed probe so cold-compile time is not charged against the 60s curl budget — the probe then covers only startup (migrations + bind). Prevents cold-`~/.cache/go-build` false-fail on ubuntu-latest (CR M2 fix, 2026-06-10).
    - Strict `[ "$code" = "200" ]` check, NOT exit-code-based — catches 500/404 server-up-but-broken (Amelia's catch).
    - `tail -100 /tmp/nx-serve.log` dump on timeout exposes panic stacks / migration errors to PR author (Murat's catch).
    - `TMDB_API_KEY` deliberately omitted — explore-blocks handler verified TMDb-free (Step 2 grep).
    - 60 iterations × 1s sleep, not 30 — accommodates `ubuntu-latest` IO stalls during cold-start migrations (Murat's tightening).

- [x] **Task 2: Local pre-push validation**
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

- [x] **Task 3: Push branch + verify happy-path CI run** ✅ DONE
  - File: (none — observation step)
  - Action: Pushed via `/ship`; PR #51 merged. `serve-smoke` ran **green in 47s**.
  - **Evidence:** https://github.com/j620656786206/vido/actions/runs/27258273213/job/80498080296 — satisfies AC1.

- [x] **Task 4 (post-merge): Gate-the-gate sabotage validation** ✅ DONE
  - File: temporary branch only (`chore/verify-serve-smoke-gate-fails-as-expected`, PR #52, closed unmerged + branch deleted)
  - **Sabotage chosen (CR-refined):** NOT the spec's original `_smoke_validator.go` with `func init(){ undefinedFunction() }`. That is a *package-mode* compile error → caught by `lint`/`test-unit` first → `serve-smoke` SKIPPED (`needs: [lint, test-unit]`), proving nothing. Instead regressed the nx `serve` target to single-file mode (`go run ./cmd/api/main.go`) — the actual 2026-03-26..2026-04-20 regression class — so `main.go`'s references to `static.go` symbols (`registerStaticRoutes`/`getPublicDir`/`securityHeadersMiddleware`) fail to compile under `go run` only.
  - **Result:** `lint` ✅ / `Unit Tests` ✅ / `build` ✅ (all package-mode, compile `static.go` fine — exactly why they MISSED the original regression) while **`serve-smoke` ❌ FAILED** in 1m41s, the `tail -100 /tmp/nx-serve.log` dump showing the three `undefined:` compile errors. PROVES the gate's unique value.
  - **Evidence:** https://github.com/j620656786206/vido/actions/runs/27258979592/job/80500364585 — satisfies AC2 + AC8.

- [x] **Task 5 (post-merge): Document validation result** ✅ DONE
  - Files: `_bmad-output/implementation-artifacts/sprint-status.yaml` (entry `retro-10-AI6-ci-smoke-gate-nx-serve` → done), this spec's Change Log + AC table.
  - **Evidence recorded:** happy-path run (#51, 47s green), sabotage run (#52, serve-smoke-only FAIL with compile-error dump), merged Task-1 SHA `2dbe695` (#51). AC4 first data point = 47s (≪ 90s p95 target); will self-confirm over the next runs.
  - Bundled micro-polish: `|| echo "000"` → `|| true` on the curl probe (the `echo` doubled curl's own `000` → confusing `Last HTTP code: 000000` in the sabotage log). Cosmetic only; string check `[ "$code" = "200" ]` unaffected.

### Acceptance Criteria

- [x] **AC1 (happy path)** ✅ VERIFIED on PR #51: `serve-smoke` passed in **47s** (well under the 5-min cap). Run: https://github.com/j620656786206/vido/actions/runs/27258273213/job/80498080296

- [x] **AC2 (class-of-bug)** ✅ VERIFIED on sabotage PR #52: the gate must exit non-zero AND surface the compile error in the `tail -100 /tmp/nx-serve.log` dump. **Validation refinement (CR/Task-4):** a generic *package-mode* compile error (the spec's original `func init(){ undefinedFunction() }`) would be caught by `lint`/`test-unit` FIRST, skipping `serve-smoke` (`needs: [lint, test-unit]`) — so it does NOT exercise the gate. The gate's *unique* class is **single-file-mode regression** (the real 2026-03-26..2026-04-20 bug). Task 4 therefore regressed the nx `serve` target to `go run ./cmd/api/main.go`; result: `lint`/`test-unit`/`build` all PASS (package mode), **`serve-smoke` FAILS** with the dump showing `cmd/api/main.go:508 undefined: securityHeadersMiddleware`, `:568 undefined: getPublicDir`, `:569 undefined: registerStaticRoutes`. Run: https://github.com/j620656786206/vido/actions/runs/27258979592/job/80500364585

- [x] **AC3 (cleanup)**: Given any failure path — timeout, non-200, server panic, compile error — when the job exits, then no `nx serve` / `go run` background process is leaked. Enforced by `set -m` (background job becomes its own process-group leader, PGID == `$!`) + `trap "kill -- -$SERVER_PID 2>/dev/null || kill $SERVER_PID 2>/dev/null || true" EXIT`, which signals the ENTIRE process tree (pnpm → nx → go run → compiled binary) on every exit path — not just the top pnpm PID. (CR H1, 2026-06-10: the earlier `kill $SERVER_PID`-only form would have leaked the grandchild server because `go run` does not forward SIGTERM to its exec'd binary; harmless under CI runner ephemerality but now correct for local/self-hosted runs too.)

- [~] **AC4 (performance — measurement-based)**: p95 < 90s over the first 10 successful runs. **First data point: 47s** on PR #51 (≪ 90s; the `Warm Go build cache` step keeps compile out of the probe budget). Well within target; will self-confirm as more runs accrue — no tuning needed at the observed timing. (Left `[~]` = on-track-pending-volume, not a blocker.)

- [x] **AC5 (scope independence)**: Given a `serve-smoke` failure, when the workflow proceeds, then `test-e2e-sharded` and `merge-test-results` MUST still run (they depend on `build`, not `serve-smoke`). The new gate is additive, NOT on the E2E critical path.

- [x] **AC6 (dev parity)**: Given a clean dev checkout, when a developer runs `mkdir -p vido-data && VIDO_DATA_DIR=$(pwd)/vido-data pnpm exec nx serve api &; sleep 5; curl http://localhost:8080/api/v1/explore-blocks`, then the response is HTTP 200 with valid JSON body. (Task 2's local validation step.)

- [x] **AC7 (HTTP code strictness)**: Given `nx serve api` running but `/api/v1/explore-blocks` returning a non-200 (e.g., 500 from broken handler logic, 404 from missing route), when `serve-smoke` probes, then MUST exit non-zero. The `[ "$code" = "200" ]` string check (NOT curl exit-code-only) MUST be used.

- [x] **AC8 (gate-the-gate — proof of teeth)** ✅ VERIFIED: sabotage PR #52 (`chore/verify-serve-smoke-gate-fails-as-expected`, closed unmerged + branch deleted) proved `serve-smoke` fails with the compile error visible in the log dump while every other check stays green. Sabotage run URL recorded in Change Log + sprint-status.yaml. AC2 is no longer theatre.

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

## Dev Agent Record

### Context Reference

- Tech-spec (this file) is the authoritative implementation guide.
- Dev-story workflow: `_bmad/bmm/workflows/4-implementation/dev-story/workflow.yaml`
- Agent: Amelia (dev) — 2026-06-10

### Step 2 Mandatory Gates

- **🔗 AC Drift:** N/A — new additive CI job `serve-smoke`; no prior story defines a serve-smoke gate (grep `serve-smoke` across `_bmad-output/implementation-artifacts/*.md` → only this spec). Prior `test.yml` touches (Stories 19-5/6/7) added separate jobs; this change alters no existing job contract. AC5 enforces scope independence.
- **📎 Contract Stamps:** NONE — no `[@contract-v*]` stamps in this spec or upstream refs (grep → 0 hits); a CI-config change consumes no wire contract.
- **🎭 A11y Pre-Flight:** N/A (100% non-frontend — only `.github/workflows/test.yml` touched; zero `apps/web/` files).

### Implementation Plan & Decisions

- **Task 1** — Inserted `serve-smoke` job into `.github/workflows/test.yml` between `test-unit` and `build`, using the party-mode-validated skeleton verbatim (60-iteration fixed-backoff curl loop, strict `[ "$code" = "200" ]` check, `trap`-based PID cleanup set immediately after `$!`, `tail -100 /tmp/nx-serve.log` failure dump, `TMDB_API_KEY` omitted). `needs: [lint, test-unit]` → runs parallel to `build`.
- **TMDb-free assumption re-verified** (Confirm-Before-Coding): the spec's claim was challenged by a grep hit (`explore_blocks_handler.go:150` references TMDb). Read confirmed the TMDb dependency lives **only** in `GetBlockContent` (the `/:id/content` sub-route). The smoke target `GET /api/v1/explore-blocks` → `ListBlocks` (line 53) → `service.GetAllBlocks` is a pure DB query. Assumption holds; no secret needed.
- **Task 2** — Local validation executed on darwin: `VIDO_DATA_DIR=/tmp/vido-smoke-data VIDO_PORT=8080 pnpm exec nx serve api` (package-mode `go run ./cmd/api`) → smoke passed iteration 5, **HTTP 200** with valid JSON body (3 seeded blocks: 熱門電影 / 熱門影集 / 近期新片). Orphan check post-run: no leaked `go run`/`nx serve` process, port 8080 free.

### Completion Notes

- 🔗 AC Drift: N/A (new additive CI job `serve-smoke` — no prior story touches this job; AC5 enforces scope independence)
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this spec or upstream refs — CI config defines/consumes no wire contract)
- 🎭 A11y Pre-Flight: N/A (100% non-frontend — only `.github/workflows/test.yml` touched, no apps/web/ files)
- ✅ Full regression gate: `pnpm nx test api` PASS (exit 0) + `pnpm nx test web` PASS (exit 0, one Nx-flagged flaky retry that then passed). No pre-existing failures detected.
- ✅ YAML well-formed (node `yaml` parser) + prettier-clean (`prettier --check` passes).
- ✅ AC6 (dev parity) verified locally — HTTP 200, valid JSON, no `TMDB_API_KEY`.
- 🔧 CR fixes applied (2026-06-10, `/code-review` adversarial, 3rd LLM): **H1** trap now `set -m` + `kill -- -$PGID` (group-kill the whole pnpm→nx→go run→binary tree; plain `kill $!` leaked the grandchild server since `go run` does not forward SIGTERM) → AC3 guarantee made real, not incidental-on-ephemerality. **M1** front-matter `status:` synced `ready-for-dev` → `review` (was stale vs bottom Status + sprint-status). **M2** added `Warm Go build cache` step (`go build -o /dev/null ./cmd/api`) before the timed probe so cold `~/.cache/go-build` compile is not charged against the 60s curl budget (false-fail guard). Lows L1/L2/L3 noted, not blocking.

### AC Verification Status

| AC | Verifiable now? | Status |
|----|-----------------|--------|
| AC1 (happy-path CI green) | Yes | ✅ PR #51 `serve-smoke` green in 47s (run 27258273213/job 80498080296). |
| AC2 (regression class fails gate) | Yes | ✅ Sabotage PR #52: serve-smoke FAIL w/ compile-error dump, others green (run 27258979592/job 80500364585). |
| AC3 (cleanup trap) | Yes | ✅ `set -m` + group-kill `trap ... EXIT` present (CR H1 hardened); local run left no orphan, port freed. |
| AC4 (p95 < 90s over 10 runs) | Partial | 🟢 On track — first run 47s (≪ 90s). Self-confirms over volume. |
| AC5 (scope independence) | Yes | ✅ `needs: [lint, test-unit]`; E2E shards + merge-results ran green on #51 unaffected; on #52 they ran while serve-smoke failed. |
| AC6 (dev parity) | Yes | ✅ Local smoke HTTP 200 + valid JSON. |
| AC7 (HTTP code strictness) | Yes | ✅ `[ "$code" = "200" ]` string check used (not exit-code-only). |
| AC8 (gate-the-gate evidence) | Yes | ✅ PR #52 closed-unmerged; sabotage FAIL evidence + URL recorded. |

### File List

- `.github/workflows/test.yml` (modified — added `serve-smoke` job)
- `_bmad-output/implementation-artifacts/tech-spec-ci-smoke-gate-nx-serve.md` (modified — task/AC checkboxes, Dev Agent Record, Change Log, Status)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified — status `ready-for-dev` → `review`)

### Change Log

| Date | Change |
|------|--------|
| 2026-06-10 | Task 1: added `serve-smoke` CI job to `.github/workflows/test.yml` (`needs: [lint, test-unit]`, package-mode `nx serve api`, strict 200 probe on `/api/v1/explore-blocks`, trap cleanup, 60×1s retry, log-dump on failure). |
| 2026-06-10 | Task 2: local pre-push validation PASS — HTTP 200 + valid JSON via `pnpm exec nx serve api`, no orphan processes. |
| 2026-06-10 | Regression gate PASS (`nx test api` + `nx test web`); YAML + prettier clean. Story status → review. Tasks 3–5 deferred (delivery + post-merge) with justification. |
| 2026-06-10 | CR (adversarial, 3rd LLM): fixed H1 (trap → `set -m` + group-kill `kill -- -$PGID`; `go run` doesn't forward SIGTERM → plain `kill $!` leaked grandchild server), M1 (front-matter `status` stale `ready-for-dev` → `review`), M2 (added `Warm Go build cache` step so cold compile isn't inside the 60s probe budget). `.github/workflows/test.yml` + this spec updated; YAML/prettier re-validated. |
| 2026-06-10 | **Shipped: PR #51 merged** (`2dbe695`). `serve-smoke` green in 47s → **AC1** ✅. Full CI green, zero self-heal. |
| 2026-06-10 | **Task 4 gate-the-gate: sabotage PR #52** (closed unmerged + branch deleted). CR-refined sabotage = regress nx `serve` to single-file mode `go run ./cmd/api/main.go` (NOT the spec's package-mode `func init(){ undefinedFunction() }`, which would be caught by lint/test-unit first and SKIP serve-smoke). Result: lint/test-unit/build ✅, **serve-smoke ❌** with `undefined: registerStaticRoutes`/`getPublicDir`/`securityHeadersMiddleware` in the log dump → **AC2 + AC8** ✅. Run: actions/runs/27258979592/job/80500364585. |
| 2026-06-10 | **Task 5: documentation + sprint-status `review → done`.** AC4 first data point 47s (≪ 90s). Bundled micro-polish: curl probe `\|\| echo "000"` → `\|\| true` (the `echo` doubled curl's own `000`, showing `Last HTTP code: 000000`); cosmetic, string check unaffected. |

## Status

done

<!-- All tasks complete. Task 1 (job) merged via PR #51; Task 2 local validation green; Task 3 happy-path CI green (47s, AC1); Task 4 gate-the-gate proven via sabotage PR #52 (serve-smoke-only FAIL with compile-error dump, AC2+AC8); Task 5 documentation + sprint-status flip done here. AC4 (p95 over 10 runs) on track at 47s first data point — self-confirms over volume, not a blocker. -->

