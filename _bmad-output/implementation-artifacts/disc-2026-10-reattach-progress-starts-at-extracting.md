# Disc：重開「管理字幕」接上進行中的生成，進度先卡在「抽取音訊」

Status: review

**Source:** 2026-10-05 `bugfix-dialog-reopen-shows-idle-during-run` 建 story 時立案（Rule 24 lane ③，P3）。

## 查到的事（2026-10-12，Bob 逐一開檔確認）

### 實際會看到的

- 生成字幕跑到一半，關掉「管理字幕」再打開（或再按一次生成、拿到 409），視窗會接上進行中的工作，但進度條一律從「抽取音訊」開始。
  - 原因：接上時前端只知道「在跑」，不知道跑到哪。`useGenerationProgress.ts` 的 `START` 一律設成 `extracting`（`apps/web/src/hooks/useGenerationProgress.ts` reducer `case 'START'`）。
  - 要等下一個進度事件來，畫面才會跳到真正的階段。
- **語音辨識那一段特別久：** 聽寫只在開始時送一次 `transcription_progress`（`apps/api/internal/services/transcription_service.go:995`），整段辨識中間沒有任何事件。See S01E02 那集 57 分鐘的聲音要辨識好幾分鐘，這段時間重開視窗，會一直看到「抽取音訊」。
  - 翻譯每一批都會送 `translation_progress`，所以翻譯中重開，幾秒後就會跳對。

### 另一個小窗口：工作剛好結束

- 視窗打開時先問「在跑嗎」（`GET …/transcribe/status`），再連上即時通知（SSE）。
- 如果工作剛好在「問完」和「連上」之間結束，結束通知就錯過了。伺服器不會重送（`sse/hub.go` 沒有 replay），畫面就一直停在進度。
- 同樣的事也會發生在：
  - **按下生成後馬上結束的工作：** 例如片內就有中文字幕、直接用（`deliver_direct`），可能在連上之前就做完了。
  - **斷線重連：** 斷線那 10 秒內結束的工作。
- 現有的補救（`STALE_TERMINAL_PROBE_MS`）只在「看到管線的結束訊號、卻沒看到這份工作的結束訊號」時才會問，上面三種情況都沒看到任何訊號，所以不會觸發。

### 現在的架構（正式機是管線模式）

- 管線模式下，`GET …/transcribe/status` 由 `SoloRunner.Status`（`apps/api/internal/subtitle/solo_runner.go:275`）回答，只回 `in_progress` 和 `job_id`。
- 一次生成會有三種來源送進度：
  - 管線本身：`subtitle_progress`（`stage` 為 probing／extracting／translating，`subtitle/progress_sse.go`）
  - 語音辨識那一段：`transcription_extracting`、`transcription_progress`、`translation_progress`（`services/transcription_service.go`）
  - SoloRunner 自己的開始和結束事件
- 三種都經過同一個 `sse.Hub.Broadcast`（`apps/api/internal/sse/hub.go:180`）。
- **backlog 原本寫的修法方向已過時：** 原本說在 `soloTranscriptionJob` 記住階段。但管線模式下那張表只看得到語音辨識那一段，看不到「抽字幕」和「翻片內英文字幕」。所以改成在 Hub 出口記。

## Story

身為生成字幕到一半、關掉視窗又打開的人，
我希望進度畫面直接顯示它現在真正在做的事（例如「轉錄中」），
而且如果工作剛好在我打開時結束，畫面不要一直卡在進度，
這樣我才知道它還在跑、跑到哪、或已經好了。

## Acceptance Criteria

1. **後端記住每部片「目前的生成階段」：**
   - 新的 `subtitle.PhaseTracker`，從 Hub 的每一個廣播事件中挑出生成進度事件，記下該片（`media_id`）最新的狀態：階段、訊息、翻譯百分比。
   - 階段對照（跟前端 `useGenerationProgress` 的對照一致）：
     - `transcription_extracting` → extracting
     - `transcription_progress` → transcribing
     - `translation_progress` → translating
     - `subtitle_progress` 的 probing／extracting → extracting，translating → translating
   - 遇到結束事件就清掉該片的紀錄：`transcription_complete`、`transcription_failed`，以及 `subtitle_progress` 的 complete／failed／skipped。
   - 沒有 `media_id` 的事件、其他類型的事件、管線的搜尋階段，一律不理。
   - `sse.Hub` 新增 `AddObserver(func(Event))`：每次 `Broadcast` 都會**同步**呼叫，而且在送進頻道**之前**呼叫。所以頻道滿了、事件被丟掉時，紀錄仍然正確。`sse` 套件本身不知道任何生成字詞。
