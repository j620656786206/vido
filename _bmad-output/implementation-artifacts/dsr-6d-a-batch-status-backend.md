# Story DSR.6d-a：批次生成的後端把「整個佇列」和「最後的結果」記住——離開頁面再回來看得到，失敗不會被當成完成

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who starts a subtitle batch and then walks away from the page,
I want 回來的時候還看得到整個佇列、哪幾部做完、哪幾部失敗、哪幾部因為預算上限停下來，
so that 我不用一直開著分頁盯著；碰到上限停下的 26 部，我回來也知道要按「下次繼續」。

## Context

`dsr-6d`（批次生成＋生成工作區對齊）建單時，三個唯讀稽核代理發現：**畫面上很多錯誤的根源在後端沒有給資料**。前端現在只能從「成功數＋失敗數＋逐部 SSE」去猜每一列的狀態，猜錯的地方都是使用者看得到的 bug。⚖️ Alexyu 2026-09-17 裁定：**先做一張小的後端單**，再做兩張前端對齊。`dsr-6d` 因此拆成三張：

| 單子 | 範圍 |
| --- | --- |
| **`dsr-6d-a`（本張）** | 後端：批次進度帶**每一部的狀態**、狀態查詢帶**最後一次結果**、開始回應帶**當下進度**、`error` 狀態真的會送出、分集帶**劇名**；前端只改型別 |
| `dsr-6d-b` | 前端：批次對話框 F8-D-v2／F9-D-v2 對齊＋對話框的 bug（依賴本張） |
| `dsr-6d-c` | 前端：生成工作區 F11–F13 對齊＋`Component/GenQueueRow-v2`＋工作區的 bug（依賴本張） |

本張**不動任何畫面**。

### 🔴 建單時查到的事（main `6ad820de`）

1. **每一部的狀態只能用猜的，猜錯就把失敗顯示成「完成」。** 批次 SSE `generation_batch_progress` 只有計數（`generation_batch.go:547-567`）。前端從 `success_count`／`fail_count` 與逐部 `transcription_*`／`subtitle_progress` 事件推每一列。三種情況一定猜錯：
   - 管線模式下 `TryReserve` 被拒（這部片正在別處處理）直接回錯，**不送任何逐部事件**（`cmd/api/generation_batch_runner_adapter.go:77-81`）→ `fail_count` 加一，但那一列顯示「完成」，「重試失敗項目」也不出現。
   - D6 在前端看到任何階段前就失敗或略過（`useGenerationProgress.ts:290` 只在看過階段後才算數）。
   - 第一部的事件在前端連上 SSE 之前就送出了（`generation_batch.go:463` 緊接在 `go p.process` 之後，hub 沒有重播）。
2. **狀態查詢在批次結束後什麼都不給。** `finish()` 把 `activeBatch` 清成 nil（`:533-539`），之後 `GET …/status` 回 `{running:false, progress:null}`（handler `:204-210`）。結束的那一個 SSE 事件如果掉了（hub 頻道滿時直接丟棄，`sse/hub.go:164-166`；或 10 秒重連空窗），前端永遠卡在「進行中」；離開頁面再回來，看到「目前沒有進行中的生成」，完全不知道 26 部卡在預算上限。這就是 `disc-2026-07-generation-batch-status-items`（backlog，2026-07-06 立，ux3-ai-1 已提高優先序）。
3. **中途接上只看得到正在跑的那一部。** 狀態查詢與 409 都沒有 `items[]`，對話框與工作區只能畫「進行中」一張卡片（F13 ② 的降級稿與它的註記就是在說這件事）。
4. **`error` 狀態從來沒送過。** `GenerationBatchStatusError` 只宣告（`:57`），沒有任何 `finish()` 用它；背景 goroutine 沒有 `recover()`——`process` 裡 panic 會讓整個 API 程序掛掉，前端的錯誤橫幅（`GenerationBatchDialogV2.tsx:361-374`）永遠到不了。
5. **開始的回應沒有進度。** 202 只有 `{batch_id,total_items,items}`（handler `:187-194`），前端只能用 `batchId`／`totalItems` 起頭（`GenerationBatchDialogV2.tsx:718-721`），所以第一部跑完前「本次用量 $0.00 / 上限 $0.00」。
6. **分集沒有劇名。** 佇列項目的標題是「S04E07 第七章」（`toEpisodeItem`，`:418-421`），沒有劇名；設計稿 F8 畫的是「怪奇物語 S4E7」。對話框與工作區的列上，連看好幾集時分不出是哪部劇。
7. **失敗原因分不出來。** `fail_count` 同時包含「管線略過（沒有可用的字幕來源）」「這部正在別處處理」「真的出錯」（`:485-503`）。計數語意不改（sub-4-2 CR H1：成功必須代表字幕真的產生了），但每一部要帶原因，前端才能寫出不騙人的字。

### 範圍外（另立單，建單時已寫入 sprint-status）

- 活動頁批次那一列的「3 / 10」用的是**正在跑的第幾部**（`ActivityProgress` 回 `CurrentIndex`，`:195-204`），對話框是「已完成 2 / 10」——`disc-2026-09-activity-batch-row-count-in-flight-index`。
- 每一部的花費（`subtitle_runs.spent_usd` 已存但沒有端點）——既有 `9R-17-ai-usage-endpoint`。
- 預算碰頂時已翻譯的段落被丟掉——既有 `backlog-translate-budget-partial-progress`。

---

## Acceptance Criteria

