# Disc：人名提示改成要開才送——六次實測從沒修對一個人名，還偶爾把名單念進字幕

Status: done

**Source:** 2026-10-07 第五、六次實測（`eval-see-s01e02-asr-vs-official.md`）。Alexyu 2026-10-07 定的規則：「沒改善就把提示預設關掉」。

## Story

身為 Vido 的使用者，我希望聽聲音生成的字幕裡不會出現演員／角色名單這種不是台詞的句子；人名一致靠翻譯側的名詞表處理。

## 設計

- `config.ASRNamePrompt` ← `VIDO_ASR_NAME_PROMPT`（預設 false）；`TranscriptionService.SetASRNamePrompt`；關閉時不查演員表、不組提示、不送。
- `prompt_echo` 的名單判斷補「多字名字的單一個字」（Kane／Jun）也算名字——開關打開時用得到。
- README／docs/development.md／.env.example 各加一列。

## Acceptance Criteria

1. 預設：log 沒有 `asr prompt built`，請求不帶 `prompt`。
2. `VIDO_ASR_NAME_PROMPT=true`：行為同 #704。
3. 「Kane, Wren, Tamacti Jun, Kofun, Haniwa, Wren, Lord Harlan, Paris」被記 `prompt_echo`。
4. `go test ./...`、lint:all 全綠。

## Dev Agent Record

- Claude Fable 5.1。🔗 AC Drift: FOUND — `disc-2026-10-asr-proper-names-inconsistent` 的「聽」那半改成 opt-in；「翻」那半（名詞表）成為主線 → `disc-2026-10-mine-shift-range-too-narrow` 升 P1。📎 Contract Stamps: NONE。🎭 A11y: N/A。🎨 UX: SKIPPED。

### File List

- `apps/api/internal/config/config.go`、`config_test.go`、`apps/api/internal/services/transcription_service.go`、`apps/api/cmd/api/main.go`、`apps/api/internal/ai/whisper_segments.go`、`asr_prompt_test.go`、`README.md`、`docs/development.md`、`.env.example`（改）


### 正式機驗證 2026-10-11

- 環境：NAS 正式機 `Vido` 跑 `main-959ac69`（映像 `sha256:2fe55178…`，ghcr `vido@sha256:12ab5761…`，Docker workflow run 37863292141 對 `959ac698` 成功）。容器 2026-10-10 00:45 就已是這個映像，`docker pull` 後 digest 不變，所以沒有再跑 `rebuild_container`；DB 已先備份到 `/mnt/cache/appdata/vido/backups/pre-verify-20261011-2155/`（db／wal／shm＋`.backup` 一致副本）。
- 要花錢的項目在丟棄容器 `vido-evalclip`（DB 是上述備份的副本，`/scratch` 掛載，沒設 `VIDO_ASR_*`）跑；正式 Vido 沒按生成。材料與 log：NAS `/mnt/cache/appdata/vido-eval-clip/verify-20261011/`（`clip.log`、`full.log`、SRT）。
- 丟棄容器跑 10 分鐘片段（env 沒有任何 `VIDO_ASR_*`）：`clip.log` 裡 `asr prompt built` **0 行**。
- **判定：通過 → done。**

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-07 | create-story＋dev 同日：開關＋預設關＋名單規則補字；全綠，狀態 review。 |
