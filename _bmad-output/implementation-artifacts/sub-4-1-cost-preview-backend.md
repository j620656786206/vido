# Story 4.1: 掃描解耦 + 字幕生成候選清單與成本估算（後端讀側）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner,
I want scanning my library to cost nothing, and a screen that tells me exactly which files would need paid speech recognition and roughly what it would cost,
so that I choose what to spend money on instead of a scan silently enqueueing my whole library.

## Context — 這個 story 為什麼存在

2026-08-07 生產首次啟用管線：按「掃描媒體庫」→ post-scan callback 自動 `EnqueueMissing` **整庫 1026 筆** → 其中約 2/3 走付費 ASR（whisper-1 $0.006/分鐘，整庫粗估 US$200±），**全程零金額提示、零確認**。NAS 已切回 `legacy` 止血。

Alexyu 裁定（2026-08-07）三件一體：**掃描只做 metadata + 掃描後篩選畫面 + 總預算上限**。D1 追加裁定（2026-08-10）：**批次要支援影集**。

UX 設計已定稿（PR #211，8 個畫面）：`_bmad-output/planning-artifacts/design-prompt-cost-consent-2026-08-09.md`。

**本 story 是 BE 的「讀側」**——回答「有哪些候選、各自要走哪條路、大概多少錢」。BE「寫側」（批次接受明確 id 清單、影集執行、使用者核准的預算上限）是 **sub-4-2**；FE 篩選畫面是 **sub-4-3**。

> **AC #1 可獨立先出貨。** 它拆掉自動派工，讓 `pipeline` mode 可以安全地在 NAS 重新開啟（手動單集生成不受影響）。若想早點解除生產封鎖，AC #1 合併後即可先部署，不必等整個 story。

## Acceptance Criteria

1. **掃描與字幕生成解耦。** `main.go` 不再把 worker pool 接進 scan-complete 回呼；掃描完成只跑既有的 metadata enrichment。`ComposeScanCallback(prev, nil, observe)` 已經支援這個降級（pool 為 nil 時原樣回傳 `prev`），所以這是接線變更而非邏輯重寫。**保留不變**：worker pool 本身、手動 FR12 端點（`main.go:761` `subtitlePipelineQueue`、`:915`）、`EnqueueMissing` 方法（sub-4-2 的批次會改用它的媒體型別感知版本）。`legacy` mode 行為 byte-identical。

2. **候選枚舉涵蓋電影與影集（D1）。** 新增一個窄介面同時枚舉兩種來源，沿用既有述詞與查詢，不自寫 SQL：`MovieRepository.FindMissingZhHantSubtitle`（`movie_repository.go:907`，述詞 `missingZhHantSubtitleWhere` at `:898`）與 `EpisodeRepository.FindMissingZhHantSubtitle`（`episode_repository.go:160`）。每筆候選帶媒體型別（`models.SubtitleRunMediaMovie` / `...Episode` 內部詞彙，非 TMDB 的 movie|tv）。

3. **探測式路線預測（新建，但零新邏輯）。** 新增 `subtitle` 套件的匯出入口（如 `PredictRoute(ctx, mediaPath) (RouteKind, error)`），**只探測不抽取**：`prober.Probe` → 已匯出的純函式 `SelectCandidates`（`extractor.go:110`）→ 目前未匯出的 `verdictWithoutTrack`（`router.go:155`）。判定二分：候選數 > 0 ⇒ 可抽取（便宜）；== 0 ⇒ 需 ASR（付費）。
   **Rule 19**：`services` 不可 import `subtitle`，所以估價服務要透過在 `services` 側定義的窄 port 取得預測，由 `cmd/api` 的 adapter 橋接——**比照 sub-3-2 的 `pipelineASRAdapter` 先例**（`cmd/api/asr_adapter.go`）。

4. **分析走「已存資料優先、缺才探測」。** 電影的 `subtitle_tracks`（JSON，含 `format` codec 欄位）由 post-scan enrichment 寫入（`enrichment_service.go:449` `applyFFprobeTechInfo`），可直接用 `IsTextSubtitleCodec`（`extractor.go:51`）判定，免探測。**三個已知覆蓋洞必須走現場探測**：(a) **影集完全沒有 tech-info 欄位**（`episodes` 表只有 `runtime`，見 migration 006）；(b) NFO 已提供 `VideoCodec` 時 `applyFFprobeTechInfo` 整個短路（`enrichment_service.go:457-462`）；(c) ffprobe 失敗時靜默略過。

