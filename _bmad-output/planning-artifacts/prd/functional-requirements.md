# Functional Requirements (v4)

## Traceability Header

This document replaces the previous FR1–FR94 numbering scheme with the v4 feature ID scheme: **P{phase}-{sequence}**. The v4 PRD reorganizes all functionality into four delivery phases aligned with the product roadmap. Requirements previously marked as "Growth" or "Post-1.0" have been either promoted into a specific phase, merged into broader features, or explicitly deleted.

Source of truth: `prd-v4-source.md` Section 3 — 功能規格.

---

## Legacy FR Mapping Table

| Old FR IDs | Topic | New v4 ID(s) | Status |
|------------|-------|--------------|--------|
| FR1–FR4 | Media search & discovery (MVP) | P2-013, P2-014 | Partially DONE (Epic 2 search) |
| FR5–FR8 | Library search, sort, filter, views | P1-007 | DONE (Epic 5) |
| FR9–FR10 | Smart recommendations, similar titles | P2-022 | NEW |
| FR11–FR14 | Filename parsing, metadata retrieval | P1-002, P1-003 | DONE (Epic 2–3) |
| FR15–FR26 | AI parsing, multi-source fallback, manual edit | P1-002, P1-004 | DONE (Epic 3) |
| FR27–FR33 | qBittorrent monitoring | P3-010 | DONE (Epic 4) |
| FR34–FR37 | Advanced download control | P3-010–P3-014 | NEW |
| FR38–FR42 | Media library management | P1-007 | DONE (Epic 5) |
| FR43–FR46 | Watch history, collections | P4-011 | NEW |
| FR47–FR51 | Deployment, Docker, config | Infrastructure | DONE (Epic 1) |
| FR52–FR66 | Settings, backup, export, metrics | Settings | Partially DONE (Epic 6) |
| FR67–FR70 | Authentication & access control | — | DELETED (single user, no login in v4) |
| FR71–FR74 | Multi-user accounts & permissions | — | DELETED (deferred to v5.0) |
| FR75–FR80 | Subtitle search & download | P1-010–P1-019 | NEW (expanded massively) |
| FR81–FR86 | Automation & organization | P1-001, P1-005 (partial) | Partial; rest DELETED |
| FR87–FR94 | External API, mobile app | — | DELETED (out of scope) |

---

## Phase 1: 字幕核心穩定 (MVP)

> Goal: Become the go-to subtitle solution for Traditional Chinese NAS users, with zero dependency on external tools.

### 3.1 媒體庫掃描器

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P1-001 | 資料夾掃描 | P0 | Specify one or more media library paths; recursively scan for video files (mkv, mp4, avi, rmvb). | DONE (Epic 5) |
| P1-002 | 檔名解析引擎 | P0 | Parse standard naming (`Movie.Name.2024.1080p.BluRay`) and fansub naming (`[SweetSub][動畫名][12][BIG5][1080P]`). | DONE (Epic 2–3) |
| P1-003 | TMDB 自動匹配 | P0 | Auto-match parsed results to TMDB IDs; fetch metadata (poster, synopsis, rating, cast). | DONE (Epic 2) |
| P1-004 | 繁中 metadata fallback | P1 | When TMDB Traditional Chinese data is incomplete, fetch supplementary info from Douban and Wikipedia. | DONE (Epic 3) |
| P1-005 | 定時掃描 | P1 | Configurable scan frequency (hourly/daily); new files are automatically ingested. | NEW |
| P1-006 | 手動掃描 | P0 | One-click trigger for full or folder-specific scan from Web UI. | DONE (Epic 5) |
| P1-007 | 媒體庫瀏覽 | P0 | Browse scanned movies/series in poster-wall view with search, sort, and filter. | DONE (Epic 5) |

