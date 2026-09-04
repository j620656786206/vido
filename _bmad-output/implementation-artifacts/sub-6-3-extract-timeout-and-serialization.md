# Story 6.3: 抽軌 timeout 可調 + 抽軌階段序列化 —— 4K remux 片庫不再整批不能用（後端）

Status: done

## Story

As a NAS owner with 4K remuxes,
I want subtitle extraction to be bounded by my hardware and my file sizes rather than a fixed 10-minute constant, and never to have two extractions fight over the same disk,
so that a 93 GB file and a pair of 20 GB files stop failing before translation even starts.

## Context — 這個 story 為什麼存在

eval-1 產品問題 3 + 發現 7：`defaultExtractTimeout = 10 * time.Minute`（`extractor.go:22`）寫死，`main.go:676/925` 都傳 `0`（=預設）；Goodfellas 93 GB 抽軌逾時。兩個 worker（`PipelineConcurrencyM1 = 2`，`worker_pool.go:21`）同時抽兩個 20 GB 檔互搶 I/O，雙雙逾時；單獨跑只要 3.5 分鐘。

## Acceptance Criteria

1. **timeout 可調且依檔案大小。** 新 config `SUBTITLE_EXTRACT_TIMEOUT_SECONDS`（預設 600，`config.go` loadInt，`docs/deployment*.md` 記錄）。`Extractor.Extract` 的有效 timeout = `max(configured, fileSizeGB × perGBSeconds)`，`perGBSeconds` 預設 30（93 GB → 46 分鐘）。file size 由 `os.Stat` 取得，失敗則用 configured。

2. **抽軌序列化。** 新增 `subtitle.ExtractGate`（容量 1 的 semaphore，Rule 14 建一次重用）注入 `Extractor`；`Extract` 進入 ffmpeg 前 `Acquire(ctx)`，結束 `Release`。翻譯階段**不受**此閘門影響（兩個 worker 仍可同時翻譯）。等待中的 item 透過 SSE `subtitle_progress` `stage=extracting` message 帶「等待抽軌（前方 N 件）」。

3. **可觀測。** 抽軌開始／結束 `slog.Info` 帶 `file_size_gb`、`timeout_seconds`、`elapsed`；逾時錯誤 message 帶「檔案 X GB，逾時 Y 秒，可調 SUBTITLE_EXTRACT_TIMEOUT_SECONDS」。

4. **測試。** (a) timeout 推導表（小檔用 configured、大檔按 GB）；(b) ExtractGate：兩個並發 Extract 序列化（fake ffmpeg sleep，斷言不重疊）；(c) ctx 取消時 Acquire 立即回錯；(d) config 讀取；(e) 全回歸。

## Tasks / Subtasks

- [x] **Task 1 — 可調 timeout（AC: #1, #3）**
  - [x] config + `NewExtractor` 接線（`main.go:676/925` 兩處）；size-aware 計算
- [x] **Task 2 — ExtractGate（AC: #2）**
  - [x] `subtitle/extract_gate.go`；`Extractor` 注入；SSE 等待訊息
- [x] **Task 3 — 測試（AC: #4）**

（全後端。）

## Dev Notes

- `extractor.go:132-136` 註解明言「deliberately NO internal semaphore — the orchestrator's fixed concurrency of 2 is the only bound」。本 story **推翻**該裁定，理由是實測 I/O 互搶；註解要改寫並引用 eval-1 發現 7。
- 閘門只包 ffmpeg 子程序區段，不包 ffprobe（ffprobe 只讀 header）。
- Rule 14：gate 建一次、隨 pipeline 生命週期。
- `docs/deployment.md` + `.zh-TW.md`（Rule 17）。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- eval-1「產品問題 3」「發現 7」；後續 Backlog P0-3
- `apps/api/internal/subtitle/extractor.go`、`worker_pool.go:21`、`apps/api/cmd/api/main.go:676,925`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Debug Log References

- `go test ./...`（apps/api）全綠；`pnpm nx run api:lint`（go vet + staticcheck）綠。

### Completion Notes List

