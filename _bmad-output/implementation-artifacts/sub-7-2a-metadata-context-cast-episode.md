# Story sub-7-2a: 翻譯提示補上演員名單與集數行（提示側，後端）

Status: review

<!-- SM Bob create-story 2026-09-30，由 sub-7-2 拆出（依 feedback_split_oversized_stories：資料側要加欄位＋改 enrichment，
     提示側只接既有資料；切在「提示 vs 資料」這條縫）。原單 Context 有過時前提（sub-7-3 已讓 enrichment 存 credits），本張以實際碼為準。
     行號為 main `9f79863d`。 -->

## Story

身為用 Vido 翻字幕的人，
我已經為每部片付過 TMDb 的查詢，演員和角色名應該真的送到翻譯模型手上，
影集每一集也該讓模型知道「這是第幾季第幾集、叫什麼」，而不是只靠一句簡介猜人名和語氣。

## 背景（查到的事）

- `subtitle/pipeline.go:86` `TranslateContext.Cast` 存在、`MetadataHash`（`segkey.MetadataHash`）與 `buildSystemBlocks`（`metadataOf`）都已納入 Cast——但 `subtitle/media_store.go` 的 `loadMovie`／`seriesContext` **從未賦值**，`prompts.MetadataCastLimit=10` 是死碼。ASR 腿 `services/transcription_service.go:1687-1694, 1712-1719` `mediaMetadataFor` 同樣沒填。
- 資料**已經在 DB**：sub-7-3 起 enrichment 在 TMDb 比對時透過 `persistCredits` 寫 `movies.credits`／`series.credits`；`models.Movie.GetCredits()`／`Series.GetCredits()` 可讀，repository scan 在非空時填 `Credits` 指標。
- 集數行：`loadEpisode` 刻意套父影集 context（跨集共用 prompt-cache 前綴與 segment-cache key），所以集數資訊只能放在 **hash 之外、cache breakpoint 之後**。extract 腿的 chunk user prompt 由 `services/translation_service.go:TranslateChunk` 組（`BuildSubtitleTranslatorPrompt(blocks, contextBlocks)`），port `ChunkTranslator.TranslateChunk(ctx, sys, contextBlocks, blocks)`；ASR 腿的 system prompt 由 `composeSystemPrompt(md, level)` 組、無 caching。
- P11 pin：`TestSubtitleTranslatorPromptVersion_PinsPromptText` 指紋所有 prompt 表面；9R-8 有「組合既有段落不 bump 版本」的先例。

## Acceptance Criteria

1. **Cast 接上（兩條腿）。** 新 `models.Credits.CastLabels(limit)`：「`Name（Character）`」（角色名有值才加全形括號）、billing 順序、上限 `MetadataCastLimit`、空白略過；`Movie.CastLabels`／`Series.CastLabels` 優先用已解析的 `Credits`，否則從 JSON 讀。`media_store.go` 電影與 `seriesContext`、`transcription_service.go` 電影與影集分支都填 `Cast`——兩腿的 `MediaMetadata` 逐位元相同（segment cache 共用的前提）。
2. **集數行在 hash 與 cache 前綴之外。** `TranslateContext.EpisodeLabel`（[@contract-v1] additive，不進 `MetadataHash`、不進 `metadataOf`）；`loadEpisode` 填「`SxxEyy · Title`」（無片名只有 `SxxEyy`；沒有集數則空）。新 `prompts.BuildEpisodeSection(label)`；extract 腿 `buildSystemBlocks` 在 breakpoint（per-show block 的 `CacheTTL1h`）**之後**追加第三個 system block（`CacheTTLNone`）；ASR 腿 `composeSystemPrompt(md, level, episode)` 在 per-show 段之後追加，`WithEpisodeLabel` option。空 label 兩腿 prompt 逐位元不變。
3. **一次讀取。** ASR 腿的 episode row 只讀一次（`episodeRowFor`）同時供 glossary key 與集數行；既有測試 `episodes.callCount == 1` 維持。
4. **`metadata_hash` 語意。** Cast 進 hash → 既有 completed run 的版本不再命中屬預期；pre-flight 的 sidecar 閘門仍優先，**不會自動重跑**，只有 force 才會（付費）。`docs/deployment.md` 加一段。**不 bump PromptVersion**（沿 9R-8 先例）：集數段是 breakpoint 之後的 additive context，bump 會把兩腿整個快取庫重 key 卻零收益；pin 測試把 `BuildEpisodeSection` 納入指紋並更新 digest。
5. **測試。** models 7 條（格式／無角色／順序與上限／空白／nil／Movie 與 Series 從 JSON 退回）；prompts 2 條（集數段格式、空 label 無輸出）＋pin 更新；subtitle 5 條（電影 cast 進 context 且改變 hash、無 credits 無 cast、影集 cast＋集數行、兩集不同 label 同 hash、`episodeLabel`）；pipeline 1 條（第三 block 在 breakpoint 之後、hash 不變、無 label 仍兩塊）；services 4 條（`composeSystemPrompt` 順序與逐位元、option 不進 metadata、ASR 腿電影 cast 進 system prompt、集數行在 show context 之外且只讀一次 episode）。

