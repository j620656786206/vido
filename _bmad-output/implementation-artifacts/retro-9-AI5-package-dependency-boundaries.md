# Story: Document Package Dependency Boundaries

Status: done

## Story

As a developer working on `apps/api/internal/` packages,
I want package dependency boundaries (especially the `subtitle ↔ services` cycle constraint) documented in project-context.md with a pre-approved workaround pattern,
so that I don't waste time rediscovering circular import constraints or attempt "clean-up" refactors that reintroduce cycles.

## Acceptance Criteria

1. Given a developer reading `project-context.md`, when they look for Go package import rules, then they find a dedicated "Package Dependency Boundaries" section (new Rule 19) that declares the allowed and forbidden import directions for the key internal packages (`handlers`, `services`, `subtitle`, `ai`, `repository`, `models`)
2. Given the `subtitle` package imports `services.TerminologyCorrectionServiceInterface`, when a developer is about to add `import "github.com/vido/api/internal/subtitle"` to any file under `internal/services/`, then the documented rule states this is FORBIDDEN and points to the mirror-types workaround (with `TranslationBlock` cited as the reference implementation)
3. Given Rule 19 is documented, when a developer needs to call subtitle-package logic from a service (e.g., parse SRT, format blocks), then the docs give a step-by-step workaround: (a) mirror the minimal type in `services/`, (b) inline the logic, (c) add an explanatory comment citing the cycle, (d) keep the two implementations in sync via code review
4. Given the existing cycle-avoidance comments in `translation_service.go:30-33` and `transcription_service.go:362-368`, when the new Rule 19 lands, then those comments can simply reference "see project-context.md Rule 19" instead of re-explaining the cycle (reduces duplication)
5. Given a Go test file in `apps/api/internal/`, when it runs in CI, then the test fails loudly if a new file under `internal/services/` adds an import of `github.com/vido/api/internal/subtitle` — preventing the documented-but-unenforced constraint from silently regressing
6. Given all changes are applied, when `pnpm lint:all` and the full test suite run, then no regressions are introduced (existing behavior unchanged; this story is documentation + one new test)

## Tasks / Subtasks

