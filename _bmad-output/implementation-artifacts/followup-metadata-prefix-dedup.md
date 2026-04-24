# Story: Deduplicate METADATA_ Error Codes — Rule 11 Canonicalization

Status: ready-for-dev

**Origin:** Winston (Architect) architectural review of retro-10-AI3 Rule 7 expansion — 2026-04-20.
**Priority:** MEDIUM (Rule 11 smell affecting live wire contract; not a user-visible bug).
**Scope estimate:** 2–3 Go files touched, ~30–60 LOC delta (may grow if Rule 19 leaf-list amendment triggered), possibly 1 boundaries_test.go amendment + Rule 7 example sync.

## Story

As a Go backend developer maintaining `apps/api/internal/`,
I want every `METADATA_*` wire-contract error code to live exactly once in the canonical `metadata/provider.go` and be imported by consumers,
so that `retry/metadata_integration.go` stops silently expanding the wire contract with 4 undeclared codes and Rule 11 (Interface Location) is satisfied across both packages.

## Problem (preserved from Winston's draft)

`apps/api/internal/metadata/provider.go` is the canonical source of `METADATA_*` wire-contract error codes (package doc comment: _"Common error codes for metadata providers"_). It declares 7 codes at lines 232–247:

```
METADATA_NO_RESULTS, METADATA_TIMEOUT, METADATA_RATE_LIMITED, METADATA_UNAVAILABLE,
METADATA_INVALID_REQUEST, METADATA_ALL_FAILED, METADATA_CIRCUIT_OPEN
```

`apps/api/internal/retry/metadata_integration.go` **violates Rule 11** in two ways:

1. **Lines 9–15** — mirrors 5 of the canonical codes as a sibling `const (...)` block with comment "Metadata error codes from provider.go". This is an explicit acknowledgment of the duplication.
2. **Lines 107, 120, 133, 144, 156, 166, 186, 195** — `ClassifyMetadataError()` constructs `RetryableError` values that introduce **4 new wire codes not declared in provider.go**:
   - `METADATA_GATEWAY_ERROR`
   - `METADATA_NETWORK_ERROR`
   - `METADATA_NOT_FOUND`
   - `METADATA_UNKNOWN_ERROR`

The second item is the more serious defect: the retry package is silently expanding the wire contract beyond what the canonical `metadata` package declares. Frontend/ops see error codes with no authoritative definition.

## Acceptance Criteria

1. Given a reader greps `apps/api/internal/` for `"METADATA_` string literals, when the grep returns hits, then **every quoted `METADATA_*` constant declaration lives in `apps/api/internal/metadata/provider.go`** and nowhere else. **Exemption:** test assertions (any `*_test.go`) and the code-review Rule 7 workflow's inline list. Consumer references go through `metadata.ErrCode*` identifiers, not hard-coded strings.

2. Given `apps/api/internal/retry/metadata_integration.go` after the change, when reading the file, then the local `const (...)` block (currently lines 9–15) is **deleted** and replaced by `import "github.com/vido/api/internal/metadata"` + usage as `metadata.ErrCodeTimeout` (etc.) at both the `strings.Contains(errStr, ...)` call sites and the `RetryableError{Code: ...}` construction sites.

3. Given `ClassifyMetadataError()` in retry after the change, when reading the function, then the four retry-only codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) are **promoted to exported constants in `metadata/provider.go`** (with doc comments following the existing style) and referenced via `metadata.ErrCode*` rather than hard-coded string literals in the retry code path.

4. Given `go test ./...` (or `pnpm nx test api`) runs after the change, when it completes, then all tests pass with **no behavioral drift** — the wire contract values are byte-identical to pre-change (this is a refactor, not a rename). `metadata_integration_test.go`'s hard-coded `"METADATA_*"` string assertions MUST continue to pass unchanged (they test the wire contract, which is preserved).

