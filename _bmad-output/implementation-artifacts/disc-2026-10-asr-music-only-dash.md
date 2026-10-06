# Disc：前面帶減號的「-♪♪」也要算音樂符號句

Status: review

**Source:** 2026-10-06 第三次實測：10 分鐘片段崩掉那次 Whisper 回 38 段音樂符號，其中 10 段是 SDH 說話者減號開頭的「-♪♪」，`music_only` 沒抓到，進了字幕檔。

## Story

身為 Vido 的使用者，我希望「-♪♪」跟「♪♪」一樣被當成純音樂符號刪掉，字幕裡不會留下只有減號和音符的句子。

## 設計

`ai.IsMusicOnlyText` 把 `-`、`–`、`—` 當成可忽略字元（跟空白一樣）；只有減號、沒有音符仍是 false。`subtitle.FilterSDH` 同時受惠（它呼叫同一個函式）。

## Acceptance Criteria

1. `-♪♪`、`- ♪♪`、`—♪`、`♪♪-` → true；`-`、`- -`、`-Hello` → false。
2. 既有 music_only／FilterSDH 測試不變。

## Dev Agent Record

- Claude Fable 5.1；隨 `disc-2026-10-asr-word-timestamps-default-off` 同一 PR。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/ai/music_only.go`、`music_only_test.go`（改）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日；全綠，狀態 review。 |
