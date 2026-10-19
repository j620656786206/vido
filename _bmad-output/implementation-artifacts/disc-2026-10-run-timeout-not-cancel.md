# Disc：排隊等抽軌逾時不再記成「取消」——活動記錄改寫「處理逾時」，不再像是使用者自己按了停止

Status: ready-for-dev

**Source:** `disc-2026-10-activity-subtitle-run-time-a`（PR #831）合併後，dev 在 Completion Notes 範圍外 #13 回報：字幕排隊等抽軌時逾時，紀錄被寫成取消（`error_message` 前面加 `cancelled: `），活動記錄因此寫「中途停止」，看起來像使用者自己按了停止。
⚖️ **Alexyu 2026-10-19 裁定 2A：** 逾時跟取消分開，寫成「處理逾時」，用一個新的失敗原因代碼。

⚖️ **SM 裁定（Bob，2026-10-19）：**
- **不拆單。** 後端 3 項（分類函式、寫紀錄的地方、自動生成的「失敗三次就不再試」規則）、前端 1 項（一個型別值＋一條文字），兩邊都沒有超過 3 項（Story Splitting Rule）。前後端放同一個 PR 還有一個好處：舊前端不認得新代碼時會顯示「生成失敗」，同一次部署就沒有這段空窗。
- **新代碼叫 `queue_timeout`**，預設文字「處理逾時」。
- **在寫紀錄的時候就分開**（加一個新前綴 `queue-timeout: `），不是只改讀的時候的字串比對。理由：錯誤原文存進資料庫前會截到 1000 字，ffmpeg 的錯誤輸出很長時，句尾的 `deadline exceeded` 會被截掉，事後比對字串就認不出來；寫的時候手上有真正的錯誤型別，判斷一定準。讀的那邊另外加一條字串規則，只用來接住沒有前綴的情況（還在記憶體裡的錯誤、以及舊紀錄）。
- **合併順序：本單先，`disc-2026-10-live-activity-failure-reasons-a`／`-b` 後。** 那兩張還沒開工（2026-10-19 沒有分支），而且 -a 的 T1 正要把「取消」的判斷抽成共用函式，判斷條件就是本單要拆開的那一行。本單先合併，-a 抽出來的就是已經拆好的版本。細節見 Dev Notes「跟即時活動兩張單的關係」。

設計稿：**沒有版面變更。** 預設文字「處理逾時」是畫面上已經在用的字（`timeout` 那條），活動記錄的列沿用 K1-D-v2 `LndN2`／`EVrLx`、K1-M-v2 `JLTBV`／`akt4i`（失敗列只是原因那格換字）。若 Sally 採用 Dev Notes 的提案句，也只是換字，不改版面。

## 查到的事（2026-10-19，讀 `origin/main` 6d31870f 與正式機）

- **逾時是怎麼被記成取消的：**
  1. 抽內嵌字幕軌一次只准一個（`ExtractGate`，`apps/api/internal/subtitle/extract_gate.go:10-27`），其他人排隊。排隊時如果 ctx 結束，`Acquire` 回傳 `ctx.Err()`（`:80-81`），不分是取消還是逾時。
  2. `Extractor` 把它包成 `ErrSubtitleExtractWaitAborted`（`extractor.go:347-350`：`"%w: %s (queued behind %d for %s): %w"`，最後的 `%w` 就是 `context.Canceled` 或 `context.DeadlineExceeded`）。另一種情況：排隊排太久、輪到時只剩一點時間，ffmpeg 跑到一半被上層的期限打斷，也包成同一個錯誤（`extractor.go:380-406`，訊息是 `ffmpeg stopped by the caller's deadline after … : context deadline exceeded`）。
  3. `failItem`（`apps/api/internal/subtitle/process_item.go:836-881`）寫紀錄時，只要 `errors.Is(cause, ErrSubtitleExtractWaitAborted)` 就加 `cancelled: `（`:860-862`）——**不看裡面包的是取消還是逾時**。註解（`:856-859`）說明原意：排隊逾時跟檔案無關，不能算進自動生成的「失敗三次就停」。
  4. 活動記錄讀到 `cancelled: ` 前綴就回 `cancelled`（`apps/api/internal/models/subtitle_run_failure.go:40-42`，規則第一條），前端寫「中途停止」（`apps/web/src/components/subtitle/generationEventCopy.ts:65-66`）。
