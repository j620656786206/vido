# Story disc-2026-09-transcription-run-5min-hard-timeout：長片的 AI 字幕生成不再被 5 分鐘砍掉——每一段依片長與檔案大小給時間，逾時時說得出是哪一段、該調哪個設定

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who presses 生成字幕 on a two-hour 4K remux,
I want the run to get as much time as that file and that length actually need, and — if it still runs out — a one-line reason that names the step and the setting to raise,
so that generation works on the films I actually own, instead of dying at exactly five minutes with thirty lines of ffmpeg noise and $0 of useful work.

## Context

### 🔴 實測證據（2026-09-18，NAS 隔離容器；正式 Vido 全程未動）

| 時間 | 發生什麼 |
| --- | --- |
| 20:20:18 | `POST /api/v1/movies/:id/transcribe?translate=true`，《哈利波特：火盃的考驗》——**157 分鐘、66.8 GB** 4K Remux，英文音軌，pipeline 模式 |
| 20:25:13 | 光是抽出音訊（`ffmpeg -map 0:1 … pcm_s16le 16k mono`）就花了 **4 分 55 秒** |
| 20:25:18 | **剛好滿 5 分鐘**，整個 run 的 `context.WithTimeout` 到期，正在切分音訊的 ffmpeg 被 `signal: killed` → `failJob` → 花費 **$0**，語音辨識與翻譯**一次都沒開始** |

而且失敗訊息（`error` 欄位）是 `transcribe: split chunks: ffmpeg chunk split at 8400s: signal: killed — ffmpeg version 6.1.2 … [30 行 banner ＋ stream mapping]`。這一整段會原樣出現在詳情頁對話框的失敗面板（`GenerationProgressV2.tsx:227-236`，Mono、`break-all`）。工作區的中文對照（`generationEventCopy.ts`）也認不出它（沒有 `timed out`／`deadline exceeded`），只能寫「生成失敗」。

**這不是邊角案例。** 片庫 55 部電影裡 **25 部超過 20 GB**（NAS DB 實查）；這台的抽音訊速度約 **4.4 s/GB**，所以 20 GB 以上的片光抽音訊就要 1.5 分鐘起跳，100 GB 級的原盤直接超過 5 分鐘。而且就算抽得完，157 分鐘的片後面還有 **16 段** Whisper（每段 600 s 音訊，`WhisperChunkDuration`）加上約 **27 分鐘**的翻譯（文件寫 Sonnet ≈ 片長的 17%）——5 分鐘無論如何裝不下。

### 現況（main `76154955`；行號為現況）

