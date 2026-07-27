# Story sub-1.3: Error codes, SSE stages, and bilingual docs

Status: ready-for-dev

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

- [ ] **Task 1 — Go surface (AC #1, #2)**
  - [ ] 1.1 Add the 4 stamped constants to `engine.go`'s const block; verify `git diff` shows only that hunk.
  - [ ] 1.2 New `internal/subtitle/errors.go` with the 4 sentinels (exact strings from AC #2).
- [ ] **Task 2 — Tests (AC #6)**
  - [ ] 2.1 `TestPipelineStageValues` (12 exact literals) in `engine_test.go`.
  - [ ] 2.2 New `errors_test.go` — wire-format + `errors.Is` round-trip.
- [ ] **Task 3 — Registries + docs (AC #3, #4)**
  - [ ] 3.1 `project-context.md:288` `SUBTITLE_` line += 4 codes; verify the 16-prefix paragraph untouched; verify `instructions.xml` needs nothing (record in Completion Notes with the 13-4b citation).
  - [ ] 3.2 Prepend the Rule 25-compliant English-only mega-line entry; `prettier --check project-context.md`.
  - [ ] 3.3 Update both `sse-event-types` files with the AC #4 tables (same commit — Rule 17).
- [ ] **Task 4 — Architecture micro-sync + gates (AC #5, #6)**
  - [ ] 4.1 D6 stage list += `correcting`; delta-tree `engine.go` marker → ✏️(+4 consts only).
  - [ ] 4.2 `go test ./...` + `pnpm lint:all` green from the respective roots.

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

### Debug Log References

### Completion Notes List

_(Record here: the instructions.xml no-op verification with the 13-4b citation — AC #3.)_

### Discovery Triage

- **Did this story discover any work outside its current scope?** — **Two items, both absorbed (lane ①) at authoring time:**
  - **① `correcting` undocumented** — live on the wire since Stage 4.5 shipped, missing from both doc tables and from D6's "current stages". Absorbed into **AC #4** (doc rows) + **AC #5** (D6 correction).
  - **① instructions.xml sync ambiguity** — epic AC says "synced", the 13-4b precedent says code-list-only changes need no CR-workflow edit. Ruled and recorded in **AC #3** (verified-no-op).
- Reference: `project-context.md` Rule 24.

### File List