- **真正的使用者取消怎麼記：** 同一行 `:860-861`：`errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)`。來源：關機、WorkerPool 停止、使用者在批次生成中途按取消（`CancelledRunPrefix` 註解 `process_item.go:711-722`）。排隊中被取消的錯誤包的是 `context.Canceled`，前兩個條件就會成立。
- **所以兩者分得開：** 排隊中斷的錯誤裡包的是 `context.Canceled` → 取消；包的是 `context.DeadlineExceeded` → 逾時。`ctx` 一旦逾時，`ctx.Err()` 永遠是 `DeadlineExceeded`，不會兩種混在一起。
- **哪裡會逾時：只有自動生成。** 全 repo 呼叫 `ProcessItem` 的五處裡，只有 `AutoGenerator`（`auto_generation.go:504` `context.WithTimeout(ctx, g.itemDeadlineFor(it.path))`，最少 15 分鐘、大檔案是抽軌時限＋5 分鐘，`:46`、`:52`、`:296-307`）給每一部設期限。單次生成（`solo_runner.go:246` `WithCancel`）、批次生成（`generation_batch.go:517`、`subtitle/batch.go:258` `WithCancel`）、WorkerPool 都沒有期限 → 在那些地方排隊**只會**被取消，不會逾時。
- **「失敗三次就不再試」只認 `cancelled: `：** `AutoGenerator.excludedMediaIDs`（`auto_generation.go:747`）數失敗紀錄時跳過 `cancelled: ` 開頭的（`:791-806`，判斷在 `:800`）。換新前綴後這裡**一定要一起改**，不然排隊逾時會開始算進三次，無辜的片子會被永久跳過。迴圈裡另有計數 `case errors.Is(err, ErrSubtitleExtractWaitAborted)`（`:520-528`）只看錯誤型別、不看前綴，不受影響，只改註解。
- **既有測試在釘住「逾時記成取消」：** `TestProcessItem_WaitAbortIsRecordedAsCancelledNotAsAFileFailure`（`process_item_test.go:1239-1252`，錯誤包 `context.DeadlineExceeded`，斷言前綴是 `CancelledRunPrefix`）——本單要改這支。其他相關：`TestProcessItem_DeadlineExceededIsNotMarkedCancelled`（`:850-866`，一般逾時不加前綴，不變）、`TestAutoGenerator_CancelledFailuresDoNotCountTowardParking`（`auto_generation_test.go:1082-1094`）、`TestAutoGenerator_CancelledRowsDoNotRescueAParkedItem`（`:1098-1107`）、`TestAutoExclusion_CancelledRowsStayExemptAfterReplacement`（`:1286`）、`TestAutoGenerator_QueueTimeoutIsNotCountedAsAFileFailure`（`:1330-1352`）、夾具 `cancelledRun`（`:137-146`）。
- **分類函式與合約：** 十個代碼（`subtitle_run_failure.go:10-21`），規則依序比對（`:36-59`），`SubtitleRunCancelledPrefix`（`:61-64`）。`TestSubtitleRunFailureReason_CodesAreTheContract`（`subtitle_run_failure_test.go:81-94`）寫死十個代碼。合約是 `disc-2026-10-activity-subtitle-run-time-a` AC #1 `[@contract-v1]`（`failure_reason` 的值域），加一個值 → Rule 20 要升 v2。活動服務呼叫分類函式在 `activity_service.go:490-491`。
- **前端：** `RunFailureReason`（`generationEventCopy.ts:47-57`）、`FAILURE_COPY_BY_REASON`（`:64-81`，型別是 `Record<RunFailureReason, FailureCopy>`，少一個鍵 typecheck 就會失敗）、`failureCopyForReason`（`:84-88`，不認得的代碼 →「生成失敗」）。測試寫死代碼清單：`generationEventCopy.spec.ts:355-369`「has exactly the ten contract codes」、`recentEventCopy.spec.ts:130-157` 逐代碼對文字。`activityService.ts:90` 註解寫「ten」。
- **舊資料（正式機，2026-10-19 唯讀查 `/mnt/cache/appdata/vido/vido.db`）：**
  - `subtitle_runs` 共 34 列（completed 27、failed 7）；7 列 failed **沒有一列**是 `cancelled: ` 開頭，也沒有 `SUBTITLE_EXTRACT_WAIT_ABORTED`。
  - `media_libraries` 兩個媒體庫（tv、movie）的 `auto_subtitle` 都是 0 → 正式機的自動生成**從來沒跑過**，所以也不可能有「被記成取消的逾時」。
  - 結論：**正式機沒有需要修的舊紀錄，不做資料遷移。** 就算其他環境有舊紀錄，AC #2 的字串規則會在讀的時候把它們認成 `queue_timeout`（訊息沒被截斷的話）；被截斷到看不出逾時的舊紀錄維持「中途停止」。