## Tasks / Subtasks

- [x] Task 1 — `Credits.CastLabels` ＋ 兩腿接 Cast（AC #1）
- [x] Task 2 — `EpisodeLabel`／`BuildEpisodeSection`／兩腿的追加位置（AC #2, #3）
- [x] Task 3 — docs ＋ pin（AC #4）
- [x] Task 4 — 測試、api 全量、vet、lint:all（AC #5）

## Dev Notes

- **與原單 AC #3 的差異（🔗 AC Drift，刻意）**：原稿寫「Episode 行放 user prompt 首行」。實作放在 **breakpoint 之後的第三個 system block**——效果相同（不在 cache 前綴、不在 hash），但不用擴 `ChunkTranslator` port、不動 `BuildSubtitleTranslatorPrompt`（那是 P11 pin 的表面，改它才真的該 bump），也讓兩腿順序一致（invariant → per-show → episode）。
- Cast 是 hash 的一部分本來就寫在 `segkey.MetadataHash`（billing 順序不排序）——本張只是讓它終於有值。
- 原單 AC #2（genres 根因、影集 countries）拆到 **sub-7-2b**：查證結果是 `metadata/tmdb_provider.go` 從不填 `MetadataItem.Genres`（search 結果只有 `genre_ids`），enrichment 只在 NFO／IMDb／手動比對路徑（有拿 details）才寫 genres；movies 的 `production_countries` 也只有 scan 的 `ConvertTMDbMovieToModel` 會寫、enrichment 不寫；series 表**沒有** countries 欄位。修法＝比對後多拿一次 details（cache 24h）＋ series 加欄，屬資料側。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- eval-1「Metadata 注入實況」；`prompts/subtitle_translator.go`；`subtitle/media_store.go`、`pipeline.go:buildSystemBlocks`；`services/translation_service.go:composeSystemPrompt`、`transcription_service.go:mediaMetadataFor`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 見 Change Log；驗證數字記在 PR。

### Discovery Triage

- 見 PR「沒做的事」。

### File List

- apps/api/internal/models/movie.go、series.go、credits_cast_labels_test.go
- apps/api/internal/ai/prompts/subtitle_translator.go、subtitle_translator_test.go
- apps/api/internal/subtitle/pipeline.go、media_store.go、pipeline_test.go、media_store_test.go
- apps/api/internal/services/translation_service.go、transcription_service.go、translation_episode_test.go、transcription_translation_test.go
- docs/deployment.md
- _bmad-output/implementation-artifacts/sub-7-2a-metadata-context-cast-episode.md、sub-7-2b-metadata-context-genres-countries.md、sub-7-2-metadata-context-complete.md、sprint-status.yaml

## Change Log

| Date       | Change                                                                 |
| ---------- | ---------------------------------------------------------------------- |
| 2026-09-30 | create-story（SM Bob，自 sub-7-2 拆出）；dev-story（Amelia）→ review。 |
