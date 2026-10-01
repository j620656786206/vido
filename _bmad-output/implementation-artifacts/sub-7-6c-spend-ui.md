# Story sub-7-6c: 活動頁「本月 AI 花費」、批次收據行、首頁 readout 改讀月報 — 前端

Status: done

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

## Task 0 提示詞（Sally 2026-10-01，節點錨定；貼給 Pencil Inline AI Agent 原樣執行，一段跑完 ⌘S 再跑下一段）

> **A · K5 本月 AI 花費（桌機＋手機）**
>
> 1. 用 `FindEmptySpace` 在 `M6ra92`（K4-D）右側找空位，`Copy` `kMeWS`（K1-D-v2）到 document，命名「K5-D-v2 · 活動中心（本月 AI 花費）」，`placeholder:true`。
> 2. 在複製稿的 `main` 裡、`sec-下載` 與 `sec-活動記錄` 之間，`Copy` `sec-下載`（原 `U2ymzn`）當骨架，改名「sec-本月AI花費」：`secTitle` 改「本月 AI 花費」，secHead 右側加一個 `Type/Body/Size`、`$text-muted` 的文字「10 月」。把 `row-下載中` 那張卡改成 `layout:vertical`、`gap:$Space/sm-plus`，內容換成三塊（都在同一張 `$bg-secondary` 卡內，內距沿用）：
>    - 第一列 horizontal、gap `$Space/xl`：兩組「標籤＋數字」——「翻譯」`$text-secondary` Body ＋「12 次 · $3.48」`$text-primary` BodyLg 600；「語音辨識」＋「2 次 · $1.90」同樣式。
>    - 第二列 horizontal、gap `$Space/xl`、`$text-secondary` Body：「略過 8 集，估算省下 $2.14」、「快取省下 $0.31」。
>    - 第三塊「依模型」：小標「依模型」`$text-muted` Label；兩列 `Type/Body/Size`：「claude-sonnet-5」左、「9 次 · $4.20」右對齊 `$text-primary`；「claude-haiku-4-5」左、「5 次 · $1.18」右。列之間 `$Space/xs`，用 `justifyContent:space_between`。
>    - 拿掉 `progressTrack`（這張卡沒有進度）。
> 3. 手機：用 `FindEmptySpace` 在剛做好的 K5-D 右側找空位，`Copy` `QIwY1`（K1-M-v2）命名「K5-M-v2 · 活動中心（本月 AI 花費・手機）」，在 `scroll-content`（原 `KwHSK`）的 `sec-下載`（原 `J43oA`）與 `sec-活動記錄`（原 `aZeU5`）之間插入同一個區塊，第一、二列改成 vertical 疊放（gap `$Space/xs`），其餘同桌機；字級維持 14（手機不縮內文）。
> 4. 兩張都 `placeholder:false`，確認 `problems` 為空、沒有文字溢出卡片。
>
> **B · F8c 批次完成收據（桌機＋手機）**
>
> 1. 用 `FindEmptySpace` 在 `i9Nun1`（F8-D-v2）右側找空位，`Copy` `i9Nun1` 命名「F8c-D-v2 · 批次生成（完成收據）」，`placeholder:true`。
> 2. 對話框內：`overall-nums` 的「2 / 5」（原 `dP5Lo`）改「5 / 5」；`item-list` 裡的 `item-active`（原 `K3X0m`）`Replace` 成 `item-奧本海默`（原 `JSEQV`）的複製品並把片名改「怪奇物語 S01E03」、狀態文字與其他完成列一致。
> 3. `cost-row`（原 `k89qkw`）的 `cost-line` 四個文字改成：`cost-prefix`「本次 」（`$text-secondary`）、`cost-used`「$0.53」（`$text-primary`、600，不變）、`cost-mid`「 · claude-sonnet-5 · 844 句 · 」（`$text-secondary`）、`cost-cap`「cache 命中 12%」（`$text-secondary`）。`sse-chip`（原 `Blkcw`）改成 `$success-tint` 底、`$success-text` 字「已完成」。
> 4. footer 的按鈕文字「全部取消」（原 `hb4bs/L9cIf`）改「關閉」，按鈕改用 `Component/Button/Secondary`（`YDPhc`）。
> 5. 手機：對 `H717g`（F8-M-v2）做同樣四步，命名「F8c-M-v2 · 批次生成（完成收據・手機）」，放在 F8c-D 右側；收據行若一行放不下，`cost-line` 改 vertical 兩行：第一行「本次 $0.53 · claude-sonnet-5」、第二行「844 句 · cache 命中 12%」。
> 6. `placeholder:false`，確認 `problems` 為空。