5. **片長來源與誠實降級。** 依序：ffprobe 實際片長 → TMDb `runtime`（`movies.runtime` / `episodes.runtime`，分鐘，可為 null）→ 標記未知並用預設值估算。ffprobe 目前**沒有解析片長**（`ffprobe_service.go:165-168` 的 `ffprobeFormat` 只取 `Filename`/`Size`，但指令已帶 `-show_format`），需在 `MediaTechInfo` 加 `Duration` 欄位。回傳給前端的每筆候選要能表達「片長未知」（F15 設計已有「片長未知，以 45 分鐘估算」與 `≈` 標記）。

6. **成本估算（全新）。** 全代碼庫目前**沒有任何前瞻估價**，且定價常數全未匯出（`ai/budget.go`：`whisperPerMinuteUSD = 0.006` at `:30`、`defaultLLMPricing` at `:19`、`llmPricing()` at `:32`）。需在 `ai` 加匯出存取器（單一真實來源，不得複製費率）。估算規則：
   - ASR 路線：`分鐘 × ASR 費率`
   - **自架 ASR（`ASR_BASE_URL` 非空）費率視為 0 / 未知**——否則自架部署的估價會 100% 高估（`RecordASR` 目前無條件套 OpenAI 費率，`budget.go:96`）
   - 翻譯成本抽取前不可精算（取決於字幕行數），以每分鐘均價概算並標示為預估
   - **抽取路線的估價是下界**：`router.go:123-130` 顯示「有文字軌」的項目若 SDH 過濾後零剩餘 cue 仍會落到 ASR。此不確定性必須在回應中可表達（或於 Dev Notes 明確記錄為已知限制）。

7. **候選 API（新端點，不破壞既有）。** 新增回傳「清單 + 彙總」的端點；**既有 `GET /subtitles/generation-batch/preview` 維持原樣**（回 `{total_items}`、僅支援 `scope=missing`），因為現有 FE 仍在消費它。回應需含：每筆 `media_id` / `media_type` / 顯示標題 / 路線 / 片長與是否已知 / 預估金額，以及彙總 `extract_count` / `asr_count` / `estimated_total_usd`。Rule 3 信封與 Rule 7 錯誤碼沿用既有 `SUBTITLE_` / `TRANSCRIPTION_` 前綴，**不新增前綴**。

8. **分析進度可觀測（F14）。** 分析涉及 N × ffprobe，受既有 3 格 semaphore + 10s timeout 限制（`main.go:401`），對含大量影集的媒體庫是實際耗時操作，不可假裝瞬時。需提供進度（沿用既有 SSE hub 與現有事件詞彙；**不得**新增 D6 `PipelineStage` 值，該契約已 stamped）。設計 F14 顯示「分析字幕軌 234 / 1,247」+「本機執行，不會產生費用」。

9. **測試。** 至少涵蓋：(a) 掃描回呼不再派工、手動端點仍可用；(b) legacy mode 行為不變；(c) 枚舉同時涵蓋電影與影集；(d) 路線預測二分（有文字軌/只有圖片軌/無軌）且**不觸發抽取**；(e) 已存 `subtitle_tracks` 走快取路徑、缺者才探測；(f) 片長三段降級（probe → runtime → 未知）；(g) 自架 ASR 估價為 0/未知；(h) 彙總金額 = 各項加總。

## Tasks / Subtasks

- [ ] **Task 1 — 掃描解耦（AC: #1）**
  - [ ] `main.go:627` 的 `ComposeScanCallback` 不再傳 pool（保留 `postScanEnrichment`）
  - [ ] 更新 `scan_callback_test.go` 的 4 個測試以反映新語意（其中 `TestComposeScanCallback_NilPreviousCallbackStillEnqueues` 需重新定位）
  - [ ] 確認手動端點與 worker pool 生命週期不受影響（`main.go:761`、`:915`）

- [ ] **Task 2 — 路線預測入口 + Rule 19 接線（AC: #3）**
  - [ ] `subtitle` 套件新增 probe-only 預測入口，重用 `Probe` + `SelectCandidates` + `verdictWithoutTrack`（後者需匯出或包一層）
  - [ ] `services` 側定義窄 port；`cmd/api` 加 adapter（比照 `asr_adapter.go`）
  - [ ] 明確測試「預測不觸發 ffmpeg 抽取」

