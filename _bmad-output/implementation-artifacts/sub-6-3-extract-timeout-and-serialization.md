# Story 6.3: 抽軌 timeout 可調 + 抽軌階段序列化 —— 4K remux 片庫不再整批不能用（後端）

Status: review

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

### File List

- `apps/api/internal/subtitle/extract_gate.go`、`extract_gate_test.go`、`extractor_gate_test.go`（new）
- `apps/api/internal/subtitle/extractor.go`、`errors.go`、`process_item.go`、`progress_sse.go`、`auto_generation.go`（modified）+ `process_item_test.go`、`progress_sse_test.go`、`auto_generation_test.go`
- `apps/api/internal/config/config.go`（modified）+ `config_test.go`
- `apps/api/cmd/api/main.go`（modified）
- `docs/deployment.md`、`project-context.md`、`_bmad-output/implementation-artifacts/sub-6-3-extract-timeout-and-serialization.md`、`sprint-status.yaml`