**設計裁定理由**：收據行沿用 F8 既有的 cost-line 樣式（金額 600、其餘 secondary），只是把「上限」換成模型／句數／快取——完成態不再需要上限，需要的是「這次到底買到什麼」。活動頁的花費區塊採卡片而非列（`ActivityRow-v2` 是「一件事一列」的語彙，花費是一張總覽），但內距、底色、標題字級與其他區塊一致，讓它讀起來仍是活動頁的一段。`—`／「估算」／`≈` 三種標示的規則寫在 AC #4，稿上示範的是「都有數字」的正常態。

## Tasks / Subtasks

- [x] Task 0 — Sally 出 K5／F8 收據行兩段提示詞 → Alexyu 跑 inline agent → 截圖（2026-10-01 完成：`xgYKA` K5-D／`ptNai` K5-M／`gWFcx` F8c-D／`LAeqW` F8c-M；Sally MCP review：文字逐一相符、桌機兩張 problems 0、K5-M 回到 390×844 且「本月 AI 花費」在摺線下（有 note `xQFZE`，同 K1-M 慣例）、F8c-D 唯一 clip 是原 F8 就有的背景活動記錄；⚖️ Alexyu 2026-10-01 裁定位置維持「下載」之下、「活動記錄」之上，閒置時自動上浮（AC #1 已含）。截圖 k5-d／k5-m／f8c-d-v2／f8c-m-v2 已進 `SCREENS`）
- [x] Task 1 — service／hook（`useSubtitleSpend`）＋ 型別
- [x] Task 2 — 活動頁區塊（AC #1, #4, #5）
- [x] Task 3 — F8 收據行（AC #2）
- [x] Task 4 — 首頁 readout（AC #3）
- [x] Task 5 — 測試與 fixtures（AC #6）

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（2026-10-01）

### Completion Notes List

- **活動頁區塊**：`SpendSection.tsx`（`SpendCard` 純 props ＋ `SpendSectionView` 外殼）。位置在「下載」之下、「活動記錄」之上（⚖️ Alexyu 2026-10-01）；沒事在跑時前三區不渲染，區塊自然成為第一個，不需要另寫排序規則。自己一條 query（`useSubtitleSpend`），端點壞掉只有這一段 `ActivitySectionError`；`isEmpty` 判斷改成「四區空 ＋ 月份沒內容 ＋ 花費端點沒壞」才顯示空狀態；骨架等兩條 query 都落定（避免空狀態閃一下再長出卡片）。
- **誠實規則落地**：0 次 → 「—」（不是 $0.00）；`cache_measured_runs=0` → 「快取估算省下 —」；所有省下都帶「估算」二字；`≈` 只在 `skipped_saved_runtime_assumed`；`unpriced_runs`／`unrouted_*` 進附註行，不進兩條付費線。
- **F8c 收據行**：`buildReceiptView(progress, byBatch)` ——ledger `by_batch`（同 batch_id 才算）優先、SSE `subtitle_run_receipt` 累加（`useGenerationBatchProgress.runTotals`，只算這個 batch 的 completed）當即時備援、`generation_batch_progress.model_id` 補模型名。沒量到的段落直接省略（不畫 0%、不畫 0 句）。完成且無失敗才給綠色「已完成」chip（🔴 #10 既有規則）。手機版 footer 同一行拆兩行。
- **首頁 readout**：`spendReadout` 三階：live batch 的 spent/ceiling → 本月 ledger 總額（翻譯＋語音＋舊紀錄，`sumUsd` 精確加）→ last_run 只在有失敗時。朗讀文字「本月已花費 $5.4」。
- **月份文案不讀瀏覽器時鐘**：「10 月」從回應的 `from`（伺服器時區文字）切字取得——伺服器定義的月份才是資料的月份，瀏覽器時區可能差一天；Rule 23 因此不需要 `now` prop，fixtures 釘 `from` 即可（與 AC #5 寫法不同，見 Discovery Triage）。
- `usePageVisibility` 抽成共用 hook（原本 useActivity／useHomeSummary 各一份），三條輪詢 hook 共用。
- 終端事件時多 invalidate `subtitleSpendKeys.month()`，活動頁與首頁卡片立即更新，不等一分鐘。

