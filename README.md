# Vido

**專為繁體中文使用者打造的自架媒體管理工具。**
整合 metadata 管理、下載監控與字幕處理——繁中優先，並用 AI 解析字幕組的複雜檔名。

> **English**: Vido is a self-hosted media management tool built for Traditional Chinese users. It unifies metadata management, download monitoring, and subtitle handling, with zh-TW as a first-class citizen and AI-powered parsing of fansub filenames. Runs on your own NAS via Docker.

---

## 為什麼做這個

現有的自架工具在繁體中文情境下都有各自的缺口：

- **TMDB 的繁中 metadata 不完整** — 很多影劇只有英文或簡中資料
- **Bazarr 的 zh-TW 字幕處理不夠好** — 簡繁不分、抓到簡中字幕就當成中文字幕
- **字幕組的檔名解析不了** — `[字幕組][作品名][01][1080p][BIG5][MP4]` 這種命名，一般的 regex parser 直接放棄

Vido 把這幾件事整合成一套：**繁中 metadata 多來源 fallback、AI 解析檔名、簡轉繁字幕引擎**，跑在自己的 NAS 上。

## 目前能做什麼

- **繁中優先的 metadata** — TMDB (zh-TW) → 豆瓣 → Wikipedia 多來源 fallback，補齊繁中資料
- **AI 字幕組檔名解析** — 解析複雜命名，內建學習機制與自動重試，失敗時優雅降級
- **字幕引擎** — opencc 簡轉繁、批次處理、AI 字幕增強
- **下載監控** — 串接 qBittorrent，即時進度
- **媒體庫管理** — 遞迴掃描、多磁碟區支援
- **瀏覽與探索** — 首頁媒體牆、進階搜尋、詳情頁（雙評分、季集列表、推薦、觀看平台、預告）
- **活動中心** — 集中檢視解析／下載／字幕處理的進行狀況
- **設定引導** — 首次啟動的 setup wizard

**開發中**：Request 系統（一鍵「想要」＋ Radarr/Sonarr 串接）、字幕 pipeline 改版。

> ⚠️ **專案狀態**：積極開發中，設計為**單人使用**（無登入機制），目前主要在作者自己的 NAS 上運行。歡迎試用與回報問題，但還不建議用於關鍵用途。

## 快速開始

需要 Docker 與 Docker Compose。

```bash
git clone https://github.com/j620656786206/vido.git
cd vido

# 設定媒體庫路徑與 TMDB API key
cp .env.example .env
#   MEDIA_PATH=/path/to/your/media
#   TMDB_API_KEY=your_key        # 選填，但強烈建議

docker compose up -d
```

開啟 `http://localhost:8080`，依照 setup wizard 完成初始設定。

**主要環境變數**

| 變數                    | 預設      | 說明                            |
| ----------------------- | --------- | ------------------------------- |
| `MEDIA_PATH`            | `./media` | 媒體庫路徑（以唯讀方式掛載）    |
| `TMDB_API_KEY`          | —         | TMDB API key，用於抓取 metadata |
| `TMDB_DEFAULT_LANGUAGE` | `zh-TW`   | metadata 語言偏好               |
| `VIDO_PORT`             | `8080`    | 對外埠號                        |

正式環境可套用資源限制設定：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 技術架構

| 層       | 技術                                             |
| -------- | ------------------------------------------------ |
| 前端     | React 19、TanStack Router/Query、Tailwind CSS v4 |
| 後端     | Go 1.25、Gin、SQLite (WAL + FTS5)                |
| 中文處理 | opencc（簡→繁）                                  |
| 部署     | Docker、單一容器                                 |

## 開發

開發環境設定、建置與測試指令請見 [docs/development.md](docs/development.md)。

## 授權

**授權條款尚未確定**，目前保留所有權利。

原因是相依鏈中有一個 GPL-2.0 的傳遞相依（簡繁轉換用的 `opencc` → `liuzl/da` → `liuzl/cedar-go`），已確認會被靜態連結進 binary。在替換掉該相依之前，宣告任何寬鬆授權都與散布的實際內容不符，因此暫不宣告。

處理完之後會補上正式的 LICENSE。

## 第三方服務與資料來源

- **TMDB** — This product uses the TMDB API but is not endorsed or certified by TMDB.
- **豆瓣** — 繁中 metadata 的補充來源之一，透過解析公開網頁取得（HTML scraping，非官方 API）。網站改版時可能失效，屬 best-effort 的 fallback。
- **Wikipedia** — metadata fallback 來源。
