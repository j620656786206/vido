# 交接：設計系統重建（PR #410）

**日期**：2026-09-10 · **分支**：`chore/pen-design-tokens` · **PR**：#410（開著，CI 綠）

---

## 這個 PR 在做什麼（價值命題）

**改變設計系統本身**，不是讓設計稿追上 App 現有的實作。

⚖️ Alexyu 2026-09-10 明確裁定：**不要拿程式碼當基準去核對設計稿**。程式碼只是目前的實作。凡是兩邊不一致，要問的是「**哪一邊才是對的設計**」；答案是程式碼錯，就寫成「程式碼要改」，另開 PR。

**這個 PR 只動設計稿與設計文件，不動 `apps/web`。** 程式碼那一批另開 PR。

**Alexyu 的後續計畫**：Design System、圖層、元件全部確定之後，會**逐一修正 Flow A–M 的所有設計稿內容**，最後才實作到程式碼裡。所以現在的優先順序是「把系統本身弄對」，畫面內容之後才動。

---

## 起因

設計稿的顏色變數停在半年前的訊號藍配色，而 App 早已換成夜行（墨綠＋泥金）並新增日巡淺色主題。405 張畫面裡 206 張渲染成舊藍色，只有 14 張看起來對——因為那 14 張把新顏色一個一個寫死、繞過變數。拿設計稿開發的 Story 都拿到錯的畫面。

---

## 已經完成的（19 個 commit）

1. **顏色**：33 個變數改成 `styles.css` 真值；913 處寫死的顏色、815 處圓角改吃變數。
2. **日巡**：同一個變數名存兩組值（夜行／日巡），沒標主題的畫面照樣是夜行；新增日巡色票頁。
3. **圖層**：最外層從 406 個節點收成 13 個（1 個 Design System ＋ 12 個流程群組）。Design System 底下再分 `Docs · 設計文件` 與 `Components · 元件`（五個分類，每類有底板與標題）。
4. **元件**：35 個母版全部叫 `Component/*`，22 個原本埋在 Component Library 排版列裡的母版搬進分類、原位置放 instance。43 個無名 instance 補上母版名稱。
5. **字級**：砍成八階全偶數，每階帶配對行高（Display 36/40 … Label 12/16），5136 個節點遷移。
6. **元件解剖頁**：六個代表性元件，左原樣右解剖。
7. **文件**：`DESIGN.md`、`.impeccable/design.json`、`design-context-pack.md` 全部改寫成夜行＋日巡。
8. **漂移檢查器**：`scripts/check-design-tokens.py` 比對四份文件的色彩 token，掛進 CI（`Lint & Format Check` 的 `Design token drift check`）。**刻意只檢查不覆寫**——不一致時要先判斷哪一邊才是對的設計。
9. **四張 P0 裁定落地**（見下）。

---

## 已裁定的四件事（不要再推翻）

| # | 裁定 | 理由摘要 |
|---|---|---|
| 1 | **泥金＝正在跑、青碧＝有答案了** | DESIGN.md 原寫「綠＝正在發生」，但 `EpisodeList.tsx:76` 有更早的相反裁定（Sally 2026-07-05）。兩份正典矛盾三個月。採用程式碼那套，因為語意更好：正在跑＝「你在這裡、正在花你的資源」正是泥金的角色，青碧留給終局狀態，與硃砂成對。 |
| 2 | **語意基色不可以當文字** | success/warning/error/info/accent-primary 當文字，五個裡四個至少在一個主題不過 AA（赭最慘，兩個主題都不過）。設計稿 379 個違反節點已改成 `-text` 階。 |
| 3 | **技術標籤一律中性** | 技術規格是檔案的屬性不是「發生了什麼」。原本 video→accent／audio→info／hdr→warning／subtitle→success 讓青碧在相鄰畫面有兩個意思。改成 `bg-tertiary` 底＋`text-secondary` 字。 |
| 4 | **金錢不給顏色** | 金額是事實不是狀態，$25.80 不會「發生」也不會「失敗」。46 個穿狀態色的金額轉中性；狀態改押在標籤與圖示上。會花錢的動作用**記號**（`$` 或硬幣圖示＋預估金額）不用顏色。 |

---

## 剩下的 25 張單子

全部在 `_bmad-output/implementation-artifacts/sprint-status.yaml`，用 `disc-2026-09-` 前綴搜尋，每張的註解開頭標著 `**P0**` / `**P1**` / `**P2**`，內含完整證據與數字。

### 進行中（設計稿已定，剩程式碼——**另開 PR**）

- `techbadge-uses-status-colors-as-taxonomy` — `TechBadge.tsx:15-18` 還是舊映射
- `no-cost-bearing-component` — 記號要先在設計稿長出母版，程式碼再跟

### P0（1 張，主要是程式碼）

- `type-scale-even-migration` — 程式碼 125 處 `text-[13px]`／`text-[11px]`、6 處 10px、4 處 15px、5 處 `text-3xl`(30)、1 處 `text-5xl`(48)；還有 `Button.tsx` 的字重寫在 cva 共用基底（六個 variant 共用，但實底深字與透空淺字的極性相反，不該共用）、`px-4` 要改 `px-5`、缺 44px 的 `touch` 尺寸

