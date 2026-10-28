# Story disc-2026-10-solo-generation-cancel-api：單集生成開跑後可以停下來（後端 API）

Status: done

**Source:** sprint-status `disc-2026-10-solo-generation-cancel-api`（傘狀 `disc-2026-10-solo-generation-cancel`）。⚖️ Alexyu 2026-10-28 選 A：一次停一集，不做「停整季」；2026-10-29 選 A：Pencil 額度用完，先做不用畫面的後端。畫面（F3 停止按鈕、F3c 確認框、F3w 停止中、F3s 已停止、K6 活動列停止）在 `-design` → `-dialog` / `-activity`，**不在本單**。不做：暫停／續跑、退款、legacy 模式、翻片內英文字幕路線的逐批保留（`backlog-translate-budget-partial-progress`）。

## 查到的事（2026-10-29，Bob 逐一開檔確認；Winston 的說法逐條驗證）

### 正式機走哪條路

- 正式機 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`（`disc-2026-10-solo-live-spend.md:11` 的 NAS 查證）。pipeline 模式下 `main.go` 建 `SoloRunner` 並 `SetSoloGenerator`（`apps/api/cmd/api/main.go:962`, `:1267-1268`）；legacy 模式 `subtitleSoloRunner` 是 nil，點擊走 `TranscriptionService.StartTranscription`（`apps/api/internal/handlers/transcription_handler.go:204-240`）。

### Winston 1：「每個單集工作已經有可取消的 runCtx，只是 cancel 沒存起來」→ **成立**

- `solo_runner.go:276-277`：`runCtx, cancel := context.WithCancel(ai.WithBudget(context.Background(), job.Budget))` + `defer cancel()`，cancel 只在 goroutine 內，`soloJob`（`:90-98`）沒有欄位存它。工作登記在 `s.jobs[ref] = job`（`:263`），結束時先刪 job（`:294`）、送終點事件（`:298`）、最後才 `s.guard.Release(ref)`（`:299`）。

### Winston 2：「每個階段都能停」→ **成立，細節如下**

| 階段 | 停的時候發生什麼 | 證據 |
|---|---|---|
| 排隊等抽取位子（字幕抽取 gate） | gate 等待被 ctx 打斷 → `ErrSubtitleExtractWaitAborted` 包 `context.Canceled`；failItem 判斷「取消優先於逾時」→ `cancelled: ` 前綴；$0 | `subtitle/extractor.go:355`, `:411`；`process_item.go:907-925` |
| 排隊等音訊抽取位子（ASR 支線） | semaphore／slot 等待回 `ctx.Err()` | `services/audio_extractor_service.go:204-226` |
| 抽音訊（ffmpeg） | `exec.CommandContext` 被殺、暫存 WAV 刪掉、錯誤包 `ctxErr` | `audio_extractor_service.go:245`, `:263`, `:268-271` |
| 管線自己的暫存資料夾 | `defer os.RemoveAll(tmpDir)` | `process_item.go:154-158` |
| 聽寫（ASR） | 每段付費結果邊做邊存 `storeChunk`，下次 `loadStoredChunks` 接著用；只有英文 SRT 存好才 `clearChunkCache` | `services/transcription_service.go:1635`, `:1679`, `:1153` |
| 翻譯（ASR 路線） | 每句付費翻譯進 `segmentStore`；ctx 結束時先把 `untranslated`＋英文 SRT 路徑寫回（續跑的鑰匙）再回錯 | `transcription_service.go:149-152`, `:1430-1438` |
| 翻譯（翻片內英文字幕路線） | 已翻的句子丟掉（片段快取只在整條翻完才寫）——已知限制，交給 `backlog-translate-budget-partial-progress` | `process_item.go:86-99` |
| 放字幕檔 | 原子寫入（tmp → rename），不會有半個檔 | `subtitle/placer.go:230` |

### Winston 3：「不用 migration」→ **成立**

- 取消已經存成 `status=failed` + `cancelled: ` 前綴：`process_item.go:921-922`（`isCallerCancellation(cause, ctx.Err())`，`:953-955`）；前綴常數 `models/subtitle_run_failure.go:101`；分類器 `cancelled` 規則 `:77-79`。
- 活動「最近」記錄：`activity_service.go:546-551` 帶 `spent_usd` 與 `failure_reason`；前端 `cancelled` → 「中途停止」（`apps/web/src/components/activity/recentEventCopy.spec.ts:131`、`generationEventCopy.spec.ts:571`）。
- 花費：failItem 在寫終點前呼叫 `stampRunSpend`（`process_item.go:928`）。

### Winston 4：「cancel 到釋放 single-flight 之間，再按生成會 409」→ **成立**；目前 409 的訊息是「已在執行中」（`transcription_handler.go:279-286`），看不出「正在停止」。

### Winston 5（⚠️ 風險）：「failItem 把字幕狀態清成 not_searched」→ **成立，而且比 Winston 說的多一個坑**

- `failItem` 一律 `setMediaStatus(cleanupCtx, ref, not_searched, "", "")`（`process_item.go:935`）。重新生成被停掉時，舊字幕檔還在，紀錄卻說沒有。
- **多出來的坑**：ASR 路線在翻譯中途被停，`TranscriptionService` 先寫回 `untranslated`＋英文 SRT 路徑（`transcription_service.go:1430-1438`），這是下次「只翻譯、不再付聽寫費」的唯一鑰匙（`resumeSource` 要求 `status == untranslated && path != ""`，`:1242`）。接著錯誤傳回管線 → `failItem` 把它清成 `not_searched`、路徑清空 → **下次生成要重付整段聽寫費**（英文 SRT 存好那刻 chunk 快取已清，`:1153`）。如果照 Winston 原句「一律還原成 Load 時的值」，同樣會蓋掉這把鑰匙。
- `restoreMediaStatus`（`process_item.go:969-981`）只在 Load 時狀態合法時寫回 status／path／language，用 `WithoutCancel`。

### Winston 6：「已花的錢照記在執行紀錄」→ **成立**（`stampRunSpend`，`process_item.go:660-686`）。

### 其他查到的

- `soloFailureReason` 目前用 `isCallerCancellation(runErr, nil)`，註解寫「solo 的 ctx 只會在 ProcessItem 回來後才取消」（`solo_runner.go:500-504`）——有了取消就不成立：ffmpeg 被殺回 `signal: killed`、不帶 `context.Canceled` 時，事件會說 `other` 而執行紀錄說 `cancelled`。要傳 `runCtx.Err()`。
- 終點失敗事件的文字目前一律「字幕生成失敗」（`solo_runner.go:556`）。
- 放檔之後的兩個寫入（`recordTerminal` `process_item.go:314`、標 found `:337`；ASR 路線 `:618`）用的是會被取消的 ctx：在「檔案已放好」之後才按停，寫入會失敗 → failItem 把一個**已經放好新字幕**的執行記成「中途停止」。
- 批次取消的前例：`generation_batch_handler.go:268-290`（`{cancelled, running}`，沒在跑回 `cancelled=false`；swagger 註解格式）。`GenerationBatchProcessor.Cancel` 只呼叫存起來的 `activeCancel`（`services/generation_batch.go:475-481`）。
- single-flight 是 pool 的 in-flight 集合，solo／批次／pool worker／免費自動線共用（`main.go:957-962`；`cmd/api/generation_batch_runner_adapter.go:82-88`）；SoloRunner 只能分辨「是不是我的 job」，分不出是批次還是 pool。
- SSE 文件 `docs/sse-event-types.md:170-177` 只提到 transcription_* 帶花費，沒有 transcription_failed 專節；`docs/swagger.yaml` 不含 `generation-batch/cancel`（swagger 沒在重產），所以 API 文件 = handler 的 swag 註解。
- 前端：`ManageSubtitleDialogV2` 第 51 行「後端沒有取消 API」註解由 `-dialog` 更新；本單不動前端。

## 設計（Bob × Winston，architecture-first）

**端點**（只在 pipeline 模式真的會停；legacy 回 `cancelled=false`）：

- `POST /api/v1/movies/:id/transcribe/cancel`
- `POST /api/v1/episodes/:id/transcribe/cancel`（跟 status 路由一樣，只在 episode getter 有接時掛）
- 不查資料庫、不確認片子存在（同 status 路由）；未知 id = 沒在跑。
- 回 200 `APIResponse.data`：

| 欄位 | 型別 | 意義 |
|---|---|---|
| `cancelled` | bool | `true` = 這部片的單集生成（詳細頁按的那種）收到停止，正在收尾；重複按照樣回 `true`（冪等）。`false` = 沒有可停的單集生成。 |
| `running` | bool | 這次呼叫之後，這部片是否還有東西在生成（含剛被停、還在收尾的那個工作）。 |
| `owner` | string，選填 | 只在 `running=true` 時出現：`"solo"`＝詳細頁單集生成（本端點能停）；`"other"`＝批次／pool／免費自動線（本端點不停；批次請用 `POST /subtitles/generation-batch/cancel`）；`"legacy"`＝legacy 模式的單集生成（不支援停止）。 |

**SoloRunner**：`soloJob` 多存 `cancel context.CancelFunc` 與 `Cancelling bool`（`s.mu` 保護）；runCtx 在 `Start` 裡、放進 map **之前**建立。新方法 `Cancel(ref) SoloCancelResult{Cancelled, Running bool; Owner string}`、`Cancelling(ref) bool`。被停的工作再按生成：`Start` 回 `ErrSoloGenerationCancelling`（包 `services.ErrTranscriptionInProgress`，舊的 `errors.Is` 照樣成立）→ handler 回 409 **同一個 code** `TRANSCRIPTION_IN_PROGRESS`、訊息改「正在停止上一次的生成，請過幾秒再按」。status 回應加 `cancelling`（bool，omitempty）。

**終點事件**：沿用 `transcription_failed`，不加新事件種類。被停時 `failure_reason: "cancelled"`、`message: "已停止生成"`（F3s 第一行），`error` 照舊放診斷字串。`soloFailureReason` 改收 `runCtx.Err()`。若停止來得太晚、工作其實已完成，送正常的 `transcription_complete`（終點事件才是真相）。

**字幕狀態（⚖️ Bob×Winston 裁定，改良 Winston 5）**：新選項 `ProcessItemOptions.RestoreMediaOnCancel`（additive，SoloRunner 設 true；批次／pool／自動線不變，`TestProcessItem_CancellationStillRecordsTheFailure` 釘的 not_searched 行為保留）。取消時 failItem 不寫 not_searched，改成：
1. 重讀這部片目前的狀態；
2. 若還是管線自己寫的「進行中」狀態（`probing`／`extracting`／`translating`），或讀不到 → 還原成 Load 時的 status／path／language（`restoreMediaStatus`；Load 時本身就是進行中或不合法 → `not_searched`）；
3. 若已經被某條支線寫成結論（例：ASR 支線寫的 `untranslated`＋英文 SRT 路徑）→ **保留**，不蓋掉續跑的鑰匙。

**放檔之後不可取消**：Place 成功後的 `recordTerminal`／標 found（ASR 路線：Transcribe 成功後的 `recordTerminal`）改用 `context.WithoutCancel`——檔案已經在磁碟上，這次就是完成。

**Legacy 模式**：handler 回 `{cancelled:false, running: IsInProgress(id), owner:"legacy"（在跑時）}`，不 panic、不碰 TranscriptionService。

## Story

身為在詳細頁按了「生成字幕」才發現按錯片、或不想再花錢的人，
我希望能把這一集的生成停下來，
這樣已經花的錢有紀錄、已經付費辨識的段落下次接著用、這部片原本的字幕紀錄也不會被清掉。

## Acceptance Criteria

1. **端點與契約 `[@contract-v1]`：** `POST /api/v1/movies/:id/transcribe/cancel`、`POST /api/v1/episodes/:id/transcribe/cancel` 回 200 `APIResponse`，`data` = `{cancelled: bool, running: bool, owner?: "solo"|"other"|"legacy"}`（語意見設計表）。沒在跑 → `{cancelled:false, running:false}`（無 `owner`）。重複呼叫冪等。
2. **只停自己的：** 這部片被批次／pool 佔著（guard in-flight 但不是 SoloRunner 的 job）→ `{cancelled:false, running:true, owner:"other"}`，那個工作不受影響。
3. **每個階段都會停，執行紀錄一致**（pipeline 模式，SoloRunner 工作）——以下每一個階段被停後：執行紀錄 `status=failed`、`error_message` 以 `cancelled: ` 開頭（→ `failure_reason=cancelled`）、`spent_usd` = 停下時這個工作已花的錢；工作從 `ActiveSoloJobs` 消失；single-flight 釋放（之後再按生成會 202）：
   - a. 排隊等抽取位子（$0）；
   - b. 抽音訊／抽字幕（子程序因取消結束，含只回 `signal: killed` 的情況）；
   - c. 聽寫（ASR 支線回錯，已付費的段落留在 chunk 快取——由既有程式保證，本單只驗證錯誤分類）；
   - d. 翻譯（翻片內英文字幕路線：翻譯器回 `context.Canceled`）；
   - e. 放檔之後才停 → 這次記成 **完成**（`completed`，送 `transcription_complete`），不是中途停止。
4. **終點事件：** 被停的工作最後一個帶它 `job_id` 的事件是 `transcription_failed`，`failure_reason: "cancelled"`、`message: "已停止生成"`、帶 `spent_usd`（`budget_usd` 有上限才帶）；`soloFailureReason` 與執行紀錄的分類一致（`signal: killed` 也是 `cancelled`）。不新增事件種類。
5. **字幕狀態不被清掉：** 被停時
   - a. 重新生成（Load 時 `found` + 舊路徑 + `zh-Hant`）→ 結束後狀態回到 `found` + 舊路徑 + `zh-Hant`；
   - b. ASR 支線已寫回 `untranslated` + 英文 SRT 路徑 → 保留，不被蓋掉；
   - c. Load 時狀態不合法／是進行中狀態 → `not_searched`（舊的底線）。
   - d. 批次／pool／自動線的取消行為**不變**（仍是 `not_searched`）。
6. **停止中再按生成：** 停止送出、工作還沒收尾前，`POST …/transcribe` 回 409 `TRANSCRIPTION_IN_PROGRESS`、訊息「正在停止上一次的生成，請過幾秒再按」；`GET …/transcribe/status` 多 `cancelling: true`（omitempty，additive）。
7. **Legacy 模式：** 沒有 SoloGenerator 時端點仍掛著，回 `{cancelled:false, running:<legacy 是否在跑>, owner:"legacy"（在跑才有）}`；不呼叫任何會停工作的東西、不 panic。
8. **文件：** 兩個 handler 加 swag 註解（格式同 `CancelGenerationBatch`）；status 的 swag 註解補 `cancelling`；`docs/sse-event-types.md` + `.zh-TW.md` 補一段「單集生成被停止 → transcription_failed + failure_reason=cancelled」。
9. **品質：** 新測試跑 `-race`；`go test ./...`、`go vet ./...`、`staticcheck-2026.1 ./...` 全綠。

## Tasks / Subtasks

- [x] T1 管線：取消時不清字幕狀態、放檔之後不可取消（AC #3e, #5）
  - [x] `ProcessItemOptions.RestoreMediaOnCancel`；`processScope` 記 `loaded *MediaItem`／`restoreOnCancel`
  - [x] failItem 取消分支：`settleCancelledMedia`（重讀 → 進行中則還原、已有結論則保留）
  - [x] Place 後與 ASR 成功後的終點寫入用 `WithoutCancel`
  - [x] 測試（新檔 `process_item_cancel_test.go`，含會反映寫入的 fake media store）
- [x] T2 SoloRunner：存 cancel、`Cancel`／`Cancelling`、停止中 409 sentinel、終點事件（AC #2, #3, #4, #6）
  - [x] `soloFailureReason(run, runErr, ctxErr)`；被停時 `message = "已停止生成"`
  - [x] 測試（新檔 `solo_runner_cancel_test.go`，各階段 a–e，`-race`）
- [x] T3 Handler：兩條 cancel 路由、legacy、409 文字、status `cancelling`、swag（AC #1, #6, #7, #8）
  - [x] 測試（handler fake 加 `Cancel`／`Cancelling`）
- [x] T4 文件：`docs/sse-event-types.md`／`.zh-TW.md`（AC #8）
- [x] T5 全量驗證（AC #9）

（後端 5 個 task、前端 0 → 不觸發跨棧拆分；規模小於 `disc-2026-10-solo-live-spend`，不拆。）

## Dev Notes

### 已知陷阱

- runCtx 必須在 job 放進 `s.jobs` **之前**建立並存好 cancel，否則 `Cancel` 可能拿到 nil cancel。
- `Cancel` 在 `s.mu` 下讀 job、設 `Cancelling`、呼叫 cancel；cancel 本身冪等。
- 重讀狀態用 `p.media.Load`（只在取消時才多一次讀）；fake media store 的 `Load` 固定回原 item，測試要用會套用寫入的 fake。
- `failItem` 的 `cleanupCtx` 已是 `WithoutCancel`（`process_item.go:894`），新的還原寫入也要用它。
- 不要改 `TestProcessItem_CancellationStillRecordsTheFailure` 的期待（批次／pool 行為不變）。
- handler 的 409 沿用 code，前端 `parseTranscribeResponse` 不用改。

### 不要做的事

- 不加新 SSE 事件種類、不改 D6 stage 詞彙。
- 不碰 legacy `TranscriptionService` 的 solo 路徑（不加取消）。
- 不改批次取消；不改前端（API client 函式留給 `-dialog`）。
- 不 migration。

### 測試

- `cd apps/api && go test -race ./internal/subtitle/ -run 'Cancel' && go test -race ./internal/handlers/ -run 'Cancel|Solo|Status'`
- `go test ./... && go vet ./... && ~/go/bin/staticcheck-2026.1 ./...`

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

### References

- `apps/api/internal/subtitle/solo_runner.go`、`process_item.go`、`pipeline.go:144-200`（ProcessItemOptions）
- `apps/api/internal/handlers/transcription_handler.go`、`generation_batch_handler.go:268-290`
- `apps/api/internal/services/transcription_service.go:1390-1490`、`audio_extractor_service.go:195-290`
- `apps/api/internal/models/subtitle_run_failure.go`
- `_bmad-output/implementation-artifacts/pen-prompts-solo-progress-cancel.md`（F3s 文案「已停止生成」）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

- `go test -race ./internal/subtitle/ -run 'Cancel|Solo' -count=5` → ok
- `go test -race ./internal/handlers/ -run 'Cancel|Solo|Status|Transcri'` → ok
- `go test ./...`（apps/api）→ 44 個套件 ok、0 失敗
- `go vet ./...` 乾淨；`~/go/bin/staticcheck-2026.1 ./...` 零輸出
- 前端沒動 → 沒跑 `pnpm nx test web`／`lint:all`（照 story「不改前端」）

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- T1 管線：`ProcessItemOptions.RestoreMediaOnCancel`（additive）＋ `processScope.loaded/restoreOnCancel`；failItem 在「呼叫端取消 + 選項開」時改走 `settleCancelledMedia`（重讀 → 仍是 probing/extracting/translating/searching 或讀不到 → 還原 Load 時的值；已被支線寫成結論 → 保留；Load 時本身是進行中／不合法 → not_searched）。Place 成功後與 ASR 支線成功後的終點寫入改用 `context.WithoutCancel`（之後的 failItem 也用它，避免把真的 DB 錯誤誤判成取消）。
- T2 SoloRunner：runCtx 移到 `Start`、放進 map 之前建立並存 `job.cancel`；`Cancel`／`Cancelling`；`ErrSoloGenerationCancelling`（包 `ErrTranscriptionInProgress`）；ProcessItem 帶 `RestoreMediaOnCancel: true`；終點事件 `soloFailureReason(run, runErr, runCtx.Err())`，被停時 `message = "已停止生成"`。
- T3 Handler：`POST /movies/:id/transcribe/cancel`、`POST /episodes/:id/transcribe/cancel`（episode 只在 getter 有接時掛）；`TranscriptionCancelResponse{cancelled, running, owner?}`；legacy 回 `owner:"legacy"`；停止中 409 同 code 不同字；status 加 `cancelling`（omitempty）；swag 註解。
- T4 文件：`docs/sse-event-types.md`／`.zh-TW.md` 加「單集生成被停止」一段。
- 🔗 AC Drift: NONE（checked: `not_searched (retryable)`／`back to \`not_searched\`` across `_bmad-output/implementation-artifacts/*.md` — 1 hit `sub-2-2b-untranslated-badge-frontend.md`，是 REUSE：預設行為不變，新行為只在 solo 的選項下）。ProcessItem 註解第 5 步補了選項的例外。
- 📎 Contract Stamps: FOUND（本單 AC #1 `[@contract-v1]` 新契約；上游 ProcessItem `[@contract-v1]`（sub-1-3/sub-1-6）只加選填欄位，FreeOnly／Regenerate 前例 additive 不升版；status 回應 additive）。
- 🎭 A11y Pre-Flight: N/A（100% backend — no apps/web/ files touched）
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- E2E／gallery 掃描：`grep -rn "transcribe/cancel\|cancelling" tests/e2e tests/visual apps/web/src/routes/test` 無命中（新端點、新欄位，沒有既有測試釘舊行為）。

