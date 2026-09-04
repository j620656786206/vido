# Story 6.2: LLM 呼叫 timeout 隨模型與輸出長度放寬 —— 15 秒寫死不再殺整支 run（後端）

Status: done

## Story

As a BYOK NAS owner,
I want a slow-but-healthy Sonnet response to be waited for instead of being cut off at 15 seconds three times in a row,
so that a 986-cue run cannot die at cue 935 after I already paid for 935 cues.

## Context — 這個 story 為什麼存在

eval-1 產品問題 4：Sonnet 5 一批 10 句偶爾超過 15 秒，`retryTransient` 三次全逾時後 `TranslateTrack` 回錯，整支 run 失敗，燒了 $0.468。第二輪 Zootopia B 又發生一次（$0.295）。

現況鏈：`ai.DefaultTimeoutSeconds = 15`（`gemini.go:28`，Claude 也用）→ `http.Client{Timeout: p.timeout}`（`claude.go:128`）→ `isTimeoutErr` 判為 transient → `retryTransient` 3 次（`retry.go`）→ 失敗回 `ErrAITimeout` → `pipeline.go:~730` `return nil, ErrSubtitleTranslateFailed`。外層 `TranslationTimeout = 60s`（`translation_service.go:26`）永遠不會先到。

## Acceptance Criteria

1. **timeout 依模型分級。** `ai` 套件新增 `RequestTimeoutFor(model string, maxTokens int) time.Duration`：基底依模型家族（haiku 30s / sonnet 60s / opus 90s / gemini flash 30s，表格與 `defaultLLMPricing` 同檔維護），再依 `maxTokens` 線性加成（每 1k output tokens +10s），上限 180s。未知模型走 sonnet 級。`ClaudeProvider` 改為**每次呼叫**用 `context.WithTimeout(ctx, RequestTimeoutFor(p.model, maxTokens))`，`http.Client.Timeout` 改為 0（由 ctx 管）——注意 `claude.go:118-127` 的註解：不得同時堆兩個 deadline。

2. **外層 timeout 不得比內層短。** `TranslationTimeout`（60s）改為由 `RequestTimeoutFor` 推得 ×（retry 次數 + 1），或直接移除該層改依賴 ai 層（擇一，authoring 建議後者：兩層 deadline 是 D8 已指出的 bug 溫床）。

3. **逾時不再殺 run。** `retryTransient` 三次逾時後，pipeline 對該 chunk 的處理改為：把該 chunk 的 cue **標記為 stubborn（保留英文原文）**並繼續下一 chunk，而不是 `return nil, err`——與 quality gate 的 `maxQualityRetries` 落敗語意一致（`pipeline.go:~760` `noteStubbornCue`）。run 結尾若 stubborn 比例 > 20% 才判 failed；否則 completed 並在 `subtitle_runs` 記 `stubborn_count`（欄位已存在則沿用，否則 additive migration）。**只有 timeout／5xx 類 transient 錯誤**走此路徑；401/404/budget 觸頂仍立即失敗（既有分類不動）。

4. **可觀測。** 每次逾時 `slog.Warn` 帶 `model`、`timeout_seconds`、`attempt`、`cue_range`；SSE `subtitle_progress` message 帶「第 N 批逾時，重試 M/3」。

5. **測試。** (a) `RequestTimeoutFor` 表格與加成；(b) Claude provider：慢 fake server（sleep > 舊 15s、< 新 timeout）成功；(c) pipeline：translator fake 對某 chunk 連續回 `ErrAITimeout` → 該 chunk stubborn、後續 chunk 照跑、run completed；stubborn 超比例 → failed；(d) 401 仍立即失敗；(e) 全回歸。

## Tasks / Subtasks

- [x] **Task 1 — 分級 timeout（AC: #1, #2）**
  - [x] `ai/timeout.go`：`RequestTimeoutFor` + 表格；`claude.go` 改 per-call ctx deadline、client Timeout 歸零；`gemini.go` 同步
  - [x] `translation_service.go` 移除／推導 `TranslationTimeout`
- [x] **Task 2 — chunk 級降級（AC: #3, #4）**
  - [x] `pipeline.go` chunk loop：transient 失敗 → stubborn 標記＋繼續；run 尾端比例判定
  - [x] 逾時 slog／SSE message
- [x] **Task 3 — 測試（AC: #5）**

