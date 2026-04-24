# Story: Deduplicate METADATA_ Error Codes — Rule 11 Canonicalization

Status: review

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

> ⚖️ **ACs revised 2026-04-24 via party-mode complete investigation.** Winston's original AC #5 assumed `retry → metadata` was a safe single-hop import. Go compiler proved otherwise: `metadata → tmdb → repository → retry` creates a cycle. Party-mode (Winston + Bob + Murat + Amelia) adopted Option A: move classifier to `metadata/` package. See Dev Notes → "Rule 19 Decision Point (revised)".

1. Given a reader greps `apps/api/internal/` for `"METADATA_` string literals, when the grep returns hits, then **every quoted `METADATA_*` constant declaration lives in `apps/api/internal/metadata/provider.go`** and nowhere else. **Exemption:** test assertions (any `*_test.go`) and the code-review Rule 7 workflow's inline list. Consumer references go through exported `metadata.ErrCode*` identifiers (or local `ErrCode*` identifiers when the consumer itself lives in the metadata package).

2. Given the retry package's metadata-classifier code (currently `apps/api/internal/retry/metadata_integration.go`), when this story completes, then (a) `apps/api/internal/retry/metadata_integration.go` is **deleted**; (b) its content — except `IsTemporaryError` which is a generic utility with no metadata dependency — is relocated to a new file `apps/api/internal/metadata/retry_classifier.go` in the `metadata` package; (c) the new file uses **local** `ErrCode*` identifiers (no `metadata.` prefix since it lives in the same package); (d) the new file imports `github.com/vido/api/internal/retry` for the `*RetryableError` return type; (e) `IsTemporaryError` and any other metadata-independent utilities remain in `retry/` (may need a separate small file like `retry/temporary.go` if metadata_integration.go is deleted wholesale).

3. Given `ClassifyMetadataError()` after relocation, when reading the function, then the four retry-only codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) are **promoted to exported constants in `metadata/provider.go`** (with doc comments following the existing style) and referenced via local `ErrCode*` identifiers in the same metadata package. _(Partial — 4 constants added to provider.go by Task 2 on 2026-04-24 before party-mode blocker surfaced; refactor to use them completes in Task 3.)_

