# Disc：即時活動的失敗原因 A（後端）——字幕生成失敗的即時事件也帶「原因代碼」，跟活動記錄用同一把尺

Status: done

**Source:** `disc-2026-10-activity-subtitle-run-time-b`（PR #828）合併後，dev 回報：生成字幕畫面的「即時活動」認不出「寫不進資料夾」和「中途停止（取消）」。同一次失敗，即時活動寫「生成失敗」，活動記錄寫「影片所在的資料夾寫不進字幕」。
⚖️ **Alexyu 2026-10-19 裁定 2A：** 讓兩邊說法一致——即時活動改用活動記錄那份失敗原因文字（`FAILURE_COPY_BY_REASON`，#828 收成一份）。

⚖️ **SM 裁定（Bob，2026-10-19）——由後端在即時事件上送原因代碼，前端不再比對英文錯誤字串；拆成兩張：**
- **為什麼要後端加欄位（不只是在前端表格補兩條）**：
  1. **前端拿不到判斷「中途停止」需要的資訊。** 活動記錄的「中途停止」靠紀錄裡的 `cancelled: ` 前綴，這個前綴只寫進資料庫（`process_item.go:857-860` 的 `failItem`）。即時事件送的是 `runErr.Error()`（`solo_runner.go:415-416`、`:458`），`failItem` 回傳時 outcome 是 `nil`（`process_item.go:877` `return nil, err`），所以**事件上永遠沒有那個前綴**。被 kill 掉的子程序錯誤是 `signal: killed`（`process_item.go:847-849` 的註解），前端無從得知它是取消。
  2. **批次裡翻譯／放檔那段失敗，前端連原始字串都拿不到。** 批次成員失敗時，`changed_item.reason` 只有 `"error"`（`generation_batch.go:800-806`）；前端只有在同一部片先收到 `transcription_failed` 時才有字串（`useGenerationJobsFeed.ts:428-436`）——那只有語音辨識那條路會送。翻完才寫不進資料夾（正式機 和平使者 S01E01）走的是翻譯路線，**前端什麼都收不到**，只能寫「生成失敗」。
  3. **長解（`feedback_architecture_prefer_long_solutions`）：** 兩邊各一套字串比對（後端 `models.SubtitleRunFailureReason`、前端 `failureCopy` 的 `TABLE`），規則已經不一樣（後端多兩條、順序也不同，`generationEventCopy.ts:38-46` 自己承認）。收成後端一套、事件送代碼，以後加一條規則只改一處。
- **拆單（Story Splitting Rule）**：後端 4 項（單次生成事件、舊路徑事件、批次項目、測試＋註解）、前端 4 項（型別與 hook、畫面文字、刪掉字串比對與改測試、夾具與註解），兩邊都 >3 → 拆。
  - **A（本單，後端）**：三種即時事件多一個 `failure_reason` 欄位（AC #1 合約）。純加欄位，舊前端不看它，**單獨上線無害**。
  - **B（前端）**：`disc-2026-10-live-activity-failure-reasons-b`，改讀代碼、刪 `TABLE`。
  - ⛔ **合併順序：A 先，B 後**（B 刪掉字串比對之後，若後端還沒送代碼，預算／額度那些原本認得的也會變「生成失敗」）。
- **依賴：** ⛔ blocked-by `disc-2026-10-activity-subtitle-run-time-a`（分類函式 `models.SubtitleRunFailureReason` 在那張，`apps/api/internal/models/subtitle_run_failure.go`，目前只在 `feat/activity-subtitle-run-time-a` 分支）。那張合併後才開工；下面引用的 `models/` 行號以那張的 commit 2bb2c102 為準，其他檔案以 `origin/main` 9331d1e2 為準。

設計稿：**無變更**（後端單）。前端畫面見 B 單。

## 查到的事（2026-10-19）

