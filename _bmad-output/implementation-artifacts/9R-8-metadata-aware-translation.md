# Story 9R-8: Metadata 感知翻譯 —— ASR 腿補接既有的作品語境區塊

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P1 S · **Depends:** 9R-7 ✅、sub-1-5a ✅、sub-1-5b ✅、bugfix-j ✅（已 merged #239，簽名協調欠務歸零）
**Source:** sprint-status `9R-8-metadata-aware-translation`；`prd.md:153`（inject TMDb title / original-title / year / genre / overview / cast / production-countries as context）。

> **⚠️ 2026-08-21 RE-AUTHORED（第三版）。** 初稿（07 月）照舊快照寫、80% 範圍早已出貨；第二版（08-19）已重寫但**在 bugfix-j merge 前定稿且檔案在 Task 清單處截斷**（無 Dev Notes / Dev Agent Record）。本版逐行 grep 實查 `main` HEAD `bfcfe9d0`，更正三處對現況的錯誤假設（見下方「對第二版的更正」），並補齊全部 template 區段。

---

## Story

As a NAS owner whose ASR-generated subtitles are translated without knowing which work they belong to,
I want the ASR translation leg to feed the SAME media-metadata prompt section the extract leg already uses,
so that the majority path (68.3% of the sampled library, sub-3-1 完工紀錄) gets the proper-noun grounding the minority path has had since M1.

---

## Context —— 已出貨 vs 真缺口（逐行查證 @ `bfcfe9d0`）

### 已出貨（本 story 一律不重做、不得重複建設）

| 資產 | 位置 | 說明 |
|---|---|---|
| `prompts.MediaMetadata` | `internal/ai/prompts/subtitle_translator.go:116-124` | 7 欄位（Title/OriginalTitle/Year/Genres/Overview/Cast/Countries），**prompts 不可 import subtitle**（會循環），故各呼叫端自行映射 |
| `prompts.BuildMetadataSection` | 同檔 `:129-159` | FR26 區塊；**零值 → 回傳 `""`**（byte-identical 契約），`MetadataCastLimit = 10` |
| P11 pin 測試 | `subtitle_translator_test.go:235-259` | sha256 釘住**本檔所有 prompt surface**（含 `BuildMetadataSection`）＋ `assert.Equal(t, "m1-v2", …)`；digest `dd8e754f…` |
| extract 腿端到端 | `subtitle/media_store.go:79-165` → `subtitle/pipeline.go:927-951` | `loadMovie`/`seriesContext` 填 `TranslateContext` → `buildSystemBlocks` 以 `metadataOf(tctx)` 渲染成 system block[1] |
| 快取指紋 | `subtitle/segment_cache.go:112-124`、`:182-189` | `MetadataHash` 已是 `RunVersion` 的一半，參與每個 segmentKey（sub-1-5b） |
| episode 語境決策 | `media_store.go:127-131` 註解 | **故意只注入父 series 層 metadata** —— per-episode 欄位會讓同劇每集 MetadataHash 分裂、快取命中率砍半 |
| episode→series 解析 | `transcription_service.go:879-891` `glossaryMediaKey` | ASR 腿**已經**做過一次 episode→SeriesID 查詢（sub-5-5 CR H1） |

### 真缺口（本 story 的全部範圍）

1. `TranscriptionService.translateSRT`（`transcription_service.go:896`）**從不載入媒體 metadata**。
2. `TranslationService.TranslateWithGlossaryHarvest`（`translation_service.go:198`）以**寫死的** `prompts.SubtitleTranslatorSystemPrompt` 呼叫 `s.provider.CompleteText`（`:262`），**無 metadata 入口**。
3. 結果：佔抽樣庫 68.3% 的 ASR 路徑翻譯時對作品一無所知（POC 名詞漂移：隱形戰士↔隱形特務、"The Deep"→深海怪物）。

### 對第二版（08-19）的更正 —— dev 必讀

| 第二版寫的 | 實況（grep 實證） | 本版做法 |
|---|---|---|
| 「以既有 mediaType/mediaID **查 Movie/Series repo**」 | `TranscriptionService` **沒有** series repo。它只有 `stateReader SubtitleStateReader`（→`*models.Movie`，`:35-37`，main.go:562 已注入 `repos.Movies`）與 `episodeReader`（→`*models.Episode`，`:52-54`，main.go:566 已注入） | **movie metadata 零新接線**（重用 `stateReader`）；**只新增一個** `SeriesMetadataReader`＋setter＋main.go 一行 |
| 「簽名策略與 bugfix-j **協調一次改**」 | bugfix-j 已 merged（#239，squash `1c1415b`，2026-08-20），`TranslationOutcome` 回傳已定案 | 欠務歸零。本案改用**變參選項**，對既有呼叫端**零編輯** |
| 「episode 沿 series 層語境」需自行再查 episode | `glossaryMediaKey` 已在同一函式內解析出 SeriesID（`translateSRT:928`） | **重用 `glossaryKey`**，禁止第二次 episode 讀取 |

---

## Acceptance Criteria

### AC #1 — ASR 腿載入 metadata（fail-soft，零重複查詢）

1.1 `translateSRT` 新增一次 metadata 解析，產出 `prompts.MediaMetadata`：
- `mediaType == movie` → `s.stateReader.FindByID(ctx, mediaID)`（**既有欄位，不新增介面**）→ 映射 `Title / OriginalTitle.String / yearOf(ReleaseDate) / Genres / Overview.String / ProductionCountries→ISO codes`。
- `mediaType == episode` → **重用同函式上方已算出的 `glossaryKey`**（它就是父 SeriesID）→ 新的 `seriesReader.FindByID(ctx, glossaryKey)` → 映射 `Title / OriginalTitle.String / yearOf(FirstAirDate) / Genres / Overview.String`（**Countries 不填** —— 鏡射 `media_store.go:seriesContext`，series 表無 production_countries）。
- 其他 mediaType（`series` / 未知）→ 空 `MediaMetadata`。ASR 腿的 `transcribable` 閘門（`process_item.go:427`）本來就排除 series，此分支只是防禦。