1. **整個 run 一個寫死的 5 分鐘。** `TranscriptionService.timeout = 5 * time.Minute`（`transcription_service.go:180`），沒有 setter、沒有 env。`StartTranscription`（`:370`，solo 路徑，`context.Background()`＋timeout）與 `RunTranscription`（`:404`，批次／pipeline 退回路徑，caller ctx＋timeout）都套在**提取音訊＋切段＋語音辨識＋翻譯＋寫回**全部之上。
2. **抽音訊自己還有第二道 5 分鐘。** `NewAudioExtractorService(1, 5*time.Minute, …)`（`main.go:581`），`ExtractAudio` 用它開 `extractCtx`（`audio_extractor_service.go:176`）。這次是 4:55 剛好躲過；**修掉第 1 條之後，這一條會變成下一個懸崖**（66.8 GB 在慢一點的磁碟就會撞到）。
3. **正確做法已經在隔壁。** 抽內嵌字幕的 `subtitle.Extractor`（sub-6-3）是 `max(SUBTITLE_EXTRACT_TIMEOUT_SECONDS 600, SUBTITLE_EXTRACT_PER_GB_SECONDS 30 × GB)`（`extractor.go:280-291`），逾時訊息只在**自己的**上限觸發時才點名那個 env（`:370-385`，測試 `extractor_gate_test.go:159`）。自動生成那一線也是 `max(15m floor, extractTimeout + 5m slack)`（`auto_generation.go:296-306`）。
4. **切段的錯誤把整份 ffmpeg 輸出塞進錯誤字串。** `ai.SplitAudioChunks`（`whisper.go:554-559`）`fmt.Errorf("… %w — %s", err, string(output))`——`CombinedOutput` 含 banner、configuration、stream mapping。被 ctx 砍掉時 `err` 是 `signal: killed`，沒有包 `context.DeadlineExceeded`，所以呼叫端與前端都分不出「逾時」和「ffmpeg 壞了」。
5. **後面每一段都有自己的上限，不需要整體上限來兜底。** Whisper 每個請求 5 分鐘（`whisper.go:156`，一段 ≤ 24 MiB ≈ 13 分鐘音訊），Claude 每次嘗試由 `RequestTimeoutFor` 推算（`claude.go:54-58`），翻譯服務刻意不再包 deadline（`translation_service.go:27-29`）。整體 5 分鐘不是安全網，只是一把砍錯人的刀。
6. **時長來源**：抽完音訊之後 WAV 的精確時長就在手上（`parseWAVInfo`，`whisper.go:624`，`SplitAudioChunks` 已經在用）；續跑翻譯（translate-only resume）沒有 WAV，但有英文 SRT。DB 的 `duration_seconds`／`runtime` 多數是空的（NAS 55 部只有 1 部有可用時長），不能當主要來源。
7. **不受影響、不要動的**：自動生成 lane 只跑免費項目（`scan_auto_free_generation=true`），語音辨識不會走那裡，它的 15 分鐘 floor 不用改；批次 lane 的 ctx 只有取消沒有 deadline（`generation_batch.go` 無 `WithTimeout`），worker pool 同（`worker_pool.go:428`）。

### ⚖️ 設計裁定（SM，2026-09-18；架構先行，dev 不得改方向，可在 Dev Notes 記偏離）

**拿掉「整個 run 一個上限」，改成兩段各自的上限，每段的上限由它真正的工作量決定：**

| 階段 | 工作量跟什麼成正比 | 上限 | 用哪些設定 |
| --- | --- | --- | --- |
| **A 提取音訊**（ffmpeg 讀整個檔） | 檔案大小 | `max(floor, perGB × GB)`——**與抽字幕共用同兩個 env**（同一顆磁碟、同一種 ffmpeg 讀全檔） | `SUBTITLE_EXTRACT_TIMEOUT_SECONDS`（600）、`SUBTITLE_EXTRACT_PER_GB_SECONDS`（30） |
| **B 切段＋語音辨識＋翻譯＋寫回** | 媒體時長 | `max(floor, perMediaMinute × 媒體分鐘)` | **新增** `TRANSCRIPTION_RUN_TIMEOUT_SECONDS`（floor，預設 **600**）、`TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE`（預設 **30**） |

- 階段 B 的時長：**完整路徑**用抽出來的 WAV 時長（精確、免查 DB）；**續跑翻譯**用英文 SRT 最後一句的結束時間；兩者都拿不到 → floor。
- 預設值的算術：157 分鐘 → 78.5 分鐘預算，估計實際 35–50 分鐘（16 段 Whisper ＋ 27 分鐘翻譯）；45 分鐘的一集 → 22.5 分鐘；20 分鐘的一集 → floor 10 分鐘（實測 200 s）。
- **不再有整體上限。** 每一段都有界，掛住的呼叫由各自 client 的 timeout 處理（現況 #5）。
- 逾時訊息照 `subtitle.Extractor` 的規矩：一行、說是**哪一段**、跑了多久、片長／檔案大小、預算多少、**該調哪個 env**；只在**我們自己的**上限觸發時才點名 env（caller 取消／關機不點名）。⛔ 不再把 ffmpeg 全文塞進錯誤字串。

## Acceptance Criteria

