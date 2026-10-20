# Disc：生成畫面的失敗原因——左邊佇列講出原因、右邊即時活動把原因放第二行，片名不再被擠掉

Status: in-progress（2026-10-20 重開：PR #851 為「對話框佇列列改兩行」重開，見「2026-10-20 追加」；原：Sally 稿 `disc-2026-10-live-activity-row-layout-design` 已跟本檔同一個 PR 合併，2026-10-19 解除 blocked）

**Source（兩件事併一張）：**
1. `disc-2026-10-live-activity-failure-reasons-b`（PR #846）合併時 dev 回報（範圍外）：生成字幕畫面右邊「即時活動」已經寫「影片所在的資料夾寫不進字幕」，左邊「生成佇列」同一部片還寫「生成失敗」——同一個畫面、同一次失敗，兩邊對不起來。⚖️ **Alexyu 2026-10-19 裁定 2A：** 開單補上，佇列失敗項目改用同一份失敗原因文字（`FAILURE_COPY_BY_REASON`）。
2. 同一份回報：即時活動失敗列「失敗 · 片名 · 原因」擠在同一行，最長的原因佔 182px，手機 390 片名只剩 34px（「和…」「魔…」「花…」），桌機 400px 側欄也把「和平使者 S01E01」截成「和平使者 …」。⚖️ **Alexyu 裁定 1A** → Sally 出版面方案 `disc-2026-10-live-activity-row-layout-design`（2026-10-19），**建議實作併進本單**（兩件事改同一批檔，分開做容易衝突）。

⚖️ **SM 裁定（Bob）：不拆。** 全前端，跨棧拆單規則（前後端各 >3 task）不觸發。規模估計約 12 檔、600～800 行（含測試），跟上一張同類單 #846（11 檔、+428／−237）同一級，沒有到「明顯超過」。照「切分線選版面」的規則，可以切成「左欄佇列」和「右欄即時活動」兩張，但：(a) Sally 的裁定是**一個**排法規則同時套在兩欄（片名最多兩行、原因在第二行），本單的價值就是「同一個畫面左右兩邊講同一句、而且都看得清楚」，要在同一張基準圖上一起證明；(b) 即時活動的批次失敗列在本單改成直接呼叫佇列的同一個函式（AC #4），切開就要先後各動一次 `generationEventCopy.ts`。**後備切分線**：dev 中途若發現超過 1,000 行或 T3 版面卡住，停在 T2 完成處，T3／T5 另開 `-b`（右欄），本單只出左欄。

## 設計稿 / Design ref

- **版面方案（本單照做）：** `disc-2026-10-live-activity-row-layout-design`（Sally，2026-10-19；scratchpad 版 `…/scratchpad/sally-live-row/disc-2026-10-live-activity-row-layout-design.md`，合併後在 `_bmad-output/implementation-artifacts/` 同名檔）。
- **改動的三張稿（Sally 稿合併後才有新節點）：**
  - **F11-SPEC-LOG `tT3JX`**（`flow-f-subtitle-v2/f11-spec-log.png`）：④ 失敗 `dNQa0`→`stf9c`、⑤ 已停止 `xPTom`→`cu3CY`、⑤b 帳號沒錢 `xkmMv`→`c5WP3` 改兩行（`evt-line` ＋ `evt-reason-row`）；新增「④ 手機」示範 `spec-entry-m`（`row-box-m` 358 寬）。⚠️ 失敗列是 **④**，不是 #846 sprint-status 寫的 ⑥（⑥ 是「批次完成」）。
  - **F11-D-v2 `l8FsB`**（`f11-d-v2.png`）：即時活動 `DUvwI`／清單 `dPQ9b` 第 3 列新增 `evt-fail`（和平使者 S01E01 · 影片所在的資料夾寫不進字幕）；佇列失敗列 `f14sNh`（`aw4Qr` 實例）同一部片、同一句、說明行 `$text-secondary`。
  - **F11-M-v2 `PXB0z`**（`f11-m-v2.png`）：佇列失敗列 `Ah8fN`（358 寬）同上。
- **元件：** `Component/GenQueueRow-v2 (aw4Qr)`——片名 `bSNmA`、說明行 `lUZol`、徽章 `IcPTo`／圖示 `CQhw2`／徽章字 `OLg1J`。
- ~~**不在 Sally 稿範圍：** 產生字幕對話框 F8-D-v2 `i9Nun1`／F8-M-v2 `H717g`（見「預設做法」）。~~ → **2026-10-20：** 對話框也有稿了——F8-D-v2 `i9Nun1`、F8-M-v2 `H717g`、新規格稿 F8-SPEC-ROW（`flow-f-subtitle-v2/f8-spec-row.png`，繪製中），見「2026-10-20 追加」。

### ⛔ 擋在哪、什麼時候解開

- Sally 稿 2026-10-19 狀態是 **ready-for-design**：等 Alexyu 在 Pencil 跑 inline agent、存檔，再由 Sally 複審、出三張截圖、更新 `pen-node-ids.json`，跟 `.pen` 同一個 PR 合併。**本單要等那個 PR 合併才開工**——AC #8 的 UX 驗收要對新截圖，`check-design-refs.py` 也要認得新節點。
- 解開條件：`origin/main` 的 `sprint-status.yaml` 裡 `disc-2026-10-live-activity-row-layout-design: done`，且 `_bmad-output/screenshots/flow-f-subtitle-v2/f11-spec-log.png` 有 `spec-entry-m`。
- 若 Sally 複審時改了下面任何一條裁定（兩行、不加「·」、片名兩行、圖示對第一行、佇列說明行次要色、批次層列不動），**以合併後的稿為準**，dev 先回報 SM 改 AC 再做。