### Discovery Triage

- ③ `backlog-batch-cancel-clears-subtitle-record`（新建）：批次／pool／自動線的失敗（含取消、翻譯逾時）仍把字幕紀錄清成 not_searched，會蓋掉聽寫支線寫的續跑鑰匙。本單只修單集點擊。
- ③ `disc-2026-10-solo-generation-cancel-activity`（既有條目加註）：活動 active_jobs 還沒有「停止中」欄位，K6 若要畫要在那張加。
- ③ `disc-2026-10-solo-generation-cancel-dialog`（既有條目加註）：前端交接（cancelling、聽寫支線的 NOTE 事件、owner=other 的極短窗口、API client 函式）。

### File List

- `apps/api/internal/subtitle/pipeline.go`
- `apps/api/internal/subtitle/process_item.go`
- `apps/api/internal/subtitle/process_item_cancel_test.go`（新）
- `apps/api/internal/subtitle/solo_runner.go`
- `apps/api/internal/subtitle/solo_runner_cancel_test.go`（新）
- `apps/api/internal/handlers/transcription_handler.go`
- `apps/api/internal/handlers/transcription_solo_handler_test.go`
- `apps/api/internal/handlers/transcription_cancel_handler_test.go`（新）
- `apps/api/internal/subtitle/segment_cache.go`
- `apps/api/internal/subtitle/cancel_commit_test.go`（新）
- `apps/api/internal/services/transcription_service.go`
- `apps/api/internal/services/transcription_cancel_commit_test.go`（新）
- `docs/sse-event-types.md`
- `docs/sse-event-types.zh-TW.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/disc-2026-10-solo-generation-cancel-api.md`

