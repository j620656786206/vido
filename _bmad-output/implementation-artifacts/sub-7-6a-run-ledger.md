# Story sub-7-6a: 每次字幕生成都記成帳（路線、快取命中、批次、收據事件）— 後端

Status: review

<!-- SM Bob create-story 2026-10-01，由 sub-7-6 拆出（BE 3 task 的門檻被「單獨語音辨識不寫紀錄」與「批次收據」撐破 → 拆成 7-6a 記帳／7-6b 月報端點／7-6c 前端）。
     ⚖️ Alexyu 2026-10-01 裁定 AA：單獨語音辨識也要記；`≈` 守 Sally 的單一語意（只在片長是猜的時出現）。
     行號為 main `34b59615`。原單 9 處與程式碼對不上（詳見 Dev Notes）。 -->

## Story

身為想知道「這個月字幕花了多少、省了多少」的人，
每一次字幕生成都要留下一筆看得懂的帳：走哪條路、花多少、用哪個模型、幾句、快取幫了幾句、屬於哪個批次——
不管是批次跑的、掃描後自動跑的，還是我在詳情頁按「生成字幕」跑的。

## 背景（查到的事）

- `subtitle_runs` 只有 `spent_usd`／`budget_usd`（migration 032，nullable）；**沒有**路線欄（只有 `process_item.go:321` 的 log）、**沒有**快取命中欄（`process_item.go:398-400` 只 log）、沒有批次欄。單子寫的 `cost_usd` 不存在。
- 只有 `subtitle/process_item.go` 會寫 run 列。**單獨按「生成字幕」（`TranscriptionService.StartTranscription`）與 legacy 模式的批次（`RouteCGenerationRunner`）完全不寫**——月報的語音辨識金額會漏掉它們。
- 管線的 ASR 後備（`cmd/api/asr_adapter.go` → `RunTranscription`）同一件工作已有管線的 run 列；引擎若也寫，會重複計。
- 快取命中：管線在 `translateWithCache` 的 `splitCachedCues` 知道 `hits`；ASR 腿在 `TranslationService.splitCachedBlocks` 只 log。
- SSE：`subtitle_progress` 的 hook 簽名 `(ref, stage, message)` 帶不了金額；`generation_batch_progress` 有 `spent_usd` 沒有 `model_id`；`transcription_complete` 沒有金額（9R-17 規劃補 `spent_usd`）。三個 payload 都是已戳記的契約。
- 批次的 ctx 已帶共享 Budget 與 model id（`generation_batch.go:476-483`）——批次 id 走同一條路最省。

## Acceptance Criteria

1. **三個新欄（migration 041，全部 nullable）。** `route TEXT`（`deliver_direct`／`convert_then_deliver`／`translate`／`skip`／`no_text_source`／`asr`）、`cache_hit_cues INTEGER`（只在 translate 腿量；deliver 腿 NULL＝沒量，不是 0）、`batch_id TEXT`（＋index）。model／repository／scan 全同步（23 欄），round-trip 與 nullable 測試更新；跑兩次冪等。
2. **管線寫帳。** 路線在 routing 後寫入（`string(decision.Kind)`），ASR 後備實際跑了改成 `asr`；快取命中由 `processScope` 記錄、translate 腿完成時寫入（冷快取＝0，非 NULL）；`batch_id` 從 ctx 讀（`services.WithGenerationBatchID`）。
3. **收據事件 `subtitle_run_receipt` [@contract-v1]。** 每個 run 在**終態寫入**（completed／failed／skipped／deferred／budget-paused）後發一次；payload = `models.SubtitleRun.ReceiptPayload()`：`run_id, media_id, media_type, status, model_id, cue_count` ＋ 選填 `route, batch_id, cache_hit_cues, spent_usd, budget_usd, completed_at`（沒記錄就**不出現**，絕不填 0）。管線經 `WithRunReceipt`／`NewSSEReceiptHook`；所有終態都走 `recordTerminal`，沒有一條 lane 能悄悄結束。
4. **語音辨識引擎自己記帳。** `TranscriptionService.SetRunLedger`：單獨點擊與 legacy 批次的 run 在開始時開一列 `running`（route `asr`、model、batch_id），結束時寫 completed／failed、cue 數、版本三元組、輸出路徑、**自己的** Budget delta（legacy 批次共用一個 Budget，與管線 scope 同一做法）、有 segment store 時才記 cache 命中；並發同一個收據事件。管線的 ASR 後備傳 `WithRunRecordedByCaller()` → 不開列，不重複計。
5. **批次事件多帶 `model_id`**（`generation_batch_progress` 與 status／last 快照，additive）；`TranslationOutcome` 多 `CachedBlocks`（additive）。
6. **測試。** migration 3、repository round-trip／nullable 更新、models 收據 payload 1、subtitle 4（路線＋快取＋批次＋一張收據；冷快取 0 非 NULL；skip 有收據且 cache NULL；failed 也有收據）、services 4（單獨 run 開列→completed 帳；caller-owned 不開列；失敗 → failed；legacy 批次帶 batch_id）、批次 SSE key set 加 `model_id`。api 全量、vet、staticcheck、lint:all 綠。

