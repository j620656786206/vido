# Disc：「每個字的時間」改成要開才送，預設不送——配樂長的片段才不會整段聽不到對白

Status: review

**Source:** 2026-10-06／07 第三次實測（`eval-see-s01e02-asr-vs-official.md` 第三節）：同一段 10 分鐘音訊，帶 `timestamp_granularities[]` 的請求 4 次崩 3 次（0、0、30 句對白），不帶的 4 次都正常（66～84 句）。Alexyu 2026-10-07 裁定選 A：不 revert、預設關、留開關。

## Story

身為 Vido 的使用者，
我希望正式機的聽聲音行為先回到沒崩過的那一版，同時留著「字級時間」這個時間最準的選項，
這樣配樂多的影集不會整集沒字幕，而等「別把配樂送去聽」做好之後可以把它打開。

## 查到的事

- 崩的樣子：Whisper 回 38 段全是「♪♪」「-♪♪」（`music_only:28`＋10 句漏抓），或配樂被編成同一句假台詞每 30 秒一次。
- 不崩的時候時間是最準的：出現在沒人說話時 7%、p90 0.59 秒（舊版 4～14%、p90 0.7～1.0 秒，且有一次早了 9 秒）。
- 崩的根源是前 3 分半的配樂把解碼器帶進音樂模式；這對舊版也有害（漂移），只是舊版崩得溫和。

## 設計

- `WhisperClient.wordTimestamps bool`（預設 false）＋ `WithWhisperWordTimestamps(on)`；`transcribeVerbose` 只在 `c.wordTimestamps && !wordTimestampsUnsupported` 時送欄位。
- `config.ASRWordTimestamps` ← env `VIDO_ASR_WORD_TIMESTAMPS`（預設 false），`main.go` 接進 `NewASRProviderHolder`。
- 既有的「4xx 退回一次並記住」邏輯不動，只在開啟時生效。
- README／docs/development.md 環境變數表各加一列（雙語規則）。

## Acceptance Criteria

1. 沒設環境變數：請求 body **沒有** `timestamp_granularities[]`（跟 #698 之前位元相同）。
2. `VIDO_ASR_WORD_TIMESTAMPS=true`：行為同 #698（送欄位、4xx 退回一次、記住）。
3. `go test ./...`、vet、staticcheck、lint:all 全綠。
4. 文件：README.md、docs/development.md 的環境變數表有這個開關與「為什麼預設關」。

## Tasks / Subtasks

- [x] T1 欄位＋option＋config＋main 接線（AC #1、#2）
- [x] T2 測試：預設不送（新）、既有字級測試改成明確開啟（AC #1、#2）
- [x] T3 文件兩份（AC #4）
- [x] T4 檢查（AC #3）

## Dev Notes

- 不要刪 #698 的程式：切短音檔（`disc-2026-10-asr-chunk-at-silence`）做完要花 $0.3 重測，不崩就把預設翻成 true。
- 自架引擎（Speaches 等）是否也會崩不知道；開關是 process-wide，由部署者決定。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- 🔗 AC Drift: FOUND — `disc-2026-10-asr-coarse-timestamps` AC #1「verbose_json 請求帶兩個 granularity」改為「開啟時才帶」；理由與數據見 eval 第三節。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

- `disc-2026-10-asr-music-only-dash`（同 PR 修）、`disc-2026-10-asr-chunk-at-silence`、`disc-2026-10-asr-vad-pretrim`、`disc-2026-10-series-credits-empty`（皆 backlog，見 sprint-status）。

### File List

- `apps/api/internal/ai/whisper.go`、`whisper_words_test.go`（改）
- `apps/api/internal/config/config.go`、`apps/api/cmd/api/main.go`（改）
- `README.md`、`docs/development.md`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-word-timestamps-default-off.md`（新）、`eval-see-s01e02-asr-vs-official.md`、`sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日：開關＋預設關＋文件；全綠，狀態 review。 |