2. **`GET …/transcribe/status` 多回三個欄位**（都是可省略，舊的前端不受影響）：
   - `phase`、`message`、`percentage`（只有 translating 才有）。
   - 只有 `in_progress=true` **而且**有紀錄時才回。不在跑時不回，這樣才不會拿上一次的舊階段。
   - 管線模式、舊模式都適用。
   - 在跑但還沒有任何進度事件時，不回 `phase`，前端照舊從「抽取」開始。
3. **前端用它當起點：**
   - `startTracking(mediaId, jobId, initial?)` 可帶起始的階段、訊息、百分比。
   - 「管理字幕」兩個接上的地方都要帶：打開時就在跑，以及按下生成拿到 409。
   - 帶了 `transcribing`，畫面一打開就是「轉錄中」，不再先顯示「抽取」。
4. **連上之後再問一次：**
   - 即時通知連上（`EventSource` 的 open）後，等 2 秒（`ATTACH_PROBE_MS`，不是畫面上的時鐘），問一次 status。
   - 如果伺服器說沒在跑，而且這段時間沒收到結束事件，就判定為「結束了但沒看到結果」：
     - 前端停止追蹤，回到一般畫面。
     - 刷新字幕清單（呼叫 `onComplete`），讓清單顯示實際結果：做好了就看到新字幕，失敗了就看到原本的狀態和生成鍵。
   - 每一次連上都會問，包括按下生成後的第一次連上、斷線重連。
   - 只有傳了 `probeInProgress` 的使用者（管理字幕視窗）才會問，生成工作區不受影響。
   - 為什麼要等 2 秒：如果工作在連上之後才結束，結束事件幾毫秒內就會送到。等 2 秒能避免「status 說結束了，但結束事件還在路上」時，把真正的結果蓋掉。
5. **測試：**
   - Go：
     - 追蹤器：表格測試涵蓋每一種事件、`map[string]string` 和 `map[string]interface{}` 兩種資料格式、結束清除、陌生事件不理。
     - Hub：observer 會被呼叫，頻道滿了也會被呼叫。
     - handler：在跑而且有紀錄時回 `phase`，沒在跑時不回，兩種模式都測。
   - 前端：
     - hook：`startTracking` 帶 `transcribing` 時一開始就是 transcribing。
     - hook：連上 2 秒後 status 回「沒在跑」，就呼叫 `onComplete`、回到 idle，而且只問一次。
     - hook：收到結束事件之後就不再問。
     - dialog：打開時 status 回 `phase: transcribing`，進度畫面直接是轉錄中。
     - dialog：結束沒看到時，回到字幕清單。
6. **檢查：** `go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm nx test web`、`pnpm run lint:all` 全綠；grep E2E 中 `transcribe/status` 的 mock，確認沒有被新欄位弄壞。

## Tasks / Subtasks

- [x] T1 `sse.Hub.AddObserver`＋測試（AC #1）
- [x] T2 `subtitle.PhaseTracker`＋測試（AC #1）
- [x] T3 handler `SetPhaseReader`、status 回應三個新欄位、`main.go` 接線＋測試（AC #2）
- [x] T4 前端 `TranscriptionStatus` 型別、`startTracking` 起點、dialog 兩處帶起點＋測試（AC #3）
- [x] T5 hook 連上後 2 秒問一次、`onEndedUnobserved`、dialog 回到 idle＋測試（AC #4）
- [x] T6 檢查全綠（AC #6）

## Dev Notes