## Tasks / Subtasks

- [x] Task 1 — migration 041 ＋ model ＋ repository（AC #1）
- [x] Task 2 — 管線：route／cache／batch_id／`recordTerminal`／收據 hook（AC #2, #3）
- [x] Task 3 — 語音辨識引擎 ledger（AC #4）＋ 批次 `model_id`（AC #5）
- [x] Task 4 — 測試、全量（AC #6）

## Dev Notes

- **原單對不上的 9 處**：`cost_usd`→`spent_usd`；runs 無路線欄；快取命中沒存；沒存 token 用量（每句均價只能用 `spent_usd / (cue_count − cache_hit_cues)` 反推，7-6b）；估價公式是「每分鐘片長」不是每句（7-6b）；單獨語音辨識不寫列（本張補）；沒有 run 詳情端點（7-6b 視需要）；「F8」是批次對話框不是單項 stepper；`≈` 單一語意（Sally 2026-09-05）與原單衝突→AA 裁定守規矩。
- **為什麼是新事件不是塞欄位**：`subtitle_progress`／`transcription_complete`／`generation_batch_progress` 都是戳記契約；收據對兩條腿是同一件事，一個事件一個 listener。`transcription_complete` 的 `spent_usd` 留給 9R-17。
- **批次收據怎麼算**：不擴 `GenerationRunner` port；批次 id 進 ctx → 每列有 `batch_id` → 7-6b 從 ledger 彙總「本次批次」。
- 收據事件在 `broadcastEvent` 上 nil-safe（測試無 hub）。

### Time-dependent visual coverage

- N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 見 Change Log；驗證數字記在 PR。

### Discovery Triage

- 見 PR「沒做的事」。

### File List

- apps/api/internal/database/migrations/041_add_subtitle_run_ledger_columns.go、_test.go
- apps/api/internal/models/subtitle_run.go、subtitle_run_test.go
- apps/api/internal/repository/subtitle_run_repository.go、subtitle_run_repository_test.go
- apps/api/internal/sse/hub.go
- apps/api/internal/subtitle/pipeline.go、process_item.go、progress_sse.go、process_item_ledger_test.go
- apps/api/internal/services/generation_batch.go、generation_batch_context.go、generation_batch_test.go、generation_batch_items_test.go、translation_service.go、transcription_service.go、transcription_ledger.go、transcription_ledger_test.go
- apps/api/cmd/api/main.go、asr_adapter.go
- _bmad-output/implementation-artifacts/sub-7-6a-run-ledger.md、sub-7-6b-spend-summary-api.md、sub-7-6c-spend-ui.md、sub-7-6-cost-receipt.md、sprint-status.yaml

## Change Log

| Date       | Change                                                                 |
| ---------- | ---------------------------------------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-6 拆出）；dev-story（Amelia）→ review。 |
