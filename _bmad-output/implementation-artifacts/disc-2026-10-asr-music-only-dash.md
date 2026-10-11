# Disc：前面帶減號的「-♪♪」也要算音樂符號句

Status: done

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


### 正式機驗證 2026-10-11

- 環境：NAS 正式機 `Vido` 跑 `main-959ac69`（映像 `sha256:2fe55178…`，ghcr `vido@sha256:12ab5761…`，Docker workflow run 37863292141 對 `959ac698` 成功）。容器 2026-10-10 00:45 就已是這個映像，`docker pull` 後 digest 不變，所以沒有再跑 `rebuild_container`；DB 已先備份到 `/mnt/cache/appdata/vido/backups/pre-verify-20261011-2155/`（db／wal／shm＋`.backup` 一致副本）。
- 要花錢的項目在丟棄容器 `vido-evalclip`（DB 是上述備份的副本，`/scratch` 掛載，沒設 `VIDO_ASR_*`）跑；正式 Vido 沒按生成。材料與 log：NAS `/mnt/cache/appdata/vido-eval-clip/verify-20261011/`（`clip.log`、`full.log`、SRT）。
- 同一次：產出的 `clip.en.srt`／`clip.zh-Hant.srt` 沒有只含「♪♪」或「-♪♪」的句子（0/70）；配樂段被丟掉：`dropped_by_reason=map[music_only:8]`、`map[music_only:12]`、`map[music_only:5]`。
- 順帶看到：Whisper 把打鬥時的喊叫寫成「$%&!」／「-$%&!」（3 句），翻譯也照抄成「$%&！」→ 另開 `disc-2026-10-asr-symbol-only-cues`。
- **判定：通過 → done。**

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日；全綠，狀態 review。 |