## 查到的事（2026-10-19，讀 `origin/main` 577d6fca）

### 左邊：生成佇列

- **佇列失敗項目在哪裡畫：**
  - 生成工作區（`/activity?view=generation`）：`QueueRow`（`apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:285-371`）。`:292-295` 呼叫 `queueRowLabel({ status: item.status, reason: item.reason }, phase)`——**沒傳 `failureReason`**。片名 `:319-321`（`truncate`，單行）；說明行 `:322-329`（`queueRowSubStatus(item, label.text)`，failed 時就是 `labelText`，`generationQueueRow.ts:131-132`；class `truncate text-sm max-sm:whitespace-normal`，failed 用 `--text-secondary`）；徽章「失敗」（`generationQueueRow.ts:104-105`）、紅底 `BADGE_TINT.failed`（`:249`）、`TriangleAlert`（`:267-268`）。列表 `:916-930`，`data-testid="workspace-queue-row-${mediaId}"`（`:309`）。
  - 產生字幕對話框（F8）：`QueueRow`（`GenerationBatchDialogV2.tsx:345-394`）→ `RowStageLabel`（`:176-192`），`:373` 同樣只傳 `{ status, reason }`；片名 `:369`（`truncate`），標籤 `shrink-0` 跟片名同一行。`data-testid="gen-batch-row-${mediaId}"`（`:358`）。
- **現在怎麼決定寫「生成失敗」：** `queueRowLabel`（`generationQueueRow.ts:69-92`）`case 'failed'`（`:73-78`）查 `FAILURE_TEXT[view.reason]`（`:48-54`）：`skipped`→沒有可用的字幕來源、`busy_elsewhere`→這部正在別處處理、`error`→**生成失敗**、`''`（舊後端）→生成失敗。不看原因代碼。
- **資料已經在：** `GenerationBatchItemState.failureReason?: string`（`apps/web/src/services/subtitleService.ts:176-183`）。後端 `GenerationBatchItemState.FailureReason`（`apps/api/internal/services/generation_batch.go:151-156`，`json:"failure_reason"` 一定在）：`failed`＋`error` 才有代碼（`:816-817`，跟活動記錄同一個分類函式 `models.SubtitleRunFailureReason`；整批當掉 `:913` 寫 `other`），其餘 `''`。`useGenerationBatchProgress` 的 `mergeItems`（`apps/web/src/hooks/useGenerationBatchProgress.ts:139-152`）以 `changed_item` 整項取代、終局 `items` 整批取代，經 `snakeToCamel`（`:287`）→ `failureReason` 一路到 `progress.items`。**不改後端、不改 hook。** 全 repo 只有上面兩個呼叫點組 `{ status, reason }`。
- **單次生成（詳情頁按的）：** 「進行中任務」（`GenerationWorkspaceV2.tsx:932-970`）的列 `status` 寫死 `'running'`（`:948-949`）；單次失敗時 hook 直接把那筆從 `singleJobs` 拿掉（`useGenerationJobsFeed.ts:456`）。**佇列裡不會出現單次失敗列**，失敗只出現在右邊即時活動（#846 已讀 `transcription_failed.failure_reason`）。→ 佇列這邊沒有東西要改。
- **預算／額度的「已停止」在佇列：** 批次撞到預算或沒額度時，後端把正在做的與排隊的都轉 `paused`（`generation_batch.go:747-750`、`:767-788`、`:905-906`），**不是 `failed`**。佇列顯示灰色「已暫停」＋「已暫停 — 下次繼續」（`generationQueueRow.ts:85-86`、`:108-109`、`:135-136`）。本單不動。
  - 邊角：`failed`＋`error`＋代碼 `budget_exceeded`／`credit_exhausted` 實務上走不到（`ai.ErrBudgetExceeded`、`*ai.CreditExhaustedError` 在 `default` 前就被接走），只有錯誤被包成字串丟了型別時才會；照即時活動批次列（`generationEventCopy.ts:222-229`，批次 `error` 一律紅色失敗、只取 `.text`）。
- **`other`／缺代碼：** `failureCopyForReason`（`generationEventCopy.ts:89-93`）對 `other`、`''`、`null`、`undefined`、不認得、`constructor`／`__proto__` 一律回「生成失敗」——跟今天佇列的字一樣。
- **循環引用的坑：** `generationEventCopy.ts:21` 已 import `generationQueueRow`；若 `generationQueueRow.ts` 反過來 import `failureCopyForReason` 就互相引用（eslint 沒開 `import/no-cycle`，執行期剛好不壞，但脆弱）→ T1 把文字搬到葉節點檔。

### 右邊：即時活動