- [ ] **Task 3 — 片長與定價的資料來源（AC: #5, #6）**
  - [ ] `MediaTechInfo` 加 `Duration`；`ffprobeFormat` 解析 `-show_format` 已回傳的 duration
  - [ ] `ai` 套件加匯出定價存取器（ASR 每分鐘費率 + LLM 費率），既有未匯出常數維持唯一真實來源
  - [ ] 自架 ASR 的零費率/未知處理

- [ ] **Task 4 — 候選枚舉 + 估價服務（AC: #2, #4, #6）**
  - [ ] 窄介面同時枚舉電影與影集（沿用既有述詞，不自寫 SQL）
  - [ ] 已存 `subtitle_tracks` 快取優先、缺者現場探測的分析流程
  - [ ] 逐項估價 + 彙總

- [ ] **Task 5 — API 與進度（AC: #7, #8）**
  - [ ] 新端點（清單 + 彙總）；既有 preview 端點原樣保留
  - [ ] 分析進度事件（沿用既有 SSE 事件詞彙，不動 D6 stamped 契約）
  - [ ] Swagger 註解與 Rule 3 信封

- [ ] **Task 6 — 測試與文件（AC: #9）**
  - [ ] AC #9 的 8 類案例
  - [ ] 全回歸閘門：`pnpm nx test api` + `pnpm nx test web`
  - [ ] `docs/deployment.md` 補「掃描不再自動產生字幕」的行為變更說明（Rule 17 的 zh-TW 對照仍缺，見既有 `backlog-deployment-doc-zh-tw-twin`，本 story 不擴大範圍）

## Dev Notes

### 這個 story 的最大風險：大量東西「不存在」

Agent 盤點確認以下**全都要新建**，不要假設現成：前瞻成本估算（全代碼庫零）、ffprobe 片長解析、匯出的定價存取器、探測式路線預測入口、批次層級預算（見 sub-4-2）。既有的只有「事後記帳」（`Budget.RecordLLM` / `RecordASR`）。

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 缺繁中字幕枚舉（電影） | `MovieRepository.FindMissingZhHantSubtitle` `:907` + 述詞 `:898` |
| 缺繁中字幕枚舉（影集） | `EpisodeRepository.FindMissingZhHantSubtitle` `:160` |
| 文字軌 vs 圖片軌判定 | `subtitle.IsTextSubtitleCodec` `extractor.go:51`（已含 `hdmv_pgs_subtitle`/`dvd_subtitle`） |
| 候選挑選（純函式、已匯出） | `subtitle.SelectCandidates` `extractor.go:110` |
| 無可用軌的裁決 | `Router.verdictWithoutTrack` `router.go:155`（目前未匯出） |
| 電影已存的軌資訊 | `movies.subtitle_tracks` JSON（`{language, format, external, stream_index}`） |
| 費率常數 | `ai/budget.go:19,27,30`（未匯出，需加存取器） |
| Rule 19 跨界 adapter 先例 | `cmd/api/asr_adapter.go`（sub-3-2） |

### 為什麼 F14「分析中」畫面是必要的（而非過度設計）

三個覆蓋洞讓「純 DB 查詢」不可行：影集**完全沒有** tech-info 欄位、NFO 來源會短路 ffprobe、探測失敗靜默略過。含大量影集的媒體庫必然要現場探測 N 次，受 3 格 semaphore 節流——這是真實耗時，設計據實呈現。

### 已知限制（要記錄，不要在本 story 解）

- **抽取路線的估價是下界**：SDH 過濾後零 cue 仍會落到 ASR（`router.go:123-130`），此時實際花費高於預估。
- **預算上限是「軟」上限**：`Budget.Exceeded()` 用 `>=` 且在呼叫**前**檢查，所以不會擋下一個會超出的呼叫——實際花費可能略超上限。UI 文案不可承諾「絕不超過」。
- **自架 ASR 費率**：目前 `RecordASR` 無條件套 $0.006/分鐘，自架部署的**事後記帳**同樣高估；本 story 只修「事前估價」，事後記帳的修正不在範圍（可另行追蹤）。

### 契約姿態（Rule 20）