1.2 **禁止第二次 episode 讀取**：episode 分支必須沿用 `glossaryKey`，不得再呼叫 `s.episodeReader.FindByID`。

1.3 **Fail-soft（Rule 13 case 3）**：任何一步失敗（reader 未接線＝nil、查無資料、DB 錯誤）→ 回傳**空 `MediaMetadata`**、記一行 `s.logger.Warn`、**翻譯照跑**。零值 → `BuildMetadataSection` 回 `""` → prompt byte-identical。
- ⚠️ `MovieRepository.FindByID`（`movie_repository.go:106-108`）與 `SeriesRepository.FindByID`（`series_repository.go:108-110`）**查無資料時回傳「包住 `sql.ErrNoRows` 的 error」，不是 `(nil, nil)`**（9R-10b CR H2 踩過這個坑）。實作必須以 `err != nil || row == nil` 兩者皆守。

1.4 新增窄介面 `SeriesMetadataReader interface { FindByID(ctx context.Context, id string) (*models.Series, error) }`（Rule 11：定義在 consumer 側 `internal/services/transcription_service.go`，緊鄰現有的 `SubtitleStateReader` / `EpisodeSubtitleStateReader`）＋ `SetSeriesMetadataReader(r SeriesMetadataReader)` setter（鏡射既有 5 個 `Set*` 的 nil-safe 慣例）。

1.5 `cmd/api/main.go` 於既有 setter 群（`:559-566`）加一行 `transcriptionService.SetSeriesMetadataReader(repos.Series)`。

### AC #2 — 訊號貫穿到 prompt（對既有呼叫端零編輯）

2.1 `TranslationService` 新增**變參選項**（鏡射同 package 已出貨的 `TranscriptionOption` 慣例，`transcription_service.go:348-366`）：
```go
type TranslateOption func(*translateConfig)
func WithMediaMetadata(md prompts.MediaMetadata) TranslateOption
```
`TranslateWithGlossaryHarvest(ctx, blocks, glossary, progressFn, opts ...TranslateOption)`；`TranslateWithGlossary` 同步加變參並 `opts...` 透傳。
- 🔴 **選變參而非新增位置參數**：位置參數會逼改 `translation_service_test.go:345/621/643/664/705` 五處＋wrapper；變參對全部既有呼叫端 byte-unchanged。
- 🔴 **不開第四個 `TranslateWithGlossaryXxx` 方法** —— 命名鏈已有 3 層（`Translate` → `TranslateWithGlossary` → `TranslateWithGlossaryHarvest`），再加一層是負債。

2.2 `TranslateWithGlossaryHarvest` 內把寫死的 system prompt 換成組合值，**在批次迴圈外算一次**：
```go
systemPrompt := prompts.SubtitleTranslatorSystemPrompt
if md := prompts.BuildMetadataSection(cfg.metadata); md != "" {
    systemPrompt = systemPrompt + "\n\n" + md
}
```
餵給 `:262` 的 `s.provider.CompleteText(batchCtx, systemPrompt, userPrompt, TranslationMaxTokens)`。
- **順序＝不變區在前、per-show 在後**，鏡射 extract 腿 `buildSystemBlocks`（block[0] 不變 prompt、block[1] metadata+glossary）。
- **無 metadata → `md == ""` → 完全不進 if → 與現行 byte-identical。**

2.3 🔴 **metadata 走 system 區，不得改動 `BuildSubtitleTranslatorPromptWithGlossary`（user prompt）**。理由：user prompt builder 在 `subtitle_translator.go`，被 P11 sha256 pin 釘住 —— 動它 = digest 變 = 被迫 bump `SubtitleTranslatorPromptVersion`，而 bump 會把 **extract 腿整個媒體庫的 segment cache 重新 key**（`RunVersion` 含 PromptVersion），付出全庫重譯的代價換零收益。

2.4 `TranslateRequest`（9R-13 通用入口，`:365`）**不動**。

### AC #3 — `SubtitleTranslatorPromptVersion` 維持 `m1-v2`（不 bump）

3.1 本案**不改動** `subtitle_translator.go` 內任何 prompt 文字 → P11 pin digest 不變 → **不 bump**。
3.2 理由必須寫進程式碼註解（在 2.2 的組合處）：bump 規則的目的是防「快取靜默回舊譯」；**ASR 腿沒有 segment cache**（`splitCachedCues` 是 pipeline 專屬），而 bump 會誤傷 extract 腿全庫快取。
3.3 新增守衛測試釘住這個決定（見 AC #4.4）。

### AC #4 — 測試

