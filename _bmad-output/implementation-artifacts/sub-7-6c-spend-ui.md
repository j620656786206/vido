# Story sub-7-6c: 活動頁「本月 AI 花費」、批次收據行、首頁 readout 改讀月報 — 前端

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-01，由 sub-7-6 拆出；依賴 sub-7-6b 端點；**開工前要 Sally 兩張稿**（走 inline agent）。行號為 main `34b59615`。 -->

## Story

身為看活動頁的人，
我要一眼看到這個月字幕花了多少、哪個模型、略過和快取幫我省了多少；
批次跑完要看到「本次 $0.53 · claude-sonnet-5 · 844 句 · cache 命中 12%」；首頁的花費小字改讀月報。

## 背景（查到的事）

- `components/activity/ActivityHub.tsx`：四個區塊各自 fail-soft；⚠️ `isEmpty(data)`（l.68-80）會把**全部**區塊換成 `<ActivityEmpty/>`，新區塊要納入判斷。
- F8 = `GenerationBatchDialogV2.tsx`（`i9Nun1`／`H717g`），`CostRow` l.194-236 畫兩次（桌機、手機 footer）；`GenerationProgressV2` 的 `costUsedText/costLimitText` 是 9R-17 的位子，本張不碰。
- `HomeReadoutBand.tsx:89-110 attentionText`：spend 只在 `spendSource==='live_batch'` 顯示，`last_run` 刻意隱藏；「absent ≠ $0」規則在檔頭。
- 金額工具：`lib/currency.ts`（decimal，禁 `+`/`-`）、`utils/formatUsdShort.ts`；`≈` 單一語意＝片長是假設的（`currency.ts:53-61`、`ButtonCost.tsx:11`）。
- SSE：`subtitle_run_receipt`（7-6a）＋ `generation_batch_progress.model_id`。
- 設計：`flow-k-activity-v2/` 沒有花費區塊；`flow-f-subtitle-v2/f8-d-v2` 沒有收據行 → **需要 Sally 兩張稿**：K5「本月 AI 花費」區塊（桌機＋手機）、F8 完成態的收據行。

## Acceptance Criteria

1. **活動頁新區塊「本月 AI 花費」**（在「活動記錄」之上）：翻譯／語音辨識兩列（金額＋次數）、依模型小表、「略過 N 集，估算省下 $X」（`skipped_saved_runtime_assumed` 才加 `≈`）、「快取省下 $Y」（`cache_measured_runs=0` → 「—」）。端點失敗 → 區塊 `ActivitySectionError`，其他區塊不受影響；`isEmpty` 納入。
2. **F8 批次完成收據行**：「本次 $0.53 · claude-sonnet-5 · 844 句 · cache 命中 12%」——金額用批次 `spent_usd`，其餘用 `by_batch`（重新開啟）或累加 `subtitle_run_receipt`（進行中）；沒量到 cache → 省略該段。
3. **首頁 readout**：無 live batch 時 spend 三聯改讀月報 `translated_usd+asr_usd`；有 live batch 仍顯示 live；absent ≠ $0 不變。
4. **誠實規則**：沒記錄顯示「—」；估算用文字「估算」；`≈` 只在片長假設。
5. **Rule 23**：月份文案「10 月」由 `now` prop 注入；gallery fixtures `month-start`／`month-end`。
6. **測試**：三處元件 spec ＋ gallery fixtures ＋ e2e 一條（活動頁區塊）；`-linux` 由 CI bootstrap。

## Tasks / Subtasks

- [ ] Task 0 — Sally 出 K5／F8 收據行兩段提示詞 → Alexyu 跑 inline agent → 截圖
- [ ] Task 1 — service／hook（`useSubtitleSpend`）＋ 型別
- [ ] Task 2 — 活動頁區塊（AC #1, #4, #5）
- [ ] Task 3 — F8 收據行（AC #2）
- [ ] Task 4 — 首頁 readout（AC #3）
- [ ] Task 5 — 測試與 fixtures（AC #6）

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                    |
| ---------- | ----------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-6 拆出）。 |