- **分類函式已經有了**：`models.SubtitleRunFailureReason(errorMessage string) string`（`models/subtitle_run_failure.go:85-92`，2bb2c102），回十個代碼之一（`:11-22`）；規則順序 cancelled → budget → credit → target_not_writable → no_audio → ffmpeg → asr_unconfigured → unauthorized → timeout（`:36-59`）；`SubtitleRunCancelledPrefix = "cancelled: "`（`:64`）。測試 `subtitle_run_failure_test.go` 有正式機四條原字串。檔頭註解（`:32-35`、`:82-84`）寫「跟前端 `failureCopy` TABLE 同一組字串，加規則兩邊一起改」——本單之後這句要改（AC #5）。
- **即時事件有三個出口：**
  1. **單次生成（新管線）**：`SoloRunner.emitTerminal`（`apps/api/internal/subtitle/solo_runner.go:374-421`）。失敗那段 `:410-418`：`data := s.jobFields(ref, job, "failed")`、`route`、`message, detail := failureMessage(run, runErr, job.Budget)`（`:441-466`）、`data["error"] = detail`，送 `transcription_failed`。`detail` 是 `runErr.Error()` 或 `run.ErrorMessage`（`:458`、`:462`）；額度用完時是一句中文（`:449-452`，含「帳號沒有額度」）。
  2. **舊路徑（Route C）**：`TranscriptionService.failJob`（`apps/api/internal/services/transcription_service.go:2355-2369`），送 `transcription_failed`，`error` 是 `errMsg`，呼叫點 `:963`、`:979`、`:985`、`:1002`、`:1075`、`:1084`、`:1097`。
  3. **批次**：`GenerationBatchProcessor.markItem`（`apps/api/internal/services/generation_batch.go:837`）送 `generation_batch_progress` 的 `changed_item`（`GenerationBatchItemState`，`:144-150`，「每個鍵都一定在」）。成員失敗的分支 `:800-806`（`reason: "error"`，手上有 `err`）；`finish(error)` 把跑到一半的項目標成 `error`（`:893-897`）。
- **取消怎麼判斷（要跟紀錄一致）**：`failItem`（`process_item.go:833-877`）在 `errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) || errors.Is(cause, ErrSubtitleExtractWaitAborted)` 時加 `cancelled: `（`:857-860`）；`ErrSubtitleExtractWaitAborted`（`internal/subtitle/errors.go:49`）。錯誤是 `fmt.Errorf("…: %w", cause)` 包起來的（`:834`），所以 `errors.Is(runErr, …)` 在 `emitTerminal` 也成立。單次生成的 `runCtx` 只在結束後才 cancel（`solo_runner.go:246-247`），所以 `ctx.Err()` 那一條在單次生成不會發生。
- **原始錯誤照送**：詳情對話框的失敗面板會顯示 `error`（`solo_runner.go:438-440` 註解「the diagnostic detail the panel shows under it」、前端 `GenerationProgressV2.tsx:94`）。本單**不拿掉** `error`／`message`，只加欄位。
- **SSE 事件鍵的既有測試**：批次 payload 鍵清單 `generation_batch_test.go:275-291`；單次失敗 `solo_runner_test.go:337-376`（`TestSoloRunner_FailedTerminals`，harness 有 `s.runs`＝`fakeRunStore`、`s.placer`，`process_item_test.go:114-127`）；舊路徑 `transcription_run_budget_test.go:194-202`、`transcription_event_title_test.go:83`。
- **SSE 合約文件**：`internal/sse/hub.go:35-46`（`generation_batch_progress` 的鍵）；`transcription_*` 的鍵沒有集中文件，寫在各 emitter。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要字幕生成一失敗，即時事件就直接帶「為什麼」的代碼，而且跟活動記錄用同一套判斷，
這樣生成畫面和活動頁對同一次失敗會講同一句話。

## Acceptance Criteria

1. **合約 `[@contract-v2]`（v1 本單定、B 單 ack；v2 由 `disc-2026-10-no-text-source-skip-a` 升，差異見檔尾 Change Log）：** 新增鍵 `failure_reason`（字串），值域＝`models.SubtitleRunFailure*` 十個代碼（`cancelled`、`budget_exceeded`、`credit_exhausted`、`target_not_writable`、`no_audio_track`、`ffmpeg_missing`、`asr_unconfigured`、`unauthorized`、`timeout`、`other`）。
   | 事件 | 什麼時候有 | 值 |
   |---|---|---|
   | `transcription_failed`（單次生成，`SoloRunner`） | 每一個失敗事件都有 | AC #2 的判斷 |
   | `transcription_failed`（舊路徑，`TranscriptionService.failJob`） | 每一個都有 | `models.SubtitleRunFailureReason(errMsg)` |
   | `generation_batch_progress` 的 `changed_item`，以及終局 `items[]` 每一項（同一個型別 `GenerationBatchItemState`） | **鍵一定在**（沿用該型別「每個鍵都一定在」的規則） | `status == "failed" && reason == "error"` 時是十個代碼之一；其餘一律 `""` |
   - `error`、`message` 與其他既有鍵**不變**。
   - 不送任何新文字（後端只送代碼，文字在前端，`activity_service.go:111-113` 的規則）。
