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

- [x] **Action: 執行 Story** — Amelia 執行 `/bmad:bmm:workflows:dev-story`，基於 bugfix-1 後的 code，實作 ColorPlaceholder component、Pending/Failed 狀態 UI、繁中文案、響應式佈局，完成後對照設計稿截圖做 UX 驗證。（完成：Story 5-11 done 2026-04-03，CR fixes applied: mediaType prop, test assertions, file list, AC1 clarification）

---

## Phase 2：Media Info Pipeline + NFO 讀取（新 Epic）

- [x] **Action: PM 更新 PRD** — John 使用 `/bmad:bmm:agents:pm`（Edit PRD），將 5 個需求重新分配：P1-030~033（Media Tech Info、NFO Sidecar、Data Source Priority P0、Series File Size）放 Phase 1 Section 1.4；P2-030（Unmatched Filter）放 Phase 2 Section 2.4。Phase 1 時程調整為 10-13 週。（完成：commit `4b1ab5f` 2026-04-03）

- [x] **Action: Architect 更新架構** — Winston 建立 ADR `adr-media-info-nfo-pipeline.md`，涵蓋五項決策：(1) Scan pipeline NFO→AI→TMDB→FFprobe 串行流程 (2) 直接擴充 movies/series tables，Migration 021 (3) FFprobe via `os/exec` + semaphore(3) + 10s timeout (4) NFO XML/URL 雙格式 parser (5) metadata_source 優先級鏈 manual>nfo>tmdb>douban>wikipedia>ai。（完成：ADR accepted 2026-04-03）

- [x] **Action: UX 設計稿** — Sally 建立 4 個 reusable badge 元件（TechBadge-Video 藍/Audio 紫/Subtitle 綠/HDR 金）+ Screen 4f（Desktop）和 5d（Mobile）展示 tech badges 整合效果。Unmatched Filter 已在 Flow H（H7/H8）設計完成，無需重做。（完成：commit `c5b36e1` 2026-04-03）

- [x] **Action: 拆 Epic & Stories** — Bob 使用 `/bmad:bmm:workflows:create-epics-and-stories`，建立 Epic 9c（Media Technical Info & NFO Integration），拆 4 stories：9c-1 DB Schema Migration → 9c-2 NFO Reader → 9c-3 FFprobe Integration → 9c-4 Badges UI + Unmatched Filter。（完成：commit `54b03a2` 2026-04-03，epic file + epic-list + sprint-status 已更新）

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