4.1 **ASR 腿整合測試（movie）**：wire 一個 fake `stateReader` 回傳帶 Title/OriginalTitle/Overview/Genres 的 `*models.Movie` → 走 `translateSRT` → fake completer 斷言收到的 **system prompt 含** `## Media context`、`- Title: …`、`- Original title: …`。
4.2 **ASR 腿整合測試（episode）**：fake `episodeReader` 回 `SeriesID` → fake `seriesReader` 回 `*models.Series` → 斷言 system prompt 帶的是**父 series 的** Title，且 `seriesReader` **只被呼叫一次**、`episodeReader` 在 metadata 路徑上**未被二次呼叫**（釘住 AC #1.2）。
4.3 **Fail-soft byte-identical 測試**：reader 回 `sql.ErrNoRows` 包裝錯誤（以及 reader 為 nil 兩種情境）→ 斷言 fake completer 收到的 system prompt **完全等於** `prompts.SubtitleTranslatorSystemPrompt`（字串相等，不是 `Contains`），且翻譯仍成功產出 zh 檔。
4.4 **版本守衛**：`assert.Equal(t, "m1-v2", prompts.SubtitleTranslatorPromptVersion)` 加註解說明「9R-8 刻意不 bump —— ASR 腿無快取，bump 會重 key extract 腿全庫」。
4.5 **wrapper 透傳測試**：`TranslateWithGlossary(..., WithMediaMetadata(md))` 的 system prompt 與 `TranslateWithGlossaryHarvest` 同參數時一致。
4.6 **零迴歸**：`translation_service_test.go` 既有 5 處 `TranslateWithGlossaryHarvest` 呼叫**一行不改**仍全綠（變參的證明）；`prompts` package 測試全綠（digest 不變）。
4.7 Rule 16：斷言必須是具體字串／呼叫次數，禁止 `assert.NotNil` 等空斷言。

### AC #5 — 範圍紅線

- ❌ **不動 extract 腿**（`subtitle/pipeline.go`、`media_store.go`、`segment_cache.go` 皆 byte-unchanged）。
- ❌ **不動快取**（ASR 腿無快取；extract 腿指紋已在）。
- ❌ **不碰 `TranslationOutcome` / verdict 三態**（bugfix-j 已定案）。
- ❌ **Cast 欄位留空**：prompt/hash 管線已支援 `Cast`，但**兩條腿目前都不填**（extract 腿 `loadMovie:99-106` 也沒填）。只在 ASR 腿填會讓兩腿語境分歧。`models.Movie.Credits`（`movie.go:248`，由 `scanMovie` 解析）是可行來源 → **另案**（見 Discovery Triage 指引）。
- ❌ 不處理 `backlog-asr-leg-unify-gated-pipeline`（ASR 腿併入閘門化 TranslateTrack）—— 本案是該長線重構前的**短線補接**，刻意在既有結構內完成。

---

## Tasks / Subtasks

- [x] **Task 1 — metadata 載入（AC #1）** `internal/services/transcription_service.go` + `cmd/api/main.go`
  - [x] 1.1 新增 `SeriesMetadataReader` 窄介面（緊鄰 `:48-54` 的 `EpisodeSubtitleStateReader`）＋ struct 欄位 `seriesReader`（放在 `:126-131` 的 sub-3-2 欄位群）
  - [x] 1.2 新增 `SetSeriesMetadataReader` setter（緊鄰 `:207` 的 `SetEpisodeSubtitleStateReader`）
  - [x] 1.3 新增 `mediaMetadataFor(ctx, mediaType, mediaID, glossaryKey) prompts.MediaMetadata` 私有 helper —— movie/episode 兩分支＋fail-soft warn；`err != nil || row == nil` 雙守
  - [x] 1.4 helper 內的 year 解析：`ReleaseDate` / `FirstAirDate` 皆 ISO 字串，取前 4 碼 `strconv.Atoi`，失敗回 0（鏡射 `media_store.go:yearOf:206-216`；⚠️ **不可 import subtitle** —— Rule 19 禁止 services → subtitle，需在 services 側自寫等價小函式並註解引用來源）
  - [x] 1.5 countries 映射：`movie.ProductionCountries` → `[]string` ISO codes（鏡射 `media_store.go:countryCodes`，同樣自寫）
  - [x] 1.6 `translateSRT` 於 `glossaryKey := s.glossaryMediaKey(...)`（`:928`）**之後**呼叫 helper
  - [x] 1.7 `cmd/api/main.go:566` 後加 `transcriptionService.SetSeriesMetadataReader(repos.Series)`＋一行說明註解

- [x] **Task 2 — 訊號貫穿（AC #2 / #3）** `internal/services/translation_service.go`
  - [x] 2.1 新增 `TranslateOption` / `translateConfig` / `WithMediaMetadata` / `newTranslateConfig(opts)`（鏡射 `transcription_service.go:348-366`）
  - [x] 2.2 `TranslateWithGlossaryHarvest` 加 `opts ...TranslateOption`；`TranslateWithGlossary` 同步加並透傳
  - [x] 2.3 批次迴圈**外**組出 `systemPrompt`，`:262` 改用它
  - [x] 2.4 於組合處寫下「不 bump PromptVersion」的理由註解（AC #3.2 逐字要點）

- [x] **Task 3 — 測試（AC #4）** `internal/services/transcription_translation_test.go` + `translation_service_test.go`
  - [x] 3.1 movie / episode 兩個整合測試（4.1 / 4.2，含呼叫次數斷言）
  - [x] 3.2 fail-soft byte-identical 測試 ×2（reader 錯誤、reader nil）（4.3）
  - [x] 3.3 版本守衛（4.4）＋ wrapper 透傳（4.5）
  - [x] 3.4 確認既有 5 處呼叫未改仍綠（4.6）

- [x] **Task 4 — 閘門（Rule 12 / Rule 15）**
  - [x] 4.1 `pnpm nx test api`（Go 全 package）
  - [x] 4.2 `pnpm nx lint api`（釘版 staticcheck-2026.1）
  - [x] 4.3 `pnpm run lint:all`（含全 repo `format:check`）
  - [x] 4.4 `gofmt -l internal/services cmd/api` 為空

---

## Dev Notes

### 重用清單（❌ 禁止重新發明）