2. **單次生成的判斷跟活動記錄一致：** `emitTerminal` 失敗事件的 `failure_reason` 用一個小函式算（建議 `soloFailureReason(run *models.SubtitleRun, runErr error) string`，放 `solo_runner.go`）：
   1. `run != nil && run.ErrorMessage != ""` → `models.SubtitleRunFailureReason(run.ErrorMessage)`（跟活動記錄讀的是同一個字串）；
   2. `runErr` 是取消類（**跟 `failItem` 加 `cancelled: ` 的條件共用同一個函式**，見 Task 1）→ `cancelled`；
   3. `errors.As(runErr, *ai.CreditExhaustedError)` → `credit_exhausted`；`errors.Is(runErr, ai.ErrBudgetExceeded)` → `budget_exceeded`；
   4. 其餘有 `runErr` → `models.SubtitleRunFailureReason(runErr.Error())`；
   5. 都沒有 → `other`。
3. **一致性測試（最重要，會變紅的那種）：** 在 `solo_runner_test.go` 為下列每一種失敗各跑一次單次生成，斷言「事件的 `failure_reason`」**等於**「`models.SubtitleRunFailureReason(s.runs 裡這次那一列的 ErrorMessage)`」，也等於預期代碼：
   - 放檔時 `permission denied`（模擬 和平使者 S01E01：翻完才寫不進）→ `target_not_writable`；
   - 取消：`cause` 是 `context.Canceled` → `cancelled`；`ErrSubtitleExtractWaitAborted` → `cancelled`；
   - 預算上限（`ai.ErrBudgetExceeded`）→ `budget_exceeded`；
   - 額度用完（`*ai.CreditExhaustedError`）→ `credit_exhausted`；
   - 路由錯誤 `ffmpeg exploded`（既有 `:366-375`）→ `other`。
   故意改壞驗證：把 Task 1 的共用判斷在 `emitTerminal` 換回只看 `runErr.Error()` → 兩條取消案例變紅（事件會是 `other`／`timeout`，紀錄是 `cancelled`）。
4. **批次與舊路徑測試：**
   - `generation_batch_test.go`：runner 回 `fmt.Errorf("place: placer: write failed: …: %w", fs.ErrPermission)` 類的錯（字串含 `placer: write failed` 與 `permission denied`）→ `changed_item.failure_reason == "target_not_writable"`；成功、`skipped`、`busy_elsewhere` 的項目 `failure_reason == ""`；`finish(error)` 標成失敗的項目是 `other`；終局 `items[]` 每一項都有這個鍵。鍵清單測試（`:275-291`）不動（它驗的是批次外層的鍵），另加一條驗 `changed_item` 的鍵。
   - 舊路徑：`failJob` 送出的事件有 `failure_reason`，`"extract audio: no audio track found in media file"` → `no_audio_track`、`"save SRT: open …: permission denied"` → `other`（後端規則只認 placer 的寫入失敗，跟活動記錄一致；見 Dev Notes）。
