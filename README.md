# Vido

**自架的家庭劇院媒體管理工具，把繁體中文字幕這件事做到底。**
從影片裡抽字幕、線上找字幕、到用語音辨識生字幕——最後都變成繁中。

> **English**: Vido is a self-hosted media manager for your NAS home theater, built around one problem: getting Traditional Chinese subtitles for content that doesn't have them. It extracts embedded subtitles, searches online sources, and falls back to speech recognition — then translates and converts everything to zh-TW.

---

## 為什麼做這個

想看的劇散在 Netflix、Disney+ 各家平台，每看一部就要切換一次。所以我架了 NAS，把影集和電影收在自己家裡，當成家庭劇院用。

但做完之後才發現真正的問題：**網路上的資源很多都沒有中文字幕**。片子有了，體驗卻缺一塊。

所以這個工具最初想解決兩件事：

1. 影片裡**已經有字幕**（通常是英文）→ 翻成繁中
2. 影片裡**沒有字幕** → 上網找現成的

開發之後又撞到第二個問題：**線上字幕資源比想像中少很多**。射手網、OpenSubtitles 對冷門片或新片經常是空的。於是策略變成一條有優先序的路徑：

```
影片內嵌字幕  →  線上字幕來源  →  ASR 語音辨識
（有就直接翻）   （射手網 / OpenSubtitles）  （前兩者都沒有時，聽聲音生字幕再翻）
```

字幕解決了，接著才是把媒體庫整理好的那些事——繁中 metadata、字幕組檔名解析、下載監控，那些是為了讓這個家庭劇院真的能用而長出來的。

## 目前能做什麼

**字幕（核心）**

- **線上字幕搜尋與下載** — 串接射手網（ASSRT）與 OpenSubtitles，自動評分挑最合適的版本
- **簡轉繁** — opencc 轉換，並可批次處理整個媒體庫
- **AI 字幕增強** — 用詞表（glossary）維持同一部劇的譯名一致
- **語音辨識生字幕** — 沒有現成字幕時，抽出音軌用 Whisper 轉錄
- **字幕軌偵測** — 掃描影片內嵌與外掛字幕，標示語言與狀態

**媒體庫**

- **繁中優先的 metadata** — TMDB (zh-TW) → 豆瓣 → Wikipedia 多來源 fallback
- **AI 字幕組檔名解析** — 解析 `[字幕組][作品名][01][1080p][BIG5]` 這種命名，內建學習機制與自動重試
- **下載監控** — 串接 qBittorrent，即時進度
- **媒體庫管理** — 遞迴掃描、多磁碟區支援
- **瀏覽與探索** — 首頁媒體牆、進階搜尋、詳情頁（雙評分、季集列表、推薦、觀看平台、預告）
- **活動中心** — 集中檢視解析／下載／字幕處理的進行狀況
- **設定引導** — 首次啟動的 setup wizard

**開發中**

- **抽取內嵌字幕來翻譯** — 目前只做到偵測有哪些字幕軌，還沒把內容抽出來翻。這是字幕 pipeline 改版的主軸（優先序：內嵌 → 線上 → ASR）
- **Request 系統** — 一鍵「想要」＋ Radarr/Sonarr 串接

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

| 層   | 技術                                             |
| ---- | ------------------------------------------------ |
| 前端 | React 19、TanStack Router/Query、Tailwind CSS v4 |
| 後端 | Go 1.25、Gin、SQLite (WAL + FTS5)                |
| 字幕 | opencc（簡→繁）、Whisper（語音辨識）、ffmpeg     |
| 部署 | Docker、單一容器                                 |

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