1. **每一部的狀態 [@contract-v1]（加在 9R-16 AC #2／#9 的形狀上，不升版，見 AC #10）。**
   - 型別：`GenerationBatchItemState` 嵌入 `GenerationBatchItem`（`media_id`、`title`、`media_type`、`series_title`）＋ `status`、`reason`。JSON `{"media_id","title","media_type","series_title","status","reason"}`，**每個鍵都一定出現**（`series_title`／`reason` 沒有值時是 `""`，不用 `omitempty`）。
   - **獨立的常數型別**，不要跟批次狀態常數（`GenerationBatchStatusRunning` 等，也用了 `"running"`／`"cancelled"`）混：`type GenerationBatchItemStatus string`，值 `queued`｜`running`｜`done`｜`failed`｜`paused`｜`cancelled`；`type GenerationBatchItemReason string`，值 `""`｜`skipped`｜`busy_elsewhere`｜`error`。
   - `reason` 只在 `failed` 時有值：
     - `skipped` ← `ErrGenerationItemSkipped`（管線刻意略過）**以及** `ErrTranscriptionDisabled`（legacy 模式同一類：沒有可用的來源／金鑰，`generation_batch.go:41-49` 與 sub-4-2 CR H1 視為同一類；管線模式同一情況本來就走 skip，`process_item.go:543-544`）
     - `busy_elsewhere` ← `ErrTranscriptionInProgress`（含 `TryReserve` 被拒，`cmd/api/generation_batch_runner_adapter.go:77-81` 已用 `%w` 包好）
     - `error` ← 其他所有錯誤，含 AC #4 的 panic
   - **佇列在 `Start` 就填好**：在建立 `activeBatch` 的同一段臨界區（`:299-307`）把 `Items` 填成全部 `queued`。`process` 只做之後的轉換。**處理器產生的任何快照裡 `Items` 都不是 nil**（否則 AC #5 的 202 可能拿到空佇列）。
   - **轉換與計數在同一段臨界區更新**（現在計數在 `:454-461`、`:505-510` 分開寫），用一個 `defer p.mu.Unlock()` 的小 helper，避免 AC #4 的 panic 發生在持鎖時：
     - 輪到第 i 部、廣播之前 → `running`。
     - 成功 → `done`；失敗 → `failed`＋原因（同時更新 `success_count`／`fail_count`）。
   - **終態轉換由 `finish` 統一做，不依賴傳進來的 index**（開頭預檢傳 `i`、跑到一半傳 `i+1`，`:441, :450, :475, :483`）：
     - `budget_ceiling`：所有仍是 `queued`／`running` 的 → `paused`。
     - `cancelled`：所有仍是 `queued`／`running` 的 → `cancelled`。
     - `error`：`running` → `failed`＋`reason: error`（並計入 `fail_count`），`queued` → `cancelled`。
     - `complete`：此時不應有 `queued`／`running`；若有，記 `logger.Error` 並照 `cancelled` 處理（不准留下 `running` 的快照）。
   - **每個終態都成立的不變式**（每個終態測試都要斷言）：`success_count` ＝ `done` 的數量、`fail_count` ＝ `failed` 的數量、`paused_count` ＝ `paused` 的數量、`len(items)` ＝ `total_items`。
   - **不准共用切片**：對外的快照（status、409、202、`last`、SSE）一律**深複製** `items`。`ActivityProgress()` 不需要 `items`，改用不複製佇列的內部讀取（它每次 `/activity` 輪詢都會呼叫）。
   - `success_count`、`fail_count`、`paused_count`、`current_*` 的語意**不變**。

2. **SSE：執行中只送變動的那一部，終態送整個佇列。** 一次選 2,400 部是有記錄的情境（`CandidateListPanel.tsx:703-707, 757` 的全選，handler 沒有 `media_ids` 上限）：每次廣播都帶整個佇列，一部兩次廣播、每次約 400 KB，hub 丟事件時還會把整包寫進 log（`sse/hub.go:165-166`）。所以：
   - **保留手寫的 map**（`broadcast`，`:552-566`），`spent_usd`／`budget_usd` 照舊取 `budget.Snapshot()`——⛔ **不要改成序列化 struct**：`generation_batch_test.go:182` 把 `ev.Data` 斷言成 `map[string]interface{}`，改了會讓所有 `eventsUntilTerminal` 逾時；而且 `config.go:309` 接受 `"NaN"`，`activeBatch.BudgetUSD` 可能是 NaN，JSON 編碼會失敗。
   - 執行中的廣播（`status: running`）加兩個鍵：`"items": null`、`"changed_item": <這次狀態改變的那一部，GenerationBatchItemState>`。
   - 終態廣播加：`"items": <整個佇列，深複製>`、`"changed_item": null`。
   - 所以 SSE `data` 的鍵集合＝原本 11 個＋`items`＋`changed_item`，**每次都有這 13 個鍵**。
   - 前端（`dsr-6d-b`／`-c`）用 202／status／409 拿到的完整佇列，套用 `changed_item`；事件掉了就重新查 status（那兩張單負責）。

3. **最後一次結果 [@contract-v1]。** `GET /api/v1/subtitles/generation-batch/status` 從 `{running, progress}` 變成 `{running, progress, last}`：
   - `running`、`progress` 的語意**完全不變**（沒有在跑時 `progress` 仍是 `null`；在跑時 `progress.items` 是完整佇列）。
   - `last`：最近一次**已結束**批次的終態快照（同一個 `GenerationBatchProgress` 形狀，含完整 `items`、結束當下的 `spent_usd`／`budget_usd`、終態 `status`）；沒有時 `null`。
   - **一次上鎖取兩個值**：新增 `Snapshot() (progress, last *GenerationBatchProgress)`，handler 只呼叫它——分兩次呼叫的話，批次可能剛好在兩次之間結束，回出 `{running:true, progress, last}`。
   - 生命週期：`finish()` 在**清掉 `activeBatch` 的同一段臨界區**寫入 `last`（否則 status 會在「已清掉、還沒寫 last」的瞬間三個都是空）；`Start` 在第二段臨界區（`:276-307`，確定真的開始之後）清掉；AC #6 的 dismiss 清掉；API 重啟就沒了（純記憶體，不寫資料庫——本張不做歷史）。正在跑時 `last` 一定是 `null`。
   - `finish` 要**防重入**：`activeBatch` 為 nil 或 `BatchID` 不同時直接返回（AC #4 的 recover 可能在 `finish` 或 `broadcast` 自己 panic 之後再進來）。