### Senior Developer Review (AI) — 2026-10-29

**結果：Approve（in-scope 問題已修）。** Git vs File List：0 差異（`solo_runner_test.go` 修完後回到原樣，已從清單移除）。

- 🔒 Rule 7 Wire Format: PASS（新增 1 個錯誤碼字串 `SUBTITLE_SOLO_CANCELLING`，前綴 `SUBTITLE_` 合法；對外 code 沿用 `TRANSCRIPTION_IN_PROGRESS`）
- 🔒 Rule 20 Contract Bump: N/A（no stamp bumps in this review；AC #1 是新的 v1，其餘 additive）
- 🔒 Rule 25 Mega-line: N/A（project-context.md 沒動）

| # | 嚴重度 | 發現 | 處理 |
|---|---|---|---|
| H1 | High | `soloFailureReason` 用 `runCtx.Err()` 判斷取消：若工作已經因別的原因失敗（例：資料夾不能寫），停止請求剛好在 failItem 記完之後、ProcessItem 回來之前到，事件會說 `cancelled`／「已停止生成」，執行紀錄卻是 `target_not_writable`——活動頁和對話框說的不一樣 | **已修**：failItem 只判斷一次 `callerCancelled`，判成取消時把 `context.Canceled` 包進回傳的錯誤；`soloFailureReason` 回到只看錯誤（不看 ctx）。測試 `TestSoloRunner_LateCancelDoesNotRelabelAnEarlierFailure`、`TestProcessItem_CancelledVerdictIsInTheReturnedError` |
| M1 | Medium | failItem 裡「是不是取消」問了兩次（run row 一次、media row 一次），理論上可能不一致 | **已修**（併入 H1：一次判斷，三處共用） |
| M2 | Medium | 放檔之後的 failItem 若沿用會被取消的 ctx，真的 DB 錯誤會被誤標成「中途停止」 | **已修**（dev 階段即改：放檔後的 failItem 用 `commitCtx`），`TestProcessItem_CancelAfterThePlaceStillCompletes` 釘住 |
| L1 | Low | 工作已從 map 移除、還沒放開 single-flight 的極短窗口，Cancel 會回 `owner: "other"` | 接受；寫進 `-dialog` 交接（當成還在停） |
| L2 | Low | 聽寫支線自己的 `transcription_failed`（英文訊息）會先到，對話框當 NOTE | 不改後端（既有契約：只有帶自己 job_id 的才是終點）；寫進 `-dialog` 交接 |
| L3 | Low | 活動列沒有「停止中」狀態 | ③ 加註到 `-activity` |
| L4 | Low | 批次取消仍清字幕紀錄 | ③ `backlog-batch-cancel-clears-subtitle-record` |

