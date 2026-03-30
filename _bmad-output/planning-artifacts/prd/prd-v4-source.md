# Vido 產品需求規格書 (PRD) v4.0

**最後更新**: 2026-03-22
**版本**: 4.0
**作者**: Alex Yu (游智為)
**狀態**: Draft

---

## 1. 產品願景

### 1.1 一句話定位

**Vido 是專為繁體中文 NAS 用戶打造的 All-in-one 媒體管理介面。**

整合媒體探索、請求管理、字幕自動化、下載任務管理與 NAS 狀態監控，以可插拔架構串接現有工具鏈，取代碎片化的 *arr stack + Seerr 組合。

### 1.2 為什麼需要 Vido

現有 self-hosted 影音管理的兩大痛點：

**痛點一：碎片化工具鏈**

一個完整的影音自動化流程需要 5-7 個獨立 App：Sonarr + Radarr + Bazarr + Seerr/Jellyseerr + Prowlarr + qBittorrent + Plex/Jellyfin。每個都有獨立的 UI、獨立的設定、獨立的更新週期，互相串接靠 API key，出問題時 debug 困難。

**痛點二：繁中用戶被忽略**

所有現有工具都以英文市場為主。繁體中文用戶面臨的具體問題：

- **Bazarr 繁中字幕 bug**：簡繁識別做反（v1.2.1 起已知）、`.zt` 副檔名不被 Plex/Jellyfin 識別、Assrt API response key 用錯（`videoname` 而非 `native_name`）、簡體字幕存在時擋住繁體被抓取
- **Seerr 探索體驗差**：時間排序出現 2031 年未上映影片、過濾維度不足（僅人氣/日期/評分/標題）、趨勢區段完全忽略語言/地區設定（TMDB API 限制）、無亞洲內容策展
- **TMDB 繁中 metadata 不完整**：缺乏豆瓣、Wikipedia 等亞洲來源的 fallback 機制

### 1.3 Vido 的解法

以 **可插拔整合架構** 為核心設計原則：

| 情境 | Vido 的行為 |
|------|------------|
| 沒有 Sonarr/Radarr | Vido 自己掃描媒體庫、管理下載 |
| 有 Sonarr/Radarr | Vido 作為統一 UI，透過 API 串接 |
| 沒有 Bazarr | Vido 內建繁中字幕引擎（修好 Bazarr 的 bug） |
| 沒有 Plex/Jellyfin | Vido 提供媒體庫瀏覽（不含播放） |
| 有 Plex/Jellyfin | Vido 同步觀看紀錄，提供個人化推薦 |

### 1.4 目標用戶

**主要用戶**：自架 NAS 的繁體中文用戶（台灣、香港、海外華人），有影音收藏管理需求，對繁中字幕品質有要求。

**用戶模型**：v4.0 先做單人使用，無需登入。多用戶（家庭成員請求 + 審核機制）列為未來版本。

### 1.5 技術棧

- **後端**：Go（Gin/Echo）
- **前端**：React 19 + TypeScript + Vite
- **資料庫**：SQLite（單用戶，輕量部署）
- **部署**：Docker（單 container）
- **API 設計**：RESTful + SSE（即時進度推送）

---

## 2. 競品分析

> 最後驗證日期：2026-03-22

### 2.1 直接競品

| 產品 | 最新版本 | GitHub Stars | 狀態 | 繁中支援 | 對 Vido 的威脅 |
|------|----------|-------------|------|----------|---------------|
| **Seerr** | v3.1.0 (2026-02-28) | ~10.1K | ⚠️ 功能凍結剛結束，即將恢復新功能開發 | 有語言設定但趨勢區段不生效（TMDB API 限制，團隊明確表示短期無計畫做 client-side filtering） | **中高** — 探索功能可能加速改善 |
| **Bazarr** | v1.5.7-beta.5 (dev) / v1.5.5-beta.1 (stable, 2026-01) | ~11K | 維護中但節奏放緩，近期僅更新依賴和 rate limit 處理 | 繁中 bug 全數未修（簡繁識別做反、`.zt` 不相容、Assrt API key 錯誤） | **低** — 繁中 bug 仍是 Vido 的核心護城河 |
| **MediaManager** | v1.12.3 (2026-03-05) | ~3K | 🔥 最活躍的競品，成長速度快，拿到 DigitalOcean 贊助 | 有多語言搜尋，支援 TVDB + TMDB，但無字幕管理、無豆瓣、無繁中字幕組解析 | **高** — 最接近 Vido「取代 *arr stack」的定位 |
| **Riven** | v0.21.21 (2025-05) | ~731 | ⚠️ 活躍度下降，近兩個月無新 release，open issues 積壓 | 無 | **低** — 走 Debrid 路線，與 Vido 目標用戶重疊度低 |

