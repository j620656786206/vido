# Story disc-2026-10-activity-generation-progress-detail：活動頁看得出字幕生成做到第幾段、第幾句

Status: done

**Source:** Alexyu 2026-10-05（See S01E02 生成時）：活動頁只寫「字幕生成中／進行中」，看不出有沒有在動。2026-10-12 討論後 Alexyu 選做法 2（段數＋句數），排除做法 1（只顯示階段）和「估整體百分比」。

## 查到的事（2026-10-12，Bob 逐一開檔確認）

### 活動頁現在

- 每 15 秒抓一次 `GET /api/v1/activity`（`useActivity`）。
- 「字幕生成中」那列（`ActivityHub.tsx`，kind=`transcription`）右邊固定寫「進行中」、不畫進度條（`NO_PERCENT_KINDS`）。理由：後端只送 `percent_done=0`，顯示「0%」會像卡住。
- 「批次生成」那列右邊是「第幾部 / 共幾部」，看不出現在那部做到哪。
- 後端：`ActivityService`（`services/activity_service.go:88-185`）組 `ActiveJob{kind, percent_done, detail, current, total}`。
  - 單部生成：正式機是管線模式，資料來自 `SoloRunner.ActivityProgress`（`subtitle/solo_runner.go:290`），`current` 其實是「幾個工作在跑」，`total` 永遠 0。
  - 批次：`GenerationBatchProcessor.ActivityProgress`，`current/total` 是第幾部／共幾部；批次知道 `CurrentMediaID`（`generation_batch.go:105`）。
- 設計稿 K1-D（`flow-k-activity-v2/k1-d.png`）的「字幕生成中」那列本來就畫了右側讀數＋下方進度條；程式碼沒畫進度條，是因為沒有數字。

### 伺服器知道、但沒送出去的數字

- **轉錄（聽寫）：** `transcribeAudio`（`transcription_service.go:1476`）把音訊切成好幾段（每段約 2 分鐘），逐段送去聽寫。目前只在開始時送一次 `transcription_progress`（`:995`），每段做完只寫 log。
- **翻譯：** `translateSRT` 的 `progressFn(pct)` 每批（約 10 句）送一次 `translation_progress`，只帶百分比。總句數 `len(blocks)` 在 `translateSRT` 裡就有。
- 早上做的 `subtitle.PhaseTracker`（PR #747）在 Hub 出口記住每部片現在的階段；它可以順便記住段數／句數。

## Story

身為等字幕生成的人，
我希望活動頁寫出「轉錄 2/6 段」「翻譯 120/659 句」，並畫出進度條，
這樣我一眼就知道它有在動、還要多久，不用猜是不是卡住了。

## Acceptance Criteria

1. **轉錄每做完一段就送一次通知：** `transcription_progress`，帶 `current`（做完幾段）、`total`（共幾段）。只有切成兩段以上才送（短片一段就一次做完）。拿存好的段（續跑）也算做完一段。`message` 變成「正在轉錄音訊（第 2/6 段）」——「管理字幕」視窗原本就照字顯示 `message`，所以視窗也看得到。
2. **翻譯的通知帶句數：** `translation_progress` 加 `current`（已翻幾句）、`total`（共幾句）。`message`、`percentage` 不變。
3. **PhaseTracker 記住數字：** `PhaseSnapshot` 加 `Current`、`Total`；收到不帶數字的同階段事件時清掉（例如進入翻譯時清掉轉錄的段數）。
4. **活動 API：** `ActiveJob` 加 `step`（省略 = 不知道）：`{phase, current, total}`，phase 用 `extracting | transcribing | translating`。
   - `transcription`：只有一部在跑時，帶那部片的 step；有段數／句數時 `percent_done` = current×100÷total。兩部以上同時跑，不帶 step（不知道該顯示哪部）。
   - `generation_batch`：帶目前那部片的 step；`current/total/percent_done` 照舊是第幾部。
   - 沒有接上 PhaseTracker 時，回應跟現在一樣（`[@contract]` 只加不改）。
5. **活動頁：**
   - 「字幕生成中」右邊：有段數／句數 →「轉錄 2/6 段」「翻譯 120/659 句」（數字等寬字），並畫進度條；只有階段 →「提取音訊」「轉錄中」「翻譯中」；什麼都沒有 → 照舊「進行中」、不畫進度條。
   - 「批次生成」：右邊和進度條照舊（第幾部）；下面那行片名後面加「· 轉錄 2/6 段」。
   - 用語跟「管理字幕」的步驟一致：提取音訊／轉錄／翻譯。