- **Task 1（AC #1/#3）**：`config.SubtitleExtractTimeoutSeconds`（`SUBTITLE_EXTRACT_TIMEOUT_SECONDS`，預設 600，非正值回預設）；`Extractor.EffectiveTimeout(path)` = `max(configured, sizeGB × perGBTimeout)`，`perGBTimeout` 預設 30s（`WithPerGBTimeout` 可調）、size 用 `os.Stat`（`withFileSize` 可注入測試）、stat 失敗用 configured。抽軌開始／結束 `slog.Info` 帶 `file_size_gb`／`timeout_seconds`／`elapsed`；逾時訊息帶「file X GB, timeout Y s — raise SUBTITLE_EXTRACT_TIMEOUT_SECONDS」，**只在是我們的 deadline 先到時**才點名這個 knob（父 ctx 的 deadline 先到 → 「stopped by the caller's deadline」）。`main.go` 兩處 `NewExtractor` 接 `cfg`。
- **Task 2（AC #2）**：`subtitle/extract_gate.go` `ExtractGate`（容量 1 chan + waiting 計數；`Acquire(ctx, onWait)` 忙碌時呼叫 `onWait(ahead)` 一次，ctx 結束立即回錯；release 冪等；nil gate no-op）。`main.go` 建一次（Rule 14）注入兩個 Extractor；`NewExtractor` 預設自建一個（未注入也自我序列化）。閘門只包 ffmpeg 子程序，ffprobe／翻譯不受影響。`extractor.go` 原「deliberately NO internal semaphore」註解改寫並引用 eval-1 發現 7。等待訊息：`WithExtractWaitNotifier(ctx, fn)` 由 `ProcessItem` 掛上（Extractor 建一次、不知 MediaRef），SSE `extracting`「等待抽軌（前方 N 件）」。
- **序列化的連帶（story 未列、實作必要）**：(a) `ErrSubtitleExtractWaitAborted`（Rule 7 code-list update under `SUBTITLE_`）— ctx 在**排隊中**結束（不是 ffmpeg 內）→ `failItem` 視同 cancelled（`CancelledRunPrefix`），不計入免費 lane 三振停權；ctx 進門前已結束仍是原本的「cancelled」路徑。(b) 免費 lane 的 `AutoGenerationItemTimeout`（15 min）改為 **floor**：`WithAutoExtractTimeout(extractor.EffectiveTimeout)`，每 item deadline = `max(floor, extractBound + 5 min)`，否則 93 GB 檔的 46 分鐘 ffmpeg deadline 會被 15 分鐘 item deadline 殺掉、三次後停權 — 正是這個 story 要救的檔案。`collect` 改回傳 `autoItem{ref, path}` 帶路徑。
- **Task 3（AC #4）**：(a) `TestExtractor_EffectiveTimeout` 表（4 GB／20 GB／93 GB／stat 失敗／自訂 per-GB）；(b) `TestExtractor_ConcurrentExtractsAreSerialized` — PATH 上放 shell `ffmpeg` shim（記 start/end 時戳、寫輸出檔），兩個 Extractor 共用 gate 並發抽軌 → 斷言 start,end,start,end 不重疊、且恰一個排隊（ahead=1）；(c) `TestExtractor_CancelledWhileQueued…` — gate 被佔、ctx 30ms 到期 → 立即回 `ErrSubtitleExtractWaitAborted` + `DeadlineExceeded`、ffmpeg 從未啟動；`TestExtractGate_*` ×6（序列化、queue depth 1/2/3、取消即回、已結束不入隊、release 冪等、nil）；(d) `TestLoad_SubtitleExtractTimeoutSeconds`（預設／覆寫／0／垃圾）；另 `ProcessItem` 等待敘事與 wait-abort 分類、auto lane deadline 推導與 collect 帶路徑、SSE 文案；(e) 全回歸綠。
- **文件**：`docs/deployment.md` 表格新增 `SUBTITLE_EXTRACT_TIMEOUT_SECONDS`、Notes 段說明 size-aware 與序列化。Rule 17 zh-TW twin：`docs/deployment.zh-TW.md` 仍不存在 → 既有 `backlog-deployment-doc-zh-tw-twin` RE-HIT。
- 🔗 AC Drift: NONE (checked: 'NewExtractor|AutoGenerationItemTimeout|defaultExtractTimeout' across _bmad-output/implementation-artifacts/*.md — bugfix-autogenerator-no-timeout-or-shutdown D2 的「subprocess deadline 先到」關係保留為 floor+slack；sub-1-3 `Extract` 簽名不變；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (`TrackExtractor` port 簽名未改；`ProcessItemOptions`／`ProcessOutcome` `[@contract-v1]` 未改)
- 🎭 A11y Pre-Flight: N/A（純後端）
- 🔌 Route Sync: N/A（無新路由）
- 🎨 UX Verification: N/A（純後端；SSE 文案走既有 `extracting` 氣泡）

### Discovery Triage

- ① expand-scope-in-place — 序列化引入「排隊中被 deadline 殺」的新失敗型態 → `ErrSubtitleExtractWaitAborted` 視同 cancelled，不計三振。
- ① expand-scope-in-place — 免費 lane 15 min item deadline 會殺掉 size-aware 的 46 min 抽軌 → `WithAutoExtractTimeout` 讓 item deadline 跟著 extractor bound。
- ③ backlog-with-carry-forward-link — `docs/deployment.zh-TW.md` 不存在 → 既有 `backlog-deployment-doc-zh-tw-twin` RE-HIT（本 story 在 EN 表格與 Notes 加了一行一段）。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — config、`EffectiveTimeout`、`main.go` 接線、逾時訊息、slog。 |
| 2026-09-04 | Task 2 — `extract_gate.go`、`Extractor` 注入、`WithExtractWaitNotifier` + SSE、`ErrSubtitleExtractWaitAborted`、auto lane size-aware item deadline。 |
| 2026-09-04 | Task 3 — gate／extractor shim／config／process_item／auto／SSE 測試；`docs/deployment.md`。 |
| 2026-09-04 | CR fixes（Opus 5 adversarial，10 findings，全部在同一分支修）— 見下方 Senior Developer Review。 |

### File List

- `apps/api/internal/subtitle/extract_gate.go`、`extract_gate_test.go`、`extractor_gate_test.go`（new）
- `apps/api/internal/subtitle/extractor.go`、`errors.go`、`process_item.go`、`progress_sse.go`、`auto_generation.go`（modified）+ `process_item_test.go`、`progress_sse_test.go`、`auto_generation_test.go`
- `apps/api/internal/config/config.go`（modified）+ `config_test.go`
- `apps/api/cmd/api/main.go`（modified）
- `apps/api/internal/services/audio_extractor_service.go`（modified，CR M3）+ `audio_extractor_service_test.go`
- `apps/api/internal/subtitle/errors_test.go`（modified，CR L9）
- `docs/deployment.md`、`project-context.md`、`_bmad-output/implementation-artifacts/sub-6-3-extract-timeout-and-serialization.md`、`sprint-status.yaml`

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → 6 個真問題與 4 個小項全部在同一分支修完 → **Approve**

Mandatory checks: Rule 7 PASS（code-list update only，`SUBTITLE_EXTRACT_WAIT_ABORTED` 已補進 project-context 第 296 行清單與 `errors_test.go` 的 wire-format 表，prefix 17 不變）· Rule 11 PASS（`WithAutoExtractTimeout` 收窄為 `func(string) time.Duration`）· Rule 14 PASS（gate 建一次）· Rule 19 PASS（`services.ExtractSlot` port 定義在 services，具體 gate 由 main.go 轉接，services 不 import subtitle）· Rule 20 N/A。

### Action Items

- [x] [H1] item deadline 被排隊時間吃掉 → 93 GB 檔仍會被判真失敗並三振停權。`Acquire` 回傳 `waited`；ffmpeg 被**呼叫端** deadline 殺死且本次曾排隊 → 額外鏈上 `ErrSubtitleExtractWaitAborted`（理由：預算完整時我們自己的 bound 一定先到，所以「呼叫端 deadline + 曾排隊」＝預算被搶走，與檔案無關）。測試涵蓋「曾排隊 → wait-abort」與「沒排隊 → 仍是檔案失敗」。
- [x] [M2] 「等待抽軌」訊息只發一次、永不清除（拿到 slot 後整段抽軌都顯示排隊中），且 fast-path 競態會誤報。`Acquire` 進場時回呼 `onWait(0)`，`ProcessItem` 收到 0 就把訊息換回「抽取內嵌字幕中…」；`onWait` 契約寫進 doc comment，三條測試釘住。
- [x] [M3] ASR 音軌抽取也是 ffmpeg、同一顆碟、未進閘門。新增 `services.ExtractSlot` 窄 port + `WithAudioExtractSlot`，`main.go` 用 `extractSlotAdapter` 轉接同一個 gate（Rule 19）。**它的 5 分鐘 timeout 仍非 size-aware** → 立案 `backlog-asr-audio-extract-size-aware-timeout`（審查者明示可選此路）。
- [x] [M4] 沒設 `cmd.WaitDelay`：卡住的 ffmpeg 會永久佔住唯一 slot、讓整個 process 的抽軌靜默停擺。`cmd.WaitDelay = 10s`；另加「排隊超過 5 分鐘」的 `slog.Warn`，讓「gate 卡住」與「前面真的有大檔」在 log 裡分得開。
- [x] [M5] 閘門無優先權：使用者按下去要付錢的 consent batch 會排在背景免費 lane 的 46 分鐘後面，且 batch 進度流不帶排隊敘述。**本輪只做文件化**（`docs/deployment.md` 明寫此排序），實作立案 `backlog-extract-gate-priority`（含「release 交棒給已被取消的 waiter」這個實作陷阱的提醒）。
- [x] [M6] 逾時訊息叫人調的 knob 與文件相反、per-GB 沒有 env。新增 `SUBTITLE_EXTRACT_PER_GB_SECONDS`（預設 30，非正值／垃圾回預設）接上既有 `WithPerGBTimeout`；`effectiveTimeout` 回報**哪一個** knob 決定了 bound，訊息只點名那一個；`docs/deployment.md` 兩處改寫一致。
- [x] [L7] `queued behind %d` 事後讀計數器會把排在自己後面的人算進去。改用第一則通知捕捉到的 `queuedAhead`。
- [x] [L8] auto lane 回合統計把 wait-abort 算成 `failed`，與自己的註解打架。新增 `queue_timeouts` 計數與 `Info` 記錄，回合繼續；測試斷言第二個 item 不被牽連。
- [x] [L9] 新 sentinel 未進 Rule 7 wire-format 鎖定測試 → `subtitleSentinels()` 補上 `SUBTITLE_EXTRACT_WAIT_ABORTED` 與既有漏列的 `SUBTITLE_TARGET_NOT_WRITABLE`。
- [x] [L10] `WithAutoExtractTimeout` port 過寬 → 收窄為 `func(string) time.Duration`，`main.go` 一行 adapter。