### Discovery Triage

- **AC #5 偏離**：月份文案改由 `from` 推導，不注入 `now`。理由：端點是伺服器當月的彙總，`from` 就是那個月；用 `now` 會讓台北 10/1 00:30 的瀏覽器在 UTC 伺服器上顯示錯月。Fixtures `month-start`／`month-end` 改為釘不同 `from` 與不同資料形狀（月初：一條線有數、語音「—」、快取「—」、≈；月底：K5 全數字）。
- **AC #1 文案**：快取那行寫「快取估算省下 $0.31」而非稿上的「快取省下 $0.31」——AC #4 要求估算一律用字，`cache_saved_usd_estimate` 是反推的均價 × 命中，是估算。Sally 的設計說明也寫「三種標示規則以 AC #4 為準，稿示範的是正常態」。
- **沒做**：首頁花費總額包含 `unrouted_usd`（舊紀錄）——AC #3 字面是 `translated_usd+asr_usd`，但舊紀錄是真的花掉的錢，首頁總額少算會跟活動頁附註對不上。若要嚴格照字面，改 `monthSpendUsd` 一行即可。

### File List

- `apps/web/src/services/subtitleSpendService.ts`（新）＋ `.spec.ts`
- `apps/web/src/hooks/useSubtitleSpend.ts`（新）、`apps/web/src/hooks/usePageVisibility.ts`（新；`useActivity.ts`／`useHomeSummary.ts` 改用）
- `apps/web/src/components/activity/SpendSection.tsx`（新）＋ `.spec.tsx`、`ActivitySectionShell.tsx`（新，自 ActivityHub 抽出＋`trailing`）
- `apps/web/src/components/activity/ActivityHub.tsx` ＋ `.spec.tsx`
- `apps/web/src/hooks/useGenerationBatchProgress.ts` ＋ `.spec.ts`（`modelId`、`runTotals`、`subtitle_run_receipt` 監聽）
- `apps/web/src/services/subtitleService.ts`（`GenerationBatchProgress.modelId?`）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` ＋ `.spec.tsx`（`buildReceiptView`、`receipt` prop、CostRow 收據態、容器讀 `by_batch`）
- `apps/web/src/components/homepage/HomeReadoutBand.tsx` ＋ `.spec.tsx`（`monthSpendUsd`、`spendReadout`）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（`activity-spend-card/month-end`／`month-start`／`month-end-mobile`、`generation-batch-dialog-v2/complete-receipt`／`-mobile`）
- `tests/e2e/activity-spend.spec.ts`（新）

## Change Log

| Date       | Change                                    |
| ---------- | ----------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-6 拆出）。 |
| 2026-10-01 | Task 0 設計稿完成（Alexyu inline agent；Sally review 通過）；截圖與 SCREENS 入 repo。 |
| 2026-10-01 | Task 1–5 實作完成（Fable 5.1）；232 條相關 vitest 綠；status → review。 |
| 2026-10-01 | PR #637 合併（CI 全綠；-linux 基準由 #638 補）；status → done。 |