### P1（11 張，**這些是設計稿的事，優先做**）

- `reference-page-covers-12-of-33-colors` — 參考頁只畫 12 個色票，所有 tint 與 `-text` 沒有色票，而「兩種金規則」與「徽章底用 tint、字用 -text」正是最容易踩壞的兩條
- `missing-component-masters` — 缺 Dialog／Sheet／Toast／Card／Input／Table／Tooltip／Skeleton／Pagination／Switch，以及 Outline／Ghost／Destructive 按鈕與四個狀態徽章
- `light-theme-screens` — 全檔只有 1 個節點帶 theme 屬性，日巡零張畫面。建議補三張：`b3-d`（scrim 疊字）、`h1-d-v3`（四種語意色同框）、`a2-d`（高密度網格）
- `radius-pill-rule-vs-reality` — 「絕不做成藥丸形」被 727 次 pill 推翻，六個母版本身違規。評審建議把規則從「用高度判斷」改成「用語意判斷」
- `line-height-still-latin-defaults` — 行高照抄 Tailwind 拉丁預設，Text 落在 1.429 低於文件自己的 1.6；Heading(18) 的 1.556 比 Text 更鬆是反的
- `type-line-height-missing` — 還要補 `Type/*/Weight` 變數與 `Text/*` 純文字母版
- `mobile-is-two-sentences` — 手機在 DESIGN.md 只有兩句話，但 PRODUCT.md 說手機與桌機同等重要
- `design-sop` — 沒有「怎麼新增一張畫面」的 SOP
- `drift-checker-blind-to-pen-and-non-color` — 檢查器看不到 `.pen`，字級／間距／圓角／陰影沒有守門
- `pen-screenshot-export-too-small` — 截圖 400px，設計系統自己的說明書最不可讀
- `code-still-uses-10px` — 程式碼那 6 處（另開 PR）

### P2（11 張）

`type-naming-unlearnable`（Headline 30 > Title 24 > Subtitle 20 > Heading 18，四個近義詞無法推導大小；且與 DESIGN.md 角色名撞名錯位）· `spacing-has-no-selection-rules` ＋ `spacing-scale-grouping`（**這兩張重複，該合併**）· `text-disabled-used-as-fourth-weight` · `accent-and-warning-are-the-same-gold` · `component-variant-naming` · `anatomy-page-role-legend` · `flow-k-activity-misfiled` · `pen-flow-inner-layers` · `pen-b4dwh-duplicate-node-id` · `design-language-v2-token-table`

---

## 工作方式（照這個做，不然會踩坑）

1. **`.pen` 只能透過 Pencil MCP 讀寫**（`mcp__pencil__execute`，filePath 是絕對路徑）。**不要用 Read 或 Grep**——那是 30 萬行 JSON。
2. **改完一定要存檔**：`osascript -e 'tell application "Pen" to activate' -e 'delay 1' -e 'tell application "System Events" to tell process "Pen" to click menu item "Save" of menu "File" of menu bar 1'`，然後確認 `git status --short ux-design.pen` 有 ` M`。
3. **新 Insert 的節點在存檔前不會被 `TakeScreenshot` 畫出來**（bounds 正確、fill 正確、就是空白）。存檔後再截。
4. **`lineHeight` 是比例不是 px**。填 20 代表 20 倍行高，裁切警告會從 150 暴增到 2164。
5. **截圖重產是非決定性的**——`git checkout` 掉沒有真的改變設計的那些 PNG，只 commit 真改動的。這次已經誤 commit 過一次重繪雜訊（11 張 byte 差、0 行變更），後來用 `--force-with-lease` 清掉。
6. **每一輪都用「截圖改變張數 == 預期被改到的畫面」當迴歸測試。** 圖層分層那輪是 0 張、補回 36px 那輪正好 5 張、元件命名那輪正好 2 張。數字對不上就是有東西被誤傷。
7. **`f24-d-v2` 的 `b4DwH` 是壞節點**（一個 text 與一個 frame 共用 id），所有批次腳本都要跳過。
8. **commit 與 PR 用白話中文**（見 CLAUDE.md），先講痛點與真實數字，技術細節放 `---` 之後。
9. **`gh` 帳號要是 `j620656786206`**。

---

## 驗證工具

- `python3 scripts/check-design-tokens.py` — 四份文件的色彩 token 漂移檢查（CI 也會跑）
- `npx vitest run apps/web/src/styles-contrast.spec.ts` — 142 個對比度守門測試
- `python3 scripts/export-pen-screenshots.py` — 重產 186 張截圖
- 全檔裁切警告基準值：**169**（`Get((n,c)=>c.problems&&...)`）。超過就是新的破版。

---

## 給新 session 的第一句話

> 接手 PR #410（分支 `chore/pen-design-tokens`）。先讀 `_bmad-output/session-handoff-2026-09-10-design-system.md`，再讀 `_bmad-output/implementation-artifacts/sprint-status.yaml` 裡 `disc-2026-09-` 開頭的 25 張未完成單子。這個 PR 只動設計稿與設計文件，不動 `apps/web`。從 P1 開始逐一裁定與執行，每一項先給我兩個選項與你的建議再動手。
