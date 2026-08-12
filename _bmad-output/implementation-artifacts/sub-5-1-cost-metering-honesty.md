# Story 5.1: 成本記帳誠實 —— 費率同源、全路徑計帳、預設值與計數曝露（後端為主）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner,
I want every AI dollar the system spends to be recorded at the true rate, capped by the same ceiling, and every number the UI shows me to come from the server that enforces it,
so that "AI 花費上限" is a fact on every code path — not just on the consent batch.

## Context — 這個 story 為什麼存在

M3 第一波第一棒（epic seed 2026-08-12）。M2.5 讓「事前估價」誠實了；本 story 讓「事後記帳」跟上。收攏四條掛帳 backlog，authoring 盤點又挖出一個**未立案的更大缺口**（見 AC #3）：

| 掛帳 | 問題 |
| --- | --- |
| `backlog-selfhosted-asr-actual-cost` | `Budget.RecordASR`（`budget.go:127`）無條件套 $0.006/min——自架 ASR 部署事後記帳 100% 高估、上限提早觸發 |
| `backlog-gemini-cost-metering` | `ai/gemini.go` 零 Governor/Budget/retry 掛鉤——不只不計帳，整個 9R-11/9R-4 韌性堆疊都繞過 |
| `backlog-budget-default-config-endpoint` | F15 預算預填是 FE 常數 $5.00，operator 改 `AI_RUN_BUDGET_USD` 不跟隨 |
| `backlog-consent-toast-count-episodes` | F17 toast 計數走 movies-only preview，低估含影集的媒體庫 |
| **（新發現，吸收）** | **FR12／worker-pool 管線的 LLM 翻譯花費完全未計帳、不受任何上限管**——pool ctx 沒掛 Budget，`RecordLLM` 在該路徑是 no-op；`resolveBudget` 只活在 ASR 步驟內部。「AI 花費上限」在自動管線的翻譯段是虛的 |

## Acceptance Criteria

1. **自架 ASR 事後記帳歸零（費率同源）。** `EstimatedASRPerMinuteUSD(selfHosted)`（`budget.go:68`）維持唯一費率解析點。新增 `Budget.RecordASRWithRate(audioSeconds, perMinuteUSD float64)`；既有 `RecordASR` 改為委派（hosted 費率），簽名不變、既有測試不動。唯一生產呼叫點 `whisper.go:243-247` 改為 `b.RecordASRWithRate(dur, EstimatedASRPerMinuteUSD(c.isSelfHosted()))`，其中 `isSelfHosted()` = `c.baseURL != WhisperAPIURL`——**讀實際打的端點，不讀可能漂移的 config 旗標**（比 config 判定更誠實；`budget.go:63-67` 的 in-code 掛帳註解一併清除）。測試：自架 budget 記帳 $0、hosted 不變（`pricing_test.go:24` 既有測試成為 hosted 守衛，補自架雙生）。

2. **Gemini 與 Claude 同級（governed＋retry＋計帳）。** `gemini.go` 補上 claude.go:267-287 先例的完整堆疊：`governed()` 包裹（budget 前置短路＋rate token）＋ `retryTransient` ＋ 解析 `usageMetadata{promptTokenCount, candidatesTokenCount}` ＋ ctx 有 Budget 時 `RecordLLM`。`defaultLLMPricing` 補 Gemini 費率列（**不得**讓 Gemini 靜默落入 Haiku fallback 計出捏造數字——查當前 Gemini 官方定價填入，來源註明）。Governor 注入：`GeminiProvider` 加 option（比照 `WithWhisperGovernor`），`factory.go:44` 接線。

3. **FR12／自動管線翻譯納入計帳與上限（吸收的新發現，Rule 24 lane ①）。** `subtitle.Pipeline.ProcessItem` 的項目 ctx 在進入處理前掛 per-item Budget（鏡射 Route C `resolveBudget` 語意：ctx 已有 Budget 則沿用——**同意批次的共享 ceiling 不得被覆蓋**，`generation_batch.go:262` 的批次路徑行為 byte-不變；無 Budget 時以 `runBudgetUSD` 造 per-item budget）。效果：FR12 手動單項與任何 pool 路徑的翻譯＋ASR 花費都被記錄且受 `AI_RUN_BUDGET_USD` 上限管；觸頂沿用 `ErrBudgetExceeded` 既有分類（翻譯段觸頂＝該項 fail 語意，依 `governed()` 前置短路自然發生，不新增狀態）。`runBudgetUSD` 的注入循 `WithModelID` 先例加 pipeline option，`main.go` 接 `cfg.AIRunBudgetUSD`。

