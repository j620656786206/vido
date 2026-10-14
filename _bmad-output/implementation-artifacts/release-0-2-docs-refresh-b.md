# Release 0.2 文件翻新 (b)：真相檔——deployment.md、PRODUCT.md、ROADMAP 說的要是現在的 Vido

Status: ready-for-dev

**Source:** 同 `release-0-2-docs-refresh-a`（2026-10-14 party mode，Alexyu 裁定 A）。負責：Paige。這三份是「Vido 是什麼、做到哪」的真相檔，之前的 story 和 ai-context 都引用它們；現在它們說的是 8 月的 Vido。

## 查到的事（2026-10-14，main `ee51f9c7`；✅ 本人開檔確認，⚠️ 只有 agent 報告）

### `docs/deployment.md`（✅ `:64`、`:318-399`、`:420` 本人 grep）
- `:64`「Official release images (`ghcr.io/…/vido`, Docker Hub)」→ 只發 GHCR（`.github/workflows/docker.yml:283` `REGISTRY_IMAGE`，⚠️ `:46`）。
- `:318`、`:321`、`:374` `curl /api-health`、`/nginx-health` → 只有 `/health`（⚠️ `main.go:1431`）；沒有 nginx。
- `:329` `"service": "vido-api"`、`:363-364` `docker-compose logs vido-api`／`vido-web`、`:399` `vido-web … vido-api:8080` → 只有一個服務 `vido`（✅ `docker-compose.yml:28`）。
- `:396` network `vido_vido-network` ⚠️ 沒定義。
- `:406-414` 用 `git pull` 重 build 更新 → NAS 使用者是 `docker compose pull`。
- `:420` `git checkout v1.0.0` → 沒這個 tag；git tag 是 `v0.1.0`、`v0.1.1`，image tag `0.1.1`。
- `:240`「media folder is mounted **read-only** for security」、`:278` → 跟 `:348`（自己說 `:ro` 會 `SUBTITLE_TARGET_NOT_WRITABLE`）矛盾。
- `:101` Claude 列已在 #788 改；`:147-149` 金鑰熱載清單缺 Gemini（⚠️ `translation_dispatcher.go:30-38` per-call 解析）；整份沒提 `GEMINI_API_KEY` 翻譯與 `VIDO_GEMINI_MIN_INTERVAL`。
- `:433` 檢查清單「Configure `TMDB_API_KEY`」→ 內建，選填。
- `:244-250` Scenario 1 用根目錄 compose（從原始碼 build，沒內建 TMDb 金鑰）→ 要說明。

### `PRODUCT.md`（✅ `:17`、`:50`、`:58`、`:60`、`:63`、`:65`、`:70`、`:89` 本人讀）
- `:17`、`:65`「無登入機制」→ `VIDO_AUTH_PASSWORD` 共用密碼登入存在（`config.go:218`）。「單人使用」的定位可以留，登入那半句要改。
- `:50`「媒體以**唯讀**掛載」→ 要可寫。
- `:58`「ASR 生字幕（Whisper）」→ ElevenLabs Scribe（預設，填了金鑰）、Whisper、自架三種。
- `:60`「抽取內嵌字幕來翻譯（目前只做到偵測）」→ 已上線（pipeline 模式，#788 起預設）。
- `:63` ⚠️「ASR 目前僅支援電影」→ 影集支援（`transcription_episode_test.go`）。
- `:66` ⚠️「陸片保留簡體字幕不轉換」vs README `:32`「轉繁體字、保留原說法」→ 以 2026-10-05 裁定為準（memory `project_cn_subtitle_policy`：OpenCC s2tw，字轉、詞不轉）。
- `:70`、`:89`「沒有正式 release、沒有公開 Docker image、沒有授權條款」→ 三者都有（v0.1.0／v0.1.1、GHCR image、Apache-2.0 `LICENSE`）。

### `ROADMAP.zh-TW.md`（✅ `:8-14` 本人讀）
- `:8`「內容抽取與翻譯還在做」→ 已上線。
- `:9`「ASR 僅支援電影；VAD 過濾開發中」→ 影集支援；幻覺過濾上線（README `:31`）；VAD 的真相是 `disc-2026-10-asr-vad-pretrim` backlog（可行性筆記：要走 sidecar）。
- `:13-14`「Request 系統、多媒體庫」在規劃中 → README `:42`、`:51` 已列為現有功能（Epic 13 大部分 done；多媒體庫看 sprint-status 的實際狀態再寫）。
- 另：ROADMAP 開頭說「以 README 為準」（08-30 精簡），這條原則保留，但列出的項目不能跟 README 矛盾。

## Acceptance Criteria

1. `docs/deployment.md`：上面每一行改正；Health／Logs／Update／Rollback 四段重寫成「單一容器、GHCR image、`docker compose pull`」的版本；補一段 Gemini 翻譯（`GEMINI_API_KEY`、`VIDO_GEMINI_MIN_INTERVAL`、免費方案會放慢）；`:240`／`:278` 改可寫。
2. `PRODUCT.md`：登入、掛載、ASR 引擎、內嵌字幕、影集 ASR、簡繁規則、release／image／授權八處改成現況，每處保留「v4 決定」之類的歷史註記但標明「2026-10 現況」。
3. `ROADMAP.zh-TW.md`：「開發中」只留真的還在做的；已上線的搬到 README 或刪；「規劃中」跟 README 不矛盾。
4. 三份檔案互相不矛盾、也不跟 README 矛盾（`:ro`、登入、金鑰、tag 四件事各只有一種說法）。
5. `pnpm run format:check` 綠。

## Tasks / Subtasks

- [ ] T1 `deployment.md`（AC 1）
- [ ] T2 `PRODUCT.md`（AC 2）
- [ ] T3 `ROADMAP.zh-TW.md`（AC 3）
- [ ] T4 交叉核對＋format（AC 4、5）

## Dev Notes

- 等 -a 合併後再開（-a 會先改 README 與指南，這裡要跟它對齊）。
- `PRODUCT.md`／`ROADMAP` 是 ai-context 引用的真相檔，改完提醒 Alexyu 回寫 `~/projects/personal/ai-context/core/projects/vido.md`。
- 白話、台灣用語。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純文件）。

### References

- `release-0-2-docs-refresh-a.md`（同一份 audit 的安裝路徑部分）。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List