4. **`error` 終態真的會送出 [@contract-v1]。**
   - `process` 的**第一行**就是 `defer` 一個 `recover()`：抓到 panic → `p.logger.Error`（帶 `service=generation_batch` 的那個 logger，`:162`；含 `batch_id`、`panic`、`stack`＝`runtime/debug.Stack()`）→ `finish(batchID, GenerationBatchStatusError, …)`。
   - 計數與佇列都從 `activeBatch` 讀（AC #1 已把它們放在同一個地方），不必另外傳閉包變數。
   - **所有持鎖的區段都用 `defer p.mu.Unlock()`**——如果 panic 發生在持鎖時而鎖沒釋放，recover 呼叫 `finish` 會永遠卡住，`IsRunning()` 一直是 true，之後每次開始都回 409，比原本直接掛掉還糟。
   - 測試：假 runner 在第 2 部 `panic` → 程序不掛、收到 `status: error` 的廣播、`last.status == "error"`、第 2 部 `failed/error`、第 3 部以後 `cancelled`、`fail_count` 含第 2 部、AC #1 不變式成立、`IsRunning()` 為 false、可以再 `Start` 新批次。
   - **寫進 Dev Notes 的限制**：runner 自己另開的 goroutine 裡的 panic 抓不到；被 recover 的那一部，它的 `subtitle_runs` 與媒體列可能停在 `running`／`extracting`，與當機後相同，目前沒有清理機制（不在本張）。
   - ⛔ 一般的逐部錯誤**不要**變成 `error` 終態——照舊計入 `fail_count` 並繼續（9R-16 AC 5）。

5. **開始的回應帶當下進度 [@contract-v1]。** 202 的 `data` 從 `{batch_id,total_items,items}` 變成 `{batch_id,total_items,items,progress}`：
   - 新增 `SnapshotFor(batchID string) *GenerationBatchProgress`（一次上鎖）：`activeBatch` 的 `BatchID` 相符就回它；否則 `last` 的 `BatchID` 相符就回 `last`（極短批次在 `Start` 回傳前已經結束）；都不相符回 nil。handler 用 `Start` 回傳的 `batchID` 呼叫它。
   - nil 時 `progress: null`——只會發生在「這個批次在幾毫秒內就被 dismiss 或被另一個批次取代」，在 Swagger 與型別註解寫明。
   - 空 scope 的 200（`total_items:0`）不加 `progress`。409 的 `data` 形狀不變（仍是 progress，現在含完整 `items`）。
   - **前端 `startGenerationBatch` 要把它帶出來**：`subtitleService.ts:537-547` 目前逐欄組 `result`，只加型別的話 `progress` 永遠是 `undefined`——202 分支要加上 `progress: data.progress ?? null`。

6. **清掉最後一次結果：`POST /api/v1/subtitles/generation-batch/dismiss` [@contract-v1]。**
   - 處理器新增 `DismissLast() (dismissed, running bool)`（一次上鎖）：正在跑 → `(false, true)`，什麼都不清；沒在跑 → 清掉 `last`，回 `(原本有沒有 last, false)`。冪等。
   - handler 200 `{"dismissed":<bool>,"running":<bool>}`；路由掛在既有群組（`generation_batch_handler.go:52-58`）；Swagger 註解比照其他端點；`GenerationBatchProcessorInterface` 加 `Snapshot`、`SnapshotFor`、`DismissLast`（`GetProgress` 保留給 409 分支與 `ActivityProgress`）。
   - 用途：工作區 F12 的「關閉」（`dsr-6d-c`）。

7. **分集帶劇名 [@contract-v1]。** `GenerationBatchItem` 新增 `series_title`（JSON 一定出現，電影為 `""`）：
   - **用 setter，不改建構子**：`SetSeriesTitleResolver(r CandidateSeriesTitleResolver)`——介面已經存在同一個 package（`generation_candidates.go:105-111`，方法完全相同），setter 比照 `SetSSEHub`／`SetModelCatalog`（`:493-498`）；`main.go` 在 `:998-999` 之後呼叫 `generationBatchProcessor.SetSeriesTitleResolver(repos.Series)`（`repos.Series` 在該處可用，`:1013` 已在用）。建構子簽名不動，`route_c_uuid_integration_test.go:131` 等呼叫點不用改。
   - `toEpisodeItem` 用 `episode.SeriesID` 查劇名，填 `series.Title`。`SeriesID` 為空或 resolver 為 nil → 直接 `""`。查不到或查詢失敗 → `p.logger.Warn`＋`""`，**不准**讓批次開始失敗（比照 `resolveSeriesMeta` 的降級，`generation_candidates.go:1101-1114`；註解寫明這是 Rule 13 第 3 種「有理由地降級」）。
   - **同一批次內依 `SeriesID` 快取**（必做）：2,400 集的全選在 202 回傳前會打 2,400 次查詢。測試證明同一部劇只查一次，而且快取不跨批次。
   - `title` 本身**不改**（仍是「S04E07 第七章」），前端自己組「怪奇物語 S04E07 第七章」。

8. **前端型別與服務（不改任何元件）。** `apps/web/src/services/subtitleService.ts`：
   - `GenerationBatchItem` 加 `seriesTitle: string`。
   - 新增 `GenerationBatchItemStatus`、`GenerationBatchItemReason`（`'' | 'skipped' | 'busy_elsewhere' | 'error'`）、`GenerationBatchItemState extends GenerationBatchItem { status; reason }`。
   - `GenerationBatchProgress` 加 `items: GenerationBatchItemState[] | null`（SSE 執行中是 `null`）；把「mirrors the SSE payload (11 keys)」改成現況（SSE 另有 `changedItem`，只在 SSE 出現——在註解說清楚，不放進這個型別，或另開 `GenerationBatchProgressEvent` 型別；擇一，說清楚即可）。
   - `GenerationBatchStatusResponse` 加 `last?: GenerationBatchProgress | null`；改寫 `:397-401` 的過期註解。
   - `GenerationBatchStartResult` 加 `progress: GenerationBatchProgress | null`，`startGenerationBatch` 的 202 分支帶出來（AC #5）。
   - 新增 `dismissGenerationBatch(): Promise<{ dismissed: boolean; running: boolean }>`。
   - `subtitleService.spec.ts` 補：status 的 `last` 與 `items` 轉成 camelCase（`series_title` → `seriesTitle`）、202 帶 `progress` → `result.progress.items[0].seriesTitle`、dismiss 打對路徑與方法。
   - `pnpm nx run web:typecheck` 用的 `tsconfig.app.json` 不含 spec；**不需要**去補夾具或 spec 裡的假資料（gallery `props` 是 `Record<string, unknown>`）。若 `pnpm nx test web` 因型別以外的原因失敗才處理。