驗證：`go test -race ./internal/subtitle/ ./internal/handlers/ -run 'Cancel|Solo|Status|Transcri'` ok；`go test ./...` 44 個套件 ok、0 失敗；`go vet` 乾淨；`staticcheck-2026.1` 零輸出。新增測試 21 個（process_item_cancel 7、solo_runner_cancel 8、handler 6）。

### Review follow-up（coordinator，2026-10-29）

| # | 嚴重度 | 發現 | 處理 |
|---|---|---|---|
| R1 | Medium | 聽寫路線：`translateSRT` 放好 `.zh-Hant.srt` 後，`harvestGlossaryTerms` 與 `translateAndPersist` 的結論寫入（found）仍用會被取消的 ctx；停止落在這裡 → 寫入失敗 → 支線回錯 → failItem 判取消、`settleCancelledMedia` 把紀錄還原成跑之前，事件說「已停止生成」，但新字幕已在磁碟上、chunk 快取已清 | **已修**：`translateSRT` 放檔後、`translateAndPersist` 的結論寫入改用 `WithTimeout(WithoutCancel(ctx), commitWriteTimeout=10s)`（同既有逾時寫回的上限）；支線回成功 → 管線記 completed。測試 `TestTranslateAndPersist_StopAfterThePlaceStillWritesTheVerdict` |
| R2 | Low | 付費呼叫回來後的快取寫入用活的 ctx：管線 `storeCachedCues`、聽寫 `storeChunk`；停止剛好落在這裡會丟掉已付費的結果，下次再付一次 | **已修**：兩個函式內部改用 `WithoutCancel`。測試 `TestProcessItem_PaidCuesAreCachedDespiteAStopAfterTheCall`、`TestStoreChunk_SurvivesTheStopThatLandedAfterThePaidCall` |
| R3 | Low | SoloRunner 在送終點事件、放開 single-flight 之前就把 job 從 map 刪掉；這段時間 Cancel 回 `owner:"other"`、Start 回一般的「已在執行中」（= 先前 CR 的 L1） | **已修**：順序改成 Untrack（仍在終點事件之前，PR #935 不變）→ 終點事件 → 刪 job（仍檢查 `cur == job`）→ Release。代價：終點事件送出到刪除之間（同一個 goroutine 內、微秒級）`ActiveSoloJobs` 仍列著它。測試 `TestSoloRunner_CancelBetweenTerminalAndReleaseStillSeesOurJob`；既有 `TestSoloRunner_TerminalIsEmittedBeforeTheClaimIsReleased`、`CleanupOnlyForgetsItsOwnJob` 照過 |
| R4 | Note | 聽寫支線自己的 failJob 會先送一則帶支線 job_id 的 `transcription_failed`（例「Transcription failed: extract audio: signal: killed」、failure_reason other）；`useGenerationJobsFeed` SINGLE_FAILED 沒看 job_id | 不改；寫進 `disc-2026-10-solo-generation-cancel-dialog` 交接（UI 必須以 job_id 判斷終點） |