- [x] Task 1: Add "Rule 19: Package Dependency Boundaries" to `project-context.md` (AC: #1, #2, #3)
  - [x] 1.1 Insert a new Rule 19 after Rule 18 (API Boundary Case Transformation, lines ~473-498). Keep the existing Rule 19+ numbering stable by only appending
  - [x] 1.2 Open with a summary import-direction rule:
    ```
    Handler → Service → Repository → Database (Rule 4 already)
    Handler → Subtitle → Service (allowed, single-direction)
    Service ↛ Subtitle    (FORBIDDEN — would cycle with subtitle → services.TerminologyCorrectionServiceInterface)
    ```
  - [x] 1.3 Include a "Known Cycle Points" subsection listing the real cycle in the codebase today:
    - `subtitle/engine.go:61,90` imports `services.TerminologyCorrectionServiceInterface`
    - Therefore no file under `internal/services/` may `import "github.com/vido/api/internal/subtitle"`
  - [x] 1.4 Add "Workaround Pattern: Mirror Types" subsection with:
    - Step 1: Define a minimal local type in `services/` (e.g., `TranslationBlock`) that mirrors only the fields you need from the `subtitle` type
    - Step 2: Inline the minimum logic you need (e.g., `parseSRTToTranslationBlocks`). Match the source package's validation rules (regex, error handling) so behavior stays identical
    - Step 3: Add a comment citing the cycle and project-context.md Rule 19
    - Step 4: Reference implementation: `apps/api/internal/services/translation_service.go:30-39` (type mirror) and `apps/api/internal/services/transcription_service.go:362-369` (inline parser)
  - [x] 1.5 Add a brief list of "Leaf packages (no internal deps — safe to import from anywhere):" → `ai`, `models`, `sse`, `retry`, `cache`, `secrets`, `errors`, `logger`. These are always safe targets
  - [x] 1.6 Update the "Last Updated" header (line ~7) to mention Rule 19 addition

- [x] Task 2: Slim existing cycle comments to reference the new rule (AC: #4)
  - [x] 2.1 In `apps/api/internal/services/translation_service.go:30-33`, shorten to: `// TranslationBlock mirrors subtitle.SubtitleBlock. services ↛ subtitle — see project-context.md Rule 19.`
  - [x] 2.2 In `apps/api/internal/services/transcription_service.go:362-368`, shorten the `parseSRTToTranslationBlocks` comment block similarly: `// Inline SRT parser. services ↛ subtitle — see project-context.md Rule 19. Mirrors subtitle.ParseSRT validation.`
  - [x] 2.3 Keep the `srtTimestampPattern` regex comment (line 363) — it explains the WHY of validation, not the cycle

- [x] Task 3: Add a Go test enforcing the `services ↛ subtitle` rule (AC: #5, #6)
  - [x] 3.1 Create `apps/api/internal/boundaries_test.go` in the `internal` package. Use a package-level test because import rules span the whole module
  - [x] 3.2 Implementation approach: use `golang.org/x/tools/go/packages` OR — simpler and dependency-free — walk `internal/services/` with `go/parser.ParseDir`, iterate each file's import specs, and fail if any path equals `github.com/vido/api/internal/subtitle`
  - [x] 3.3 Prefer the dependency-free approach: `go/parser` + `go/ast` are in the stdlib, no new `go.mod` entries needed
  - [x] 3.4 Test name: `TestServicesMustNotImportSubtitle`. On violation, report the offending file + import line with a message pointing at project-context.md Rule 19
  - [x] 3.5 Run the test locally: `cd apps/api && go test ./internal/ -run TestServicesMustNotImportSubtitle` → MUST pass (no current violation)
  - [x] 3.6 Add a second negative-path sanity test `TestServicesMustNotImportSubtitle_detectsViolation` using an inline synthetic fileset (via `go/parser.ParseFile` with source string) to confirm the rule triggers when the bad import IS present — proves the test isn't vacuously passing

- [x] Task 4: Verify docs + test end-to-end (AC: #6)
  - [x] 4.1 Run `pnpm lint:all` (post retro-9-AI4) — PASS (0 errors; 108 pre-existing warnings unchanged; Prettier clean; `nx run api:lint` covers go vet + staticcheck@2026.1)
  - [x] 4.2 Run `pnpm nx test api` — PASS (full Go regression, all packages incl. new `internal/boundaries_test.go`)
  - [x] 4.3 staticcheck verified via `pnpm lint:all` (`nx run api:lint` chain) — PASS (refactored from deprecated `parser.ParseDir` (SA1019) to `os.ReadDir` + `parser.ParseFile` to satisfy staticcheck@2026.1)
  - [x] 4.4 `grep -n "Rule 19" project-context.md` → 3 matches (line 7 Last Updated, line 517 section heading, line 552 in-line workaround example) — exceeds the spec's "two matches" expectation in a benign way (the third is the literal comment shown in the workaround code block)

- [x] Task 5: Update sprint-status.yaml (AC: #1)
  - [x] 5.1 Marked `retro-9-AI5-package-dependency-boundaries: in-progress` at start; will move to `review` at completion (final `done` belongs to /code-review per dev-story workflow)

## Dev Notes

### Root Cause

Epic 9 retro identified that during `9-2b-ai-subtitle-translation` implementation, the dev agent discovered the `subtitle ↔ services` circular import constraint **at implementation time** and had to invent a workaround on the spot: mirror `subtitle.SubtitleBlock` as `services.TranslationBlock` and inline an SRT parser rather than reuse `subtitle.ParseSRT`. Code review (retro-9-AI3 precedent) caught and documented it with comments (L1 finding in 9-2b retro) but the rule isn't in project-context.md — so the **next** developer hitting this will retrace the same discovery path.

Retro insight #3: "Package dependency boundaries undocumented — subtitle↔engine circular import constraint was discovered during 9-2b implementation. Forced inline SRT parser instead of reusing subtitle.ParseSRT."

Retro action AI-5 (owner: QD Barry, priority: LOW): "Document package dependency boundaries — success criteria: project-context.md has package dependency section."

### Current Import Graph (verified 2026-04-13)

```
ai         → (no internal deps — LEAF)
models     → (no internal deps — LEAF)
sse        → (LEAF per convention)
repository → models, retry
services   → ai, cache, config, database, health, images, learning, media, metadata,
             models, parser, qbittorrent, repository, retry, secrets, sse, testutil, tmdb
             ❌ must NOT import: subtitle
subtitle   → models, repository, services, sse
handlers   → database, events, health, learning, media, metadata, models, parser,
             qbittorrent, repository, retry, services, sse, subtitle, tmdb
```

Key cycle point:
- `apps/api/internal/subtitle/engine.go:61` — `terminologyService services.TerminologyCorrectionServiceInterface`
- `apps/api/internal/subtitle/engine.go:90` — `SetTerminologyService(svc services.TerminologyCorrectionServiceInterface)`

Since `subtitle → services`, adding `services → subtitle` would create `services → subtitle → services` — Go compiler rejects this with `import cycle not allowed`.

### Why Only services → subtitle Is Forbidden

- `handlers → subtitle` is **allowed and used** (e.g., `subtitle_handler.go:27,39` uses `*subtitle.Placer`). Handlers never reach back up the graph.
- `ai → subtitle` is not forbidden structurally but `ai` has zero internal deps today (intentional leaf) and should stay that way.
- `repository → subtitle` would also be forbidden (repository is below services in the layer — see Rule 4).
- Only `services → subtitle` is specifically cycle-forming because `subtitle → services` already exists.

### Workaround Pattern (Reference Implementation)

Already in production as of Epic 9:

**Type mirror** (`services/translation_service.go:30-39`):
```go
// TranslationBlock represents a subtitle block for the translation service.
// Mirrors subtitle.SubtitleBlock but is a separate type because subtitle.Engine
// imports services (for TerminologyCorrectionServiceInterface), creating a cycle
// if services also imported subtitle.
type TranslationBlock struct {
    Index int
    Start string
    End   string
    Text  string
}
```

**Inline parser** (`services/transcription_service.go:362-369`):
```go
// srtTimestampPattern matches SRT timestamp lines: 00:00:01,000 --> 00:00:04,000
// Mirrors subtitle.ParseSRT's validation to reject malformed timestamps.
var srtTimestampPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)

// parseSRTToTranslationBlocks parses SRT content into TranslationBlocks.
// Inline implementation — cannot import subtitle package due to circular dependency:
// subtitle.Engine imports services.TerminologyCorrectionServiceInterface.
func parseSRTToTranslationBlocks(content string) ([]TranslationBlock, error) { ... }
```

Task 2 will slim these to single-line cross-refs pointing at the new Rule 19.

### Enforcement Test Design

A doc rule without a test rots. Task 3 adds a dependency-free AST walk:

```go
// apps/api/internal/boundaries_test.go (sketch)
func TestServicesMustNotImportSubtitle(t *testing.T) {
    fset := token.NewFileSet()
    pkgs, err := parser.ParseDir(fset, "services", nil, parser.ImportsOnly)
    require.NoError(t, err)
    for _, pkg := range pkgs {
        for filename, file := range pkg.Files {
            for _, imp := range file.Imports {
                path := strings.Trim(imp.Path.Value, `"`)
                if path == "github.com/vido/api/internal/subtitle" {
                    t.Errorf("%s imports %s — violates Rule 19 (services ↛ subtitle). See project-context.md Rule 19 for the mirror-types workaround.", filename, path)
                }
            }
        }
    }
}
```

Uses stdlib only: `go/parser`, `go/token`, `go/ast`, `strings`, `testing`. No `golang.org/x/tools/go/packages` dependency needed for this narrow check.

**Sanity test** (AC #6): inject a synthetic source string containing the bad import via `parser.ParseFile` and assert the rule fires. This prevents the test from silently passing if the parsing logic breaks.

### Alternatives Considered (and Rejected)

- **Break the existing `subtitle → services` cycle by extracting `TerminologyCorrectionServiceInterface` to a new `contracts` package.** Clean in theory but high-churn (renames across `subtitle/engine.go`, `services/terminology_service.go`, all wiring in `main.go`, all tests). LOW-priority retro item doesn't justify it. Revisit if a second cycle appears.
- **Use `go vet` or `staticcheck` for this.** Neither has a per-path import policy check out of the box. Custom tools (`depguard`, `gocyclo`) would require adding a linter. Overkill for one rule.
- **Put the rule in architecture/*.md (sharded docs) instead of project-context.md.** project-context.md is explicitly "the bible I plan and execute against" per agent activation step 5 — rules that block the dev agent belong here, not in architecture reference docs.
- **Lift `subtitle` into its own module (`libs/subtitle/`).** Go modules would enforce the boundary via build graph. But subtitle is tightly coupled to `services`, `repository`, `sse` — extraction cost is enormous. Not for a LOW retro item.

### Files to Modify

| File | Change |
|------|--------|
| `project-context.md` | Insert new "Rule 19: Package Dependency Boundaries" after Rule 18; update "Last Updated" header |
| `apps/api/internal/services/translation_service.go` | Slim cycle comment (lines 30-33) to reference Rule 19 |
| `apps/api/internal/services/transcription_service.go` | Slim cycle comment (lines 362-368) to reference Rule 19 |
| `apps/api/internal/boundaries_test.go` | NEW — `TestServicesMustNotImportSubtitle` + positive-path sanity test |

### What NOT to Change

- **`apps/api/internal/subtitle/engine.go`** — the `services.TerminologyCorrectionServiceInterface` dependency is deliberate (subtitle needs AI-based terminology correction). Don't refactor this to break the cycle from the other direction.
- **The mirror-type / inline-parser implementations** — these are working code. Only the comments change.
- **`go.mod`** — no new deps; Task 3 uses stdlib `go/parser`.
- **CI** — no workflow changes. The new test runs as part of `go test ./...` already in place.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-9-retro-2026-04-10.md:82-83`] — Insight #3: undocumented dependency boundaries
- [Source: `_bmad-output/implementation-artifacts/epic-9-retro-2026-04-10.md:116`] — AI-5 action
- [Source: `_bmad-output/implementation-artifacts/9-2b-ai-subtitle-translation.md:130`] — L1 code-review finding that added the existing cycle comments
- [Source: `apps/api/internal/subtitle/engine.go:61,90`] — the cycle-forming import
- [Source: `apps/api/internal/services/translation_service.go:30-39`] — type mirror reference
- [Source: `apps/api/internal/services/transcription_service.go:362-369`] — inline parser reference
- [Source: `project-context.md:247-252`] — existing Rule 4 (layered architecture) that this rule extends
- [Source: `project-context.md:473-498`] — Rule 18 (for insertion point reference)