- **文字：** `feedRowView`（`generationEventCopy.ts:154-304`）。`FeedRowView`（`:103-122`）只有 `parts: string[]`（「每段前面都有 `·`」，`:111`）。④ 失敗走 `failedView`（`:306-315`，`parts: [name, reason]`）；⑤／⑤b 單次 `stopped`（`:210-218`，`parts: [name, copy.text]`）；批次 `error`（`:225-229`）。`announce` 是「失敗：片名，原因」／「已停止：片名，原因」。
- **版面：** `FeedRowItem`（`GenerationWorkspaceV2.tsx:383-420`）——`<li class="flex items-center gap-2 px-3.5 py-2">`，圖示 16、階段字 `shrink-0`、每個 part 前面一個 `·`；第一段（片名）`min-w-0 shrink`＋`truncate`、其餘 `shrink-0`（`:398-405`「The film name may truncate; the reason after it stays whole」）、尾端 `ml-auto shrink-0`（`:413-417`）。**原因越長、片名越短。**
- **#846 實測（dev 量）：** 390（F11-M-v2 展開即時活動）列寬 356px，原因 182px，片名 34px（1 字＋…）；1440 側欄 398px，片名 76px（約 5 字），「和平使者 S01E01」→「和平使者 …」。
- **Sally 試算（兩行之後）：** 390 片名那行約 240px（約 17 字），原因那行約 300px；400 側欄片名約 290px（約 20 字）；兩行片名手機約 34 字、桌機約 40 字。
- **批次層列：** ⑦ 批次完成 `parts: ['本批次', '完成 N 部、失敗 M 部']`（`:282-293`）、⑧ 批次已取消（`:271-281`）、⑩⑪ 預算／額度（`:233-260`，有 `trail`）——Sally 裁定 3：**不動**。

### 測試、夾具、基準、E2E

- **寫死 `parts` 的單元測試：** `generationEventCopy.spec.ts` ④ `:89-125`、單次 `:127-145`、⑤ `:147-160`、⑤b `:162-176`、三方一致性 `:317-357`（`single.parts[1]`、`:343`、`:354`）。這些 `parts[1]` 在本單會搬到 `detail`。批次列 `:211-240` 不變。
- **寫死「生成失敗」的佇列測試**（沒帶代碼，改完**仍應是「生成失敗」**，不改）：`generationQueueRow.spec.ts:65-71`；`GenerationWorkspaceV2.spec.tsx:293-309`；`GenerationBatchDialogV2.spec.tsx:1355`、`:1424`、`:2019`。
- **元件測試對即時活動：** `GenerationWorkspaceV2.spec.tsx` `rows()`（`:593`）用 `toHaveTextContent`，不看結構；`:656-740`（#13，hook＋假 EventSource）斷言四列的原因字與「不含 `/mnt/media`」——改兩行後文字仍在，應仍綠。
- **畫廊夾具：** 生成工作區夾具 5 個（`-gallery.fixtures.tsx:7097` running、`:7158` budget_ceiling、`:7217` credit_exhausted、`:7272` complete-with-failures、`:7322` idle），都是 `width: 1200`（桌機，即時活動是側欄）。**只有 `complete-with-failures`（`:7272-7320`）有失敗的即時活動列**（`:7299-7300`：`busy_elsewhere`、`skipped`，各一列 ④）→ 這兩列會變兩行，**這張基準會變**。其他四張的列都是單行、片名短，**不應變**（若變了要查，可能是圖示對齊改動造成的位移）。夾具片名都短（沙丘：第二部／奧本海默／怪奇物語 S04E07 第七章），兩行片名不會被觸發。對話框夾具（`:6895-6910`）只有 `busy_elsewhere`／`skipped` → 不變。
- **基準檔：** `tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/{running,budget_ceiling,credit_exhausted,complete-with-failures,idle}/default-visual-{darwin,linux}.png`。
- **E2E：** `tests/e2e/generation-workspace.spec.ts:42-44`、`:62-63`、`batch-subtitle.spec.ts:254-256` 只斷言 `busy_elsewhere`／`skipped` 的佇列文字；`generation-workspace.spec.ts`／`-mobile.spec.ts` 對即時活動只數 `workspace-feed-row` 列數、找「翻譯中」（#846 查過：`:147`、`:203`、`:214`、mobile `:155`），stub 不送 `transcription_failed`。→ **E2E 不受影響**，仍要照規矩 grep＋跑一次（AC #10）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要生成字幕畫面左邊的佇列和右邊的即時活動，對同一部失敗的片講出同一句原因（例如「影片所在的資料夾寫不進字幕」），而且在手機上也看得到是哪一部片（「和平使者 S01E01」整個看得到，不是「和…」），
這樣我一眼就知道是哪部片、要去修什麼，不用對照兩邊或跑去活動頁查。

## Acceptance Criteria

0. **開工前確認解開條件**（見「⛔ 擋在哪」）：Sally 稿 PR 已合併、三張截圖已更新；稿子的裁定跟本單一致。結果寫進 Completion Notes。

**左邊：佇列講出原因**

1. **文字搬成葉節點檔（純搬家，行為不變）：** 新檔 `apps/web/src/components/subtitle/runFailureCopy.ts`（檔頭 `// Implements: <utility — no .pen counterpart>`），放 `FailureCopy`、`RunFailureReason`、`FAILURE_COPY_BY_REASON`、`failureCopyForReason`（原 `generationEventCopy.ts:26-34`、`:36-93`，註解一起搬，不改任何字或 `stopped`）。`generationEventCopy.ts` 以 `export { … } from './runFailureCopy'` 轉出（`recentEventCopy.ts:17` 與既有 spec 不用動；要直接改指新檔也可，擇一）。`runFailureCopy.ts` **不 import** `generationQueueRow`／`generationEventCopy`。
2. **佇列標籤讀代碼：** `QueueRowView`（`generationQueueRow.ts:25-28`）加 `failureReason?: string | null`。`queueRowLabel` 的 `failed`：
   - `reason === 'error'` → `text = failureCopyForReason(view.failureReason).text`；顏色、圖示照舊（`--error-text`、`alert`），**不看 `stopped`**。
   - `skipped`／`busy_elsewhere`／`''` → 照舊 `FAILURE_TEXT`，帶了代碼也不看。
   - 檔頭（`:1-16`）與 `FAILURE_TEXT` 上方註解補：`error` 的字來自 `FAILURE_COPY_BY_REASON`，`FAILURE_TEXT.error` 只是沒代碼時的退回。