1. **共用的「依大小給時間」公式搬到 `services`，兩個 ffmpeg 讀全檔的地方都用它。**
   - 新增純函式（建議 `services.SizedFFmpegTimeout(floor, perGB time.Duration, sizeBytes int64) time.Duration`）＝ `max(floor, sizeGB × perGB)`，`sizeBytes ≤ 0` → floor。`subtitle.Extractor.effectiveTimeout`（`extractor.go:280-291`）改呼叫它（`subtitle` 已 import `services`，反向不行——Rule 19），**`extractor_gate_test.go` 一行斷言都不准改**。
   - `AudioExtractorService` 加 `WithAudioExtractPerGB(d time.Duration)` 選項與 `EffectiveTimeout(path) (time.Duration, float64)`（照 `subtitle.Extractor` 的形狀，含 `withFileSize` 測試縫），`ExtractAudio` 的 `extractCtx` 改用它。`main.go:581` 傳入 `subtitleExtractTimeout`／`subtitleExtractPerGB`（`:441-442` 已算好），不再寫死 5 分鐘。
   - 抽音訊逾時的錯誤（`ErrAudioExtractionTimeout` 那條路，`audio_extractor_service.go:198-200`）改成一行，含檔案大小、上限秒數、觸發的 env 名——只在我們的 `extractCtx` 觸發而 caller ctx 仍活著時點名（`extractor.go:370-385` 的分辨法）。

2. **階段 B 的預算由片長決定，兩個新 env。**
   - `config.go`：`TranscriptionRunTimeoutSeconds`（`TRANSCRIPTION_RUN_TIMEOUT_SECONDS`，`loadInt` 預設 600）、`TranscriptionSecondsPerMediaMinute`（`TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE`，預設 30）；照 `SUBTITLE_EXTRACT_*` 的寫法（`:88-93, :205-209`）加註解，並加進啟動 log 的那組（`:397` 附近的 `Sources` 列印，若該區有列同類 knob 就跟著列）。
   - `TranscriptionService`：拿掉 `timeout` 欄位與 `:180`；新增 `SetRunBudget(floor, perMediaMinute time.Duration)`（main.go 接線），純函式 `runPhaseBudget(floor, perMediaMinute time.Duration, mediaSeconds float64) time.Duration` ＝ `max(floor, mediaSeconds/60 × perMediaMinute)`，`mediaSeconds ≤ 0` → floor。
   - `runPipeline`（`:566-687`）：**不再**在 `StartTranscription`／`RunTranscription` 外層包 timeout（`:370`／`:404`）；改成——階段 A 由 `ExtractAudio` 自己的上限管；抽完音訊後讀 WAV 時長（把 `parseWAVInfo` 以 `ai.WAVDuration(path) (float64, error)` 匯出，或等價的最小匯出），開 `phaseCtx, cancel := context.WithTimeout(ctx, runPhaseBudget(...))` 包住**切段 → 語音辨識 → 翻譯 → 寫回**。續跑翻譯（`tryTranslateOnlyResume` 成功時）用 SRT 最後一句結束時間當 `mediaSeconds`（小 helper，解析最後一個 `-->` 行；解析失敗 → 0 → floor）。
   - 開始時 `slog.Info` 一行記下兩段的預算與依據（`extract_budget`、`file_gb`、`run_budget`、`media_minutes`），讓 NAS 上看 log 就知道為什麼給這麼多時間。
   - `RunTranscription` 對 caller ctx 的既有語意不變：`TestRunTranscription_DerivesTimeoutFromCallerCtx`（`transcription_generation_test.go:105`）**照樣綠**。