| 需求 | 已存在的東西 | 位置 |
|---|---|---|
| metadata 區塊渲染 | `prompts.BuildMetadataSection` | `ai/prompts/subtitle_translator.go:129` |
| metadata 型別 | `prompts.MediaMetadata` | 同檔 `:116`（**不得開第三個 struct**） |
| movie 資料來源 | `stateReader.FindByID` → `*models.Movie` | `transcription_service.go:35`，main.go:562 已注入 |
| episode→series 解析 | `glossaryKey`（`glossaryMediaKey` 產物） | `transcription_service.go:879` |
| 欄位映射參考 | `metadataOf` / `loadMovie` / `seriesContext` | `subtitle/pipeline.go:941`、`media_store.go:99/196` |
| 變參選項慣例 | `TranscriptionOption` / `newTranscriptionConfig` | `transcription_service.go:348-366` |
| nil-safe setter 慣例 | 既有 6 個 `Set*` | `transcription_service.go:157-209` |

### 架構護欄

- **Rule 19（package 邊界）**：`services` **不得 import `internal/subtitle`**。`yearOf` / `countryCodes` 必須在 services 側自寫等價版本並在註解引用 `media_store.go` 出處（`OpenCCConverter` 介面就是這個 pattern 的先例，`transcription_service.go:56-62`）。`services` import `ai/prompts` 是既有且允許的（`:262` 已在用）。
- **Rule 11（介面定義在 consumer 側）**：`SeriesMetadataReader` 定義在 `services`，`*repository.SeriesRepository` 結構性滿足它。窄到只有 `FindByID`。
- **Rule 13（錯誤處理完整性）**：所有 fail-soft 吞掉的錯誤都要有 `s.logger.Warn`（Rule 2：slog only，本檔用 `s.logger`）。
- **Rule 20（AC 契約版本）**：`TranslateWithGlossaryHarvest` / `TranslateWithGlossary` **未帶 `[@contract-vN]` 戳記**（實查：`translation_service.go` 全檔只有 `:471` 的 harvest-trailer wire format 帶 v1）→ 前向唯一原則下屬隱含 v0，**不欠 bump、不欠 ack**。變參本就是 additive，即使已戳記也適用 `HarvestedTerms` / `default_budget_usd` 先例（additive on vN 不 bump）。
- **Rule 7（錯誤碼）**：本案**不新增任何錯誤碼**（全部 fail-soft，無新 sentinel）→ `code-review/instructions.xml` 零編輯。
- **Rule 16（測試斷言品質）**：AC #4.3 必須用字串**相等**（`assert.Equal`），不能用 `Contains` —— byte-identical 是本案的核心保證。

### 三個最容易踩的坑

1. **改錯 prompt 面**：把 metadata 塞進 user prompt builder → P11 pin 紅 → 被迫 bump → extract 腿全庫快取失效。走 system 區（AC #2.3）。
2. **`(nil, nil)` 假設**：兩個 repo 的 `FindByID` 查無資料回**包裝過的 error**，不是 nil-nil。只守 `row == nil` 會漏（9R-10b CR H2 的原案）。
3. **第二次 episode 讀取**：`glossaryKey` 已經是 SeriesID，再查一次 episode 是白費 DB round-trip，且兩次結果可能不一致。

### Testing standards

- Go 測試與被測檔**同 package 同目錄**（Rule 9）。既有 `transcription_translation_test.go` 已有 `translateSRT` 的整合測試骨架（`:174-292`，5 個案例）與 fake completer 慣例 —— 直接沿用該檔的 helper，不要另建 fixture 體系。
- `translation_service_test.go:336-705` 有 `TranslateWithGlossaryHarvest` 的既有斷言樣式可鏡射。

### Project Structure Notes

- 動到的檔案全部落在既有位置，無新目錄、無新 package：
  - `apps/api/internal/services/transcription_service.go`（介面＋欄位＋setter＋helper＋一處呼叫）
  - `apps/api/internal/services/translation_service.go`（option 型別＋兩處簽名＋system prompt 組合）
  - `apps/api/cmd/api/main.go`（一行 setter）
  - 兩個既有 `_test.go`
- 與統一結構無衝突：`services` 已是 Rule 4 分層裡的 service 層，`cmd/api/main.go` 已是唯一 composition root。

### Time-dependent visual coverage

- **N/A —— 本 story 不觸及任何 `apps/web/src/components/**` 檔案**（純 Go 後端）。無 wall-clock-reading 元件、無 fixture 基準需求。Rule 23 不適用。

### References

- [Source: `_bmad-output/planning-artifacts/prd.md#153`] — Metadata-aware translation (9R-8) FR 原文
- [Source: `_bmad-output/planning-artifacts/subtitle-pipeline-architecture.md#50`] — 「metadata context from 9R-8」列為 M1 已重用資產
- [Source: `apps/api/internal/ai/prompts/subtitle_translator.go#116-159`] — `MediaMetadata` / `BuildMetadataSection` / `MetadataCastLimit`
- [Source: `apps/api/internal/ai/prompts/subtitle_translator.go#11-21`] — P11 bump 規則原文
- [Source: `apps/api/internal/ai/prompts/subtitle_translator_test.go#230-259`] — P11 pin 測試
- [Source: `apps/api/internal/subtitle/pipeline.go#910-951`] — `buildSystemBlocks` 排序理由 + `metadataOf` 映射
- [Source: `apps/api/internal/subtitle/media_store.go#99-165`] — `loadMovie` / `loadEpisode`（父 series 語境的快取理由）/ `seriesContext`
- [Source: `apps/api/internal/services/transcription_service.go#27-62`] — 既有窄介面群（Rule 11 樣板）
- [Source: `apps/api/internal/services/transcription_service.go#879-891`] — `glossaryMediaKey`（episode→series 已解析）
- [Source: `apps/api/internal/services/translation_service.go#188-262`] — 三層 Translate 方法鏈 + 寫死的 system prompt 呼叫點
- [Source: `apps/api/internal/repository/movie_repository.go#100-114`] / [`series_repository.go#102-116`] — `FindByID` 的 not-found 契約
- [Source: `project-context.md#Rule 11 / 13 / 19 / 20 / 16`] — 介面位置、錯誤處理、package 邊界、契約版本、斷言品質
- [Source: sprint-status `backlog-asr-leg-unify-gated-pipeline`] — 本案刻意不做的長線重構

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (BMAD `dev-story`, 2026-08-21)