### Previous Story Intelligence (retro-9-AI3 + retro-9-AI4)

- retro-9-AI3 added `staticcheck ./...` to CI — the new `boundaries_test.go` MUST pass staticcheck too. Use simple stdlib APIs (no deprecated patterns).
- retro-9-AI4 added `pnpm lint:all` — Task 4 uses this command, relies on AI-4 being merged first. If this story lands before AI-4, fall back to `pnpm run lint && pnpm run format:check && pnpm nx run-many -t lint`.
- Both prior retro stories (AI3, AI4) are QD-flavored — docs + small config/test additions, no runtime code. This story follows the same pattern.

### Git Intelligence

Recent commits relevant to this story:

- `74a1ac5 fix(ci): address code review findings for retro-9-AI3` — the staticcheck pin that Task 3's new test must not violate
- `281c06a feat: add dead code detection to CI — staticcheck, go vet, ESLint error` — new boundaries_test.go must not accumulate dead code

### Testing Approach

Primary verification is the new Go test itself:

1. **Positive path (AC #5):** `TestServicesMustNotImportSubtitle` → PASS on current main (no violations exist today)
2. **Sanity path (AC #6):** `TestServicesMustNotImportSubtitle_detectsViolation` with synthetic source string → PASS (rule fires when violation present)
3. **Regression (AC #6):** `go test ./...` from `apps/api/` → 0 failures

No new frontend behavior. No API changes. UX verification not applicable.

### UX Verification

SKIPPED — no UI changes in this story.

### Risk & Rollback

- **Risk**: Future refactor moves files between packages and triggers the test false-positively. Mitigation: the test walks the directory literally — if someone moves `translation_service.go` out of `services/`, the walk simply won't inspect it. Low risk.
- **Risk**: The test path `"services"` is relative to CWD. Mitigation: test must use a resolvable path. The sketch uses `parser.ParseDir(fset, "services", ...)` which works because `go test` runs from `apps/api/internal/` when the test file lives there. Alternative: use `runtime.Caller` to anchor the path. Pick whichever is simpler during implementation.
- **Rollback**: Revert 4 files — `project-context.md`, the two service comments, delete `boundaries_test.go`. No schema or data changes.

## Dev Agent Record

### Agent Model Used

claude-opus-4-6 (1M context) — BMM dev agent "Amelia"

### Debug Log References

- staticcheck@2026.1 SA1019 deprecation hit on first commit of `boundaries_test.go` (used `parser.ParseDir` per story sketch). Refactored to `os.ReadDir` + `parser.ParseFile` per file — still stdlib-only (no `golang.org/x/tools/go/packages` dep), no deprecation, behaviorally identical. Both tests still pass.

### Completion Notes List

- ✅ Rule 19 added to `project-context.md` after Rule 18; `Last Updated` header revised to call out Rule 19. New section covers: import-direction summary, `services ↛ subtitle` FORBIDDEN line, Known Cycle Points (with `subtitle/engine.go:61,90` citations), Workaround Pattern (mirror types, 4 steps, with reference impl line numbers), Leaf packages list, Enforcement subsection pointing at `boundaries_test.go`.
- ✅ Cycle comments slimmed to single-line cross-refs in `translation_service.go:30` and `transcription_service.go:362-363`. `srtTimestampPattern` regex comment preserved (validation rationale, separate concern).
- ✅ `boundaries_test.go` created at `apps/api/internal/boundaries_test.go` (`package internal`). Two tests: `TestServicesMustNotImportSubtitle` (positive — walks `internal/services/` via stdlib `os.ReadDir` + `go/parser`, fails on `github.com/vido/api/internal/subtitle` import) and `TestServicesMustNotImportSubtitle_detectsViolation` (sanity — synthetic source proves the rule fires, prevents vacuous pass). Both PASS on current main.
- ✅ Lint gate: `pnpm lint:all` → 0 errors (108 pre-existing warnings unchanged). Includes `go vet`, staticcheck@2026.1, ESLint, Prettier.
- ✅ Regression gate (Epic 9 retro AI-1): `pnpm nx test api` (Go) and `pnpm nx test web` (132 files / 1629 tests) — ALL PASS. Test cleanup auto-ran successfully (no orphaned vitest workers).
- ⚠️ One deviation from story spec: story sketch used `parser.ParseDir` (SA1019-deprecated in Go 1.25). Changed to `os.ReadDir` + `parser.ParseFile` per-file walk — same outcome, satisfies staticcheck. Documented in Debug Log.
- 🎨 UX Verification: SKIPPED — no UI changes in this story.

### File List

- `project-context.md` — added Rule 19 section (~60 lines after Rule 18); updated `Last Updated` header on line 7. TA pass: corrected leaf list (removed 3 wrong entries), updated Enforcement subsection to list all 5 invariant tests, exported function name reference
- `apps/api/internal/services/translation_service.go` — slimmed `TranslationBlock` cycle comment from 4 lines to 1 (line 30)
- `apps/api/internal/services/transcription_service.go` — slimmed `parseSRTToTranslationBlocks` cycle comment from 3 lines to 1 (lines 362-363); TA pass: exported `parseSRTToTranslationBlocks` → `ParseSRTToTranslationBlocks` so the parity test can call it from package internal
- `apps/api/internal/services/transcription_translation_test.go` — TA pass: updated 8 callsites for the export rename
- `apps/api/internal/boundaries_test.go` — NEW; `package internal` with `TestServicesMustNotImportSubtitle` + `TestForbiddenImportEdges` (3 subtests) + `TestLeafPackagesHaveNoInternalDeps` (5 subtests). CR pass: extracted pure `scanImports` helper, replaced tautological sanity test with `TestScanImports_DetectsViolation` (exercises helper on a tempdir fixture), added external-test-package skip, captured stderr in `go list` invocation, removed dead `t.Skip` branch, rewrote dead-comment
- `apps/api/internal/services/srt_parity_test.go` — NEW (CR pass); `package services_test` (moved from `apps/api/internal/srt_parity_test.go` per CR M2 — locality + avoids false-positive from assertNoImport). `TestParseSRT_ParityWithSubtitle` with 10 fixture subtests, now using an ordered slice of structs for deterministic failure output
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `retro-9-AI5-package-dependency-boundaries`: ready-for-dev → in-progress → review → done

### Change Log

- 2026-04-13: Implemented retro-9-AI5 — Package Dependency Boundaries (Rule 19) + AST-based enforcement test (`services ↛ subtitle`). All 6 ACs satisfied. Lint + full Go/web regression green.
- 2026-04-13: TA (Master Test Architect Murat) expansion on top of base story:
  - **R1 (HIGH):** Risk scan caught a documentation bug — Rule 19's leaf list contained `secrets` (depends on `crypto`), `logger` (depends on `models`/`retry`/`repository`), and `errors` (no such package). Corrected the leaf list to the verified 5 (`ai`, `models`, `sse`, `retry`, `cache`) and added `TestLeafPackagesHaveNoInternalDeps` (uses `go list -deps` per claimed leaf, self-skips if `go` binary unavailable) so the doc claim cannot silently rot.
  - **R2 (MEDIUM):** Added `TestParseSRT_ParityWithSubtitle` in `apps/api/internal/srt_parity_test.go` (`package internal`) — runs both `subtitle.ParseSRT` and `services.ParseSRTToTranslationBlocks` against 10 shared fixtures (basic/empty/multi-line/BOM/CRLF/CR/blank-lines/unicode/malformed-timestamp). Exported `parseSRTToTranslationBlocks` → `ParseSRTToTranslationBlocks` to enable the cross-package call (Mirror-Types parity inherently requires importing both packages, only possible at the `internal` package level). Catches Mirror-Types drift that Rule 19 Step 4 currently relies on human review for.
  - **R3 (LOW):** Added `TestForbiddenImportEdges` (3 subtests) covering the other forbidden directions documented in Rule 19 (`services↛handlers`, `repository↛services`, `repository↛subtitle`). Today blocked transitively by Go's import-cycle compiler check, but the test encodes the architectural intent for future-proofing.
  - All 19 new test cases (5 funcs across 2 files) pass; lint:all + full Go/web regression remain green; refactored `boundaries_test.go` to share an `assertNoImport` helper.
- 2026-04-13: Code-review pass (Amelia /code-review) — 7 findings triaged and fixed:
  - **H1 (HIGH) — Rule 19 contradicted Rule 4**: the allowed-directions list in Rule 19 included `Handler → Repository (read-only paths)` which Rule 4 explicitly forbids (line 251). Dropped the line and added an explicit NOTE reaffirming that Rule 19 does not introduce an exception to Rule 4.
  - **H2 (HIGH) — Sanity test was tautological**: `TestServicesMustNotImportSubtitle_detectsViolation` re-implemented `parser.ParseFile` inline and did NOT call `assertNoImport`, so it sanity-checked itself rather than the real enforcement code path. Refactored `assertNoImport` → pure `scanImports(dir, forbidden) ([]violations, scanned, err)`; replaced the old sanity test with `TestScanImports_DetectsViolation` which plants a good.go + bad.go + external-test x_test.go in a tempdir and asserts scanImports flags exactly bad.go (covers both the violation path AND the M1 external-test-pkg skip in one test).
  - **M1 (MEDIUM) — False-positive risk for external test packages**: `assertNoImport` walked every .go file including `*_test.go`. External test packages (`package foo_test`) are a separate compilation unit and can legitimately import peer packages that production code cannot (which is why the parity test lives in one). scanImports now skips files whose package name ends in `_test`; covered by the new sanity test above.
  - **M2 (MEDIUM) — Parity test lived in wrong package**: moved `srt_parity_test.go` from `apps/api/internal/` (package internal) → `apps/api/internal/services/` (package services_test). Better locality (parity test sits next to the code it's testing), possible only after M1 fix so `assertNoImport` does not false-flag the file. `ParseSRTToTranslationBlocks` remains exported (external test packages cannot access unexported identifiers — this is a documented trade-off in the function doc comment and in Rule 19 reference block).
  - **M3 (MEDIUM) — Non-deterministic fixture iteration**: parity test used `map[string]string` for its 10 fixtures → Go's map iteration order is randomized → failure logs were non-deterministic. Switched to an ordered `[]struct{name, input string}` for reproducible failure output.
  - **M4 (MEDIUM) — Silent stderr in leaf invariant test**: `TestLeafPackagesHaveNoInternalDeps` used `exec.Command(...).Output()` which discards stderr. When `go list -deps` fails (e.g., a build error in a claimed leaf) the test reported only `exit status 1` with no underlying reason. Now captures stderr via `cmd.Stderr = &bytes.Buffer{}` and includes it in the failure message.
  - **L1 (LOW) — Dead code path**: `TestLeafPackagesHaveNoInternalDeps` guarded on `exec.LookPath("go")` and `t.Skip`-ed if missing. But the test is executed BY `go test`, so `go` is tautologically on PATH. Removed the guard.
  - **L2 (LOW) — Misleading comment**: `assertNoImport` claimed sub-packages "have their own context" as justification for not recursing, but services/handlers/repository have no sub-packages today. Rewrote the comment to state what scanImports actually filters (directories, non-.go, external test packages) and why.
  - **Updated `project-context.md` Last Updated header** to reflect this CR pass and the new enforcement structure (scanImports helper, 4 tests in boundaries_test.go + 1 parity test in services_test). Updated the "Reference Implementation" block to explain why `ParseSRTToTranslationBlocks` is exported (external-test-pkg parity only, not for runtime callers).
  - All tests still pass: `go test ./internal/ ./internal/services/` → 4 funcs + 1 parity func, 20 subtests. `pnpm lint:all` → 0 errors (108 pre-existing warnings unchanged). `pnpm nx test api` → full Go regression PASS.