3. **逾時說人話，ffmpeg 雜訊不上畫面。**
   - `ai.SplitAudioChunks`（`whisper.go:554-559`）：ffmpeg 失敗時錯誤只帶 stderr **最後 3 行、上限 300 字元**（`subtitle.extractFailure` 的「stderr tail」做法），永遠不含 `ffmpeg version` banner；若 `ctx.Err() != nil`，錯誤改包 `ctx.Err()`（`%w`）並寫 `stopped by the run deadline at <start>s`，不寫 `signal: killed`。
   - `transcribeAudio`／`translateAndPersist` 回到 `runPipeline` 的錯誤若 `errors.Is(err, context.DeadlineExceeded)` 且是**我們的** `phaseCtx` 觸發（caller ctx 仍活著）→ `failJob` 收到的一行是：`transcription run timed out after <elapsed> in <chunking|transcribing|translating|writeback> (media <N> min, budget <M> s — raise TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE)`；預算由 floor 決定時點名 `TRANSCRIPTION_RUN_TIMEOUT_SECONDS`（`effectiveTimeout` 回傳 knob 的同一招）。caller 取消 → `stopped by the caller's deadline/cancel`，不點名任何 env。
   - 這一行必須含 `timed out`：前端 `failureCopy`（`generationEventCopy.ts:47-48`）靠這個字串對到「處理逾時」——在 `generationEventCopy.spec.ts` 補一條用**新的完整句子**當輸入的案例釘住。⛔ 前端其他檔案不動。

4. **文件跟上。** `docs/deployment.md` 的「Subtitle Generation Variables」表加兩列（新 env、預設、一句話說明），Notes 加一段：抽音訊現在也受 `SUBTITLE_EXTRACT_*` 約束、語音辨識＋翻譯的預算怎麼算、逾時訊息會點名哪個 knob（照 sub-6-3 那段的寫法）。`deployment.md` 目前只有英文版（無 zh-TW 孿生），照現狀。`unraid-template/vido.xml` 沒列 `SUBTITLE_EXTRACT_*`，本張也**不**加新 env（與既有 knob 同等曝光）。

5. **測試（先寫紅測試）。**
   - `SizedFFmpegTimeout` 表格測試（沿用 `extractor_gate_test.go:40-43` 的四組數字）；`subtitle` 那邊全套照跑不改。
   - `AudioExtractorService.EffectiveTimeout`：小檔 → floor、93 GB → 46m30s、stat 失敗 → floor；逾時訊息含 `SUBTITLE_EXTRACT_PER_GB_SECONDS` 只在我們的上限觸發時（fake ffmpeg：precedent `installFakeFFmpeg`，`extractor_gate_test.go`）。
   - `runPhaseBudget` 表格：0 → floor；9425 s → 78m30s；2700 s（45 min）→ 22m30s；1200 s → floor（等於 600 s，取 floor）。
   - SRT 尾句時間 helper：正常／空字串／沒有 `-->`。
   - `SplitAudioChunks`：fake `execCommandContext`（`whisper_test.go:501-507` 的縫）印 40 行後 exit 1 → 錯誤 ≤ 3 行 stderr、不含 `ffmpeg version`；ctx 已到期 → `errors.Is(err, context.DeadlineExceeded)` 且不含 `signal: killed`。
   - `runPipeline` 逾時句子：用會 sleep 的 fake ASR（`ai.ASRProvider` 介面）＋極小 `SetRunBudget`，斷言 `failJob` 送出的 `error` 是一行、含 `timed out`、含 `TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE` 或 `TRANSCRIPTION_RUN_TIMEOUT_SECONDS`（依情境）、不含 banner；caller ctx 先取消 → 不含任何 env 名。⚠️ 需要真的 WAV：測試用 `os.WriteFile` 寫一個最小 44-byte header 的 WAV（`parseWAVInfo` 讀 header）或抽 `ai` 套件既有的 WAV 測試 fixture。
   - **對抗式 mutation check**：把「用 WAV 時長」換回常數、把 stderr tail 換回全文、把 knob 分辨拿掉——各至少一條紅，列在 Completion Notes。

