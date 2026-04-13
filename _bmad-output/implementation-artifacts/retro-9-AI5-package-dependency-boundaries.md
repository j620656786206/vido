# Story: Document Package Dependency Boundaries

Status: ready-for-dev

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

- [ ] Task 1: Add "Rule 19: Package Dependency Boundaries" to `project-context.md` (AC: #1, #2, #3)
  - [ ] 1.1 Insert a new Rule 19 after Rule 18 (API Boundary Case Transformation, lines ~473-498). Keep the existing Rule 19+ numbering stable by only appending
  - [ ] 1.2 Open with a summary import-direction rule:
    ```
    Handler → Service → Repository → Database (Rule 4 already)
    Handler → Subtitle → Service (allowed, single-direction)
    Service ↛ Subtitle    (FORBIDDEN — would cycle with subtitle → services.TerminologyCorrectionServiceInterface)
    ```
  - [ ] 1.3 Include a "Known Cycle Points" subsection listing the real cycle in the codebase today:
    - `subtitle/engine.go:61,90` imports `services.TerminologyCorrectionServiceInterface`
    - Therefore no file under `internal/services/` may `import "github.com/vido/api/internal/subtitle"`
  - [ ] 1.4 Add "Workaround Pattern: Mirror Types" subsection with:
    - Step 1: Define a minimal local type in `services/` (e.g., `TranslationBlock`) that mirrors only the fields you need from the `subtitle` type
    - Step 2: Inline the minimum logic you need (e.g., `parseSRTToTranslationBlocks`). Match the source package's validation rules (regex, error handling) so behavior stays identical
    - Step 3: Add a comment citing the cycle and project-context.md Rule 19
    - Step 4: Reference implementation: `apps/api/internal/services/translation_service.go:30-39` (type mirror) and `apps/api/internal/services/transcription_service.go:362-369` (inline parser)
  - [ ] 1.5 Add a brief list of "Leaf packages (no internal deps — safe to import from anywhere):" → `ai`, `models`, `sse`, `retry`, `cache`, `secrets`, `errors`, `logger`. These are always safe targets
  - [ ] 1.6 Update the "Last Updated" header (line ~7) to mention Rule 19 addition

- [ ] Task 2: Slim existing cycle comments to reference the new rule (AC: #4)
  - [ ] 2.1 In `apps/api/internal/services/translation_service.go:30-33`, shorten to: `// TranslationBlock mirrors subtitle.SubtitleBlock. services ↛ subtitle — see project-context.md Rule 19.`
  - [ ] 2.2 In `apps/api/internal/services/transcription_service.go:362-368`, shorten the `parseSRTToTranslationBlocks` comment block similarly: `// Inline SRT parser. services ↛ subtitle — see project-context.md Rule 19. Mirrors subtitle.ParseSRT validation.`
  - [ ] 2.3 Keep the `srtTimestampPattern` regex comment (line 363) — it explains the WHY of validation, not the cycle

- [ ] Task 3: Add a Go test enforcing the `services ↛ subtitle` rule (AC: #5, #6)
  - [ ] 3.1 Create `apps/api/internal/boundaries_test.go` in the `internal` package. Use a package-level test because import rules span the whole module
  - [ ] 3.2 Implementation approach: use `golang.org/x/tools/go/packages` OR — simpler and dependency-free — walk `internal/services/` with `go/parser.ParseDir`, iterate each file's import specs, and fail if any path equals `github.com/vido/api/internal/subtitle`
  - [ ] 3.3 Prefer the dependency-free approach: `go/parser` + `go/ast` are in the stdlib, no new `go.mod` entries needed
  - [ ] 3.4 Test name: `TestServicesMustNotImportSubtitle`. On violation, report the offending file + import line with a message pointing at project-context.md Rule 19
  - [ ] 3.5 Run the test locally: `cd apps/api && go test ./internal/ -run TestServicesMustNotImportSubtitle` → MUST pass (no current violation)
  - [ ] 3.6 Add a second negative-path sanity test `TestServicesMustNotImportSubtitle_detectsViolation` using an inline synthetic fileset (via `go/parser.ParseFile` with source string) to confirm the rule triggers when the bad import IS present — proves the test isn't vacuously passing

- [ ] Task 4: Verify docs + test end-to-end (AC: #6)
  - [ ] 4.1 Run `pnpm lint:all` (post retro-9-AI4) — MUST pass
  - [ ] 4.2 Run `cd apps/api && go test ./...` — MUST pass including new boundaries test
  - [ ] 4.3 Run `cd apps/api && staticcheck ./...` — MUST pass (no new warnings from the test file)
  - [ ] 4.4 Sanity-check docs: in a fresh terminal, `grep -n "Rule 19" project-context.md` — should return two matches (the header + "Last Updated" reference)

- [ ] Task 5: Update sprint-status.yaml (AC: #1)
  - [ ] 5.1 Change `retro-9-AI5-package-dependency-boundaries: backlog` → `done` with completion note and date once Tasks 1-4 are verified green

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

### Debug Log References

### Completion Notes List

### File List
