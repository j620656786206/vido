# Disc：沒有字幕來源也沒設語音辨識 B（前端）——生成畫面寫灰色「已略過」＋第二行「沒有可用的字幕來源，也沒有設定語音辨識」

Status: ready-for-dev（⛔ 等 `disc-2026-10-generation-queue-failure-reason`（PR #851，2026-10-20 為對話框兩行重開）合併才開工；`-a` 已合併 #855）

**Source:** 傘狀單 `disc-2026-10-no-text-source-skip`（2026-10-19 party mode：founder-pm／Winston／Sally／John／Bob／Mary＋#848 的另一個 session；⚖️ Alexyu：由這邊直接做）。

⚖️ **SM 裁定（Bob，2026-10-19）：**
- 拆單後的前端那張（後端 `-a` 送代碼 `no_text_source`）。前端 5 項。
- **依賴：**
  - ⛔ blocked-by `disc-2026-10-generation-queue-failure-reason`（2026-10-19 正在另一個 dev agent 手上，未合併）。那張會新增 `apps/web/src/components/subtitle/runFailureCopy.ts`（`FAILURE_COPY_BY_REASON` 搬過去）、讓 `queueRowLabel` 收 `failureReason`、把即時活動 ④⑤⑤b 改成兩行（`FeedRowView.detail`）。本單**建立在那三件事上**，下面引用的行號是 `origin/main` 49d96125（那張合併前），開工時以合併後的程式為準。
  - ⛔ **（2026-10-20 追加）對話框佇列列的兩行結構也由那張做**：Sally 2026-10-20 新裁定（見檔尾）要產生字幕對話框的失敗／已略過列一律兩行，版面（`QueueRow` 第一行片名＋狀態字、第二行 `gen-batch-row-detail-${mediaId}` 原因）由 `disc-2026-10-generation-queue-failure-reason` 的「2026-10-20 追加」做（PR #851 重開）。**本單必須等 #851 合併後才開工**；本單在對話框只負責讓 `queueRowBadge`／`queueRowLabel` 對略過項目回「已略過」＋原因，不碰兩行結構。
  - ✅ `-a` 已合併 #855（2026-10-20）：契約是 `[@contract-v2]`，本單讀 `no_text_source`（另有舊路徑 `asr_unconfigured`）。原本的說明：本單單獨先上也不會壞：批次 `reason: skipped` 會變灰色（沒代碼時第二行寫「沒有可用的字幕來源」）；單次要等 `-a` 送代碼才會變灰。