6. **真機驗證（done 的門檻，不是可選）。** 照 `.claude/memory/project_unraid_nas_access.md` 的隔離容器食譜，**同一部片**（`ceb9fec6-4a61-4a0f-887c-e37961f310f7`，66.8 GB／157 min）再跑一次：`GOOS=linux GOARCH=amd64 go build ./cmd/api` 出來的二進位以 `-v` 掛進正式 image 的容器覆蓋 `/app/api`（image 已有 ffmpeg／opencc），其餘設定照食譜。要記錄：兩段預算的 log 行、每段實際耗時、`transcription run AI usage` 那行的花費（估 ≈ $3；`AI_RUN_BUDGET_USD` 預設 5 會擋）、以及 sidecar 是否落在 scratch。⛔ 正式 Vido 不動；跑完 `docker rm -f` ＋ `rm -rf` scratch。若二進位覆蓋在該 image 上跑不起來（例如 entrypoint 的 `su-exec` 路徑），改為 PR 合併、`:main` image 建好後再驗，驗完才把 sprint-status 改 done——在 Completion Notes 寫明是哪一種。

7. **不做的事。** 不改 Whisper 每請求 5 分鐘、不改翻譯服務、不改自動生成 lane 的 15 分鐘、不加整體上限、不改 SSE payload 形狀（無 contract 變動）、不做 `failJob` 的 `message` 中文化（Rule 3 既有缺口，另案）、前端只補一條 spec。

8. **CI 全綠**：`pnpm nx test api`、`pnpm nx test web`（只有一條新 spec）、`pnpm run lint:all`（含 `gofmt`／`go vet` 走 nx lint）、`pnpm run format:check`。⛔ 測試不准 `run_in_background`；跑完 `pnpm run test:cleanup`。

## Tasks / Subtasks

- [x] **Task 1 — 共用公式＋抽音訊依大小給時間（AC: #1）**
  - [x] 先寫紅測試：`SizedFFmpegTimeout` 表格、`AudioExtractorService.EffectiveTimeout`、逾時訊息點名 knob
  - [x] `services.SizedFFmpegTimeout`；`subtitle.Extractor` 改呼叫；`AudioExtractorService` 選項＋`EffectiveTimeout`＋訊息；`main.go:581` 接線
- [x] **Task 2 — 階段 B 預算與兩個 env（AC: #2）**
  - [x] 先寫紅測試：`runPhaseBudget` 表格、SRT 尾句時間 helper、`runPipeline` 不再 5 分鐘（fake ASR sleep 6 s＋預算 10 s 能完成）
  - [x] `config.go` 兩個 env；`SetRunBudget`；`ai.WAVDuration` 匯出；`runPipeline` 改兩段；啟動與 run-start 的 log 行
- [x] **Task 3 — 逾時說人話（AC: #3）**
  - [x] 先寫紅測試：`SplitAudioChunks` stderr tail／`DeadlineExceeded` 包裝；`runPipeline` 逾時句子（我們的 vs caller 的）；前端 `failureCopy` 新句子
  - [x] 實作
- [x] **Task 4 — 文件（AC: #4）**
- [x] **Task 5 — 收尾（AC: #5, #7, #8）**：全套閘門、mutation check、Completion Notes
- [ ] **Task 6 — 真機驗證（AC: #6）**：NAS 隔離容器重跑同一部片，記錄耗時與花費，清理

## Dev Notes

### 這張的重點

- **根因只有一行**（`:180` 的 5 分鐘），但正確的修法不是把 5 改成 60：抽音訊的成本看檔案大小，語音辨識與翻譯的成本看片長，兩者差一個數量級，同一個常數不可能兩邊都對。隔壁的 `subtitle.Extractor` 與 `AutoGenerator` 已經示範過「依工作量給時間」與「訊息點名 knob」，本張是把同一套搬過來。
- **第二道 5 分鐘（抽音訊）是修完第一道之後的下一個懸崖**——一定要一起處理，否則 NAS 上的 66.8 GB 只是從「5:00 被砍」變成「4:55 差一點被砍」。
- **不加整體上限**是刻意的：每一段各自有界，整體上限只會再砍錯人。

### 上游契約（Rule 20）

- 無 `[@contract-v*]` 變動：SSE `transcription_*` payload 形狀不變（只有 `error` 字串內容變短、變成一行）；新增 env 是純加寬；`ai.WAVDuration` 是新匯出的純函式。
- 📎 消費者提醒：`useGenerationProgress`／`GenerationProgressV2` 直接顯示 `error`；`generationEventCopy.failureCopy` 靠 `timed out` 子字串——AC #3 的句子必須保留這兩個字。