4. **檔名解析路徑明文裁定為不計帳（記錄，不實作）。** 盤點事實：`AIService.ParseFilename/ParseFansubFilename/GenerateKeywords` 全路徑無 Budget ctx——**兩個 provider 都一樣**，Gemini 補了 `RecordLLM` 在該路徑仍是 no-op。本 story 不為解析路徑掛 budget（掃描解析量大、語意是背景 metadata 非使用者同意的生成花費，且無單一「run」邊界）；於 `ai_service.go` 檔頭與 `docs/deployment.md` 記錄此設計事實，並立案 `backlog-parse-path-ai-metering`（lane ③）供未來裁定（選項：process 級 observability 計數器，非 capping）。

5. **預算預設值曝露（ride 候選信封）。** `AnalysisSnapshot` 加 `DefaultBudgetUSD float64`（`json:"default_budget_usd"`，值＝`cfg.AIRunBudgetUSD`，經 `NewGenerationCandidateService` 新參數注入——`selfHostedASR` 參數即先例，`main.go:785-793`）。**Rule 20**：sub-4-1 AC #7 `[@contract-v1]` 的 additive 欄位，不 bump、記 ack 與 Change Log；同時把缺席的 inline stamp 註解補到 `generation_candidates_handler.go`（比照 `generation_batch_handler.go:28`）。**不開新 `/config` 端點**（一個 float 不值得新公共面＋新 Rule 7 考量）。

6. **F15 預算預填改讀後端。** `CandidateAnalysisSnapshot`（FE 型別）加 `defaultBudgetUsd?: number`；`GenerationConsentView.bootstrap` 在**分支前**（ready／analyzing／kick 三出口都要吃到）以 snapshot 值 `setBudgetText(v.toFixed(2))`，`isCancelled()` token 守衛；`DEFAULT_BUDGET_TEXT = '5.00'` **保留**為 fallback（error phase／舊伺服器），header doc 的 backlog 註解更新為已解。WYSIWYG 語意不變（送出值仍＝畫面值）。

7. **F17 計數含影集（additive key）。** `EpisodeRepository.CountMissingZhHantSubtitle`（鏡射 `:151` 既有述詞的 ~8 行 twin）＋ `generationEpisodeFinder` 窄介面加該方法（nil-guard 降級 movies-only；**測試 fakes 需補實作**）＋ `PreviewMissing` 回雙數字 ＋ preview 回應 additive key `total_items_including_episodes`（**`total_items` 語意不動**——batch scope=missing 仍 movies-only，兩數字各自誠實，handler 註解講清楚；additive-no-bump 循該檔「existing keys unchanged」先例）。FE：`GenerationBatchPreviewResult` 加欄位，`ScanProgress.tsx` 的 toast 計數改讀 `totalItemsIncludingEpisodes ?? totalItems`（舊伺服器 fallback）；文案「N 部影片缺繁中字幕」**維持設計定稿不改**（影集計入 N，zh 語境「影片」可涵蓋，authoring 裁定記錄於此）。`GenerationWorkspaceV2` 同 query 的既有消費（`:550`）零影響（additive）。

8. **測試。** 至少：(a) 自架 ASR 記帳 $0＋hosted 不變＋估價與記帳同源斷言；(b) Gemini：usageMetadata 解析、`RecordLLM` 進 ctx budget（claude_test.go:590 模板）、governed 觸頂短路、retryTransient 生效、pricing 列存在（非 fallback）；(c) pipeline per-item budget：FR12 路徑翻譯花費被記錄、觸頂 fail、**同意批次共享 budget 不被覆蓋**（ctx 已有 budget 沿用的斷言）；(d) episode Count twin（真 sqlite，movie_repository_test.go:1855 模式）；(e) preview 雙數字＋handler fake 更新；(f) FE：prefill 從 snapshot、fallback 保留、toast 讀新欄位含舊伺服器 fallback。全回歸閘門照常。

