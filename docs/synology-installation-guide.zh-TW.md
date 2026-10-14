# Synology DSM／Container Manager 安裝指南

這份指南適用於已安裝 Container Manager 的 Synology NAS。Vido 使用公開的
multi-architecture Docker image，不需要在 NAS 上安裝 Node、Go 或 ffmpeg，
也不需要自行 build 原始碼。

## 前置準備

- DSM 7.2.1 或更新版本（Container Manager Project 需求）
- 已從套件中心安裝 **Container Manager**
- NAS CPU 必須支援 `linux/amd64` 或 `linux/arm64`
- 一個媒體資料夾，例如 `/volume1/video`。**這個資料夾要能寫入**：Vido 產生的字幕會存在影片旁邊
- 要用 AI 字幕的話，準備一把翻譯金鑰：**Claude 或 Gemini 擇一**（Gemini 用你自己的額度，免費方案 Vido 會自動放慢）。完全沒有字幕的片要先聽聲音，另外需要 ElevenLabs（建議）或 OpenAI 的金鑰
- TMDb 金鑰**不用準備**，image 裡已經內建；想用自己的再填 `TMDB_API_KEY`

## 建立資料夾

在 File Station 建立：

```text
/volume1/docker/vido/data
/volume1/docker/vido/backups
```

請確認 Container Manager 可讀寫這兩個資料夾**和媒體資料夾**。
若你的 NAS 使用不同的 UID／GID，請在 Compose 的 `PUID`／`PGID` 改成資料夾擁有者；那個使用者要對媒體資料夾有寫入權限。

## 建立 Project

1. 開啟 **Container Manager → Project → Create**。
2. 專案名稱輸入 `vido`。
3. 將 [`docker-compose.nas.yml`](docker-compose.nas.yml) 內容貼到 Compose 欄位。
4. 將三個 host path 改成：

   ```yaml
   - /volume1/docker/vido/data:/vido-data
   - /volume1/docker/vido/backups:/vido-backups
   - /volume1/video:/media
   ```

   媒體那行**不要加 `:ro`**，否則字幕寫不進去。

5. 填金鑰，兩種方式擇一：
   - 直接寫在 Compose 的環境變數（`CLAUDE_API_KEY` 或 `GEMINI_API_KEY`；聽聲音再加 `ELEVENLABS_API_KEY`）。
   - 先設 `ENCRYPTION_KEY`（**剛好 32 個字**的隨機字串），裝好後在網頁的「設定 → 金鑰設定」填；沒設這把的話網頁存不了金鑰。
6. 要分享給家人或從外面連回來，設 `VIDO_AUTH_PASSWORD`，整個 Vido 就會要密碼登入。
7. 建立並啟動 Project。

開啟 `http://<NAS-IP>:8088`。第一次會進六步設定精靈（歡迎・qBittorrent・媒體庫・API 金鑰・匿名回報・完成），每一步都可以先跳過、之後在「設定」裡補。

第一次部署前，請先在 Project 的環境變數中確認 `VIDO_PORT`（預設為 `8088`）沒有和其他服務衝突。

## 更新

在 Project 頁面停止 Vido，重新拉取 `ghcr.io/j620656786206/vido:latest` 後再啟動。
`latest` 會隨主線更新；需要可回滾時，請改用指定版本 tag（格式是 `0.1.1`，**沒有 `v`**；也可以用 `0.1` 跟著小版本更新）。
更新前請先備份 `/volume1/docker/vido/data` 與 `/volume1/docker/vido/backups`。

最簡單的備份方式是停止 Project 後，將這兩個資料夾複製到另一個儲存區；不要只備份容器本身。

## 常見問題

- **容器不健康**：先查看 Container Manager 的 Log，並確認 data／backups 可寫入。
- **找不到影片**：確認 `/volume1/video` 是實際包含媒體檔案的資料夾，且容器掛載為 `/media`。
- **字幕產生失敗、訊息說「資料夾無法寫入」**：媒體資料夾掛成了唯讀（`:ro`），或 `PUID`／`PGID` 那個使用者沒有寫入權限。
- **網頁存不了金鑰**：沒設 `ENCRYPTION_KEY`，或長度不是剛好 32 個字。
- **連接埠衝突**：把 `8088:8080` 的左側改成未使用的 NAS port，例如 `18088:8080`。
- **遠端存取**：請使用 Synology 反向代理、VPN、Tailscale 或 Cloudflare Access，並設定 `VIDO_AUTH_PASSWORD`；沒設密碼時不要把 8088 port 暴露到網際網路。