3. **兩個呼叫點都傳代碼：** `GenerationWorkspaceV2.tsx:293`、`GenerationBatchDialogV2.tsx:373` 的 view 加 `failureReason: item.failureReason`。對話框的退回列（`:514-520`、`:769-795`）沒代碼 → 照舊「生成失敗」。
4. **即時活動批次列收成同一個呼叫：** `generationEventCopy.ts:225-228` 改成 `queueRowLabel({ status: 'failed', reason: row.reason, failureReason: row.failureReason }).text`，刪掉那裡自己的 `error` 分支。單次（`row.reason === null`）那條不動。
5. **佇列說明行顏色維持次要色**（Sally 裁定 6：app 本來就對，改的是稿子）：失敗列說明行仍是 `--text-secondary`，不改成紅色；只加測試釘住（AC #12）。
6. **佇列片名最多兩行**（Sally 裁定 2、6）：工作區 `QueueRow` 片名（`GenerationWorkspaceV2.tsx:319`）`truncate` → `line-clamp-2`（超過兩行結尾「…」，可折行）。**只改工作區**；對話框片名（`GenerationBatchDialogV2.tsx:369`）不在 Sally 稿範圍，不動。（⚠️ 2026-10-20 取代：對話框片名也改 `line-clamp-2`，見 AC #18。）

**右邊：即時活動原因放第二行**

7. **文字層把原因跟片名分開：** `FeedRowView` 加 `detail: string | null`（註解：「第二行，完整顯示、前面不加 `·`」，F11-SPEC-LOG ④⑤⑤b）；`view()` 預設 `detail: null`。
   - ④ `failedView`：`parts: [name]`、`detail: reason`；⑤／⑤b 單次 stopped：`parts: [name]`、`detail: copy.text`。批次 `error`／`skipped`／`busy_elsewhere` 都走 `failedView`，一樣。
   - `announce` 字串**完全不變**（「失敗：片名，原因」「已停止：片名，原因」）。
   - 其他列（進行中、已過、中斷、完成、⑦⑧⑩⑪ 批次列）`detail: null`，`parts` 不變。
8. **列的版面改成「圖示＋內容欄」**（`FeedRowItem`，`GenerationWorkspaceV2.tsx:383-420`）：
   - 內容欄第一行：階段字 · 各段 parts（各段前面照舊加 `·`）· 尾端（`ml-auto`）。第一段（片名）`min-w-0`＋`line-clamp-2`（取代 `truncate`）、可折行；其餘段 `shrink-0` 照舊。
   - 有 `detail` 時第二行顯示原因：左緣對齊階段字（＝圖示 16＋間距 8 之後，內容欄自然對齊）、**前面沒有 `·`**、`text-sm text-[var(--text-secondary)]`、**不截斷**，太長就折行。兩行之間不加間距。
   - 圖示垂直置中在**第一行**（不是整列置中）；列的上下內距不變（`py-2`／`px-3.5`）。
   - 桌機與手機同一個排法（Sally 裁定 5）；`data-testid="workspace-feed-row"` 不變。可新增 `data-testid="workspace-feed-row-detail"` 給原因那一行（新增，不改既有）。
   - 更新註解：`generationEventCopy.ts:5-6` 與 `FeedRowItem` 上方（`:382`、`:403`「the reason after it stays whole」）改寫成兩行排法，`Design ref` 仍是 `tT3JX`／`DUvwI`。
9. **批次層列不動：** ⑦⑧ 仍一行「批次完成 · 本批次 · 完成 N 部、失敗 M 部」；⑩⑪ 尾端金額／廠商位置不變。

**測試**