## Tasks / Subtasks

- [ ] **Task 1 — ASR 費率同源（AC: #1）**
  - [ ] `RecordASRWithRate` ＋ `RecordASR` 委派；`whisper.go` 呼叫點改造＋`isSelfHosted()`
  - [ ] 測試：自架 $0／hosted 守衛／同源斷言；清除 `budget.go:63-67` 掛帳註解

- [ ] **Task 2 — Gemini 同級化（AC: #2）**
  - [ ] usageMetadata 解析結構＋pricing 列（來源註明）＋`RecordLLM`
  - [ ] `governed()`＋`retryTransient` 包裹＋Governor option＋factory 接線
  - [ ] gemini_test 補 budget/governed/retry 案例（httptest fakes 帶 Content-Type）

- [ ] **Task 3 — pipeline per-item budget（AC: #3）**
  - [ ] `ProcessItem` ctx budget（已有沿用／無則造）＋pipeline option＋main.go 接線
  - [ ] 測試：FR12 翻譯計帳、觸頂 fail、批次共享 budget 不覆蓋

- [ ] **Task 4 — 解析路徑裁定記錄（AC: #4）**
  - [ ] `ai_service.go` 檔頭＋`docs/deployment.md` 記錄；`backlog-parse-path-ai-metering` 立案（雙向）

- [ ] **Task 5 — 預設值與計數曝露（AC: #5, #7 BE 半）**
  - [ ] `AnalysisSnapshot.DefaultBudgetUSD`＋service 參數＋inline stamp 註解補齊
  - [ ] episode Count twin＋窄介面＋`PreviewMissing` 雙數字＋additive key＋Swagger／註解

- [ ] **Task 6 — FE 消費與回歸（AC: #6, #7 FE 半, #8）**
  - [ ] prefill 改讀 snapshot（fallback 保留）；toast 讀 `totalItemsIncludingEpisodes`
  - [ ] 契約清點（sub-4-1 AC #7 additive ack、9R-16 preview additive 先例引用）＋全回歸

（後端 task 5 個、前端 1 個 —— 未觸發跨端拆分門檻。）

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 費率唯一解析點 | `EstimatedASRPerMinuteUSD(selfHosted)` `budget.go:68`（sub-4-1） |
| LLM 計帳先例（governed→retry→Record） | `claude.go:267-287`（nesting 理由在 `:260-266`，勿改順序） |
| Governor option 先例 | `WithWhisperGovernor`（whisper.go）；Gemini 已有 `WithGeminiBaseURL/HTTPClient` option 機制 |
| ctx budget 沿用語意 | `TranscriptionService.resolveBudget` `transcription_service.go:308-320` |
| config 值進 candidate service | `selfHostedASR bool` 參數（`generation_candidates.go:194`、`main.go:785-793`） |
| 影集缺字幕述詞 | `missingZhHantSubtitleEpisodeWhere` `episode_repository.go:151`（Count twin 直接引用） |
| repo Count 測試模式 | `movie_repository_test.go:1855-1892`（真 sqlite 三段斷言） |
| FE snapshot→state | `GenerationConsentView.bootstrap` `:105-140`（`isCancelled` token 已存在；prefill 要放分支前） |

### 關鍵決策（authoring 已裁）

- **自架判定讀 client 實際端點**（`c.baseURL != WhisperAPIURL`）非 config 旗標——config 與 client options 理論上可漂移，記帳跟著真實呼叫走。
- **批次共享 budget 絕不被覆蓋**：AC #3 的 per-item budget 只在 ctx 無 budget 時創建——同意批次的 ceiling 語意（sub-4-2）是不可回歸的紅線，測試釘住。
- **解析路徑不計帳＝明文設計**而非默默略過：兩個 provider 對稱、記錄＋立案，未來要 observability 計數器再裁。
- **`total_items` 語意凍結**：batch scope=missing 仍 movies-only（sub-4-2 AC #3 的一致性理由不變），新數字走 additive key，兩個數字各自對應自己的消費者。
- **F17 文案不改**：設計定稿「N 部影片」保留，影集計入 N（zh 語境可涵蓋；改文案要回設計輪，不值得）。