**Seerr 動態觀察**：Overseerr 和 Jellyseerr 於 2026-02-10 正式合併為 Seerr。v3.0 帶來了 TheTVDB metadata 支援、PostgreSQL 支援、Blocklist 功能。v3.1.0 修補了三個 CVE（含一個高危漏洞）。合併後功能凍結已結束，團隊宣布將恢復新功能開發。這意味著 Seerr 在未來幾個月可能加速改善探索體驗，**Vido Phase 2 的時間壓力比預期更大**。

**MediaManager 動態觀察**：由奧地利學生 Maximilian Dorninger 主導開發（Python + Svelte），半年內從 0 到 3K stars。近期新增多語言搜尋、Prowlarr/Jackett 整合、OAuth/OIDC 認證、torrent 重試/手動匯入。已上架 TrueNAS Apps Market。但仍無字幕管理功能，這是 Vido 的關鍵差異。

### 2.2 間接競品

| 產品 | 最新版本 | Stars | 狀態 | 與 Vido 的重疊 |
|------|----------|-------|------|---------------|
| **Sublarr** | v0.13.2-beta | 早期 | 作者自稱「vibe-coded solo project」，功能多但未打磨。定位為 Bazarr 補充而非替代 | 字幕管理 + LLM 翻譯 + ASS 優先 + AniDB 支援。無繁中字幕組命名解析，依賴 *arr 生態 |
| **Lingarr** | v1.0.3 (2025-12) | 中 | 穩定迭代中，改善了 zh-TW 語言代碼支援，新增匿名使用統計 | 只做字幕翻譯（LibreTranslate/DeepL/AI），不做搜尋/下載。可作為互補工具 |
| **MediaMaster V2** (smysong) | v2.5.17 (2025-12-16) | 219 | ⚠️ 三個月無新 release，可能停更 | 簡中影視訂閱系統，有豆瓣標記同步。只服務簡中用戶，無繁中支援 |
| **Subgen** | 持續更新 (2026-01) | ~782 | 活躍，專注 Whisper 自動轉錄 | 用 Whisper 生成字幕（非搜尋下載），可整合 Bazarr/Plex/Jellyfin。與 Vido P1-021 功能重疊 |

### 2.3 Vido 的差異化護城河

1. **繁中字幕引擎**：修好 Bazarr 所有已知繁中 bug（簡繁識別、副檔名、Assrt API、轉換方向）。截至 2026-03-22，這些 bug 在 Bazarr 中仍未修復，無任何競品涉足此領域
2. **亞洲內容策展**：在 TMDB API 之上建 server-side filtering + 繁中 metadata fallback（豆瓣、Wikipedia）。Seerr 團隊明確表示短期不會做 client-side filtering，此缺口至少持續 3-6 個月
3. **All-in-one 可插拔架構**：單一 Docker container 取代整套 *arr stack，但有外部工具時也能無縫串接。MediaManager 走類似路線但缺字幕管理
4. **NAS Dashboard**：媒體庫統計 + 下載任務管理 + 服務健康狀態，一個介面掌握全局
5. **獨立運作**：不依賴 Sonarr/Radarr，降低新手入門門檻。MediaManager 和 Sublarr 都依賴 *arr 生態

### 2.4 競品風險矩陣

| 風險情境 | 可能性 | 影響 | Vido 的應對 |
|---------|--------|------|------------|
| Seerr 加速改善探索/過濾體驗 | 高 | Phase 2 差異化縮小 | 加速 Phase 2 開發，聚焦亞洲內容策展（Seerr 不太可能做） |
| MediaManager 新增字幕管理 | 中 | Phase 1 護城河被侵蝕 | 繁中字幕組命名解析是技術壁壘，英文開發者難以複製 |
| Bazarr 修復繁中 bug | 低 | Phase 1 核心差異化消失 | 從歷史看這些 bug 存在 2+ 年未修，維護者優先級不在此 |
| 新的 All-in-one 中文競品出現 | 中低 | 全面競爭 | 先發優勢 + 社群口碑 + 開源貢獻者 |

