# Disc：音檔切短之後，「每個字的時間」預設打開

Status: review

**Source:** 2026-10-07 第四次實測（`eval-see-s01e02-asr-vs-official.md` 第四節）：切成兩分鐘一段後，字級時間開著 2/2 沒崩、時間最準。Alexyu 選 A 的第 4 步：「不崩了就把預設打開」。

## Story

身為 Vido 的使用者，我希望聽聲音生成的字幕預設就貼著演員開口出現，不用自己去找開關。

## 設計

- `config.ASRWordTimestamps` 預設 `true`（`VIDO_ASR_WORD_TIMESTAMPS=false` 可關）；client 端預設仍是關，由 app 設定。
- README／docs/development.md／.env.example 三處同步改成預設 true＋為什麼。

## Acceptance Criteria

1. 沒設環境變數 → 請求帶 `timestamp_granularities[]`；設 `false` → 不帶。
2. `go test ./...`、lint:all 全綠。
3. 正式機更新後第一次聽聲音生成的 log 沒有 `rejected word timestamps`，字幕時間貼著開口。

## Dev Agent Record

- Claude Fable 5.1。🔗 AC Drift: FOUND — `disc-2026-10-asr-word-timestamps-default-off` AC #1「預設不送」→ 預設送，理由與數據見 eval 第四節。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/config/config.go`、`config_test.go`、`apps/api/internal/ai/whisper.go`（註解）、`README.md`、`docs/development.md`、`.env.example`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日：預設翻 true＋文件；全綠，狀態 review。 |