### 3.2 繁中字幕引擎

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P1-010 | 多來源字幕搜尋 | P0 | Search subtitles from Assrt (射手網), Zimuku (字幕庫), and OpenSubtitles. | NEW |
| P1-011 | Assrt API 修正 | P0 | Use the correct response key (`native_name`); fix the bug where existing subtitles are skipped. | NEW |
| P1-012 | 簡繁正確識別 | P0 | Detect Simplified vs Traditional via content analysis (not filename) before download; prevent Simplified subtitles from blocking Traditional ones. | NEW |
| P1-013 | 副檔名正規化 | P0 | Normalize output to `.zh-Hant` or `.cht` extension so Plex/Jellyfin/Infuse correctly identify the language. | NEW |
| P1-014 | 簡繁轉換 | P0 | OpenCC conversion in the correct direction (Simplified → Traditional) with cross-strait terminology correction (軟件→軟體, 內存→記憶體). | NEW |
| P1-015 | 字幕組命名解析 | P1 | Parse common Traditional Chinese fansub naming patterns: `[Group][Title][Episode][Language][Resolution]` and match to the correct video file. | NEW |
| P1-016 | 字幕評分與排序 | P1 | Score search results: language match > resolution match > source trustworthiness > download count. | NEW |
| P1-017 | 自動下載最佳字幕 | P1 | Automatically select and download the best Traditional Chinese subtitle based on score; place it at the correct path. | NEW |
| P1-018 | 手動搜尋與選擇 | P0 | Search subtitles, preview results, and manually select a download from Web UI. | NEW |
| P1-019 | 批次字幕處理 | P2 | Batch search and download subtitles for all episodes of a full season at once. | NEW |

### 3.3 AI 輔助功能

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P1-020 | AI 用語校正 | P2 | Use Claude API to fine-tune cross-strait terminology in Simplified→Traditional conversion results (requires user-provided API key). | NEW |
| P1-021 | MKV 英文軌翻譯 | P3 | No-subtitle fallback: extract English audio track from MKV → Whisper transcription → DeepL/Claude translation to Traditional Chinese (requires user-provided API keys). | NEW |

---

## Phase 2: 媒體探索

> Goal: Replace Seerr's discovery experience with better Asian content browsing.

### 3.4 首頁電視牆

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P2-001 | Hero Banner 輪播 | P0 | Large hero carousel at top of home page showcasing featured/trending content; supports auto-playing trailers when available. | NEW |
| P2-002 | 可自訂探索區塊 | P0 | Users can add, remove, and reorder content sections on the home page (e.g., "Recent Taiwan Theatrical," "Trending J/K-Drama," "Netflix Taiwan New Arrivals"). | NEW |
| P2-003 | 智慧趨勢區段 | P0 | Trending content **enforces** language/region filtering via server-side filtering, bypassing TMDB API endpoint limitations. | NEW |
| P2-004 | 隱藏遠期內容 | P0 | Auto-filter content with release dates more than 6 months in the future (e.g., Avatar 5 2031). | NEW |
| P2-005 | 隱藏低品質內容 | P1 | Auto-filter content with TMDB rating < 3 and vote count < 50. | NEW |
| P2-006 | 已有/已請求標記 | P1 | Content already in the media library shows a "Available" badge; content already requested shows a "Requested" badge. | NEW |

### 3.5 進階搜尋與過濾

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P2-010 | 多維過濾器 | P0 | Multi-facet filtering: genre, year range, region/language, rating range, streaming platform — all simultaneously. | NEW |
| P2-011 | 過濾器常駐 UI | P0 | Filters displayed as persistent pill/chip UI at page top (not hidden in dropdowns). | NEW |
| P2-012 | 複合排序 | P0 | Sort by popularity, release date, rating, or date added; sort results respect active filters. | NEW |
| P2-013 | 即時搜尋 | P0 | Search bar with instant suggestions (debounced); shows movies, series, and people as separate result categories. | Partially DONE (Epic 2 search) |
| P2-014 | 繁中搜尋優先 | P1 | Search queries simultaneously match TMDB Chinese titles and original titles; Traditional Chinese results ranked first. | Partially DONE (Epic 2 search) |
| P2-015 | 儲存篩選條件 | P2 | Users can save frequently-used filter combinations (e.g., "Korean dramas after 2024") for quick access. | NEW |