---

## 3. 功能規格

### Phase 1：字幕核心穩定（MVP）

> 目標：成為繁中 NAS 用戶的首選字幕解決方案，不依賴任何外部工具。

#### 3.1 媒體庫掃描器

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P1-001 | 資料夾掃描 | P0 | 指定一或多個媒體庫路徑，自動遞迴掃描影片檔案（mkv, mp4, avi, rmvb） |
| P1-002 | 檔名解析引擎 | P0 | 解析標準命名（`Movie.Name.2024.1080p.BluRay`）和字幕組命名（`[SweetSub][動畫名][12][BIG5][1080P]`） |
| P1-003 | TMDB 自動匹配 | P0 | 根據解析結果自動匹配 TMDB ID，取得 metadata（海報、簡介、評分、演員） |
| P1-004 | 繁中 metadata fallback | P1 | 當 TMDB 繁中資料不完整時，從豆瓣、Wikipedia 取得補充資訊 |
| P1-005 | 定時掃描 | P1 | 可設定掃描頻率（每小時/每日），新增檔案自動入庫 |
| P1-006 | 手動掃描 | P0 | Web UI 上一鍵觸發全量或指定資料夾掃描 |
| P1-007 | 媒體庫瀏覽 | P0 | 以海報牆形式瀏覽已掃描的電影/影集，支援搜尋、排序、篩選 |

#### 3.2 繁中字幕引擎

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P1-010 | 多來源字幕搜尋 | P0 | 支援射手網（Assrt）、字幕庫（Zimuku）、OpenSubtitles 三個來源 |
| P1-011 | Assrt API 修正 | P0 | 使用正確的 response key（`native_name`），修復字幕明明存在卻被 skip 的問題 |
| P1-012 | 簡繁正確識別 | P0 | 字幕下載前透過內容分析（而非檔名）判斷簡繁體，不讓簡體字幕擋住繁體 |
| P1-013 | 副檔名正規化 | P0 | 統一輸出 `.zh-Hant` 或 `.cht` 副檔名，確保 Plex/Jellyfin/Infuse 正確識別 |
| P1-014 | 簡繁轉換 | P0 | OpenCC 轉換，方向正確（簡→繁），並校正兩岸用語差異（軟件→軟體、內存→記憶體） |
| P1-015 | 字幕組命名解析 | P1 | 解析繁中字幕組常見命名格式：`[組名][作品名][集數][語言][解析度]`，正確對應影片 |
| P1-016 | 字幕評分與排序 | P1 | 對搜尋結果評分：語言匹配度 > 解析度匹配 > 來源可信度 > 下載量 |
| P1-017 | 自動下載最佳字幕 | P1 | 根據評分自動選擇並下載最佳繁中字幕，放到正確路徑 |
| P1-018 | 手動搜尋與選擇 | P0 | Web UI 上搜尋字幕、預覽結果、手動選擇下載 |
| P1-019 | 批次字幕處理 | P2 | 對整季影集一次搜尋並下載所有集數的字幕 |

#### 3.3 AI 輔助功能（可選）

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P1-020 | AI 用語校正 | P2 | 使用 Claude API 對簡轉繁結果做兩岸用語細修（需用戶自帶 API Key） |
| P1-021 | MKV 英文軌翻譯 | P3 | 無字幕 fallback：提取 MKV 英文音軌 → Whisper 轉錄 → DeepL/Claude 翻譯成繁中（需用戶自帶 API Key） |

### Phase 2：媒體探索

> 目標：取代 Seerr 的探索功能，提供更好的亞洲內容瀏覽體驗。

#### 3.4 首頁電視牆

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P2-001 | Hero Banner 輪播 | P0 | 首頁頂部大圖輪播，展示精選/熱門內容，支援自動播放預告片（如有） |
| P2-002 | 可自訂探索區塊 | P0 | 用戶可新增、移除、排序首頁的內容區塊（如「近期台灣院線」「熱門日韓劇」「Netflix 台灣新上架」） |
| P2-003 | 智慧趨勢區段 | P0 | 趨勢內容 **強制套用** 語言/地區過濾（server-side filtering，不依賴 TMDB API 的 endpoint 限制） |
| P2-004 | 隱藏遠期內容 | P0 | 自動過濾掉上映日期超過 6 個月的佔位內容（如 Avatar 5 2031） |
| P2-005 | 隱藏低品質內容 | P1 | 自動過濾 TMDB 評分 < 3 且投票數 < 50 的內容 |
| P2-006 | 已有/已請求標記 | P1 | 媒體庫中已有的內容顯示「可觀看」badge，已請求的顯示「已請求」 |