5. **把前端那份表的每一條搬進後端測試：** `apps/web/src/components/subtitle/generationEventCopy.spec.ts:12-80`（`describe('failureCopy')`）的每一個輸入字串，在 `models/subtitle_run_failure_test.go` 都要有一條（預期代碼照前端預期的文字對回去：已達預算上限→`budget_exceeded`、找不到可用的音軌→`no_audio_track`、伺服器缺少 ffmpeg→`ffmpeg_missing`、語音辨識未設定→`asr_unconfigured`、API 金鑰無效→`unauthorized`、處理逾時→`timeout`、AI 帳號沒有額度→`credit_exhausted`、生成失敗→`other`）。B 單會刪掉前端那份測試，所以這些案例要先在這裡落地。**若有任何一條後端結果跟前端不同，停下來在 Completion Notes 列出**（今天預期全部相同；順序不同只在「同時含兩種關鍵字」時才有差）。
6. **註解：** `models/subtitle_run_failure.go:32-35`、`:82-84` 改成「即時事件（`transcription_failed`、批次 `changed_item`）與活動記錄都用這個函式；前端只讀代碼，不再比對字串（`disc-2026-10-live-activity-failure-reasons-b`）」。`sse/hub.go:35-46` 的 `generation_batch_progress` 說明補 `changed_item.failure_reason`。`solo_runner.go:438-440`、`transcription_service.go:2355` 上方各一行說明新鍵。
7. `go test ./...`、`go vet ./...`、`~/go/bin/staticcheck ./...`（`apps/api` 下）全綠。

## Tasks / Subtasks

- [x] **T1 取消判斷共用（AC #2、#3）**：從 `failItem`（`process_item.go:857-860`）抽出 `isCallerCancellation(cause, ctxErr error) bool`（`errors.Is(cause, context.Canceled) || errors.Is(ctxErr, context.Canceled) || errors.Is(cause, ErrSubtitleExtractWaitAborted)`），`failItem` 改呼叫它（行為不變，既有測試照綠）。
- [x] **T2 單次生成（AC #1、#2、#3）**：`solo_runner.go` 加 `soloFailureReason`；`emitTerminal` 失敗段（`:410-418`）加 `data["failure_reason"] = soloFailureReason(run, runErr)`。`solo_runner_test.go` 加 AC #3 的表格測試。
- [x] **T3 舊路徑＋批次（AC #1、#4）**：`failJob`（`transcription_service.go:2361-2368`）加 `"failure_reason": models.SubtitleRunFailureReason(errMsg)`。`GenerationBatchItemState`（`generation_batch.go:144-150`）加 `FailureReason string \`json:"failure_reason"\``；`markItem`（`:837`）多收一個參數（或在 `:800-806` 的 `default` 分支另外設定），`default` 分支傳 `models.SubtitleRunFailureReason(err.Error())`，其他呼叫點傳 `""`；`finish(error)`（`:893-897`）設 `models.SubtitleRunFailureOther`。測試照 AC #4。
- [x] **T4 測試搬家＋註解（AC #5、#6、#7）**：`subtitle_run_failure_test.go` 補前端那份字串；改註解；`go test`／`go vet`／`staticcheck`。

## Dev Notes

> ⚠️ STALE [@contract-v1→v2]: upstream disc-2026-10-activity-subtitle-run-time-a AC #1 bumped 2026-10-19; re-confirm against v2 before dev — see disc-2026-10-run-timeout-not-cancel

### 上游合約已變（`disc-2026-10-run-timeout-not-cancel`，合併順序在本單之前，2026-10-19）

- `failure_reason` 值域變成**十一個**代碼：多了 `queue_timeout`（排隊等抽軌逾時，活動記錄寫「處理逾時」）。AC #1 的「十個代碼」→ 十一個。
- `failItem` 的取消判斷已經拆成兩段：①`errors.Is(cause, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)` → `cancelled: `；②否則 `errors.Is(cause, ErrSubtitleExtractWaitAborted)` → `queue-timeout: `（`QueueTimeoutRunPrefix`）。**T1 抽出的 `isCallerCancellation(cause, ctxErr)` 只能包第 ① 段，不能再含 `ErrSubtitleExtractWaitAborted`**；第 ② 段可另抽 `isQueueTimeout(cause)`。
- AC #2 單次生成的判斷：第 1 步讀 `run.ErrorMessage` 已經會得到 `queue_timeout`。第 2 步「取消類」之後加一步「`errors.Is(runErr, ErrSubtitleExtractWaitAborted)` → `queue_timeout`」（單次生成沒有期限，實際上不會發生，但規則要跟紀錄一致）。
- AC #3 一致性測試的「`ErrSubtitleExtractWaitAborted` → `cancelled`」改成兩條：包 `context.Canceled` → `cancelled`；包 `context.DeadlineExceeded` → `queue_timeout`。
- 「查到的事」第一條的行號、「十個代碼」與規則順序（cancelled 開頭）是 v1 時的樣子，開工時以 `origin/main` 為準。
- 分類函式的第一條規則現在是 `queue_timeout`（排在 `cancelled` 前面，接住舊的 `cancelled: …WAIT_ABORTED…deadline exceeded` 紀錄）；本單「不要改規則或順序」指的是這個 v2 版本。

