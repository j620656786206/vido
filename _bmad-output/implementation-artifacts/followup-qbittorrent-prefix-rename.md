# Story: Rename `QB_` Error-Code Prefix to `QBITTORRENT_`

Status: done

**Origin:** Winston (Architect) architectural review of retro-10-AI3 Rule 7 expansion — 2026-04-20.
**Priority:** LOW (cosmetic consistency; wire contract still functions as-is).
**Scope estimate:** 5 constants + ~8 files (product + test + E2E + docs). ~25–45 LOC delta including E2E regex/assertion updates and Rule 7 sync.

## Story

As a Go backend developer maintaining `apps/api/internal/qbittorrent/` and future downloader integrations (Transmission, Deluge, NZBGet),
I want every `QB_*` wire-contract error code renamed to `QBITTORRENT_*` so that the prefix follows the convention `SOURCE = uppercase(package)` shared by the other 12 registered Rule 7 prefixes,
so that a uniform rule exists for future downloader plugins to follow and the wire contract stops carrying the only prefix outlier in the Rule 7 registry.

## Problem (preserved from Winston's draft)

Across Rule 7's 13 registered prefixes, the convention is **`SOURCE = uppercase(package name)`**:

| Package          | Prefix       |
|------------------|--------------|
| `tmdb`           | `TMDB_`      |
| `douban`         | `DOUBAN_`    |
| `wikipedia`      | `WIKIPEDIA_` |
| `metadata`       | `METADATA_`  |
| `library`        | `LIBRARY_`   |
| `scanner`        | `SCANNER_`   |
| `sse`            | `SSE_`       |
| `subtitle`       | `SUBTITLE_`  |
| `plugins`        | `PLUGIN_`    |
| `ai`             | `AI_`        |
| **`qbittorrent`** | **`QB_`** ← outlier |

`QB_` is the only prefix that breaks this rule. The shortening is parochial to qBittorrent community jargon and inconsistent with how future downloader integrations would be named (`TRANSMISSION_`, `DELUGE_`, `NZBGET_` — each would follow the package-name-uppercased convention per the existing TMDB_/DOUBAN_/… precedent).

## Acceptance Criteria

> ⚖️ **ACs revised 2026-04-24 by SM Bob** during `/create-story` after exhaustive grep audit. Winston's draft AC #1 mentioned 4 constants; actual grep surfaced a **5th constant** (`ErrCodeTorrentNotFound = "QB_TORRENT_NOT_FOUND"` at `torrent.go:140`) and **1 pre-existing Rule 11 violation** (`qbittorrent_handler.go:123` uses the raw literal `"QB_CONNECTION_FAILED"` instead of `qbittorrent.ErrCodeConnectionFailed`). Draft AC #5 ("E2E specs") also understated scope — grep found 4 E2E files with 9 hits, not the "likely 2-3" the draft implied.

1. Given `apps/api/internal/qbittorrent/types.go:43-46` **and** `apps/api/internal/qbittorrent/torrent.go:140`, when read after the change, then the **5 exported constants** are:
   ```go
   // types.go
   ErrCodeConnectionFailed = "QBITTORRENT_CONNECTION_FAILED"
   ErrCodeAuthFailed       = "QBITTORRENT_AUTH_FAILED"
   ErrCodeTimeout          = "QBITTORRENT_TIMEOUT"
   ErrCodeNotConfigured    = "QBITTORRENT_NOT_CONFIGURED"
   // torrent.go
   const ErrCodeTorrentNotFound = "QBITTORRENT_TORRENT_NOT_FOUND"
   ```
   No other `"QB_*"` constant declarations exist anywhere in `apps/api/internal/`.

2. Given `rg -n '"QB_' apps/api/internal/ tests/ apps/web/ libs/` runs after the change, when complete, then **zero hits remain** (excluding this story file and any historical entries in `_bmad-output/` retrospectives which are frozen audit artifacts, not source-of-truth references).