#### 3.5 進階搜尋與過濾

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P2-010 | 多維過濾器 | P0 | 支援同時篩選：類型（動作/劇情/動畫...）、年份範圍、地區/語言、評分範圍、串流平台 |
| P2-011 | 過濾器常駐 UI | P0 | 過濾器不藏在 dropdown 裡，以 pill/chip 形式常駐在頁面頂部，一目了然 |
| P2-012 | 複合排序 | P0 | 排序選項：人氣、發行日期、評分、新增日期。排序結果受過濾器影響 |
| P2-013 | 即時搜尋 | P0 | 搜尋框支援即時建議（debounce），顯示電影、影集、人物三類結果 |
| P2-014 | 繁中搜尋優先 | P1 | 搜尋時同時查 TMDB 中文標題和原文標題，繁中結果優先排序 |
| P2-015 | 儲存篩選條件 | P2 | 用戶可儲存常用篩選組合（如「2024 年以後的韓劇」），快速取用 |

#### 3.6 媒體詳情頁

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P2-020 | 豐富資訊展示 | P0 | 海報、背景圖、繁中簡介、演員/導演、類型、片長、上映日期、評分（TMDB + 豆瓣雙評分） |
| P2-021 | 影集季/集列表 | P0 | 影集頁面展開季別、各集標題與簡介，標記已有/缺少字幕狀態 |
| P2-022 | 相關推薦 | P1 | 根據當前內容推薦相似作品，套用地區/語言過濾 |
| P2-023 | 串流平台資訊 | P1 | 顯示該內容在台灣可用的串流平台（Netflix/Disney+/KKTV/...），資料來自 TMDB Watch Providers |
| P2-024 | 預告片播放 | P2 | 嵌入 YouTube 預告片（如有） |
| P2-025 | 豆瓣連結 | P2 | 提供豆瓣頁面直連，方便查看中文評論 |

### Phase 3：請求流程 + 下載管理

> 目標：一鍵請求 → 自動下載 → 自動抓字幕，全流程自動化。

#### 3.7 請求系統

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P3-001 | 一鍵請求 | P0 | 在探索頁面或詳情頁直接請求電影/影集 |
| P3-002 | 影集部分請求 | P0 | 可選擇請求特定季或特定集數 |
| P3-003 | 請求狀態追蹤 | P0 | 請求列表頁，顯示每個請求的狀態：待處理/搜尋中/下載中/已完成/失敗 |
| P3-004 | Sonarr/Radarr 串接（可選） | P0 | 有設定 Sonarr/Radarr 時，請求透過其 API 發送；未設定時走 Vido 內建流程 |
| P3-005 | 自動字幕觸發 | P1 | 下載完成後自動觸發字幕搜尋（Phase 1 的字幕引擎） |

#### 3.8 下載任務管理

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P3-010 | qBittorrent 串接 | P0 | 透過 qBittorrent Web API 管理下載任務（新增/暫停/刪除/查看進度） |
| P3-011 | NZBGet 串接（可選） | P2 | 同上，支援 Usenet 下載 |
| P3-012 | 下載進度即時更新 | P0 | SSE 推送下載進度到前端，不需手動刷新 |
| P3-013 | 下載完成通知 | P1 | 下載完成時透過 Web UI 通知（未來可擴展 Telegram/Discord） |
| P3-014 | 內建 BT 引擎（未來） | P3 | 使用 Go BT library（anacrolix/torrent）內建下載功能，消除 qBittorrent 依賴 |

#### 3.8.1 下載任務管理 — 技術需求補充 (2026-03-30)

| ID | 需求 | 優先級 | 說明 |
|----|------|--------|------|
| P3-010a | qBittorrent 4.x/5.0+ 相容性 | P0 | 支援 qBT 4.x (pausedDL/pausedUP) 和 5.0+ (stoppedDL/stoppedUP) 雙版本 state mapping，遵循 Sonarr/Radarr 業界標準 |
| P3-010b | 下載列表分頁 | P0 | 後端分頁 API (page + pageSize)，預設每頁 100 筆，可選 50/100/200/500，避免一次回傳所有種子造成效能問題 |
| P3-010c | 下載頁設計稿 | P0 | UX 設計稿 Flow G 已補齊：G1 下載列表 Desktop、G2 展開詳情 Desktop、G3 列表 Mobile、G4 空狀態 Mobile |

**參考資料：**
- qBittorrent 5.0 API 文件：https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-5.0)
- State mapping 遵循 Sonarr 標準：stoppedUP/pausedUP/stalledUP → completed, stoppedDL/pausedDL → paused

#### 3.9 Indexer 管理

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P3-020 | Prowlarr 串接（可選） | P1 | 有設定 Prowlarr 時，透過其 API 搜尋 indexer |
| P3-021 | 內建 indexer 搜尋 | P2 | 未設定 Prowlarr 時，Vido 內建基礎的 torrent 搜尋能力（公開 tracker） |

### Phase 4：NAS Dashboard

> 目標：一個介面掌握 NAS 影音系統的全局狀態。

#### 3.10 媒體庫統計

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P4-001 | 媒體庫總覽 | P0 | 電影/影集總數、磁碟用量、平均檔案大小、解析度分佈（4K/1080p/720p） |
| P4-002 | 字幕覆蓋率 | P0 | 有繁中字幕 / 有其他字幕 / 無字幕的比例圖 |
| P4-003 | 類型分佈 | P1 | 收藏的類型/地區/年份分佈圖表 |
| P4-004 | 最近新增 | P0 | 最近 7 天/30 天新增的媒體列表 |

#### 3.11 Plex/Jellyfin 串接

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P4-010 | 觀看紀錄同步 | P1 | 從 Plex/Jellyfin 同步觀看紀錄，用於個人化推薦 |
| P4-011 | 正在觀看 | P1 | 首頁顯示「繼續觀看」區塊（基於 Plex/Jellyfin 的觀看進度） |
| P4-012 | 庫存同步 | P0 | 定期掃描 Plex/Jellyfin 庫存，標記已有內容 |

#### 3.12 服務健康監控

| ID | 功能 | 優先級 | 說明 |
|----|------|--------|------|
| P4-020 | 外部服務狀態 | P1 | 顯示已串接服務的連線狀態（Sonarr/Radarr/qBittorrent/Plex/Jellyfin） |
| P4-021 | 磁碟空間警告 | P1 | 媒體庫磁碟用量超過閾值時顯示警告 |
| P4-022 | 活動日誌 | P2 | 記錄所有自動化動作（字幕下載、請求處理、掃描結果）的日誌，可搜尋可過濾 |

---

## 4. 不做清單 (Out of Scope)

以下功能 **明確排除** 在 v4.0 範圍之外：

| 項目 | 原因 |
|------|------|
| 影片播放功能 | Plex/Jellyfin/Infuse 已經做得很好，不重複造輪子 |
| 多用戶 + 權限管理 | v4.0 先做單人，多用戶是 v5.0 的考量 |
| 推薦引擎（ML-based） | 用 TMDB 的推薦 API + 觀看紀錄做簡單推薦，不自己訓練模型 |
| 字幕組社群功能 | 不做字幕上傳、分享、評論 |
| Whisper 本地模型 | AI 功能依賴外部 API（用戶自帶 key），不內建 Whisper 推理 |
| Docker/容器管理 | 不做 Portainer 或 Unraid Docker Manager 的替代品 |
| 音樂管理 | 只做影片（電影/影集/動畫），不做音樂 |
| 直播/IPTV | 不在範圍內 |

---

## 5. 技術架構

### 5.1 系統架構圖