### 已知陷阱

- **`services` 不能 import `subtitle`**（`extractor.go` 已 import `services`，Rule 19 循環點）——公式放 `services`，`subtitle` 來呼叫。
- **`StartTranscription` 是 `context.Background()` 分離**（`:364-372`，刻意讓 job 活過 HTTP 請求）：拿掉外層 timeout 後，solo 路徑的 ctx 沒有任何 deadline，直到抽完音訊才開 `phaseCtx`——這段時間由 `ExtractAudio` 的 `extractCtx` 管，所以沒有無界區間。ffprobe 那段有自己的 30 s（`audio_extractor_service.go:111`）。
- **`SplitAudioChunks` 的 ctx 是 `phaseCtx`**，chunk ffmpeg 讀的是已抽出的 WAV（小、快），被砍多半是整段預算耗盡，不是 ffmpeg 慢——訊息要說 run deadline，不要說 ffmpeg 逾時。
- **`ErrAudioExtractionTimeout` 是 sentinel**（`audio_extractor_service.go:20`），既有測試可能 `ErrorIs` 它——改訊息時用 `fmt.Errorf("%w: …", ErrAudioExtractionTimeout, …)` 保留 sentinel。
- **NAS 的 `AI_PROVIDER=gemini` 但 pipeline 用 Claude holder**，與本張無關；ASR key 走 `ASRProviderHolder`（secrets → env），隔離容器只要帶齊 env（含 `ENCRYPTION_KEY`）就能解到 key——食譜已驗證。
- **真機驗證會花錢**（估 ≈ $3，上限 $5）。跑之前確認 scratch 目錄可寫，sidecar 才不會寫回正式片庫。

### Source tree

```
apps/api/internal/services/transcription_service.go        ← Task 2, 3（拿掉 timeout；runPhaseBudget；phaseCtx；逾時句子）
apps/api/internal/services/audio_extractor_service.go(+test) ← Task 1
apps/api/internal/services/ffmpeg_timeout.go(+test)（新，SizedFFmpegTimeout）← Task 1
apps/api/internal/subtitle/extractor.go                     ← Task 1（只改 effectiveTimeout 呼叫共用公式；測試不動）
apps/api/internal/ai/whisper.go(+test)                      ← Task 2（WAVDuration 匯出）、Task 3（SplitAudioChunks 錯誤）
apps/api/internal/config/config.go                          ← Task 2
apps/api/cmd/api/main.go                                    ← Task 1, 2（接線）
apps/api/internal/services/transcription_generation_test.go ← Task 2, 3
apps/web/src/components/subtitle/generationEventCopy.spec.ts ← Task 3（一條）
docs/deployment.md                                          ← Task 4
```

### Cross-Stack Split Check

後端 task 5（Task 1–4 ＋ 6），前端 1 條 spec（Task 3 內）→ 跨棧門檻（兩邊都 >3）**不觸發**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 純後端（前端只加一條 spec）。

### References