### 3.6 媒體詳情頁

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P2-020 | 豐富資訊展示 | P0 | Poster, backdrop, Traditional Chinese synopsis, cast/director, genre, runtime, release date, dual ratings (TMDB + Douban). | NEW |
| P2-021 | 影集季/集列表 | P0 | Series detail page with expandable season/episode list showing titles, synopses, and subtitle availability status. | NEW |
| P2-022 | 相關推薦 | P1 | Recommend similar content based on current item; apply region/language filtering. | NEW |
| P2-023 | 串流平台資訊 | P1 | Show streaming platforms where the content is available in Taiwan (Netflix/Disney+/KKTV/…) via TMDB Watch Providers data. | NEW |
| P2-024 | 預告片播放 | P2 | Embed YouTube trailer when available. | NEW |
| P2-025 | 豆瓣連結 | P2 | Direct link to Douban page for Chinese-language reviews. | NEW |

---

## Phase 3: 請求流程 + 下載管理

> Goal: One-click request → auto-download → auto-fetch subtitles — fully automated pipeline.

### 3.7 請求系統

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P3-001 | 一鍵請求 | P0 | Request a movie or series directly from the explore page or detail page. | NEW |
| P3-002 | 影集部分請求 | P0 | Select specific seasons or individual episodes to request. | NEW |
| P3-003 | 請求狀態追蹤 | P0 | Request list page showing each request's status: pending / searching / downloading / completed / failed. | NEW |
| P3-004 | Sonarr/Radarr 串接（可選） | P0 | When Sonarr/Radarr is configured, route requests via their API; otherwise use Vido's built-in flow. | NEW |
| P3-005 | 自動字幕觸發 | P1 | Automatically trigger subtitle search (Phase 1 subtitle engine) upon download completion. | NEW |

### 3.8 下載任務管理

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P3-010 | qBittorrent 串接 | P0 | Manage download tasks via qBittorrent Web API (add/pause/delete/view progress). | DONE (Epic 4) |
| P3-011 | NZBGet 串接（可選） | P2 | Same as above but for Usenet downloads via NZBGet. | NEW |
| P3-012 | 下載進度即時更新 | P0 | SSE push of download progress to frontend; no manual refresh needed. | NEW |
| P3-013 | 下載完成通知 | P1 | Web UI notification on download completion (future: extend to Telegram/Discord). | NEW |
| P3-014 | 內建 BT 引擎（未來） | P3 | Built-in BT download using Go BT library (anacrolix/torrent), eliminating qBittorrent dependency. | NEW |

### 3.9 Indexer 管理

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P3-020 | Prowlarr 串接（可選） | P1 | When Prowlarr is configured, search indexers via its API. | NEW |
| P3-021 | 內建 indexer 搜尋 | P2 | When Prowlarr is not configured, Vido provides basic built-in torrent search capability (public trackers). | NEW |

---

## Phase 4: NAS Dashboard

> Goal: A single interface to monitor the full state of the NAS media system.

### 3.10 媒體庫統計

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P4-001 | 媒體庫總覽 | P0 | Movie/series count, disk usage, average file size, resolution distribution (4K/1080p/720p). | NEW |
| P4-002 | 字幕覆蓋率 | P0 | Pie/bar chart: has Traditional Chinese subtitles / has other subtitles / no subtitles. | NEW |
| P4-003 | 類型分佈 | P1 | Charts showing genre, region, and year distribution of the collection. | NEW |
| P4-004 | 最近新增 | P0 | List of media added in the last 7/30 days. | NEW |

### 3.11 Plex/Jellyfin 串接

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P4-010 | 觀看紀錄同步 | P1 | Sync watch history from Plex/Jellyfin for personalized recommendations. | NEW |
| P4-011 | 正在觀看 | P1 | Home page "Continue Watching" section based on Plex/Jellyfin watch progress. | NEW |
| P4-012 | 庫存同步 | P0 | Periodically scan Plex/Jellyfin libraries to mark content as already owned. | NEW |

### 3.12 服務健康監控

| ID | 功能 | Priority | Description | Status |
|----|------|----------|-------------|--------|
| P4-020 | 外部服務狀態 | P1 | Display connection status of all integrated services (Sonarr/Radarr/qBittorrent/Plex/Jellyfin). | NEW |
| P4-021 | 磁碟空間警告 | P1 | Alert when media library disk usage exceeds a configurable threshold. | NEW |
| P4-022 | 活動日誌 | P2 | Log all automated actions (subtitle downloads, request processing, scan results); searchable and filterable. | NEW |