```
┌─────────────────────────────────────────────────────┐
│                    Vido (Docker)                     │
│                                                     │
│  ┌──────────┐  ┌──────────────────────────────────┐ │
│  │ React 19 │  │         Go Backend               │ │
│  │ Vite     │◄►│  ┌─────────┐ ┌────────────────┐  │ │
│  │ TypeScript│  │  │ REST API│ │ Plugin Manager │  │ │
│  └──────────┘  │  └─────────┘ └────────────────┘  │ │
│                │  ┌─────────┐ ┌────────────────┐  │ │
│                │  │ Scanner │ │ Subtitle Engine│  │ │
│                │  └─────────┘ └────────────────┘  │ │
│                │  ┌─────────┐ ┌────────────────┐  │ │
│                │  │ SSE Hub │ │   SQLite DB    │  │ │
│                │  └─────────┘ └────────────────┘  │ │
│                └──────────────────────────────────┘ │
└────────────┬───────────┬──────────┬────────────────┘
             │           │          │
     ┌───────▼───┐ ┌─────▼────┐ ┌──▼──────────┐
     │ TMDB API  │ │ Assrt    │ │ 外部服務     │
     │ 豆瓣 API  │ │ Zimuku   │ │ (可選串接)   │
     │ Wikipedia │ │ OpenSub  │ │              │
     └───────────┘ └──────────┘ │ Sonarr       │
                                │ Radarr       │
                                │ qBittorrent  │
                                │ Prowlarr     │
                                │ Plex         │
                                │ Jellyfin     │
                                └──────────────┘
```

### 5.2 可插拔整合層

每個外部服務都是一個 **Plugin**，實作統一的 Go interface：

```go
// 以下為概念設計，非最終實作

type MediaServerPlugin interface {
    Name() string
    TestConnection(config PluginConfig) error
    SyncLibrary() ([]MediaItem, error)
    GetWatchHistory(userID string) ([]WatchRecord, error)
}

type DownloaderPlugin interface {
    Name() string
    TestConnection(config PluginConfig) error
    AddDownload(request DownloadRequest) (string, error)
    GetStatus(id string) (DownloadStatus, error)
    Pause(id string) error
    Remove(id string) error
}

type DVRPlugin interface {
    Name() string
    TestConnection(config PluginConfig) error
    AddMovie(tmdbID int, qualityProfile string) error
    AddSeries(tmdbID int, qualityProfile string, seasons []int) error
    GetQueue() ([]QueueItem, error)
}
```

### 5.3 Server-side 過濾策略

解決 Seerr 的 TMDB API 限制問題：

1. **Trending endpoint** 不支援語言過濾 → Vido 在 Go 後端對 TMDB 回傳結果做 post-filtering
2. **遠期未上映內容** → 過濾 `release_date > now + 6 months` 且 `vote_count < 10` 的條目
3. **低品質佔位內容** → 過濾 `vote_average < 3` 且 `vote_count < 50` 的條目
4. **地區相關性** → 優先顯示 `original_language` 為 `zh`/`ja`/`ko`/`en` 的內容
5. **結果快取** → Redis 或記憶體快取（TTL 1 小時），避免重複呼叫 TMDB API

### 5.4 字幕引擎架構

```
輸入：MediaItem (影片檔案路徑 + TMDB metadata)
  │
  ├─ Step 1: 多來源搜尋
  │   ├─ Assrt（修正 API key）
  │   ├─ Zimuku（爬蟲，需注意維護成本）
  │   └─ OpenSubtitles（官方 API）
  │
  ├─ Step 2: 字幕評分
  │   ├─ 語言判斷（內容分析，非檔名）
  │   ├─ 解析度匹配
  │   ├─ Release Group 匹配
  │   └─ 來源信任度
  │
  ├─ Step 3: 下載最佳字幕
  │
  ├─ Step 4: 後處理
  │   ├─ 簡繁轉換（OpenCC，方向正確）
  │   ├─ 兩岸用語校正
  │   └─ 副檔名正規化（→ .zh-Hant.srt）
  │
  └─ Step 5: 放置到影片同目錄
```

---

## 6. UI/UX 設計原則

### 6.1 設計語言

- **深色主題為主**：NAS 用戶通常在暗光環境操作
- **Mobile-first 響應式**：手機上也能舒適操作（審核請求、瀏覽媒體庫）
- **海報牆優先**：視覺化呈現 > 文字列表
- **中文排版優化**：使用繁中友善字體（Noto Sans TC）、正確的行高與字距

### 6.2 Seerr 的 UI 問題 & Vido 的改進

| Seerr 問題 | Vido 改進 |
|------------|----------|
| 首頁只有水平滾動卡片列 | Hero Banner + 可自訂區塊 + 垂直無限捲動 |
| 過濾器藏在 dropdown 裡 | 過濾器以 chip 常駐頁面頂部 |
| 排序只有 4 個選項 | 多維過濾 + 複合排序 |
| 趨勢無視語言設定 | Server-side 強制過濾 |
| 電影/影集頁面資訊不足 | 雙評分（TMDB + 豆瓣）+ 串流平台 + 字幕狀態 |
| 搜尋只有基礎文字搜尋 | 即時建議 + 繁中優先 + 人物搜尋 |