### Debug Log References

- **RED 實證（Task 1+2 共同驅動）**：先寫 `TestTranslateSRT_MovieMetadataReachesTheSystemPrompt`，在**尚未實作**的狀態下執行 →
  6 條 metadata 斷言全數失敗，失敗輸出印出的 system prompt 就是**未加工的** `SubtitleTranslatorSystemPrompt`（`... 不含 "- Year: 1998"` 等）。
  同一輪 RED 中，兩條 byte-identical 測試與版本守衛**已經是綠的** —— 它們釘的是既有行為，正是本案不得破壞的部分。
- **GREEN**：Task 2 → Task 1 → `cmd/api/main.go` 接線後，四條測試轉綠（`ok github.com/vido/api/internal/services 0.521s`）。
- **gofmt 既有漂移查證**：`gofmt -l internal/services` 列出 19 個檔案。`git stash` 回到 `origin/main` 後**同樣是 19 個**，
  且**本案觸及的 5 個檔案皆不在名單內**。依 9R-10a 先例（「本票七檔乾淨，全庫既有漂移未動」）不擴大範圍、不立條目；
  專案的實際閘門 `nx lint api`（go vet + 釘版 staticcheck-2026.1）全綠。

### Completion Notes List

**實作總結（4/4 task、5/5 AC）**

1. **AC #1 —— metadata 載入。** 新增窄介面 `SeriesMetadataReader`（Rule 11，定義在 consumer 側，`*repository.SeriesRepository` 結構性滿足）
   ＋ nil-safe `SetSeriesMetadataReader` setter ＋ `cmd/api/main.go` 一行接線。
   **movie 半邊零新接線** —— `SubtitleStateReader.FindByID` 回的就是完整 `*models.Movie`（main.go 早已注入 `repos.Movies`），
   authoring 期的實查結論在實作時成立。
   新增 `mediaMetadataFor(ctx, mediaType, mediaID, glossaryKey)`：movie 走 `stateReader`、episode/series 走 `seriesReader`，
   **episode 沿用上方 `glossaryMediaKey` 已解析出的父 SeriesID**，episode 列零二次讀取（`TestTranslateSRT_EpisodeUsesParentSeriesMetadata`
   以 `episodes.callCount == 1` 釘住）。
   兩個 helper `releaseYear` / `productionCountryCodes` 在 services 側自寫（Rule 19 禁 services → subtitle），
   註解引用 `media_store.go` 的 `yearOf` / `countryCodes` 出處，沿用同檔 `OpenCCConverter` 的重宣告先例。
2. **AC #1.3 fail-soft 實證。** 兩個 repo 的 `FindByID` 查無資料回**包 `sql.ErrNoRows` 的 error** 而非 `(nil, nil)`（9R-10b CR H2 踩過），
   故 `err != nil || row == nil` 雙守。測試 `TestTranslateSRT_MetadataLookupFailureKeepsThePromptByteIdentical` 直接餵
   `fmt.Errorf("...: %w", sql.ErrNoRows)`，斷言翻譯**仍成功產出 zh 檔**且 prompt 回退到 byte-identical。
3. **AC #2 —— 訊號貫穿，既有呼叫端零編輯。** 採**變參選項** `TranslateOption` / `WithMediaMetadata`（鏡射同 package 的 `TranscriptionOption`）。
   **實證**：`git diff --numstat` 顯示 `translation_service_test.go` 為 **103 added / 0 deleted** —— 既有 5 處
   `TranslateWithGlossaryHarvest` 呼叫**一個字元都沒改**仍全綠。
   新增 `composeSystemPrompt`：metadata 區塊接在不變 prompt **之後**（鏡射 extract 腿 `buildSystemBlocks` 的 block[0] 不變 / block[1] per-show 排序），
   且在**批次迴圈外組一次**（`TestTranslateWithGlossaryHarvest_MetadataComposedOncePerRun` 以 25 cues / 3 批次斷言三批 system prompt 相同）。
4. **AC #2.3 —— 沒有動 user prompt builder。** metadata 全程走 system 區；
   `TestTranslateWithGlossaryHarvest_MediaMetadataRidesTheSystemPrompt` 反向斷言 `UserPrompt` **不含** `"Media context"`。
5. **AC #3 —— `m1-v2` 不 bump。** `prompts/subtitle_translator.go` **零編輯** ⇒ P11 pin digest `dd8e754f…` 不變（prompts package 測試全綠即證）。
   理由寫進 `composeSystemPrompt` 的註解，並由 `TestSubtitleTranslatorPromptVersion_NotBumpedBy9R8` 釘住。
6. **AC #5 紅線全數守住。** `internal/subtitle/**` 一行未動（`git diff --numstat` 只有 5 個檔案，全在 `services` / `cmd/api`）；
   `TranslationOutcome` / verdict 三態未動；Cast 兩腿仍皆留空（見 Discovery Triage）。

**新增測試（9 條）**
- `transcription_translation_test.go`：`MovieMetadataReachesTheSystemPrompt`、`MetadataLookupFailureKeepsThePromptByteIdentical`、
  `NoReaderWiredKeepsThePromptByteIdentical`、`EpisodeUsesParentSeriesMetadata`、`EpisodeWithoutSeriesReaderStaysByteIdentical`、
  `SubtitleTranslatorPromptVersion_NotBumpedBy9R8`
