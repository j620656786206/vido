# Story 6.7: TMDb 未比對的片不得把檔名當 Title 送進 prompt（後端）

Status: done

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
- 📎 Contract Stamps: FOUND (1 across 1 file — `TranslateContext` `[@contract-v1]`（sub-1-5b）：欄位與簽名不變；值的來源規則改變屬 usage-semantics 變更，依 Rule 20 兩個消費者 sub-1-5b／sub-1-6 皆 done（forward-only frozen），無 stale-mark 可欠；Change Log 記錄。CR L9 修正原「不算 usage semantics」的錯誤敘述)
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
| 2026-09-04 | CR fixes — 規則改為 `prompts.UntrustedTitleReason`（兩條翻譯路徑共用，Rule 19 mirror）：**檔名形狀**才動 Title/OriginalTitle；未比對時才連 Year/Overview 一起清；未比對但片名乾淨（parser 或手動編輯）**保留**（H3、M4）。`transcription_service.go` `mediaMetadataFor` 兩臂套同一規則（H1）。`LooksLikeFilename`：`[` 開頭需再有點分年份才算（`[REC]` 不誤判，H2）；強 token（解析度／WEB-DL／BluRay／REMUX／x26x／HEVC／HDR10）有空白也算（M5）；移除 DV/DD/DTS/AAC 二字 token（M6）。log 改 Debug（M7）。測試：`[REC]`×3、spaced release、`Dun.DV.Two`、`Amélie.2001`、`2001: A Space Odyssey`、`M*A*S*H`、`Se7en`；media_store 加 loadSeries 路徑、未比對乾淨片名保留、已比對檔名形狀保留 Year/Overview、未比對含 genres 的 hash 斷言（M8）；services 加 `TestWithoutUntrustedIdentity_MirrorsMediaStoreRule`。參數改名 `tctx`（L10）；Rule 20 註記改述（L9）。 |

### File List

- `apps/api/internal/ai/prompts/looks_like_filename.go`（new）+ `looks_like_filename_test.go`（new）
- `apps/api/internal/subtitle/media_store.go`（modified）+ `media_store_test.go`（modified）
- `_bmad-output/implementation-artifacts/sub-6-7-no-filename-as-title.md`（this file）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（status）

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → all items resolved in-session (branch `fix/sub-6-4-5-7-code-review`) → **Approve**

Mandatory checks: Rule 7 PASS（0 codes）· Rule 20 N/A（無 bump token；見 L9）· Rule 25 N/A · Rule 19 PASS（subtitle→prompts 既有邊；mirror 同步義務本輪補齊）。

### Action Items

- [x] [H1] ASR 路徑（`transcription_service.go` `mediaMetadataFor`）未修 — 兩臂套 `withoutUntrustedIdentity`（services 側 mirror），單元測試 `TestWithoutUntrustedIdentity_MirrorsMediaStoreRule`。
- [x] [H2] `[REC]` 被誤判 — `[` 開頭需再有點分年份；測試 `[REC]`、`[REC] 2`、`[Rec]³ Génesis`。
- [x] [H3] 已比對列過度清空 — 已比對只清 Title/OriginalTitle，保留 TMDb 的 Year/Overview；測試。
- [x] [M4] 未比對一律清空會丟掉手動編輯 — 規則改為「檔名形狀才清」；未比對乾淨片名保留；含 genres 的 hash 斷言改為對 genres-only context。
- [x] [M5] 有空白的 release 名漏網 — 強 token 不受空白限制；測試 spaced release。
- [x] [M6] 二字 token 過度比對 — 移除 DV/DD/DTS/AAC；測試 `Dun.DV.Two`。
- [x] [M7] 全域 slog Info 重複噴 — 改 Debug（media_store 無注入 logger 為 sub-1-6 CR L1 既定；services 側用注入 logger）。
- [x] [M8] 測試缺口 — loadSeries 路徑、`[`-prefixed 真片名、spaced release、matched 保留欄位皆補。
- [x] [L9] Rule 20 敘述錯誤 — Completion Notes 改述（usage-semantics 變更、消費者皆 done 故無 stale-mark）。
- [x] [L10] 參數名遮蔽 `ctx` — 改 `tctx`。