- **契約 ack：** confirmed against `[@contract-v2]` (Story disc-2026-10-live-activity-failure-reasons-a AC #1)（由 `-a` 升版：多 `no_text_source`；批次 `skipped` 項目帶 `no_text_source`／`asr_unconfigured`／`""`）。
- **灰色用什麼：** 設計稿**沒有**「已略過」這種即時活動列或佇列徽章 → ⚖️ **待 Sally 追認**，預設沿用既有中性色（見 AC #3、#4）。不擋開工。

## 設計稿 / Design ref

- **沒有對應的稿。** 最接近的既有狀態（本單照它們的色調）：
  - 即時活動：F11-SPEC-LOG `tT3JX` 的 ⑧「批次已取消」（圖示 `--text-muted`、階段字 `--text-secondary`）；兩行排法照 ④ `stf9c`（`evt-line`＋`evt-reason-row`，`disc-2026-10-live-activity-row-layout-design` 裁定 1、2、4、5）。
  - 生成佇列：`Component/GenQueueRow-v2`（`aw4Qr`）的「已取消／已暫停」徽章（`bg-[var(--bg-tertiary)]`＋`--text-muted`）；說明行 `lUZol` 仍是 `$text-secondary`（Sally 裁定 6）。
  - 片庫徽章的前例：`apps/web/src/utils/libraryStatus.ts:166-171`「無字幕源」「已略過」都是 `TINT.neutral`（`bg-[var(--bg-tertiary)] text-[var(--text-muted)]`，`:48`）。
- 產生字幕對話框（**2026-10-20 改，Sally 新裁定**）：F8-D-v2 `i9Nun1`、F8-M-v2 `H717g`、新規格稿 **F8-SPEC-ROW**（截圖 `_bmad-output/screenshots/flow-f-subtitle-v2/f8-spec-row.png`，2026-10-20 繪製中；節點 id 等稿 PR 合併、`pen-node-ids.json` 更新後再寫進 `Design ref:`，否則 `check-design-refs.py` 會紅）。失敗與已略過列兩行：第一行片名（最多兩行）＋右邊短狀態字，第二行原因。已略過＝灰色「已略過」（`--text-muted`、無圖示）＋第二行原因。桌機 F8-D-v2 同一排法。
- ⚖️ Sally 若要補稿（F11-SPEC-LOG 加一列「已略過」、GenQueueRow-v2 加一個狀態），另開 design 單；本單照預設先做，稿出來後以稿為準修。

## 查到的事（2026-10-19，`origin/main` 49d96125；另參考工作目錄裡 generation-queue-failure-reason 的進行中改動）

### 即時活動（右欄）

- 單次生成失敗：hook `SINGLE_FAILED`（`apps/web/src/hooks/useGenerationJobsFeed.ts:427-457`）寫一列 `kind: 'failed'`、`reason: null`、`failureReason: payload.failureReason`（`:453`）。`feedRowView`（`apps/web/src/components/subtitle/generationEventCopy.ts:206-221`）：`copy.stopped` → ⑤「已停止」赭色；否則 `failedView`（`:306-315`）紅色「失敗」。→ 今天 skipped 的單次是「失敗 · 片名 · 生成失敗」。
- 批次成員：hook `changed_item`（`useGenerationJobsFeed.ts:505-525`）寫 `reason: changed.reason`、**`failureReason` 只在 `reason === 'error'` 時保留**（`:524`）。`feedRowView` 的批次分支（`:222-229`）：`skipped` → `queueRowLabel(...).text`＝「沒有可用的字幕來源」，一樣走紅色 `failedView`。gen-queue 那張會把 `:225-228` 收成一個 `queueRowLabel({ status: 'failed', reason, failureReason }).text`（它的 AC #4）。
- 既有中性色的列：⑧ 批次已取消（`:271-281`，`circle-pause`、`MUTED`／`SECONDARY`）、階段列「已中斷」（`:169-181`，`circle-dashed`、`MUTED`／`SECONDARY`）。

### 生成佇列（左欄）＋產生字幕對話框

- 文字：`queueRowLabel`（`apps/web/src/components/subtitle/generationQueueRow.ts:69-92`）`failed` → `FAILURE_TEXT[reason]`（`:48-54`，`skipped`＝「沒有可用的字幕來源」）、紅色 `--error-text`、圖示 `alert`。徽章字 `queueRowBadge`（`:100-115`）`failed` →「失敗」。說明行 `queueRowSubStatus`（`:127-142`）`failed` → 標籤文字。
- **舊裁定（要推翻）：** `FAILURE_TEXT` 上方註解（`:42-47`）＝ dsr-6d-b AC #3／dsr-6d-a CR L8：「`skipped` 不准用『已略過』開頭——舊路徑沒設金鑰也走這個原因，『略過』像是系統自己決定不做」。三支測試釘住：`generationQueueRow.spec.ts:51-57`（`startsWith('已略過')` 為 false、紅色、`alert`）、`GenerationWorkspaceV2.spec.tsx:288-290`（`not.toHaveTextContent('已略過')`）、`GenerationBatchDialogV2.spec.tsx:1327-1329`（同）。2026-10-19 party mode 改成要用，理由：第二行把原因講清楚（「也沒有設定語音辨識」），不會被誤會成系統自己決定；舊路徑沒設金鑰另給「語音辨識未設定」（`-a` AC #1）。→ ⚖️ 待 Sally 追認（傘狀單 #2）。
- 工作區列：`QueueRow`（`apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:285-371`）；徽章底色 `BADGE_TINT`（`:247-254`，依 `status`，`failed` → `--error-tint`；`paused`／`cancelled`／`queued` → `--bg-tertiary`）；徽章圖示 `BadgeIcon`（`:262-283`，`failed` → `TriangleAlert`、`cancelled` → 無）；`data-state={item.status}`（`:310`）。
- 對話框列：`QueueRow`（`apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx:345-394`）→ `RowStageLabel`（`:176-192`）只有**一個**標籤（沒有徽章），顏色與圖示來自 `queueRowLabel`；`data-state={item.status}`（`:359`）。
  - ⚠️ 2026-10-20：#851（gen-queue「2026-10-20 追加」）合併後，失敗列改成兩行——狀態字＝`queueRowBadge`（失敗列「失敗」），顏色／圖示仍來自 `queueRowLabel`，第二行 `detail`＝`queueRowLabel(view).text`（只在 `failed` 時有）。本單的略過列吃同一條路：`queueRowBadge` 回「已略過」、`queueRowLabel` 回灰色無圖示＋原因句 → 對話框自動變成「已略過」＋第二行原因。開工時以合併後的程式為準。
- 型別：`GenerationBatchItemReason`（`apps/web/src/services/subtitleService.ts:164-170`）；`failureReason` 註解（`:176-183`）寫「failed＋error 才有」→ 要改。
- **總結句不動：** 「完成 N 部、失敗 M 部」（工作區 `workspace-terminal-label`、即時活動 ⑦）用後端 `failCount`，略過仍算在失敗裡（傘狀單 #3，預設不改）。

### 文字表

- `FAILURE_COPY_BY_REASON`（今天在 `generationEventCopy.ts:65-86`；gen-queue 之後在 `runFailureCopy.ts`），`FailureCopy` 只有 `text`、`stopped`（`:26-34`）；`RunFailureReason`（`:47-58`）十一個。活動頁 `recentEventCopy.ts:89` 也讀這份（只讀 `.text`）。
- 對應的後端句：`solo_runner.go:508`「這部影片沒有可用的字幕來源，也沒有設定語音辨識」（管理字幕視窗結果區顯示的那句）。本單文字去掉「這部影片」：「沒有可用的字幕來源，也沒有設定語音辨識」（19 字）。

### 測試、夾具、基準、E2E

- 寫死代碼清單：`generationEventCopy.spec.ts:368-383`「has exactly the eleven contract codes」；三方一致性 `:315-357`（gen-queue 之後擴成 live log ⇄ activity ⇄ queue，對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一；`stage` 斷言 `stopped ? '已停止' : '失敗'`）；`recentEventCopy.spec.ts:130-142` 逐代碼對文字。
- 寫死「skipped 是紅色失敗／不帶代碼」：`generationQueueRow.spec.ts:51-57`；`generationEventCopy.spec.ts:89-125`（④ 含 `reason: 'skipped'` 那列 `:91`、`:115` 註解「A code only refines `error`」）；`useGenerationJobsFeed.spec.ts:697-717`（「a batch failure that is not `error` carries no code, even if one is sent」）；`GenerationWorkspaceV2.spec.tsx:276-290`；`GenerationBatchDialogV2.spec.tsx:1315-1329`。
- 畫廊夾具：工作區 `generation-workspace-v2/complete-with-failures`（`apps/web/src/routes/test/-gallery.fixtures.tsx:7310` 起；佇列 `wsFxItem(2, 'failed', 'skipped')` `:7331`、即時活動 `wsFxFeed(3, 2, { kind: 'failed', reason: 'skipped', failureReason: null })` `:7338`）；對話框 `generation-batch-dialog-v2/complete-with-failures`（`:6921` 起，`batchFxItem(3, 'failed', 'skipped')` `:6943`）。這兩張基準**會變**。其他生成相關夾具沒有 skipped。gen-queue 那張會先更新 `generation-workspace-v2/complete-with-failures` 並新增 `failures-long-titles`。
- E2E：`tests/e2e/generation-workspace.spec.ts:42-44`（`busy_elsewhere` 列 `data-state="failed"`＋文字）、`:62-64`＋`:76`（stub 送沒有代碼的 `skipped`，斷言總結「完成 1 部、失敗 2 部」）、`batch-subtitle.spec.ts:254-256`（`busy_elsewhere` 文字）。→ 都不受影響（`data-state` 不改、總結不改、`busy_elsewhere` 不改），仍要跑一次。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要生成字幕畫面遇到「片子沒有可用的字幕、我又沒設語音辨識」時，用灰色「已略過」告訴我，第二行寫「沒有可用的字幕來源，也沒有設定語音辨識」，
這樣我一眼分得出「這部沒壞，只是缺東西」跟「真的壞掉了」，也知道要去設語音辨識，而不是看到一句紅色「生成失敗」。

## Acceptance Criteria

0. **開工前確認**（寫進 Completion Notes）：`disc-2026-10-generation-queue-failure-reason`（PR #851）已合併（`runFailureCopy.ts` 存在、`queueRowLabel` 收 `failureReason`、`FeedRowView` 有 `detail`、**對話框 `QueueRow` 已是兩行：失敗列狀態字用 `queueRowBadge`、第二行 `gen-batch-row-detail-${mediaId}`**）；`-a` 已合併 #855（2026-10-20，契約 `[@contract-v2]`）。若 gen-queue 被標了 `⚠️ STALE [@contract-v1→v2]`，確認已處理。

**文字表**

1. **新代碼與「略過」旗標**（`runFailureCopy.ts`）：
   - `RunFailureReason` 加 `'no_text_source'`（十二個）；註解寫明它是唯一給「沒做成但沒壞」的代碼，來源 `disc-2026-10-live-activity-failure-reasons-a` AC #1 `[@contract-v2]`。
   - `FailureCopy` 多一個 `skipped: boolean`（註解：「沒做成，但不是壞掉、也不是錢的問題——畫成灰色『已略過』」）。**每一條都要寫**（`Record` 型別少一個就 typecheck 失敗）：除了 `no_text_source` 全是 `false`。
   - `no_text_source: Object.freeze({ text: '沒有可用的字幕來源，也沒有設定語音辨識', stopped: false, skipped: true })`。
   - （替代寫法可接受：把 `stopped`／`skipped` 收成 `tone: 'failed' | 'stopped' | 'skipped'`；選哪個寫進 Completion Notes。不要兩個都 `true`。）

**生成佇列（左欄）＋對話框**

2. **一個判斷函式：** `generationQueueRow.ts` 新增並 export `queueRowIsSkipped(view: QueueRowView): boolean` ＝ `view.status === 'failed' && (view.reason === 'skipped' || failureCopyForReason(view.failureReason).skipped)`。佇列、對話框、即時活動批次列**都只用它**判斷要不要畫成「已略過」。
3. **佇列的字與顏色：**
   - `queueRowLabel`：`queueRowIsSkipped` 時回 `{ text, className: 'text-[var(--text-muted)]', icon: null }`——`text` 是：帶了認得、且不是 `other` 的代碼就用 `failureCopyForReason(code).text`（`no_text_source` → 那句 19 字；`asr_unconfigured` →「語音辨識未設定」），沒代碼、不認得或 `other` 就用 `FAILURE_TEXT.skipped`（「沒有可用的字幕來源」，略過列不該寫「生成失敗」）。其他 `failed`（`error`、`busy_elsewhere`、`''`）**完全不變**（紅色、`alert`）。
   - `queueRowBadge`：`queueRowIsSkipped` 時回「已略過」，其餘 `failed` 仍「失敗」。
   - `queueRowSubStatus`：`failed` 仍回標籤文字（略過時就是原因句）。
   - `FAILURE_TEXT` 上方註解改寫：舊的「不准用已略過」由 2026-10-19 party mode 推翻（理由見「查到的事」），`skipped` 現在是灰色徽章「已略過」＋第二行原因；舊路徑沒設金鑰後端改送 `asr_unconfigured`。
4. **工作區徽章：** `GenerationWorkspaceV2.tsx` 的 `QueueRow` 在 `queueRowIsSkipped(item)` 時：徽章底色 `bg-[var(--bg-tertiary)]`（同已取消）、徽章文字色 `--text-muted`（來自 `label.className`）、**不畫圖示**（同已取消）。做法建議：`BADGE_TINT`／`BadgeIcon` 收一個「顯示狀態」（`'skipped' | GenerationBatchItemStatus`），不要在 JSX 裡散寫判斷。說明行顏色維持 `--text-secondary`（Sally 裁定 6）。**`data-state` 仍是 `item.status`（`failed`）**——E2E 靠它；要給測試抓「略過」可**新增** `data-skipped="true"`（新增，不改既有）。
5. **對話框（2026-10-20 改寫，Sally 新裁定推翻追認 #3）：** 產生字幕對話框（F8，`GenerationBatchDialogV2.tsx` 的 `QueueRow`）的略過列跟失敗列一樣是**兩行**，兩行結構由 #851 做好，本單只改字與顏色：
   - **第一行：** 片名（最多兩行，`line-clamp-2`，#851 已改）＋右邊短狀態字「**已略過**」（`queueRowBadge`，AC #3），灰色 `--text-muted`、**沒有圖示**（來自 `queueRowLabel`，AC #3），對齊片名第一行。
   - **第二行：** 原因（`gen-batch-row-detail-${mediaId}`，Body 字級、`--text-secondary`），左緣對齊片名、**前面沒有「·」、不截斷**、太長就折行。`no_text_source` →「沒有可用的字幕來源，也沒有設定語音辨識」；`asr_unconfigured` →「語音辨識未設定」；沒代碼／不認得／`other` →「沒有可用的字幕來源」。
   - 桌機 F8-D-v2 同一排法。其他狀態（完成、排隊、進行中、已暫停、已取消）維持一行，本單不動。
   - **不要**在本單改對話框 `QueueRow` 的版面、`RowStageLabel` 的高度／對齊、`detail` 的條件——那是 #851 的事；若 #851 合併後的寫法讓略過列拿不到狀態字或第二行，先回報 SM，不要在本單重做版面。
   - 退回列（沒有後端 items 時，`:514-520`、`:769-795`）不受影響。
   - ~~舊預設：對話框不顯示「已略過」三個字、只放灰色原因句~~（Sally 2026-10-19 追認 #3，2026-10-20 推翻）。

**即時活動（右欄）**

6. **hook 保留略過的代碼：** `useGenerationJobsFeed.ts:524` 改成 `reason === 'error' || reason === 'skipped'` 時保留 `changed.failureReason || null`；`busy_elsewhere` 仍是 `null`。`FeedRow` 的 `failureReason` 註解（`:101-108`）、`subtitleService.ts:176-183` 註解跟著改（skipped 也可能帶代碼）。
7. **列的字與顏色**（`feedRowView`）：
   - 單次（`row.reason === null`）：`copy.skipped` → 略過列；否則照舊（`stopped` → ⑤，其餘 → ④）。
   - 批次：`queueRowIsSkipped({ status: 'failed', reason: row.reason, failureReason: row.failureReason })` → 略過列，原因文字＝同一個 `queueRowLabel(...).text`（佇列與即時活動同一句）；其餘照舊。
   - 略過列（新增 `skippedView(name, text)`，放在 `failedView` 旁）：`glyph: 'circle-dashed'`、`glyphClass: MUTED`、`stage: '已略過'`、`stageClass: SECONDARY`、`parts: [name]`、`detail: text`（兩行排法照 ④，`FeedRowItem` 不用改）、`announce: '已略過：' + name + '，' + text`。
   - 檔頭註解（`generationEventCopy.ts:1-19`）補一段：skipped 是灰色「已略過」，不是失敗；F11-SPEC-LOG 目前沒有這一列（待 Sally）。
8. **不動：** ⑦「批次完成 · 完成 N 部、失敗 M 部」與工作區總結句（仍把略過算進失敗，傘狀單 #3）；活動頁（`recentEventCopy.ts` 不改邏輯，只是型別多一個代碼）；管理字幕對話框結果區。

**測試**

9. **三方一致（會變紅的那種）：** gen-queue 擴好的「live log ⇄ activity record ⇄ queue」一致性測試（對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一）：
   - `stage` 斷言改成 `skipped ? '已略過' : stopped ? '已停止' : '失敗'`；
   - 加一條：對 `no_text_source`，單次列 `detail`、批次 `reason: 'skipped'` 帶代碼那列的 `detail`、`queueRowLabel({ status: 'failed', reason: 'skipped', failureReason: 'no_text_source' }).text`、`recentSubtitleDetail(...)` 的 ` · ` 後半段，四者 `===` `FAILURE_COPY_BY_REASON.no_text_source.text`；三個畫面的「略過」判斷都是 `true`（單次 stage、批次 stage、`queueRowBadge`）。
   - 「has exactly the eleven contract codes」改名「twelve」並加 `no_text_source`；新增「只有 `no_text_source` 的 `skipped` 是 `true`、沒有任何一條 `stopped` 與 `skipped` 同時是 `true`」。
   - `recentEventCopy.spec.ts:130-142` 加 `['no_text_source', '沒有可用的字幕來源，也沒有設定語音辨識']`。
10. **改寫既有測試（改寫，不是刪；新斷言寫明為什麼推翻）：**
    - `generationQueueRow.spec.ts:51-57` →「skipped 是灰色已略過」：`skipped` 沒代碼 → 文字「沒有可用的字幕來源」、`--text-muted`、`icon: null`、`queueRowBadge` 是「已略過」；`skipped`＋`no_text_source` → 19 字那句；`skipped`＋`asr_unconfigured` →「語音辨識未設定」灰色；`skipped`＋`'nope'`／`'constructor'` →「沒有可用的字幕來源」灰色。迴歸：`error`＋`target_not_writable` 仍紅色「失敗」；`busy_elsewhere`＋`no_text_source` 仍紅色「這部正在別處處理」（代碼只在 skipped／略過時看）。
    - `generationEventCopy.spec.ts:89-125`：`reason: 'skipped'` 那列改成略過列（`stage` 已略過、`glyph` circle-dashed、`detail`）；`:115` 註解改寫。新增單次 `failureReason: 'no_text_source'` → 略過列（`announce` 逐字）。
    - `useGenerationJobsFeed.spec.ts:697-717` 改名「a `skipped` batch failure keeps its code; `busy_elsewhere` still carries none」：`skipped`＋`'no_text_source'` → 保留；`busy_elsewhere`＋任何代碼 → `null`；`error` 沒代碼 → `null`（保留）。
    - `GenerationWorkspaceV2.spec.tsx:276-290`：`m4` 列徽章「已略過」、徽章沒有 `--error-tint`、有 `--bg-tertiary`、沒有警告三角形、`data-state="failed"` 不變、說明行「沒有可用的字幕來源」且 `--text-secondary`。新增：同一次 render，佇列 `m5`（`skipped`＋`no_text_source`）與即時活動同一部片的略過列講同一句；即時活動略過列原因在 `workspace-feed-row-detail`（gen-queue 新增的 testid）裡。
    - `GenerationBatchDialogV2.spec.tsx:1315-1329`（2026-10-20 改寫，推翻「不含已略過」）：略過列狀態字是「已略過」、`--text-muted`、沒有 `CircleAlert`；原因「沒有可用的字幕來源」在 `gen-batch-row-detail-<id>` 裡（第二行），**不在**狀態字那格；`data-state` 仍是 `failed`。新增 `skipped`＋`no_text_source` 那列：狀態字「已略過」、第二行恰為 19 字那句；`skipped`＋`asr_unconfigured` 第二行「語音辨識未設定」。迴歸：同一份 render 的 `error` 失敗列狀態字仍是紅色「失敗」。#851 新增的對話框兩行測試與既有 `:1355`、`:1424`、`:2019`、E2E `batch-subtitle.spec.ts:254-256` 仍綠。
11. **故意改壞驗證**（每項改壞→看到紅→還原，寫進 Debug Log）：(i) `FAILURE_COPY_BY_REASON.no_text_source` 文字改一個字 → 三方一致紅；(ii) `no_text_source` 的 `skipped` 改 `false` → 單次略過列測試、三方一致的 stage 紅；(iii) hook `:524` 改回只留 `error` → hook 測試與工作區左右一致測試紅；(iv) `queueRowIsSkipped` 拿掉 `reason === 'skipped'` → 佇列與批次列測試紅；(v) 工作區徽章底色忘了換 → 10 的 `--error-tint` 斷言紅；(vi) 讓 `busy_elsewhere` 也看代碼 → 迴歸測試紅。

**量測、基準、收尾**

12. **真頁面量測**（本機、stub；片名「和平使者 S01E01」「魔戒首部曲：魔戒現身 加長版導演剪輯特別收藏版」，原因用 19 字那句）。數字寫進 Completion Notes：
    - 工作區，390／320／1440：即時活動略過列原因整句（手機可折行）、片名照 gen-queue 規則（最多兩行）；佇列略過列徽章「已略過」一行、說明行整句。
    - 產生字幕對話框（2026-10-20 改，沿用 #851 的合格線）：F8-M-v2 sheet 寬 320／390、桌機對話框 880，略過列（19 字原因）：片名那格 390 ≥200px、320 ≥150px，「和平使者 S01E01」兩寬都整個看得到；狀態字「已略過」與片名第一行對齊（差 ≤1px）；第二行原因整句、可折行。不合格 → 停下來回報 Sally，不合併。
13. **UX 驗收：** 對話框略過列對 F8-SPEC-ROW（`f8-spec-row.png`）與 F8-M-v2 `H717g`／F8-D-v2 `i9Nun1`（2026-10-20 新稿）。工作區與即時活動沒有對應稿，改對「最接近的既有狀態」：1440 截圖裡略過列的圖示／階段字色調與 ⑧ 批次已取消一致（`f11-spec-log.png`）、兩行位置與 ④ 一致；佇列略過徽章色調與已暫停／已取消一致（`f9-d-v2.png` 或畫廊 `budget_ceiling` 的已暫停列）。截圖附在 PR，請 Sally 追認（見下）。
14. **視覺基準：**
    - 夾具：`generation-workspace-v2/complete-with-failures` 的佇列 `skipped` 項目與即時活動 `skipped` 列加 `failureReason: 'no_text_source'`（展示完整那句）；`generation-batch-dialog-v2/complete-with-failures` 的 `skipped` 項目同樣加（這張 #851 會先因兩行改版更新過——以 #851 合併後的 darwin 為起點）。不新增夾具（gen-queue 的 `failures-long-titles` 不含略過，不動）。
    - **預期會變：** 上面兩張。**預期不變：** 其餘生成相關基準（`running`、`budget_ceiling`、`credit_exhausted`、`idle`、`failures-long-titles`、對話框其他）——變了要查，不要直接更新。
    - 做法：兩張的 `-darwin` 本機產（刪舊 darwin 後 `AI_PROVIDER=claude pnpm test:visual:update-missing`）；兩張的 `default-visual-linux.png` **`git rm`**，交給 CI 的 Visual Regression 開 `chore(visual): bootstrap N missing -linux baselines` PR。**絕不跑 `test:visual:update`、不 commit 本機產的 `-linux.png`。**
15. `pnpm nx run web:test`（subtitle＋hooks＋activity 相關 spec）、`pnpm nx run web:typecheck`、`pnpm run lint:all`（看「✖ N problems (X errors」那行）、`pnpm run format:check`、`python3 scripts/check-design-refs.py` 綠；跑一次 `tests/e2e/generation-workspace.spec.ts`、`generation-workspace-mobile.spec.ts`、`batch-subtitle.spec.ts` 綠。

## Tasks / Subtasks

- [ ] **T0 解開條件（AC #0）**：確認 gen-queue 已合併、`-a` 狀態、契約 ack。
- [ ] **T1 文字表（AC #1）**：`runFailureCopy.ts` 型別、旗標、新條目；註解。
- [ ] **T2 佇列＋對話框（AC #2–#5）**：`generationQueueRow.ts`（`queueRowIsSkipped`、label、badge、`FAILURE_TEXT` 註解）、`GenerationWorkspaceV2.tsx`（徽章底色／圖示、可選 `data-skipped`）、`GenerationBatchDialogV2.tsx`（只確認略過列吃到 `queueRowBadge`「已略過」與灰色無圖示；兩行結構是 #851 的）。
- [ ] **T3 即時活動（AC #6–#8）**：`useGenerationJobsFeed.ts:524`、`generationEventCopy.ts`（`skippedView`、單次與批次分支、檔頭）、`subtitleService.ts` 註解。
- [ ] **T4 測試（AC #9–#11）**：三方一致擴充、六個 spec 改寫／新增、六種故意改壞。
- [ ] **T5 量測＋UX＋基準＋收尾（AC #12–#15）**：390／320／1440／880 量測、對最接近稿的截圖、兩個夾具、兩張 darwin 更新與 linux 刪除、lint／typecheck／format／design-refs／E2E；更新 `sprint-status.yaml`。

## Dev Notes

### ⚖️ 待 Sally 追認（不擋開工，預設照做）

1. **灰色樣式**：即時活動略過列 `circle-dashed`＋`--text-muted` 圖示＋`--text-secondary` 階段字（同 ⑧ 批次已取消的色調；圖示借「已中斷」的虛線圓，意思是「沒完成」）；佇列徽章 `--bg-tertiary`＋`--text-muted`、無圖示（同已取消）。設計稿沒有這一列／這個徽章狀態——要補稿另開 design 單。
2. **推翻 dsr-6d-b／CR L8「不准用已略過」**：party mode 裁定要用；第二行講清楚「也沒有設定語音辨識」，舊路徑沒設金鑰改送「語音辨識未設定」。
3. ~~**對話框不顯示「已略過」三個字**（只有一個標籤位，放原因句、灰色、無圖示）。~~ → **2026-10-20 Sally 新裁定推翻**：對話框略過列兩行，第一行灰色「已略過」、第二行原因（見 AC #5、檔尾）。
4. **總結句仍說「失敗 M 部」**（傘狀單 #3，預設不改）。

### 不要做的事

- 不要改後端（`-a` 的事），不要比對任何英文或中文錯誤字串判斷略過——只看 `reason === 'skipped'` 與代碼的 `skipped` 旗標。
- 不要改 `data-state`（E2E `generation-workspace.spec.ts:43` 靠它）、不要改既有 testid；只可新增 `data-skipped`。
- 不要改對話框 `QueueRow` 的兩行結構、`RowStageLabel` 高度／對齊、`detail` 條件（#851 的範圍）；本單只改 `queueRowBadge`／`queueRowLabel` 的回傳。
- 不要改「完成 N 部、失敗 M 部」、重試集合（對話框 `remainingIds`，`GenerationBatchDialogV2.tsx:144` 一帶）、批次層列 ⑦⑧⑩⑪。
- 不要讓 `busy_elsewhere`、`error` 的顏色或文字跟著變（迴歸測試釘住）。
- 不要動活動頁的列與篩選（skipped 不列，傘狀單「不做」）；`recentEventCopy.ts` 只因型別多一個代碼而受影響。
- 不要動管理字幕對話框、詳細頁按鈕、`generateCostView.ts`（#848 `-b3` 的範圍）。
- 不要用 `text-[Npx]` 任意字級；不要跑 `test:visual:update`、不要產或 commit `-linux.png`。

### 已知但不處理

- 片庫海報徽章對同一部片寫「無字幕源」（媒體狀態 `no_text_source`，`libraryStatus.ts:166-169`），生成畫面寫「已略過」——兩個畫面、兩個情境（片子的狀態 vs. 這次生成的結果），本單不統一。
- 批次 `und` 軌不聽寫（傘狀單 #4）：那種列是灰色「已略過 · 沒有可用的字幕來源」，不帶「也沒有設定語音辨識」（後端送 `""`，正確）。

### 測試標準

- Vitest＋Testing Library；斷言用可見文字、testid、`data-state`、token class（Rule 16）。版面寬度只能在真瀏覽器量（AC #12）。不要背景跑測試。

### E2E data-testid

- 既有 testid 全部不變（`workspace-queue-row-*`、`gen-batch-row-*`、`workspace-feed-row`、`workspace-feed-row-detail`、`workspace-terminal-label`）。可新增 `data-skipped`。
- `git grep -n "已略過\|沒有可用的字幕來源" tests/e2e` 開工時應為零（2026-10-19 是零）；E2E 只斷言 `busy_elsewhere` 文字、`data-state="failed"`、總結句 → 不受影響，仍跑一次（AC #15）。

### 視覺基準

- 會變：`generation-workspace-v2/complete-with-failures`、`generation-batch-dialog-v2/complete-with-failures`（darwin 本機更新、linux 刪掉交給 CI bootstrap）。注意前者 gen-queue 那張剛更新過——以合併後的 darwin 為起點。

### Project Structure Notes

- 動到：`apps/web/src/components/subtitle/{runFailureCopy.ts,generationQueueRow.ts,generationQueueRow.spec.ts,generationEventCopy.ts,generationEventCopy.spec.ts,GenerationWorkspaceV2.tsx,GenerationWorkspaceV2.spec.tsx,GenerationBatchDialogV2.tsx,GenerationBatchDialogV2.spec.tsx}`、`apps/web/src/hooks/{useGenerationJobsFeed.ts,useGenerationJobsFeed.spec.ts}`、`apps/web/src/components/activity/recentEventCopy.spec.ts`、`apps/web/src/services/subtitleService.ts`（註解）、`apps/web/src/routes/test/-gallery.fixtures.tsx`、`tests/visual/components.visual.spec.ts-snapshots/components/{generation-workspace-v2,generation-batch-dialog-v2}/complete-with-failures/`、`_bmad-output/implementation-artifacts/sprint-status.yaml`。
- 不動 `ux-design.pen`、`_bmad-output/screenshots/`。
- 規模估計：約 13 檔、400～600 行（含測試），比 gen-queue（約 12 檔、600～800 行）小，不再拆。

### Time-dependent visual coverage

- N/A — 沒有讀時鐘的元件改動。

### References

- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:1-34, 47-93, 103-152, 169-181, 206-230, 271-281, 306-315]
- [Source: apps/web/src/components/subtitle/generationQueueRow.ts:25-54, 69-142]
- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:243-371, 373-420]
- [Source: apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx:163-192, 345-394]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:96-109, 427-457, 505-525]
- [Source: apps/web/src/services/subtitleService.ts:155-184]
- [Source: apps/web/src/utils/libraryStatus.ts:42-48, 160-171]（中性色前例）
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:6921-6961, 7310-7358]
- [Source: generationQueueRow.spec.ts:45-60；generationEventCopy.spec.ts:89-125, 315-383；useGenerationJobsFeed.spec.ts:697-717；GenerationWorkspaceV2.spec.tsx:276-290；GenerationBatchDialogV2.spec.tsx:1315-1329；recentEventCopy.spec.ts:130-142]
- [Source: tests/e2e/generation-workspace.spec.ts:37-79；batch-subtitle.spec.ts:244-257]
- [Source: apps/api/internal/subtitle/solo_runner.go:507-508]（後端原句）
- [Source: _bmad-output/implementation-artifacts/dsr-6d-b-batch-dialog.md:92, 205]（被推翻的「不准已略過」）
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-live-activity-row-layout-design.md 裁定 1–6]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-generation-queue-failure-reason.md AC #1–#4、#7、#10、#16]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-no-text-source-skip-a.md AC #1 `[@contract-v2]`]（#855 已合併）
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-generation-queue-failure-reason.md「2026-10-20 追加」]（對話框兩行結構，PR #851）
- [Source: F8-D-v2 `i9Nun1`、F8-M-v2 `H717g`、F8-SPEC-ROW `flow-f-subtitle-v2/f8-spec-row.png`（2026-10-20 繪製中）]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-19 | 建單（Bob）。party mode 裁定灰色「已略過」＋第二行原因；⚖️ Alexyu 由這邊直接做。樣式預設中性色，待 Sally 追認 4 項。 |
| 2026-10-20 | Bob：套 Sally 新裁定（推翻追認 #3）——對話框略過列兩行，第一行灰色「已略過」、第二行原因；改寫 AC #0、#5、#10（對話框）、#12（對話框量測）、#13、#14、T2、Dev Notes。記錄依賴：對話框兩行結構由 gen-queue「2026-10-20 追加」做（PR #851），本單等 #851 合併才開工。`-a` 已合併 #855，契約 v2。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## ⚖️ Sally 追認（2026-10-19，開工前）

