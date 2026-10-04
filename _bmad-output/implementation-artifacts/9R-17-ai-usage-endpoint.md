# Story 9R-17: AI 用量可見性 —— 單項轉錄成本 SSE＋統一 /ai/usage 讀端點

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c · **Priority:** P2 S
**Source:** sprint-status `9R-17-ai-usage-endpoint`（Rule-24 ③，SM Bob 2026-07-05）。
**⚠️ 2026-08-19 authoring 重大盤點**：原條目的「NO HTTP/SSE surface」驗證是 2026-07-05 的，**已被 9R-16（07-06）超車** —— 批次面早就出貨了。本檔依實況重列範圍；初稿經對抗驗證後重寫。

---

## Story

As a NAS owner running a single-item transcription (not a batch),
I want the per-item progress stepper to show live spend against budget like the batch dialog already does,
so that cost visibility covers every paid run — and one endpoint can answer "what is AI spending right now" regardless of run type.

---

## Context —— 已出貨 vs 真缺口（逐行查證，2026-08-19）

**已出貨（本 story 不得重做）：**
- **批次 SSE 已帶成本**：`generation_batch.go:541-562` 廣播的 progress payload 含 `spent_usd`/`budget_usd`（9R-16；wire-shape 測試釘住 `generation_batch_test.go:285,295,724`，其註解明言「cost line rides the batch SSE (no 9R-17 needed)」）。
- **批次 status 端點已回成本**：`GET /api/v1/subtitles/generation-batch/status`（`generation_batch_handler.go:178` → GetProgress，`generation_batch.go:73-74,188`）。
- **批次 FE 已渲染**：`useGenerationBatchProgress.ts:40-41,83-84` 解析、`GenerationBatchDialogV2.tsx:453-458` 本次用量成本列、F12 budget_ceiling 終局 `:272,347` 顯示最終 spent/budget。

**真缺口：**
1. **單項轉錄面**：`transcription_*` SSE 事件**不帶任何成本欄位**（Budget 只在 `transcription_service.go:404-410` snapshot 進 log）——`GenerationProgressV2.tsx:54-56` 的 dormant props（`costUsedText`/`costLimitText`，`:212` render-gated）吃的是這條 SSE，**永遠等不到資料**。
2. **手動 run 無讀取口**：批次的 Budget 由 `GenerationBatchProcessor`（main.go:794 singleton）的 `activeBudget` 持有可讀；單項轉錄的 Budget 只活在 `runPipeline` 的 ctx（`transcription_service.go:326-332,402`）——**沒有 holder**，任何端點都讀不到。

## Acceptance Criteria

### AC #1 — 單項轉錄 SSE 成本欄位（additive）
- `transcription_*` progress/complete 事件 payload 增 `spent_usd`/`budget_usd` additive 欄位，值取自 ctx Budget snapshot（`:404-410` 已有 snapshot 取值先例）；無 Budget（未設限）→ 欄位缺席（不是 0）。
- 既有 key 一個不動；wire-shape 測試釘 snake_case。Rule 20 additive 慣例 ack。

### AC #2 — `GenerationProgressV2` dormant props 甦醒
- `costUsedText`/`costLimitText` 接 AC #1 的欄位；欄位缺席 → 隱藏（既有 `:212` gate 沿用，不顯示 0 —— capability-honor）。
- 批次 dialog／workspace 的成本顯示**零改動**（已出貨，回歸釘）。

### AC #3 — `GET /api/v1/ai/usage` 統一讀端點
- **存在理由**（與既有批次 status route 的區隔）：單一入口涵蓋**批次與手動兩種 run**；批次 status route 只知批次。
- data：`active_run`（`kind: batch|manual`、`spent_usd`、`budget_usd`、`remaining_usd`）或 `null`（idle 是合法常態，**不是 404**）。**不含 started_at**（兩側皆無此欄位，不為端點發明時間戳）。
- 批次側：`GenerationBatchProcessor` 增窄方法 `ActiveSnapshot()`（mutex 下讀 activeBudget，Rule 11 窄介面）。
- 手動側：`TranscriptionService` 增 active-run Budget holder（run 開始設、defer 清；併發語意跟隨既有 single-slot 轉錄語意）。兩者同時活躍時批次優先報（單一 `active_run`；併發細分屬未來範圍）。
- `{success,data}` 信封；Swagger **註解即可** —— apps/api 無 swag-gen（9R-16 完工註記裁定在案，consolidation P1.2），不執行 `swag init`。

### AC #4 — 測試
- SSE wire-shape：新欄位 key＋缺席語意（無 Budget 不出欄位）。
- FE：props 有值渲染／缺席隱藏兩態；批次成本列回歸釘（byte-unchanged）。
- handler：batch-active／manual-active／idle 三態。