### 不要做的事

- 不要拿掉或改寫 `error`、`message`：詳情對話框還在用（`GenerationProgressV2.tsx:94`）。
- 不要在後端送中文原因文字（文字只在前端 `FAILURE_COPY_BY_REASON`）。
- 不要改 `models.SubtitleRunFailureReason` 的規則或順序（那是活動記錄的合約，A 單 AC #1）；本單只是**多一個地方用它**。如果 AC #5 搬測試時發現前端認得、後端不認得的字串，記下來，不要順手改規則——要改就是兩邊（活動記錄＋即時活動）一起變，需要另開單。
- 不要讓 `failItem` 回傳 outcome 來帶 run（會改到 `ProcessItem` 所有呼叫者的語意）；用 AC #2 的順序判斷即可。
- `transcription_complete`、`subtitle_progress`、`subtitle_run_receipt` 不加這個鍵（不在裁定範圍；`subtitle_run_receipt` 是另一份合約，`sse/hub.go` 的說明）。

### 已知但不處理

- 舊路徑「`save SRT: open …: permission denied`」其實也是寫不進資料夾，但後端規則只認 placer（`subtitle_run_failure.go:45-53`），活動記錄同樣寫「生成失敗」。兩邊一致，本單不改；要改另開單（影響活動記錄）。
- `AI_INVALID_RESPONSE` 落在 `other`（⚖️ Sally 2026-10-19 追認 4）。

### 測試標準

- Go：testify；表格測試；不打真的 AI。staticcheck 一定要跑（`lint:all` 不含它）。

### E2E data-testid／視覺基準

- 後端單，**不影響**任何 testid、E2E、視覺基準。前端 E2E 的 SSE stub（`tests/e2e/generation-workspace*.spec.ts`）沒有送 `transcription_failed`（`git grep -n "transcription_failed" origin/main -- tests` 無命中），不用改。

### Project Structure Notes

- 動到：`apps/api/internal/subtitle/{process_item.go,solo_runner.go,solo_runner_test.go}`、`apps/api/internal/services/{transcription_service.go,generation_batch.go,generation_batch_test.go}`（＋舊路徑事件測試，放在既有的 `transcription_*_test.go` 之一）、`apps/api/internal/models/{subtitle_run_failure.go,subtitle_run_failure_test.go}`、`apps/api/internal/sse/hub.go`、`_bmad-output/implementation-artifacts/sprint-status.yaml`。

### References

- [Source: apps/api/internal/models/subtitle_run_failure.go:11-92（2bb2c102）]
- [Source: apps/api/internal/subtitle/solo_runner.go:240-260, 346-362, 374-466]
- [Source: apps/api/internal/subtitle/process_item.go:833-877；errors.go:49]
- [Source: apps/api/internal/services/transcription_service.go:2355-2369]
- [Source: apps/api/internal/services/generation_batch.go:144-150, 740-806, 837, 880-910]
- [Source: apps/api/internal/sse/hub.go:35-46]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:38-46, 90-121；generationEventCopy.spec.ts:12-80]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:428-447, 507-522]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-a.md AC #1 `[@contract-v1]`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev agent，2026-10-19）

### Debug Log References

- `go test ./...`、`go vet ./...`、`~/go/bin/staticcheck ./...`（apps/api）全綠。
- 故意改壞驗證：①`emitTerminal` 改成只分類 `error` 文字 → 「原因只寫 signal: killed 的取消」「排隊逾時但文字沒寫 deadline」兩條變紅；②三個出口拿掉截斷 → 三支「關鍵字落在截斷點後面」的測試各自變紅。

### Completion Notes List

