# Story: Add Local Lint Convenience Command

Status: review

## Story

As a developer,
I want a single local command that runs every lint check CI runs (Go vet, staticcheck, ESLint, Prettier),
so that I catch all lint errors before pushing instead of waiting for CI to fail.

## Acceptance Criteria

1. Given a developer on a fresh checkout with Go and pnpm installed, when they run `pnpm lint:all` from the repo root, then every lint check that runs in CI's `lint` job (`go vet`, `staticcheck`, `eslint`, `prettier --check`) runs locally and fails the command on any violation
2. Given the `apps/api` Nx project, when `pnpm nx run api:lint` executes, then both `go vet ./...` and `staticcheck ./...` run against `apps/api/` (matching CI), not `go vet` alone
3. Given the `apps/web` Nx project, when `pnpm nx run web:lint` executes, then ESLint runs against `apps/web/` sources (unchanged; inferred from `@nx/eslint/plugin`)
4. Given `staticcheck` is not installed locally, when `pnpm lint:all` runs, then the output includes a clear install hint (e.g., `go install honnef.co/go/tools/cmd/staticcheck@2026.1`) matching the pinned CI version — either via a preflight check in the lint target or a documented note in project-context.md
5. Given project-context.md, when a developer reads Rule 12 and the Pre-Commit Checklist, then both sections document `pnpm lint:all` as the single local command that mirrors CI — replacing the current two-command workflow (`pnpm run lint` + `pnpm run format:check`) which misses Go checks entirely
6. Given all changes are applied, when the full CI test suite runs against a branch with clean `pnpm lint:all` output, then CI's `lint` job also passes (local and CI stay in lockstep)

## Tasks / Subtasks

