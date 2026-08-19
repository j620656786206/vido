# Story 9R-17: AI 用量可見性 —— 單項轉錄成本 SSE＋統一 /ai/usage 讀端點

Status: ready-for-dev

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

- [ ] Task 1: AC #1 單項 SSE 欄位＋wire-shape 測試
- [ ] Task 2: AC #2 dormant props 接線＋批次回歸釘
- [ ] Task 3: AC #3 ActiveSnapshot()＋TranscriptionService holder＋handler
- [ ] Task 4: AC #4 測試矩陣
