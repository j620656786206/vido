---
name: TestSprite Integration Plan
description: TestSprite MCP installed but deferred until Epic 5+6 complete — tracks prerequisites, test strategy, and 6 critical user journeys
type: project
---

## TestSprite 旅程測試整合計畫

**狀態：** 已安裝，等待啟動（目標：Epic 5 + 6 完成後）
**日期：** 2026-03-15
**帳號：** Free plan, 150 credits, j620656786206@gmail.com

### 為什麼延後

2026-03-15 首次執行結果：15 測試跑了，3 通過 12 失敗。失敗原因：
1. TestSprite 雲端沙箱連不到 localhost（需要 tunnel 或 staging）
2. 媒體庫是空的（需要 seed data）
3. 測試預期英文 UI 但實際是 zh-TW（標籤不匹配）
4. Epic 5 才完成 3/8，核心頁面功能不齊全

**Why:** 跑不完整的 app 只會測到基礎設施問題，不是產品品質。
**How to apply:** Epic 5+6 完成後再啟動，屆時提醒使用者準備 prerequisites。

### 啟動前 Prerequisites

1. **Epic 5（媒體庫管理）全部完成** — 列表視圖、搜尋、排序、篩選、詳情頁完整版、批次操作
2. **Epic 6（系統設定）完成** — 設定頁面完整
3. **Seed data 腳本** — 建立 `scripts/seed-test-data.sh` 預填 10-20 筆媒體和模擬下載
4. **外部存取方式** — ngrok/cloudflared tunnel 或部署到 staging
5. **Production mode 執行** — `npm run build && npm run preview` 避免 dev server 15 測試限制

### 6 條 P0 關鍵旅程

1. **搜尋→瀏覽→詳情** — 輸入中文搜尋→結果列表→點進詳情頁→返回
2. **檔名解析→元資料匹配→手動修正** — fansub 檔名→AI 解析→TMDb 配對→失敗時手動搜尋→學習修正
3. **下載監控全流程** — Dashboard→即時下載→篩選→展開詳情→排序
4. **連線健康→降級→恢復** — 來源不可用→降級橫幅→核心功能仍可用→恢復後橫幅消失
5. **媒體庫瀏覽與互動** — Grid View→最近新增→點擊海報→詳情頁→返回
6. **qBittorrent 連線設定** — 設定頁→輸入連線資訊→測試連線→儲存→Dashboard 顯示資料

### 與 Playwright 的分工

| 層級 | 工具 | 覆蓋 | 何時跑 |
|------|------|------|--------|
| L1 關鍵旅程 | TestSprite | PRD AC 級別完整流程 (62 cases) | 每次 PR / staging deploy |
| L2 功能點 | Playwright | 個別畫面/API 細節驗證 (328 cases) | CI nightly / 改到相關 story 時 |

### 已產生的 TestSprite 資產

- Config: `testsprite_tests/tmp/config.json`
- Code summary: `testsprite_tests/tmp/code_summary.yaml`
- PRD files: `testsprite_tests/tmp/prd_files/` (9 個檔案)
- Test plan: `testsprite_tests/testsprite_frontend_test_plan.json` (62 test cases)
- Raw report: `testsprite_tests/tmp/raw_report.md`
- MCP 安裝: `claude mcp add TestSprite` (project scope, key in .claude.json)

### Epic 7 (Auth) 完成後追加

- 加入登入流程的旅程測試（needLogin: true）
- 納入 CI/CD pipeline