6. **不估整體百分比**（Alexyu 排除）。
7. **檢查：** Go 與前端全套測試、staticcheck、`lint:all` 全綠；E2E grep `activity-job-transcription`。

## Tasks / Subtasks

- [x] T1 轉錄逐段通知＋翻譯句數（AC #1、#2）
- [x] T2 PhaseTracker 記數字（AC #3）
- [x] T3 活動 API 的 `step`（AC #4）
- [x] T4 活動頁兩列（AC #5）
- [x] T5 檢查全綠（AC #7）

## Dev Notes

- 設計：K1-D 這列本來就是「標題／片名／右側讀數／進度條」的版型，這張只換右側讀數的字和讓進度條出現，不新增元素，所以不改 `.pen`。右側讀數沿用「12 / 30」那種等寬數字的寫法。
- 管線模式的 extract 路線（翻片內字幕軌）走 `subtitle_progress`，沒有句數；那條只會顯示「翻譯中」。
- 活動頁 15 秒更新一次，段數可能晚 15 秒，可以接受；「管理字幕」視窗走即時通知，是即時的。

## Dev Agent Record

### 做了什麼（2026-10-12，Opus 5.5）

- **轉錄逐段通知：** `transcribeAudioWithProgress`（`transcribeAudio` 包一層、回報傳 nil，18 個舊測試不用改）。每做完一段（含拿存好的段）呼叫 `onChunk(done, total)`；只有兩段以上才回報。`runTranscription` 收到後送 `transcription_progress`（`current`、`total`、`message`「正在轉錄音訊（第 2/6 段）」）。
- **翻譯句數：** `translateSRT` 的 `progressFn` 加 `current`（`round(pct/100×len(blocks))`，pct 本來就是 processed/total×100）和 `total`。`message`、`percentage` 不變。
- **PhaseTracker：** `PhaseSnapshot.Current/Total`，只從帶數字的事件拿；不帶數字的事件不留舊數字。新 `Step(mediaID)` 給 services 用（services 不能 import subtitle）。
- **活動 API：** `ActiveJob.Step *ActiveJobStep{phase, current, total}`。`ActivityService.SetStepReader`；來源實作選用介面 `ActiveMediaID()`（`SoloRunner`、`TranscriptionService` 的 solo 工作、`GenerationBatchProcessor` 的目前那部、`composedSoloJobs` 兩邊合計剛好一部才回）。`transcription` 有數字時 `percent_done` = current×100÷total；`generation_batch` 的數字照舊是第幾部。`main.go` 接上早上的 `generationPhases`。
- **活動頁：** `ActivityHub` 的「字幕生成中」右邊改 `StepReadout`（有數字「轉錄 2/6 段」＋進度條；只有階段「提取音訊／轉錄中／翻譯中」；沒有就「進行中」）；「批次生成」的片名後面加「· 轉錄 2/6 段」。
- 「生成工作區」的即時紀錄：同一階段的事件是原地更新（`useGenerationJobsFeed` 既有行為），逐段通知不會灌出很多行。

### 驗證

- Go：新測試檔 `generation_progress_detail_test.go`（逐段回報含續跑、一段不回報、訊息字樣、翻譯事件帶句數、活動列的 step 四種情況、`TranscriptionService.ActiveMediaID`）；`phase_tracker_test.go` 加數字測試；`solo_runner_test.go` 加 `ActiveMediaID`；新 `cmd/api/solo_job_source_test.go`。`go vet`、staticcheck、`go test ./...` 全綠。
- 前端：`ActivityHub.spec.tsx` 新增 7 條。全套 287 檔、4432 測試全綠；tsc 綠；`lint:all` 0 errors。
- 突變：`countedStep` 一律回 null → 3 條紅。
- E2E grep：沒有 E2E 用到活動頁這兩列；`manage-subtitle-mobile.spec.ts` 送的 `transcription_progress` 沒帶數字，行為不變。

### 範圍外

- 管線模式翻片內字幕軌（`subtitle_progress`）沒有句數，那條只會顯示「翻譯中」。
