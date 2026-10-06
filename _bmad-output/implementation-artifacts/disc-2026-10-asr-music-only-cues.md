# Disc：聽聲音生成的字幕不再出現只有「♪♪」的句子

Status: review

**Source:** Alexyu 2026-10-06 party mode（「字幕裡面有很多的音樂符號，這個應該也要修」）。See S01E02 10/5 聽聲音那次 659 句裡 **22 句只有 ♪♪**；10/6 改翻片內英文字幕後是 0 句，但**沒有片內字幕、只能聽聲音的片**仍會有。

## Story

身為 Vido 的使用者，
我希望聽聲音生成的字幕不要在音樂時跳出一行「♪♪」，
這樣看片時畫面乾淨，也不用為這些句子付翻譯費。

## 查到的事（2026-10-06，逐一開檔確認）

- 抽字幕那條路：`subtitle/sdh_filter.go:112-124` 的 `isMusicOnly` 在翻譯前把只有音符的句子丟掉（sub-6-4），所以片內字幕路線沒有這個問題。
- 聽聲音那條路：`ai/whisper_segments.go:172-235` 的 `filterHallucinations` 有四條規則（silence／repetition／repeat_run／tail），**沒有**音符規則；whisper 在配樂段落會吐 `♪♪`，而且常常連續好幾段——第 1 段 log 的 `repeat_run:28` 有一部分就是連續 ♪♪ 被當成重複句刪掉，剩下的第一段就留下來變成 22 句。
- 兩條路看不到彼此（Rule 19：`services` 不能 import `subtitle`；`ai` 是 leaf，`subtitle` 可以 import `ai`）。
- `whisper.go:323-331` 用 `DropRatio()` 超過 0.2 就 Warn；音符句算進去會讓配樂多的段落容易觸發這個 Warn，但那只是 log。

## 設計

把音符判斷搬到 leaf 套件 `ai`（`ai.IsMusicOnlyText`、`ai.MusicMarks`），`subtitle.isMusicOnly` 改呼叫它（一份實作）；`filterHallucinations` 加第 0 條規則 `music_only`，排在 silence 之前，log 的 `dropped_by_reason` 會多一個 `music_only` 鍵。不動門檻、不動其他規則。

## Acceptance Criteria

1. `ai.IsMusicOnlyText`：`♪`、`♪♪`、`♪ ♪`、`♫`、`♬`、`#`、含零寬字元的 ♪ → true；空字串、`♪ lyrics ♪`、`Hello`、`# comment` → false。
2. `filterHallucinations`：只有音符的段落以 `music_only` 丟掉；`♪ Happy birthday to you ♪`（有歌詞）保留；連續五段 ♪♪ 全部記 `music_only`，不再有一段被記成 `repeat_run`。
3. `subtitle.FilterSDH` 既有測試（`sdh_filter_test.go`）全部不改照常通過。
4. `go test ./...`、`go vet`、`staticcheck`、`lint:all` 全綠。
5. **NAS 實測（部署後補、要花錢）：** 找一部沒有片內字幕的片聽聲音生成，輸出裡 `♪♪`-only 句數為 0；或等下次任何 ASR 路線的 run，看 log `dropped_by_reason` 有 `music_only`。

## Tasks / Subtasks

- [x] T1 `ai.IsMusicOnlyText`＋`MusicMarks`；`subtitle.isMusicOnly` 改呼叫（AC #1、#3）
- [x] T2 `filterHallucinations` R0 `music_only`＋測試（AC #2）
- [x] T3 檢查（AC #4）

## Dev Notes

- 不要在 `services`（transcription_service.go）或翻譯前另寫一份判斷——那會是第三份；規則就放在吐出 ♪♪ 的那一層（whisper 過濾器）。
- 不要把 ♪♪ 當 `silence` 處理：它們的 `no_speech_prob` 常常不高（有配樂），走 R1 抓不到。
- `#` 也算音符是 SDH 慣例；whisper 不會吐 `#`，共用同一張表無害。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Completion Notes List

- 🔗 AC Drift: FOUND — sub-6-4 AC（`FilterSDH` 丟純音符句）行為不變，實作搬到 `ai`；9R-5 AC #3（四條幻覺規則）多一條 R0，既有四條與門檻不動。
- 📎 Contract Stamps: NONE（無 wire contract）。
- 🎭 A11y Pre-Flight: N/A (100% backend)
- 🎨 UX Verification: SKIPPED — no UI changes

### Discovery Triage

N/A — no out-of-scope work discovered

### File List

- `apps/api/internal/ai/music_only.go`（新）
- `apps/api/internal/ai/music_only_test.go`（新）
- `apps/api/internal/ai/whisper_segments.go`（改）
- `apps/api/internal/subtitle/sdh_filter.go`（改：改用 `ai.IsMusicOnlyText`）
- `_bmad-output/implementation-artifacts/disc-2026-10-asr-music-only-cues.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | create-story＋dev 同日：音符判斷搬到 `ai`，whisper 過濾器加 `music_only`；全綠，狀態 review。 |