### seam 資料層觸及（retro-m2-AI3 慣例）

- `Budget`：純記憶體，零 DB。
- `WhisperClient.TranscribeWithLanguage`：讀 WAV 檔、打 ASR API；計帳走 ctx budget。
- `GeminiProvider.Parse`：純 HTTP；本 story 不讓它碰 DB。
- `Pipeline.ProcessItem`：既有觸及不變（subtitle_runs／movies／episodes／cache_entries）；AC #3 只加 ctx 包裝，零新表。
- `EpisodeRepository.CountMissingZhHantSubtitle`：`episodes` 表 COUNT，讀 only。
- `GenerationCandidateService`：注入純 config 值，資料層觸及不變。

### 已知限制（記錄，不在本 story 解）

- 解析路徑不計帳（AC #4 裁定，`backlog-parse-path-ai-metering` 追蹤）。
- Gemini 定價需人工維護（無 API 可查）；填入值需註明查證日期。
- per-item budget（AC #3）讓 FR12 每項各有 $5 上限——與 Route C 手動單項語意一致，但**跨項累計**仍無 process 級總上限（同意批次才有共享 ceiling）；屬既有語意，非本 story 引入。

### 契約姿態（Rule 20）

- sub-4-1 AC #7 `[@contract-v1]`（候選信封）：additive `default_budget_usd`，不 bump；ack ＋ Change Log ＋ 補 inline stamp 註解。
- 9R-16 preview（AC #3 lineage，未 stamp 面）：additive key 循「existing keys unchanged」先例，記錄不 bump。
- `RecordASR` 簽名不變（whisper_test 等消費者零改動）；`RecordASRWithRate` 為新 API。
- D2/D6/`transcription_*`/`generation_batch_progress`：皆不動。

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.`（FE 僅資料來源切換與新欄位讀取。）

### References

- [Source: sprint-status `epic-subtitle-pipeline-m3` seed] — A+B+D+G 裁定與 story 序列
- [Source: `apps/api/internal/ai/budget.go:63-68,108-145`] — 費率解析點、RecordASR/RecordLLM 現況
- [Source: `apps/api/internal/ai/whisper.go:20,110,243-247`] — baseURL 預設、唯一 RecordASR 呼叫點
- [Source: `apps/api/internal/ai/claude.go:260-287`] — governed→retry→Record 先例
- [Source: `apps/api/internal/ai/gemini.go:24,94-160,239-250`] — 零掛鉤現況與 usageMetadata 缺口
- [Source: `apps/api/internal/services/transcription_service.go:308-320`] — resolveBudget 沿用語意
- [Source: `apps/api/internal/services/generation_batch.go:262-264`] — 批次共享 budget（不可覆蓋紅線）
- [Source: `apps/api/internal/services/generation_candidates.go:137-150,188-207`] — 信封與 config 注入先例
- [Source: `apps/api/internal/repository/{movie,episode}_repository.go:898,935,151,160`] — 述詞與 Count
- [Source: `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx:17-20,42,105-140`] — prefill 現況
- [Source: `apps/web/src/components/scanner/ScanProgress.tsx:50-65`] — toast 計數消費者
- [Source: `project-context.md`] — Rule 3/7/11/17/19/20/24

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **① expand-scope-in-place** — FR12／worker-pool 管線 LLM 翻譯未計帳、不受上限（authoring 盤點發現，先前零掛帳）→ 吸收為 **AC #3**（本 story 的核心任務即為此類缺口）。
    - **③ backlog-with-carry-forward-link** — `backlog-parse-path-ai-metering`：檔名解析／fansub／keyword 路徑（兩個 provider 皆然）無 Budget ctx，計帳為 no-op。本 story 明文裁定不計帳並記錄（AC #4）；未來選項＝process 級 observability 計數器（非 capping）。非阻塞。

### File List
