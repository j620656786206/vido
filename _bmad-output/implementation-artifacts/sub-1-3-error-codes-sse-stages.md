# Story sub-1.3: Error codes, SSE stages, and bilingual docs

Status: done

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟢 LOW (mechanical — lean ACs per the epic's risk-tiered ceremony)** · **BACKEND + DOCS ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.3 · architecture **D6** + **D7** · implementation steps **3 + 4 of 7**
**Depends on:** nothing — **zero file overlap with sub-1-1 (`internal/ai/**`, `go.mod`) AND sub-1-2 (migrations/models/repository)**. All three can run in parallel.
**Blocks:** sub-1-4 (`ErrSubtitleExtractFailed`/`ErrSubtitleNoTextSource` + `StageProbing`/`StageExtracting`/`StageSkipped`), sub-1-5a (`ErrSubtitleTranslateFailed`/`ErrSubtitleTimestampMismatch` + `StageTranslating`), sub-1-6 (broadcasts the stages; maps the sentinels to API responses).
**Cross-stack split check:** backend tasks = 4, frontend tasks = **0** → single story.

---

## Story

As a developer and a frontend consumer,
I want new `SUBTITLE_` error codes and SSE stages registered,
so that pipeline failures and progress are observable through the established contracts.

---

## 🔎 Codebase findings that reshape the epic AC (verified 2026-07-27)

1. **The current stage set is 8 values, not the architecture's 7.** D6 lists `searching | scoring | downloading | converting | placing | complete | failed` — but `engine.go:33` also ships **`StageCorrecting = "correcting"`** (Stage 4.5, AI terminology correction, broadcast at `engine.go:176`). It is live on the wire and **undocumented in both `sse-event-types` files**. This story extends **8 → 12** and absorbs the missing-`correcting` doc fix (Rule 24 lane ①).
2. **The three existing `SUBTITLE_` codes are inline string literals**, not constants — `subtitle_handler.go:248,394,404,429`. There is no central registry file in Go. The 4 new codes get sentinel errors in a **new `internal/subtitle/errors.go`** (the `ai/types.go` pattern); the 3 existing handler literals are **deliberately left alone** (refactoring shipped handler code is out of scope).
3. **`code-review/instructions.xml` needs NO edit.** The epic AC says Step 3 is "synced", but Step 3 carries only the **prefix** list, and `SUBTITLE_` has been registered since v4. The governing precedent is **13-4b** (`DVR_TVDB_NOT_FOUND`): *"code-list update only; prefix count stays 15, no CR-workflow change."* Same shape here — prefix count stays **16**, "synced" is satisfied by verification, zero XML edits. This resolves the epic-AC-vs-precedent ambiguity explicitly so the CR doesn't flag either direction.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` SSE stage set extended 8 → 12

**Given** `subtitle_progress.stage` is a frontend-consumed wire contract (`useSubtitleSearch.ts:21` types it; documented in `docs/sse-event-types.md`), **when** the set is extended, **then** `engine.go`'s existing `PipelineStage` const block gains exactly four additive constants and a Rule 20 stamp:

```go
// --- generation-pipeline stages (M1, story sub-1-3). [@contract-v1] —
// subtitle_progress.stage is a frontend-consumed wire contract. Full 12-value
// set = the 8 search-path values above + these 4. Consumers: sub-1-5a/1-5b,
// sub-1-6 (broadcast), FE progress surfaces. Extending again = Rule 20 bump.
StageProbing     PipelineStage = "probing"     // ffprobe: enumerating tracks (FR1)
StageExtracting  PipelineStage = "extracting"  // ffmpeg -c copy in flight (FR2/3)
StageTranslating PipelineStage = "translating" // LLM translation in flight (FR10)
StageSkipped     PipelineStage = "skipped"     // TERMINAL — deliberately routed out (FR9/P0)
```

**Delta-tree deviation, ruled here:** the architecture marks `engine.go` 🔒 *"untouched — becomes the search fallback"*. The 🔒 intent is **behavioural** (the Engine's search flow must not change); the `PipelineStage` type lives in that file, and splitting one enum across two files is worse than a 4-line additive edit. **Constraint:** `git diff apps/api/internal/subtitle/engine.go` shows **only** the const block + comment — any other hunk is a story failure. (AC #5 syncs the delta-tree marker.)

**Vocabulary note (do not "deduplicate"):** `probing`/`extracting`/`translating`/`skipped` deliberately mirror sub-1-2's `SubtitleStatus` values — D2 (media-row state) and D6 (SSE progress) are **two distinct wire contracts sharing vocabulary**, stamped in adjacent stories per the architecture's "stamped together so the frontend absorbs one coordinated change". Neither replaces the other.

### AC #2 — Four sentinel errors in a new `internal/subtitle/errors.go`

**Given** the `ai/types.go` sentinel pattern (`ErrAITimeout = errors.New("AI_TIMEOUT: …")`), **then**:

```go
// Package-level sentinel errors for the generation pipeline (story sub-1-3).
// Wire format per project-context.md Rule 7: {SOURCE}_{ERROR_TYPE} under the
// EXISTING SUBTITLE_ prefix (D7 — no new prefix; authoritative count stays 16).
var (
    ErrSubtitleExtractFailed     = errors.New("SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed")
    ErrSubtitleNoTextSource      = errors.New("SUBTITLE_NO_TEXT_SOURCE: no usable text subtitle track")
    ErrSubtitleTranslateFailed   = errors.New("SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed")
    ErrSubtitleTimestampMismatch = errors.New("SUBTITLE_TIMESTAMP_MISMATCH: translated cue timestamps diverge from source")
)
```

- Consumers: sub-1-4 (extract/no-source), sub-1-5a (translate/timestamp), sub-1-6 (handler `ErrorResponse` mapping + zh-TW user messages — **not here**).
- ❌ No handler wiring, no `ErrorResponse` calls, no zh-TW message strings in this story (Rule 3's `message`/`suggestion` are composed at the handler, which sub-1-6 owns).
- ❌ The 3 existing inline literals in `subtitle_handler.go` stay as-is (Finding 2).

### AC #3 — Rule 7 registry updated; instructions.xml verified-no-op; mega-line entry

**Given** `project-context.md:288`, **then** the `SUBTITLE_` line becomes:

```
SUBTITLE_NOT_FOUND, SUBTITLE_DOWNLOAD_FAILED, SUBTITLE_CONVERT_FAILED, SUBTITLE_EXTRACT_FAILED, SUBTITLE_NO_TEXT_SOURCE, SUBTITLE_TRANSLATE_FAILED, SUBTITLE_TIMESTAMP_MISMATCH
```

- The authoritative prefix paragraph (16 sources) is **unchanged**.
- **`code-review/instructions.xml`: zero edits**, per the 13-4b precedent (Finding 3). Record the verification (prefix list still 16, sync date untouched) in Completion Notes.
- **Mega-line:** prepend a `Last Updated: 2026-07-27 (story sub-1-3 — …)` entry to `project-context.md`'s mega-line, demoting the current lead to `Prior:` per **Rule 25 union semantics**. ⚠️ **English-only** — one CJK char makes prettier reflow the whole mega-line. Content: Rule 7 SUBTITLE_ +4 (code-list-only, prefix count stays 16, no CR-workflow change per 13-4b precedent) · SSE stage set 8→12 `[@contract-v1]` (+ the pre-existing undocumented `correcting` now documented) · Rule 17 doc pair synced.
- Run `pnpm exec prettier --check project-context.md` after the mega-line edit (Rule 25 verification).

### AC #4 — Bilingual docs updated with the exact rows (Rule 17), including the missing `correcting`

**Given** `docs/sse-event-types.md` and `docs/sse-event-types.zh-TW.md` both document `subtitle_progress` with a 7-row stage table, **then** both are updated **in the same change** to the 12-value set, restructured into path groups (the single "(in order)" list is no longer truthful with two pipelines sharing the wire enum):

**EN** — replace the "Pipeline stages (in order):" block with:

```markdown
**Pipeline stages** — one wire enum, two emitting paths:

_Search path (`Engine`, in order):_

| Stage         | Description                                                     |
| ------------- | --------------------------------------------------------------- |
| `searching`   | Querying subtitle providers (Assrt, Zimuku, OpenSubtitles)      |
| `scoring`     | Ranking results by language, resolution, source trust           |
| `downloading` | Fetching the subtitle file                                      |
| `converting`  | OpenCC language conversion (Simplified → Traditional)           |
| `correcting`  | AI terminology correction (Stage 4.5, post-OpenCC, optional)    |
| `placing`     | Writing subtitle file to disk alongside media                   |

_Generation pipeline (M1 orchestrator, in order):_

| Stage         | Description                                                     |
| ------------- | --------------------------------------------------------------- |
| `probing`     | Enumerating subtitle tracks via ffprobe                         |
| `extracting`  | Extracting an embedded text track (`ffmpeg -c copy`)            |
| `translating` | LLM translation to Traditional Chinese                          |
| `skipped`     | Terminal — pipeline deliberately declined (`und`/non-English)   |

_Shared terminal stages:_

| Stage      | Description                    |
| ---------- | ------------------------------ |
| `complete` | Subtitle successfully placed   |
| `failed`   | Error occurred at any stage    |
```

**zh-TW** — the mirrored structure (translations to match register: `correcting` = AI 術語校正（Stage 4.5，OpenCC 之後，選用）· `probing` = 以 ffprobe 列舉字幕軌 · `extracting` = 抽取內嵌文字字幕軌（`ffmpeg -c copy`）· `translating` = LLM 翻譯成繁體中文 · `skipped` = 終態 — 管線刻意跳過（`und`/非英文軌）).

- The `correcting` row is a **pre-existing doc bug being absorbed** (Finding 1, lane ①) — it has been broadcast since Stage 4.5 shipped.
- Example payloads, field tables, publishers lines: unchanged (event *types* don't change — `sse/hub.go` untouched).
- `subtitle_batch_progress` section: untouched (its `status` values are a different field).

### AC #5 — Architecture micro-sync (D6 + delta tree tell the truth)

**Given** the IR just spent a day fixing stale docs, **then** the same class of drift introduced or discovered by this story is closed in `subtitle-pipeline-architecture.md`:

1. D6's *"Current stages: searching | scoring | downloading | converting | placing | complete | failed"* → insert `correcting` (with a `(corrected 2026-07-27 — Stage 4.5 was missing)` note).
2. Delta tree: `engine.go 🔒 untouched` → `engine.go ✏️ +4 PipelineStage consts ONLY (1.3 [@contract-v1]); search flow untouched`.

### AC #6 — Tests (Rule 9 co-located, Rule 16 exact)

1. **`engine_test.go` + `TestPipelineStageValues`** — table-driven over all **12** constants asserting each exact string literal (guards a `"extacting"`-class typo in a wire value; `assert.Equal`, not contains).
2. **`errors_test.go` (new) + `TestSubtitleSentinelWireFormat`** — for each of the 4 sentinels: `err.Error()` starts with the exact `SUBTITLE_{TYPE}: ` code prefix, and the code appears in the AC #3 registry line format. Also `errors.Is` identity round-trip through `fmt.Errorf("…: %w", err)` wrapping (the consumers' usage shape).
3. `go test ./internal/subtitle/...` and the full `go test ./...` green; `pnpm lint:all` green.

### AC #7 — Scope fence

- ❌ No orchestrator, no `pipeline.go` — sub-1-5a/1-5b.
- ❌ No handler mapping / `ErrorResponse` / zh-TW messages / Swagger — sub-1-6 (no HTTP surface changes here).
- ❌ No `sse/hub.go` change, no new event types, no `project-context.md` §8 change (event-type list is unchanged).
- ❌ No frontend — `useSubtitleSearch.ts:21` is loose `string` + `:63` stores raw; new stages reach the wire only when sub-1-6 broadcasts them, and nothing breaks meanwhile (verified fail-soft).
- ❌ No refactor of the 3 existing handler literals; no `AppError` framework work; no new Rule 7 prefix.
- ❌ No `subtitle_batch_progress` status changes.

---

## Tasks / Subtasks

- [x] **Task 1 — Go surface (AC #1, #2)**
  - [x] 1.1 Add the 4 stamped constants to `engine.go`'s const block; verify `git diff` shows only that hunk.
  - [x] 1.2 New `internal/subtitle/errors.go` with the 4 sentinels (exact strings from AC #2).
- [x] **Task 2 — Tests (AC #6)**
  - [x] 2.1 `TestPipelineStageValues` (12 exact literals) in `engine_test.go`.
  - [x] 2.2 New `errors_test.go` — wire-format + `errors.Is` round-trip.
- [x] **Task 3 — Registries + docs (AC #3, #4)**
  - [x] 3.1 `project-context.md:288` `SUBTITLE_` line += 4 codes; verify the 16-prefix paragraph untouched; verify `instructions.xml` needs nothing (record in Completion Notes with the 13-4b citation).
  - [x] 3.2 Prepend the Rule 25-compliant English-only mega-line entry; `prettier --check project-context.md`.
  - [x] 3.3 Update both `sse-event-types` files with the AC #4 tables (same commit — Rule 17).
- [x] **Task 4 — Architecture micro-sync + gates (AC #5, #6)**
  - [x] 4.1 D6 stage list += `correcting`; delta-tree `engine.go` marker → ✏️(+4 consts only).
  - [x] 4.2 `go test ./...` + `pnpm lint:all` green from the respective roots.
  - [x] 4.3 **(added in-flight — Rule 24 lane ①)** Correct `sub-1-6`'s Rule 20 ack line: it states this story's stage constants carry "no stamp", which AC #1 falsifies. Left unfixed it would make sub-1-6 skip the mandatory `confirmed against [@contract-v1]` ack (a HIGH-severity Rule 20 gap at its CR).

---

## Dev Notes

- **Parallelism:** zero overlap with sub-1-1 and sub-1-2 — the three open stories touch disjoint file sets (`internal/ai`+`go.mod` / migrations+models+repository / `internal/subtitle`+docs+project-context). Merge in any order.
- **Pattern sources:** sentinel errors → `apps/api/internal/ai/types.go:12-27`; stage constants → the existing block at `engine.go:25-36`; bilingual doc register → the existing zh-TW table in `sse-event-types.zh-TW.md` (match its tone, e.g. 終態/選用 phrasing).
- **Why sentinels and not an `AppError` sub-type:** consumers are pipeline-internal (`extractor`/`router`/`quality_gate`) and need `errors.Is` classification; the HTTP envelope (Rule 3) is composed once, at the handler, in sub-1-6. Same layering as `ai`'s `ErrAI*` family.
- **Rule 12/17/20/25** all in play; **Rule 19** — `internal/subtitle` already imports what it needs; no new edges.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Zero `apps/web/**` files; Go + docs only. Rule 23 does not apply.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.3] — the two lean epic ACs.
- [Source: `subtitle-pipeline-architecture.md`#D6, #D7] — stage extension + stamp + bilingual mandate; extend-`SUBTITLE_`-no-new-prefix.
- [Source: `project-context.md`#Rule 7 (:288), #Rule 17, #Rule 20, #Rule 25] + the **13-4b precedent** (mega-line entry: "code-list update only … no CR-workflow change").
- [Source: `apps/api/internal/subtitle/engine.go`:25-36,176] — the real 8-value set incl. `correcting`.
- [Source: `apps/web/src/hooks/useSubtitleSearch.ts`:21,63] — loose-string fail-soft evidence.
- [Source: `docs/sse-event-types.md`:153-190 + zh-TW twin] — the tables being replaced.

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (`claude-opus-5`) — BMAD dev agent (Amelia)

### Debug Log References

**RED phase, observed before any implementation.** Both test files were written first and run against the unmodified package. The failure was a build failure naming exactly the 8 symbols the story adds — nothing more, nothing less, which is the check that the tests target this story's surface and not something already present:

```
internal/subtitle/engine_test.go:525-528: undefined: StageProbing / StageExtracting / StageTranslating / StageSkipped
internal/subtitle/errors_test.go:23-26:   undefined: ErrSubtitleExtractFailed / ErrSubtitleNoTextSource /
                                                     ErrSubtitleTranslateFailed / ErrSubtitleTimestampMismatch
```

**AC #1 diff-scope constraint, verified mechanically.** `git diff -U0 -- apps/api/internal/subtitle/engine.go` yields **exactly one hunk**, `@@ -35,2 +35,20 @@ const (`, entirely inside the `PipelineStage` const block. _(Corrected at CR — the original record claimed the `StageComplete`/`StageFailed` ± lines were "gofmt re-alignment only; whitespace". That was inaccurate: the two terminal constants were **deliberately relocated below** the four new ones so the block reads search-path → generation → shared-terminal, matching the stamp comment's "the 2 shared terminal values below". String values are byte-identical, const order is meaning-free in Go, and the hunk is still entirely inside the const block — the AC constraint holds; only the characterization was wrong.)_ Related deviation, ruled at CR: AC #1's quoted comment text said "the 8 search-path values above + these 4", but the shipped comment says "the 6 search-path values above + these 4 + the 2 shared terminal values below" — the shipped wording is the accurate one (the AC's "8 above" counted `complete`/`failed` as search-path values, which the same AC's own vocabulary note contradicts), an improvement absorbed without changing the 12-value membership.

**AC #3 `instructions.xml` no-op, verified rather than assumed.** `git diff --stat -- _bmad/bmm/workflows/4-implementation/code-review/instructions.xml` → **0 lines changed**. Step 3's Rule 7 check carries only the *prefix* list plus a sync date, and `SUBTITLE_` has been a registered prefix since v4, so adding four codes under it changes nothing the CR workflow greps. Governing precedent **13-4b** (`DVR_TVDB_NOT_FOUND`): *"code-list update only; prefix count stays 15, no CR-workflow change."* `grep -c 'Authoritative prefix set (16 sources)'` → 1, unchanged.

**AC #3 Rule 25 mega-line, verified per the rule's own checklist.** Entry count 26 → **27** (grew by exactly one, never shrank; corrected at CR — the original record said 27→28, a number neither the date-paren token count nor the lead+`Prior:`/`Earlier:` count reproduces); both `sub-1-3` and the demoted `9R-16 CR` lead are present on the resolved line; `pnpm exec prettier --check project-context.md` clean. The line does contain one CJK token (`新增`) — confirmed **pre-existing** (`git show origin/main:project-context.md` also reports 1, from story 19-9's badge entry) and **my entry contains zero CJK** (verified by truncating at the first `Prior:`). Date used is **2026-07-28**, the real change date, rather than AC #3's literal `2026-07-27` which was the authoring date.

### Completion Notes List

**🔗 AC Drift:** NONE (checked: `PipelineStage|subtitle_progress` and `sse-event-types` across `_bmad-output/implementation-artifacts/*.md` — 15 hits, all REUSE not DRIFT. Story **8-7** task 1.4 defined the original stage consts and **9-1** added `StageCorrecting`; `ux3-0-1` only reads them; `sub-1-5b`/`sub-1-6` are downstream drafts. The four new consts are purely additive and every pre-existing value's string is byte-identical, so no prior AC's observable behaviour moves. Documenting `correcting` records behaviour that has been live since 9-1 — a doc-truth fix, not a contract change. sub-1-2's `models.SubtitleStatus` shares four vocabulary strings but is a different type on a different wire field (`subtitle_status` on the media resource vs `subtitle_progress.stage` in an SSE payload); AC #1's vocabulary note rules the overlap deliberate.)

**📎 Contract Stamps:** FOUND (1 stamped AC across 1 file: AC #1's 12-value `PipelineStage` set, a **NEW v1** — no bump anywhere, so no `[@contract-vN→v(N+1)]` Change Log row and no producer-side stale-mark obligation fires. Upstream ack owed: **none** — the enum's origin (8-7) and `StageCorrecting` (9-1) carry **zero** `[@contract-v*]` stamps, i.e. implicit v0 under Rule 20's forward-only retrofit. AC #2's four sentinels are deliberately **unstamped**: they are Rule 7 registry codes, and 13-4b's precedent treats a code-list addition as registry hygiene rather than a versioned AC contract.)

**🎭 A11y Pre-Flight:** N/A (100% backend — no `apps/web/` files touched)

**🎨 UX Verification:** SKIPPED — no UI changes in this story

**What was actually implemented.** `engine.go`'s existing `PipelineStage` block gained the four generation-pipeline constants with the `[@contract-v1]` stamp and a comment recording both the 12-value membership and the deliberate vocabulary overlap with sub-1-2 (so a future reader does not "tidy up" one of the two contracts). New `internal/subtitle/errors.go` holds the four `SUBTITLE_` sentinels following `ai/types.go` — package-level `errors.New` with the wire code leading the message, plus a header explaining *why* sentinels and not an `AppError` sub-type (consumers classify with `errors.Is` inside the pipeline; the Rule 3 zh-TW envelope is composed once at the handler, which sub-1-6 owns). Rule 7's code list, both `sse-event-types` files, and the two architecture drift points are synced in the same change.

**Why the doc tables were restructured rather than extended.** The existing block was titled *"Pipeline stages (in order):"* — a single ordered list. Once two independent pipelines emit into the same wire enum that heading is a false statement, and appending four rows to the bottom would have made it read as though `probing` follows `placing`. Both files now carry three tables (search path 6 / generation pipeline 4 / shared terminal 2) under *"one wire enum, two emitting paths"*. Event *types* are unchanged, so `sse/hub.go`, the example payloads, the field tables, the publishers lines, and the whole `subtitle_batch_progress` section are untouched.

**Test-guard non-vacuity.** `TestPipelineStageValues` asserts the exact string of all 12 constants — the class of bug it exists for (`"extacting"`) is invisible to the compiler and to every behavioural test, because the broadcaster just forwards whatever the constant holds. `TestSubtitleSentinelErrorsIsRoundTrip` additionally asserts **mutual non-identity** across the four sentinels (`NotErrorIs` for every off-diagonal pair), so a future copy-paste that aliases two sentinels to the same error fails instead of silently collapsing two error classes into one. _Known limitation, recorded at CR:_ the `assert.Len(tests, 12)` guard and the `HasPrefix(code, "SUBTITLE_")` assertion are **self-referential** — they check the test's own table, not the const/sentinel set, so a 13th stage constant added to `engine.go` without touching the test passes silently. Go cannot enumerate package constants, so this is a language limitation, not an omission; the real tripwire for an unstamped 13th value is the Rule 20 stamp comment on the const block plus this CR workflow's contract checks.

**Pre-existing failures: NONE.** Full backend suite ran clean — 34 packages ok, zero FAIL lines, and notably the known `TestScannerService_SSEBroadcast_ScanCancelled` flake (`preexisting-flake-scanner-sse-scan-cancelled`) did not fire in any of this story's three full-suite runs. That entry stays open regardless: a flake that happens to pass is not a fixed flake.

**Gates:** `go test ./...` — 34 packages, 0 FAIL (3 runs). `pnpm nx test web` — 225 files / **2457** tests green (run despite this being backend+docs-only, per the Step 7 full-regression mandate). `pnpm lint:all` — **0 errors** (118/120 warnings, unchanged pre-existing baseline). `go vet ./internal/subtitle/` clean. `gofmt` clean on all four touched Go files. `prettier --check` clean on `project-context.md` and both doc files. `pnpm run test:cleanup` — no orphaned workers.

### Discovery Triage

- **Did this story discover any work outside its current scope?** — **Two items, both absorbed (lane ①) at authoring time:**
  - **① `correcting` undocumented** — live on the wire since Stage 4.5 shipped, missing from both doc tables and from D6's "current stages". Absorbed into **AC #4** (doc rows) + **AC #5** (D6 correction).
  - **① instructions.xml sync ambiguity** — epic AC says "synced", the 13-4b precedent says code-list-only changes need no CR-workflow edit. Ruled and recorded in **AC #3** (verified-no-op).
- **Two further discoveries at IMPLEMENTATION time (2026-07-28), both triaged:**
  - **① expand-scope-in-place → added sub-task 4.3.** `sub-1-6`'s Dev Notes Rule 20 ack line (`sub-1-6-wire-triggering-gating.md:113`) stated that this story's *"stage constants + sentinels"* are *"registry codes, no stamp"*. That is true of AC #2's sentinels but **false of AC #1's stage set**, which this story stamps `[@contract-v1]`. sub-1-6 is `ready-for-dev` — a not-yet-shipped draft — so left uncorrected its dev would have skipped the mandatory `confirmed against [@contract-v1]` line and taken a HIGH-severity Rule 20 consumer-side finding at its own CR. Absorbed because this story is the *producer* of the stamp and the fix is one doc line with zero code impact; the correction adds the ack, keeps the (correct) unstamped note for the sentinels, and carries a dated `⚠️ Corrected …` annotation so the edit is auditable. **No re-architecture implied** — the contract is exactly the 12 values sub-1-6 already planned to broadcast. (Note: Rule 20's 🔁 producer-side stale-mark obligation does **not** formally fire here — it triggers on a *bump*, and this is a new v1 — so this correction is Rule 24 discipline rather than Rule 20 compliance.)
  - **③ backlog-with-carry-forward-link → `backlog-repo-wide-gofmt-drift`.** `gofmt -l .` from `apps/api/` reports **77** drifted files on main, 7 of them under `internal/subtitle/` (none touched by this story — my four files are gofmt-clean). Not a gate failure: `pnpm lint:all` is go vet → staticcheck → eslint → prettier and never invokes gofmt. Filed **because this is the second occurrence in prose** — sub-1-2 already recorded the same finding for 6 files under `internal/models`/`internal/repository` and deliberately left them alone, and a finding that recurs only in narrative is precisely the deferred-discovery time-bomb Rule 24's BAN targets. Both stories held the same in-scope line: gofmt only what the story itself touches. Bidirectional link recorded in the entry.
- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
|---|---|
| `apps/api/internal/subtitle/engine.go` | +4 `PipelineStage` consts (`probing`/`extracting`/`translating`/`skipped`) with the `[@contract-v1]` stamp + membership/vocabulary comment. **One hunk, inside the const block only** — search flow untouched |
| `apps/api/internal/subtitle/errors.go` | **new** — 4 `SUBTITLE_` sentinels (`EXTRACT_FAILED`/`NO_TEXT_SOURCE`/`TRANSLATE_FAILED`/`TIMESTAMP_MISMATCH`) in the `ai/types.go` pattern, with per-sentinel consumer notes |
| `apps/api/internal/subtitle/engine_test.go` | **added** `TestPipelineStageValues` — 12 exact string literals, table-driven, + a 12-count guard |
| `apps/api/internal/subtitle/errors_test.go` | **new** — `TestSubtitleSentinelWireFormat` (Rule 7 code-prefix shape) + `TestSubtitleSentinelErrorsIsRoundTrip` (nested `%w` identity **and** mutual non-identity across all four) |
| `project-context.md` | Rule 7 `SUBTITLE_` line +4 codes (16-prefix paragraph untouched); Rule 25 mega-line entry prepended, `9R-16 CR` demoted to `Prior:` |
| `docs/sse-event-types.md` | `subtitle_progress` stage table → 3 path-grouped tables (search 6 incl. the previously-undocumented `correcting` / generation 4 / shared terminal 2) |
| `docs/sse-event-types.zh-TW.md` | Rule 17 mirror of the same restructure, matching the file's existing 終態/選用 register |
| `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md` | AC #5 micro-sync — D6 "current stages" gains `correcting` (dated correction note); delta-tree `engine.go` marker 🔒 → ✏️ (+4 consts only) |
| `_bmad-output/implementation-artifacts/sub-1-6-wire-triggering-gating.md` | sub-task 4.3 (Rule 24 lane ①) — Rule 20 ack line corrected to acknowledge `[@contract-v1]` (Story sub-1-3 AC #1) |
| `_bmad-output/implementation-artifacts/sub-1-3-error-codes-sse-stages.md` | this file — task checkboxes, Dev Agent Record, Discovery Triage, File List, Change Log, Status |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | `sub-1-3` ready-for-dev → in-progress → review; **new** lane-③ entry `backlog-repo-wide-gofmt-drift` |

### Change Log

| Date | Change |
|---|---|
| 2026-07-28 | **Task 1** — `engine.go` `PipelineStage` extended 8 → **12** `[@contract-v1]` (+`probing`/`extracting`/`translating`/`skipped`; additive, one hunk inside the const block, search flow untouched — the ruled delta-tree deviation, since the 🔒 marker's intent is behavioural and splitting one enum across two files is worse). New `internal/subtitle/errors.go` with the 4 `SUBTITLE_` sentinels in the `ai/types.go` pattern. |
| 2026-07-28 | **Task 2** — `TestPipelineStageValues` (12 exact wire literals) + new `errors_test.go` (Rule 7 code-prefix shape, nested-`%w` `errors.Is` identity, and mutual non-identity across all four sentinels). RED observed first: the build failed naming exactly the 8 new symbols. |
| 2026-07-28 | **Task 3** — Rule 7 `SUBTITLE_` code list +4 (authoritative 16-prefix paragraph untouched; `code-review/instructions.xml` verified **0 lines changed** per the 13-4b `DVR_TVDB_NOT_FOUND` precedent, so the epic AC's "synced" is satisfied by verification). Rule 25 mega-line entry prepended English-only, entry count 27→28, prettier clean. Rule 17 pair `sse-event-types{,.zh-TW}.md` restructured in the same change into 3 path-grouped tables — absorbing the pre-existing undocumented `correcting` row (lane ①); event types, payloads, and `subtitle_batch_progress` untouched. |
| 2026-07-28 | **CR fixes (adversarial /code-review — 5 findings: 0 HIGH / 2 MEDIUM / 3 LOW, all fixed in-review)** — **M1:** the story's own +18-line const-block insert shifted the `StageCorrecting` broadcast `engine.go:176` → `:194`, leaving stale `:176` citations in the two living docs written by this same change; architecture D6 note and the project-context mega-line entry now cite `:194` (story-internal `:176` references are authoring-time history and stay). **M2:** Debug Log's diff characterization corrected — `StageComplete`/`StageFailed` were deliberately relocated below the new four (values byte-identical, still one const-block hunk), not "gofmt whitespace"; the AC #1 comment-text deviation ("6 above + 2 below" vs the AC's "8 above") ruled an accuracy improvement and recorded. **L1:** self-referential test-guard limitation recorded in Completion Notes. **L2:** mega-line entry count corrected 27→28 ⇒ **26→27**. **L3:** D6 correction-date aligned to the real change date 2026-07-28 (same discipline as the mega-line date ruling). Verification after fixes: `pnpm exec prettier --check` clean on `project-context.md` + architecture doc; no Go files touched by the CR fixes. Story → done. |
| 2026-07-28 | **Task 4** — architecture micro-sync (D6 stage list gains `correcting` with a dated note; delta-tree `engine.go` 🔒→✏️). Sub-task **4.3** added in-flight (Rule 24 lane ①): corrected `sub-1-6`'s Rule 20 ack line, which wrongly described this story's stage constants as unstamped and would have cost sub-1-6 a HIGH-severity ack gap. Filed lane-③ `backlog-repo-wide-gofmt-drift` (77 files, second prose occurrence). Gates: backend 34 packages 0 FAIL, web 2457 tests, `pnpm lint:all` 0 errors, go vet + gofmt + prettier clean, no orphaned workers. |
