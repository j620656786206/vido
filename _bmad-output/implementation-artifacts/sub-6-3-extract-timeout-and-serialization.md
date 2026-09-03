# Story 6.3: 抽軌 timeout 可調 + 抽軌階段序列化 —— 4K remux 片庫不再整批不能用（後端）

Status: ready-for-dev

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

- [ ] **Task 1 — 可調 timeout（AC: #1, #3）**
  - [ ] config + `NewExtractor` 接線（`main.go:676/925` 兩處）；size-aware 計算
- [ ] **Task 2 — ExtractGate（AC: #2）**
  - [ ] `subtitle/extract_gate.go`；`Extractor` 注入；SSE 等待訊息
- [ ] **Task 3 — 測試（AC: #4）**

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

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