- [Source: `apps/api/internal/services/transcription_service.go:125-130, 163-181, 338-372, 375-406, 566-687, 1056-1105`]
- [Source: `apps/api/internal/services/audio_extractor_service.go:20, 44-96, 111-126, 150-215`、`cmd/api/main.go:441-442, 575-620`]
- [Source: `apps/api/internal/subtitle/extractor.go:30, 46, 255-291, 300-390`、`extractor_gate_test.go:21-60, 157-200, 252-291`、`auto_generation.go:46, 52, 170-190, 276-330`]
- [Source: `apps/api/internal/ai/whisper.go:27-33, 42, 94-97, 156, 483-570, 578, 624`、`whisper_test.go:501-507`、`claude.go:54-58, 99-103`、`translation_service.go:27-29`]
- [Source: `apps/api/internal/config/config.go:88-93, 168, 205-209, 287-310, 397`]
- [Source: `apps/web/src/components/subtitle/GenerationProgressV2.tsx:70-90, 215-236`、`generationEventCopy.ts:37-53`]
- [Source: `docs/deployment.md:70-90, 140-150`]
- [Source: NAS 實測 2026-09-18（本檔 Context 表）；`.claude/memory/project_unraid_nas_access.md`（隔離容器食譜）]
- [Source: project-context.md#Rule 3 / #Rule 11 / #Rule 13 / #Rule 19 / #Rule 20 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — dev-story (Amelia)，2026-09-18，branch `fix/transcription-run-timeout`。

### Debug Log References

- 第一版「caller 的 deadline」測試是**假的**：用 300 ms 的 caller timeout，結果在 `ListAudioTracks`（ffprobe）就到期了，訊息是 `ffprobe timeout: <path>`——而 `assert.Contains(msg, "caller")` 因為**暫存目錄名字裡有 `the_callers_`** 而過。改成由 fake ASR 的 hook 在轉錄步驟中取消 caller，並斷言 `stopped by the caller's deadline`＋`transcribing`。教訓：`Contains` 一個常見英文單字不算斷言。
- `TestRunPhaseBudget` 第一版的期望值算錯（157.08 min × 30 s ＝ 4712.5 s ＝ 1h18m32.5s，我寫成 12.5s）；程式是對的，測試改。
- Go 的 mutation 用 `mediaSeconds = 0` 會變成 unused variable 編譯錯誤（假陽性「紅」），改用 `d * 0`。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-18；根據 NAS 隔離容器實測＋程式碼逐行查證）。
- 🔗 **AC Drift: NONE**（checked: `ErrAudioExtractionTimeout|audio extraction timed out|split chunks|derives its timeout|s.timeout` across `_bmad-output/implementation-artifacts/*.md` — 命中 9R-16（Task 2 註記「RunTranscription derives its timeout from the CALLER ctx」）與 sub-6-3；9R-16 AC 6a 的可觀察承諾是「caller 的取消／預算會傳進來」，本張保留 caller ctx 傳遞、只拿掉外層固定 timeout → REUSE；sub-6-3 的 `EffectiveTimeout` 語意不變、只把公式搬到 `services` 共用 → REUSE）。
- 📎 **Contract Stamps: FOUND（本張 0 個自有 stamp、1 條上游 ack）** —— confirmed against [@contract-v1] (Story sub-6-3 AC #1)：`Extractor.EffectiveTimeout` ＝ `max(floor, size × perGB)`，本張把公式抽成 `services.SizedFFmpegTimeout` 由它呼叫，`extractor_gate_test.go` 一行未改、全綠。SSE payload 形狀不變、新增 env 純加寬 → 無 bump。
- **Task 1**：`services.SizedFFmpegTimeout`／`BytesPerGB`／兩個 env 常數（`ffmpeg_timeout.go`）；`subtitle.Extractor` 改呼叫共用公式；`AudioExtractorService` 加 `WithAudioExtractPerGB`、`EffectiveTimeout`、`withAudioFileSize` 測試縫，逾時訊息一行點名 knob（只在我們的上限觸發時）；`main.go` 改傳 `subtitleExtractTimeout`／`subtitleExtractPerGB`。
- **Task 2**：`TranscriptionService.timeout` 欄位刪除；`SetRunBudget`、`runPhaseBudget`（回傳 knob）、`srtSpanSeconds`；`StartTranscription`／`RunTranscription` 改 `WithCancel`（無整體 deadline）；`runPipeline` 抽完音訊後讀 `ai.WAVDuration` → `phaseCtx` 包住切段／辨識／翻譯／寫回；續跑翻譯用 SRT 尾句時間；兩行預算 log（`audio extraction budget`、`transcription run budget`）。config 兩個 env（含 `config_test.go` 三組）。
- **Task 3**：`ai.SplitAudioChunks` 錯誤只帶 stderr 最後 3 行／300 字元（`stderrTail`），ctx 到期時包 `ctx.Err()` 並寫 `stopped by the run deadline`；`phaseTimeoutError` 產生一行句子（步驟、耗時、片長、預算、knob；caller 取消時不點名）；前端 `generationEventCopy.spec.ts` 用兩句真實的新句子釘住「處理逾時」。
- **Task 4**：`docs/deployment.md` 表加兩列、Notes 加一段（英文版唯一，照現狀）。
- **對抗式 mutation check（7 項修法逐一拿掉，全部有牙）**：WAV 時長換常數 → 1 紅；stderr tail 換全文 → 1 紅；run 逾時的「caller vs 我們」分辨 → 1 紅（第一版測試沒抓到，見 Debug Log）；抽音訊逾時的分辨 → 1 紅；切段的 ctx 包裝 → 1 紅；phase budget 退回只看 floor → 2 紅；後端 `title` 的 Solo 守衛（前一張）不在本張範圍。
- 🎭 **A11y Pre-Flight: N/A**（前端只改一條 spec，沒有元件變動）。
- **Pre-existing failures**：無。`pnpm nx test api` 全綠、`pnpm nx test web` 270 檔／3870 條全綠、`lint:all` 0 errors（warning 數與改動前相同）、`format:check` 綠。
- ⚠️ **沒修、記錄於此**：`ListAudioTracks` 的 `ffprobe timeout: <path>`（`audio_extractor_service.go:126`）在 caller 取消時也會這樣寫，而且帶伺服器路徑——本張的 AC #3 只涵蓋我們自己的兩段上限；這條走的是 30 秒的 ffprobe 探測，觸發機率低，交代到 `disc-2026-09-single-job-title-missing` 同批的「錯誤字串邊界」時一起看。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES**
- **① expand-scope-in-place**
  - 抽音訊自己的第二道 5 分鐘（`main.go:581`）→ **AC #1**（修完主因後的下一個懸崖，一起修）
  - 切段錯誤帶整份 ffmpeg 輸出、砍掉時不包 `DeadlineExceeded` → **AC #3**
- **② spawn-blocking-story**：無。
- **③ backlog-with-carry-forward-link**
  - `failJob` 的 `message` 是英文（`"Transcription failed: " + err`，Rule 3 邊界缺口）→ 既有的前端對照已擋在畫面前，**不另立**；記錄於 AC #7。
- Reference: `project-context.md` Rule 24

### File List

**新增**

- `apps/api/internal/services/ffmpeg_timeout.go`（＋`ffmpeg_timeout_test.go`）
- `apps/api/internal/services/transcription_run_budget_test.go`
- `apps/api/internal/ai/whisper_chunk_error_test.go`

**修改**

- `apps/api/internal/services/transcription_service.go`、`audio_extractor_service.go`
- `apps/api/internal/subtitle/extractor.go`（只改 `effectiveTimeout` 呼叫共用公式）
- `apps/api/internal/ai/whisper.go`（`WAVDuration`、`stderrTail`、`SplitAudioChunks` 錯誤）
- `apps/api/internal/config/config.go`（＋`config_test.go`）
- `apps/api/cmd/api/main.go`
- `apps/web/src/components/subtitle/generationEventCopy.spec.ts`
- `docs/deployment.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-18 | Story 建立（SM Bob, create-story；main `76154955`）。由 `dsr-6d-c-2` 建單時立的 disc 升格：NAS 隔離容器實測《火盃的考驗》（157 min／66.8 GB）——抽音訊 4:55，整個 run 在 5:00 整被砍、$0 花費、語音辨識沒開始；片庫 25/55 部超過 20 GB。⚖️ 裁定：拿掉整體 5 分鐘，改成「抽音訊依檔案大小（共用 `SUBTITLE_EXTRACT_*`）」＋「切段／辨識／翻譯依片長（新 `TRANSCRIPTION_RUN_TIMEOUT_SECONDS` 600、`TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE` 30）」，時長取自抽出的 WAV；逾時一行點名該調的 env，ffmpeg 全文不再進錯誤字串；真機重跑同一部片是 done 的門檻。 |