9. **過期註解與文件。**
   - 要改寫的註解：`sse/hub.go:34-40`（列 11 個鍵、寫 `[@contract-v1]`，實際是 v2，現在又多兩個鍵）、`generation_batch.go:61-62`（status 形狀）、`:77-80`（items 形狀）、`:399-400`（「no series join」）、`:533-536`（「activeBatch is cleared (not status-stamped) at terminal」）、handler Swagger `@Success 202`（`:88`）與 status 的 `@Success`（`:202`）。
   - `docs/swagger.{json,yaml}`、`docs.go` 已經多年沒更新、也沒有 generation-batch——⛔ **不要跑 `swag init`**。`docs/` 沒有這個 API 的文件，Rule 17 不適用；在 Completion Notes 寫明。

10. **Rule 20 契約紀錄（Dev Agent Record 必填）。** 本張新增的形狀（AC #1、#2、#3、#4、#5、#6、#7）標 `[@contract-v1]`，因為 `dsr-6d-b`／`dsr-6d-c` 會讀它們。9R-16 AC #1 [@contract-v3]、#2／#7／#9 [@contract-v2] **不升版**：只加鍵、既有鍵的意思不變（先例：sub-5-1 AC #7、sub-6-8a AC #4、retro-19-P3）。對比：sub-4-2 當年替 AC #1 升版，是因為它同時改了意思（接受分集、換引擎）。既有的下游（`ux3-subtitle-v2-batch`、`ux3-ai-2-workspace-frontend`、`sub-4-3`）都已 done，不需要 stale-mark。在 Completion Notes 寫一行 `📎 Contract Stamps:` 記錄以上。

11. **測試（Go，先寫紅測試）。** `generation_batch_test.go`（沿用 `newTestGenerationProcessor` 與 `sse.Client` 收廣播，`:204-227`）＋ `generation_batch_handler_test.go`（唯一要改的假物件是 `mockGenerationProcessor`）：
    - 狀態：全部成功；`ErrGenerationItemSkipped` 與 `ErrTranscriptionDisabled` → `failed/skipped`；`ErrTranscriptionInProgress` → `failed/busy_elsewhere`；其他 → `failed/error`；成功失敗交錯時每次 `changed_item` 正確。
    - 預算：開頭預檢碰頂、中途 `ErrBudgetExceeded` → `queued`／`running` 全 `paused`。取消：開頭檢查與跑到一半 → 全 `cancelled`、已完成的不動。
    - **每個終態**都斷言 AC #1 的不變式。
    - `Start` 回傳後立刻 `SnapshotFor(batchID)` → `items` 長度＝`total_items`、全是 `queued` 或第一部 `running`（不會是 nil）。
    - `last`：結束後 `Snapshot()` 的 `last` 是終態快照；新批次開始後為 nil；`DismissLast` 的回傳值、冪等、跑的時候不清；`SnapshotFor` 對已結束的批次回 `last`、對別的 id 回 nil。
    - AC #4 panic 測試。
    - 劇名：分集帶 `series_title`；同一部劇多集只查一次；快取不跨批次；查詢失敗 → `""` 且批次照常開始；nil resolver、空 `SeriesID`、電影 → `""`。
    - SSE：執行中廣播的鍵集合＝13 個、`items` 為 nil、`changed_item` 不為 nil；終態廣播 `items` 完整、`changed_item` 為 nil。
    - handler：status 回應完整鍵集合 `{running, progress, last}`；202 鍵集合 `{batch_id,total_items,items,progress}` 且 `progress.items` 長度＝`total_items`；dismiss 兩種回應；409 的 `data.items` 存在。既有逐鍵比對的測試改成新形狀——**只加鍵，不刪鍵**。
    - **`-race`**：CI 不跑 race（`test.yml:193`），dev 要在本機跑 `go test -race ./internal/services/... ./internal/handlers/...`，把結尾輸出貼進 Completion Notes（建單驗證時兩個 package 都是乾淨的，services 約 39 秒）。

12. **CI 全綠**：`pnpm nx test api`、`gofmt`／`go vet`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`pnpm nx test web`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 卡住（建專案圖逾時）時用 `NX_DAEMON=false` 重跑，並在 Completion Notes 記下。

## Tasks / Subtasks

- [x] **Task 1 — 每一部的狀態與終態轉換（AC: #1, #4, #11）**
  - [x] 先寫紅測試：各種結束方式的狀態與原因、每個終態的不變式、`Start` 後立刻取快照不是 nil、panic → `error`
  - [x] 常數型別、`GenerationBatchItemState`、`Start` 填佇列、持鎖 helper、`finish` 統一轉換＋防重入、`recover()`、深複製、`ActivityProgress` 不複製佇列
- [x] **Task 2 — SSE 變動項目（AC: #2, #11）**
  - [x] 先寫紅測試：執行中 13 鍵＋`changed_item`、終態完整 `items`
  - [x] `broadcast` 手寫 map 補鍵
- [x] **Task 3 — 最後一次結果、開始回應、dismiss（AC: #3, #5, #6, #11）**
  - [x] 先寫紅測試：`Snapshot`、`SnapshotFor`、`DismissLast`、handler 鍵集合
  - [x] 處理器方法、handler status／202／dismiss 路由、Swagger 註解
- [x] **Task 4 — 分集劇名（AC: #7, #11）**
  - [x] 先寫紅測試：劇名、同劇只查一次、不跨批次、降級
  - [x] `SetSeriesTitleResolver`、批次內快取、`main.go` 接線
- [x] **Task 5 — 前端型別與服務（AC: #8）**
  - [x] 型別、`startGenerationBatch` 帶出 `progress`、`dismissGenerationBatch`＋spec
- [x] **Task 6 — 註解、契約紀錄與收尾（AC: #9, #10, #12）**
  - [x] 過期註解；`📎 Contract Stamps` 紀錄；本機 `-race` 輸出；全套閘門

## Dev Notes

### 這張的重點

- **把真相放在後端**：前端現在用「計數＋逐部事件」去猜每一列，猜錯的都是真的 bug（失敗顯示成完成、重試按鈕不出現、回來看不到上限停下）。後端本來就知道每一部怎麼結束的，只是沒說。
- **全部是加欄位、加一個端點**，既有鍵的意思一個都不改（AC #10）。前端舊程式不讀新欄位也照常運作（`dsr-6d-b`／`dsr-6d-c` 才開始讀）。
- **量要算清楚**：全選 2,400 部是真實情境，所以 SSE 執行中只送變動的那一部（AC #2），劇名查詢要快取（AC #7），`/activity` 輪詢不要複製佇列（AC #1）。
- **純記憶體**：`last` 重啟就沒了，這是刻意的（本張不做歷史）。