先前 CR 表的 L1 由 R3 取代（已修）、L2 的交接文字由 R4 更正。

驗證（follow-up 後）：`go test ./...` 44 個套件 ok、0 失敗；`go test -race` 的 subtitle／handlers 都 ok；services 用 `-race` 跑時，只有計時測試 `TestSegmentStore_ReadStaysFastAgainst200kRows` 失敗（race 檢測讓速度變慢，128ms 超過 100ms 上限；這個檔案本單沒動，不加 `-race` 時 1.3 秒就過）。排除它之後，services `-race` ok；`go vet` 乾淨；`staticcheck-2026.1` 零輸出。新增測試累計 26 個。

## Change Log

| Date | Change |
|---|---|
| 2026-10-29 | Story 建檔（Bob），ready-for-dev |
| 2026-10-29 | T1–T5 實作完成（停止端點、各階段停止、字幕狀態不被清、放檔後不可取消、文件），review |
| 2026-10-29 | CR：H1／M1 修（取消判斷一次、包進錯誤），L1–L4 記錄／立案；done |
| 2026-10-29 | Review follow-up：R1（聽寫支線放檔後不可取消）、R2（付費結果的快取寫入不可取消）、R3（終點事件後才刪 job）修好並加測試；R4 寫進 -dialog 交接 |
