# Story: Add Dead Code Detection to CI Pipeline

Status: ready-for-dev

## Story

As a developer,
I want the CI pipeline to automatically detect unused code (dead functions, unused exports, unreferenced variables),
so that dead code from DS/CR/TA stages is caught before merge instead of accumulating silently.

## Acceptance Criteria

1. Given the CI lint stage, when a Go source file contains an unused exported function, then the pipeline fails with a clear error message identifying the dead code
2. Given the CI lint stage, when a TypeScript/React file contains an unused export or variable (not prefixed with `_`), then the pipeline fails with a warning or error
3. Given the ESLint config already has `@typescript-eslint/no-unused-vars: warn`, when the severity is upgraded to `error`, then CI lint fails on unused variables instead of just warning
4. Given Go's `staticcheck` or `go vet` is added to CI, when run against `apps/api/...`, then it reports unused functions, deprecated API usage, and dead code patterns
5. Given all changes are applied, when the existing CI test suite runs, then no regressions are introduced (all existing lint + tests pass)

## Tasks / Subtasks

- [ ] Task 1: Add Go static analysis to CI pipeline (AC: #1, #4, #5)
  - [ ] 1.1 Install `staticcheck` in CI lint job: add `go install honnef.co/go/tools/cmd/staticcheck@latest` step after Go setup
  - [ ] 1.2 Add `staticcheck ./...` step to lint job, running from `apps/api/` working directory
  - [ ] 1.3 Add `go vet ./...` step to lint job (if not already present), running from `apps/api/` working directory
  - [ ] 1.4 Verify existing Go code passes both checks; fix any legitimate findings before merging

- [ ] Task 2: Upgrade ESLint unused detection severity (AC: #2, #3, #5)
  - [ ] 2.1 In `eslint.config.mjs`, change `@typescript-eslint/no-unused-vars` from `warn` to `error`
  - [ ] 2.2 Run `pnpm run lint` locally to verify no new failures from severity upgrade (existing `warn` items are already fixed or need fixing now)
  - [ ] 2.3 If existing violations found: fix them (they are dead code by definition)

- [ ] Task 3: Add Go setup to CI lint job (AC: #1, #4)
  - [ ] 3.1 The current CI lint job only has Node.js setup. Add Go setup step (`actions/setup-go@v5` with `go-version: ${{ env.GO_VERSION }}`) so `staticcheck` and `go vet` can run
  - [ ] 3.2 Add `go mod download` step with `working-directory: apps/api` for dependency caching

- [ ] Task 4: Verify CI pipeline end-to-end (AC: #5)
  - [ ] 4.1 Run full lint locally: `pnpm run lint` (ESLint) + `cd apps/api && staticcheck ./... && go vet ./...` (Go)
  - [ ] 4.2 Verify all existing tests still pass: `pnpm nx test api` + `pnpm nx test web`

## Dev Notes

### Root Cause

Epic 9 retro identified dead code accumulating across DS (dev story), CR (code review), and TA (test automation) stages. When a CR removes a function's caller but not the function itself, or TA adds a helper that later becomes unused, there is no automated gate to catch it. The CI lint job currently only runs ESLint for frontend — no Go static analysis exists.

### Current CI Pipeline (`test.yml`)

- **Lint job (Stage 1):** ESLint + Prettier check for frontend only. No Go lint/vet/staticcheck.
- **Go setup** only exists in the `build` job (Stage 3a), not in lint.
- ESLint has `@typescript-eslint/no-unused-vars: warn` — warns but doesn't fail CI.

### ESLint Config (`eslint.config.mjs`)

- Line 130-136: `@typescript-eslint/no-unused-vars` is set to `warn` with `_` prefix ignore pattern
- Line 138: Base `no-unused-vars` disabled in favor of TS version
- Upgrading to `error` will make CI fail on unused variables — this is the desired behavior

### Go Static Analysis Tools

- **`staticcheck`** (honnef.co/go/tools): Industry standard. Detects unused code (U1000), deprecated APIs, performance issues. More comprehensive than `go vet`.
- **`go vet`**: Built-in. Catches suspicious constructs, unreachable code, incorrect format strings.
- Both should run from `apps/api/` working directory.

### CI Workflow Structure (`test.yml`)

The lint job needs:
1. Go setup step (currently only in `build` job)
2. `go mod download` for caching
3. `go vet ./...` step
4. `staticcheck ./...` step

### What NOT to Do

- DO NOT add golangci-lint (too heavy for this scope, staticcheck is sufficient)
- DO NOT modify test jobs — only the lint job needs changes
- DO NOT change ESLint ignore patterns — the `_` prefix convention is correct
- DO NOT add new ESLint plugins — the existing `@typescript-eslint/no-unused-vars` rule covers the need

### References

- [Source: .github/workflows/test.yml] — CI pipeline with lint job at lines 34-66
- [Source: eslint.config.mjs:130-136] — Current unused vars rule (warn)
- [Source: _bmad-output/implementation-artifacts/epic-9-retro-2026-04-10.md] — AI-3 action item
- [Source: project-context.md] — Rule 12: pre-commit disabled, lint in CI only

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Change Log

### File List