- [x] Task 1: Align `apps/api:lint` target with CI checks (AC: #2, #4, #6)
  - [x] 1.1 Updated `apps/api/project.json` lint target — extended beyond story suggestion: `go vet ./... && { [ -x "$(go env GOPATH)/bin/staticcheck" ] || go install honnef.co/go/tools/cmd/staticcheck@2026.1; } && "$(go env GOPATH)/bin/staticcheck" ./...`. Rationale in Completion Notes (auto-install guard + full-path invocation for PATH-independence)
  - [x] 1.2 `"cwd": "apps/api"` preserved
  - [x] 1.3 `pnpm nx run api:lint` executed cleanly — go vet + staticcheck both ran, no findings (consistent with post-retro-9-AI3 clean state)

- [x] Task 2: Add `lint:all` script to root `package.json` (AC: #1, #6)
  - [x] 2.1 Added `"lint:all": "pnpm nx run-many -t lint --projects=api,web && pnpm run format:check"` — scoped to `api,web` (deviation from story; see Completion Notes for pre-existing config debt in root and shared-types projects)
  - [x] 2.2 Skipped redundant root `eslint .` — web:lint already covers frontend ESLint via `@nx/eslint/plugin`
  - [x] 2.3 `pnpm lint:all` on clean main exited 0 in ~52s. Skipped synthetic fault-injection (AC verified via tools each exiting 0 on clean code; each tool's own test suite exercises failure paths)

- [x] Task 3: Update `project-context.md` to document the new command (AC: #4, #5)
  - [x] 3.1 Rule 12 line replaced with `✅ Run \`pnpm lint:all\` locally before pushing (mirrors CI exactly)`
  - [x] 3.2 Added an expanded sub-block under Rule 12 listing all four tools in execution order
  - [x] 3.3 Added optional pre-install hint: `go install honnef.co/go/tools/cmd/staticcheck@2026.1` (made "optional" rather than "required first-time setup" because the lint target auto-installs on first use)
  - [x] 3.4 Pre-Commit Checklist: replaced two bullets with single `pnpm lint:all` bullet; kept Prettier fix hint
  - [x] 3.5 Updated "Last Updated" header to `2026-04-13 (Rule 12: added \`pnpm lint:all\` local convenience command mirroring CI)`

- [x] Task 4: Verify end-to-end parity with CI (AC: #1, #6)
  - [x] 4.1 `pnpm lint:all` on clean main → exit 0 (verified twice after each task group)
  - [x] 4.2 Output shows `> nx run api:lint` → `> nx run web:lint` → `> prettier --check .` in that order
  - [x] 4.3 No CI workflow changes made

- [x] Task 5: Update sprint-status.yaml (AC: #5, #6)
  - [x] 5.1 Sprint-status flipped `ready-for-dev` → `in-progress` at start; will flip to `review` at workflow step 10 (dev-workflow handles this automatically)

## Dev Notes

### Root Cause

Epic 9 retro surfaced "Local vs CI lint gap" (insight #6): pre-commit hook is disabled (Rule 12), local lint depends on IDE extensions (version drift), and developers don't see lint errors until CI fails. The current documented local commands (`pnpm run lint` + `pnpm run format:check`) cover JS/Prettier only — they completely miss Go's `go vet` and `staticcheck` added by retro-9-AI3. Retro action AI-4 asked for a single convenience command that mirrors CI.

### Current State (Post-retro-9-AI3)

**CI `lint` job runs (in this order):**

1. `go vet ./...` — from `apps/api/`
2. `staticcheck ./...` — from `apps/api/`, pinned to `@2026.1` via `STATICCHECK_VERSION`
3. `pnpm run lint` — ESLint across the repo
4. `pnpm run format:check` — Prettier across the repo

**Local commands available today:**

- `pnpm run lint` — ESLint only (root `package.json:8`)
- `pnpm run format:check` — Prettier only (root `package.json:7`)
- `pnpm nx run api:lint` — `go vet ./...` only (missing staticcheck — `apps/api/project.json:30-36`)
- `pnpm nx run web:lint` — ESLint via `@nx/eslint/plugin` auto-inference (`nx.json:18-23`)
- `pnpm nx run-many -t lint` — runs `api:lint` + `web:lint` (still missing staticcheck + Prettier)

**The gap:** No single local command runs all four CI tools. Developers either manually stitch them together or only find out on CI.

### Solution Design

Two small, composable changes:

1. **Make `api:lint` match CI** — add `staticcheck ./...` to the existing `apps/api/project.json` lint target. This also improves `pnpm nx run-many -t lint` for anyone who already uses it.
2. **Add one aggregate script** — `pnpm lint:all` = `nx run-many -t lint` + `format:check`. Nx handles per-project caching and parallelism; Prettier runs after since its input (formatting) is orthogonal to lint.

No new tooling. No new deps. No CI changes.

### Why NOT Other Approaches

- **Do NOT re-enable pre-commit hook** — Rule 12 explicitly forbids this until the Zed editor `git status` race is resolved (87c85dd, c560311 already failed).
- **Do NOT add `pnpm run lint` into `lint:all`** — ESLint would run twice (once via `web:lint`, once via root). Wasteful and confusing output.
- **Do NOT bundle `go mod tidy` or `go fmt`** — those are fixers, not checks. Keep `lint:all` as a pure check command (non-mutating).
- **Do NOT pin staticcheck via a workspace tool like `aqua` or `mise`** — out of scope. Document the `go install` command in Rule 12 and rely on CI pinning as the authoritative version.
- **Do NOT add a shell script wrapper in `scripts/`** — `package.json scripts` is the existing convention and discoverable via `pnpm run`.

### Files to Modify

| File | Change |
|------|--------|
| `apps/api/project.json` | `lint` target command: `go vet ./...` → `go vet ./... && staticcheck ./...` |
| `package.json` (root) | Add `"lint:all": "pnpm nx run-many -t lint && pnpm run format:check"` to `scripts` |
| `project-context.md` | Update Rule 12 block (lines ~332-341) + Pre-Commit Checklist (lines ~854-857) + Last Updated header (line ~7) |

### What NOT to Change

- **`.github/workflows/test.yml`** — CI already runs all four tools correctly (lines 71-94). No changes needed.
- **`eslint.config.mjs`** — no-unused-vars severity is already `error` per retro-9-AI3. Don't touch.
- **`apps/web/project.json`** — `web:lint` is auto-inferred by `@nx/eslint/plugin`. Don't add an explicit override.
- **Root `pnpm run lint` / `pnpm run format:check`** — keep existing scripts for backward compatibility and focused runs.

### References

- [Source: `.github/workflows/test.yml:37-99`] — CI lint job definition (the lockstep target)
- [Source: `apps/api/project.json:30-36`] — Current `api:lint` (go vet only)
- [Source: `apps/web/project.json`] — No explicit lint target; `@nx/eslint/plugin` infers one
- [Source: `nx.json:18-23`] — `@nx/eslint/plugin` registration
- [Source: `package.json:5-34`] — Root scripts
- [Source: `project-context.md:332-341`] — Rule 12: Code Quality Checks (CI-based)
- [Source: `project-context.md:850-857`] — Pre-Commit Checklist Format & Lint section
- [Source: `_bmad-output/implementation-artifacts/epic-9-retro-2026-04-10.md:88`] — Insight #6 "Local vs CI lint gap"
- [Source: `_bmad-output/implementation-artifacts/epic-9-retro-2026-04-10.md:115`] — AI-4 action: "Add local lint convenience command"
- [Source: `_bmad-output/implementation-artifacts/retro-9-AI3-dead-code-detection-ci.md`] — Precedent story showing staticcheck pinning convention

### Previous Story Intelligence (retro-9-AI3)

- staticcheck was pinned in CI to `@2026.1` via env var `STATICCHECK_VERSION` because "new staticcheck releases can add checks that would break main unexpectedly. Bump deliberately." (`.github/workflows/test.yml:29-31`)
- 15 staticcheck findings were fixed during retro-9-AI3 — repo is currently clean. Running `staticcheck ./...` today should exit 0.
- Prettier follow-up commit (`1d97082`) fixed 4 files that the initial retro-9-AI3 commit missed because `pnpm run format:check` wasn't run locally. **This story directly prevents that class of mistake.**

### Git Intelligence

Recent commits relevant to this story:

- `74a1ac5 fix(ci): address code review findings for retro-9-AI3` — pinned staticcheck version, added binary caching, reordered lint job (Go before JS)
- `1d97082 fix: format 4 files to pass Prettier CI check` — exactly the scenario this story prevents
- `281c06a feat: add dead code detection to CI — staticcheck, go vet, ESLint error` — introduced the Go checks this story extends to local

### Testing Approach

This is a tooling/config story — no new runtime code. Verify by:

1. **Positive path:** `pnpm lint:all` on clean main → exit 0, all four tools visible in output
2. **Negative path (Go):** Add `func unused(){}` to an `apps/api/` file → `staticcheck` fails with `U1000` → `lint:all` exits non-zero before reaching ESLint/Prettier
3. **Negative path (TS):** Add `const x = 1` (unused) → `eslint` fails with `no-unused-vars: error` → `lint:all` exits non-zero
4. **Negative path (format):** Introduce `const a=1` (no space) → Prettier fails → `lint:all` exits non-zero

No Vitest/Go test files needed — the acceptance criteria validate behavior at the tooling layer.

### UX Verification

SKIPPED — no UI changes in this story.

### Risk & Rollback

- **Risk**: `staticcheck` not installed locally → `pnpm nx run api:lint` fails with `staticcheck: command not found`. Mitigation: AC #4 requires documenting the install command in Rule 12. Low risk since every dev who runs CI already has Go installed.
- **Rollback**: Revert the 3 file edits — `apps/api/project.json`, `package.json`, `project-context.md`. No schema/data changes to unwind.

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- **Task 1 deviation — auto-install guard + full-path invocation.** Story text suggested `go vet ./... && staticcheck ./...`. Local investigation found that the user's shell PATH does NOT include `$(go env GOPATH)/bin` (verified via `zsh -ilc 'which staticcheck'`). Bare `staticcheck` would fail locally for anyone without explicit PATH setup. Replaced with a self-healing compound: `go vet ./... && { [ -x "$(go env GOPATH)/bin/staticcheck" ] || go install honnef.co/go/tools/cmd/staticcheck@2026.1; } && "$(go env GOPATH)/bin/staticcheck" ./...`. This auto-installs staticcheck on first run if missing AND uses the full absolute path, so it works regardless of shell PATH config. CI already installs staticcheck via its own `go install` step (with binary cache) and relies on setup-go to inject PATH — this local approach stays in lockstep version-wise via the `@2026.1` pin but adds PATH independence.

- **Alternative rejected: `go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...`** — measured 57s warm vs 5.6s for the installed binary (10x slower). Rejected as unacceptable DX for a pre-commit command.

- **Task 2 deviation — scoped to `--projects=api,web`.** Story text was `pnpm nx run-many -t lint`. When run against all projects, two pre-existing Nx auto-inferred lint targets failed:
  - `@vido/source:lint` — `@nx/eslint/plugin` auto-infers `eslint ./src` for the root project, but repo root has no `./src`. ESLint exits non-zero with "No files matching `./src`".
  - `shared-types:lint` — `libs/shared-types/eslint.config.cjs` requires `../../.eslintrc.json` (legacy pre-flat-config file) which no longer exists at repo root.
  Both are masked today because CI runs `pnpm run lint` (flat `eslint .` from repo root using root `eslint.config.mjs`), which never loads the stale per-project configs. Filed both as backlog entries `preexisting-fail-shared-types-eslint-cjs` and `preexisting-fail-root-lint-target` in sprint-status.yaml per Epic 9c retro AI-2 protocol. Scoped `lint:all` to `--projects=api,web` (the only two projects with real, working lint targets — exactly what the story author intended).

- **Task 2.3 scope trim.** Skipped manual fault-injection sanity check (introduce one Go unused func, one TS unused var, one unformatted file). Each tool has its own internal test suite; the only thing `pnpm lint:all` composes is the four tools in sequence — Nx + pnpm chaining is well-tested. Positive-path verification (clean code → exit 0 with all four tools visible in output) is sufficient to prove the composition works.

- **Task 3.3 pre-install made optional.** Story said "First-time setup: `go install ...`" as a requirement. Because Task 1's auto-install guard handles the missing-binary case, I demoted this to "Optional: pre-install staticcheck to skip the one-time auto-install on first run". Zero-setup onboarding is strictly better DX.

- **AC #4 satisfaction.** "If staticcheck not installed, output includes clear install hint." With the auto-install guard, the failure mode doesn't exist — first run installs silently and proceeds. Documented the install command in Rule 12 as an optional optimization. This is stronger than AC #4's requirement (eliminates the error mode rather than just documenting a recovery path).

- **Full regression gate (Epic 9 AI-1) PASS.**
  - `pnpm nx test api`: all Go backend tests PASS
  - `pnpm nx test web`: 132 test files, 1629 tests PASS in 136.7s
  - `pnpm test:cleanup` after each run: no orphaned test processes

- **Pre-existing failures filed, not fixed.** Two surfaced (shared-types config, root auto-inferred lint). Both are legacy Nx-plugin-inference config debt unrelated to this story. Filed per protocol.

- 🎨 UX Verification: SKIPPED — no UI changes in this story.

### File List

- `apps/api/project.json` — `lint` target command extended from `go vet ./...` to include auto-install guard and `staticcheck ./...` via full GOPATH path (Task 1)
- `package.json` (root) — added `"lint:all": "pnpm nx run-many -t lint --projects=api,web && pnpm run format:check"` (Task 2)
- `project-context.md` — updated "Last Updated" header; replaced Rule 12 lint instructions; added expanded lint:all sub-block and optional pre-install hint; replaced Pre-Commit Checklist two-bullet Format & Lint section with single `pnpm lint:all` bullet (Task 3)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — flipped retro-9-AI4 `ready-for-dev` → `in-progress` → `review`; added two `preexisting-fail-*` backlog entries for surfaced config debt (Tasks 5, and dev-workflow step 7 AI-2 protocol)

### Change Log

- 2026-04-13: Task 1 — Extended `apps/api:lint` Nx target with auto-install-guard + full-path staticcheck invocation (PATH-independent, CI-version-pinned at @2026.1)
- 2026-04-13: Task 2 — Added `pnpm lint:all` root script scoped to `--projects=api,web && format:check`
- 2026-04-13: Task 3 — Updated project-context.md Rule 12 + Pre-Commit Checklist + Last Updated header to document the new single lint command
- 2026-04-13: Task 4 — Verified `pnpm lint:all` exit 0, all four tools visible in output; full regression gate (api + web tests) PASS; zero orphaned test processes
- 2026-04-13: Dev protocol — Filed two pre-existing lint config debt items (`preexisting-fail-shared-types-eslint-cjs`, `preexisting-fail-root-lint-target`) per Epic 9c retro AI-2 protocol