### 建單裁定（2026-09-17）

1. ⚖️ **Alexyu**：離開頁面再回來要看得到整個佇列與「碰到上限停下」的結果 → 先做本張後端單（Q2 選 B）。
2. ⚖️ **Alexyu**：碰到上限時，批次層的提示用赭色、每一列與數字中性（Q1 選 A）——本張不碰畫面，記在 `dsr-6d-b`／`dsr-6d-c`。
3. SM：`取消`與 `error` 之後沒跑到的項目標 `cancelled`（不是 `queued`，否則回來看的快照會永遠寫「排隊中」）；`error` 當下那一部標 `failed/error`。
4. SM：`reason` 只分三類（`skipped`／`busy_elsewhere`／`error`；`skipped` 同時涵蓋管線略過與 legacy 的未設定，兩種模式同一個名字），前端的字由 Sally 在 `dsr-6d-b`／`-c` 定。
6. SM（建單驗證後）：SSE 執行中只送 `changed_item`、終態與查詢才送完整佇列——全選 2,400 部時每次廣播帶整個佇列約 400 KB，不可行。這是技術取捨，不影響使用者看到的東西。
5. SM：dismiss 用獨立端點，不在 status GET 裡順手清（GET 不能有副作用）。

### 不要做的事

- **不要改任何元件或畫面**（對話框、工作區、夾具、基準線）——`dsr-6d-b`／`dsr-6d-c`。
- **不要改 `success_count`／`fail_count`／`paused_count`／`current_index` 的意思**；活動頁的「3 / 10」另有單子。
- **不要把 `last` 寫進資料庫**，也不要加「歷史」。
- **不要在劇名查詢失敗時讓批次開始失敗**。
- **不要改 SSE hub 的丟棄策略**——`last`＋前端重新查詢（`dsr-6d-b`／`-c`）就是補救路徑。
- **不要改 `transcription_service.go` 的英文錯誤字串**（工作區紀錄顯示它的問題記在 `dsr-6d-c`）。

### 已知陷阱

- **`sse.Hub` 的註冊是非同步的**：測試要等 `hub.ClientCount() == 1` 再開始批次（`generation_batch_test.go:213-217` 已有註解）。
- **`GetProgress` 目前是淺複製**（`prog := *p.activeBatch`，`:187`）——加了 `items` 切片之後淺複製會共用底層陣列，`-race` 一定會抓到。
- **recover 與鎖**：panic 發生在持鎖時、鎖又沒有 `defer` 釋放，recover 裡的 `finish` 會永遠卡住（AC #4）。
- **`sse.Client` 收到的 `ev.Data` 是 `map[string]interface{}`**（`generation_batch_test.go:182` 斷言它）——`changed_item`／`items` 放進 map 時是 Go struct 值，測試要用型別斷言或轉 JSON 再比，不要以為它是巢狀 map。
- **`finish` 先清 `activeBatch` 再廣播**（`:537-541`）；`last` 要在同一段臨界區內寫入，否則 status 查詢會在「已清掉、還沒寫 last」的瞬間回 `{running:false, progress:null, last:null}`。
- **Start 的兩段鎖**（`:259-280`）：清掉 `last` 要放在第二段（確定真的開始之後），不然 409 或空 scope 會誤清。
- **handler 介面變動**會讓假 processor 編譯失敗——唯一要改的是 `generation_batch_handler_test.go` 的 `mockGenerationProcessor`。
- **`models.Series` 的標題欄位是 `Title string`**（`models/series.go:31`）；`SeriesRepository.FindByID` 查不到回包著 `sql.ErrNoRows` 的錯（`series_repository.go:110-111`）。
- **不要跑 `swag init`**（AC #9）。
- **行號以建單時為準**（2026-09-17，main `6ad820de`）。

### Source tree

```
apps/api/internal/services/generation_batch.go(+_test)             ← Task 1–4
apps/api/internal/handlers/generation_batch_handler.go(+_test)     ← Task 2, 3
apps/api/cmd/api/main.go                                           ← Task 4（SetSeriesTitleResolver(repos.Series)）
apps/api/internal/services/generation_candidates.go                ← 只讀（CandidateSeriesTitleResolver、resolveSeriesMeta 先例）
apps/api/internal/sse/hub.go                                       ← Task 6（只改 :34-40 註解）
apps/api/cmd/api/generation_batch_runner_adapter.go                ← 只讀（busy_elsewhere 的來源）
apps/web/src/services/subtitleService.ts(+spec)                    ← Task 5
```

### Cross-Stack Split Check

後端 task **4 個**（Task 1–4），前端 task **1 個**（Task 5，型別與服務）。前端 ≤ 3 → **本張不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 本張不改任何 `components/**`；後端也不加時間欄位（沒有 `started_at`／`finished_at`，dsr-6b 已裁定不顯示系統給不出的時間）。

### References

