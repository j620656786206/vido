# Unraid 安裝指南

> **最後更新：** 2026-03-26

## 前置需求

- Unraid 6.x 或更新版本，已啟用 Docker
- 已安裝 Community Applications 外掛（建議，可一鍵安裝）
- TMDb API 金鑰，從 <https://www.themoviedb.org/settings/api> 取得（選用但建議）

## 方法 A：Community Applications（建議）

1. 開啟 Unraid 網頁介面，前往 **Apps**
2. 搜尋 **Vido**
3. 點擊 **Install**
4. 設定必要的路徑（請參閱下方[設定](#設定)）
5. 點擊 **Apply**

## 方法 B：手動 Docker 設定

1. 在 Unraid 網頁介面中，前往 **Docker** > **Add Container**
2. 設定以下欄位：

| 欄位             | 值                                            |
| ---------------- | --------------------------------------------- |
| Name             | `Vido`                                        |
| Repository       | `ghcr.io/j620656786206/vido:main`             |
| Network Type     | `Bridge`                                      |
| WebUI            | `http://[IP]:[PORT:8088]`                     |
| Extra Parameters | `--read-only --tmpfs /tmp:size=64M,mode=1777` |

3. 依照下方[設定](#設定)章節新增 port、path 和變數對應
4. 點擊 **Apply**

## 設定

### 連接埠

| 名稱       | 容器連接埠 | 主機連接埠       | 協定 |
| ---------- | ---------- | ---------------- | ---- |
| WebUI Port | `8080`     | `8088`（可自訂） | TCP  |

Vido 在單一連接埠上同時提供網頁介面和 API。

### 路徑（磁碟區）

| 名稱          | 容器路徑        | 主機路徑                             | 模式     | 說明                              |
| ------------- | --------------- | ------------------------------------ | -------- | --------------------------------- |
| App Data      | `/vido-data`    | `/mnt/user/appdata/vido`             | 讀寫     | SQLite 資料庫、快取和應用程式資料 |
| Backups       | `/vido-backups` | `/mnt/user/appdata/vido/backups`     | 讀寫     | 資料庫備份和中繼資料匯出          |
| Media Library | `/media`        | 你的媒體路徑（如 `/mnt/user/media`） | **唯讀** | 你的電影和影集資料庫              |

### 環境變數

**一般顯示：**

| 變數                    | 預設值   | 說明                                        |
| ----------------------- | -------- | ------------------------------------------- |
| `TMDB_API_KEY`          | （空白） | TMDb API 金鑰，用於取得中繼資料、海報和圖片 |
| `TMDB_DEFAULT_LANGUAGE` | `zh-TW`  | 預設中繼資料語言（ISO 639-1 格式）          |

**進階（點擊「Show more settings」）：**

| 變數             | 預設值    | 說明                                           |
| ---------------- | --------- | ---------------------------------------------- |
| `GIN_MODE`       | `release` | 設為 `debug` 可用於疑難排解                    |
| `AI_PROVIDER`    | `gemini`  | AI 檔名解析提供者：`gemini` 或 `claude`        |
| `GEMINI_API_KEY` | （空白）  | Google Gemini API 金鑰，用於 AI 解析           |
| `CLAUDE_API_KEY` | （空白）  | Anthropic Claude API 金鑰（Gemini 的替代方案） |

## 安裝後驗證

1. 從 Docker 分頁啟動 Vido 容器
2. 等待健康檢查通過（綠色圖示，通常在 30 秒內）
3. 開啟 WebUI：`http://[你的 Unraid IP]:8088`
4. 前往**設定**頁面設定 TMDb API 金鑰
5. 前往媒體庫並觸發第一次掃描

### 健康檢查

容器內建健康檢查，每 30 秒查詢 `GET /health`。如果連續 3 次檢查失敗，Docker 會將容器標記為不健康。

## 疑難排解

### 容器無法啟動

- 檢查 Docker 日誌中的錯誤訊息（點擊 Vido 容器圖示 > **Log**）
- 確認 App Data 路徑存在且可寫入
- 確保連接埠 8088 未被其他容器使用（8080 通常已被 qBittorrent 佔用）

### 找不到媒體檔案

- 確認 Media Library 路徑指向 Unraid 伺服器上的正確目錄
- 媒體路徑以唯讀方式掛載 — Vido 只讀取，不會修改你的媒體檔案
- 支援的影片格式：`.mkv`、`.mp4`、`.avi`、`.m4v`、`.ts`、`.wmv`、`.flv`、`.webm`、`.mov`

### 權限錯誤

- 容器以使用者 `vido`（UID 1000、GID 1000）身分執行
- 確保 App Data 和 Backups 目錄可被 UID 1000 存取
- 如有需要，從 Unraid 終端機執行 `chown -R 1000:1000 /mnt/user/appdata/vido`

### 資料庫錯誤

- SQLite 資料庫儲存在 `/vido-data/vido.db`
- 預設啟用 WAL 模式以提升效能
- 如果資料庫損壞，可從 `/vido-backups` 中的備份還原

## 安全注意事項

- 容器以**非 root 使用者**身分執行（UID 1000）
- 根檔案系統為**唯讀**（`--read-only` 旗標）
- 媒體檔案以**唯讀**方式掛載 — Vido 不會修改你的媒體
- API 金鑰在 Unraid UI 中已遮蔽（不以明文顯示）