### 6.3 關鍵頁面

1. **首頁**：Hero Banner → 繼續觀看 → 自訂探索區塊 → 最近新增 → 字幕待處理
2. **探索頁（電影/影集）**：常駐過濾器 → 海報牆網格 → 無限捲動
3. **媒體詳情頁**：大背景圖 → metadata → 字幕狀態 → 請求按鈕 → 相關推薦
4. **字幕管理頁**：媒體庫字幕覆蓋率 → 缺少字幕列表 → 批次處理
5. **下載中心**：活躍下載 → 佇列 → 歷史紀錄
6. **Dashboard**：媒體庫統計 → 磁碟空間 → 服務狀態 → 活動日誌
7. **設定頁**：外部服務連線 → 字幕偏好 → 媒體庫路徑 → 通知設定

---

## 7. 開發路線圖

### Phase 1：字幕核心穩定（MVP）

**預計時間**：8-10 週
**目標**：可獨立運作的繁中字幕管理工具

| 里程碑 | 內容 | 週數 |
|--------|------|------|
| M1.1 | Go 後端骨架 + SQLite + 基礎 API | 1-2 |
| M1.2 | 媒體庫掃描器 + 檔名解析引擎 | 2-3 |
| M1.3 | TMDB 匹配 + 繁中 metadata | 1-2 |
| M1.4 | 字幕引擎（Assrt 修正 + 簡繁識別 + 轉換） | 2-3 |
| M1.5 | React 19 前端 + 媒體庫瀏覽 + 字幕管理 UI | 2-3 |
| M1.6 | Docker 打包 + 首次公開發布 | 1 |

**Phase 1 驗收指標**：

| 指標 | 目標 |
|------|------|
| 標準命名檔案解析成功率 | > 99% |
| 字幕組命名檔案解析成功率 | > 95% |
| 繁中字幕搜尋命中率 | > 85% |
| 不誤抓簡體字幕 | 100% |
| 兩岸用語校正正確率 | > 95% |
| Docker 啟動到可用時間 | < 10 秒 |

### Phase 2：媒體探索

**預計時間**：6-8 週
**前置條件**：Phase 1 已穩定

| 里程碑 | 內容 | 週數 |
|--------|------|------|
| M2.1 | TMDB API 整合 + server-side 過濾層 | 2 |
| M2.2 | 首頁電視牆（Hero Banner + 自訂區塊） | 2-3 |
| M2.3 | 進階搜尋與多維過濾器 | 2-3 |
| M2.4 | 媒體詳情頁（雙評分 + 串流平台 + 字幕狀態） | 1-2 |

### Phase 3：請求流程 + 下載管理

**預計時間**：6-8 週
**前置條件**：Phase 2 已穩定

| 里程碑 | 內容 | 週數 |
|--------|------|------|
| M3.1 | 請求系統（一鍵請求 + 狀態追蹤） | 2 |
| M3.2 | 可插拔整合層（Sonarr/Radarr Plugin） | 2-3 |
| M3.3 | qBittorrent 串接 + 下載進度 SSE | 2 |
| M3.4 | 全流程自動化（請求→下載→字幕） | 1-2 |

### Phase 4：NAS Dashboard

**預計時間**：4-6 週
**前置條件**：Phase 3 已穩定

| 里程碑 | 內容 | 週數 |
|--------|------|------|
| M4.1 | 媒體庫統計 + 字幕覆蓋率圖表 | 2 |
| M4.2 | Plex/Jellyfin 串接 + 觀看紀錄同步 | 2-3 |
| M4.3 | 服務健康監控 + 磁碟空間警告 | 1-2 |

---

## 8. 技術風險

