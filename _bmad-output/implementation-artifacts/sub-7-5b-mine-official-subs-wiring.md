# Story sub-7-5b: 掃描完自動從官方字幕學譯名、寫進詞彙表、設定頁一顆按鈕 — 後端＋前端

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-01，由 sub-7-5 拆出；依賴 sub-7-5a（`mine` 套件與 NAS 實測結果）。行號為 main `12a4f507`。 -->

## Story

身為 NAS 使用者，
我要掃描完片庫後 Vido 自己去讀同一部劇已有的官方繁中字幕、把學到的譯名放進那部劇的詞彙表，
也可以在設定頁按一顆按鈕重新學一次，之後翻沒字幕的那幾集就會用同樣的譯名。

## 背景（查到的事）

- `GlossaryScopeResolver.Resolve` 第一次解到 `tmdb:*` 時會呼叫 `GlossarySeeder.EnsureSeeded`（sub-7-3 的 lazy seeding 範例）——本單的「學習」可掛同一個 seam，或掛 `ScannerService.SetOnScanComplete`（`main.go:532` 已被 `postScanEnrichment` 佔用，callback 只有一個，要串）。
- 寫入：`GlossaryRepository.InsertIfAbsent`、`source=official_subtitle`、`confirmed=false`；既有 manual／confirmed 永不覆寫（insert-only 天然成立）。
- 排除 Vido 自產：7-5a 以檔名規則排除 `.zh-Hant.srt`；`subtitle_runs.output_path` 可再對照一次。
- SSE：`sse.EventNotification`（`hub.Broadcast`）。
- 設定頁：`apps/web/src/routes/settings/subtitle.tsx` 目前只有 `LocalizationLevelForm`；按鈕要 Sally 補 C9-D／C9-M 的一列（成本 $0 的按鈕，不走 ButtonCost）。
- ~~可選 LLM 精煉~~：7-5a 實測 Shadow and Bone 68 詞、抽查 19/20，**不做**。
- ⚠️ 7-5a 實測兩個前提：(a) `.zh-TW.hi` 檔名不保證官方——Scorpion S01 六集全是人人影視／ZiMuZu 字幕組檔，人名不一致、時間軸對不上；**學到的詞要帶來源檔名進 GlossaryPanel**，偵測到字幕組署名（檔頭「字幕組」「人人影視」「ZiMuZu」「翻譯：」）的檔跳過；(b) partial 判定用 ffprobe 實探（Supernatural 的 .mp4 沒有任何字幕串流，eval CSV 算錯）。

## Acceptance Criteria

1. **服務**：`services.OfficialSubtitleMiner`：給 series id → 列該劇所有集（`EpisodeRepository.FindBySeriesID`）→ 對每集 `mine.Classify`（ffprobe 軌＋sidecar 檔名）→ 可用的集抽軌／讀 sidecar → `mine.Align` → 合併全部段 → `mine.Mine`（`Known`＝該 scope 既有 `metadata`／`manual`／confirmed 詞）→ `InsertIfAbsent(scope, src→zh, source=official_subtitle, confirmed=false)`；回 `{episodes_used, terms_found, terms_inserted}`；單劇一次最多 N 分鐘、ffmpeg 走既有 ExtractGate。
2. **自動觸發**：掃描完成後，對「partial」影集（有 zh 的集 > 0 且沒 zh 但有英文軌的集 > 0）背景跑，純本機 $0；結果 `notification` SSE 一則（「從《Scorpion》的 13 集官方字幕學到 18 個譯名」）；失敗只 log。
3. **手動**：`POST /subtitles/glossary/mine?series_id=` 與設定頁「重新從官方字幕學習」按鈕（列出 partial 影集、逐劇進度）。
4. **測試**：partial 判定；Known 由既有詞彙表組成；insert-only 不覆寫；SSE 內容；handler 400／404／202。
5. **設計稿**：Sally 補 C9 的一列（按鈕＋結果行）→ Alexyu inline agent → 截圖。

## Tasks / Subtasks

- [ ] Task 0 — 設計稿（AC #5）
- [ ] Task 1 — `OfficialSubtitleMiner` 服務＋寫入（AC #1）
- [ ] Task 2 — 掃描後自動觸發＋SSE（AC #2）
- [ ] Task 3 — 端點＋設定頁按鈕（AC #3）
- [ ] Task 4 — 測試（AC #4）

## Dev Notes

- 先拿 7-5a 的 NAS 實測結果（Scorpion S01 ≥15 詞／≥90%）；不到門檻再考慮原單 AC #4 的 LLM 精煉，並另立 7-5c。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-5 拆出）。 |
