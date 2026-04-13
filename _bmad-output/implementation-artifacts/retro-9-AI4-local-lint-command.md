# Story: Add Local Lint Convenience Command

Status: ready-for-dev

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

- [ ] Task 1: Align `apps/api:lint` target with CI checks (AC: #2, #4, #6)
  - [ ] 1.1 In `apps/api/project.json`, update the `lint` target's `command` so it runs both `go vet ./...` and `staticcheck ./...` sequentially (fail fast on first error). Use `go vet ./... && staticcheck ./...` — Nx's `run-commands` executor honors shell chaining via `command:`
  - [ ] 1.2 Verify `apps/api/project.json` still has `"cwd": "apps/api"` so the relative package paths resolve correctly
  - [ ] 1.3 Run `pnpm nx run api:lint` locally and confirm both tools execute; fix any new findings (if CI already passes today, this should be clean)

- [ ] Task 2: Add `lint:all` script to root `package.json` (AC: #1, #6)
  - [ ] 2.1 In root `package.json` `scripts`, add `"lint:all": "pnpm nx run-many -t lint && pnpm run format:check"` — this runs `api:lint` (go vet + staticcheck) and `web:lint` (eslint) in parallel via Nx, then runs Prettier's format check
  - [ ] 2.2 Do NOT add a redundant `pnpm run lint` — `web:lint` (via `@nx/eslint/plugin`) already covers ESLint. Adding root `eslint .` would double-run frontend ESLint
  - [ ] 2.3 Run `pnpm lint:all` locally on a clean checkout and confirm exit code 0; intentionally introduce a lint error in each layer (one Go unused func, one TS unused var, one unformatted file) to verify each surfaces the failure

- [ ] Task 3: Update `project-context.md` to document the new command (AC: #4, #5)
  - [ ] 3.1 In Rule 12 (Code Quality Checks, lines ~332-341), replace the `✅ Run \`pnpm run lint\` and \`pnpm run format:check\` locally before pushing` line with `✅ Run \`pnpm lint:all\` locally before pushing (mirrors CI exactly)`
  - [ ] 3.2 Add a sub-note under Rule 12 listing what `lint:all` actually runs: `go vet + staticcheck + eslint + prettier --check`
  - [ ] 3.3 Add a one-time setup note: `First-time setup: \`go install honnef.co/go/tools/cmd/staticcheck@2026.1\` — pinned to match CI's STATICCHECK_VERSION`
  - [ ] 3.4 In "Pre-Commit Checklist" section (lines ~850-857), replace the two bullet `pnpm run format:check` + `pnpm run lint` lines with a single bullet: `Run \`pnpm lint:all\` — runs go vet + staticcheck + eslint + prettier --check (mirrors CI)`. Keep the fix hint: `fix formatting with \`pnpm exec prettier --write <files>\``
  - [ ] 3.5 Update the "Last Updated" header line at the top of project-context.md to note the new lint convenience command (e.g., `2026-04-13 (Rule 12: added \`pnpm lint:all\` local convenience command)`)

- [ ] Task 4: Verify end-to-end parity with CI (AC: #1, #6)
  - [ ] 4.1 Run `pnpm lint:all` on a clean main branch — must pass (no findings; matches current CI state)
  - [ ] 4.2 Confirm each tool runs in the expected order/layer by reviewing Nx output (look for `> nx run api:lint`, `> nx run web:lint`, then Prettier output)
  - [ ] 4.3 No new CI changes required for this story — CI's existing `lint` job already runs the same four tools. If CI changes ARE needed (e.g., to re-order or consolidate), push that to a separate follow-up; this story is local-only

- [ ] Task 5: Update sprint-status.yaml (AC: #5, #6)
  - [ ] 5.1 Change `retro-9-AI4-local-lint-command: backlog` → `done` with completion note and date once Tasks 1-4 are verified green

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

### Debug Log References

### Completion Notes List

### File List
