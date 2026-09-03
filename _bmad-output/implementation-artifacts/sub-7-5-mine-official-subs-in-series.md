# Story 7.5: 同影集官方繁中字幕挖詞餵其他集（加速器②，後端）

Status: ready-for-dev

## Story

As a NAS owner whose Scorpion S01 has 13 official zh-TW episodes and 79 without,
I want Vido to learn the show's names from the official subtitles it already has,
so that the 79 paid translations use the same renderings the professionals did — for free.

## Context

eval-1「片庫實測」：partial 影集只有 8/80（10%），**但走翻譯的 190 集裡有 131 集（69%）落在這 8 部**。Scorpion 一部 13 → 79。這是 A 路線的宣傳點：「Vido 會讀你片庫裡已有的官方中文字幕，學會這部劇的譯名，再用同樣譯名翻沒有字幕的那幾集。」**範圍只限同影集／同系列，不跨片**（Alexyu 修正）。

**前置：** sub-7-1（scope）、sub-7-3（`GlossarySeeder` 介面）。

## Acceptance Criteria

1. **來源判定。** 對同 scope 的每一集，找「官方繁中」來源：外掛 `.zh-TW.*`／`.cht.*`／`.tc.*` sidecar，或內嵌 `chi/zho` 文字軌（**排除** Vido 自產 `.zh-Hant.srt`，以 `subtitle_runs` 對照）。同時需有英文文字軌（內嵌或外掛）作對齊母本。
2. **對齊。** 英中兩份 SRT 以時間重疊（IoU ≥ 0.5）配對成句對；一對多／多對一合併為段。無 LLM。
3. **抽詞（零成本）。** 英文段裡的候選專有名詞 = 首字大寫連續 token（排除句首、排除停用詞、長度 2–4 詞）；在中文段裡找對應：優先用 sub-7-3 已播種的 TMDb 名單（英文角色名 → 已知 zh），其次用「同一英文詞在 ≥ 3 個句對出現且中文段有 ≥ 3 次共同的 2–4 字子串」的共現規則。命中 → `InsertNew(scope, {src: zh})`，`source=official_subtitle`，`confirmed=0`。
4. **可選 LLM 精煉（預設關）。** 設定 `SUBTITLE_MINE_WITH_LLM=false`；開啟時把「共現規則沒解出的高頻專有名詞 + 其句對」一次送 Haiku 要 term map（每影集一次，成本估算顯示於 log；上限 $0.05／影集）。
5. **觸發。** 掃描完成後對 partial 影集自動跑（純本機、$0）；設定頁「字幕」區一顆「重新從官方字幕學習」按鈕（手動）。進度走 SSE `notification`。
6. **驗收樣本。** 用 NAS 片庫 Scorpion S01 跑：至少挖出 ≥ 15 個角色／地名，人工抽查 20 筆正確率 ≥ 90%；結果記 Completion Notes。
7. **測試。** 對齊（重疊、合併、時間偏移 ±300ms）、抽詞（正反例表、句首大寫排除）、共現規則、不覆寫既有詞、Vido 自產字幕排除、partial 判定與 `eval/scan-partial-zh.sh` 同一規則。

## Tasks / Subtasks

- [ ] **Task 1 — 來源判定 + 對齊（AC: #1, #2）**
- [ ] **Task 2 — 抽詞與寫入（AC: #3）**
- [ ] **Task 3 — 觸發與進度（AC: #5）**
- [ ] **Task 4 — 可選 LLM 精煉（AC: #4）**
- [ ] **Task 5 — 驗收樣本 + 測試（AC: #6, #7）**

（後端 5 task；FE 只有一顆按鈕，併入 sub-7-6 的設定頁工作或本 story 1 task。）

## Dev Notes

- 對齊與抽詞放 `internal/subtitle/mine/`（subtitle 套件內，可用 `ParseSRT`）；不得 import services（Rule 19）。
- 中文分詞不引第三方庫：用「2–4 字子串共現」的統計法就夠抓人名／地名；品質靠 AC #6 實測，不靠假設。
- 港式用詞來源（eval `candidates.md` 標 See S01E02「鎖匙」）→ `confirmed=0` 讓使用者在 GlossaryPanel 審。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「片庫實測」；`eval/scan-partial-zh.sh`；`eval/candidates.md`；`subtitle/extractor.go` `SelectCandidates`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
