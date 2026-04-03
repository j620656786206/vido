# Graceful Degradation — 執行手冊

> **日期**：2026-03-30
> **來源**：Party Mode 討論共識
> **議題**：電影 Detail 頁 404 Bug + 無 Metadata 媒體的 Graceful Degradation

---

## 總覽

| 區塊 | 名稱 | 流程 | 預估 Stories |
|------|------|------|-------------|
| Phase 0 | Hotfix：修 404 | 直接執行（無 BMAD） | 0 |
| Phase 1 | Fallback UX 優化 | 輕量 BMAD（Story 級） | 1 |
| Phase 2 | Media Info + NFO 讀取 | 完整 BMAD（Epic 級） | 4-5 |
| Phase 3 | AI 差異化功能 | 完整 BMAD（Epic 級） | 2-3 |

---

## Phase 0：Cherry-pick Bugfix-1

- [x] **Action: Cherry-pick bugfix-1 commits** — `git cherry-pick 6167599 89b98a5 d1f6da1`，resolve conflicts，跑測試，push。驗證 `/media/movie/<UUID>` 不再 404。（完成：bugfix-1 done 2026-03-28）

---

## Phase 1：Fallback UX 優化

- [x] **Action: UX 設計稿** — Sally 出 Fallback UI 設計（Color Placeholder Poster、Pending/Failed 狀態、CTA 層級、檔案資訊區塊），Desktop + Mobile 各兩個畫面，放在 ux-design.pen Flow B 附近。（完成：Screen 4d/4e/5b/5c，commit `735c67e` 2026-04-02）

- [x] **Action: 建立 Story** — Bob 建立 Story 5-11 Fallback UI Enhancement，屬 Epic 5 增強。純前端改動，7 條 AC、6 個 Tasks。（完成：`5-11-fallback-ui-enhancement.md` ready-for-dev 2026-04-02）

- [x] **Action: 執行 Story** — Amelia 執行 `/bmad:bmm:workflows:dev-story`，基於 bugfix-1 後的 code，實作 ColorPlaceholder component、Pending/Failed 狀態 UI、繁中文案、響應式佈局，完成後對照設計稿截圖做 UX 驗證。（完成：Story 5-11 review 2026-04-03，3 new components + 30 unit tests + 4 E2E tests，1593 total tests pass）

---

## Phase 2：Media Info Pipeline + NFO 讀取（新 Epic）

- [ ] **Action: PM 更新 PRD** — John 使用 `/bmad:bmm:workflows:prd`（Edit 模式），新增 P2-001~P2-005 需求（Media Technical Info、NFO Sidecar 讀取、Data Source Priority、Unmatched Filter、Series File Size）。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: Architect 更新架構** — Winston 使用 `/bmad:bmm:workflows:create-architecture`，決策 Scan Pipeline 擴充順序、DB Schema（media_info 新 table 或現有 table 擴充）、FFprobe Docker 整合、NFO Parser、Data Source Priority 實作。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: UX 設計稿** — Sally 出技術資訊 badges（video/audio/subtitle pill badges）和 Unmatched Filter UI（Library 頁面篩選列），Desktop + Mobile，放 ux-design.pen Flow A/B 附近。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: 拆 Epic & Stories** — Bob 使用 `/bmad:bmm:workflows:create-epics-and-stories`，建立新 Epic（Media Technical Info & NFO Integration），拆 4 stories：DB Schema Migration → NFO Reader → FFprobe Integration → Badges UI + Unmatched Filter。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: 實作準備度檢查** — 使用 `/bmad:bmm:workflows:check-implementation-readiness`，確認 PRD、架構、設計稿、Stories 齊備且一致。特別注意 Story 2/3 平行開發 conflict、DB migration 完整性、Data Source Priority 一致性。

- [ ] **Action: 逐 Story 開發** — 對每個 Story 執行 `/bmad:bmm:workflows:dev-story`，遵循 project-context.md 規範，每個 task 完成後跑測試，涉及前端時對照設計稿，DB migration 使用 SQLite 相容語法。

---

## Phase 3：AI 差異化功能（新 Epic）

- [ ] **Action: PM 更新 PRD** — John 使用 `/bmad:bmm:workflows:prd`（Edit 模式），新增 P3-011（AI Confidence + One-Click Confirm）、P3-012（Metadata Lock）、P3-013（Smart Match Suggestions）。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: Architect 架構設計** — Winston 使用 `/bmad:bmm:workflows:create-architecture`，決策 AI Confidence Score 計算層與 threshold、Metadata Lock 機制（DB + API + Scan pipeline skip）、Smart Match Suggestions 觸發與快取策略。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: UX 設計 AI 確認流程** — Sally 出 AI Confidence 確認卡片、Smart Match Suggestions 多候選卡片、Metadata Lock Indicator，Desktop + Mobile，放 ux-design.pen Flow B 附近。

  <details><summary>提示詞</summary>

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

  </details>

- [ ] **Action: 拆 Epic & Stories + 準備度檢查 + 開發** — 同 Phase 2 流程：create-epics-and-stories → check-implementation-readiness → 逐 Story dev-story。

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
