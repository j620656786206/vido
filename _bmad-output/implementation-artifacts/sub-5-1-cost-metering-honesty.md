# Story 5.1: 成本記帳誠實 —— 費率同源、全路徑計帳、預設值與計數曝露（後端為主）

Status: done

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

- [x] **Task 1 — ASR 費率同源（AC: #1）**
  - [x] `RecordASRWithRate` ＋ `RecordASR` 委派；`whisper.go` 呼叫點改造＋`isSelfHosted()`
  - [x] 測試：自架 $0／hosted 守衛／同源斷言；清除 `budget.go:63-67` 掛帳註解

- [x] **Task 2 — Gemini 同級化（AC: #2）**
  - [x] usageMetadata 解析結構＋pricing 列（來源註明）＋`RecordLLM`
  - [x] `governed()`＋`retryTransient` 包裹＋Governor option＋factory 接線
  - [x] gemini_test 補 budget/governed/retry 案例（httptest fakes 帶 Content-Type）

- [x] **Task 3 — pipeline per-item budget（AC: #3）**
  - [x] `ProcessItem` ctx budget（已有沿用／無則造）＋pipeline option＋main.go 接線
  - [x] 測試：FR12 翻譯計帳、觸頂 fail、批次共享 budget 不覆蓋

- [x] **Task 4 — 解析路徑裁定記錄（AC: #4）**
  - [x] `ai_service.go` 檔頭＋`docs/deployment.md` 記錄；`backlog-parse-path-ai-metering` 立案（雙向）

- [x] **Task 5 — 預設值與計數曝露（AC: #5, #7 BE 半）**
  - [x] `AnalysisSnapshot.DefaultBudgetUSD`＋service 參數＋inline stamp 註解補齊
  - [x] episode Count twin＋窄介面＋`PreviewMissing` 雙數字＋additive key＋Swagger／註解

- [x] **Task 6 — FE 消費與回歸（AC: #6, #7 FE 半, #8）**
  - [x] prefill 改讀 snapshot（fallback 保留）；toast 讀 `totalItemsIncludingEpisodes`
  - [x] 契約清點（sub-4-1 AC #7 additive ack、9R-16 preview additive 先例引用）＋全回歸

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

Claude Fable 5 (claude-fable-5)

### Debug Log References

- Full API suite: `go test ./...` green (all packages, -count=1).
- Full web suite: `pnpm nx test web` — 233 files / 2619 tests green.
- `pnpm run lint:all` — 0 errors, 119 pre-existing warnings (retro-11-AI1b batch; 0 on files this story touched); prettier clean.
- Gemini pricing verified 2026-08-12 against https://ai.google.dev/gemini-api/docs/pricing (source cited in `budget.go`).

### Completion Notes List