1. 「已略過」沿用「已取消」「批次已取消」的中性灰。理由：沒壞、沒花錢，不該是紅色。
2. 推翻 dsr-6d-b「佇列不准用『已略過』開頭」：當時怕使用者以為是系統自作主張；現在第二行講明原因，這個顧慮不在了。
3. ~~產生字幕對話框只放灰色原因句、不顯示「已略過」三個字：同意。手機片名若只剩 1～2 個字，照停止條件停下來回報。~~（2026-10-20 推翻，見下）
4. 批次結尾「完成 N 部、失敗 M 部」先維持把略過算進失敗——跟灰色「已略過」說法不一致，另立 backlog `disc-2026-10-batch-summary-counts-skip-as-fail`，不在本單。

## ⚖️ Sally 新裁定（2026-10-20，推翻上面 #3）

Alexyu 選 B（對話框佇列列改兩行），Sally 裁定：

- 「產生字幕」批次對話框（F8，`GenerationBatchDialogV2.tsx` 的 `QueueRow`）裡，**有東西要解釋的列（失敗、已略過）一律兩行**；其他狀態維持一行。
- **第一行：** 片名，最多兩行（`line-clamp-2`）＋右邊短狀態字，對齊片名第一行。
- **第二行：** 原因，Body 字級、`--text-secondary`，左緣對齊片名，前面不加「·」，不截斷、會折行。
- **已略過：** 灰色「已略過」（`--text-muted`）、無圖示；第二行是原因，例如「沒有可用的字幕來源，也沒有設定語音辨識」。
- 桌機 F8-D-v2 同一排法。
- 設計稿：F8-D-v2 `i9Nun1`、F8-M-v2 `H717g`、新規格稿 F8-SPEC-ROW（`flow-f-subtitle-v2/f8-spec-row.png`，繪製中）。
- 分工：兩行結構由 `disc-2026-10-generation-queue-failure-reason`（PR #851 重開）做；本單只讓 `queueRowBadge`／`queueRowLabel` 對帶 `no_text_source`／`asr_unconfigured`（或 `reason: skipped`）的略過項目回「已略過」＋原因。**本單等 #851 合併才開工。**