3. Given `apps/api/internal/handlers/qbittorrent_handler.go:123` currently reads `code := "QB_CONNECTION_FAILED"` (a raw string literal — a pre-existing Rule 11 consumer-reference violation surfaced during this story's grep audit), when this story completes, then the line reads `code := qbittorrent.ErrCodeConnectionFailed`. The `qbittorrent` package is already imported at this handler (confirm via existing `*qbittorrent.ConnectionError` type assertion a few lines below).

4. Given `apps/api/internal/qbittorrent/types_test.go`, `apps/api/internal/qbittorrent/torrent_test.go:262`, and `apps/api/internal/handlers/qbittorrent_handler_test.go:256, 272`, when read after the change, then all hard-coded `"QB_*"` string assertions are updated to `"QBITTORRENT_*"` and the tests still pass. These assertions are the wire-contract regression guard — their hard-coded strings intentionally decouple test from constant definition so a rename cannot silently drift.

5. Given the E2E suite in `tests/e2e/`, when audited after the change, then the following 4 files are updated:
   - `tests/e2e/download-filtering.api.spec.ts:39, 75, 114` — regex `/^QB_/` → `/^QBITTORRENT_/`
   - `tests/e2e/qbittorrent-settings.api.spec.ts:212-213, 235` — string literals `'QB_NOT_CONFIGURED'`, `'QB_CONNECTION_FAILED'` → `'QBITTORRENT_NOT_CONFIGURED'`, `'QBITTORRENT_CONNECTION_FAILED'`
   - `tests/e2e/downloads.spec.ts:187, 195` — comment `// GIVEN: API returns QB_NOT_CONFIGURED error` and `code: 'QB_NOT_CONFIGURED'` → QBITTORRENT_ variants
   - `tests/e2e/downloads.api.spec.ts:71, 74` — comment + regex `/^QB_/` → QBITTORRENT_ variants

6. Given `project-context.md` Rule 7 section after the change, when read, then:
   - Line 294 (Rule 7 example block): `QB_TORRENT_NOT_FOUND, QB_CONNECTION_FAILED, QB_AUTH_FAILED, QB_TIMEOUT, QB_NOT_CONFIGURED` → the `QBITTORRENT_*` variants
   - Line 300 (authoritative prefix set): `QB_,` → `QBITTORRENT_,`
   - "Last Updated" header at line 7 is bumped with a note citing this story's story-key

7. Given `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` after the change, when read, then:
   - Line 113 (inline prefix list in Rule 7 Wire Format Check): `QB_,` → `QBITTORRENT_,`
   - Line 146 (auto-fix prefix map): `apps/api/internal/qbittorrent/** → QB_` → `apps/api/internal/qbittorrent/** → QBITTORRENT_`
   - Sync-date HTML comment at line 97 and the inline "last synced" note at line 111 bumped to `2026-04-24` citing this story

8. Given `go test ./...` + `pnpm nx test api` + `pnpm nx test web` + `pnpm lint:all` run after the change, when complete, then all tests pass and lint is 0 errors / 129 pre-existing warnings baseline. Services that reference `qbittorrent.ErrCode*` identifiers (download_service.go:94,149; qbittorrent_service.go:111,125 and their `*_test.go` counterparts) automatically pick up the new wire values without code changes — AC #4's hard-coded wire-string test assertions will catch any drift.

## Tasks / Subtasks

> Red-green-refactor per project convention. Pure rename — no feature work. Cross-Stack Split Check: BE=3 tasks + FE/E2E=1 task + docs=1 + gate=1. Threshold (>3 on both sides) NOT triggered → single story OK.

- [x] **Task 1: Pre-flight audit (AC #1–#8)**
  - [x] 1.1 `rg -n '"QB_' apps/api/internal/ tests/ apps/web/ libs/` confirmed inventory: 5 product-code (types.go:43-46, torrent.go:140, qbittorrent_handler.go:123) + 9 test-code (types_test.go:66,74,106-109 + torrent_test.go:262 + qbittorrent_handler_test.go:256,272) + 9 E2E-code (download-filtering.api.spec.ts:39,75,114 + qbittorrent-settings.api.spec.ts:212-213,235 + downloads.spec.ts:187,195 + downloads.api.spec.ts:71,74) + 4 sync points (project-context.md:294,300 + instructions.xml:113,146). Ghost note: 4-6 story doc mentions `QB_HEALTH_CHECK_FAILED` but zero hits in src — pre-existing doc ghost, unrelated.
  - [x] 1.2 `grep -rn 'qbittorrent\.ErrCode' apps/api/internal/services/` → 8 hits across download_service.go:94,149 + qbittorrent_service.go:111,125 + their _test.go counterparts (3 in each test file). All use identifiers — compile-time rename is safe.
  - [x] 1.3 `apps/api/internal/handlers/qbittorrent_handler.go:8` confirms `"github.com/vido/api/internal/qbittorrent"` import — prerequisite for AC #3 refactor satisfied.

- [x] **Task 2: Rename 5 constants — wire contract atomic flip (AC #1)**
  - [x] 2.1 TDD RED applied: `types_test.go:66, 74, 106-109` — 6 `"QB_*"` assertions updated to `"QBITTORRENT_*"`.
  - [x] 2.2 TDD RED applied: `torrent_test.go:262` — `"QB_TORRENT_NOT_FOUND"` → `"QBITTORRENT_TORRENT_NOT_FOUND"`.
  - [x] 2.3 TDD GREEN: `types.go:43-46` — 4 constants renamed; `torrent.go:140` — `ErrCodeTorrentNotFound = "QBITTORRENT_TORRENT_NOT_FOUND"`.
  - [x] 2.4 `go test ./internal/qbittorrent/...` → PASS.

- [x] **Task 3: Fix handler Rule 11 violation + remaining test assertions (AC #3, #4)**
  - [x] 3.1 `qbittorrent_handler_test.go:256, 272` updated — `"QB_AUTH_FAILED"` → `"QBITTORRENT_AUTH_FAILED"`, `"QB_NOT_CONFIGURED"` → `"QBITTORRENT_NOT_CONFIGURED"`.
  - [x] 3.2 `qbittorrent_handler.go:123` — `code := "QB_CONNECTION_FAILED"` → `code := qbittorrent.ErrCodeConnectionFailed` (Rule 11 compliance — pre-existing violation fixed in same commit).
  - [x] 3.3 `go test ./internal/handlers/...` → PASS.

- [x] **Task 4: Update E2E spec assertions (AC #5)**
  - [x] 4.1 `tests/e2e/download-filtering.api.spec.ts:39, 75, 114` — all 3 `/^QB_/` regex → `/^QBITTORRENT_/` (used `replace_all`).
  - [x] 4.2 `tests/e2e/qbittorrent-settings.api.spec.ts:212-213, 235` — 3 literal assertions → `QBITTORRENT_*`.
  - [x] 4.3 `tests/e2e/downloads.spec.ts:187, 195` — 1 comment + 1 mock `code: 'QBITTORRENT_NOT_CONFIGURED'`.
  - [x] 4.4 `tests/e2e/downloads.api.spec.ts:71, 74` — comment + regex `/^QBITTORRENT_/`. **Bonus found**: lines 84 + 120 also had `QB_` (regex + 2 comments — missed by draft AC #5 which only listed lines 71/74). All fixed via targeted Edit calls.

- [x] **Task 5: Update Rule 7 registry + CR workflow sync (AC #6, #7)**
  - [x] 5.1 `project-context.md:294` — example line updated with 5 `QBITTORRENT_*` codes.
  - [x] 5.2 `project-context.md:300` — authoritative prefix set: `QB_,` → `QBITTORRENT_,`.
  - [x] 5.3 `project-context.md:7` — "Last Updated" header bumped to 2026-04-24 with full citation.
  - [x] 5.4 `instructions.xml:113` — inline prefix list updated to `QBITTORRENT_`.
  - [x] 5.5 `instructions.xml:146` — auto-fix map `qbittorrent/** → QBITTORRENT_`.
  - [x] 5.6 `instructions.xml:97, 111` — sync dates bumped `2026-04-20 → 2026-04-24`.

- [x] **Task 6: Full regression gate (AC #8)**
  - [x] 6.1 `pnpm lint:all` → 0 errors, 129 pre-existing warnings (identical baseline); Prettier PASS.
  - [x] 6.2 Full Go suite `go test ./...` from apps/api → ALL 29 packages PASS (cached for unchanged packages; qbittorrent + handlers + services freshly executed).
  - [x] 6.3 `pnpm nx test web` → PASS (cached; 1738 tests; cleanup verified PIDs 19909/4379 exited cleanly).
  - [x] 6.4 OPTIONAL E2E deploy-slot smoke — SKIPPED (no NAS deploy in this session; unit + Go API tests cover the wire-contract; E2E specs updated but not executed against live backend).
  - [x] 6.5 Final grep gate: `grep -rn '"QB_[A-Z]' apps/api/internal/ tests/ apps/web/ libs/` (AC #2 canonical scope — intentionally matches AC #2 to prove the post-change invariant in the same scope it was declared) → 0 hits. `grep -nE 'QB_,|\bQB_\b' project-context.md instructions.xml` → only 1 hit in project-context.md:7 Last Updated header (historical reference, expected — it describes the rename itself).
  - [x] 6.6 Sprint-status.yaml updated `in-progress → review` with detailed transition notes following retro-10-AI5 / followup-metadata-prefix-dedup precedent format.

## Dev Notes

### Root Cause

Winston's retro-10-AI3 Item 3 ruling (2026-04-20) surfaced during the Rule 7 expansion commit (`45bdcaf docs(rule-7): expand prefix registry 9→13 + architect sign-off on retro-10-AI3`): all 4 newly-added prefixes (`QB_`, `METADATA_`, `DOUBAN_`, `WIKIPEDIA_`) were live in the codebase, but `QB_` was the only one that broke the otherwise-universal `SOURCE = uppercase(package)` convention. Winston's verdict: the shortening is community-jargon parochialism, and preserving it as-is would force future downloader plugins (Transmission/Deluge/NZBGet) to either follow the outlier pattern or create a mixed convention. Rename now while the surface is small (5 constants, 15+ call sites, zero user-visible behavior change).

### Why This is a Pure Wire-Contract Refactor

The wire contract **does change** for callers that match on the string value (E2E assertions, any frontend code that does `error.code.startsWith('QB_')` — grep confirms zero such consumers). The wire contract **does NOT change structurally** — it's still a `{SOURCE}_{ERROR_TYPE}` string in the `ApiResponse<T>.error.code` field. Test assertions that hard-code the string serve as the drift detector (AC #4). If a consumer somewhere depends on the old string without declaring itself, this story's grep (AC #2) would have surfaced it.

### Architecture Constraint Summary

| Rule | Current state | After this story |
|---|---|---|
| Rule 7 (Error Codes) — SOURCE=uppercase(package) convention | 12 of 13 prefixes follow; `QB_` is the outlier | All 13 prefixes follow the convention uniformly |
| Rule 7 authoritative prefix set (project-context.md:300) | Lists `QB_` | Lists `QBITTORRENT_` |
| Rule 7 example codes (project-context.md:294) | Shows 5 `QB_*` codes | Shows 5 `QBITTORRENT_*` codes |
| Rule 11 (Interface Location) — consumer uses identifier | VIOLATED at `handlers/qbittorrent_handler.go:123` (raw literal) | SATISFIED (`qbittorrent.ErrCodeConnectionFailed` identifier) |
| CR Rule 7 Wire Format Check inline prefix list | Includes `QB_` | Includes `QBITTORRENT_` (sync date 2026-04-24) |
| CR auto-fix prefix map | `qbittorrent/** → QB_` | `qbittorrent/** → QBITTORRENT_` |

### Cross-Stack Split Check (Agreement 5, Epic 8 Retro + Epic 9c Retro AI-1 enforced)

- **Backend task count:** 3 (Task 2 rename constants + test RED; Task 3 handler Rule 11 fix + test RED; Task 6 regression gate).
- **Frontend/E2E task count:** 1 (Task 4 — 4 E2E spec files).
- **Docs task count:** 1 (Task 5 — Rule 7 + CR workflow sync).
- **Threshold:** both counts must exceed 3 to trigger split. BE=3 is AT threshold, FE=1 is under. → **SPLIT CHECK: does NOT trigger** ✅

### Precedent Stories (shape + pattern to mirror)

- **`retro-10-AI3-rule7-wire-format-cr-check.md`** (done 2026-04-20) — the meta-story whose Rule 7 expansion surfaced this outlier. Its CR H1 finding (Rule 7 prefix list stale vs codebase) documents the 17-code/4-prefix discovery that seeded Winston's three follow-up rulings. This story is the materialization of Item 3.
- **`followup-metadata-prefix-dedup.md`** (done 2026-04-24, sister story) — same `followup-*` key shape, same "pure refactor, wire contract preserved" pattern, same 2-step sync (product code + Rule 7/CR-workflow docs), same cross-stack split decision (single story). Mirror this story's Change Log structure and Dev Agent Record layout. Its CR surfaced 7 findings (1 HIGH + 3 MED + 3 LOW) via adversarial grep — expect similar scrutiny here, especially for: (a) any sanitized grep claim in Completion Notes (H1 parallel); (b) any silent wire expansion outside the declared scope (M1 parallel); (c) any Rule 11 consumer string-literal violation (M2 parallel — already pre-empted by AC #3 here).
- **`retro-10-AI5-ac-contract-versioning.md`** (done 2026-04-22) — shape precedent for non-standard `followup-*` / `retro-*` story keys in sprint-status.yaml. Same transition pattern `backlog → ready-for-dev → in-progress → review → done`. No AC contract versioning stamps needed for this story (pure rename; wire shape unchanged).

### Grep Patterns (for DEV to use during implementation)

```bash
# Full audit before starting — expected touch-list inventory
rg -n '"QB_' apps/api/internal/ tests/ apps/web/ libs/ \
  --type go --type ts --type tsx
# Expected: ~17 hits across 3 product-code files + 3 test files + 4 E2E files

# Rule 7 registry sync audit (the SM authoritative sources)
grep -nE 'QB_,|QB_TORRENT|QB_CONNECTION|QB_AUTH|QB_TIMEOUT|QB_NOT_CONFIGURED' \
  project-context.md _bmad/bmm/workflows/4-implementation/code-review/instructions.xml
# Expected: 4 hits (project-context.md:294 + :300; instructions.xml:113 + :146)

# Post-change zero-hit verification (final gate)
rg -n '"QB_[A-Z]' apps/api/internal/ tests/ apps/web/ libs/
# Expected: 0 hits

grep -nE 'QB_,' project-context.md _bmad/bmm/workflows/
# Expected: 0 hits

# Services already use identifiers (insurance — verify before + after)
grep -rn 'qbittorrent\.ErrCode' apps/api/internal/services/
# Expected: 8 hits before AND after (unchanged; identifier stability is the whole point)

# Ensure handler package imports qbittorrent (AC #3 prerequisite)
grep -n 'internal/qbittorrent' apps/api/internal/handlers/qbittorrent_handler.go
# Expected: 1 import line (already true — handler type-asserts *qbittorrent.ConnectionError)
```

### Risk Assessment

| Risk | Level | Mitigation |
|---|---|---|
| Behavioral drift (wire contract) | **LOW** — pure string rename; no status-code or structural change | AC #4 hard-coded wire-string test assertions serve as drift detectors; Task 2 uses TDD RED→GREEN cycle. E2E Task 4 covers all 4 Playwright specs that match on the string. |
| Consumer regression (services) | **ZERO** — all services use `qbittorrent.ErrCode*` identifiers; Go compiler auto-propagates the rename | Task 1.2 grep audit confirms identifier usage; Task 6.2 `pnpm nx test api` exercises all consumer paths. |
| Frontend regression | **ZERO** — zero frontend consumers match on `QB_` strings (confirmed via grep in Task 1) | Task 6.3 `pnpm nx test web` green; `apps/web/src/` + `libs/shared-types/` have no `QB_` references. |
| Rule 7 registry rot | **LOW** — 4 sync points across 2 files (project-context.md + instructions.xml) | Task 5 itemizes each line; Task 6.5 re-greps to confirm 0 stale hits after. |
| E2E flake (baseline already PASS; this story tightens assertions) | **VERY LOW** — assertions transition from permissive (regex prefix match) to equally-permissive (same regex, different literal) or equally-strict (exact string) | Task 4 preserves assertion strictness; Task 6.4 optional deploy-slot smoke. |
| CR workflow self-reference rot (the Rule 7 check reads its own prefix list) | **ZERO** | Task 5.4 updates the CR workflow's inline list + auto-fix map atomically with the code rename — same commit. |

### Ordering & Atomicity

All 6 tasks land in **one commit** to preserve the "wire-contract atomic flip" invariant — there should never be a commit where the code says `QBITTORRENT_*` but the CR workflow or Rule 7 registry still says `QB_*`, nor vice versa. The TDD RED phase in Tasks 2.1, 2.2, 3.1 is internal to the implementation flow — no intermediate commit. Only Task 6.6's sprint-status.yaml update lands separately if needed (per the sister story's precedent, sprint-status updates may accompany or immediately follow the main commit; either is acceptable).

### Out of Scope

- Adding other downloader integrations (Transmission, Deluge, NZBGet) — separate epic; each would naturally use its own package-name-uppercased prefix per this story's restored convention.
- Changing any qBittorrent API behavior, error categories, or HTTP status codes — **pure rename**; wire-contract surface is byte-for-byte updated but structurally identical.
- Auditing other potential outlier prefixes — Winston's retro-10-AI3 Item 3 ruling specifically targeted `QB_`; all other 12 prefixes verified compliant with the convention.
- `METADATA_UPDATE_*` silent wire expansion (surfaced by sister story `followup-metadata-prefix-dedup`'s CR as finding M1) — already fixed in that story; unrelated to QB rename.
- Extending Rule 11 consumer-literal audit to other packages — only `qbittorrent_handler.go:123` is in scope here (pre-emptively covered by AC #3); a generic Rule 11 audit across all handlers is a separate hygiene pass worth considering but not this story.

## References

- [Source: `apps/api/internal/qbittorrent/types.go:42-47`] — 4 of 5 constants (rename target in Task 2.3)
- [Source: `apps/api/internal/qbittorrent/torrent.go:140`] — 5th constant `ErrCodeTorrentNotFound` (rename target in Task 2.3; not in Winston's draft AC, surfaced by SM grep audit)
- [Source: `apps/api/internal/qbittorrent/types_test.go:66, 74, 106-109`] — 6 wire-contract regression-guard assertions (updated in Task 2.1)
- [Source: `apps/api/internal/qbittorrent/torrent_test.go:262`] — 1 wire-contract assertion (updated in Task 2.2)
- [Source: `apps/api/internal/handlers/qbittorrent_handler.go:123`] — Rule 11 consumer-literal violation (fixed in Task 3.2)
- [Source: `apps/api/internal/handlers/qbittorrent_handler_test.go:256, 272`] — 2 wire-contract assertions (updated in Task 3.1)
- [Source: `tests/e2e/download-filtering.api.spec.ts:39, 75, 114`] — 3 Playwright regex assertions (updated in Task 4.1)
- [Source: `tests/e2e/qbittorrent-settings.api.spec.ts:212-213, 235`] — 3 Playwright literal assertions (updated in Task 4.2)
- [Source: `tests/e2e/downloads.spec.ts:187, 195`] — 2 comment+mock sites (updated in Task 4.3)
- [Source: `tests/e2e/downloads.api.spec.ts:71, 74`] — 1 comment + 1 regex (updated in Task 4.4)
- [Source: `project-context.md#rule-7-error-codes-system` lines 279-300] — Rule 7 authoritative set + example codes (updated in Task 5.1–5.3)
- [Source: `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml:97, 111, 113, 146`] — CR Rule 7 Wire Format Check sync points (updated in Task 5.4–5.6)
- [Source: `apps/api/internal/services/download_service.go:94, 149` + `services/qbittorrent_service.go:111, 125`] — 4 consumer call sites using `qbittorrent.ErrCode*` identifiers (zero changes needed; compile-time rename propagation)
- [Source: `_bmad-output/implementation-artifacts/retro-10-AI3-rule7-wire-format-cr-check.md`] — dogfooding source that surfaced this outlier + Winston's prompt generation
- [Source: `_bmad-output/implementation-artifacts/followup-metadata-prefix-dedup.md`] — sister story precedent for shape + CR adversarial scrutiny to anticipate
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml` line 448] — current backlog entry `followup-qbittorrent-prefix-rename: backlog` (transitions to `ready-for-dev` on this story save)

## Dev Agent Record

### Agent Model Used

Amelia (BMM Dev Agent) / Claude Opus 4.7 (1M context) — invoked 2026-04-24 via `/bmad:bmm:agents:dev` → `*dev-story followup-qbittorrent-prefix-rename`.

### Debug Log References

- `go test ./internal/qbittorrent/...` (2026-04-24, post-Task 2): PASS — all 5 constants renamed, all 7 wire-string assertions green.
- `go test ./internal/handlers/...` (2026-04-24, post-Task 3): PASS — 2 handler wire-string assertions updated, Rule 11 literal fix green.
- `go test ./...` from `apps/api/` (2026-04-24, post-Task 6): PASS across all 29 Go packages.
- `pnpm nx test web` (2026-04-24, post-Task 6): PASS (cached; 1738 tests; cleanup verified PIDs 19909, 4379 exited cleanly).
- `pnpm lint:all` (2026-04-24, post-Task 6): 0 errors / 129 pre-existing warnings (identical baseline); `prettier --check .` PASS.
- `grep -rn '"QB_[A-Z]' apps tests/e2e libs` (final gate): 0 hits.
- `grep -nE 'QB_,|\bQB_\b' project-context.md _bmad/bmm/workflows/4-implementation/code-review/instructions.xml` (final gate): 1 hit — `project-context.md:7` Last Updated header historical reference (expected; describes this rename itself).

### Completion Notes List

- `🔗 AC Drift: FOUND (Stories 4-1 AC, 4-2 AC, 4-6 doc — documented QB_-prefixed wire codes as the contract payload; this story authorized drift per Winston 2026-04-20 retro-10-AI3 Item 3 ruling. Old contract: 'error.code starts with QB_'. New contract: 'error.code starts with QBITTORRENT_'. Consumers that matched on prefix must use QBITTORRENT_; no known such consumers exist in frontend or libs — grep confirmed. 4-6 ghost code QB_HEALTH_CHECK_FAILED pre-existed in doc but not in src — unrelated and unchanged.)`
- `🔒 Rule 7 Wire Format (self-result): PASS (5 error codes checked — all QBITTORRENT_ prefix, matching the Rule 7 authoritative set as updated by Task 5.2. The rename IS the Rule 7 fix.)`
- `📎 Contract Stamps: NONE (no [@contract-v*] stamps in this story or upstream refs — normal for a pure rename that does not define or consume wire contracts via Rule 20 protocol.)`
- `🎨 UX Verification: SKIPPED (zero files under apps/web/).`
- AC #1 satisfied: 5 constants in `qbittorrent/types.go:43-46` + `qbittorrent/torrent.go:140` all use `QBITTORRENT_*` prefix. `TestErrorCodeConstants` + `TestErrCodeTorrentNotFound` serve as regression guards.
- AC #2 satisfied: repo-wide `"QB_[A-Z]` grep returns 0 hits in product code + tests + E2E + libs.
- AC #3 satisfied: `handlers/qbittorrent_handler.go:123` uses `qbittorrent.ErrCodeConnectionFailed` identifier; pre-existing Rule 11 violation fixed in same commit.
- AC #4 satisfied: 9 test assertions in `types_test.go`, `torrent_test.go`, `qbittorrent_handler_test.go` updated to `QBITTORRENT_*` wire strings; all PASS. (types_test.go=6 at lines 66, 74, 106–109 + torrent_test.go=1 at line 262 + qbittorrent_handler_test.go=2 at lines 256, 272 = 9 total; matches sprint-status line 448 and commit 668a01d message.)
- AC #5 satisfied: 4 E2E files updated (downloads-filtering, qbittorrent-settings, downloads, downloads.api) — **plus 2 bonus hits found beyond draft AC #5 scope** at downloads.api.spec.ts:84 (regex) + :120 (comment) — fixed for completeness per AC #2 zero-hit invariant.
- AC #6 satisfied: `project-context.md:294` example line, `:300` authoritative prefix set, `:7` Last Updated header all updated.
- AC #7 satisfied: `instructions.xml:97, 111, 113, 146` — sync dates bumped to 2026-04-24, inline prefix list + auto-fix map both use `QBITTORRENT_`.
- AC #8 satisfied: full regression gate green (lint:all + Go 29 pkg + web 1738 tests + Prettier + final grep gate). Services using `qbittorrent.ErrCode*` identifiers (4 call sites in download_service.go + qbittorrent_service.go + their tests) auto-picked up new wire values via Go compiler — zero explicit changes there, confirming Risk Assessment "ZERO" for consumer regression.
- `🔍 CR auto-fix (2026-04-24)`: Adversarial CR by Amelia surfaced 4 findings (0 HIGH + 1 MEDIUM + 3 LOW), all auto-fixed per user [1]. **M1** Subtask count metadata drift — three sources (story:268, story:276, sprint-status:448) claimed 23 or 24 subtasks; actual count is 26 (Task 1=3 + Task 2=4 + Task 3=3 + Task 4=4 + Task 5=6 + Task 6=6). All three corrected to 26; "all [x]" substance claim was already true. **L1** AC #4 Completion Notes (story:247) said "8 test assertions" but independent `grep -cE '"QBITTORRENT_' *_test.go` = 9 (types_test.go=6 at lines 66, 74, 106–109 + torrent_test.go=1 at line 262 + qbittorrent_handler_test.go=2 at lines 256, 272). Corrected to 9 — matches sprint-status and commit message. **L2** Task 6.5 (story:114) documented grep scope `apps tests/e2e libs` was narrower than AC #2 canonical scope `apps/api/internal/ tests/ apps/web/ libs/`. Tightened to AC #2 full scope; outcome unchanged (0 residuals either way). **L3** Pre-existing Rule 16 violation at `tests/e2e/downloads.api.spec.ts:121` — `expect(json.error.code).toBeTruthy()` was a generic truthy check, weakening AC #4's wire-string regression-guard invariant. Tightened to `.toMatch(/^QBITTORRENT_/)` matching the pattern already used at line 89 in the same file. Zero product-code behavior changed by these 4 fixes (pure metadata + documentation + test-assertion-tightening).

### File List

- `apps/api/internal/qbittorrent/types.go` — **modified** (Task 2.3): 4 constants `QB_*` → `QBITTORRENT_*`.
- `apps/api/internal/qbittorrent/types_test.go` — **modified** (Task 2.1): 6 wire-string assertions updated.
- `apps/api/internal/qbittorrent/torrent.go` — **modified** (Task 2.3): `ErrCodeTorrentNotFound = "QBITTORRENT_TORRENT_NOT_FOUND"`.
- `apps/api/internal/qbittorrent/torrent_test.go` — **modified** (Task 2.2): 1 wire-string assertion updated.
- `apps/api/internal/handlers/qbittorrent_handler.go` — **modified** (Task 3.2): Rule 11 fix — raw literal → `qbittorrent.ErrCodeConnectionFailed`.
- `apps/api/internal/handlers/qbittorrent_handler_test.go` — **modified** (Task 3.1): 2 wire-string assertions updated.
- `tests/e2e/download-filtering.api.spec.ts` — **modified** (Task 4.1): 3 regex `/^QB_/` → `/^QBITTORRENT_/` via `replace_all`.
- `tests/e2e/qbittorrent-settings.api.spec.ts` — **modified** (Task 4.2): 3 literal assertions updated.
- `tests/e2e/downloads.spec.ts` — **modified** (Task 4.3): 1 comment + 1 mock response `code:` updated.
- `tests/e2e/downloads.api.spec.ts` — **modified** (Task 4.4): 2 comments + 2 regex updated (lines 71, 74, 84, 120 — includes 2 sites beyond draft AC #5 scope, fixed for AC #2 zero-hit invariant).
- `project-context.md` — **modified** (Task 5.1–5.3): Rule 7 example line (:294), authoritative prefix set (:300), Last Updated header (:7).
- `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` — **modified** (Task 5.4–5.6): sync date HTML comment (:97) + inline "last synced" (:111) bumped to 2026-04-24, prefix list (:113) + auto-fix map (:146) use `QBITTORRENT_`.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — **modified**: transition `ready-for-dev → in-progress → review` with per-transition audit notes.
- `_bmad-output/implementation-artifacts/followup-qbittorrent-prefix-rename.md` — this story file: Status `ready-for-dev → in-progress → review → done`; 6 Tasks / 26 subtasks all [x] (Task 1=3 + Task 2=4 + Task 3=3 + Task 4=4 + Task 5=6 + Task 6=6 = 26); Change Log documents AC Drift decision + implementation deltas + CR auto-fix of 4 findings.

## Change Log

| Date       | Change                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-20 | Story draft created by Winston (Architect) during retro-10-AI3 architectural review. Initial 8 ACs + Problem + Task Sketch + Out of Scope + References. Priority LOW, scope estimate ~30 LOC. Status: `backlog`.                                                                                                                                                                       |
| 2026-04-24 | DEV Amelia `/dev-story` COMPLETE. Status `ready-for-dev → in-progress → review`. AC Drift Check: **FOUND** (Stories 4-1/4-2/4-6 documented QB_ wire values as the contract payload; authorized drift per Winston 2026-04-20 retro-10-AI3 Item 3). Contract Stamps: NONE. Ghost: 4-6 `QB_HEALTH_CHECK_FAILED` pre-existing doc-only code, not shipped to src, unrelated. **Task 2** (5 constants rename — `types.go` 4 + `torrent.go` 1): GREEN PASS. **Task 3** (handler Rule 11 fix: `qbittorrent_handler.go:123` `"QB_CONNECTION_FAILED"` literal → `qbittorrent.ErrCodeConnectionFailed` identifier; 2 handler_test assertions updated): GREEN PASS. **Task 4** (4 E2E files: download-filtering + qbittorrent-settings + downloads + downloads.api): done, **+2 bonus sites found** at `downloads.api.spec.ts:84, 120` beyond draft AC #5 scope — fixed for AC #2 zero-hit invariant. **Task 5** (Rule 7 sync: project-context.md:294,300,7 + instructions.xml:97,111,113,146): all 6 sync points updated; sync dates bumped 2026-04-20 → 2026-04-24. **Task 6** (full regression gate): `pnpm lint:all` 0 errors / 129 pre-existing warnings (identical baseline); Prettier PASS; Go suite 29 packages PASS; `pnpm nx test web` cached 1738 tests PASS + cleanup verified; final grep `"QB_[A-Z]` → 0 product/E2E hits; `QB_,|\bQB_\b` → 1 hit (project-context.md:7 historical Last Updated note, expected). Services using `qbittorrent.ErrCode*` identifiers (download_service.go, qbittorrent_service.go + tests — 8 call sites) auto-picked up new wire values via Go compiler — confirmed zero explicit consumer changes needed. 🔗 AC Drift: FOUND (authorized). 🔒 Rule 7 Wire Format: PASS (5 QBITTORRENT_ codes, all prefix-valid per updated Rule 7). 📎 Contract Stamps: NONE. 🎨 UX: SKIPPED. Sprint-status.yaml synced: `in-progress → review`. Final `review → done` is CR's responsibility. |
| 2026-04-24 | CR Amelia `/code-review` COMPLETE. Status `review → done`. Adversarial CR surfaced 4 findings (0 HIGH + 1 MEDIUM + 3 LOW), all auto-fixed per user [1]. **M1** (MED) Subtask count metadata drift across 3 sources (story:268 + story:276 + sprint-status:448) claimed 23/24; actual = 26 (3+4+3+4+6+6) — synced all three to 26. **L1** AC #4 Completion Notes count "8 test assertions" → corrected to 9 (matches sprint-status and commit message). **L2** Task 6.5 grep scope `apps tests/e2e libs` → widened to AC #2 canonical `apps/api/internal/ tests/ apps/web/ libs/` (outcome unchanged; 0 residuals in both scopes). **L3** Pre-existing Rule 16 violation at `tests/e2e/downloads.api.spec.ts:121` (`toBeTruthy()` on wire error code) → tightened to `.toMatch(/^QBITTORRENT_/)` matching sibling assertion at line 89. Zero product-code behavior changed by CR fixes (pure metadata + docs + test-assertion tightening). 🔒 Rule 7 Wire Format (self-result): PASS (5 QBITTORRENT_ codes prefix-valid). 🔗 AC Drift: FOUND (authorized per Winston retro-10-AI3 Item 3; unchanged by CR). 📎 Contract Stamps: NONE. 🎨 UX: N/A (0 apps/web/ files). Post-CR regression gate: pnpm lint:all + go test ./... + pnpm nx test web — see Dev Agent Record. Files touched by CR: `_bmad-output/implementation-artifacts/followup-qbittorrent-prefix-rename.md` (Status + File List count + AC #4 count + Task 6.5 grep + Change Log + Completion Notes), `tests/e2e/downloads.api.spec.ts:121` (assertion tightening), `_bmad-output/implementation-artifacts/sprint-status.yaml:448` (review → done + count 23→26). |
| 2026-04-24 | Story bootstrapped to `ready-for-dev` by SM Bob via `/bmad:bmm:workflows:create-story` (yolo mode). Added: Story statement (As-a/I-want/So-that), 6 Tasks with 26 subtasks mapped to ACs #1–#8 (ACs revised by SM after exhaustive grep audit surfaced (a) 5th constant `ErrCodeTorrentNotFound` at `torrent.go:140` not in Winston's draft; (b) Rule 11 violation at `qbittorrent_handler.go:123` using raw literal — added as AC #3; (c) E2E scope understated — actual 4 files / 9 hits, enumerated in AC #5). Dev Notes with Root Cause + Why This is a Pure Wire-Contract Refactor + Architecture Constraint table + Cross-Stack Split Check (BE=3 + FE/E2E=1 + docs=1 → single story, pass) + Precedent Stories (retro-10-AI3 / followup-metadata-prefix-dedup / retro-10-AI5) + Grep Patterns (5 audit queries) + Risk Assessment (all 6 risk categories ZERO-LOW) + Ordering & Atomicity note. File List scaffolding (14 expected touch points). Dev Agent Record placeholder (retro-10-AI5 / followup-metadata-prefix-dedup audit-line pattern). Exhaustive artifact analysis: re-read `qbittorrent/types.go` + `torrent.go`, greped `apps/api/internal/`, `tests/e2e/`, `apps/web/`, `libs/` for `"QB_`, cross-checked services for identifier usage (4 consumer files, 8 hits, all already via `qbittorrent.ErrCode*` — compile-time rename is safe). Cross-Stack Split Check: 3 BE + 1 FE/E2E + 1 docs — single story OK. Sprint-status.yaml transition: `backlog → ready-for-dev`. |
