# TestSprite 測試報告 — 2026-08-31 八月額度衝刺(內測前基本面驗證)

## 1️⃣ Document Metadata

- **Project:** vido
- **Date:** 2026-08-31
- **環境:** serve-test-env.sh :8090(production build,seeded fixtures)
- **額度:** 64+ credits(150 月配額,當日到期前使用)
- **背景:** 內測首個真機安裝(Synology)踩到 randomUUID 崩潰後,⚖️ Alexyu 指示先以基本 integration test 驗證再定內測開放程度

## 2️⃣ Requirement Validation Summary

### Setup Wizard(新增 TC106–109,首次覆蓋)— 4/4 ✅

- TC106 跳過 qBittorrent 後媒體庫步驟正常渲染(= v0.1.0 崩潰路徑,驗證 #358 修復)✅
- TC107 上一步/下一步保留輸入值 ✅
- TC108 完整走完精靈落地主畫面 ✅
- TC109 完成後不再重入精靈 ✅

### 核心瀏覽(B1)— 9/10

- 首頁排序/海報開詳情/Explore 區塊(TC093/094/096)✅、即時搜尋(TC092)✅
- 媒體庫 list/sort/filter/toggle(TC009/012/104/105)✅、詳情面板(TC084)✅
- TC085 ❌ → 判定**測試計畫問題**:挑到無技術 metadata 的 fixture + 猜測 data-testid(立案 plan-drift-tc085)

### 活動/下載/字幕/掃描(B2)— 5 過、1 真 bug、1 計畫過期、3 環境限制

- 活動中心(TC101/102)✅、下載頁載入(TC079)✅、字幕對話框兩入口(TC071/077)✅
- **TC064 ❌ 真 bug(P2 UX)**:小片庫掃描毫秒完成,SSE 未及連上 → 無進度卡/完成通知,按鈕看似無反應。本地 Playwright 重現確認(立案 bugfix-scan-instant-completion-no-feedback)
- TC072 ❌ 計畫過期:期待已退役的 provider-checkbox 對話框(含已移除的 Zimuku);現行 UI 為 ManageSubtitleDialogV2(立案 plan-drift-tc072)
- TC043/080/081 🚧 環境限制:測試環境未接 qBittorrent,連線類旅程誠實卡住 — 非產品缺陷

### 設定/驗證/深連結(B3)— 10/10 ✅

- qBT 設定表單/驗證/遮罩(TC035/040/041)✅、掃描設定+排程持久化(TC063/068)✅
- 字幕對話框 Esc/選單入口(TC076/078)✅、URL-UI 一致性三連(TC089/090/091)✅

## 3️⃣ Coverage & Matching Metrics

| 批次                   | 案數   | ✅     | ❌真bug | 📝計畫過期 | 🚧環境 |
| ---------------------- | ------ | ------ | ------- | ---------- | ------ |
| Wizard                 | 4      | 4      | 0       | 0          | 0      |
| B1 核心瀏覽            | 10     | 9      | 0       | 1          | 0      |
| B2 活動/下載/字幕/掃描 | 10     | 5      | 1       | 1          | 3      |
| B3 設定/深連結         | 10     | 10     | 0       | 0          | 0      |
| **合計(34 案)**        | **34** | **28** | **1**   | **2**      | **3**  |

隧道抽風紀錄:wizard 首輪 2 案與 TC102 首輪誤判,重試即過(已知 MCP tunnel 特性;先本地 Playwright 求證再定罪產品)。

## 4️⃣ Key Gaps / Risks

1. **bugfix-scan-instant-completion-no-feedback**(P2)— 唯一真 bug,朋友第一次掃小片庫會遇到;內測附註可帶過,修復已排入
2. 測試計畫兩處落後於 v2 UI(TC072/TC085)— 更新案例,非產品風險
3. qBittorrent 連線類旅程(3 案)在本環境不可測 — 內測依賴朋友真機回饋
4. **內測判讀:主動脈(安裝→精靈→掃描→瀏覽→搜尋→詳情→字幕→活動)28 案實證通過,v0.1.1 可開放內測**