- D2 媒體狀態、D6 `PipelineStage`、`transcription_*` SSE：**本 story 皆不改**，只讀。`PipelineStage` 是 stamped 契約，AC #8 明文禁止新增值。
- 既有 `preview` 端點與 `generation_batch` 的 202 回應：**不動**（AC #7），避免破壞既有 FE。
- 新端點與新回應形狀是本 story 新產出，實作時視是否被 sub-4-3（FE）跨 story 消費決定要不要 stamp `[@contract-v1]`。
- **實作時必跑**：AC Drift Check 與 Contract Stamp Check（dev-story Step 2 的強制項）。

### seam 資料層觸及（retro-m2-AI3 的預先套用）

本 story 騎在既有 seam 上，先記錄其資料層觸及範圍以免重蹈 sub-3-1 中途縮限的覆轍：

- `MovieRepository.FindMissingZhHantSubtitle` → 只查 `movies` 表；**不含 series/episodes**
- `EpisodeRepository.FindMissingZhHantSubtitle` → 只查 `episodes` 表
- `enrichment_service.applyFFprobeTechInfo` → 只寫 `*models.Movie`；**episodes 無對應路徑**
- `FFprobeService.Probe` → 純讀檔案，不寫 DB
- `generationCandidateFinder`（`generation_batch.go:80`）→ 三個方法全為 movie-repo；**非電影被拒是「查不到」的副作用，不是顯式型別檢查**（sub-4-2 改造時要知道這點）

### Project Structure Notes

後端為主。預期觸及：

- `apps/api/cmd/api/main.go` — 掃描回呼解耦、新 adapter 接線
- `apps/api/internal/subtitle/` — probe-only 預測入口（`extractor.go` / `router.go` 既有零件重用）
- `apps/api/internal/services/` — 候選枚舉 + 估價服務、窄 port
- `apps/api/internal/ai/budget.go` — 匯出定價存取器
- `apps/api/internal/services/ffprobe_service.go` — `Duration` 欄位
- `apps/api/internal/handlers/` — 新端點
- 前端：**本 story 不動**（FE 是 sub-4-3）

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.` 後端 story，不新增或修改任何前端元件。

### References

- [Source: `_bmad-output/planning-artifacts/design-prompt-cost-consent-2026-08-09.md`] — 8 個定稿畫面、四條驗收準則、D1 裁定與 BE 範圍
- [Source: `apps/api/internal/subtitle/scan_callback.go:28`] — `ComposeScanCallback`，nil pool 即降級
- [Source: `apps/api/internal/services/generation_batch.go:80`] — `generationCandidateFinder`（movies-only 的來源）
- [Source: `apps/api/internal/subtitle/extractor.go:51,110`] — `IsTextSubtitleCodec` / `SelectCandidates`
- [Source: `apps/api/internal/subtitle/router.go:111,123,155`] — 抽取為無條件、SDH 歸零回落、`verdictWithoutTrack`
- [Source: `apps/api/internal/services/ffprobe_service.go:87,165`] — `Probe` 與丟棄 duration 的 `ffprobeFormat`
- [Source: `apps/api/internal/services/enrichment_service.go:449`] — `applyFFprobeTechInfo` 的 movie-only 與 NFO 短路
- [Source: `apps/api/internal/ai/budget.go:19,30,32`] — 未匯出的定價常數
- [Source: `apps/api/internal/database/migrations/006_media_entities_enhancement.go`] — `episodes` 表無 tech-info 欄位
- [Source: `project-context.md`] — Rule 3（回應信封）、Rule 7（錯誤碼）、Rule 11（窄介面）、Rule 19（`services ↛ subtitle`）、Rule 20、Rule 24

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **③ backlog-with-carry-forward-link** — `backlog-gemini-cost-metering`：`ai/gemini.go` 完全沒有 `Budget`/`Governor` 掛鉤，走 Gemini 的呼叫花費永遠記為 0 且 `Exceeded()` 不會觸發。**不影響本 story 的字幕估價**（字幕翻譯走 `claudeHolder`，`main.go:560`，有計費；Gemini 用於檔名解析），但它讓「AI 花費上限」在檔名解析路徑上是虛的。
    - **③ backlog-with-carry-forward-link** — `backlog-selfhosted-asr-actual-cost`：`Budget.RecordASR`（`budget.go:96`）無條件套用 $0.006/分鐘，自架 ASR 部署的**事後**記帳同樣高估。本 story 只修事前估價（AC #6）。

### File List