（全後端。）

## Dev Notes

### 既有可重用零件

| 需求 | 現成零件 |
| --- | --- |
| transient 分類 | `claude.go:225-260` `classifyErr`；`isTimeoutErr` `:193` |
| 重試 | `ai/retry.go` `retryTransient`（**不加第二層重試**，D8） |
| stubborn 語意 | `pipeline.go` `noteStubbornCue`／`stubborn` 計數（quality gate 落敗路徑） |
| 模型表 | `ai/budget.go` `defaultLLMPricing`（timeout 表放同檔旁，更新時一起改） |

### 為什麼不是「把 15 改 60」

一個常數改大只是把懸崖搬遠。Sonnet 一批 10 句 + 詞彙表 trailer 可到 2k output tokens；Opus 更慢。**依模型＋輸出長度**才是長解（architecture-prefer-long-solutions）。

### Rule 20

`PipelineStage` 詞彙不擴；`subtitle_runs` 若加欄位是 additive（Rule 15 SELECT/scan 同步）。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- eval-1「產品問題 4」、第二輪「Zootopia B 又是 15 秒 timeout」；後續 Backlog P0-2
- `apps/api/internal/ai/{claude.go,gemini.go,retry.go}`、`apps/api/internal/subtitle/pipeline.go:670-770`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Debug Log References

- `go test ./...`（apps/api）全綠；`pnpm nx run api:lint`（go vet + staticcheck）綠。

### Completion Notes List