- **不動設計稿：** 沒有新元件、沒有新字句。「結束了但沒看到結果」時，回到一般的字幕清單，清單本身就會顯示結果。
- **不加資料庫：** 紀錄只放在記憶體。伺服器重開後工作本來就不在了，status 也會回「沒在跑」。
- **紀錄用 `media_id` 當 key，不分電影和單集：** 兩者都是 UUID，實際上不會撞號。事件裡也不一定有 `media_type`（例如語音辨識那一段的事件就沒有）。
- **observer 的成本：** 每一次 Broadcast 都會呼叫，所以只能做一次型別判斷，加上一次有鎖的 map 寫入。不能做 I/O，也不能 log。
- **範圍外：**
  - 生成工作區（`GenerationWorkspaceV2`）接上時也從 extracting 開始，但它看的是批次快照，不在這張的範圍內。
  - 語音辨識中間沒有進度（沒有「第幾段」）是另一件事，這張不加。

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- **後端：**
  - `sse.Hub.AddObserver`：每次 `Broadcast` 都會同步呼叫，而且在送進頻道之前。
  - `subtitle.PhaseTracker`：記住每部片的階段、訊息、百分比，遇到結束事件就清掉。
  - `main.go` 在建好 Hub 之後馬上註冊 tracker，交給 `TranscriptionHandler.SetPhaseReader`。
  - status 回應多了 `phase`／`message`／`percentage`，三個都可省略。只有 `in_progress=true` 而且有紀錄時才回。
- **前端：**
  - `startTracking(mediaId, jobId, from?)`：`START` 從 `from.phase` 開始。不認得的階段，一律從 extracting 開始。
  - hook 在即時通知連上（`onopen`）後等 2 秒問一次 status。回「沒在跑」就回到 idle，並呼叫 `onComplete`（null）和 `onEndedUnobserved`。
  - 「管理字幕」兩個接上的地方都會帶 status 的答案。
  - `onEndedUnobserved` 會先把快取的 status 改成「沒在跑」。不改的話，打開時接上的那段程式會把視窗又拉回進度。改完再回到 idle，並刷新報價和字幕清單。

### 驗證

- **Go：** `go vet ./...`、`~/go/bin/staticcheck ./...`、`go test ./...` 全綠。新測試：
  - `sse`：observer 會看到每一個事件，包括被丟掉的。
  - `subtitle`：tracker 的 6 個測試。
  - `handlers`：status 帶階段的 3 個測試，兩種模式都測，用真的 tracker。
- **前端：** `pnpm nx test web` 300 檔、4693 測試全綠。
  - hook 新增 10 個測試：起點、帶百分比的起點、沒有起點／不認得的起點、結束沒看到、還在跑、結束事件先到、reset 之後才回答、問失敗、重連再問、沒有 probe 就不問。
  - dialog 新增 3 個測試，另外 6 個舊測試補上第三個參數。
- **突變檢查：** 把 `onEndedUnobserved` 寫進快取的值改成 `inProgress: true`，「不再接回去」那個測試會變紅。
- **lint 和格式：** `pnpm run lint:all` 0 errors（warning 都是原本就有的，這次改的檔案 eslint 0 警告）。`format:check` 綠。
- **E2E：** grep 過 `transcribe/status` 的 E2E。mock 都回 `{in_progress:false}`，API 測試查的是不存在的 id。新欄位是可省略的，不影響這些測試。

### 範圍外（沒修）

- **生成工作區** `GenerationWorkspaceV2`：接上時也是從 extracting 開始。它看的是批次快照，另外處理。
- **語音辨識中間沒有進度事件**（例如「第幾段」）：這張只修「接上時的起點」，不修這個。
- **同一部片同時在搜尋字幕：** 搜尋完成的 `subtitle_progress complete` 會清掉紀錄。這時接上的視窗會照舊從 extracting 開始，下一個事件來了就會跳對。不會顯示錯的階段，只是少了起點。

### File List

- `apps/api/internal/sse/hub.go`、`hub_test.go`
- `apps/api/internal/subtitle/phase_tracker.go`（新）、`phase_tracker_test.go`（新）
- `apps/api/internal/handlers/transcription_handler.go`、`transcription_status_phase_test.go`（新）
- `apps/api/cmd/api/main.go`
- `apps/web/src/hooks/useGenerationProgress.ts`、`useGenerationProgress.spec.ts`
- `apps/web/src/services/transcriptionService.ts`
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`、`ManageSubtitleDialogV2.spec.tsx`
