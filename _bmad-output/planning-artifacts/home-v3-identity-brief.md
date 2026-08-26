# Home v3 — 首頁身分改版設計 Brief（shape 產出，2026-08-26）

> /impeccable shape 訪談結論。裁定人：Alexyu（兩輪五題）。
> 下一步：照慣例先進 ux-design.pen（H1-D-v3 桌機＋手機），Sally 出 Inline
> Agent 提示詞 → Alexyu 跑 → Sally MCP review → 截圖 → 才實作。
> 收斂三條立案：disc-2026-08-home-identity-hero（本體）、
> disc-2026-08-home-explore-owned-sea（裁：濾掉已有）、
> disc-2026-08-home-hero-autoplay-alien（裁：靜止＋手動）。

## 1. Job and audience

回來查的人。他把 Vido 當無人值守的字幕工人：關掉視窗走開，回來的第一個
問題是「我不在的時候你做了什麼？有沒有事需要我？」。第二身分才是「今晚
看什麼」的瀏覽者。Operate surface——先讀數，再瀏覽，最後才是探索。

## 2. Outcome and proof

三秒內回答四件事（讀數帶四格，全部經裁定）：
1. **繁中覆蓋率**「42/55 部有繁中字幕」——產品存在理由的單一數字
2. **不在時處理報告**「今天處理了 3 部」
3. **需要注意的事**「2 部失敗待處理 · 花費 $1.2/$5」——例外＋錢
4. **正在進行中**「2 個任務執行中」——與側欄 badge 同源

每格都是門（點了去對應的地方：媒體庫缺字幕篩選／活動中心／活動中心
例外區／活動中心）。誠實規則承襲全站：量不到的格不顯示數字；0 的格
顯示 0（0 是資訊，不是例外——覆蓋率 55/55 是慶祝，不是隱藏）。

## 3. Selected direction（裁定：「兩個都要」）

首頁骨架由上而下：
1. **讀數帶**（新）——一條密集的四格橫帶，Operate 語氣，小字＋mono 數
   字，貼在頁首。不是 dashboard 卡片群，是一條「帶」。
2. **自家片庫 hero**（改）——大幅劇照輪播，內容改為**自己片庫**最新入
   庫且有 backdrop 的 3–5 部；**靜止預設，圓點/箭頭手動切換，無自轉**
   （裁定：移除操作者的產品不該有自己會動的元件）。字幕狀態條（繁中✓
   已就緒／缺字幕→門）直接貼在 hero 資訊區。CTA=查看詳情。無 backdrop
   可用時 hero 整段不渲染（例外訊號原則）。
3. **最近新增**（保留現狀）。
4. **TMDb 外部策展**（降級＋過濾）——退到頁尾；探索列**濾掉已擁有**
   （裁定：發現才是它的工作）；TMDb hero 輪播退場（自家 hero 取代）。
   沒 key 時整塊不存在，頁面仍完整——降級態與完整態同構。

## 4. Scope and boundaries

- Fidelity：先 .pen 設計稿（H1-D-v3 桌機＋手機兩張，含降級態變體），
  視覺確認後才開實作 story。
- 動骨架不動世界：夜行 token、既有元件庫（PosterCardV2、Status Rows、
  徽章配方）全部沿用；這是 Operate 改版，不是 reskin。
- 不動：側欄/shell、最近新增列內部、活動中心。
- Anti-goals：不做 dashboard 卡片牆；不引入新狀態色；讀數帶不放圖表
  （sparkline 禁區）；hero 不自轉；不假造「上次造訪」若後端只能給
  「今天」——文案跟著資料能力走。

## 5. States and ranges

- 片庫 0 部（首跑）：讀數帶顯示 0/0＋「開始掃描」門；hero 不渲染。
- 片庫 1–5 部：hero 可能只有 1 張——圓點隱藏，單張靜止。
- 55 部/2406 集（作者實庫）：覆蓋率分母以「部」計，影集以劇集為單位
  的細節留給媒體庫頁。
- 降級（無 TMDb）：§3 之 4 整塊消失；讀數帶與 hero 不依賴 TMDb。
- 全部完成（0 失敗 0 進行中）：注意格顯示「一切正常」而非消失——
  「沒有壞消息」本身是這個產品要賣的好消息。

## 6. Interaction and layout

- 讀數帶：單行 flex，4 格等權，11px 標籤＋mono 數字（比較才用等寬）；
  行動裝置 2×2。每格整格可點（≥44px）。例外格（失敗>0）文字穿
  --warning-text——琥珀語意（要求了但沒發生）。
- hero：與 R2/R3 對齊紀律一致（mx-auto max-w-7xl px-4 sm:px-6 共線）；
  高度沿用 250/400；字幕狀態條用既有徽章配方。
- 探索列過濾：isOwned 已在 hoisted ownership——過濾在前端做，零新 API。

## 7. Constraints and open decisions（builder 不得自行發明）

- **需要一個便宜的彙總來源**：覆蓋率與處理報告目前沒有 API
  （/library/stats 只有片數）。方案：擴充 GET /api/v1/activity 或新
  GET /api/v1/home-summary，一次回四格。**「不在時」的時間錨點**（上次
  造訪 vs 過去 24h）依後端成本決定，文案跟著改（「今天處理了 N 部」
  是安全預設）——實作前需確認。
- 花費 $X/$Y 取數路徑需確認（AI_RUN_BUDGET_USD 是單次上限，讀數帶顯示
  的應是「本次/最近一次執行」的消耗——若無現成值，此格先只放例外數）。
- .pen 畫布：新 frame H1-D-v3（-d/-m），依 flow 佈局慣例放 flow-h 區塊；
  SCREENS dict 與截圖匯出照 CLAUDE.md 流程。