- **Task 1（AC #1/#2）**：`ai/timeout.go` `RequestTimeoutFor(model, maxTokens)`；家族表 `llmTimeoutBase` 依 AC 放 `budget.go` 與 `defaultLLMPricing` 同檔（substring 比對、大小寫不敏感，未知 → sonnet 級 60s）；每 1k output +10s 線性、上限 180s。`claude.go`／`gemini.go`：`timeout` 欄位歸零＝每次呼叫推導；deadline **只有一層** — `send`／`Parse`／`Ping` 在 `retryTransient` 的每一次 attempt 內 `context.WithTimeout`，預設 `http.Client` 無 `Timeout`，SDK `WithRequestTimeout` 分支整段移除（自訂 client 也由 ctx 管，既有 `TestClaudeProvider_TimeoutEnforcedWithCustomHTTPClient` 仍守）。`WithClaudeTimeout`／`WithGeminiTimeout` 保留為「釘死一個數字」的覆寫（測試與需要固定值的部署）。`DefaultTimeoutSeconds` 常數刪除。`services.TranslationTimeout` 與三處 `WithTimeout` 包裝整個移除（authoring 建議的「後者」），`TranslationMaxTokens` 註解說明它現在同時是 deadline 的輸入。
- **Task 2（AC #3/#4）**：`retryTransient` 在三次 transient 失敗後把最後錯誤與新 sentinel `ai.ErrAIRetriesExhausted` 一起包（sentinel 放後面，開頭錯誤碼不變；permanent 與 ctx 取消**不**包）— 這是 pipeline 分辨「供應商整段 retry 視窗都在抖」與「請求本身錯」的唯一依據，不靠比對 status 字串。`TranslateTrack`：`isTransientExhausted(ctx, err)`（排除 `ErrBudgetExceeded` 與 ctx 已取消）→ pending cues `noteStubbornCue`（排除 segment cache）、計入 `transient`、繼續下一 chunk；**第二道天花板** `transientCeilingDenominator=5`（20%，分母＝delivered track）套在「品質 stubborn + transport stubborn」總和，且在**確定超過的那一個 chunk 就失敗**（不把剩餘 chunk 全送進同一場故障）；FR16 5% 品質天花板不動。品質重試（attempt>0）途中逾時：清空 `verdict` 避免同一 cue 被計兩次（測試抓到的 bug）。`TranslateResult` additive `TransientCues`。`ai.WithRetryObserver`／`RetryNotice`：pipeline 每 chunk 掛觀察者，slog 帶 chunk／attempt／cue_range；SSE `translating` 三種訊息（`chunk N/M timed out, retrying k/3`、`failed transiently, retrying`、`kept English after transport retries`）→ zh-TW「第 N/M 段逾時，重試 k/3」等；provider 端 `Claude API timeout` log 補 `model`。
- **stubborn_count**：`subtitle_runs` 無此欄 → additive migration 034（nullable INTEGER，NULL＝034 前未計）；Rule 15 `subtitleRunColumns` 18 → 19 同步；`ProcessItem` completed 時 stamp `len(scope.stubbornIndexes)`（無 LLM 路線＝0，不是 NULL）；完成 log 加 `stubborn_cues`。
- **Task 3（AC #5）**：(a) `TestRequestTimeoutFor_*` 表格／加成／上限／未知；(b) `TestClaudeProvider_DerivedTimeoutIsThePerAttemptDeadline` — 用測試縮小家族表（與 `TestMain` 縮 backoff 同一手法）證明「慢於推導值逾時、快於推導值成功、max_tokens 放寬」，並斷言三次 attempt 都到伺服器（每次新 deadline）；不用 16 秒 sleep；Gemini 同級；(c) `pipeline_transient_test.go` — chunk stubborn 繼續／超 20% 立即失敗且不再送 chunk／品質＋transport 合計／品質重試途中逾時只降級 pending／SSE 敘事序列；(d) 401／404／budget／malformed／單次 timeout 未 exhausted／ctx 取消 → 立即失敗且只送 1 chunk；(e) 全回歸綠。
- 🔗 AC Drift: NONE (checked: 'TranslationTimeout|DefaultTimeoutSeconds|StubbornCues|stubborn' across _bmad-output/implementation-artifacts/*.md — sub-1-5a 的 stubborn 5% 品質政策不變、本 story 在其上加第二道；sub-5-1 AC #2「同級化」保持：Claude／Gemini 兩邊都改為推導 deadline；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (`TranslateContext`／`TranslateResult` `[@contract-v1]` — `TransientCues` additive、簽名不變、stubborn 政策「加一道 20% 上限」為放寬非收窄，不 bump；`ChunkTranslator` unstamped port 未改)
- 🎭 A11y Pre-Flight: N/A（純後端）
- 🔌 Route Sync: N/A（無新路由）
- 🎨 UX Verification: N/A（純後端；SSE 文案走既有 `翻譯中…` 氣泡）

### Discovery Triage

- ① expand-scope-in-place — `subtitle_runs.stubborn_count` 欄位不存在 → AC #3 明寫「否則 additive migration」，migration 034 + Rule 15 同步。
- ① expand-scope-in-place — 品質重試途中逾時會讓同一 cue 同時計入品質與 transport stubborn（測試發現）→ 清 `verdict`。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `ai/timeout.go`、`budget.go` 家族表、`claude.go`／`gemini.go` per-attempt ctx deadline、`TranslationTimeout` 移除。 |
| 2026-09-04 | Task 2 — `ErrAIRetriesExhausted`、`WithRetryObserver`、`TranslateTrack` transient 降級 + 20% 天花板、SSE 三訊息、migration 034 `stubborn_count`。 |
| 2026-09-04 | Task 3 — `timeout_test.go`、`pipeline_transient_test.go`、SSE 文案案例、034 測試、repo round-trip。 |
| 2026-09-04 | CR fixes（Opus 5 adversarial，13 findings，全部在同一分支修）— H1 wall-clock 熔斷 `maxConsecutiveTransientChunks=2`；H2 **partial delivery 可重跑**：migration 034 加 `transient_count`、completed 時 `TransientCount>0` → `restoreMediaStatus`（不標 `found`）+ SSE `complete` 帶「N 句暫留英文」、`preflightSkip` 讀 version-matched run 的 `transient_count` 放行重跑（只重送英文 cue，其餘 cache hit）；M3 兩個母體改 index set（品質重試途中逾時的 cue 留在 FR16 5% 池，union 給 20%）；M4 `ErrAIQuotaExceeded` 不降級；M5 `contextWindow` 跳過未翻譯 cue、往回找已翻譯的 5 句、全無則 nil；M6 `Ping` 走 `ProbeRequestTimeout`（10s）；M7 project-context Rule 7 清單補 `AI_RETRIES_EXHAUSTED`（連同既有漏列的四個）；M8 `ProcessItem` 層級測試（stamp 計數／partial 還原狀態／重跑只送英文／full delivery 仍 early-exit）、ASR 路線留 NULL 並把註解改誠實；L9 narrator 改每 attempt 建立、帶 `cue_range`；L10 provider.go／claude_test 註解、repo 測試改名、PRD NFR-I12 改為推導規則；L11 `maxTimeoutTokens` 先夾再乘；L12 `emitScopedProgress` 無 scope 不廣播；L13 由 `transient_count` 解決；SDK 10 分鐘 `WithRequestTimeout` 事實寫進 `MaxRequestTimeout`／claude.go 註解；新增 pricing 表每個 key 都命中家族表的測試。 |

### File List

- `apps/api/internal/ai/timeout.go`、`timeout_test.go`（new）
- `apps/api/internal/ai/budget.go`、`claude.go`、`gemini.go`、`retry.go`、`types.go`（modified）+ `claude_test.go`、`gemini_test.go`
- `apps/api/internal/services/translation_service.go`（modified）
- `apps/api/internal/subtitle/pipeline.go`、`process_item.go`、`progress_sse.go`（modified）+ `pipeline_transient_test.go`（new）、`progress_sse_test.go`
- `apps/api/internal/models/subtitle_run.go`、`apps/api/internal/repository/subtitle_run_repository.go`（modified）+ `subtitle_run_repository_test.go`
- `apps/api/internal/database/migrations/034_add_subtitle_run_stubborn_count.go`、`_test.go`（new；`stubborn_count` + `transient_count`）
- `apps/api/internal/ai/provider.go`（comments）
- `project-context.md`（mega-line + Rule 7 清單）、`_bmad-output/planning-artifacts/prd/non-functional-requirements.md`（NFR-I12）、`_bmad-output/implementation-artifacts/sub-6-2-llm-timeout-by-model.md`、`sprint-status.yaml`

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → all 13 items resolved in-session on the same branch → **Approve**

Mandatory checks: Rule 7 PASS（code-list update only，`AI_RETRIES_EXHAUSTED` 補進第 293 行清單，prefix 17 不變）· Rule 15 PASS（`subtitleRunColumns` 20 欄，全部 SELECT 走同一常數）· Rule 20 N/A（`TranslateResult` additive）· D8 PASS（一層 retry、一層 deadline；SDK 自帶的 10 分鐘 `WithRequestTimeout` 因 180s 上限永不生效，已寫入註解）。

### Action Items

- [x] [H1] 掛住的 provider 要燒 1.7 小時才放棄 — `maxConsecutiveTransientChunks=2` 熔斷；測試（兩個連死 → 只送 2 chunk；中間有成功 → 重置）。
- [x] [H2] 降級後的英文 cue 被 P5 永久凍結 — **產品裁定（dev 代決，可逆）**：partial delivery 不標 `found`、`transient_count` 入庫、pre-flight 放行重跑；測試（完整雙輪流程）。
- [x] [M3] `verdict` 清空讓 FR16 5% 可被繞過 — index set 雙母體；測試（1/10 品質重試逾時仍 fail 5%）。
- [x] [M4] 429 也降級 — `ErrAIQuotaExceeded` 排除；測試。
- [x] [M5] stubborn chunk 後整窗英文 context — `contextWindow` 只取已翻譯 cue；測試（窗口為 6-10 不是 16-20）。
- [x] [M6] 金鑰測試最壞 3 分鐘 — `ProbeRequestTimeout` 10s；Claude／Gemini `Ping` 同級；測試。
- [x] [M7] Rule 7 清單未更新 — 補齊。
- [x] [M8] `stubborn_count` stamp 零覆蓋、ASR 路線 NULL 與註解矛盾 — `ProcessItem` 測試 ×3；ASR 留 NULL、model／migration 註解改為「NULL＝未計（034 前或 ASR 路線）」。
- [x] [L9] AC #4 四欄位散落、narrator 無 `cue_range` — 每 attempt 建 narrator、帶 `cue_range`；`model`／`timeout_seconds` 在 provider 行（分工記錄在註解）。
- [x] [L10] 過期註解／命名／PRD — 全部更新。
- [x] [L11] `RequestTimeoutFor` 溢位 — 先夾再乘；測試。
- [x] [L12] `emitProgress` 無 scope guard — `emitScopedProgress`。
- [x] [L13] `TransientCues` write-only — `transient_count` 欄位承接。
- Checked-OK 之外的兩點記錄：`whisper.go` 仍是 client Timeout + per-attempt ctx 同值雙層（無害，本 story 不動，見 PR body）；`ProviderConfig.TimeoutSeconds` 為死欄位（只改註解）。