### AC #5 — 範圍紅線
- 不做歷史用量存表／月報彙總。
- 不做 per-provider 拆帳（Governor 單池照池報）。
- 不動預算設定寫入面。
- 同步修正 sprint-status 條目的過時「NO HTTP/SSE surface」描述（本 story 落地時一併改）。

## Tasks / Subtasks

- [x] Task 1: AC #1 單項 SSE 欄位＋wire-shape 測試
- [x] Task 2: AC #2 dormant props 接線＋批次回歸釘
- [x] Task 3: AC #3 ActiveSnapshot()＋TranscriptionService holder＋handler
- [x] Task 4: AC #4 測試矩陣

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-04）

### Completion Notes List

- **2026-10-04 dev 前重新核對**：sub-7-6a 已出貨 `subtitle_run_receipt`（run 結束時一次、帶 spent_usd）——它是「結束時的收據」，不是「進行中的即時花費」，本單仍需要。`GenerationProgressV2` 的 `costUsedText`/`costLimitText` 仍是休眠狀態；`/ai/usage` 不存在。
- **AC #1**：`resolveBudget` 回報「這個 run 自己建立的 budget」；`runPipeline` 把它記在 in-flight 的 `soloTranscriptionJob.budget`。`broadcastEvent` 在 **只限 stepper 的五種事件**（extracting／progress／translation_progress／complete／failed）上加 `spent_usd`（一律）與 `budget_usd`（有上限才帶）。批次項目花的是批次共用的 budget，**不帶**（批次 dialog 已顯示批次花費；項目的 stepper 若顯示整批數字會誤導）。
  - Contract ack：既有 key 一個不動，additive（Rule 20）。`subtitle_run_receipt`（sub-7-6a [@contract-v1]）同樣經過 `broadcastEvent`，**刻意排除**——它有自己的「本 run 起算」數字，不可被覆寫（CR 抓到）。
- **AC #2**：`useGenerationProgress` 與 `useGenerationJobsFeed` 帶 `spentUsd`/`budgetUsd`（事件沒帶就沿用上一次；START/RESET 清空）；`ManageSubtitleDialogV2` 與 `GenerationWorkspaceV2` 單項列在兩個數字都有時才傳 `costUsedText`/`costLimitText`（`lib/currency` 的 `usd()`）。批次面零改動。
- **AC #3**：`GET /api/v1/ai/usage` → `{active_run: {kind, spent_usd, budget_usd, remaining_usd} | null}`；批次優先；`remaining_usd` 下限 0（無上限時 budget 0、remaining 0）。窄介面 `BatchUsageReader`（`GenerationBatchProcessor.ActiveSnapshot`）／`ManualUsageReader`（`TranscriptionService.ActiveManualUsage`，多個 solo run 同時進行時加總）。Swagger 只補註解。
- **測試**：Go——`resolveBudget` 三值、`costFields`（own／batch／無上限／不在進行中）、真 hub 的 wire-shape（五種事件帶 key、收據不被改）、`ActiveManualUsage`、`ActiveSnapshot`、handler 三態（idle 200 null／批次優先／手動＋下限）；`-race` 綠。前端——兩個 hook 的取值與沿用、dialog「有花費顯示 $0.42 / $2.50、沒有就不顯示」。
- **Adversarial CR（2026-10-04，獨立 agent）**：無確認 bug。✅ 修：成本 key 原本也會加到 `subtitle_run_receipt`（數值目前相同但語意不同）→ 限定 stepper 事件；補上 AC #4 要的 wire-shape 與 dialog 顯示測試。驗證無誤：無死鎖（`s.mu` 都是葉鎖、沒有在持鎖時 broadcast）；所有入口都經 `runPipeline`、批次項目不會帶 key；終局事件在 `inProgress` 移除前送出；`/ai/usage` 路由無衝突。📝 記錄：預設 `AI_RUN_BUDGET_USD=5`，所以 solo run 從提取音訊起就會顯示「$0.00 / 上限 $5.00」——0 是真實花費，不是「未知」；`POST /subtitles/pipeline/run`（單項 D6 管線）的 per-item budget 不在 `/ai/usage` 範圍內（dialog 不走那條，本單範圍外）。

### File List

- apps/api/internal/services/transcription_service.go、transcription_generation_test.go
- apps/api/internal/services/generation_batch.go、generation_batch_test.go
- apps/api/internal/handlers/ai_usage_handler.go（新）、ai_usage_handler_test.go（新）
- apps/api/cmd/api/main.go
- apps/web/src/hooks/useGenerationProgress.ts、useGenerationProgress.spec.ts
- apps/web/src/hooks/useGenerationJobsFeed.ts、useGenerationJobsFeed.spec.ts
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx、ManageSubtitleDialogV2.spec.tsx
- apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx
