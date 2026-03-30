# Graceful Degradation — 執行手冊

> **日期**：2026-03-30
> **來源**：Party Mode 討論共識
> **議題**：電影 Detail 頁 404 Bug + 無 Metadata 媒體的 Graceful Degradation

---

## 總覽

| Phase | 名稱 | 流程 | 預估 Stories |
|-------|------|------|-------------|
| Phase 0 | Hotfix：修 404 | 直接執行（無 BMAD） | 0 |
| Phase 1 | Fallback UX 優化 | 輕量 BMAD（Story 級） | 1 |
| Phase 2 | Media Info + NFO 讀取 | 完整 BMAD（Epic 級） | 4-5 |
| Phase 3 | AI 差異化功能 | 完整 BMAD（Epic 級） | 2-3 |

---

## Phase 0：Cherry-pick Bugfix-1（立即執行）

### 不需要 Agent — 直接在 terminal 操作

```bash
# Step 1: Cherry-pick 三個 commits
git cherry-pick 6167599 89b98a5 d1f6da1

# Step 2: 如果有 conflict，手動 resolve 後
git cherry-pick --continue

# Step 3: 跑完整測試
cd apps/web && npx vitest run && cd ../..

# Step 4: 確認通過後 push
git push origin main
```

**驗證**：開瀏覽器訪問 `/media/movie/<任意UUID>`，確認不再 404。

---

## Phase 1：Fallback UX 優化

### 步驟 1/3 — Sally (UX Designer) 出設計稿

**指令**：`/bmad:bmm:agents:ux-designer`

**提示詞**：

```
我需要你設計「媒體 Detail 頁的 Fallback UI」—— 當電影/劇集沒有 TMDB metadata 時的呈現方式。

## 背景
- Vido 是 NAS 媒體庫管理工具，使用者的檔案真實存在於 NAS
- 目前 bugfix-1 已有基礎 fallback：顯示 filePath、fileSize、createdAt、parseStatus
- 但視覺上太「技術感」，使用者會覺得頁面壞了

## 設計需求

### 1. Color Placeholder Poster
- 無海報時，用檔名 hash 生成色塊背景 + 電影名首字母（類似 Gmail avatar）
- 需要跟現有的有 metadata 版本視覺一致

### 2. Fallback 頁面佈局
兩種狀態：
- **Pending 狀態**（正在 enrichment）：spinner + 「正在搜尋電影資訊⋯」
- **Failed/No-match 狀態**：顯示檔案資訊 + 引導動作

### 3. CTA 層級
- Primary CTA：「搜尋 Metadata」按鈕（大、顯眼）
- Secondary：「手動編輯」連結（小、次要）

### 4. 檔案資訊區塊
顯示：檔案名稱、檔案路徑、檔案大小、加入時間、解析狀態
用 icon + 文字，不要只顯示 raw path

### 5. 文案
全部使用繁體中文，語氣友善：
- 「我們找不到這部電影的資料」
- 「你可以手動搜尋，或等待系統自動比對」

## 設計稿位置
請在 ux-design.pen 中新增設計，可放在 Flow B（Detail 相關）附近。
需要包含：Desktop 版和 Mobile 版各一個畫面。

## 參考
- 現有 Detail 頁設計：Flow B 的 Movie Detail / TV Detail 畫面
- 現有色彩系統：參考 _bmad-output/planning-artifacts/ux-design-specification.md
```

---

### 步驟 2/3 — Bob (SM) 寫 Story

**指令**：`/bmad:bmm:workflows:create-story`

**提示詞**：

```
請建立一個新的 Story，屬於 Epic 5（Media Library Management）的增強。

## Story 概要
**Title**: Fallback UI Enhancement for Media Detail Page
**類型**: Enhancement / UX Improvement

## User Story
作為一個 Vido 使用者，
當我點進一部還沒有 TMDB metadata 的電影 detail 頁時，
我希望看到友善的 fallback 介面而不是空白或技術資訊，
讓我知道系統狀態並能主動搜尋 metadata。

## Acceptance Criteria
1. 無 poster 時顯示 color placeholder（檔名 hash 色塊 + 首字母）
2. Pending 狀態顯示 spinner + 「正在搜尋電影資訊⋯」文案
3. Failed 狀態顯示檔案資訊區塊（名稱、路徑、大小、時間）+ 引導文案
4. 「搜尋 Metadata」為 Primary CTA，「手動編輯」為 Secondary
5. 所有文案使用繁體中文
6. Desktop 和 Mobile 響應式佈局
7. 視覺風格與現有 Detail 頁一致

## 設計稿參考
- Sally 的設計稿位於 ux-design.pen（Flow B 附近的 Fallback 畫面）
- 截圖位於 _bmad-output/screenshots/

## 技術範圍
- 僅前端改動（apps/web/）
- 主要修改 apps/web/src/routes/media/$type.$id.tsx 的 fallback 區塊
- 新增 ColorPlaceholder component
- 不涉及 DB 或 API 改動

## 前置條件
- Phase 0 的 bugfix-1 cherry-pick 已完成（detail 頁基礎 fallback 已存在）
```

---

### 步驟 3/3 — Amelia (Dev) 執行 Story

**指令**：`/bmad:bmm:workflows:dev-story`

**提示詞**：

```
請執行 Phase 1 的 Fallback UI Enhancement Story。

Story 檔案位置：[Bob 建立的 story 路徑]

重點注意事項：
1. 基於 bugfix-1 cherry-pick 後的 code（apps/web/src/routes/media/$type.$id.tsx）
2. ColorPlaceholder component 用檔名做 hash 生成 HSL 色相，白色首字母
3. 文案全部繁體中文
4. 完成後對照 Sally 的設計稿截圖做 UX 驗證
```

---

## Phase 2：Media Info Pipeline + NFO 讀取（新 Epic）

### 步驟 1/6 — John (PM) 更新 PRD

**指令**：`/bmad:bmm:workflows:prd`（選擇 Edit 模式）

**提示詞**：

```
我需要在現有 PRD 中新增 Phase 2 的需求。PRD 位置：_bmad-output/planning-artifacts/prd/

## 新增需求

### P2-001: Media Technical Information（媒體技術資訊）
- 系統應在 scan 時提取影片的技術資訊：video codec、resolution、audio codec、audio channels、subtitle tracks
- 技術資訊應顯示在 Detail 頁面，以視覺 badges 形式呈現（如 H.265 · 4K · DTS）
- 資料來源優先級：NFO streamdetails > FFprobe 提取
- 必須支援的格式：MKV、MP4、AVI

### P2-002: NFO Sidecar 讀取（唯讀）
- 系統應在 scan 時偵測影片同名 .nfo 檔案
- 支援兩種 NFO 格式：完整 XML 和單行 TMDB URL
- NFO 提供的 metadata 優先級高於 AI parsing 和 TMDB enrichment
- NFO 中的 uniqueid（tmdb/imdb）應用於精準 TMDB match
- 只讀取，不寫入 NFO

### P2-003: Data Source Priority（資料來源優先級）
- 建立明確的 metadata 優先級：使用者手動修正 > NFO > TMDB enrichment > AI parsing
- 每筆媒體記錄應標記 metadata_source 欄位

### P2-004: Unmatched Media Filter（未匹配媒體篩選）
- Library 頁面新增「未匹配」篩選條件
- 讓使用者可以批量檢視所有沒有 TMDB metadata 的媒體
- 顯示未匹配數量 badge

### P2-005: Series File Size（劇集檔案大小）
- Series model 補齊 file_size 欄位（目前只有 Movie 有）
- Scan 時計算整季/整部劇集的總檔案大小

## 產品背景
這些需求來自與主流 NAS 媒體管理工具（Plex、Jellyfin、Emby、Infuse、Kodi）的差距分析。
技術資訊和 NFO 讀取是所有主流工具的標配功能。
NFO 讀取特別重要：可大幅加速從其他工具遷移來的使用者的 onboarding 體驗。
```

---

### 步驟 2/6 — Winston (Architect) 更新架構

**指令**：`/bmad:bmm:workflows:create-architecture`

**提示詞**：

```
我需要為 Phase 2（Media Info + NFO）做架構設計。

## 需要架構決策的項目

### 1. Scan Pipeline 擴充
現有 pipeline：檔案發現 → AI Parse → TMDB Enrichment
新 pipeline：檔案發現 → NFO 偵測 → (有 NFO) NFO Parse / (無 NFO) AI Parse → TMDB Enrichment → FFprobe 技術資訊提取

需要決定：
- NFO reader 和 FFprobe 在 pipeline 中的位置和執行順序
- 平行 or 串行？
- 失敗處理（NFO parse 失敗 fallback 到 AI parse）

### 2. DB Schema 變更
新增欄位（movies + series tables）：
- video_codec TEXT
- video_resolution TEXT (e.g., "3840x2160")
- audio_codec TEXT
- audio_channels INTEGER
- subtitle_tracks TEXT (JSON array)
- Series 補 file_size INTEGER
- metadata_source TEXT 需要支援新值 "nfo"

需要決定：
- 技術資訊存在同一 table 還是新建 media_info table？
- subtitle_tracks 的 schema（language、format、external flag）

### 3. FFprobe 整合
- FFprobe binary 怎麼打包？Docker image 加入 ffprobe？
- Go 層用什麼 library 呼叫？（推薦 floostack/transcoder 或直接 exec）
- 效能考量：大量檔案 scan 時 FFprobe 的並發策略

### 4. NFO Parser
- Go XML parsing strategy
- 兩種格式處理：完整 XML vs 單行 URL
- NFO 中 artwork 路徑（poster.jpg, fanart.jpg）要不要讀？

### 5. Data Source Priority 實作
- metadata_source 欄位的語義和寫入邏輯
- 後續 refresh 時如何尊重優先級（不覆蓋高優先級來源）

## 現有架構參考
- 架構文件：_bmad-output/planning-artifacts/architecture/
- Backend：apps/api/（Go + Gin）
- DB：SQLite with WAL mode
- 現有 scan 邏輯：apps/api/internal/services/scanner/
```

---

### 步驟 3/6 — Sally (UX Designer) 出設計稿

**指令**：`/bmad:bmm:agents:ux-designer`

**提示詞**：

```
我需要為 Phase 2 設計兩個 UI 元素。

## 1. 視覺 Badges（技術資訊標籤）
設計 Detail 頁面上的技術資訊 badges，參考 Infuse 的做法：
- Video：codec badge（H.264 / H.265 / AV1）+ resolution badge（4K / 1080p / 720p）
- Audio：codec badge（DTS / Atmos / AAC）+ channels（5.1 / 7.1 / Stereo）
- HDR badge（HDR10 / Dolby Vision / SDR — SDR 不顯示）
- Subtitle：語言 badges（繁中 / 簡中 / 英文 / 日文）

位置：Detail 頁面的 title 下方，年份和 runtime 同一列
風格：小型 pill badges，用顏色區分類別（video=藍、audio=紫、subtitle=綠）

## 2. Unmatched Filter
Library 頁面的篩選列新增「未匹配」選項：
- Filter chip 或 toggle
- 顯示未匹配數量（如 「未匹配 (12)」）
- 啟用後只顯示 tmdb_id 為空的媒體

## 設計稿位置
在 ux-design.pen 中：
- Badges 設計放在 Flow B（Detail）的現有畫面旁邊
- Unmatched filter 放在 Flow A（Browse）的現有畫面旁邊

## 參考
- 現有設計：ux-design.pen 的 Flow A 和 Flow B
- 色彩系統：_bmad-output/planning-artifacts/ux-design-specification.md
- Infuse badges 風格：小型 pill shape，半透明背景
```

---

### 步驟 4/6 — Bob (SM) 拆 Epic & Stories

**指令**：`/bmad:bmm:workflows:create-epics-and-stories`

**提示詞**：

```
請根據更新後的 PRD 和架構文件，為 Phase 2 建立一個新 Epic 並拆分 Stories。

## Epic 資訊
- 名稱建議：Epic N — Media Technical Info & NFO Integration
- 編號：接在現有 epic-list.md 最後一個 epic 之後
- 涵蓋 PRD 需求：P2-001 ~ P2-005

## 建議的 Story 拆分（依賴順序）

### Story 1: DB Schema Migration
- 新增 video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks 欄位
- Series 補 file_size
- metadata_source 支援 "nfo" 值
- 純 backend，無前端

### Story 2: NFO Sidecar Reader
- Scan pipeline 新增 NFO 偵測 stage
- Go XML parser 讀取 NFO
- 支援完整 XML 和單行 URL 兩種格式
- NFO 的 uniqueid 用於 TMDB 精準 match
- 前置：Story 1

### Story 3: FFprobe Integration
- Docker image 加入 ffprobe
- Go service 呼叫 FFprobe 提取技術資訊
- 寫入 DB 新欄位
- Media Info API endpoint
- 前置：Story 1
- 可與 Story 2 平行開發

### Story 4: Technical Info Badges UI + Unmatched Filter
- 前端 Badge components
- Detail 頁面整合 badges
- Library 頁面 Unmatched filter
- 前置：Story 2, Story 3（需要 API 有資料）

## 參考文件
- PRD：_bmad-output/planning-artifacts/prd/
- 架構：_bmad-output/planning-artifacts/architecture/
- 設計稿：ux-design.pen + _bmad-output/screenshots/
- 現有 epic list：_bmad-output/planning-artifacts/epics/epic-list.md
```

---

### 步驟 5/6 — 實作準備度檢查

**指令**：`/bmad:bmm:workflows:check-implementation-readiness`

**提示詞**：

```
請檢查 Phase 2 Epic（Media Technical Info & NFO Integration）的實作準備度。

確認以下文件已齊備且一致：
1. PRD 中的 P2-001 ~ P2-005 需求
2. 架構文件中的 scan pipeline、DB schema、FFprobe 整合方案
3. Sally 的設計稿（badges + unmatched filter）
4. Bob 的 Stories（4 個 stories 的 AC 和 tasks）

特別注意：
- Story 2 (NFO) 和 Story 3 (FFprobe) 的平行開發是否有 conflict
- DB migration 是否完整涵蓋所有新欄位
- Data source priority 邏輯在架構和 story AC 中是否一致
```

---

### 步驟 6/6 — 逐 Story 開發

**指令**：對每個 Story 執行 `/bmad:bmm:workflows:dev-story`

**提示詞（通用模板）**：

```
請執行 [Story 名稱]。

Story 檔案位置：[Bob 建立的 story 路徑]

注意事項：
- 遵循 project-context.md 的開發規範
- 每個 task 完成後跑測試
- 完成後對照設計稿做 UX 驗證（如涉及前端）
- DB migration 使用 SQLite 相容語法
```

---

## Phase 3：AI 差異化功能（新 Epic）

### 步驟 1/5 — John (PM) 更新 PRD

**指令**：`/bmad:bmm:workflows:prd`（Edit 模式）

**提示詞**：

```
在 PRD 中新增 Phase 3 的需求。

### P3-011: AI Confidence Display & One-Click Confirm
- 當 AI file name parsing 產生結果但信心度不足以自動 match 時
- Detail 頁面 fallback UI 顯示：「AI 辨識結果：可能是《電影名》（85%）— 是這部嗎？」
- 使用者可一鍵確認（[✓ 是這部] / [✗ 不是，搜尋其他]）
- 確認後觸發 TMDB enrichment，metadata_source 標記為 "ai_confirmed"

### P3-012: Metadata Lock
- 使用者手動修正或確認 metadata 後，可鎖定防止自動 refresh 覆蓋
- UI 上顯示鎖定 indicator（小鎖頭 icon）
- 鎖定的項目在 re-scan 時跳過 metadata 更新
- 可隨時解鎖

### P3-013: Smart Match Suggestions
- 基於 AI parsing 結果 + 檔案特徵（年份、解析度、語言）
- 自動推薦 2-3 個可能的 TMDB 結果
- 在 fallback UI 中以卡片形式展示候選結果
- 使用者點選即完成 match

## 產品定位
這些是 Vido 的差異化功能，主流工具（Plex/Jellyfin/Emby/Infuse/Kodi）都沒有。
核心價值：利用 AI parsing pipeline 減少使用者手動操作。
```

---

### 步驟 2/5 — Winston (Architect) 架構設計

**指令**：`/bmad:bmm:workflows:create-architecture`

**提示詞**：

```
為 Phase 3（AI 差異化功能）做架構設計。

## 需要決策的項目

### 1. AI Confidence Score
- 現有 parse pipeline 是否已產生 confidence score？還是只有 pass/fail？
- 如果沒有，需要在哪一層加入 confidence 計算？
- Score 的 threshold 設計：多少以上自動 match？多少以下顯示確認 UI？
  建議：>90% 自動 match、50-90% 顯示確認、<50% 不建議

### 2. Metadata Lock 機制
- DB 層：新增 metadata_locked BOOLEAN + locked_at TIMESTAMP
- Scan pipeline：遇到 locked 項目跳過 metadata 更新
- API：PATCH /api/v1/movies/:id/lock、/unlock

### 3. Smart Match Suggestions
- 觸發時機：AI parse 完成但 confidence < threshold
- 實作：用 AI parsed title + year 查 TMDB search API，取 top 3
- 快取策略：suggestions 存 DB 還是 on-demand？

## 現有參考
- AI parse pipeline：apps/api/internal/services/parser/
- TMDB integration：apps/api/internal/services/tmdb/
- DB models：apps/api/internal/models/
```

---

### 步驟 3/5 — Sally (UX) 設計 AI 確認流程

**指令**：`/bmad:bmm:agents:ux-designer`

**提示詞**：

```
設計 Phase 3 的 AI 差異化 UI。

## 1. AI Confidence 確認卡片
在 fallback UI 中，當 AI 有辨識結果時顯示：
- AI 辨識結果區塊：「AI 認為這可能是⋯」
- 候選電影卡片（poster 縮圖 + 標題 + 年份 + 信心度百分比）
- 兩個按鈕：[✓ 是這部] [✗ 不是]
- 點 [✓] 後 → 觸發 enrichment → transition 到完整 detail
- 點 [✗] 後 → 展開搜尋框讓使用者手動搜

## 2. Smart Match Suggestions（多候選）
- 當 confidence 中等時，顯示 2-3 個候選
- 水平排列的小卡片，每個有 poster + title + year + match %
- 使用者點選其中一個即完成 match
- 底部有「以上都不是，手動搜尋」連結

## 3. Metadata Lock Indicator
- 鎖定狀態：Detail 頁 title 旁邊顯示小鎖頭 🔒
- Hover/click 顯示 tooltip：「Metadata 已鎖定，不會被自動更新。點擊解鎖。」
- 鎖定切換：在 Detail 頁的 ⋯ menu 中加入「鎖定/解鎖 Metadata」選項

## 設計稿位置
ux-design.pen，Flow B 附近新增畫面。
需要：Desktop + Mobile 各一組。
```

---

### 步驟 4/5 — Bob (SM) 拆 Epic & Stories

**指令**：`/bmad:bmm:workflows:create-epics-and-stories`

**提示詞**：

```
為 Phase 3 建立新 Epic 並拆分 Stories。

## Epic 資訊
- 名稱：Epic N — AI-Powered Metadata Matching & Lock
- 涵蓋：P3-011, P3-012, P3-013

## 建議 Story 拆分

### Story 1: AI Confidence Score Infrastructure
- Parse pipeline 加入 confidence score 計算
- DB 新增 parse_confidence FLOAT 欄位
- API 回傳 confidence score
- 純 backend

### Story 2: One-Click Confirm UI + Metadata Lock
- 前端 AI confidence 確認卡片
- 確認後觸發 enrichment
- Metadata lock toggle（DB + API + UI）
- 跨全棧

### Story 3: Smart Match Suggestions
- Backend：用 parsed title 查 TMDB，取 top 3 候選
- API：GET /api/v1/movies/:id/suggestions
- 前端：候選卡片 UI
- 前置：Story 1

## 前置條件
- Phase 2 Epic 已完成（需要 media info 和 NFO 基礎設施）
```

---

### 步驟 5/5 — 準備度檢查 + 開發

同 Phase 2 的步驟 5/6 和 6/6。

---

## 決策記錄（Party Mode 共識）

### 該跟主流學的
- [x] 技術資訊（codec/resolution/audio）— 所有主流工具標配
- [x] 視覺 badges — Infuse 開創的趨勢
- [x] 顯眼的 manual match CTA — 主流共識
- [x] Unmatched 批量篩選 — Plex/Emby 有
- [x] NFO 讀取（只讀）— 加速遷移使用者 onboarding

### 不學、走自己路的
- [ ] ~~影片截圖當 thumbnail~~ — 增加 scan 時間，Vido 非播放器
- [ ] ~~NFO 寫入~~ — 只讀就夠
- [ ] ~~檔名嵌入 ID（{tmdb-123}）~~ — 我們有 AI parsing
- [x] AI confidence + 一鍵確認 — **Vido 獨有差異化**
- [x] 極致繁中 UX — **zh-TW 使用者痛點**

### Data Source 優先級
```
1. 使用者手動修正（最高）
2. NFO sidecar
3. TMDB enrichment
4. AI file name parsing（最低）
```

### 主流工具比較摘要

| 功能 | Plex | Jellyfin | Emby | Infuse | Kodi | Vido (目標) |
|------|------|----------|------|--------|------|------------|
| 技術資訊 | ✅ | ✅ | ✅ | ✅ Badge | ✅ | ✅ Badge (Phase 2) |
| NFO 支援 | ❌ | ✅ | ✅ | ❌ | ✅ | ✅ 只讀 (Phase 2) |
| Manual match | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (Phase 1) |
| Unmatched filter | ✅ | ❌ | ✅ | ❌ | ❌ | ✅ (Phase 2) |
| AI auto-suggest | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (Phase 3) |
| Metadata lock | ❌ | ✅ | ✅ | ❌ | ❌ | ✅ (Phase 3) |
