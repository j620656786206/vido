# Synology DSM／Container Manager 安裝指南

這份指南適用於已安裝 Container Manager 的 Synology NAS。Vido 使用公開的
multi-architecture Docker image，不需要在 NAS 上安裝 Node、Go 或 ffmpeg，
也不需要自行 build 原始碼。

## 前置準備

- DSM 7.2.1 或更新版本（Container Manager Project 需求）
- 已從套件中心安裝 **Container Manager**
- NAS CPU 必須支援 `linux/amd64` 或 `linux/arm64`
- 一個媒體資料夾，例如 `/volume1/video`
- （選用）TMDb API key；沒有 key 仍可啟動，但 metadata 功能會受限

## 建立資料夾

在 File Station 建立：

```text
/volume1/docker/vido/data
/volume1/docker/vido/backups
```

請確認 Container Manager 可讀寫這兩個資料夾；媒體資料夾只需要讀取權限。
若你的 NAS 使用不同的 UID／GID，請在 Compose 的 `PUID`／`PGID` 改成資料夾擁有者。

## 建立 Project

1. 開啟 **Container Manager → Project → Create**。
2. 專案名稱輸入 `vido`。
3. 將 [`docker-compose.nas.yml`](docker-compose.nas.yml) 內容貼到 Compose 欄位。
4. 將三個 host path 改成：

   ```yaml
   - /volume1/docker/vido/data:/vido-data
   - /volume1/docker/vido/backups:/vido-backups
   - /volume1/video:/media:ro
   ```

5. 如有需要，填入 `TMDB_API_KEY` 或其他 AI API key。
6. 建立並啟動 Project。

開啟 `http://<NAS-IP>:8088`，看到首頁後即可執行初始設定與媒體庫掃描。

第一次部署前，請先在 Project 的環境變數中確認 `VIDO_PORT`（預設為 `8088`）沒有和其他服務衝突。

## 更新

在 Project 頁面停止 Vido，重新拉取 `ghcr.io/j620656786206/vido:latest` 後再啟動。
`latest` 會隨主線更新；需要可回滾時，請改用指定版本 tag（例如 `v0.1.0-beta.1`）。
更新前請先備份 `/volume1/docker/vido/data` 與 `/volume1/docker/vido/backups`。

最簡單的備份方式是停止 Project 後，將這兩個資料夾複製到另一個儲存區；不要只備份容器本身。

## 常見問題

- **容器不健康**：先查看 Container Manager 的 Log，並確認 data／backups 可寫入。
- **找不到影片**：確認 `/volume1/video` 是實際包含媒體檔案的資料夾，且容器掛載為 `/media`。
- **連接埠衝突**：把 `8088:8080` 的左側改成未使用的 NAS port，例如 `18088:8080`。
- **遠端存取**：請使用 Synology 反向代理、VPN、Tailscale 或 Cloudflare Access；不要直接把
  8088 port 暴露到網際網路。Vido 目前沒有內建登入機制。
