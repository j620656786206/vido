# Story 9R-8: Metadata 感知翻譯 —— ASR 腿補接既有的作品語境區塊

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P1 S（比原估更小 —— 見下）· **Depends:** 9R-7 ✅、sub-1-5a/5b ✅
**Source:** sprint-status `9R-8-metadata-aware-translation`；`prd.md:153` 品質依賴（POC 名詞漂移：隱形戰士↔隱形特務、"The Deep"→深海怪物）。
**⚠️ 2026-08-19 authoring 重大盤點**：本 story 的多數原始範圍**已在 M1 出貨** —— 初稿經對抗驗證發現是照 7 月快照寫的，已依實況重列。本檔為重寫版。

---

## Story

As a NAS owner whose ASR-generated subtitles translate without knowing which work they belong to,
I want the ASR translation leg to feed the SAME media-metadata prompt section the extract leg already uses,
so that the majority path (68.3% of the sampled library) gets the proper-noun grounding the minority path has had since M1.

---

## Context —— 已出貨 vs 真缺口（逐行查證，2026-08-19）

**已出貨（本 story 一律不重做、不得重複建設）：**
- `prompts.MediaMetadata`（`subtitle_translator.go:116-124`）＋ `BuildMetadataSection`（`:129-159`，FR26，sub-1-5a）—— nil/零值 → byte-identical `''` 契約、`MetadataCastLimit=10`、完整單元測試（`_ZeroValueYieldsNoSection`/`_RendersEveryField`/`_PartialMetadataOmitsAbsentRows`/`_CapsCastAtTen`），pin 測試已含 metadata 區塊（`subtitle_translator_test.go:249`）。
- 快取指紋：`MetadataHash(tctx)`（`segment_cache.go:112-124`）已是 `RunVersion` 的一半（`:182-189`），參與每個 segmentKey（`:146-153`，sub-1-5b）；per-field key 分歧有測試（`segment_cache_test.go:170-177`）。
- **extract 腿端到端已接**：`media_store.go` loadMovie（`:92-106`）/seriesContext（`:193-201`）填 `TranslateContext` → `process_item.go` 傳入 `TranslateTrack` → `pipeline.go` buildSystemBlocks（`:887-905`）以 `metadataOf(tctx)` 渲染進 system blocks。
- **設計決定在案**：loadEpisode（`media_store.go:123-127`）**故意只注入父 series 層 metadata** —— per-episode 欄位會讓同劇每集的 MetadataHash 分裂、快取命中率砍半（程式碼註解明示）。

**真缺口（本 story 的全部範圍）：**
- ASR 腿的 `TranslateWithGlossaryHarvest`（`translation_service.go:148`）經 `BuildSubtitleTranslatorPromptWithGlossary` 組 prompt，**無 metadata 參數**；`translateSRT`（`transcription_service.go:823-863`）從不載入媒體 metadata —— 佔多數的路徑翻譯時對作品一無所知。
- ASR 腿**沒有 segment cache**（`splitCachedCues` 是 pipeline 專屬）→ **本案零快取工作、零 PromptVersion bump**（`m1-v2` 不動：規則是 prompt 文字變才 bump，本案只是把既有區塊接到新呼叫者）。

## Acceptance Criteria

### AC #1 — ASR 腿載入 metadata（fail-soft）
- `translateSRT` 以既有 `mediaType/mediaID` 查 Movie/Series repo，組 `prompts.MediaMetadata`（**重用既有 struct**，鏡射 `pipeline.go metadataOf(:903)` 的 TranslateContext→MediaMetadata 映射；不新增第三個 struct）。
- episode 沿用 extract 腿的**父 series 層**語境為基準；若要加 SxxEyy/集名，**只准加在 ASR 腿**（無快取無分裂代價）且需在程式碼註解引用 loadEpisode 的快取理由說明為何兩腿不同 —— 預設做法：與 extract 腿一致，只給 series 層（最小差異原則）。
- 查詢失敗 → 空 `MediaMetadata`，翻譯照跑（Rule 13；`BuildMetadataSection` 零值 → `''` 既有契約保證 prompt byte-identical）。

### AC #2 — 訊號貫穿到 prompt
- `TranslateWithGlossaryHarvest` 增 metadata 參數（新 variant 或 additive 參數，wrapper `TranslateWithGlossary` 同步；簽名策略與 bugfix-j 的 partial 訊號變更**協調一次改**，避免同檔兩次 churn —— Dev Notes 註記與 bugfix-j 的先後序）。
- 批次 prompt 前置 `BuildMetadataSection(meta)`（位置鏡射 extract 腿：進 system 區，不佔 user 批次額度）。
- 🔴 無 metadata 呼叫者 byte-identical（既有零值契約，測試釘）。

### AC #3 — 測試
- ASR 腿整合測試：metadata 有進 prompt（fake completer 斷言含 title/original_title）；repo 查詢失敗 → prompt 與現行 byte-identical。
- `TranslateWithGlossary` wrapper 透傳測試。
- `SubtitleTranslatorPromptVersion` **仍為 m1-v2** 的守衛斷言（防未來誤 bump）。
- 26 個 Claude-touching 測試＋retry guards 全綠。

### AC #4 — 範圍紅線
- 不動 extract 腿（已出貨，byte-unchanged）。
- 不動快取（ASR 腿無快取；extract 腿指紋已在）。
- cast 欄位：prompt/hash 管線已支援 Cast，**只缺 DB 填值**（`models.Movie.CreditsJSON:235` 是可行來源）—— 填值另案，本案 Cast 留空即可（零值契約吸收）。

## Tasks / Subtasks

- [ ] Task 1: AC #1 translateSRT metadata 載入＋MediaMetadata 組裝（fail-soft）
- [ ] Task 2: AC #2 簽名貫穿＋system 區前置（與 bugfix-j 協調）
- [ ] Task 3: AC #3 測試（含 version 守衛）
