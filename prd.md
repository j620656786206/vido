# 影音管家 (Vido) 技術需求規格書 v2.0

**核心目標：** 高韌性、在地化、自動化影音資產管理

## 1. 韌性架構與部署 (Resilient Architecture)

* **Monorepo:** Nx (管理 `apps/web`, `apps/api`, `libs/metadata-engine`)。
* **Backend:** Go (Gin) + SQLite (WAL)。
* **數據抽象層 (Critical):** 實作 `MetadataProvider` 介面，支援多來源動態切換。
* **AI 備援路徑:** 當 API 失效時，自動觸發 **Gemini/Claude API** 進行檔名語義解析。
* **部署:** 支援 Docker 與單一二進位檔 (Embedded UI assets)。

## 2. 核心技術棧

* **Frontend:** React + TanStack Router/Query + Tailwind CSS。
* **API Client:** 從 Go OpenAPI 自動生成 TypeScript SDK。
* **通信:** SSE (Server-Sent Events) 用於下載進度與 AI 處理狀態的實時推送。

---

## 3. 詳細功能需求 (Functional Requirements)

### 3.1 高韌性數據引擎 (The Metadata Shield)

* **多源適配器 (Multi-Source Adapters):**
* **Primary:** TMDb API (zh-TW)。
* **Secondary:** 豆瓣 Scraper (用於彌補陸港台片名差異)。
* **Tertiary:** Trakt.tv / TheTVDB。


* **AI Fallback Parser:** 若所有 API 失敗，將檔名傳送至 LLM 解析（例如：`[Nekomoe] Oshi no Ko - 01 (1080p).mkv` -> 自動識別為《我推的孩子》第一季第一集）。
* **本地 NFO 優先:** 若目錄已存在 `.nfo` 或 `.xml`，優先讀取本地元數據。

### 3.2 智慧搜尋與下載管理

* **在地化優化:** 強制鎖定台灣用語，自動將「太空梭」修正為「太空船」（依此類推），確保 UI 呈現符合台灣習慣。
* **一鍵請求:** 整合 Jellyseerr/Radarr/Sonarr API，支援「訂閱制」追劇。
* **即時監控:** 透過 WebSocket/SSE 對接 qBittorrent，顯示下載速度、健康度與預計剩餘時間。

### 3.3 字幕自動化管家 (Subtitle Automation)

* **多來源抓取:** 整合射手網、字幕庫 (Zimuku) 與 OpenSubtitles。
* **AI 繁體化與校對:** * 自動將簡體字幕轉為**正體中文**。
* 利用 AI 修正字幕中的兩岸用語差異（例如：軟件 -> 軟體）。


* **編碼自動修正:** 自動偵測 Big5/GBK 並統一轉碼為 UTF-8。

---

## 4. 詳細任務拆解 (Atomic Tasks for Auto-Claude)

### 階段 1：基礎架構與介面定義 (Scaffolding & Interface)

* **Task 1.1:** 初始化 Nx + Go (Gin) + Vite React 環境。
* **Task 1.2:** **[關鍵]** 定義 `MetadataProvider` Interface 與 `Movie` / `Series` 核心數據結構。
* **Task 1.3:** 實作 Go 端 `//go:embed` 與 SQLite 初始化邏輯。

### 階段 2：數據適配器與 AI Fallback (Resilience Logic)

* **Task 2.1:** 實作 `TMDbAdapter` 並加入 Rate Limit 熔斷機制。
* **Task 2.2:** 實作 `AIParserAdapter`：整合 Gemini/Claude API，解析複雜檔名並回傳 JSON 格式數據。
* **Task 2.3:** 實作前端搜尋頁面：支援來源切換與手動修正元數據功能。

### 階段 3：下載監控與動態推送 (Real-time Flow)

* **Task 3.1:** 實作 qBittorrent API 客戶端。
* **Task 3.2:** 建立 Go SSE 端點，定時廣播當前下載任務的進度數據。
* **Task 3.3:** 前端實作下載 Dashboard，使用 TanStack Query 進行數據同步。

### 階段 4：自動化字處理 (Subtitle Logic)

* **Task 4.1:** 實作字幕爬蟲與 API 對接邏輯。
* **Task 4.2:** 實作 OpenCC 繁簡轉換服務，並加入 AI 精準校對模組。
* **Task 4.3:** 實作自動命名與檔案搬移邏輯 (File Auto-organizer)。

---

## 5. 創業驗證指標 (MVP Validation)

1. **數據成功率:** 隨機輸入 100 個冷門或非正規命名的影片檔名，AI 解析成功率需 > 95%。
2. **搜尋摩擦力:** 從搜尋到加入下載清單，操作流程需在 3 次點擊內完成。
3. **在地化程度:** 字幕繁體化後，台灣特有詞彙（如「電訊」->「電信」）正確率需達標。

---