10. **三方一致（會變紅的那種）：** 把 `generationEventCopy.spec.ts:317-357` 擴成「live log ⇄ activity record ⇄ queue」，對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一：單次列 `detail`、批次 `error` 列 `detail`、`queueRowLabel({ status:'failed', reason:'error', failureReason: code }).text` 三者 `===` `FAILURE_COPY_BY_REASON[code].text`，且 `===` `recentSubtitleDetail(...)` 的 ` · ` 後半段；單次列 `stage` 依 `stopped` 是「已停止」／「失敗」（既有）。
11. **改寫既有文字層測試（改寫，不是刪）：** `generationEventCopy.spec.ts` ④ `:89-125`、單次 `:127-145`、⑤ `:147-160`、⑤b `:162-176` 的 `parts: [name, text]` → `parts: [name]` ＋ `detail: text`，`announce` 斷言保留逐字；新增「進行中／完成／批次列 `detail === null`」與「⑦ 批次完成帶失敗仍是 `parts: ['本批次', '完成 3 部、失敗 2 部']`、`detail: null`」。
12. **元件層測試：**
    - `generationQueueRow.spec.ts`：`error`＋`target_not_writable`／`cancelled`／`budget_exceeded`（紅、`alert`、「已達預算上限」）；`error`＋`null`／`''`／`'nope'`／`'constructor'` →「生成失敗」；`busy_elsewhere`＋`'target_not_writable'` → 仍「這部正在別處處理」。既有 `:65-71` 保留。
    - `GenerationWorkspaceV2.spec.tsx`：
      (a) 佇列一列 `failed`＋`error`＋`target_not_writable` → `workspace-queue-row-<id>` 含原因、徽章仍「失敗」、`data-state="failed"`、不畫進度條、**說明行有 `--text-secondary`、沒有 `--error-text`**、片名元素有 `line-clamp-2` 沒有 `truncate`；
      (b) **同一次 render 左右對得起來**：`progress.items` 與 `feed` 同一部片同代碼 → 佇列那列與即時活動那列含同一句；
      (c) 即時活動 ④ 列：原因在 `workspace-feed-row-detail`（或等價的第二行元素）裡、**不在**片名那一行的容器裡、原因前沒有 `·`（第二行元素的 `textContent` 恰等於原因）；片名元素有 `line-clamp-2`；⑦ 批次列沒有第二行元素；
      (d) 既有 `:293-309`、`:656-740` 保留且綠。
    - `GenerationBatchDialogV2.spec.tsx`：新增一列 `failed`＋`error`＋`no_audio_track` → `gen-batch-row-<id>` 含「找不到可用的音軌」。既有 `:1355`、`:1424`、`:2019` 保留。
    - `useGenerationBatchProgress.spec.ts`（`:303` 旁）：`changed_item` 帶 `failure_reason: 'target_not_writable'` → 該項 `failureReason === 'target_not_writable'`。
13. **故意改壞驗證**（每項改壞→看到紅→還原，寫進 Debug Log）：(i) `queueRowLabel` 把任一代碼寫死成別的字 → 三方一致性紅；(ii) `queueRowLabel` 改回只查 `FAILURE_TEXT` → 除 `other` 全紅；(iii) 工作區 `QueueRow` 拿掉 `failureReason: item.failureReason` → 12(a)(b) 紅；(iv) `failedView` 把原因塞回 `parts` → 11 與 12(c) 紅；(v) `FeedRowItem` 原因前加回 `·` 或把原因放進第一行 → 12(c) 紅；(vi) 即時活動片名改回 `truncate` → 12(c) 紅；(vii) 佇列片名改回 `truncate` 或說明行改 `--error-text` → 12(a) 紅。

**量測、基準、收尾**

14. **真頁面量測**（本機、假 SSE 或狀態查詢 stub；片名用「和平使者 S01E01」「花月殺手」「魔戒首部曲：魔戒現身 加長版導演剪輯特別收藏版」，原因用最長的 `target_not_writable`「影片所在的資料夾寫不進字幕」）。數字寫進 Completion Notes：
    - 即時活動，**390 與 320**（手機展開即時活動）、**1440**（側欄）：片名那格寬度與行數、原因那行是否整句、圖示中心與第一行文字中心的差（≤1px）、原因左緣 x 與階段字左緣 x 的差（≤1px）。**合格：** 「和平使者 S01E01」「花月殺手」整個看得到；長片名最多兩行、結尾「…」；原因整句看得到（320 可折行）。
    - 佇列，390／320／1440：失敗列說明行整句（手機可折行）；長片名最多兩行。
    - 產生字幕對話框，390（F8-M-v2 手機 sheet）／880：片名那格寬度與看得到幾個字（不改版面，只記錄）。**對話框手機片名只剩「…」或 1～2 個字 → 停下來回報 Alexyu／Sally，不合併。**（⚠️ 2026-10-20 取代：對話框改用 AC #22 的量測與合格線。）
15. **UX 驗收（必做，Sally 稿合併後）：** 1440 截圖對 `f11-d-v2.png`（`evt-fail` 第 3 列、佇列 `f14sNh`）；390 展開即時活動截圖對 F11-SPEC-LOG `spec-entry-m`；390 佇列對 `f11-m-v2.png` `Ah8fN`。文字、兩行位置、顏色逐項比。
16. **視覺基準：**
    - 新增夾具 `generation-workspace-v2/failures-long-titles`（`width: 1200`，`penNode: 'screen-section'` 或沿用既有夾具寫法）：佇列＋即時活動同時有「和平使者 S01E01」`target_not_writable`、長片名「魔戒首部曲：魔戒現身 加長版導演剪輯特別收藏版」`no_audio_track`，證明兩邊同一句、原因在第二行、長片名兩行截斷。
    - **預期會變：** `complete-with-failures`（兩列 ④ 變兩行）。**預期不變：** `running`、`budget_ceiling`、`credit_exhausted`、`idle`——變了要查原因，不要直接更新。
    - 做法：`complete-with-failures` 與新夾具的 `-darwin` 本機產（刪舊 darwin 檔後 `AI_PROVIDER=claude pnpm test:visual:update-missing`）；`complete-with-failures/default-visual-linux.png` **刪掉**（`git rm`），新夾具**不產 `-linux`**，交給 CI 的 Visual Regression 開 `chore(visual): bootstrap N missing -linux baselines` PR。**絕不跑 `test:visual:update`、不 commit 本機產的 `-linux.png`。** 其餘基準 `AI_PROVIDER=claude pnpm test:visual`（不更新）全綠。
