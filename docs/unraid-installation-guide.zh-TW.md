# Unraid 安裝指南

> **最後更新：** 2026-10-14

## 前置需求

- Unraid 6.x 或更新版本，已啟用 Docker
- 一個媒體資料夾（例如 `/mnt/user/media`）。**要能寫入**：Vido 產生的字幕會存在影片旁邊
- 要用 AI 字幕的話，一把翻譯金鑰：**Claude 或 Gemini 擇一**（Gemini 用你自己的額度，免費方案 Vido 會自動放慢）。完全沒有字幕的片要先聽聲音，另外需要 ElevenLabs（建議）或 OpenAI 的金鑰
- TMDb 金鑰**不用準備**，image 裡已經內建；想用自己的再填 `TMDB_API_KEY`

Vido **還沒上架 Community Applications**。請用這個 repo 的模板安裝（方法 A），或手動設定（方法 B）。

## 方法 A：用這個 repo 的模板（建議）

1. 在 Unraid 網頁介面前往 **Docker**，捲到最底下打開 **Template repositories**
2. 加入 `https://github.com/j620656786206/vido`，按 **Save**
3. 按 **Add Container**，在 **Template** 下拉選 **Vido**
4. 依照下方[設定](#設定)填路徑和金鑰
5. 按 **Apply**

模板沒出現的話改用方法 B——模板檔是 [`unraid-template/vido.xml`](../unraid-template/vido.xml)，下面每個欄位都跟它一致。

## 方法 B：手動 Docker 設定

1. 在 Unraid 網頁介面中，前往 **Docker** > **Add Container**
2. 設定以下欄位：

| 欄位             | 值                                            |
| ---------------- | --------------------------------------------- |
| Name             | `Vido`                                        |
| Repository       | `ghcr.io/j620656786206/vido:latest`           |
| Network Type     | `Bridge`                                      |
| WebUI            | `http://[IP]:[PORT:8088]`                     |
| Extra Parameters | `--read-only --tmpfs /tmp:size=64M,mode=1777` |

3. 依照下方[設定](#設定)章節新增 port、path 和變數對應
4. 按 **Apply**

## 設定

### 連接埠

| 名稱       | 容器連接埠 | 主機連接埠       | 協定 |
| ---------- | ---------- | ---------------- | ---- |
| WebUI Port | `8080`     | `8088`（可自訂） | TCP  |

Vido 在單一連接埠上同時提供網頁介面和 API。

### 路徑（磁碟區）

| 名稱          | 容器路徑        | 主機路徑                             | 模式     | 說明                                                       |
| ------------- | --------------- | ------------------------------------ | -------- | ---------------------------------------------------------- |
| App Data      | `/vido-data`    | `/mnt/user/appdata/vido`             | 讀寫     | SQLite 資料庫、快取和應用程式資料                          |
| Backups       | `/vido-backups` | `/mnt/user/appdata/vido/backups`     | 讀寫     | 資料庫備份和中繼資料匯出                                   |
| Media Library | `/media`        | 你的媒體路徑（如 `/mnt/user/media`） | **讀寫** | 你的電影和影集。產生的字幕會寫在影片旁邊，所以不能設成唯讀 |

### 環境變數

**金鑰（都是選填，一把都沒填 Vido 也能啟動）：**

| 變數                 | 預設值   | 說明                                                                                             |
| -------------------- | -------- | ------------------------------------------------------------------------------------------------ |
| `CLAUDE_API_KEY`     | （空白） | 字幕翻譯（預設用它）。要用 AI 字幕，Claude／Gemini 擇一                                          |
| `GEMINI_API_KEY`     | （空白） | 用你自己的 Gemini 額度翻字幕，也用於 AI 檔名解析                                                 |
| `ELEVENLABS_API_KEY` | （空白） | 沒字幕的片聽聲音用（ElevenLabs Scribe，每小時約 US$0.26，含人名提示）。比 Whisper 準，建議用這把 |
| `OPENAI_API_KEY`     | （空白） | 用 OpenAI Whisper 聽聲音（每小時約 US$0.36）；沒填 ElevenLabs 時用它                             |
| `TMDB_API_KEY`       | （空白） | 你自己的 TMDb 金鑰。留空就用 image 內建的                                                        |
| `ENCRYPTION_KEY`     | （空白） | **剛好 32 個字。** 想在網頁上（設定 → 金鑰設定）存金鑰就要設；沒設的話那一頁只能看不能存         |
| `VIDO_AUTH_PASSWORD` | （空白） | 整個 Vido 的共用登入密碼。要分享給別人或從外面連回來之前先設                                     |

**行為：**

| 變數                          | 預設值      | 說明                                                                             |
| ----------------------------- | ----------- | -------------------------------------------------------------------------------- |
| `PUID`／`PGID`                | `99`／`100` | 容器用哪個使用者執行（Unraid 的 `nobody:users`）。要能寫入媒體資料夾             |
| `TMDB_DEFAULT_LANGUAGE`       | `zh-TW`     | 預設中繼資料語言（ISO 639-1 格式）                                               |
| `VIDO_SUBTITLE_PIPELINE_MODE` | `pipeline`  | `pipeline` 先讀片內英文字幕、沒有才聽聲音；`legacy` 回到只搜尋線上字幕的舊引擎   |
| `VIDO_ASR_PROVIDER`           | `auto`      | 兩把聽寫金鑰都填時用哪家：`auto`（有 ElevenLabs 就用它）、`elevenlabs`、`openai` |
| `ASR_BASE_URL`／`ASR_MODEL`   | （空白）    | 改用自架的語音辨識服務（例如 Speaches），不用雲端金鑰                            |
| `AI_RUN_BUDGET_USD`           | `5`         | 每次執行最多花幾美元。超過會暫停，做好的部分保留，下次只做剩下的                 |
| `AI_PROVIDER`                 | `gemini`    | AI 檔名解析用哪家：`gemini` 或 `claude`                                          |
| `VIDO_LOG_LEVEL`              | `info`      | 設為 `debug` 可用於疑難排解                                                      |

## 安裝後驗證

1. 從 Docker 分頁啟動 Vido 容器
2. 等待健康檢查通過（綠色圖示，通常在 30 秒內）
3. 開啟 WebUI：`http://[你的 Unraid IP]:8088`
4. 第一次會進六步設定精靈（歡迎・qBittorrent・媒體庫・API 金鑰・匿名回報・完成），每一步都可以先跳過、之後在**設定**裡補
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
- 支援的影片格式：`.mkv`、`.mp4`、`.avi`、`.m4v`、`.ts`、`.wmv`、`.flv`、`.webm`、`.mov`

### 字幕產生失敗、訊息說「資料夾無法寫入」

- 媒體路徑掛成了唯讀，或 `PUID`／`PGID` 那個使用者沒有寫入權限。字幕會存在影片旁邊，所以媒體資料夾一定要能寫

### 設定頁存不了金鑰

- 沒設 `ENCRYPTION_KEY`，或長度不是剛好 32 個字

### 權限錯誤

- 容器以 root 啟動、先把 `/vido-data` 和 `/vido-backups` 的擁有者改成 `PUID`／`PGID`，再切換成那個使用者執行（模板預設 `99:100`，也就是 Unraid 的 `nobody:users`）
- 如果你改了 `PUID`／`PGID`，請確認那個使用者對 App Data、Backups **和媒體路徑**都能讀寫

### 資料庫錯誤

- SQLite 資料庫儲存在 `/vido-data/vido.db`
- 預設啟用 WAL 模式以提升效能
- 如果資料庫損壞，可從 `/vido-backups` 中的備份還原

## 安全注意事項

- 啟動完成後程序以 `PUID`／`PGID` 執行，不是 root
- 根檔案系統為**唯讀**（`--read-only` 旗標）；只有 `/tmp`、資料、備份和媒體路徑可寫
- Vido 只會把**字幕檔**寫進媒體資料夾（放在影片旁邊），不會修改或刪除影片
- API 金鑰在 Unraid UI 中已遮蔽（不以明文顯示）
- 要把連接埠開到家用網路以外之前，先設 `VIDO_AUTH_PASSWORD`