- [Source: `apps/api/internal/services/generation_batch.go:51-75, 77-88, 181-204, 257-312, 401-429, 434-568`]
- [Source: `apps/api/internal/handlers/generation_batch_handler.go:15-25, 51-58, 151-210`]
- [Source: `apps/api/cmd/api/generation_batch_runner_adapter.go:60-97`、`apps/api/cmd/api/main.go:998-1000, 1013, 1026-1029`、`apps/api/internal/services/generation_candidates.go:105-111, 493-498, 1091-1114`、`apps/api/internal/subtitle/process_item.go:543-544`、`apps/api/internal/config/config.go:309`、`.github/workflows/test.yml:193`]
- [Source: `apps/api/internal/sse/hub.go:160-170`、`apps/api/internal/services/generation_batch_test.go:204-227`]
- [Source: `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:703-707, 757`、`apps/web/src/services/subtitleService.ts:141-147, 380-430, 513-581, 537-547`、`hooks/useGenerationProgress.ts:290`、`components/subtitle/GenerationBatchDialogV2.tsx:361-374, 711, 718-721`]
- [Source: `_bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md` AC #1 [@contract-v3]、#2 [@contract-v2]、#7、#9 [@contract-v2]；`sub-4-2-consent-batch-backend.md`（H1 計數語意、M2 查詢失敗傳遞）]
- [Source: `sprint-status.yaml` → `disc-2026-07-generation-batch-status-items`（本張取代）、`9R-17-ai-usage-endpoint`、`backlog-translate-budget-partial-progress`]
- [Source: project-context.md#Rule 2 / #Rule 11 / #Rule 13 / #Rule 17 / #Rule 19 / #Rule 20 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — Amelia（dev-story），2026-09-18

### Debug Log References

- 紅測試：`generation_batch_items_test.go` 先寫，編譯失敗（型別不存在）；handler 測試先寫，第一條型別斷言 panic 讓整個 package 中止（紅）。
- 突變檢查（實作後把修正拿掉再跑，確認測試會抓到）：拿掉 `Start` 裡的 `p.lastBatch = nil` → `LastLifecycle` 紅；拿掉劇名快取寫入 → `EpisodesCarryTheSeriesTitleOncePerSeries` 紅；拿掉 panic 那一部的 `FailCount++` → `PanicEndsTheBatchWithError` 紅。還原後全綠。
- 唯一需要改的既有 Go 測試：`TestGenerationBatch_SSEPayloadFields` 逐鍵比對 11 個鍵 → 13 個（只加鍵）。
- 前端 `web:typecheck` 抓到 `GenerationBatchDialogV2.tsx:423` 的 attach 降級卡片自己組了一個 `GenerationBatchItem`，新增必填欄位 `seriesTitle` 後型別不合——只補 `seriesTitle: ''`（純型別，畫面輸出不變，註解指向 dsr-6d-b）。這是本張唯一碰到的元件檔。
- `-race`（本機，AC #11 要求貼輸出）：
  ```
  $ go test -race ./internal/services/... ./internal/handlers/...
  ok  	github.com/vido/api/internal/services	40.438s
  ok  	github.com/vido/api/internal/handlers	4.385s
  ```
- Nx daemon：本張的 `lint:all`／`typecheck`／`test web`／`test api` 一律用 `NX_DAEMON=false` 跑（dsr-6c 時 daemon 建專案圖逾時，這次直接避開），全部一次通過。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 🔗 AC Drift: NONE（checked: `generation-batch`／`generation_batch_progress`／`GenerationBatchProgress` across `_bmad-output/implementation-artifacts/*.md` — 命中 9R-16、sub-4-2、sub-4-3、sub-5-3、ux3-subtitle-v2-batch、ux3-ai-2-workspace-frontend，全部 REUSE：既有鍵的意思都沒變；9R-16 AC #2「沒在跑時 progress 為 null」照舊，`last` 是新鍵）
- 📎 Contract Stamps: FOUND（上游 9R-16 AC #1 [@contract-v3]、#2／#7／#9 [@contract-v2] — **不升版**：只加鍵（SSE `items`／`changed_item`、status `last`、202 `progress`、items 的 `series_title`），既有鍵意思不變，先例 sub-5-1 AC #7、sub-6-8a AC #4、retro-19-P3；sub-4-2 當年升 AC #1 是因為同時改了意思。本張新形狀 AC #1／#2／#3／#4／#5／#6／#7 標 [@contract-v1]，下游 dsr-6d-b／dsr-6d-c 建單時要 ack。既有下游 ux3-subtitle-v2-batch／ux3-ai-2-workspace-frontend／sub-4-3 都已 done，不需要 stale-mark。）
- 🎭 A11y Pre-Flight: PASS（觸及 1 個元件檔 `GenerationBatchDialogV2.tsx`，只加一個型別欄位、畫面輸出不變；`pnpm run lint:all` 0 errors、128 warnings 與開工前相同，0 個由本張引入）
- 🎨 UX Verification: SKIPPED — no UI changes in this story（唯一的元件改動是型別欄位，渲染輸出不變）
- **AC #1 每一部的狀態**：`GenerationBatchItemStatus`／`GenerationBatchItemReason` 各自獨立型別；`GenerationBatchItemState` 嵌入 `GenerationBatchItem`；佇列在 `Start` 第二段臨界區填成全部 `queued`；`markItem` 在同一段臨界區改狀態、改計數、組 SSE payload；`finish` 統一把還在 `queued`／`running` 的轉成終態（不看傳入的 index），並在清掉 `activeBatch` 的同一段臨界區寫入 `lastBatch`；`finish` 與 `markItem` 對不符的 `batchID` 直接返回（防重入）。原因對應：`ErrGenerationItemSkipped`＋`ErrTranscriptionDisabled` → `skipped`；`ErrTranscriptionInProgress` → `busy_elsewhere`（從原本的 default 分支獨立出來，log 文案變成「already being processed elsewhere」）；其他 → `error`。每個終態測試都斷言「計數＝各狀態數量」。`ActivityProgress` 直接讀計數，不複製佇列。
- **AC #2 SSE**：保留手寫 map（`batchEventData`），執行中 `items: nil`＋`changed_item`、終態完整 `items`＋`changed_item: nil`，每次 13 個鍵。payload 在鎖內組好、鎖外 `Broadcast`。
- **AC #3／#5／#6**：`Snapshot()`、`SnapshotFor(batchID)`、`DismissLast()` 都是一次上鎖；handler status 回 `{running, progress, last}`、202 多 `progress`（`SnapshotFor(batchID)`）、新路由 `POST …/dismiss`。409 分支仍用 `GetProgress()`，現在含完整 `items`。
- **AC #4**：`process` 第一行 `defer recover()` → `p.logger.Error`（含 stack）→ `finish(error)`；所有持鎖區段改用 `withLock`（defer 解鎖）。限制照 AC 記錄：runner 另開 goroutine 的 panic 抓不到；被 recover 的那一部 `subtitle_runs`／媒體列可能停在 running，與當機後相同，沒有清理機制。
- **AC #7 劇名**：`SetSeriesTitleResolver(CandidateSeriesTitleResolver)`（沿用 `generation_candidates.go` 的既有介面）；`collectItems` 每次呼叫建一個 memo，`seriesTitle()` 查不到或失敗都記一次（含失敗結果），所以同一部劇只查一次；nil resolver／空 `SeriesID` → `""`；失敗 → `p.logger.Warn`＋`""`。`main.go` 接 `repos.Series`。建構子簽名沒動。
- **AC #8 前端**：`subtitleService.ts` 新增 `GenerationBatchItemStatus`／`Reason`／`ItemState`／`DismissResult` 型別、`GenerationBatchProgress.items`、`GenerationBatchStatusResponse.last`、`GenerationBatchStartResult.progress`（202 分支帶出來，空 scope 與舊伺服器為 `null`）、`dismissGenerationBatch()`。spec：202 帶 `progress`（含 `seriesTitle`）、status 的 `last`、dismiss；兩條既有 `toEqual` 的期望加上 `progress: null`。
- **AC #9 註解與文件**：`sse/hub.go` 事件註解（鍵、版本改正為 v2、新鍵說明）、`generation_batch.go` 的 struct／items／`toEpisodeItem`／`finish` 註解、handler Swagger（202、status、dismiss）。`docs/` 沒有這個 API 的文件，Rule 17 不適用；沒有跑 `swag init`。
- **閘門（AC #12）**：`pnpm nx test api` exit 0（35 個 package ok）；`pnpm run lint:all` exit 0（0 errors／128 warnings）；`pnpm nx run web:typecheck --skip-nx-cache` 通過；`pnpm nx test web` **267 檔／3651 條全過**；`gofmt -l`（本張檔案）乾淨、`go vet` 乾淨；`-race` 見 Debug Log。每次測試後 `pnpm run test:cleanup`。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 每一部的狀態靠猜、失敗顯示成完成（🔴 #1）→ **AC #1、#2**
  - 結束後狀態查詢什麼都不給（🔴 #2；取代 `disc-2026-07-generation-batch-status-items`）→ **AC #3、#6**
  - 中途接上沒有佇列（🔴 #3）→ **AC #1、#3**（完整 `items` 出現在 status／409）
  - `error` 從來沒送、goroutine 沒有 recover（🔴 #4）→ **AC #4**
  - 開始回應沒有進度（🔴 #5）→ **AC #5**
  - 分集沒有劇名（🔴 #6）→ **AC #7**
  - 失敗原因分不出來（🔴 #7）→ **AC #1**（`reason`）

- **② spawn-blocking-story**：無（`dsr-6d-b`／`dsr-6d-c` 依賴本張，不是本張依賴它們）。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `dsr-6d-b-batch-dialog`、`dsr-6d-c-generation-workspace` — `dsr-6d` 其餘範圍
  - `disc-2026-09-activity-batch-row-count-in-flight-index` — 活動頁「3 / 10」與對話框「已完成 2 / 10」不一致

- Reference: `project-context.md` Rule 24

### File List

**新增：**
- `apps/api/internal/services/generation_batch_items_test.go`
- `_bmad-output/implementation-artifacts/dsr-6d-a-batch-status-backend.md`（本檔）

**修改：**
- `apps/api/internal/services/generation_batch.go` — 每一部的狀態、`last`、`Snapshot`／`SnapshotFor`／`DismissLast`、`recover`、劇名、SSE payload
- `apps/api/internal/services/generation_batch_test.go` — SSE 鍵集合 11 → 13
- `apps/api/internal/handlers/generation_batch_handler.go`（＋`_test.go`）— status `last`、202 `progress`、dismiss 路由、Swagger
- `apps/api/internal/sse/hub.go` — 事件註解
- `apps/api/cmd/api/main.go` — `SetSeriesTitleResolver(repos.Series)`
- `apps/web/src/services/subtitleService.ts`（＋spec）— 型別與 `dismissGenerationBatch`
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` — 只補 `seriesTitle: ''`（型別）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 本張 → review

## 對抗式 Code Review（/ship，2026-09-18）

獨立 reviewer（fresh context，只讀；自己跑 `go test`／`go vet`／`-race`（services 45.9s、handlers 4.0s）與 `vitest subtitleService.spec.ts`，並用 `go test -overlay` 做了 6 個原始碼突變）回報 **2 HIGH、7 MED、8 LOW**。**修 11、交代 3、不修 1**（`reason` 快取失敗結果維持原設計）。修完 api 全過、`-race` 三個 package 乾淨、web 3651/3651、lint 0 errors、typecheck 通過。

| # | 等級 | 問題 | 處置 |
| --- | --- | --- | --- |
| H1 | HIGH | AC #3／#4 要求的「重入保護」完全沒有測試——突變把 `finish`／`markItem` 的 batchID 檢查拿掉，全部測試照過。真實情境：A 批次在 `finish` 裡 panic，recover 再進 `finish` 時 B 批次已經開始，沒有保護就會把 B 的佇列改成 A 的終態、灌 B 一個假的失敗數 | 補兩條測試（過期 batchID 的 `finish`／`markItem` 什麼都不做且不廣播；同一個批次再 `finish` 一次是冪等）。突變驗證：拿掉檢查 → 紅 |
| H2 | HIGH | 11 個新的 TypeScript 錯誤藏在 CI 看不到的地方（`tsconfig.spec.json` 有 3,115 個既有錯誤，沒有進 CI）：`seriesTitle`／`progress`／`items` 變必填後，對話框與工作區的 spec 假資料、以及 hook 的 `GenerationBatchProgressState` 都不再相容——`dsr-6d-b` 一開工就會踩到 | 補齊 8 個 spec 假資料、4 個 `GenerationBatchStartResult` 字面值；`GenerationBatchProgressState` 加上 `items`（型別對稱、初值 null），hook spec 期望同步。驗證：`tsc -p tsconfig.spec.json` 的錯誤數 3,106 → 3,103，且沒有任何一條提到 `items`／`seriesTitle`／`progress` |
| M1 | MED | `Start` 第二段（本張重構過）的 409 分支沒有測試，突變把 `processCancel()` 拿掉也全綠——那是唯一防止 context 洩漏的呼叫 | 補測試：兩個 `Start` 同時卡在 `collectItems` 裡，確認只有一個成功、另一個拿 `ErrGenerationBatchRunning`、只枚舉了一份佇列 |
| M2 | MED | 本張改過的檔案裡留了一句被自己改成假的註解（對話框：「status probe 沒有 items[]」） | 改寫並指向 `dsr-6d-b` |
| M3 | MED | SSE hub 丟事件時會把整包 payload 寫進 log——終態事件現在帶整份佇列，全選 2,400 部就是一行 400 KB 的 WARN | 改成只記事件類型與位元組數；補 hub 測試（log 裡不能出現 payload 內容、整行 < 500 字元） |
| M4 | MED | 兩個 SSE hook 會把終態事件的整份佇列跑一次 `snakeToCamel` 卻不讀它 | 交代給 `dsr-6d-b`／`dsr-6d-c`（它們才是讀 `items` 的人）；已寫進 sprint-status |
| M5 | MED | `last` 沒有 TTL，而 `dismissGenerationBatch()` 目前沒有呼叫端——大批次結束後佇列會一直留在記憶體 | 刻意（AC #3），寫進 Dev Notes 與 `dsr-6d-c` 條目 |
| M6 | MED | 完工 log 改成回頭讀計數後，讀不到就會安靜地印 0/0 | 讀不到時改印 `logger.Error`，不假裝是 0 |
| M7 | MED | 新測試 `QueueIsFilledBeforeStartReturns` 用 `t.Cleanup` 放行，批次 goroutine 會在測試結束後才跑完、對著正在關閉的 hub 廣播 | 改成測試內 `close(release)` ＋ `waitUntilIdle` |
| L1 | LOW | `SetSeriesTitleResolver` 寫的欄位沒有鎖保護 | 註解寫明「只在啟動接線時呼叫」（實際呼叫點在 `main.go`，服務啟動前） |
| L2 | LOW | 劇名查詢失敗也會進快取，一次暫時性錯誤會讓整批的劇名都空白 | **不修**：這正是避免 N+1 的設計；失敗會留一筆 WARN，顯示欄位空白不影響批次 |
| L3 | LOW | `finish` 在持鎖時呼叫 `logger.Error` | 收集起來、離開鎖之後再記 |
| L4 | LOW | `Start` 第一段快速檢查仍是手動 lock／unlock，與 AC #4 的「一律 defer」不一致 | 改用 `withLock` |
| L5 | LOW | `batchEventData` 有三個參數其實都來自同一個 struct，未來很容易傳成不一致 | 砍掉，直接讀 struct |
| L6 | LOW | 失敗原因的測試用 map 取值，出現預期外的項目時會靜靜通過 | 加 `require.Contains` |
| L7 | LOW | 409 的 `data` 理論上可能是 null，但前端型別寫成必有 | 交代給 `dsr-6d-b`（前端處理），已寫進 sprint-status |
| L8 | LOW | `reason: skipped` 同時代表「管線略過」與「legacy 沒金鑰」，文案要小心 | 交代給 `dsr-6d-b` 的 Sally 文案，已寫進 sprint-status |

查過不成立：終態的 `current_index`／`paused_count`／計數與改動前逐項相同（reviewer 用 git HEAD 對照並突變驗證）、沒有死結或鎖反轉、`lastBatch` 每次交出都是深複製、`lastBatch != nil ⟹ activeBatch == nil` 成立、panic 路徑的狀態與計數正確、加的鍵不會弄壞任何前端執行期解析（`snakeToCamel` 對巢狀陣列與 null 正確）、沒有 e2e 或 TestSprite 測試碰這些端點、handler 的三個形狀與 Swagger 正確、劇名快取不跨批次、`seriesTitle: ''` 佔位真的不影響渲染。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-18 | 🔍 **/ship 對抗式 CR**：2 HIGH／7 MED／8 LOW，修 11、交代 3、不修 1。最重要：① 重入保護（過期 batchID 的 finish／markItem）完全沒測試，突變全綠——補了兩條，現在拿掉保護就會紅；② `seriesTitle`／`items`／`progress` 變必填後，spec 那一側多了 11 個 TypeScript 錯誤而 CI 看不到（spec 的 tsconfig 有 3,115 個既有錯誤不進 CI）——全部補齊，hook 的狀態型別也加上 `items` 保持兩邊可互相指派。另外：SSE hub 丟事件時不再把整包 payload 寫進 log（終態事件現在帶整份佇列）、完工 log 讀不到計數時說實話、`finish` 不在持鎖時寫 log、單飛第二段補測試。 |
| 2026-09-18 | ✅ **dev-story 完成 → review**（Amelia）。api PASS（含本機 `-race`）、web 3651/3651、lint 0 errors、typecheck。後端現在直接說出每一部片怎麼結束的（完成／失敗＋原因／預算暫停／取消），批次結束後狀態查詢還留著最後結果（可 dismiss），開始的回應帶當下進度，背景 panic 變成 `error` 終態而不是讓 API 掛掉，分集帶劇名；SSE 執行中只送變動的那一部。前端只加型別與 `dismissGenerationBatch`（外加對話框一個型別欄位）。 |
| 2026-09-17 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；抽查約 40 個行號、本機 `go test -race` services／handlers 皆乾淨）：2 項 CRITICAL、10 項 SHOULD FIX、11 項 NIT，**全部併入**。最重要：① 佇列原本在背景 goroutine 才填，202 可能拿到空佇列 → 改在 `Start` 同一段臨界區填好；② 前端 `startGenerationBatch` 逐欄組結果，只加型別的話 `progress` 永遠拿不到 → 服務要帶出來；③ status 與 202 原本分兩次上鎖讀取會讀到不一致 → `Snapshot()`／`SnapshotFor(batchID)`；④ 終態轉換改由 `finish` 統一做並訂出「計數＝各狀態數量」的不變式；⑤ recover 若遇到持鎖 panic 會卡死 → 所有持鎖區段用 defer；⑥ 不改成序列化 struct（會讓 SSE 測試全部逾時、NaN 上限會編碼失敗）；⑦ 全選 2,400 部時每次廣播帶整個佇列約 400 KB → SSE 執行中只送 `changed_item`、劇名查詢必須快取；⑧ legacy 的未設定與管線略過同一個原因 `skipped`；⑨ 用既有的 `CandidateSeriesTitleResolver`＋setter，不改建構子；⑩ 補 Rule 20 契約紀錄與要改的過期註解清單、不要跑 `swag init`、本機 `-race` 輸出貼進紀錄。 |
| 2026-09-17 | Story 建立（SM Bob, create-story）。`dsr-6d` 三個唯讀稽核代理（Pencil 節點、程式碼與測試、金額／狀態色／預算規則）後拆成三張；⚖️ Alexyu 當日兩個裁定（上限提示赭色、列中性；先做後端單）。本張是後端：每一部的狀態與原因、最後一次結果＋dismiss、開始回應帶進度、`error` 終態真的送出、分集劇名；前端只改型別。 |