5. Given `apps/api/internal/boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` runs after the change, when the leaf-list contains `"retry"` and retry now imports `metadata`, then the test WILL fail — because `retry` will no longer be a zero-internal-deps leaf. **Remediation (mandatory in this same commit):** (a) remove `"retry"` from the `leaves` slice in `boundaries_test.go:64`, AND (b) remove `retry` from the **Leaf packages** list in `project-context.md` Rule 19 (line 565), AND (c) update Rule 19's "Allowed" section (line 546) removing `retry` from the `*  → ai, models, sse, retry, cache` enumeration. After amendment, the full test suite must be green.

6. Given the CR auto-fix prefix map in `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` (retro-10-AI3 final placement ~line 149) currently maps `metadata/** or retry/** → METADATA_`, when this story completes, then the map stays as-is (both packages still legitimately emit METADATA_ codes via the canonical constants). No XML edit required. **Optional housekeeping:** the example code list at `project-context.md:295` MAY be extended with the 4 newly-promoted codes for doc completeness, but this is non-blocking.

## Tasks / Subtasks

- [ ] **Task 1: Pre-flight — Confirm Rule 19 leaf-impact (AC #5)**
  - [ ] 1.1 Read `apps/api/internal/boundaries_test.go:63-108` — note the `leaves := []string{"ai", "models", "sse", "retry", "cache"}` slice at line 64 and the `TestLeafPackagesHaveNoInternalDeps` harness (uses `go list -deps` per package).
  - [ ] 1.2 Read `project-context.md` Rule 19 at lines 538–613 — specifically line 546 `*  → ai, models, sse, retry, cache` (Allowed section) and lines 564–565 (Leaf packages declaration). These are the two doc edits required in Task 4.
  - [ ] 1.3 Record in Debug Log: "Rule 19 leaf-list amendment required" (expected — `retry → metadata` breaks the zero-internal-deps property; Winston's AC #5 pre-authorized this).

- [ ] **Task 2: Promote 4 retry-only codes to `metadata/provider.go` (AC #3)**
  - [ ] 2.1 Open `apps/api/internal/metadata/provider.go`, locate the const block at lines 232–247.
  - [ ] 2.2 Append 4 new exported constants AFTER `ErrCodeCircuitOpen` (before the closing `)` at line 247), following the existing doc-comment style:
    ```go
    // ErrCodeGatewayError indicates a bad gateway or gateway timeout from the provider
    ErrCodeGatewayError = "METADATA_GATEWAY_ERROR"
    // ErrCodeNetworkError indicates a network-layer failure (connection refused/reset)
    ErrCodeNetworkError = "METADATA_NETWORK_ERROR"
    // ErrCodeNotFound indicates the requested resource was not found at the provider
    ErrCodeNotFound = "METADATA_NOT_FOUND"
    // ErrCodeUnknownError indicates an unclassified provider error
    ErrCodeUnknownError = "METADATA_UNKNOWN_ERROR"
    ```
  - [ ] 2.3 Verify file compiles: `pnpm nx build api` or `go build ./apps/api/internal/metadata/...` — must pass.
  - [ ] 2.4 **Optional (AC #6 housekeeping):** extend `project-context.md:295` Rule 7 example line by appending `METADATA_GATEWAY_ERROR, METADATA_NETWORK_ERROR, METADATA_NOT_FOUND, METADATA_UNKNOWN_ERROR`. Non-blocking; skip if scope is getting tight.

- [ ] **Task 3: Deduplicate retry's mirror block + route all references through `metadata.ErrCode*` (AC #1, #2)**
  - [ ] 3.1 Open `apps/api/internal/retry/metadata_integration.go`.
  - [ ] 3.2 Delete local `const (...)` block at lines 9–15 (the 5-code mirror).
  - [ ] 3.3 Add `"github.com/vido/api/internal/metadata"` to the import block (line 3–6).
  - [ ] 3.4 Update `IsRetryableMetadataError()` (lines 19–91): replace `ErrCodeTimeout` → `metadata.ErrCodeTimeout`, `ErrCodeRateLimited` → `metadata.ErrCodeRateLimited`, `ErrCodeUnavailable` → `metadata.ErrCodeUnavailable`, `ErrCodeCircuitOpen` → `metadata.ErrCodeCircuitOpen` in the `retryableCodes` slice (lines 28–33). `ErrCodeNoResults` is currently imported into the local const block but not used in this function — verify before deleting.
  - [ ] 3.5 Update `ClassifyMetadataError()` (lines 94–199): replace every hard-coded string at the `Code:` fields with `metadata.ErrCode*` identifiers:
    - line 107 `"METADATA_TIMEOUT"` → `metadata.ErrCodeTimeout`
    - line 120 `"METADATA_RATE_LIMITED"` → `metadata.ErrCodeRateLimited`
    - line 133 `"METADATA_UNAVAILABLE"` → `metadata.ErrCodeUnavailable`
    - line 144 `"METADATA_CIRCUIT_OPEN"` → `metadata.ErrCodeCircuitOpen`
    - line 156 `"METADATA_GATEWAY_ERROR"` → `metadata.ErrCodeGatewayError` **(newly promoted)**
    - line 166 `"METADATA_NETWORK_ERROR"` → `metadata.ErrCodeNetworkError` **(newly promoted)**
    - line 176 `"METADATA_NO_RESULTS"` → `metadata.ErrCodeNoResults`
    - line 186 `"METADATA_NOT_FOUND"` → `metadata.ErrCodeNotFound` **(newly promoted)**
    - line 195 `"METADATA_UNKNOWN_ERROR"` → `metadata.ErrCodeUnknownError` **(newly promoted)**
  - [ ] 3.6 Update the `strings.Contains(errStr, ErrCode*)` pattern-match call sites inside `ClassifyMetadataError()` (lines 104, 116, 129, 142, 174): replace with `metadata.ErrCode*` identifiers so pattern matching still works. **Note:** `strings.Contains` is case-insensitive here via `strings.ToLower` wrapping at line 99 — no behavior change.
  - [ ] 3.7 Update `ShouldQueueRetry()` (line 234) and any other `ErrCodeNoResults` references — replace with `metadata.ErrCodeNoResults`.
  - [ ] 3.8 Run `goimports -w apps/api/internal/retry/metadata_integration.go` (or equivalent) to clean up imports.
  - [ ] 3.9 Verify `go build ./apps/api/internal/retry/...` — must pass.

- [ ] **Task 4: Amend Rule 19 leaf list — remove `retry` (AC #5)**
  - [ ] 4.1 Open `apps/api/internal/boundaries_test.go`, line 64. Change `leaves := []string{"ai", "models", "sse", "retry", "cache"}` → `leaves := []string{"ai", "models", "sse", "cache"}`.
  - [ ] 4.2 Open `project-context.md` Rule 19, line 546. Change `*  → ai, models, sse, retry, cache  (leaf packages — see list below)` → `*  → ai, models, sse, cache  (leaf packages — see list below)`.
  - [ ] 4.3 Open `project-context.md` Rule 19, lines 564–565. Change `Leaf packages (zero internal deps — always safe to import from anywhere):\n  ai, models, sse, retry, cache` → `Leaf packages (zero internal deps — always safe to import from anywhere):\n  ai, models, sse, cache`. Adjust the `Verified 2026-04-13` commentary to note `retry` was demoted on 2026-04-24 when `retry` legitimately acquired `metadata` as an internal dep (per this story).
  - [ ] 4.4 Update `project-context.md` "Last Updated" header at line 7 with a retro-10-followup citation (pattern: same as retro-10-AI4 Rule 15 extension note).
  - [ ] 4.5 Run `pnpm nx test api` — `TestLeafPackagesHaveNoInternalDeps` must pass with the updated `leaves` slice.

- [ ] **Task 5: Full regression gate + sprint-status sync (AC #4)**
  - [ ] 5.1 `pnpm lint:all` — expected 0 errors (Go code edits + doc edits; Prettier/go-vet checks apply).
  - [ ] 5.2 `pnpm nx test api` — expected PASS. Key sub-checks:
    - `metadata_integration_test.go` — 16+ test cases asserting hard-coded `"METADATA_*"` strings; MUST all pass (wire contract byte-identical).
    - `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` — MUST pass with updated `leaves` slice (Task 4.1).
    - `boundaries_test.go::TestForbiddenImportEdges` — MUST pass (no new forbidden edges introduced; retry → metadata is allowed because metadata doesn't depend on any of retry's parents).
    - `metadata_handler_test.go` — 1 test asserts `"METADATA_INVALID_REQUEST"` (line 158); MUST pass unchanged.
  - [ ] 5.3 `pnpm nx test web` — expected PASS (zero frontend code change; wire values byte-identical means FE contract untouched).
  - [ ] 5.4 Update `_bmad-output/implementation-artifacts/sprint-status.yaml` entry `followup-metadata-prefix-dedup`: transition `backlog → ready-for-dev → in-progress → review`. Final `review → done` is CR's responsibility (per retro-10-AI5 pattern).

## Dev Notes

### Root Cause

Winston's retro-10-AI3 Item 1 ruling (2026-04-20) surfaced during the code-review-dogfooding pass that Amelia ran on Story 10-AI3: the Rule 7 grep extension discovered **17 production `METADATA_*` code string literals** across two packages — 7 canonical in `metadata/provider.go` and 10 in `retry/metadata_integration.go` (of which 5 are mirrors and 5 are silent wire-contract expansions). Winston's verdict: the mirror violates Rule 11 (Interface Location) in spirit, and the 4 new codes violate the "canonical source of truth" invariant implicit in Rule 7's authoritative prefix set design.

### ⚠️ Rule 19 Decision Point (Winston's pre-authorization)

`retry` is currently a leaf package per Rule 19 (zero internal-only deps, enforced by `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps`). Adding `import "github.com/vido/api/internal/metadata"` to `retry/metadata_integration.go` will **necessarily break the leaf property**. Winston's AC #5 explicitly pre-authorizes the remediation: remove `retry` from the leaf list in both `boundaries_test.go:64` and `project-context.md:546, 565` in the same commit.

**Why this is acceptable:** `retry` is still a low-level utility package. Its only new internal dep is `metadata`, which itself is a thin leaf-adjacent package (depends only on `models`). The transitive impact on retry's consumers (`repository/`, `handlers/`, `services/`) is zero — they all already import `metadata` or could. The "leaf" claim was a cleanliness assertion, not a hard architectural constraint.

**Alternative rejected:** moving `metadata_integration.go` out of `retry` into `metadata/retry_classifier.go` (flipping the import direction `metadata → retry`). This would keep `retry` a leaf but would require updating all 5 retry-package importers + renaming the adapter in call sites. Winston chose the lighter-touch fix: amend the leaf list.

### Architecture Constraint Summary

| Rule | Current state | After this story |
|---|---|---|
| Rule 7 (Error Codes) | METADATA_ prefix listed; 11 production codes (7 canonical + 4 undeclared in retry) | All 11 codes canonical in `metadata/provider.go`; retry emits via `metadata.ErrCode*` identifiers |
| Rule 11 (Interface Location) | VIOLATED — retry mirrors 5 canonical codes + expands 4 silently | SATISFIED — retry imports, does not redeclare |
| Rule 19 (Leaf Packages) | `retry` listed as leaf in both code + doc | `retry` removed from leaf list (both `boundaries_test.go:64` + `project-context.md:546,565`) |

### Cross-Stack Split Check (Agreement 5, Epic 8 Retro + Epic 9c Retro AI-1 enforced)

- **Backend task count:** 5 (Task 1 pre-flight + Task 2 provider.go edit + Task 3 retry refactor + Task 4 Rule 19 amendment + Task 5 regression gate).
- **Frontend task count:** 0 (zero React/TS files touched; wire contract byte-identical).
- **Threshold:** both counts ≤3 → FE=0 passes, BE=5 exceeds threshold → **SPLIT CHECK: does NOT trigger** (the split rule triggers only when BOTH counts > 3; single-side heavy work is OK). ✅

### Precedent Stories (shape + pattern to mirror)

- **`retro-10-AI3-rule7-wire-format-cr-check.md`** (2026-04-20, done) — the meta-story whose dogfooding surfaced this defect. Its CR H1 finding (Rule 7 prefix list stale vs codebase) documents the 17-code/4-prefix discovery that seeded Winston's follow-up rulings. This story is the materialization of Item 1 of those rulings.
- **`retro-10-AI5-ac-contract-versioning.md`** (2026-04-22, done) — shape precedent for non-standard `followup-*` / `retro-*` story keys. Same sprint-status transition pattern (`backlog → ready-for-dev → in-progress → review → done`).
- **`followup-qbittorrent-prefix-rename.md`** (backlog, sister story) — Winston Item 3 ruling; same epoch + same "pure refactor, wire contract preserved" shape but targets QB_ prefix rename. Not a blocker; can run independently.

### Grep Patterns (for DEV to use during implementation)

```bash
# Confirm canonical: every METADATA_* string literal before change
grep -rn '"METADATA_' apps/api/internal/ --include="*.go"

# After Task 2: verify 4 new constants added to provider.go
grep -n 'ErrCodeGatewayError\|ErrCodeNetworkError\|ErrCodeNotFound\|ErrCodeUnknownError' apps/api/internal/metadata/provider.go

# After Task 3: retry/metadata_integration.go should have ZERO "METADATA_ string literals
grep -n '"METADATA_' apps/api/internal/retry/metadata_integration.go
# Expected output: empty

# After Task 3: retry should import metadata
grep -n 'internal/metadata' apps/api/internal/retry/metadata_integration.go
# Expected: one import line

# Test files exempt per AC #1 — should still have hardcoded strings
grep -c '"METADATA_' apps/api/internal/retry/metadata_integration_test.go
# Expected: >0 (unchanged)
```

### Risk Assessment

- **Behavioral-drift risk:** ZERO. Pure refactor; wire contract values byte-identical (AC #4). `metadata_integration_test.go`'s hardcoded string assertions serve as the drift detector — if any ErrCode* value accidentally changes, tests fail.
- **Leaf-list regression risk:** LOW. Task 4 is explicit + covered by `TestLeafPackagesHaveNoInternalDeps`. Forgetting to amend the slice will cause the test to fail hard with a clear error message referencing `retry`.
- **Import-cycle risk:** ZERO. Verified: `metadata` imports `models` + stdlib only. `retry` currently imports only stdlib. Adding `retry → metadata` introduces no cycle (metadata does not transitively depend on retry).
- **Downstream consumer risk:** ZERO. 5 packages import retry (`repository/`, `handlers/retry_handler.go`, `services/retry_service.go`, `services/metadata_service.go`, `repository/retry_repository.go`). None of these depend on retry being a leaf; they depend on retry's exported API, which is unchanged.
- **Rule 7 check rot:** ZERO. Rule 7's authoritative prefix set at `project-context.md:300` is unchanged (METADATA_ prefix already in the list). 4 new codes fall under the existing prefix. The code-review Rule 7 inline grep will correctly classify them as `metadata/**` or `retry/**` → METADATA_ prefix (AC #6 already confirmed).

### Out of Scope

- **Renaming `METADATA_*` codes.** The prefix is correct per Winston's Item 2 approval. Only deduplication, no rename.
- **Restructuring `ClassifyMetadataError()` logic.** The string-matching classifier is preserved verbatim; only the constant-reference mechanism changes.
- **Touching provider implementations under `metadata/tmdb/`, `metadata/douban/`, `metadata/wikipedia/`.** Those already use their source-specific prefixes (TMDB_, DOUBAN_, WIKIPEDIA_) and are unaffected.
- **QB_ prefix rename (sister follow-up `followup-qbittorrent-prefix-rename`).** Independent story; can run in parallel.
- **Code-review `instructions.xml` auto-fix prefix map update** (AC #6 explicitly confirmed no edit required).
- **Epic 11+ scope.** This is a retro-10 follow-up, not Epic 11 prep work.

## References

- [Source: `apps/api/internal/metadata/provider.go:232-247`] — canonical ErrCode* declarations (destination of Task 2)
- [Source: `apps/api/internal/retry/metadata_integration.go:9-15, 107, 120, 133, 144, 156, 166, 176, 186, 195`] — offending mirror + silent expansion sites (target of Task 3)
- [Source: `apps/api/internal/retry/metadata_integration_test.go:23-305`] — 16+ hardcoded `"METADATA_*"` assertions (regression gate for Task 5.2)
- [Source: `apps/api/internal/boundaries_test.go:63-108`] — `TestLeafPackagesHaveNoInternalDeps` enforcement of Rule 19 leaf list (amendment site Task 4.1)
- [Source: `project-context.md#rule-11-interface-location` line 328-336] — Rule 11 principle being satisfied
- [Source: `project-context.md#rule-19-package-dependency-boundaries` lines 538-613] — Rule 19 leaf list + amendment sites (Task 4.2–4.4)
- [Source: `project-context.md#rule-7-error-codes-system` lines 279-300] — Rule 7 authoritative prefix set (unchanged; optional example extension in Task 2.4)
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI3-rule7-wire-format-cr-check.md`] — dogfooding source that surfaced this defect + Winston's prompt generation
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml` line 447] — current backlog entry `followup-metadata-prefix-dedup: backlog` (transitions to `ready-for-dev` on this story save)
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI5-ac-contract-versioning.md`] — precedent for non-standard `followup-*` story-key handling

## Dev Agent Record

### Agent Model Used

_(Populated by DEV Amelia on `/dev-story` invocation.)_

### Debug Log References

_(Populated by DEV during Task 1 pre-flight + Task 5 regression gate.)_

### Completion Notes List

_(Populated by DEV. Must include:)_

- `🔗 AC Drift: NONE | FOUND | N/A` (retro-10-AI2 pattern; expected N/A — no upstream story-ID references)
- `🔒 Rule 7 Wire Format (self-result): N/A` (retro-10-AI3 pattern; no new prefix introduced — METADATA_ already canonical)
- `📎 Contract Stamps: N/A` (retro-10-AI5 pattern; no AC in this story is `[@contract-v*]` stamped — no cross-story contract commitment)
- `🎨 UX Verification: SKIPPED` (zero files under `apps/web/`)
- Per-AC satisfaction notes citing final line ranges

### File List

_(Populated by DEV. Expected files:)_

- `apps/api/internal/metadata/provider.go` — Task 2 (4 new ErrCode* constants appended to existing block at lines 232–247)
- `apps/api/internal/retry/metadata_integration.go` — Task 3 (local const block deleted, metadata import added, 13+ hard-coded strings replaced with `metadata.ErrCode*` references)
- `apps/api/internal/boundaries_test.go` — Task 4.1 (line 64: `"retry"` removed from `leaves` slice)
- `project-context.md` — Task 4.2–4.4 (line 546 + lines 564–565 Rule 19 amendments; line 7 "Last Updated" note; optional line 295 Rule 7 example extension if Task 2.4 executed)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — Task 5.4 (`followup-metadata-prefix-dedup` transitions + final comment with line ranges)
- `_bmad-output/implementation-artifacts/followup-metadata-prefix-dedup.md` — this story file (status transitions + Change Log entries)

## Change Log

| Date       | Change                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-20 | Story draft created by Winston (Architect) during retro-10-AI3 architectural review. Initial 6 ACs + Problem + Out of Scope + References. Priority MEDIUM, scope ~30 LOC. Status: `backlog`.                                                                                                                                                                                           |
| 2026-04-24 | Story bootstrapped to `ready-for-dev` by SM Bob via `/bmad:bmm:workflows:create-story` (yolo mode). Added: Story statement (As-a/I-want/So-that), 5 Tasks with 22+ subtasks mapped to ACs #1-6, Dev Notes with Rule 19 Decision Point + Architecture Constraint table + Cross-Stack Split Check (5 BE + 0 FE → single story, pass) + Precedent Stories + Grep Patterns + Risk Assessment (all 5 risk categories ZERO-LOW), File List scaffolding, Dev Agent Record placeholder (retro-10-AI2/AI-3/AI-5 audit-line pattern). Exhaustive artifact analysis: re-read `metadata/provider.go`, `retry/metadata_integration.go`, `retry/metadata_integration_test.go`, `boundaries_test.go`, `project-context.md` Rule 7/11/19, and 5 call-site importers. Cross-Stack Split Check: 5 BE tasks + 0 FE tasks — BE-heavy but rule only triggers when BOTH sides >3; single story OK. Sprint-status.yaml transition: `backlog → ready-for-dev`. |
