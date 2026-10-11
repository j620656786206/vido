# Disc：音檔切短之後，「每個字的時間」預設打開

Status: done

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


### 正式機驗證 2026-10-11

- 環境：NAS 正式機 `Vido` 跑 `main-959ac69`（映像 `sha256:2fe55178…`，ghcr `vido@sha256:12ab5761…`，Docker workflow run 37863292141 對 `959ac698` 成功）。容器 2026-10-10 00:45 就已是這個映像，`docker pull` 後 digest 不變，所以沒有再跑 `rebuild_container`；DB 已先備份到 `/mnt/cache/appdata/vido/backups/pre-verify-20261011-2155/`（db／wal／shm＋`.backup` 一致副本）。
- 要花錢的項目在丟棄容器 `vido-evalclip`（DB 是上述備份的副本，`/scratch` 掛載，沒設 `VIDO_ASR_*`）跑；正式 Vido 沒按生成。材料與 log：NAS `/mnt/cache/appdata/vido-eval-clip/verify-20261011/`（`clip.log`、`full.log`、SRT）。
- 同一次：log 沒有「rejected word timestamps」；程式沒有「已送出」的 info 行，所以用結果判斷——70 句裡只有 1 句長度是整秒（舊版整秒時間 93%），時間到毫秒（例：4:22.066–4:22.686「Whoa!」）。
- 對官方英文字幕前 600 秒：開口時間差中位 +0.25 秒、p90 +0.86 秒（第四次實測開著是 0.65～0.74、關著 0.90～0.97）；出現在沒人說話時 1/70（1%）；早 2 秒以上只有 1 句（5:19.8「Who's there?」，官方 5:22.3）。8:17 那句在 8:20.5（官方 8:21.6），沒有提早 9 秒的漂移。
- **判定：通過 → done。**

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日：預設翻 true＋文件；全綠，狀態 review。 |
