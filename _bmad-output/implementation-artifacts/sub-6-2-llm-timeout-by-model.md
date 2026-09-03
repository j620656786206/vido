# Story 6.2: LLM 呼叫 timeout 隨模型與輸出長度放寬 —— 15 秒寫死不再殺整支 run（後端）

Status: ready-for-dev

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

- [ ] **Task 1 — 分級 timeout（AC: #1, #2）**
  - [ ] `ai/timeout.go`：`RequestTimeoutFor` + 表格；`claude.go` 改 per-call ctx deadline、client Timeout 歸零；`gemini.go` 同步
  - [ ] `translation_service.go` 移除／推導 `TranslationTimeout`
- [ ] **Task 2 — chunk 級降級（AC: #3, #4）**
  - [ ] `pipeline.go` chunk loop：transient 失敗 → stubborn 標記＋繼續；run 尾端比例判定
  - [ ] 逾時 slog／SSE message
- [ ] **Task 3 — 測試（AC: #5）**

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

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