- `translation_service_test.go`：`MediaMetadataRidesTheSystemPrompt`、`NoOptionsIsByteIdentical`、`ZeroMetadataIsByteIdentical`、
  `TranslateWithGlossary_ForwardsMediaMetadata`、`MetadataComposedOncePerRun`

**閘門結果（全部實跑）**
| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（Go 全 package，0 FAIL） |
| `pnpm nx test web` | ✅ 235 檔 / **2722 測試**全綠 |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 乾淨 |
| `pnpm run lint:all` | ✅ **0 errors** / 119 warnings（皆為 main 既有基準，本案未新增） |
| `prettier --check .` | ✅ All matched files use Prettier code style |
| `gofmt -l`（本案 5 檔） | ✅ 乾淨（全庫 19 檔既有漂移＝`origin/main` 同數，未動） |
| `pnpm run test:cleanup` | ✅ No test processes found |

**強制稽核項**

- 🔗 **AC Drift: NONE** —— grep `BuildMetadataSection` / `SubtitleTranslatorSystemPrompt` / `TranslateWithGlossaryHarvest`
  於 `_bmad-output/implementation-artifacts/*.md`，4 hits（本檔＋`bugfix-j`＋`sub-1-5a`＋`sub-5-5`），**全部 REUSE 非 DRIFT**：
  sub-5-5 AC #1 的 `===TERMS===` trailer 指令活在 `SubtitleTranslatorSystemPrompt` 內、本案**未編輯該常數**只在其後串接，
  wire 格式逐字不變（`MediaMetadataRidesTheSystemPrompt` 另加 `assert.Contains(sys, "===TERMS===")` 直接釘住）。