- **🔗 AC Drift: FOUND — sanctioned.** sub-4-3 documented the F17 toast count as "凍結的 preview 端點（movies-only）… 已記 backlog"（sub-4-3 story, 已知限制 section）. This story's AC #7 changes that observable behavior: the toast now reads the additive `total_items_including_episodes` (backlog-consent-toast-count-episodes was PROMOTED into this AC at authoring). Grep sweep: `total_items|preview|toast|RecordASR|hosted rate` across `_bmad-output/implementation-artifacts/*.md` AND `tests/e2e/**`+`tests/visual/**` (retro-m25-AI1 scope) — all other hits are REUSE. E2E stubs (`batch-subtitle.spec.ts:213` `{total_items: 2}`) omit the new key, and the FE `?? totalItems` fallback keeps behavior byte-identical under them — no E2E edits required, verified by reading every hit.
- **📎 Contract Stamps: FOUND (0 bumps produced; 2 acks recorded)** — confirmed against `[@contract-v1]` (Story sub-4-1 AC #7): additive `default_budget_usd` on the candidates state envelope, existing keys unchanged, no bump; inline stamp comment added to `generation_candidates_handler.go` (the `generation_batch_handler.go` precedent). · confirmed against `[@contract-v2]` (Story 9R-16 AC #3): additive `total_items_including_episodes` on the preview response. **Authoring correction:** the story's 契約姿態 called this surface "未 stamp"; it IS stamped `[@contract-v2]` (9R-16 AC #3). Pure-widening rule applied (retro-19-P3 precedent — "widening-not-narrowing needs no Rule 20 bump", 19-5 AC #1): existing `total_items` key semantics FROZEN, additive key only → no bump, ack recorded here. No bumps produced → no downstream stale-mark obligation.
- **🎭 A11y Pre-Flight: PASS** (2 components checked — GenerationConsentView, ScanProgress; 0 jsx-a11y warnings on touched files, 0 introduced by this story). Changes are data-source-only: no new interactive surface, no modal/image/live-region changes.
- **🎨 UX Verification: PASS** — zero visual delta by construction: F15's budget input renders the same $5.00 under the factory default (the design-stated value; the prefill only diverges when an operator changes `AI_RUN_BUDGET_USD`), and F17's toast layout/copy are untouched (only the number's data source changed, per the authoring ruling 文案不改). No `.pen` change, no gallery-fixture change, no baseline regen.
- **AC #1** — `RecordASRWithRate` added; `RecordASR` is now a thin delegate at the hosted rate (signature & existing tests untouched). `whisper.go` meters at `EstimatedASRPerMinuteUSD(c.isSelfHosted())` where `isSelfHosted()` reads the ACTUAL endpoint (`c.baseURL != WhisperAPIURL`), not config. The `budget.go:63-67` "Known asymmetry" 掛帳註解 replaced with the same-source statement. Tests: self-hosted $0 (unit + through-the-client with a real parseable WAV), hosted delegate parity, 同源斷言 both directions.
- **AC #2** — `gemini.go` Parse now runs `governed()` → `retryTransient` → per-attempt timeout (the claude.go D8 nesting; malformed-200 permanent per classifyErr rationale), parses `usageMetadata{promptTokenCount,candidatesTokenCount}` and `RecordLLM`s into a ctx Budget. `WithGeminiGovernor` option + `FactoryConfig.Governor` wired at the factory's Gemini branch. Pricing rows added with source+date: `gemini-2.0-flash` $0.10/$0.40 (final published rate), `gemini-2.5-flash` $0.30/$2.50, `gemini-2.5-flash-lite` $0.10/$0.40 — the default model can no longer fall into the Haiku fallback. Tests: metering into ctx budget at the Gemini row, no-budget no-op, budget pre-call short-circuit (0 wire hits), 503-then-success retry, malformed-200 NOT retried, governor wiring, pricing-row guard.
- **AC #3** — `Pipeline.WithRunBudgetUSD` option (WithModelID precedent) + `ProcessItem` attaches a per-item Budget ONLY when the ctx carries none (resolveBudget semantics mirrored); `main.go` wires `cfg.AIRunBudgetUSD`. 紅線釘住: `assert.Same` proves a ctx-carried (consent-batch) Budget is never replaced. Translate-leg ceiling hit = item FAIL (`ErrBudgetExceeded` survives the wrap chain, run row failed, media reverted to retryable) — the ASR leg keeps its existing pause classification untouched.
- **AC #4** — parse paths ruled unmetered-by-design: package-header ruling in `ai_service.go`, operator-facing note in `docs/deployment.md` ("What AI_RUN_BUDGET_USD covers"), `backlog-parse-path-ai-metering` already filed at authoring (雙向 link verified). NOTE: `docs/deployment.md` has no zh-TW twin (pre-existing state; sub-4-2 edited it the same way) — Rule 17 not newly violated by this story.
- **AC #5** — `AnalysisSnapshot.DefaultBudgetUSD` (`json:"default_budget_usd"`) + `defaultBudgetUSD` constructor param (selfHostedASR precedent) + `main.go` passes `cfg.AIRunBudgetUSD`. Present on EVERY snapshot state (idle included — the prefill must not wait for an analysis). Wire-shape test pins the snake_case key.
- **AC #6** — FE `CandidateAnalysisSnapshot.defaultBudgetUsd?: number`; `GenerationConsentView.bootstrap` prefills BEFORE the phase branches (ready/analyzing/kick 三出口都吃到, proven by the SSE-ready-survival test), guarded by the existing `isCancelled` token; non-positive/absent → `DEFAULT_BUDGET_TEXT` fallback kept (0=unlimited must not prefill 0.00). WYSIWYG untouched — test proves the prefilled value is exactly what confirm sends.
- **AC #7** — `EpisodeRepository.CountMissingZhHantSubtitle` twin over the SAME shared predicate (count-agrees-with-enumeration asserted; writeback-shrink asserted; real-sqlite migrated schema per the episode_generation_test pattern) + `EpisodeRepositoryInterface` sync + `generationEpisodeFinder` widened (nil → movies-only degrade, episode-count error surfaces loudly) + `PreviewMissing` returns both numbers + handler emits the additive key with the semantics-frozen comment. FE: `GenerationBatchPreviewResult.totalItemsIncludingEpisodes?` and the toast reads `?? totalItems`. 文案不改 per authoring ruling. Test fakes updated: `fakeEpisodeFinder` (services), `mockGenerationProcessor` (handlers), `mockPQEpisodeRepo` + `stubEpisodeRepo` (interface conformance stubs).
- **AC #8** — 26 new/updated tests: budget_test ×3, whisper_test ×4, gemini_test ×7, process_item_test ×3, episode_generation_test ×2, generation_batch_test ×3 (preview trio), generation_candidates_test ×2, handler preview test widened, GenerationConsentView.spec ×5, ScanProgress.spec ×2. Full regression gates green (see Debug Log).
- **Pre-existing fix: NONE needed** — no pre-existing failures encountered; the `gofmt -l` hit on `gemini.go` is a pre-existing one-space struct-tag misalignment in an untouched line (left alone to avoid noise; repo-wide gofmt drift is not CI-gated).

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **① expand-scope-in-place** — FR12／worker-pool 管線 LLM 翻譯未計帳、不受上限（authoring 盤點發現，先前零掛帳）→ 吸收為 **AC #3**（本 story 的核心任務即為此類缺口）。
    - **③ backlog-with-carry-forward-link** — `backlog-parse-path-ai-metering`：檔名解析／fansub／keyword 路徑（兩個 provider 皆然）無 Budget ctx，計帳為 no-op。本 story 明文裁定不計帳並記錄（AC #4）；未來選項＝process 級 observability 計數器（非 capping）。非阻塞。
  - **YES** — filed at implementation time (2026-08-12)：
    - **③ backlog-with-carry-forward-link** — `bugfix-gemini-default-model-retired`：AC #2 查證官方定價時發現 `DefaultGeminiModel="gemini-2.0-flash"` 已於 **2026-06-01 被 Google 關閉**（官方定價頁明示 shut down）。`AI_PROVIDER` 預設即 `gemini` 且無 `GEMINI_MODEL` env 覆寫 → 預設部署的檔名解析自 6/1 起必 404（9R-1 Claude 模型退役同類事故）。本 story 只保住計帳誠實（保留 2.0-flash 最終費率列＋補現役 2.5 系列列）；模型 bump＋env 覆寫需獨立裁定，已立案（雙向）。非阻塞（Claude provider 部署不受影響）。

### File List

**Backend**
- apps/api/internal/ai/budget.go — RecordASRWithRate + RecordASR delegate; Gemini pricing rows (source+date cited); asymmetry 掛帳註解 cleared
- apps/api/internal/ai/budget_test.go — rate-aware ASR metering tests (self-hosted $0 / hosted parity / nil-safe)
- apps/api/internal/ai/whisper.go — isSelfHosted() (reads actual endpoint) + RecordASRWithRate call site
- apps/api/internal/ai/whisper_test.go — self-hosted zero-spend through the client; endpoint-based detection; 同源斷言
- apps/api/internal/ai/gemini.go — governed()+retryTransient+per-attempt timeout; usageMetadata parse; RecordLLM; WithGeminiGovernor
- apps/api/internal/ai/gemini_test.go — 7 parity tests (metering / short-circuit / retry / permanent-garbage / governor / pricing row)
- apps/api/internal/ai/factory.go — FactoryConfig.Governor + wiring on BOTH provider branches (CR H2)
- apps/api/internal/ai/factory_test.go — Governor propagation test (CR H2)
- apps/api/internal/subtitle/pipeline.go — runBudgetUSD field + WithRunBudgetUSD option
- apps/api/internal/subtitle/process_item.go — per-item Budget attach (ctx-budget reuse red line)
- apps/api/internal/subtitle/process_item_test.go — ctxBudgetSpy + 3 AC #3 tests (ceiling wired / assert.Same red line / translate-leg fail)
- apps/api/internal/services/ai_service.go — package-header unmetered-by-design ruling (AC #4); NewAIService gains the governor param (CR H2)
- apps/api/internal/services/generation_candidates.go — AnalysisSnapshot.DefaultBudgetUSD + constructor param
- apps/api/internal/services/generation_candidates_test.go — call sites + default-budget snapshot/wire-shape tests
- apps/api/internal/services/generation_batch.go — generationEpisodeFinder widened; PreviewMissing dual counts
- apps/api/internal/services/generation_batch_test.go — fakeEpisodeFinder count support + preview trio tests
- apps/api/internal/repository/episode_repository.go — CountMissingZhHantSubtitle twin
- apps/api/internal/repository/episode_generation_test.go — count-twin real-sqlite tests (agree-with-enumeration + writeback shrink)
- apps/api/internal/repository/interfaces.go — EpisodeRepositoryInterface + CountMissingZhHantSubtitle (Rule 15 sync)
- apps/api/internal/handlers/generation_batch_handler.go — PreviewMissing interface + additive total_items_including_episodes (semantics-frozen comments, Swagger synced)
- apps/api/internal/handlers/generation_batch_handler_test.go — mock + dual-count preview assertion
- apps/api/internal/handlers/generation_candidates_handler.go — sub-4-1 AC #7 [@contract-v1] inline stamp comment (deferred-stamp fulfillment)
- apps/api/internal/services/parse_queue_service_test.go — interface-conformance stub (CountMissingZhHantSubtitle)
- apps/api/internal/services/series_season_test.go — interface-conformance stub (CountMissingZhHantSubtitle)
- apps/api/cmd/api/main.go — WithRunBudgetUSD wiring + candidate-service defaultBudget wiring

**Frontend**
- apps/web/src/services/subtitleService.ts — CandidateAnalysisSnapshot.defaultBudgetUsd? + GenerationBatchPreviewResult.totalItemsIncludingEpisodes?
- apps/web/src/components/subtitle/consent/GenerationConsentView.tsx — prefill from snapshot before phase branches; fallback kept; header doc updated
- apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx — 5 prefill tests (snapshot / fallback / non-positive / kick-exit survival / WYSIWYG)
- apps/web/src/components/scanner/ScanProgress.tsx — toast count reads totalItemsIncludingEpisodes ?? totalItems
- apps/web/src/components/scanner/ScanProgress.spec.tsx — 2 toast-count tests (episodes-inclusive + old-server fallback)

**Docs / tracking**
- docs/deployment.md — "What AI_RUN_BUDGET_USD covers" note (AC #4)
- _bmad-output/implementation-artifacts/sprint-status.yaml — sub-5-1 in-progress→review; bugfix-gemini-default-model-retired filed
- _bmad-output/implementation-artifacts/sub-4-3-cost-consent-frontend.md — (AC drift reference — see Completion Notes; file NOT modified)

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-12 | Task 1 (AC #1): RecordASRWithRate + whisper endpoint-based self-hosted detection — self-hosted ASR retrospective spend now $0, rate single-sourced from EstimatedASRPerMinuteUSD. |
| 2026-08-12 | Task 2 (AC #2): Gemini brought to Claude parity — governed+retryTransient+usageMetadata metering+Governor option+factory wiring; pricing rows added (verified 2026-08-12, ai.google.dev). Discovery: gemini-2.0-flash shut down 2026-06-01 → bugfix-gemini-default-model-retired filed (lane ③). |
| 2026-08-12 | Task 3 (AC #3): per-item Budget on Pipeline.ProcessItem (WithRunBudgetUSD + main.go wiring); consent-batch shared ceiling never overridden (assert.Same red-line test); translate-leg ceiling hit fails the item. |
| 2026-08-12 | Task 4 (AC #4): parse paths ruled unmetered-by-design — ai_service.go package header + deployment.md note; backlog-parse-path-ai-metering carry-forward verified. |
| 2026-08-12 | Task 5 (AC #5, #7 BE): default_budget_usd rides the candidates envelope (additive on sub-4-1 AC #7 [@contract-v1], no bump, ack + inline stamp comment); episode Count twin + PreviewMissing dual counts + additive total_items_including_episodes (widening-no-bump on 9R-16 AC #3 [@contract-v2], total_items frozen). |
| 2026-08-12 | Task 6 (AC #6, #7 FE, #8): F15 prefill reads the snapshot default (fallback kept); F17 toast reads the episodes-inclusive count (old-server fallback); full regression green (api all packages, web 233/2619, lint 0 errors, prettier clean). |
| 2026-08-12 | Senior Developer Review (Opus adversarial, 換模型慣例) — 2H/4M/2L, all adjudicated in-session: H2 governor threading fixed (main.go→NewAIService→factory both branches + propagation test); M1 single-source self-hosted predicate (ai.IsSelfHostedASRBaseURL + agreement test); M2 episode-count failure degrades to movies-only (frozen-key availability preserved); M3 429-amplification pinned at retryMaxAttempts (claude-parity ruling documented); M4 zero-usage Gemini success warns loudly (+log-capture test); L1 upper-bound comments corrected; L2 exact pricing pins + governor serialization test. H1 (translate-leg ceiling discards partial work on explicit retry) adjudicated as documented limitation — no auto-loop exists post-sub-4-1 — tracked as backlog-translate-budget-partial-progress (lane ③). Status review → done. |

## Senior Developer Review (AI)

**Reviewer model:** Claude Opus 5 subagent（換模型 adversarial CR 慣例 — implementation by Fable 5）· **Date:** 2026-08-12 · **Outcome:** Changes Requested → all findings adjudicated and resolved in-session → **Approve (done)**

**Mandatory checks:** 🔒 Rule 7 Wire Format: N/A (0 new error-code constants in 24 changed Go files) · 🔒 Rule 20 Contract Bump: PASS (0 bumps; both widening acks verified against upstream stamps — reviewer confirmed the story's 契約姿態 self-correction on 9R-16 AC #3 being [@contract-v2]) · 🔒 Rule 25 Mega-line: N/A (project-context.md untouched) · Git vs File List: 0 discrepancies.

### Findings & resolutions（2H / 4M / 2L）

- [x] **[H1] Translate-leg budget ceiling discards partial work → repeat spend on retry.** Adjudicated: the reviewer's "scan re-enqueues → unbounded loop" premise is outdated — since sub-4-1, scanning never enqueues generation (repo-guarded by cost_consent_test), so every retry is an explicit user action under a visible ceiling. The residual waste (an item whose translation costs > ceiling re-spends from cue 1 per explicit retry, because the segment cache writes only after full-track success) is REAL and is now: (1) documented at the budget-attach site in process_item.go, (2) tracked as `backlog-translate-budget-partial-progress`（Rule 24 lane ③, 雙向）— the fix touches the stamped TranslateTrack [@contract-v1] surface and deserves its own story. Fail semantics itself was an explicit authoring ruling (AC #3) and stands.
- [x] **[H2] `FactoryConfig.Governor` never populated — wiring inert in production.** CONFIRMED and fixed: `aiGovernor` creation moved before the parse-path AI service in main.go; `NewAIService(cfg, db, governor)` threads it; factory now passes the Governor on BOTH branches (Claude too — parse-path Claude was equally ungoverned). New `TestNewProvider_PropagatesTheGovernor` pins propagation for both providers.
- [x] **[M1] Two independent self-hosted detectors can disagree**（config-non-empty vs endpoint-differs; trap value = official URL set explicitly → $0 quote billed at hosted rate）. Fixed: `ai.IsSelfHostedASRBaseURL` is now the ONE predicate; main.go's candidate-service flag uses it; `TestSelfHostedJudgment_SingleSource` proves estimate-side and metering-side agreement for all three value classes including the trap.
- [x] **[M2] Episode-count failure 500ed the whole preview** — the frozen `total_items` key's availability began depending on the episodes table. Fixed: episode leg degrades to movies-only with a Warn log (= pre-sub-5-1 toast behavior, undercount direction-safe); movie-half failure still 500s. Tests updated (degrade + movie-fail-still-fails).
- [x] **[M3] Gemini lost its overall deadline + 429 now retried ×3.** Adjudicated as deliberate claude-parity（同級化 IS the AC — Claude's parse path has identical 429/timeout retry and no outer deadline）; the misleading NFR-I12 comment corrected to state the real bound (3×timeout + backoff), and the 1→3 request amplification pinned by `TestGeminiProvider_Parse_QuotaExceededRetriesExactlyMaxAttempts`.
- [x] **[M4] usageMetadata-less 200 metered $0 silently** — disables the ceiling while real money is spent. Fixed: loud `slog.Warn` on zero-usage success (LLMCalls still recorded); `TestGeminiProvider_Parse_MissingUsageMetadataWarnsAndMetersZero` captures the log and pins the $0+1-call behavior.
- [x] **[L1] "Matches the consent list" rationale false**（the list filters skipped/unprobeable items）. Fixed: handler + ScanProgress + PreviewMissing comments reworded to "upper bound on the consent list".
- [x] **[L2] Wiring-not-behavior assertions.** Fixed: pricing test pins the EXACT published rates; governor test replaced with a 1-slot serialization test over two concurrent Parses (proves acquire AND release).

### Reviewer verifications that held (not re-checked by orchestrator)

snakeToCamel generic mapping for both new FE keys · consent shared-Budget genuinely never overridden (assert.Same) · no double-metering (1 ASR call site, 2 LLM call sites) · Rule 19 clean · Rule 21 headers present · Rule 23 N/A · Gemini pricing values match published rates · gofmt hit on gemini.go pre-existing · api build/vet/tests green.