- **目前影響：** 正式機今天看不到這個問題（自動生成沒開）。Alexyu 一打開「自動字幕」，大檔案排隊排太久時，活動記錄就會出現假的「中途停止」。→ P3。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要字幕因為排隊等太久而沒做成時，活動記錄寫「處理逾時」，不要寫「中途停止」，
這樣我不會以為是自己或重開機把它停掉的，也知道那部片子本身沒有壞。

## Acceptance Criteria

1. **新代碼 `[@contract-v2]`（升級 `disc-2026-10-activity-subtitle-run-time-a` AC #1 `[@contract-v1]`）：** `failure_reason` 的值域多一個 `queue_timeout`，共十一個。其他代碼的意思、欄位、何時出現都不變。後端仍只送代碼、不送文字。
2. **分類函式（`models.SubtitleRunFailureReason`）：** 新規則排在**第一條**（在 `cancelled` 之前），符合任一就回 `queue_timeout`：
   - 開頭是新前綴 `models.SubtitleRunQueueTimeoutPrefix`（`"queue-timeout: "`）；
   - 同時含 `SUBTITLE_EXTRACT_WAIT_ABORTED` 與 `deadline exceeded`（沒有前綴的情況：還在記憶體裡的錯誤、或本單以前寫的舊紀錄 `cancelled: …SUBTITLE_EXTRACT_WAIT_ABORTED…context deadline exceeded`）。
   
   排第一的原因：舊紀錄同時有 `cancelled: ` 前綴和逾時字樣，要認成逾時。不會誤判真的取消：排隊中被取消的訊息結尾是 `context canceled`，不含 `deadline exceeded`。
3. **寫紀錄時分開（`failItem`）：** 判斷順序：
   1. 使用者／系統取消（`errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)`）→ 前綴 `cancelled: `（不變）；
   2. 否則，`errors.Is(cause, ErrSubtitleExtractWaitAborted)` → 前綴 `queue-timeout: `（`subtitle.QueueTimeoutRunPrefix = models.SubtitleRunQueueTimeoutPrefix`，比照 `CancelledRunPrefix` 的寫法）；
   3. 其他 → 不加前綴（一般逾時照舊，算進三次）。
   
   排隊中被取消（錯誤同時是 `ErrSubtitleExtractWaitAborted` 和 `context.Canceled`）走第 1 條，仍是「中途停止」。