- 📎 **Contract Stamps: FOUND**（1 個上游引用 across 1 file —— **confirmed against `[@contract-v1]` (sub-5-5 AC #1，harvest trailer wire format)**：
  指令端文字未動、parser 端未動，v1 語意原封不動，本案**不 bump、不新增 stamp**。
  另查證：`TranslateWithGlossaryHarvest` / `TranslateWithGlossary` / `BuildMetadataSection` **皆未帶 stamp**
  （`translation_service.go` 全檔僅 `:471` 的 trailer wire 帶 v1）⇒ Rule 20 前向唯一原則下屬隱含 v0，不欠 ack；
  且變參本就是 additive，適用 `HarvestedTerms` / `default_budget_usd`「additive on vN 不 bump」先例。
  sub-1-5a AC #2 的 `[@contract-v1]`（`TranslateContext` / `TranslateResult` / `TranslateTrack`）位於 `internal/subtitle`，本案未觸及、不消費。
- 🔒 **Rule 7 Wire Format: PASS** —— 0 個新錯誤碼（全路徑 fail-soft，無新 sentinel）⇒ `project-context.md` 與 `code-review/instructions.xml` **零編輯**。
- 🔌 **Route Sync: N/A**（no backend route touched —— 本案不新增/不修改任何 HTTP 路由）。
- 🎭 **A11y Pre-Flight: N/A**（100% backend —— `git diff` 零 `apps/web/` 檔案）。
- 🎨 **UX Verification: SKIPPED** —— no UI changes in this story。
- 🕰️ **Rule 23: N/A** —— 未觸及任何 `apps/web/src/components/**`，無 wall-clock 相依元件。
- **Pre-existing failures: NONE**（`nx test api` / `nx test web` 於本分支皆 0 FAIL）。
  唯一的既有瑕疵是全庫 gofmt 漂移，已於 Debug Log 以 `git stash` 對照 `origin/main` 證明為既有、且非專案閘門，依 9R-10a 先例不立條目。

### Discovery Triage

**YES —— 發現 1 項超出範圍的工作。**

| Lane | 發現 | 追蹤條目 |
|---|---|---|
| ③ backlog-with-carry-forward-link | **Cast 欄位兩腿皆未填。** `prompts.MediaMetadata.Cast` 與 `MetadataCastLimit = 10` 早已出貨且被 P11 pin 測試涵蓋，`models.Movie.Credits` / `models.Series.Credits`（`scanMovie` / `scanSeries` 讀取時已解析）是現成來源，但 `subtitle/media_store.go` 的 `loadMovie` / `seriesContext` 與本案的 `mediaMetadataFor` **都沒填**。本案 AC #5 明確排除（只填一腿會讓 extract／ASR 兩腿語境分歧，且會改動 extract 腿的 `MetadataHash` ⇒ 全庫 segment cache 重新 key）。 | `backlog-metadata-cast-both-legs`（本次 dev-story 於發現時立案，雙向連結本 story） |

其餘：本案未發現任何需要 lane ① 或 lane ② 的項目。

### File List

| 檔案 | 變更 |
|---|---|
| `apps/api/internal/services/transcription_service.go` | modified —— `SeriesMetadataReader` 窄介面、`seriesReader` 欄位、`SetSeriesMetadataReader`、`mediaMetadataFor`、`releaseYear`、`productionCountryCodes`；`translateSRT` 傳入 `WithMediaMetadata` |
| `apps/api/internal/services/translation_service.go` | modified —— `TranslateOption` / `translateConfig` / `WithMediaMetadata` / `newTranslateConfig` / `composeSystemPrompt`；`TranslateWithGlossary(Harvest)` 加變參；批次呼叫改用組合後的 system prompt |
| `apps/api/cmd/api/main.go` | modified —— `transcriptionService.SetSeriesMetadataReader(repos.Series)` |
| `apps/api/internal/services/transcription_translation_test.go` | modified —— mock 記錄 `lastSystemPrompt`；+6 測試、+3 fake reader |
| `apps/api/internal/services/translation_service_test.go` | modified —— +5 測試（既有 5 處呼叫 0 行變更） |
| `_bmad-output/implementation-artifacts/9R-8-metadata-aware-translation.md` | modified —— tasks 全勾、Dev Agent Record、File List、Change Log、Status → review |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | modified —— `9R-8` ready-for-dev → in-progress → review；新增 `backlog-metadata-cast-both-legs` |

### Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **Task 1（AC #1）** —— ASR 腿載入 FR26 媒體語境。新增 `SeriesMetadataReader` 窄介面＋setter＋`main.go` 接線；`mediaMetadataFor` fail-soft 解析（movie 重用既有 `stateReader`，episode 重用 `glossaryKey` 的父 SeriesID，零二次讀取）。 |
| 2026-08-21 | **Task 2（AC #2 / #3）** —— 變參 `TranslateOption` / `WithMediaMetadata`（既有呼叫端 0 行編輯）＋ `composeSystemPrompt` 於批次迴圈外組一次，metadata 接在不變 prompt 之後。`SubtitleTranslatorPromptVersion` 刻意維持 `m1-v2`（ASR 腿無快取；bump 會重 key extract 腿全庫），理由入註解。 |
| 2026-08-21 | **Task 3（AC #4）** —— +11 測試：movie/episode 語境入 prompt、episode 走父 series 且 episode 列只讀一次、4 條 byte-identical 回退（reader 錯誤／reader nil／無選項／零值選項）、版本守衛、wrapper 透傳、每輪只組一次。 |
| 2026-08-21 | **Task 4（Rule 12 / 15）** —— 閘門全綠：`nx test api` 0 FAIL、`nx test web` 2722 測試、`nx lint api` 乾淨、`lint:all` 0 errors、prettier 乾淨、本案 5 檔 gofmt 乾淨、test cleanup 無殘留。 |
| 2026-08-21 | **Rule 24 lane ③** —— 立案 `backlog-metadata-cast-both-legs`（Cast 欄位兩腿皆未填），雙向連結本 story。Status → review。 |

---

## Senior Developer Review (AI)

**Reviewer:** Bob (SM) 代跑 `/code-review`，claude-opus-5[1m] · **Date:** 2026-08-21 · **Outcome:** APPROVED WITH FIXES

> ⚠️ **同 context 自審警告。** 本輪 review 由**實作者本人在同一個 session** 執行，不是換模型的獨立審查。
> 工作流程自己的建議是「用不同的 LLM 跑 code-review」。**強烈建議 PR 上另跑一輪跨模型審查**，
> 特別是 prompt 組合這種「錯了很難從測試看出來」的區域。

**Git vs Story File List：0 落差**（`git status` 的 7 個檔案與 File List 逐一對上；`.claude/memory/*` 的既有異動不屬本票，依 CR 指示排除 `_bmad/`、`_bmad-output/`、`.claude/`）。

### 強制閘門

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **PASS** —— 本次範圍的 3 個 Go 檔以 `Err[A-Z]\w*[^=]*= *"[A-Z][A-Z0-9_]*"` 掃描，**0 個錯誤碼常數**（全路徑 fail-soft，無新 sentinel）⇒ `project-context.md` 與 `code-review/instructions.xml` 零編輯 |
| 🔒 Rule 20 Contract Bump | **N/A** —— 本票 Change Log 與 code diff 皆無 `[@contract-vN→vM]` bump 列（唯一 grep 命中是 sprint-status 內 bugfix-j 的既有紀錄，非本次異動） |
| 🔒 Rule 25 Mega-line | **N/A** —— `project-context.md` 未被修改 |

### findings：0 HIGH / 4 MEDIUM / 3 LOW —— **M 全修、L 3/3 全修**

| # | 嚴重度 | 發現 | 處置 |
|---|---|---|---|
| **M1** | MEDIUM | **靜默降級無訊號。** `mediaMetadataFor` 的兩個 `reader == nil` 分支直接回零值、**一行 log 都沒有**。若哪天 `main.go` 的 setter 被重構掉，所有片子會安靜地變回「翻譯時不知道自己是哪部片」，沒有任何 log、且我自己的 `NoReaderWired…` / `EpisodeWithoutSeriesReader…` 測試反而是在**斷言這個行為是對的** ⇒ 測試不會紅。這正是 `main.go:535` 那段註解（sub-2-1a keyless boot「silently DEGRADED transcription run」）在講的同一類病。 | ✅ 修 —— 兩個 nil 分支各補 `s.logger.Warn`，明說「reader 未接線 ⇒ 無作品語境」 |
| **M2** | MEDIUM | **未揭露的成本變化 ＋ 零可觀測性。** ASR 腿**沒有 prompt caching**（實證：extract 腿的 `TranslateChunk` 會 `type-assert ai.CachingCompleter`（`translation_service.go:491`），而 `TranslateWithGlossaryHarvest` 只呼叫 plain `CompleteText`）⇒ media-context 區塊**每一批都重送、重計費**。batch=10 時一部長片約 120-180 批，約 200 token 的區塊 ≈ **每件多 25-35k input tokens**，且會讓 9R-11 的 run budget 天花板提早觸頂。story 與 log 對此完全沉默，操作者無法把「預算提早爆掉」或「名詞還是漂移」跟「語境到底有沒有套上」對上帳。 | ✅ 修 —— `translateSRT` 新增一行 `translating with media context` INFO（`metadata_applied` / `metadata_chars` / `resent_per_batch`），並在程式碼註解寫下成本估算與其來源；同時把「併軌後接上 caching 即可消除此成本」carry-forward 進 `backlog-asr-leg-unify-gated-pipeline` |
| **M3** | MEDIUM | **Rule 19 逼出來的兩個重複實作零直接測試。** `releaseYear` / `productionCountryCodes` 是 `media_store.go` 的 `yearOf` / `countryCodes` 的手抄副本（services 不得 import subtitle）。重複實作**一定會漂移**，而原本只有一條 happy-path 間接覆蓋（`"1998-07-14"`）。空字串、長度不足、非數字、空白 code 全無覆蓋。 | ✅ 修 —— 新增 `TestReleaseYear`（6 案表）與 `TestProductionCountryCodes`（nil／空 slice／正常／空白 code 剔除），逐案對齊 `media_store.go` 原版行為 |
| **M4** | MEDIUM | **誤導性的故障日誌。** `glossaryMediaKey` 在父劇集查詢失敗時**fail-soft 回退成 episode 自己的 id**。原實作接著拿這個 id 去查 **series 表**，然後記 `series_id=<其實是 episode id>` —— 操作者會跑去找一個從未存在的 series 列，真故障被包裝成正常故障。這是 **9R-10a CR M1 的同款問題**（「SQLite 鎖死時告訴使用者『找不到這一集』」）。 | ✅ 修 —— episode 分支先判 `glossaryKey == mediaID`（＝父劇集未解析）就直接誠實記 `parent series unresolved` 並返回，**不發那次查詢**；新增 `TestTranslateSRT_UnresolvedParentSeriesSkipsTheSeriesLookup` 以 `series.callCount == 0` 釘住 |
| **L1** | LOW | **兩腿 prompt 結構分歧。** extract 腿把 metadata 與 glossary 放在同一個 system block；ASR 腿的 metadata 進 system prompt、glossary 仍留在 per-batch **user** prompt（9R-7 原位）。兩者都送達模型故非行為差異，但是結構債。 | ✅ 修（記錄型）—— `composeSystemPrompt` 加 `KNOWN DIVERGENCE` 註解指向 `backlog-asr-leg-unify-gated-pipeline`，該 backlog 條目同步擴寫 |
| **L2** | LOW | **弱斷言。** `assert.NotContains(sys, "The Body")` 在 metadata 區塊**整段消失**時也會通過 —— 單看這一條證明不了任何事，它其實是靠同測試的 `Contains(Title: Buffy…)` 撐著。 | ✅ 修 —— 補註解說明它與上方正向斷言成對，缺一不可 |
| **L3** | LOW | **未 commit 狀態未記錄。** CR 步驟 3 明列「uncommitted changes not documented」為透明度項目。 | ✅ 修（記錄型）—— 於此處明載：截至 review 完成，7 個檔案位於 `feat/9R-8-metadata-aware-translation` 分支且**尚未 commit**，交由 `/ship` 處理 |

### 看過但**判定不是** finding（避免下一輪重查）

- **`m1-v2` 不 bump** —— 查證屬實：`prompts/subtitle_translator.go` 的 `git diff` 為**零行**，P11 pin digest `dd8e754f…` 不變（prompts package 測試綠即證）。bump 反而會重 key extract 腿全庫。判定正確。
- **sub-5-5 `[@contract-v1]` harvest trailer 未被破壞** —— `===TERMS===` 指令活在**未被編輯**的 `SubtitleTranslatorSystemPrompt` 常數內，本案只在其後串接；且 `MediaMetadataRidesTheSystemPrompt` 已直接 `assert.Contains(sys, "===TERMS===")`。
- **變參 additive 不需 bump** —— `TranslateWithGlossaryHarvest` / `TranslateWithGlossary` 自身**未帶 stamp**（`translation_service.go` 全檔僅 trailer wire 帶 v1）⇒ Rule 20 前向唯一原則下屬隱含 v0，不欠 ack。
- **episode／series UUID 撞號** —— 兩者皆 `uuid.New()`，碰撞機率可忽略；M4 的修法另外讓這條路根本不會走到。
- **`internal/subtitle/**` 零改動** —— `git diff --name-only` 只有 5 個 Go 檔，全在 `services` / `cmd/api`。extract 腿 byte-unchanged 屬實。
- **全庫 gofmt 漂移（19 檔）** —— `git stash` 對照 `origin/main` **同樣 19 檔**，本案 5 檔全乾淨；專案實際閘門是 `nx lint api`（go vet + staticcheck-2026.1）且全綠。依 9R-10a 先例不擴大範圍、不立條目。

### 修後閘門（全部重跑）

| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（34 packages、0 FAIL） |
| `pnpm nx test web` | ✅ 235 檔 / 2722 測試全綠（本票零 `apps/web/` 異動） |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 乾淨 |
| `pnpm run format:check` | ✅ 乾淨 |
| `gofmt -l`（本案 5 檔） | ✅ 乾淨 |

**最終 `git diff --numstat`（apps/api）**：`main.go` 4+/0− · `transcription_service.go` 170+/1− · `transcription_translation_test.go` 279+/0− · `translation_service.go` 68+/4− · `translation_service_test.go` **103+/0−**（後者再次證明既有 5 處呼叫一字未改）。

### Action Items

無 —— 4 個 MEDIUM 與 3 個 LOW **全數在 review 內修畢**，測試同步補齊。唯一 carry-forward 是既有的 `backlog-asr-leg-unify-gated-pipeline`（已擴寫）與 `backlog-metadata-cast-both-legs`（dev-story 立案）。

### Change Log（review 追加）

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **CR 修復 7/7** —— M1 nil-reader 靜默降級補 Warn；M2 補 media-context 可觀測性 INFO（含每批重送的成本估算註解）並 carry-forward 進 backlog；M3 補 `releaseYear` / `productionCountryCodes` 兩張 table test；M4 父劇集未解析時**不發**那次 series 查詢並誠實記錄（+1 測試釘 `callCount == 0`）；L1 兩腿 prompt 結構分歧註記；L2 弱斷言補配對說明；L3 未 commit 狀態載明。+3 測試（共 14）。閘門全部重跑綠。Status review → done。 |