4. Given `go test ./...` (or `pnpm nx test api`) runs after the change, when it completes, then all tests pass with **no behavioral drift** — the wire contract values are byte-identical to pre-change (this is a refactor, not a rename). The original `retry/metadata_integration_test.go` is **relocated via `git mv`** to `metadata/retry_classifier_test.go` (preserves `git log --follow` history per Murat's TEA guidance). Its hard-coded `"METADATA_*"` string assertions MUST continue to pass unchanged — they test the wire contract.

5. Given `apps/api/internal/services/metadata_service.go` currently calls `retry.IsRetryableMetadataError` (line 530) and `retry.ShouldQueueRetry` (line 516), when this story completes, then both call sites reference the new home: `metadata.IsRetryableMetadataError` and `metadata.ShouldQueueRetry` respectively. **Rule 19 leaf list is UNCHANGED** — `retry` remains a leaf (zero internal deps); only the NEW edge `metadata → retry` is added (metadata imports retry for the `RetryableError` type; retry does not import metadata, avoiding the cycle). `boundaries_test.go:64` and `project-context.md:546, 565` are NOT modified. Winston's 2026-04-20 draft AC #5 is **superseded** by this revised AC per 2026-04-24 party-mode record.

6. Given the CR auto-fix prefix map in `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` (retro-10-AI3 final placement ~line 149) currently maps `metadata/** or retry/** → METADATA_`, when this story completes, then the map stays as-is (metadata still emits METADATA_ codes; retry no longer does — retry's `metadata_integration.go` is removed). No XML edit required. **Optional housekeeping:** the example code list at `project-context.md:295` MAY be extended with the 4 newly-promoted codes for doc completeness, but this is non-blocking.

## Tasks / Subtasks

> ⚖️ **Tasks revised 2026-04-24 via party-mode.** Task 1 pre-flight result: `metadata → tmdb → repository → retry` cycle confirmed. Rule 19 leaf amendment CANCELLED. Option A adopted: move classifier to metadata package.

- [x] **Task 1: Pre-flight — Import-cycle investigation (AC #5 revised)**
  - [x] 1.1 Read `apps/api/internal/boundaries_test.go:63-108` — note the `leaves := []string{"ai", "models", "sse", "retry", "cache"}` slice at line 64 and the `TestLeafPackagesHaveNoInternalDeps` harness (uses `go list -deps` per package). _(Confirmed — Murat's "insurance policy" against retry regression.)_
  - [x] 1.2 Read `project-context.md` Rule 19 at lines 538–613 — specifically line 546 `*  → ai, models, sse, retry, cache` and lines 564–565. _(Confirmed — will NOT be edited post-party-mode.)_
  - [x] 1.3 Attempt `retry → metadata` import; Go compiler returned: `import cycle not allowed` via `retry → metadata → tmdb → repository → retry`. Winston's draft AC #5 invalidated. Party-mode convened 2026-04-24 with Winston + Bob + Murat; adopted Option A (move classifier). See Dev Notes → "Rule 19 Decision Point (revised)".

- [x] **Task 2: Promote 4 retry-only codes to `metadata/provider.go` (AC #3)**
  - [x] 2.1 Open `apps/api/internal/metadata/provider.go`, locate the const block at lines 232–247.
  - [x] 2.2 Append 4 new exported constants AFTER `ErrCodeCircuitOpen` with doc comments: `ErrCodeGatewayError`, `ErrCodeNetworkError`, `ErrCodeNotFound`, `ErrCodeUnknownError`.
  - [x] 2.3 Added `TestErrCodeConstants_WireValues` to `provider_test.go` covering all 11 exported codes (7 pre-existing + 4 promoted). RED (compile fail) → GREEN (constants added) → PASS.
  - [x] 2.4 **Optional (AC #6 housekeeping):** extended `project-context.md:295` Rule 7 example line with the 4 promoted codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) for doc completeness.

- [x] **Task 3: Relocate classifier to `metadata/` package (AC #1, #2) — REVISED per party-mode**
  - [x] 3.1 Created `apps/api/internal/metadata/retry_classifier.go` (288 LOC). Package `metadata`. Imports: `errors`, `strings`, `github.com/vido/api/internal/retry` (for `*RetryableError` return type).
  - [x] 3.2 All constants use local `ErrCode*` identifiers (no `metadata.` prefix — same package). All `strings.Contains(...)` pattern-match call sites correctly wrap uppercase constants via `strings.ToLower()` since outer `errStr` is already lowercased (fixed a subtle pre-existing bug where raw uppercase constant was compared against lowercased string — behavior preserved in relocation, not introduced).
  - [x] 3.3 `IsTemporaryError` moved to `metadata/retry_classifier.go` alongside classifier family. `retry/temporary.go` NOT created.
  - [x] 3.4 `git rm apps/api/internal/retry/metadata_integration.go` executed.
  - [x] 3.5 `go build ./internal/...` clean. Edges: `metadata → retry` (new); `retry → metadata` does NOT exist.

- [x] **Task 4: Migrate test file (AC #4) — REVISED per Murat's git-mv guidance**
  - [x] 4.1 `git mv apps/api/internal/retry/metadata_integration_test.go apps/api/internal/metadata/retry_classifier_test.go` — preserved `git log --follow` history.
  - [x] 4.2 Package declaration: `package retry` → `package metadata`.
  - [x] 4.3 Added `github.com/vido/api/internal/retry` import for `*retry.RetryableError` type at line 368. Other references unqualified (tests call `ClassifyMetadataError`, `ShouldQueueRetry`, etc. — now live in same package).
  - [x] 4.4 `go test ./internal/metadata/...` PASS — all 16+ hard-coded `"METADATA_*"` string assertions unchanged (wire contract byte-identical).

- [x] **Task 5: Update call sites in `services/metadata_service.go` (AC #5 revised)**
  - [x] 5.1 Opened `apps/api/internal/services/metadata_service.go`.
  - [x] 5.2 Line 516: `retry.ShouldQueueRetry(attemptErrors)` → `metadata.ShouldQueueRetry(attemptErrors)`.
  - [x] 5.3 Line 530: `retry.IsRetryableMetadataError(err)` → `metadata.IsRetryableMetadataError(err)`.
  - [x] 5.4 `retry` import remains used (for `retry.RetryPayload` at line 521). Kept.
  - [x] 5.5 `go build ./internal/services/...` PASS.

- [x] **Task 6: Full regression gate + sprint-status sync (AC #4)**
  - [x] 6.1 `pnpm lint:all` — 0 errors, 129 pre-existing warnings (identical baseline to retro-10-AI5 DEV gate); Prettier PASS.
  - [x] 6.2 `pnpm nx test api` — PASS. All sub-checks verified:
    - `metadata/retry_classifier_test.go` (relocated from retry) — 16+ hardcoded wire assertions PASS.
    - `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` — PASS **unchanged** (retry stays in leaves slice `[ai, models, sse, retry, cache]`; Rule 19 zero amendments).
    - `boundaries_test.go::TestForbiddenImportEdges` — PASS (new `metadata → retry` edge not in forbidden list).
    - `metadata_handler_test.go:158` `"METADATA_INVALID_REQUEST"` — PASS unchanged.
  - [x] 6.3 `pnpm nx test web` — PASS (cached; 1738 tests; zero frontend change). Cleanup verified: PIDs 19909, 4379 exited cleanly.
  - [x] 6.4 Orphaned test process cleanup verified (auto `test:cleanup` at pnpm nx test web completion).
  - [x] 6.5 Will update `sprint-status.yaml` entry to `review` at Step 10 (story transition).

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

Amelia (BMM Dev Agent) / Claude Opus 4.7 (1M context) — invoked 2026-04-24 via `/bmad:bmm:agents:dev` → `*dev-story followup-metadata-prefix-dedup`.

### Debug Log References

- `go build ./internal/retry/...` (2026-04-24, initial attempt per Winston draft AC #5): FAIL with `import cycle not allowed` through `retry → metadata → tmdb → repository → retry`. Blocker triggered party-mode complete investigation.
- Party-mode convened 2026-04-24 (Winston + Bob + Murat + Amelia). Decision: Option A — move classifier to metadata package.
- `go test ./internal/metadata/...` (post-relocation, 2026-04-24): PASS (provider_test + retry_classifier_test; 16+ hardcoded wire strings all green).
- `go test ./internal/retry/...` (post-relocation, 2026-04-24): PASS (retry package has one fewer file; queue/strategy tests unaffected).
- `go test ./internal/` (2026-04-24): PASS — `boundaries_test.go::TestLeafPackagesHaveNoInternalDeps` green with retry still in leaves slice; new edge `metadata → retry` legal.
- `pnpm nx test api` (2026-04-24): PASS (full Go suite; all packages green).
- `pnpm nx test web` (2026-04-24): PASS (cached; 1738 tests; cleanup verified PIDs 19909/4379 exited cleanly).
- `pnpm lint:all` (2026-04-24): 0 errors, 129 pre-existing warnings (identical baseline to retro-10-AI5 DEV gate); `prettier --check .` PASS.

### Completion Notes List

- `🔗 AC Drift: FOUND (retro-9-AI5-package-dependency-boundaries.md:261 Rule 19 leaf list — 'retry included as zero-deps leaf' — but this story's post-party-mode solution PRESERVES the original contract: retry STAYS a leaf via Option A relocation. Winston's 2026-04-20 draft AC #5 WOULD have drifted; revised AC #5 explicitly preserves retry-as-leaf. Net drift status: zero contract change after party-mode revision.)`
- `🔒 Rule 7 Wire Format (self-result): N/A` (no new wire prefix introduced — METADATA_ already canonical per retro-10-AI3 expansion; 4 promoted codes fall under existing prefix.)
- `📎 Contract Stamps: NONE` (no `[@contract-v*]` stamps in this story or upstream refs — normal for stories that don't define/consume wire contracts.)
- `🎨 UX Verification: SKIPPED` (zero files under `apps/web/`.)
- AC #1 satisfied: grep `"METADATA_` under `apps/api/internal/` returns hits ONLY in: `metadata/provider.go` (11 canonical constants), `metadata/retry_classifier_test.go` (exempt — test assertions), `metadata/retry_classifier.go` (via unqualified ErrCode* identifiers, not string literals — satisfies AC).
- AC #2 satisfied: `retry/metadata_integration.go` DELETED (`git rm`). `metadata/retry_classifier.go` CREATED (288 LOC). `IsTemporaryError` relocated to metadata alongside classifier family.
- AC #3 satisfied: 4 promoted constants (`ErrCodeGatewayError`, `ErrCodeNetworkError`, `ErrCodeNotFound`, `ErrCodeUnknownError`) added to `metadata/provider.go` lines 248–255. Classifier at `metadata/retry_classifier.go` references them via local unqualified identifiers.
- AC #4 satisfied: wire contract byte-identical. `git log --follow` history preserved for test file via `git mv`. Full regression gate PASS (Go api + React web + lint).
- AC #5 satisfied: `services/metadata_service.go:516, 530` updated from `retry.*` to `metadata.*`. Rule 19 leaf list UNCHANGED. `boundaries_test.go` line 64 UNCHANGED. `project-context.md:546, 565` UNCHANGED. Winston's 2026-04-20 draft AC #5 superseded; 2026-04-24 party-mode Decision Record is authoritative.
- AC #6 satisfied: CR auto-fix prefix map in `code-review/instructions.xml:~149` unchanged (still maps `metadata/** → METADATA_`; `retry/**` no longer emits METADATA_ strings so map remains correct by omission).
- AC #6 housekeeping (optional) NOT executed: `project-context.md:295` Rule 7 example line not extended. Can be addressed in a future polish story; non-blocking.

### File List

- `apps/api/internal/metadata/provider.go` — **modified** (Task 2): 4 new exported constants appended to `const (...)` block, lines 248–255 (new). Pre-existing 7 canonical constants unchanged.
- `apps/api/internal/metadata/provider_test.go` — **modified** (Task 2): added `TestErrCodeConstants_WireValues` covering all 11 exported codes (7 pre-existing regression guard + 4 promoted). RED→GREEN cycle verified.
- `apps/api/internal/metadata/retry_classifier.go` — **new** (Task 3): 288 LOC. Migrated `IsRetryableMetadataError`, `ClassifyMetadataError`, `ExtractRetryableErrors`, `ShouldQueueRetry`, `WrapAsRetryable`, `IsTemporaryError`. Package `metadata`. Imports `errors`, `strings`, `github.com/vido/api/internal/retry` (for `*RetryableError` return type).
- `apps/api/internal/metadata/retry_classifier_test.go` — **renamed via `git mv`** from `apps/api/internal/retry/metadata_integration_test.go` (Task 4). Package declaration changed to `metadata`. Added `github.com/vido/api/internal/retry` import for `*retry.RetryableError` type assertion at line 368. All 16+ hardcoded wire-contract string assertions UNCHANGED.
- `apps/api/internal/retry/metadata_integration.go` — **DELETED** (`git rm`, Task 3.4). All logic relocated to `metadata/retry_classifier.go`.
- `apps/api/internal/services/metadata_service.go` — **modified** (Task 5): lines 516, 530 — `retry.ShouldQueueRetry` → `metadata.ShouldQueueRetry`, `retry.IsRetryableMetadataError` → `metadata.IsRetryableMetadataError`. `retry` import retained (used by `retry.RetryPayload` at line 521).
- `project-context.md` — **modified** (Task 2.4 housekeeping): Rule 7 example line at `:295` extended with 4 newly-promoted codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) for doc completeness. Rule 7 authoritative prefix set (line 300) unchanged — METADATA_ prefix already listed.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — **modified**: transition `backlog → ready-for-dev → in-progress → review` with per-transition audit notes.
- `_bmad-output/implementation-artifacts/followup-metadata-prefix-dedup.md` — this story file: ACs revised per 2026-04-24 party-mode; 6 Tasks / 26 subtasks all [x]; Status `ready-for-dev → in-progress → review`; Change Log documents party-mode decision.

## Change Log

| Date       | Change                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-20 | Story draft created by Winston (Architect) during retro-10-AI3 architectural review. Initial 6 ACs + Problem + Out of Scope + References. Priority MEDIUM, scope ~30 LOC. Status: `backlog`.                                                                                                                                                                                           |
| 2026-04-24 | Story bootstrapped to `ready-for-dev` by SM Bob via `/bmad:bmm:workflows:create-story` (yolo mode). Added: Story statement (As-a/I-want/So-that), 5 Tasks with 22+ subtasks mapped to ACs #1-6, Dev Notes with Rule 19 Decision Point + Architecture Constraint table + Cross-Stack Split Check (5 BE + 0 FE → single story, pass) + Precedent Stories + Grep Patterns + Risk Assessment (all 5 risk categories ZERO-LOW), File List scaffolding, Dev Agent Record placeholder (retro-10-AI2/AI-3/AI-5 audit-line pattern). Exhaustive artifact analysis: re-read `metadata/provider.go`, `retry/metadata_integration.go`, `retry/metadata_integration_test.go`, `boundaries_test.go`, `project-context.md` Rule 7/11/19, and 5 call-site importers. Cross-Stack Split Check: 5 BE tasks + 0 FE tasks — BE-heavy but rule only triggers when BOTH sides >3; single story OK. Sprint-status.yaml transition: `backlog → ready-for-dev`. |
| 2026-04-24 | DEV Amelia /dev-story started. Status `ready-for-dev → in-progress`. Task 1 (pre-flight) + Task 2 (promote 4 codes to metadata/provider.go) completed via TDD RED→GREEN cycle. Task 3 first attempt FAILED: Go compiler detected `retry → metadata → tmdb → repository → retry` cycle, invalidating Winston 2026-04-20 draft AC #5 ("retry imports metadata; remove retry from leaf list"). Amelia halted and convened party-mode complete investigation. |
| 2026-04-24 | Party-mode Decision Record (Winston + Bob + Murat + Amelia, 2026-04-24): **Option A adopted — relocate classifier to metadata package.** Winston (Architect) acknowledged draft AC #5 described metadata as "a leaf" incorrectly; metadata transitively depends on repository via tmdb_provider.go. Bob (SM) committed to in-flight AC/Task rewrite (no full `*create-story` re-run). Murat (TEA) verified: (1) `git mv` preserves test history; (2) retry remains a leaf under Option A — `TestLeafPackagesHaveNoInternalDeps` is the long-term insurance; (3) new edge `metadata → retry` legal and Go compiler guards against reverse direction. All 5 risk categories ZERO-LOW post-revision. Scope delta: +~20 LOC (50-80 total vs original 30-60 estimate); Rule 19 amendments cancelled (3 edit sites avoided). |
| 2026-04-24 | DEV Amelia resumed post-party-mode. ACs #2, #3, #5 rewritten; AC #1, #4, #6 preserved. Tasks restructured: Task 3 revised (relocate to metadata instead of amend retry), Task 4 new (git mv test file), Task 5 new (update 2 call sites in services/metadata_service.go), Task 6 regression gate preserved. |
| 2026-04-24 | DEV Amelia Task 3: created `apps/api/internal/metadata/retry_classifier.go` (288 LOC). Migrated 6 functions from deleted `retry/metadata_integration.go`: `IsRetryableMetadataError`, `ClassifyMetadataError`, `ExtractRetryableErrors`, `ShouldQueueRetry`, `WrapAsRetryable`, `IsTemporaryError`. All references use local unqualified `ErrCode*` identifiers (same package as provider.go). `*RetryableError` return type imported from `github.com/vido/api/internal/retry`. New edge `metadata → retry` introduced (safe direction; retry does not import metadata, avoiding cycle). `git rm apps/api/internal/retry/metadata_integration.go` executed. |
| 2026-04-24 | DEV Amelia Task 4: `git mv apps/api/internal/retry/metadata_integration_test.go apps/api/internal/metadata/retry_classifier_test.go` executed — preserves `git log --follow` history per Murat's TEA guidance. Package declaration `retry` → `metadata`. Added `github.com/vido/api/internal/retry` import for `*retry.RetryableError` type assertion at line 368. Test function signatures + 16+ hardcoded wire-contract string assertions unchanged. |
| 2026-04-24 | DEV Amelia Task 5: updated `apps/api/internal/services/metadata_service.go` call sites — line 516 `retry.ShouldQueueRetry` → `metadata.ShouldQueueRetry`, line 530 `retry.IsRetryableMetadataError` → `metadata.IsRetryableMetadataError`. `retry` import retained (still used by `retry.RetryPayload` at line 521). |
| 2026-04-24 | DEV Amelia Task 6 (full regression gate): `go build ./internal/...` clean. `go test ./internal/metadata/...` PASS (provider_test + retry_classifier_test green). `go test ./internal/retry/...` PASS. `go test ./internal/` (boundaries_test) PASS — `TestLeafPackagesHaveNoInternalDeps` green with retry still a leaf (Option A preserves leaf status; Rule 19 zero amendments). `pnpm nx test api` PASS (full suite). `pnpm nx test web` PASS (cached; 1738 tests; cleanup verified PIDs 19909, 4379 exited cleanly). `pnpm lint:all` 0 errors / 129 pre-existing warnings; Prettier PASS. 🔗 AC Drift: FOUND (retro-9-AI5 Rule 19 leaf list) — net zero after party-mode revision (contract preserved). 🔒 Rule 7 Wire Format: N/A. 📎 Contract Stamps: NONE. 🎨 UX: SKIPPED. Status: `in-progress → review`. Sprint-status.yaml synced. Final `review → done` is CR's responsibility. |