4. **「失敗三次就不再試」行為不變：** `excludedMediaIDs` 跳過 `cancelled: ` **和** `queue-timeout: ` 兩種前綴（抽一個小函式，例如 `notAboutTheFile(msg string) bool`，兩處註解都指向它）。排隊逾時今天不算進三次，改完後**仍然不算**。
5. **活動記錄看得到：** `GET /api/v1/activity` 對一筆排隊逾時的紀錄送 `"failure_reason":"queue_timeout"`，JSON 裡不出現原始錯誤（路徑、`SUBTITLE_EXTRACT_WAIT_ABORTED` 都不行）；同一份資料裡一筆真的取消送 `"cancelled"`。
6. **前端：** `RunFailureReason` 加 `'queue_timeout'`；`FAILURE_COPY_BY_REASON` 加 `queue_timeout: Object.freeze({ text: '處理逾時', stopped: false })`（文字與 `stopped` 見 Dev Notes 的 Sally 待追認）。活動記錄失敗列顯示「片名 · 處理逾時」。
7. **會變紅的測試（每一條都要能被下面的「故意改壞」打紅）：**
   - **寫紀錄**：`TestProcessItem_WaitAbortIsRecordedAsCancelledNotAsAFileFailure` 改名為 `…RecordedAsQueueTimeoutNotAsAFileFailure`：錯誤包 `context.DeadlineExceeded` → 前綴是 `QueueTimeoutRunPrefix`、**不是** `CancelledRunPrefix`，且 `models.SubtitleRunFailureReason(run.ErrorMessage) == "queue_timeout"`。新增 `TestProcessItem_WaitAbortUnderCancellationIsStillCancelled`：同一個錯誤改包 `context.Canceled` → `cancelled: ` 前綴、分類 `cancelled`。
     - 故意改壞①：`failItem` 改回原本那一行 → 第一支紅。
     - 故意改壞②：把第 2 條判斷搬到第 1 條前面 → 第二支紅。
   - **三次規則**：新增 `TestAutoGenerator_QueueTimeoutFailuresDoNotCountTowardParking`（照 `:1082-1094`：上限減一筆真失敗＋兩筆 `queue-timeout: ` 紀錄 → 片子仍被列出）與 `…QueueTimeoutRowsDoNotRescueAParkedItem`（照 `:1098-1107`）。夾具加 `queueTimeoutRun(mediaID)`，訊息用真實形狀（`queue-timeout: subtitle pipeline: movie <id>: route: subtitle route: extract /media/movies/…mkv: SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot: ….mkv (queued behind 1 for 15m0s): context deadline exceeded`）。
     - 故意改壞③：`excludedMediaIDs` 只認 `cancelled: ` → 第一支紅。
   - **分類函式表格**（`subtitle_run_failure_test.go`）新增：新前綴 → `queue_timeout`；舊紀錄形狀 `cancelled: …SUBTITLE_EXTRACT_WAIT_ABORTED…(queued behind 2 for 15m0s): context deadline exceeded` → `queue_timeout`；排隊中被取消 `cancelled: …SUBTITLE_EXTRACT_WAIT_ABORTED…: context canceled` → `cancelled`；沒前綴的記憶體形狀 → `queue_timeout`；排隊後 ffmpeg 被打斷的形狀（`…SUBTITLE_EXTRACT_WAIT_ABORTED: SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg stopped by the caller's deadline after 3m0s on x.mkv … : context deadline exceeded`）→ `queue_timeout`；**正式機三條既有逾時原字串（`prodAITimeout`、`prodFFmpegTimeout`）照舊是 `timeout`**；舊紀錄被截斷到沒有 `deadline exceeded`（`cancelled: …SUBTITLE_EXTRACT_WAIT_ABORTED…(ffmpeg stderr: …)…`）→ `cancelled`（釘住已知限制）。`TestSubtitleRunFailureReason_CodesAreTheContract` 的清單加 `queue_timeout`，訊息改 `[@contract-v2]`。
     - 故意改壞④：新規則排在 `cancelled` 後面 → 舊紀錄那條紅。
     - 故意改壞⑤：新規則拿掉 `SUBTITLE_EXTRACT_WAIT_ABORTED` 條件（只看 `deadline exceeded`）→ `prodFFmpegTimeout` 那條紅。
   - **跨套件防呆**（`subtitle` 套件的測試）：`models.SubtitleRunFailureReason(fmt.Errorf("x: %w: m.mkv (queued behind 1 for 15m0s): %w", ErrSubtitleExtractWaitAborted, context.DeadlineExceeded).Error()) == "queue_timeout"`。`models` 不能 import `subtitle`，分類函式裡的 `SUBTITLE_EXTRACT_WAIT_ABORTED` 是寫死的字串——有人改了哨兵字串，這支會紅。
   - **活動服務**（`activity_recent_test.go`，真遷移鏈）：種一筆 `queue-timeout: ` 紀錄＋一筆 `cancelled: …context canceled` 紀錄，斷言 JSON 的 `failure_reason` 分別是 `queue_timeout`、`cancelled`，整串 JSON 不含 `SUBTITLE_EXTRACT_WAIT_ABORTED` 與 `/media/`。
   - **前端**：`recentEventCopy.spec.ts:130-157` 的 `it.each` 加 `['queue_timeout', '處理逾時']`（故意改壞⑥：漏加 `FAILURE_COPY_BY_REASON` 那一條 → 文字退回「生成失敗」→ 紅；typecheck 也會紅）。`generationEventCopy.spec.ts:355-369` 改名「has exactly the eleven contract codes」並加 `queue_timeout`。
