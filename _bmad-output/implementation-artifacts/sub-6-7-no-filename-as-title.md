# Story 6.7: TMDb 未比對的片不得把檔名當 Title 送進 prompt（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want an unmatched file to be translated with *no* title context rather than with its raw release filename,
so that I stop paying to send `[bitsearch.to] Wake.Up.Dead.Man...NAHOM.mkv` to the model as if it were the show's name.

## Context — 這個 story 為什麼存在

eval-1 發現 12：Wake Up Dead Man TMDb 比對失敗，`movie.Title` 就是原始檔名；`media_store.go:104` 直接塞進 `TranslateContext.Title`，`BuildMetadataSection`（`subtitle_translator.go:137`）原樣 addRow。這是雜訊不是上下文，還會進 `metadata_hash` 影響 cache key。

## Acceptance Criteria

1. **未比對 → 略過 Title 行。** `media_store.go` 電影路徑與 `seriesContext`：當 `TMDbID` 無效（`!movie.TMDbID.Valid`／`!series.TMDbID.Valid`）時，`Title`／`OriginalTitle`／`Overview`／`Year` 全部留空（此時這些欄位也全來自檔名解析，同樣不可信）。`BuildMetadataSection` 的既有「全空 → 回空字串」路徑自然生效，prompt byte-identical 於無 metadata。

2. **檔名形狀防呆。** 即使 TMDb 有比對，若 `Title` 仍長得像檔名（含 `.mkv|.mp4|.avi` 副檔名，或 `[` 開頭的 tracker 前綴，或連續 `.` 分隔 ≥ 3 段），也略過——`prompts.LooksLikeFilename(s) bool` 小函式，表格驅動測試。

3. **可觀測。** 略過時 `slog.Info("metadata title skipped", "reason", "unmatched|filename-shaped", "media_id", ...)` 一次。

4. **測試。** (a) 未比對電影／影集 → metadata section 為空；(b) `LooksLikeFilename` 表格（正反例含中文片名、含年份的正常片名 `Dune: Part Two` 不得誤判）；(c) 已比對正常片名不受影響；(d) `metadata_hash` 在未比對案例下等於無 metadata 的 hash（cache 語意）。

## Tasks / Subtasks

- [ ] **Task 1 — 略過邏輯（AC: #1, #2, #3）**
- [ ] **Task 2 — 測試（AC: #4）**

## Dev Notes

- 與 P1-2（Cast/genres/countries 補齊）同區碼但不同目的：本 story 是**減雜訊**，P1-2 是**加訊號**；先做本 story 讓 P1-2 的對照乾淨。
- 純後端；`TranslateContext` `[@contract-v1]` 欄位不變，只是值的來源規則變——不 bump，記 Change Log。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「Metadata 注入實況」「發現 12」；後續 Backlog P0-7；`apps/api/internal/subtitle/media_store.go:95-112,204-212`、`apps/api/internal/ai/prompts/subtitle_translator.go:110-160`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
