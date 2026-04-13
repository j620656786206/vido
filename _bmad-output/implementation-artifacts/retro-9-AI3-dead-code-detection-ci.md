# Story: Add Dead Code Detection to CI Pipeline

Status: done

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

- [x] Task 1: Add Go static analysis to CI pipeline (AC: #1, #4, #5)
  - [x] 1.1 Install `staticcheck` in CI lint job: add `go install honnef.co/go/tools/cmd/staticcheck@latest` step after Go setup
  - [x] 1.2 Add `staticcheck ./...` step to lint job, running from `apps/api/` working directory
  - [x] 1.3 Add `go vet ./...` step to lint job (if not already present), running from `apps/api/` working directory
  - [x] 1.4 Verify existing Go code passes both checks; fix any legitimate findings before merging

- [x] Task 2: Upgrade ESLint unused detection severity (AC: #2, #3, #5)
  - [x] 2.1 In `eslint.config.mjs`, change `@typescript-eslint/no-unused-vars` from `warn` to `error`
  - [x] 2.2 Run `pnpm run lint` locally to verify no new failures from severity upgrade (existing `warn` items are already fixed or need fixing now)
  - [x] 2.3 If existing violations found: fix them (they are dead code by definition)

- [x] Task 3: Add Go setup to CI lint job (AC: #1, #4)
  - [x] 3.1 The current CI lint job only has Node.js setup. Add Go setup step (`actions/setup-go@v5` with `go-version: ${{ env.GO_VERSION }}`) so `staticcheck` and `go vet` can run
  - [x] 3.2 Add `go mod download` step with `working-directory: apps/api` for dependency caching

- [x] Task 4: Verify CI pipeline end-to-end (AC: #5)
  - [x] 4.1 Run full lint locally: `pnpm run lint` (ESLint) + `pnpm run format:check` (Prettier) + `cd apps/api && staticcheck ./... && go vet ./...` (Go)
  - [x] 4.2 Verify all existing tests still pass: `pnpm nx test api` + `pnpm nx test web`

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

### Follow-up risks from dead code removal

The following functions/vars were removed as truly dead in `apps/api/` per staticcheck, but represent designed-but-never-wired features. If their behavior is ever needed, reimplement rather than revert:

- `apps/api/internal/parser/movie_parser.go:extractTitleWithEmbeddedYear` — handled filenames with embedded years ("2001.A.Space.Odyssey.1968", "Blade.Runner.2049.2017"). The deprecated root backend (`/internal/parser/`) has working tests for these cases; apps/api had the scaffold but no call site.
- `apps/api/internal/wikipedia/infobox.go:extractInfobox` — Infobox template extraction. Only `findAllInfoboxPositions` + `extractBalancedBraces` are used now.
- `apps/api/internal/parser/patterns.go:tvPatternAnime` — anime `[Group] Show - 01 [1080p].mkv` pattern. `tv_parser.go` uses different `tvPatternAnimeEp`/`tvPatternAnimeDash` patterns (different module).

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 3+1: Added Go setup, go vet, and staticcheck (pinned `@2026.1`, cached binary) to CI lint job in test.yml. Go checks run before JS checks so one does not mask the other.
- Task 1.4: Fixed 15 staticcheck findings — 5 dead code (U1000), 2 error capitalization (ST1005), 4 nil context (SA1012), 1 struct conversion (S1016), 3 unused append (SA4010)
- Task 1.4: Fixed 13 go vet findings — unkeyed struct literals in parse_queue_service.go (NullInt64/NullString)
- Task 2: Upgraded ESLint no-unused-vars from warn → error, fixed 26 violations (deleted dead code, removed unused imports, prefixed intentionally-unused vars with _)
- Task 4: Verified go vet clean, staticcheck clean, ESLint 0 errors, pnpm nx test api PASS, pnpm nx test web PASS
- Follow-up commit `1d97082` fixed 4 files for Prettier formatting — initial commit missed `pnpm run format:check` locally; Task 4.1 updated to include it
- Code review follow-up: reverted misapplied `_` prefix on used vars in full-app-audit.spec.ts; switched test `context.TODO()` → `context.Background()`; removed unused `_routeTree` binding in SettingsLayout.spec.tsx
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Change Log

- 2026-04-10: Added Go static analysis (go vet + staticcheck) to CI lint job, fixed 28 existing findings across Go and TypeScript codebases, upgraded ESLint no-unused-vars to error severity (Task 1, 2, 3, 4)
- 2026-04-10: Follow-up Prettier fix (commit `1d97082`) — reformatted 4 files that initial commit missed
- 2026-04-13: Code review fixes — pinned staticcheck to `@2026.1` with binary caching, reordered lint job so Go runs before JS, reverted misapplied `_` prefixes on used vars, normalized test contexts to `context.Background()`, removed unused `_routeTree` binding, added follow-up-risk notes for removed dead functions

### File List

- `.github/workflows/test.yml` — Added Go setup, go mod download, go vet, staticcheck steps to lint job; pinned staticcheck version with binary cache; reordered so Go runs before JS
- `project-context.md` — Markdown list formatting (Prettier follow-up from commit `1d97082`)
- `eslint.config.mjs` — Changed @typescript-eslint/no-unused-vars from warn to error
- `apps/api/internal/services/parse_queue_service.go` — Fixed unkeyed struct literals (go vet)
- `apps/api/internal/ai/fansub_detector.go` — Removed unused var anyBracketStartPattern
- `apps/api/internal/database/migrations/runner.go` — Removed unused const schemaMigrationsTable
- `apps/api/internal/health/checker.go` — Lowercased error strings
- `apps/api/internal/parser/movie_parser.go` — Removed unused func extractTitleWithEmbeddedYear
- `apps/api/internal/parser/patterns.go` — Removed unused var tvPatternAnime
- `apps/api/internal/services/audio_extractor_service_test.go` — Replaced nil context with context.TODO()
- `apps/api/internal/services/nfo_reader_service.go` — Used struct conversion instead of literal
- `apps/api/internal/services/scanner_service_test.go` — Removed unused append results
- `apps/api/internal/services/transcription_service_test.go` — Replaced nil context with context.TODO()
- `apps/api/internal/subtitle/batch.go` — Removed unused failedItems append
- `apps/api/internal/wikipedia/infobox.go` — Removed unused func extractInfobox
- `apps/web/src/components/dashboard/DownloadPanel.spec.tsx` — Removed unused createWrapper function
- `apps/web/src/components/downloads/DownloadFilterTabs.spec.tsx` — Removed unused FilterStatus import
- `apps/web/src/components/downloads/DownloadList.spec.tsx` — Removed unused SortField, SortOrder imports
- `apps/web/src/components/library/LibraryTable.spec.tsx` — Removed unused container destructure
- `apps/web/src/components/media/PosterCard.spec.tsx` — Removed unused container destructure
- `apps/web/src/components/settings/SettingsLayout.spec.tsx` — Prefixed unused routeTree with _
- `apps/web/src/hooks/useLibrary.ts` — Removed unused type imports
- `tests/e2e/connection-health.spec.ts` — Removed unused mockEmptyHistory
- `tests/e2e/downloads.spec.ts` — Removed unused API_BASE_URL
- `tests/e2e/graceful-degradation.spec.ts` — Removed unused mockHealthyResponse, prefixed movies with _
- `tests/manual/full-app-audit.spec.ts` — Removed unused expect import, prefixed side-effect vars with _
- `tests/support/global-setup.ts` — Prefixed unused config param with _