| 風險 | 影響 | 緩解策略 |
|------|------|----------|
| **Zimuku/豆瓣爬蟲維護成本** | 網站改版會導致爬蟲失效 | 以 TMDB + Assrt（正式 API）為主，爬蟲為 fallback；定期監控爬蟲健康度 |
| **Assrt API 穩定性** | 射手網服務不穩定，可能下線 | 多來源冗餘設計；Assrt 不可用時 fallback 到 OpenSubtitles |
| **TMDB API rate limit** | 免費 tier 有請求限制 | Server-side 快取（TTL 1 小時）+ 請求 debounce + 批次查詢 |
| **繁中字幕組命名格式多變** | 新字幕組可能使用未知格式 | 正則引擎可配置化 + 社群貢獻格式定義 |
| **Go BT library 成熟度** | anacrolix/torrent 可能不如 qBittorrent 穩定 | Phase 3 先串接 qBittorrent，內建 BT 引擎列為 Phase 3 的 P3 優先級 |
| **單人開發頻寬** | 4 個 Phase 全做完可能超過 6 個月 | 嚴格按 Phase 順序，每個 Phase 獨立可用可發布；Phase 1 MVP 為最優先 |

---

## 9. 成功指標

### 9.1 Phase 1 發布後（3 個月內）

| 指標 | 目標 |
|------|------|
| GitHub Stars | > 100 |
| Docker Hub pulls | > 500 |
| 社群回報的繁中字幕 bug | < 5 個 |
| 個人 dogfooding：日常使用取代 Bazarr | 是 |

### 9.2 Phase 2 發布後（6 個月內）

| 指標 | 目標 |
|------|------|
| GitHub Stars | > 500 |
| Docker Hub pulls | > 2,000 |
| PTT/巴哈 NAS 相關板有自然討論 | 是 |
| 個人 dogfooding：日常使用取代 Seerr | 是 |

### 9.3 Phase 4 完成後（12 個月內）

| 指標 | 目標 |
|------|------|
| GitHub Stars | > 1,000 |
| Docker Hub pulls | > 5,000 |
| 外部貢獻者 | > 5 人 |
| 繁中 NAS 社群提到 Vido 作為推薦工具 | 是 |

---

## 10. 部署規格

```yaml
# docker-compose.yml 最小配置
services:
  vido:
    image: vido:latest
    ports:
      - "8888:8888"
    volumes:
      - /path/to/media:/media:ro       # 媒體庫（唯讀掛載）
      - /path/to/config:/config         # 設定與 SQLite DB
    environment:
      - TZ=Asia/Taipei
      - TMDB_API_KEY=your_key           # 必要
      - CLAUDE_API_KEY=                 # 可選，AI 功能
      - DEEPL_API_KEY=                  # 可選，字幕翻譯
```

**進階配置（有外部服務時）**：

```yaml
    environment:
      # ... 上述基礎設定
      - SONARR_URL=http://sonarr:8989
      - SONARR_API_KEY=your_key
      - RADARR_URL=http://radarr:7878
      - RADARR_API_KEY=your_key
      - QBITTORRENT_URL=http://qbittorrent:8080
      - QBITTORRENT_USERNAME=admin
      - QBITTORRENT_PASSWORD=your_password
      - PLEX_URL=http://plex:32400
      - PLEX_TOKEN=your_token
      - JELLYFIN_URL=http://jellyfin:8096
      - JELLYFIN_API_KEY=your_key
      - PROWLARR_URL=http://prowlarr:9696
      - PROWLARR_API_KEY=your_key
```

**Web UI**：`http://your-nas-ip:8888`
**單人使用**：無需登入，直接使用

---

## 11. 與 v3.0 PRD 的差異摘要

| 項目 | v3.0 | v4.0 |
|------|------|------|
| **定位** | 繁中字幕管理工具 | All-in-one 媒體管理介面 |
| **範圍** | 字幕搜尋/下載/轉換 | 字幕 + 媒體探索 + 請求 + 下載 + Dashboard |
| **Seerr 替代** | 不在範圍內 | Phase 2 的核心目標 |
| **NAS Dashboard** | 不在範圍內 | Phase 4 |
| **外部工具依賴** | 獨立運作 | 可插拔架構：可獨立，也可串接 |
| **Phase 1（MVP）** | 與 v3.0 一致 | 與 v3.0 一致（字幕核心穩定） |
| **開發策略** | 一次到位 | 四階段漸進發布，每個 Phase 獨立可用 |

**關鍵決策**：Phase 1（MVP）的內容與 v3.0 完全一致，不增加範圍。新增的探索、請求、Dashboard 功能都在 Phase 2-4，確保 MVP 能最快發布。

---

*本文件為活文件，隨開發進展持續更新。*
*最後更新：2026-03-22*
