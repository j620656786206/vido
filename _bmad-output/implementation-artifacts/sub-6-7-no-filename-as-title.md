# Story 6.7: TMDb 未比對的片不得把檔名當 Title 送進 prompt（後端）

Status: review

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

- [x] **Task 1 — 略過邏輯（AC: #1, #2, #3）**
- [x] **Task 2 — 測試（AC: #4）**

## Dev Notes

- 與 P1-2（Cast/genres/countries 補齊）同區碼但不同目的：本 story 是**減雜訊**，P1-2 是**加訊號**；先做本 story 讓 P1-2 的對照乾淨。
- 純後端；`TranslateContext` `[@contract-v1]` 欄位不變，只是值的來源規則變——不 bump，記 Change Log。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「Metadata 注入實況」「發現 12」；後續 Backlog P0-7；`apps/api/internal/subtitle/media_store.go:95-112,204-212`、`apps/api/internal/ai/prompts/subtitle_translator.go:110-160`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Completion Notes List

- `prompts.LooksLikeFilename`（新檔 `looks_like_filename.go`）：三個訊號——影片副檔名結尾、`[` 開頭的 tracker／group 標籤、無空白且 ≥2 個點並帶 release token（解析度／WEB-DL／HEVC／x265／HDR／DV／點分隔年份）。保守：`Dune: Part Two`、`S.W.A.T.`、`Mr. Robot`、`Youth (2017)` 都不誤判；`Youth.2017.HQ` 判為檔名。
- `media_store.go` `withoutUntrustedIdentity(ctx, tmdbMatched, id, type)`：`!TMDbID.Valid` 或 `LooksLikeFilename(Title)` → 清空 Title／OriginalTitle／Year／Overview，`slog.Info` 一次帶 reason；Genres／Countries 保留（未比對時本來就空，已比對時是 TMDb 的）。電影路徑與 `seriesContext` 都套用（集數經 series）。效果：`BuildMetadataSection` 回空、`MetadataHash` 等於無 metadata 的 hash。
- 既有 fixture 修正：`TestMediaStore_LoadEpisodeUsesTheParentSeriesMetadata` 的 series fixture 補 `TMDbID`（它模擬的是已比對的影集；原 fixture 缺欄位只是省略，不是「未比對」語意）。
- 🔗 AC Drift: NONE (checked: 'TranslateContext|MetadataHash|seriesContext' across _bmad-output/implementation-artifacts/*.md — sub-1-5b AC #3.1「同影集各集 MetadataHash 相同」仍成立（未比對影集各集都得到同一個空 identity）；欄位與簽名不變；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (1 across 1 file — `TranslateContext` `[@contract-v1]`（sub-1-5b）：欄位、簽名、stubborn 政策皆不變，只是值的來源規則（未比對／檔名形狀 → 空），不 bump；Change Log 記錄)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🔌 Route Sync: N/A (no backend route touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- 全回歸：`pnpm nx test api` ✅、`pnpm nx test web` 255 files / 3125 tests ✅、`test:cleanup` 無殘留；`pnpm lint:all` go vet／staticcheck 過、eslint 0 errors、prettier 唯一紅字為未追蹤本機檔。

### Discovery Triage

- N/A — no out-of-scope work discovered

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `prompts.LooksLikeFilename` + `media_store.withoutUntrustedIdentity`（電影與影集路徑）；`TranslateContext` v1 值來源規則變更（不 bump）。 |
| 2026-09-04 | Task 2 — `LooksLikeFilename` 16 列表格測試；media_store 四條新測試（未比對電影／已比對但檔名形狀／正常片名保留／未比對影集）+ 既有 fixture 補 TMDbID。 |

### File List

- `apps/api/internal/ai/prompts/looks_like_filename.go`（new）+ `looks_like_filename_test.go`（new）
- `apps/api/internal/subtitle/media_store.go`（modified）+ `media_store_test.go`（modified）
- `_bmad-output/implementation-artifacts/sub-6-7-no-filename-as-title.md`（this file）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（status）