17. `pnpm nx run web:test`（subtitle＋hooks＋activity 相關 spec）、`pnpm nx run web:typecheck`、`pnpm run lint:all`（看「✖ N problems (X errors」那行）、`pnpm run format:check`、`python3 scripts/check-design-refs.py` 綠；`git grep -n "生成失敗" tests/e2e` 仍為零，跑一次 `tests/e2e/generation-workspace.spec.ts`、`generation-workspace-mobile.spec.ts`、`batch-subtitle.spec.ts` 綠。

## 2026-10-20 追加：對話框佇列列改兩行（Alexyu 選 B，Sally 裁定）

**為什麼重開：** 本單原本對話框「只改字、不改版面」。PR #851 實測：對話框裡失敗列的原因（例如「影片所在的資料夾寫不進字幕」）跟片名擠在同一行，手機上片名被擠到看不見是哪一部。Alexyu 選 B（改兩行），Sally 2026-10-20 裁定版面，**在 PR #851 上做**（重開，不另開單）。

**設計稿：** F8-D-v2 `i9Nun1`、F8-M-v2 `H717g`、新規格稿 **F8-SPEC-ROW**（截圖 `_bmad-output/screenshots/flow-f-subtitle-v2/f8-spec-row.png`，2026-10-20 繪製中；節點 id 等稿 PR 合併、`pen-node-ids.json` 更新後才寫進 `Design ref:`，否則 `check-design-refs.py` 會紅）。

**裁定：** 產生字幕對話框（F8，`GenerationBatchDialogV2.tsx` 的 `QueueRow`）裡，有東西要解釋的列（失敗、已略過）一律兩行；其他狀態維持一行。第一行片名（最多兩行）＋右邊短狀態字、對齊片名第一行；第二行原因（Body、`--text-secondary`、左緣對齊片名、前面不加「·」、不截斷、會折行）。桌機 F8-D-v2 同一排法。已略過（灰色「已略過」＋原因）由 `disc-2026-10-no-text-source-skip-b` 做，本單只做結構與失敗列。

**下游：** `disc-2026-10-no-text-source-skip-b` 等本單合併才開工，它只改 `queueRowBadge`／`queueRowLabel` 的回傳，靠本單的兩行結構顯示「已略過」＋原因。

18. **版面**（`QueueRow`，`GenerationBatchDialogV2.tsx:345-394`）：
    - 縮圖之後改成一個內容欄 `div.min-w-0.flex-1.flex-col`。
    - 內容欄第一行 `div.flex.items-start.gap-3`，裝片名與 `RowStageLabel`。片名由 `truncate` 改成 `line-clamp-2 break-words`，保留 `min-w-0 flex-1`。
    - 內容欄第二行：`{detail && <p data-testid={`gen-batch-row-detail-${mediaId}`} className="break-words text-sm text-[var(--text-secondary)]">{detail}</p>}`。
    - **不變：** 外層 `flex items-center gap-3.5`、進度條 `sm:pl-12`、`gen-batch-row-*` testid、`data-state`。
19. **狀態字與第二行：**
    - 失敗列的狀態字改成短字「失敗」（`queueRowBadge`）；顏色與圖示仍來自 `queueRowLabel`（紅色、`alert`）。
    - `detail` ＝ `item.status === 'failed'` 時 `queueRowLabel(view).text`，否則 `null`。
    - 非失敗列（完成、排隊、進行中、已暫停、已取消）狀態字照舊用 `queueRowLabel(...).text`、沒有第二行。
    - 退回列（沒有後端 items，`:514-520`、`:769-795`）照同一規則：失敗列狀態字「失敗」、第二行「生成失敗」。
20. **對齊：** `RowStageLabel` 的高度等於片名一行的行高（例如 20px 行高就 `h-5`），讓狀態字跟片名第一行置中對齊；**不要**用 `mt-[Npx]` 硬推。
21. **測試：**
    - 失敗列：原因在 `gen-batch-row-detail-<id>` 裡、狀態字是「失敗」（原因**不在**狀態字那格）；片名元素有 `line-clamp-2`、沒有 `truncate`。
    - 完成、排隊、進行中列沒有 detail 元素。
    - 既有 `GenerationBatchDialogV2.spec.tsx:1355`、`:1424`、`:2019` 與 E2E `tests/e2e/batch-subtitle.spec.ts:254-256` 仍綠（若既有斷言是「列含原因文字」，仍成立——原因還在同一列裡）。
    - 故意改壞（寫進 Debug Log）：(viii) 片名改回 `truncate` → 紅；(ix) 原因塞回狀態字 → 紅；(x) 非失敗列也給 `detail` → 紅。
22. **真瀏覽器量測**（數字寫進 Completion Notes）：F8-M-v2 sheet 寬 **320** 與 **390**、桌機對話框 **880**。片名用「和平使者 S01E01」與「和平使者 S01E01 — 一個很長很長的集名標題」，原因用「影片所在的資料夾寫不進字幕」。
    - **合格：** 片名那格 390 ≥200px、320 ≥150px；「和平使者 S01E01」兩個寬度都整個看得到；長片名最多兩行、結尾「…」。
    - 狀態字中心與片名第一行中心的差 ≤1px。
    - 一行的列高不變（桌機 82px、手機 78px）。
    - 不合格 → 停下來回報 Sally，不合併。
23. **視覺基準：** 有失敗列的對話框夾具（至少 `generation-batch-dialog-v2/complete-with-failures`）會變：`-darwin` 本機更新（刪舊 darwin 後 `AI_PROVIDER=claude pnpm test:visual:update-missing`）、`-linux` **`git rm`** 交給 CI bootstrap。其他對話框夾具（沒有失敗列）不應變——變了要查。**絕不跑 `test:visual:update`、不 commit 本機產的 `-linux.png`。**
24. **UX 驗收：** 390 與 880 截圖對 F8-M-v2 `H717g`、F8-D-v2 `i9Nun1`、F8-SPEC-ROW（`f8-spec-row.png`）；兩行位置、狀態字對齊、顏色逐項比。

## Tasks / Subtasks

- [ ] **T0 解開條件（AC #0）**：確認 Sally 稿 PR 已合併、三張截圖與 `pen-node-ids.json` 已更新、裁定沒變。
- [ ] **T1 文字搬家（AC #1）**：`runFailureCopy.ts`；`generationEventCopy.ts` 轉出；既有 `generationEventCopy.spec.ts`／`recentEventCopy.spec.ts` 零變化地綠。
- [ ] **T2 左欄佇列（AC #2–#6）**：`generationQueueRow.ts`、`GenerationWorkspaceV2.tsx`（`QueueRow` 兩處：`:293` 傳代碼、`:319` 片名兩行）、`GenerationBatchDialogV2.tsx:373`、`generationEventCopy.ts:225-228`。
- [ ] **T3 右欄即時活動（AC #7–#9）**：`generationEventCopy.ts`（`FeedRowView.detail`、`view()`、`failedView`、⑤⑤b）、`GenerationWorkspaceV2.tsx`（`FeedRowItem` 兩行排法、圖示對第一行、片名兩行）、註解。
- [ ] **T4 測試（AC #10–#13）**：三方一致性、改寫既有 `parts` 斷言、四個 spec 新增、七種故意改壞。
- [ ] **T5 量測＋UX 驗收＋基準＋收尾（AC #14–#17）**：390／320／1440 量測、對三張稿、新夾具、`complete-with-failures` darwin 更新與 linux 刪除、lint／typecheck／format／design-refs／E2E；更新 `sprint-status.yaml`。
- [ ] **T6 對話框佇列列兩行（2026-10-20 追加，AC #18–#24）**：contract v2 重新確認（Dev Notes STALE 下方）；`GenerationBatchDialogV2.tsx` `QueueRow`／`RowStageLabel`；測試與三種故意改壞；320／390／880 量測；對話框基準 darwin 更新、linux 刪除；對 F8 三張稿。

## Dev Notes

> ⚠️ STALE [@contract-v1→v2]: upstream disc-2026-10-live-activity-failure-reasons-a AC #1 bumped 2026-10-20; re-confirm against v2 before dev — see disc-2026-10-no-text-source-skip-a
>
> 上游變了什麼：`failure_reason` 多一個值 `no_text_source`（共十二個）；批次 `reason == "skipped"` 的項目現在也可能帶代碼（`no_text_source`／`asr_unconfigured`，其餘仍是 `""`）。本單只細分 `reason == "error"`，照字面不受影響，但 `FAILURE_COPY_BY_REASON` 的鍵清單／一致性測試若用 `Object.keys` 逐一比對，要確認 `no_text_source` 由誰加（`-b` 計畫在 `runFailureCopy.ts` 加）。
>
> **2026-10-20 重開時追加（Bob）：dev 要對 contract v2 重新確認，結果寫進 Completion Notes。** `-a` 已合併 #855，後端現在真的會送 `no_text_source`（單次即時事件，以及批次 `skipped` 項目）。`no_text_source` 是本單文字表沒有的新代碼，要確認：
> - 佇列與對話框：`queueRowLabel` 只對 `error` 看代碼，`skipped` 項目照舊走 `FAILURE_TEXT.skipped`「沒有可用的字幕來源」——這是合理的退回，**不要**把它畫成「生成失敗」。對話框第二行（AC #19）因此是「沒有可用的字幕來源」，狀態字暫時是紅色「失敗」；灰色「已略過」與 19 字原因是 `disc-2026-10-no-text-source-skip-b` 的事。
> - 即時活動單次列：`failureCopyForReason('no_text_source')` 在 `-b` 加條目前會退回「生成失敗」（跟今天一樣），可接受；但要寫一條測試釘住「不認得的代碼 → 退回，不會壞」。
> - 三方一致測試用 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一比，**本單不加 `no_text_source`**（`-b` AC #1 加），測試鍵數維持十一。

### 不要做的事

- 不要改後端、不要改 `useGenerationBatchProgress`／`useGenerationJobsFeed` 的資料流。
- 不要在佇列另寫一份「代碼 → 文字」表或英文字串比對；只能經 `failureCopyForReason`。
- 不要讓 `stopped` 影響佇列失敗列顏色（真正的預算／額度停止是 `paused`）。
- 不要給單次生成的「進行中任務」加失敗列（那是新行為，要設計稿）。
- 不要改 `announce`／`srState` 字串；截斷只是視覺，螢幕閱讀器仍唸全名與原因。
- 不要動 ⑦⑧⑩⑪ 批次層列的排法；不要改「排隊中」等其他佇列說明行的顏色（Sally 稿說不在範圍）。
- 不要在前端拆片名字串去「硬保留集數」（Sally 裁定 2 明確不選）。
- 不要用 `text-[Npx]` 任意字級；用 `text-sm` 等 token class。
- 不要跑 `test:visual:update`，不要產或 commit `-linux.png`。

### 預設做法（要改請 Alexyu／Sally 裁定）

- ⚠️ 2026-10-20：下面這條的「只改字，不改版面」被「2026-10-20 追加」取代（對話框也改兩行）。
- **產生字幕對話框（F8）的佇列也一起講具體原因**（只改字，不改版面、不改片名截斷）。兩畫面共用 `queueRowLabel` 是刻意的（`generationQueueRow.ts:94-98`、`GenerationWorkspaceV2.tsx:20-23`，dsr-6d-c-1 建單裁定 #2「同一部片不能講兩種話」）。代價：對話框手機標籤變長、片名變短（估約 90px／6 字，AC #14 實測）。Sally 稿沒畫對話框；若 Alexyu 只要工作區，改成只在工作區傳 `failureReason`。對話框片名要不要也兩行，另找 Sally。
- **極少見的 `error`＋`budget_exceeded`／`credit_exhausted`**：佇列與即時活動批次列都是紅色「失敗」＋「已達預算上限」／「AI 帳號沒有額度」，不改黃色。

### 實作提示（不是規定）

- `line-clamp-2` 是 Tailwind 內建（`display:-webkit-box`），配 `min-w-0`／`break-words` 才能在 flex 裡折行；中文自然可斷，英文長字串靠空白斷。
- 「圖示對第一行置中」：`li` 改 `items-start`，圖示外包一個高度等於第一行行高（`text-sm` × `leading`）的 `flex items-center` 盒子，比算 `mt-[Npx]` 穩。量測時用 `getBoundingClientRect` 比圖示中心與階段字中心。
- 第一行裡階段字與片名可能是一行對兩行：階段字、`·` 對齊片名**第一行**（`items-start` 或 `items-baseline`），不要被兩行片名撐到中間。

### 測試標準

- Vitest＋Testing Library；斷言用可見文字、testid、`data-state`、token class（Rule 16）。版面寬度只能在真瀏覽器量（AC #14），jsdom 只斷言結構與 class。不要背景跑測試。

### E2E data-testid

- 既有 testid 全部不變（`workspace-queue-row-*`、`gen-batch-row-*`、`workspace-feed-row`）。可新增 `workspace-feed-row-detail`。E2E 只斷言 `busy_elsewhere`／`skipped` 文字與即時活動列數 → 不受影響（AC #17 仍跑一次）。

### 視覺基準

- 會變：`generation-workspace-v2/complete-with-failures`（darwin 本機更新、linux 刪掉交給 CI bootstrap）；新增：`generation-workspace-v2/failures-long-titles`（darwin 本機、linux 交給 CI）。其餘四張不應變。見 AC #16。

### Project Structure Notes

- 新增：`apps/web/src/components/subtitle/runFailureCopy.ts`。
- 動到：`generationQueueRow.ts`（＋spec）、`generationEventCopy.ts`（＋spec）、`GenerationWorkspaceV2.tsx`（＋spec）、`GenerationBatchDialogV2.tsx`（＋spec）、`apps/web/src/hooks/useGenerationBatchProgress.spec.ts`、`apps/web/src/services/subtitleService.ts`（只改 `failureReason` 註解，補「佇列也讀」）、`apps/web/src/routes/test/-gallery.fixtures.tsx`（新夾具）、`tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/`（`complete-with-failures` darwin 更新／linux 刪除、新夾具 darwin）、`_bmad-output/implementation-artifacts/sprint-status.yaml`。
- 不動 `ux-design.pen`、`_bmad-output/screenshots/`（那是 Sally 稿 PR 的事）。

### References

- [Source: scratchpad `sally-live-row/disc-2026-10-live-activity-row-layout-design.md`（合併後 `_bmad-output/implementation-artifacts/` 同名檔）——裁定 1–8、「實作單要做的事」]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-b.md Completion Notes（AC #9 量測、範圍外回報）]
- [Source: apps/web/src/components/subtitle/generationQueueRow.ts:25-28, 48-54, 69-92, 94-142]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:1-19, 21, 26-93, 103-152, 154-315]
- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:20-23, 285-371, 383-420, 916-970]
- [Source: apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx:163-192, 345-394, 505-520, 750-795]
- [Source: apps/web/src/services/subtitleService.ts:155-184]
- [Source: apps/web/src/hooks/useGenerationBatchProgress.ts:130-152, 287]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:96-109, 427-456]
- [Source: apps/api/internal/services/generation_batch.go:144-156, 736-818, 846-930]
- [Source: apps/api/internal/models/subtitle_run_failure.go:14-26, 53-77, 143-150]
- [Source: generationEventCopy.spec.ts:89-176, 211-240, 317-357；generationQueueRow.spec.ts:65-71；GenerationWorkspaceV2.spec.tsx:293-309, 593, 656-740；GenerationBatchDialogV2.spec.tsx:1355, 1424, 2019；useGenerationBatchProgress.spec.ts:303-324]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:475-479, 6895-6910, 7097, 7158, 7217, 7272-7320, 7322]
- [Source: tests/e2e/generation-workspace.spec.ts:42-44, 62-63；tests/e2e/batch-subtitle.spec.ts:254-256]
- [Source: project-context.md Rule 16, Rule 21]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-19 | 建單（Bob）。 |
| 2026-10-20 | `-a`（#855）stale-mark：契約升 `[@contract-v2]`。 |
| 2026-10-20 | Bob：重開（PR #851）。新增「2026-10-20 追加：對話框佇列列改兩行（Alexyu 選 B，Sally 裁定）」AC #18–#24、T6；STALE 下方加 contract v2 重確認說明；AC #6、#14 對話框部分標為被取代；Status → in-progress。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