8. **註解跟著改：** `subtitle_run_failure.go` 檔頭與規則說明（`:3-9`、`:23-35`、`:77-90`，十→十一、新規則為什麼排第一、為什麼寫死哨兵字串）；`process_item.go` `CancelledRunPrefix`（`:711-722`）與 `failItem`（`:850-859`）；`errors.go:43-48`（「the item flow treats it like a cancellation」改成「記成 queue-timeout，不算進三次」）；`auto_generation.go:520-528`、`:787-799`（`:509-512` 講的是真的取消，不用改）；`generationEventCopy.ts:37-46`（後端多認的代碼加上 `queue_timeout`）；`activityService.ts:90`（ten → eleven）。
9. **驗證：** `apps/api` 下 `go test ./...`、`go vet ./...`、`~/go/bin/staticcheck ./...` 綠；`pnpm nx run web:test`（activity＋subtitle 相關 spec）、typecheck、`pnpm run lint:all`（看「✖ N problems (X errors」那行）、`pnpm run format:check` 綠。

## Tasks / Subtasks

- [ ] **T1 models（AC #1、#2、#7 分類表格）**：`subtitle_run_failure.go` 加 `SubtitleRunFailureQueueTimeout = "queue_timeout"`、`SubtitleRunQueueTimeoutPrefix = "queue-timeout: "`、第一條規則；表格測試、合約清單測試。
- [ ] **T2 寫紀錄（AC #3、#7 寫紀錄＋跨套件）**：`process_item.go` 加 `QueueTimeoutRunPrefix`，`failItem` 改成 AC #3 的三段判斷；改名／新增 `process_item_test.go` 兩支；跨套件防呆測試。
- [ ] **T3 三次規則＋活動服務測試（AC #4、#5、#7）**：`auto_generation.go` 抽 `notAboutTheFile`、`:800` 改用它；`auto_generation_test.go` 夾具＋兩支；`activity_recent_test.go` 一支。
- [ ] **T4 前端（AC #6、#7 前端）**：`generationEventCopy.ts` 型別＋文字；兩個 spec；`activityService.ts:90` 註解。
- [ ] **T5 合約與文件（AC #1、#8、Rule 20）**：
  - `disc-2026-10-activity-subtitle-run-time-a.md` AC #1 的戳改 `[@contract-v2]`，Change Log 加一列：`| 2026-10-?? | [@contract-v1→v2] AC #1: failure_reason 多一個值 queue_timeout（排隊等抽軌逾時，原本記成 cancelled）；下游：前端 RunFailureReason／FAILURE_COPY_BY_REASON 要多一個鍵，舊前端會把它顯示成「生成失敗」 |`。
  - 下游掃描（Rule 20 🔁）：`disc-2026-10-activity-subtitle-run-time-b`（done → 不動）；`disc-2026-10-live-activity-failure-reasons-a`、`-b`（ready-for-dev → **stale-mark**：兩張的 Dev Notes 與 sprint-status 條目各加 `⚠️ STALE [@contract-v1→v2]: upstream disc-2026-10-activity-subtitle-run-time-a AC #1 bumped 2026-10-??; re-confirm against v2 before dev — see disc-2026-10-run-timeout-not-cancel`），並在兩張的 Dev Notes 補下面「跟即時活動兩張單的關係」列的要點。
  - AC #8 的註解；AC #9 的驗證。

## Dev Notes

### 跟即時活動兩張單的關係（`disc-2026-10-live-activity-failure-reasons-a`／`-b`）

- **合併順序：本單 → live-a → live-b。** live 兩張 2026-10-19 還沒開工（沒有分支）。
- **live-a 開工時要涵蓋新代碼（本單 T5 寫進它的 Dev Notes）：**
  - live-a T1 要從 `failItem` 抽出 `isCallerCancellation(cause, ctxErr)`，它寫的條件**含** `ErrSubtitleExtractWaitAborted`（live-a T1、AC #2 第 2 步）。本單合併後，那一行已經拆成「取消」與「排隊逾時」兩段——`isCallerCancellation` 只能包第 1 段（`context.Canceled`），**不能**再含 `ErrSubtitleExtractWaitAborted`。
  - live-a AC #2 單次生成的判斷：第 1 步讀 `run.ErrorMessage` 已經會得到 `queue_timeout`（本單寫了前綴）。第 2 步「取消類」之後要加一步「`errors.Is(runErr, ErrSubtitleExtractWaitAborted)` → `queue_timeout`」（實際上單次生成沒有期限，不會發生，但規則要跟紀錄一致）。
  - live-a AC #3 一致性測試的「`ErrSubtitleExtractWaitAborted` → `cancelled`」要改成兩條：包 `context.Canceled` → `cancelled`；包 `context.DeadlineExceeded` → `queue_timeout`。
  - live-a AC #1 合約的值域「十個代碼」→ 十一個。
- **live-b 開工時要涵蓋新代碼：** AC #5「對十個代碼逐一」→ 改成對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一（以後再加代碼就不用改測試）。live-b 會刪 `TABLE`／`failureCopy()`；本單**不動** `TABLE`（見下）。
- **如果 live-a 搶先合併了：** 本單 T2 改成修改 live-a 抽出的 `isCallerCancellation`（拿掉 `ErrSubtitleExtractWaitAborted`）並新增 `isQueueTimeout`，`soloFailureReason` 補上述那一步；其餘 AC 不變。

### 不要做的事

- 不要拿掉「排隊逾時不算進三次」的行為（AC #4）。這是 sub-6-3 CR H1 的裁定：排隊排到逾時，跟片子本身無關。
- 不要改 `TestProcessItem_DeadlineExceededIsNotMarkedCancelled`（`process_item_test.go:850-866`）的意思：真的處理太久（不是排隊）照舊不加前綴、算進三次、活動記錄寫「處理逾時」（`timeout`）。
- 不要動 `failureCopy` 的 `TABLE`（`generationEventCopy.ts:91-111`）：即時活動實際上收不到排隊逾時（自動生成不送 `transcription_failed`，只送 `subtitle_progress`），而且 live-b 會整個刪掉它。分類函式的註解「跟 TABLE 同一組字串」要寫明 `queue_timeout` 跟 `cancelled`、`target_not_writable` 一樣是後端才認得的。
- 不要做資料遷移（正式機沒有這種紀錄，見「查到的事」）。
- 不要改 `ExtractGate`、`Extractor` 的錯誤包法；本單只改寫紀錄與讀紀錄的地方。
- 不要加畫廊夾具或重產視覺基準（文字跟既有的 `timeout` 一模一樣，畫面不會變）。

### ⚖️ 待 Sally 追認（不擋開工，預設照做）

1. **文字：預設「處理逾時」**（跟 `timeout` 同一句，沿用畫面上已有的字）。**Bob 提案：「等待抽軌逾時」**——「等待抽軌」是生成畫面排隊時已經在用的字（`progress_sse.go:89`「等待抽軌（前方 N 件）」），讓人分得出「排隊排太久」跟「片子處理太久」，也暗示片子本身沒問題。長度 6 個字，比「影片所在的資料夾寫不進字幕」短，手機不會多折行。Sally 採用的話只改 `FAILURE_COPY_BY_REASON.queue_timeout.text` 與 `recentEventCopy.spec.ts` 那一條。
2. **顏色：預設 `stopped: false`**（紅色「失敗」，跟 `timeout` 一樣）。活動記錄目前不看 `stopped`（標題一律「字幕沒做成」），只影響 live-b 之後的即時活動，而即時活動實際上不會出現這個代碼。

### 測試標準

- Go：testify、表格測試；repository／service 測試走真遷移鏈（Rule 28）。`lint:all` 不含 staticcheck，push 前另跑。
- Vitest：斷言用看得到的文字（Rule 16）。不要背景跑測試。

### E2E data-testid／視覺基準

- **data-testid：** 沒有新增、改名、移除。
- **E2E：** 不受影響（`git grep -n "failure_reason\|中途停止\|處理逾時" origin/main -- tests` 沒有命中）。
- **視覺基準：** 不受影響。活動記錄的畫廊夾具（`-gallery.fixtures.tsx:1020`）用的是 `target_not_writable`，沒有任何夾具用到 `cancelled` 或新代碼。不跑 `test:visual:update`、不 commit 任何 png。

### Project Structure Notes

- 後端：`apps/api/internal/models/{subtitle_run_failure.go,subtitle_run_failure_test.go}`、`apps/api/internal/subtitle/{process_item.go,process_item_test.go,errors.go,auto_generation.go,auto_generation_test.go}`（跨套件防呆測試放 `errors_test.go` 或 `process_item_test.go`）、`apps/api/internal/services/activity_recent_test.go`。沒有 migration、沒有新 endpoint。
- 前端：`apps/web/src/components/subtitle/{generationEventCopy.ts,generationEventCopy.spec.ts}`、`apps/web/src/components/activity/recentEventCopy.spec.ts`、`apps/web/src/services/activityService.ts`（註解）。
- 文件：`disc-2026-10-activity-subtitle-run-time-a.md`（戳＋Change Log）、`disc-2026-10-live-activity-failure-reasons-{a,b}.md`（stale-mark＋要點）、`sprint-status.yaml`。
- 跟 `disc-2026-10-activity-run-time-sr-label`（`RecentSection.tsx`、`recentEventCopy.ts`）共用 `recentEventCopy.spec.ts`，但改的是不同的 `describe`，衝突只會是相鄰行。

### Time-dependent visual coverage

- N/A — 沒有讀時鐘的元件改動。

### References

- [Source: apps/api/internal/subtitle/process_item.go:711-722, 836-881]
- [Source: apps/api/internal/subtitle/errors.go:43-49]
- [Source: apps/api/internal/subtitle/extractor.go:347-350, 380-406, 470-479]
- [Source: apps/api/internal/subtitle/extract_gate.go:51-84]
- [Source: apps/api/internal/subtitle/auto_generation.go:46, 52, 296-307, 501-534, 747, 787-806]
- [Source: apps/api/internal/subtitle/solo_runner.go:246；services/generation_batch.go:517；subtitle/batch.go:258]（沒有期限的呼叫者）
- [Source: apps/api/internal/models/subtitle_run_failure.go:10-92；subtitle_run_failure_test.go:12-94]
- [Source: apps/api/internal/services/activity_service.go:490-491；activity_recent_test.go:154-288]
- [Source: apps/api/internal/subtitle/process_item_test.go:815-866, 1239-1252；auto_generation_test.go:137-146, 1082-1107, 1286, 1330-1352]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:37-88；generationEventCopy.spec.ts:341-379]
- [Source: apps/web/src/components/activity/recentEventCopy.spec.ts:129-157；apps/web/src/services/activityService.ts:90]
- [Source: apps/api/internal/subtitle/progress_sse.go:85-90]（「等待抽軌」原句）
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-a.md AC #1 `[@contract-v1]`、Completion Notes 範圍外 #13]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-a.md T1、AC #1–#3；-b.md AC #5]
- [Source: project-context.md Rule 20（升版＋下游 stale-mark）、Rule 16、Rule 28]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-19 | 建單（Bob）。⚖️ Alexyu 2A：排隊逾時與取消分開。本單把 `disc-2026-10-activity-subtitle-run-time-a` AC #1 升 `[@contract-v1→v2]`（`failure_reason` 加 `queue_timeout`），實際改戳與下游 stale-mark 在 T5。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## ⚖️ Sally 追認（2026-10-19）

- 文字用「處理逾時」，不用「等待抽軌逾時」：「抽軌」是內部用語，使用者看不懂；這張要解決的是「不要寫成中途停止」，「處理逾時」已經做到。
- 顏色跟 `timeout` 一樣，紅色「失敗」。