**開工前以 `origin/main` 583b5a3e（#843 合併後）重新確認的合約與差異（story 與程式不一致處以程式為準）：**
- 值域是**十一個**代碼（v2 多 `queue_timeout`），分類規則順序 queue_timeout → cancelled → budget → credit → target_not_writable → no_audio → ffmpeg → asr_unconfigured → unauthorized → timeout；未改規則或順序。
- T1 照 v2 拆成兩個：`isCallerCancellation(cause, ctxErr)`（只含 Canceled 兩條）與 `isQueueTimeout(cause)`（`ErrSubtitleExtractWaitAborted`），`failItem` 改呼叫兩者，行為不變。
- `soloFailureReason` 順序：row 的 `ErrorMessage` → 取消 → 排隊逾時 → 額度 → 預算 → 分類 `runErr` 文字。
- 批次鍵清單測試實際在 `generation_batch_items_test.go`（`TestGenerationBatchItems_SSEKeySet`），不是 story 寫的 `generation_batch_test.go:275-291`；未動，另加 `TestGenerationBatchItems_ItemWireKeySet` 驗 `changed_item`／`items[]` 每一項的鍵。
- `markItem` 多一個 `failureReason` 參數（呼叫點與一支既有測試跟著改）。

**story 沒寫、這次多做的（對抗式自我審查抓到，範圍內）：**
- **截斷一致**：紀錄存的 `error_message` 會被切在約 1 KB（管線 1000、舊路徑 1024 bytes）。ffmpeg stderr 很長時，唯一的關鍵字（例如結尾的 `context deadline exceeded`）會被切掉，活動記錄就寫「生成失敗」；事件若分類完整字串會寫「處理逾時」，兩邊又不一致。三個出口現在都分類「跟紀錄一樣切過」的文字（單次用 `truncateErrorMessage`、舊路徑與批次用 `truncateLedgerMessage`），各補一條會變紅的測試。額度／預算看錯誤型別，不受截斷影響（實際訊息裡代碼都在前幾百 bytes）。
- **202 回應也多這個鍵**：`POST /subtitles/generation-batch` 的 `progress.items[]` 用同一個型別，所以也帶 `failure_reason`（`""`）；`generation_batch_handler_test.go` 的鍵清單跟著加。B 單在 `GenerationBatchItemState` 加 `failureReason?` 就涵蓋。

**AC #5 搬測試結果：** 前端 `describe('failureCopy')` 的 19 個輸入在 Go 端（`TestSubtitleRunFailureReason_LiveLogStrings`）**全部**得到跟前端文字對應的代碼，沒有不一致。

**其他觀察：**
- 單次生成「沒有字幕來源也沒設語音辨識」（skip）也送 `transcription_failed`，`failure_reason` 是分類路由理由的結果 → `other`（跟今天前端的「生成失敗」相同）。活動記錄對 skip 不給原因代碼，所以這一種沒有「跟紀錄比」的對象。
- 舊路徑 `save SRT: … permission denied` 維持 `other`（Dev Notes 已知、兩邊一致）。

### File List

- apps/api/internal/subtitle/process_item.go
- apps/api/internal/subtitle/solo_runner.go
- apps/api/internal/subtitle/solo_runner_test.go
- apps/api/internal/services/transcription_service.go
- apps/api/internal/services/transcription_event_title_test.go
- apps/api/internal/services/generation_batch.go
- apps/api/internal/services/generation_batch_items_test.go
- apps/api/internal/handlers/generation_batch_handler_test.go
- apps/api/internal/models/subtitle_run_failure.go
- apps/api/internal/models/subtitle_run_failure_test.go
- apps/api/internal/sse/hub.go
- _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-a.md
- _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-b.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-20 | [@contract-v1→v2] AC #1: failure_reason 多一個值 no_text_source（沒有可用的字幕來源、也接不起語音辨識的 skipped 紀錄；單次 transcription_failed 從 other 改成它），批次 reason=skipped 的項目改帶 no_text_source／asr_unconfigured／""（原本一律 ""）；下游：前端 RunFailureReason／FAILURE_COPY_BY_REASON 要多一個鍵、useGenerationJobsFeed 要保留 skipped 的代碼，舊前端照舊顯示「生成失敗」／「沒有可用的字幕來源」（disc-2026-10-no-text-source-skip-a） |
