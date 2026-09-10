# Vido — Design Context Pack（給 Pencil App AI / Fable 5 的專案 primer）

> **這份文件的用途**：當你（Fable 5，在 Pencil App 內建 AI）要 **review 或修改整份 `ux-design.pen`** 時，你看不到這個專案的程式碼與 BMAD 規劃文件。把這份 primer 附給你，讓你一次理解 Vido 的產品、設計系統、畫布慣例與流程規範，做出**對齊專案脈絡**的判斷。
> 全程請以**繁體中文 (zh-TW)** 回應。

---

## 1. 產品一句話

**Vido** 是一套自架（家用 NAS）的**影音媒體管理 Web App** —— 掃描本地影片庫、抓取繁中中繼資料（TMDB / 豆瓣 / Wikipedia）、管理下載（qBittorrent）、搜尋與下載字幕、AI 字幕校正/翻譯，並提供豐富的瀏覽與詳情頁。

- **使用者**：自架媒體庫的中文（台灣）使用者，重視繁中介面、字幕品質、整齊的中繼資料。
- **介面語言**：**繁體中文 zh-TW 為預設**（簡體一律以官方 C++ OpenCC helper 的 `s2twp` 轉繁；遷移期間保留 Go backend fallback）。
- **平台**：響應式 Web App（桌面為主、手機可用）。**雙主題**：夜行（Nightwalk，深色，預設）／日巡（Daywalk，淺色，opt-in）。

---

## 2. 技術與介面約束（影響設計判斷）

| 項目 | 約束 |
|------|------|
| 前端 | React + TanStack Router、Tailwind CSS v4 |
| 主題 | **夜行（深色，預設）＋ 日巡（淺色）**。`:root` 就是夜行；淺色寫在 `[data-theme='light']`，深色**不寫任何 attribute**。色彩以 CSS 變數 token 表達（見 §3） |
| 語言/排版 | 繁中優先；**用全形標點「，。：」**，避免簡體殘留 |
| 響應式 | Tailwind 預設斷點：`sm 640 / md 768 / lg 1024 / xl 1280`；手機需單欄可讀、不橫向溢出 |
| 無障礙 | 深色底文字對比、觸控目標 ≥44px、可見焦點態、圖示/星等需文字替代（`aria-label`） |
| 動效 | 尊重 `prefers-reduced-motion` |

> ⚠️ **重要區分**：**深色主題只存在於「畫面框（frame / mockup）內部」**。畫布上**圍繞畫面的標註文字**（流程標題、畫面 caption）是 Pencil **淺色畫布**上的註解，用深色字（見 §5），不要把這兩者的配色搞混。

---

## 3. 設計系統 Token（取自實作程式碼 `apps/web/src/styles.css`，為地面真相）

設計稿中**畫面內部 UI 的所有顏色/間距，都應對齊這些 token**。任何一次性硬編碼色值 = **design-system drift**，review 時要揪出。

### 色彩

夜行（預設）／日巡（淺色）。**同一個 token 名稱、兩組值**；設計稿 `ux-design.pen` 的顏色變數名與這裡一字不差。

```
token                    夜行 Nightwalk        日巡 Daywalk
# 背景
--bg-primary            #0c1512              #faf6ea    
--bg-secondary          #132320              #f0e7d3    
--bg-tertiary           #1b302b              #e5d9c3    ⚠️ 日巡的最深階，所有文字對比的最壞情況
--border-subtle         #274039              #cdbe9b    
# 主色
--accent-primary        #c9a24b              #886208    泥金
--accent-hover          #e0be72              #725205    ⚠️ 日巡往「暗」走，不是變亮
--accent-pressed        #a8853c              #5b4103    
--accent-subtle         #c9a24b26            #c9a24b40  
--accent-tint           #c9a24b1f            #c9a24b33  
--accent-text           #e0be72              #654804    要被「讀」的金
--focus-ring            #c9a24b              #886208    
# 語意 · 青碧＝正在發生
--success               #6fbfa8              #0b7352    
--success-tint          #6fbfa81f            #6fbfa833  
--success-text          #8fd3be              #06583e    
# 語意 · 赭＝要求了但沒發生
--warning               #d4763f              #a6510c    
--warning-pressed       #d97706              #8b4208    
--warning-tint          #d4763f1f            #d4763f33  
--warning-text          #e8b04b              #7b3a06    
# 語意 · 硃砂＝壞掉了
--error                 #c0392b              #c0392b    唯一不隨主題翻轉
--error-pressed         #9c3a2b              #9c3a2b    
--error-tint            #c0392b1f            #c0392b33  
--error-text            #e08a76              #87251b    
# 語意 · 靛青＝純告知
--info                  #1391b2              #0b657d    
--info-tint             #1391b21f            #1391b233  
--info-text             #5bc4dd              #075469    
# 文字
--text-primary          #eae4d6              #16231d    
--text-secondary        #a8b3ac              #32493e    
--text-muted            #8fa096              #41554c    
--text-disabled         #5e6e66              #7a8980    
--text-inverse          #0c1512              #faf6ea    ⚠️ 不等於「暗底」，日巡是紙色
--text-on-accent        #14161a              #fdfaf2    
--text-on-scrim         #faf6ea              #faf6ea    兩主題相同
# 遮罩
--overlay-scrim         #000000b3            #0c1512b3  
```

**內文顏色規則**：`--accent-primary` / `--error` 是拿來**填色與按**的，當內文字對比不足（最差 3.57 / 2.42）。要被讀的文字一律用 `--accent-text` / `--error-text`。

### 圓角 / 陰影
```
--radius-sm 4 / md 8 / lg 12 / xl 16
--shadow-sm / md / lg / xl（深色用，黑色 30%→60% 遞增）
```

### 間距

程式碼走 Tailwind 4px 基準（4 / 8 / 12 / 16 / 24 / 32）。設計稿的階梯較長，因為要描述已經畫出來的版面，變數名為 `Space/*`：

```
2 hairline · 4 xs · 6 xs-plus · 8 sm · 10 sm-plus · 12 md · 14 md-plus · 16 lg
20 lg-plus · 24 xl · 28 xl-plus · 32 2xl · 40 3xl · 48 4xl · 64 5xl · 80 6xl
```

半階（6 / 10 / 14 / 28）對應 Tailwind 的 `.5` 級距，不是隨手畫的數字。**不要在設計稿裡寫階梯以外的數字。**

### 字型與字級
- **介面內文字**：`Noto Sans TC`（設計稿變數 `Type/Family/Primary`）。
- **等寬/技術數值**：`JetBrains Mono`（`Type/Family/Mono`）——檔名、編碼、解析度、任何會被跨列比較的數字。
- **畫布大標題（流程標題註解）**：`DM Sans`（`Type/Family/Canvas`，見 §5）。**只用於畫布註記，不是產品字體。**

字級變數（`Type/*/Size`，px）：

```
Display L 36 · Display M 32 · Title XL 28 · Title L 24 · Title M 22 · Title S 20
Heading L 18 · Heading M 16 · Heading S 15
Body L 14（預設內文）· Body M 13 · Body S 12（標籤）
Meta M 11（殼層 chrome）· Meta S 10
```

**硬界線**：11px 只給「不是內容的東西」（分頁列、側軌群組標題）。使用者要讀的字不得低於 12px；可點擊的目的地或動作是 14px。

---

## 4. 設計原則（Vido 的「好」長這樣）

1. **資訊密度但不雜亂**：媒體庫類產品天生資訊多 —— 用層級（標題 > 中繼資訊 > 次要）和留白把密度組織好，而非塞滿。
2. **海報優先**：PosterCard 是瀏覽主角，圖像清晰、fallback（無圖時的 ColorPlaceholder）要優雅。
3. **狀態完整**：每個資料畫面都要涵蓋 **空 / 載入(skeleton) / 錯誤 / 無資料** 四態，不能只畫「快樂路徑」。
4. **外部資料 fail-soft**：TMDB/豆瓣/字幕等外部區塊載入失敗時，**該區塊優雅省略或顯示空態，頁面其餘照常**（per-section 隔離），絕不整頁壞掉。
5. **繁中體驗**：全形標點、合理行長、繁體用字。

---

## 5. 畫布 IA 慣例（`ux-design.pen` 的整理規範 —— 改/評時務必遵守）

**① 以「使用者流程」為單位合併**：不再按版面分。同一流程的**桌面 + 手機**畫面放進**同一個區塊**：上排桌面、下排手機，**相同步驟左右對齊**；某平台獨有的步驟各佔自己的欄。

**② 命名 = 流程碼＋序號＋語意＋平台**
- 畫布可見「語意標題」（caption text node，**會進截圖**）：`B3 · 詳情面板（桌面）` / `B3 · 詳情面板（手機）`，樣式 **Noto Sans TC 14 / 600 / `#888888`**。
- 「圖框名稱」用精簡短碼：`B3-D`（桌面）、`B3-M`（手機）。

**③ 標題在畫面上方、不重疊**：caption 放畫面上方會撞到 Pencil 圖框名 chrome，需拉開 ~45px。

**④ 座標範本**（合併欄 x=17040，各流程往下堆疊、間距 2600）
- 流程標題 `(blockX, blockY)`：**DM Sans 24 / 700 / `#222222`**「中文 — English」；描述 `(blockX, blockY+34)`：Noto Sans TC 14 `#666666`。
- 桌面 frame `y=blockY+120`(h≈900)、caption `y=blockY+75`；欄 `x=blockX+col*1540`。
- 手機 frame `y=blockY+1130`(h≈844)、caption `y=blockY+1085`；x 對齊同步驟的桌面欄。
- root 畫面用**絕對 x/y 定位** → 移動時用 **Update 改 x/y/name（不要 Move）**；絕不改畫面內部 UI、不動 components。

**⑤ spec / 設計決策畫面要「獨立成自己的框」**，不可塞進既有 mockup。

---

## 6. 完整流程清單（review 整份 .pen 的覆蓋範圍）

| 碼 | 流程 | 涵蓋畫面（重點） |
|----|------|------------------|
| **A** | 瀏覽主流程 | 空 / 載入 / 網格 / 列表 / 排序 / 篩選 |
| **B** | 詳情與互動 | Hover / 右鍵選單 / 詳情（電影·影集）/ Fallback / 技術徽章 / 圖片載入 fallback (B9) |
| **C** | 搜尋·篩選·設定 | 搜尋+篩選 / 批次操作 / 設定 / 備份 |
| **D** | 下載管理 | 下載監控與管理 |
| **E** | 媒體庫掃描 | 掃描設定 / 進度 / 完成 toast / 未匹配過濾 |
| **F** | 字幕搜尋 | 字幕搜尋對話框 / 預覽下載 / 批次進度 |
| **G** | AI 字幕增強 | AI 校正 / 轉錄進度 / 翻譯確認 |
| **H** | 首頁 TV Wall | 首頁牆 / 載入骨架 / 區塊 CRUD modal / ExploreBlock |
| **I** | 進階搜尋篩選 | 篩選 chips / 建議下拉 / 儲存預設 / 篩選 sheet |
| **J** | 規格畫面 | 設計決策 spec 畫面（如 PosterCard 資訊密度） |
| **design-system** | 設計系統參考 | Design System Reference + Component Library 文件 |

> 截圖對照位於 `_bmad-output/screenshots/<flow-folder>/`（每流程一夾，含 `-d` 桌面 / `-m` 手機）。

---

## 7. BMAD 開發框架的設計相關規範（為什麼某些畫面「故意」缺/獨立）

此專案以 **BMAD（BMM / DSDD）** 多代理流程開發，下列規範會影響你對 .pen 的判讀：

- **Rule 21 — 元件↔設計覆蓋**：每個前端元件都要對應一個 `.pen` 畫面；若**功能比 `.pen` 設計還晚出現**（design-coverage-gap），該元件會帶一行 `// Design ref: ux-design.pen — no current screen frame …` 註解。代表 **.pen 可能「落後」於部分已上線功能**（例：Epic 12 詳情頁的豆瓣評論區 `DoubanSection` 目前無對應畫面）。review 時可標記「.pen 缺此畫面」為**待補**，而非錯誤。
- **spec 畫面獨立成框**：設計決策/規格註解一律自成一個畫面，不塞進 mockup（流程 J 與各處 spec）。
- **標題不重疊**：見 §5③。
- **截圖工作流（關鍵整合點）**：**任何** `.pen` 變更後，都要回 repo 跑 `python3 scripts/export-pen-screenshots.py` 重產截圖，且**只 commit 設計真正變動的 PNG**（全量重產是非確定性的，每張都有 byte 差）。**Pencil App 不會自動做這步** —— 改完請提醒使用者。
- **雙語文件**：使用者面向文件需 EN + zh-TW 兩版（與畫面標註的「中文 — English」格式呼應）。

---

## 8. 整份 .pen Review 的優先準則（建議的檢查清單）

逐**畫面框 (frame)** 檢查，並做跨畫面的一致性掃描：

1. **設計系統一致性**：色/間距/圓角/字級是否全對齊 §3 token；列出所有 drift。
2. **視覺層級與資訊密度**：標題/中繼資訊/次要內容的對比與權重。
3. **8pt 間距節奏**：跨畫面留白是否一致。
4. **可操作性 affordance**：主/次動作層級、外連連結、可點區塊是否看得出可點。
5. **無障礙**：深色底對比、觸控 ≥44px、焦點態、圖示/星等文字替代。
6. **響應式一致**：每個流程的 `-D` 與 `-M` 是否同步、手機單欄可讀。
7. **繁中排版**：全形標點、行長、無簡體殘留。
8. **狀態完整性**：空 / 載入 / 錯誤 / 無資料 四態是否齊全（尤其外部資料區塊的 fail-soft 空態）。
9. **流程連貫性**：A–J 同流程內步驟順序與導覽是否合理、對齊對應的使用者旅程。
10. **覆蓋缺口**：對照 §6 清單，標出 `.pen` 尚缺、或落後於已上線功能的畫面（design-coverage-gap）。

### 建議輸出格式
- **主表**：`| 畫面碼(如 B3-D) | 流程 | 問題 | 嚴重度(高/中/低) | 具體修法 |`
- **設計系統偏移清單**：所有偏離 token 的硬編碼樣式。
- **覆蓋缺口清單**：缺/落後的畫面。
- 先**只給報告、不動畫布**；待使用者確認後，再依「變更提示詞」動手。

---

## 9. 正典文件索引（若使用者願意額外附上，可加深理解）

| 文件 | 路徑 | 內容 |
|------|------|------|
| UX 主規格 | `_bmad-output/planning-artifacts/ux-design-specification.md` | UX 設計規格 |
| 媒體庫設計簡報 | `_bmad-output/planning-artifacts/epic5-media-library-design-brief.md` | look & feel / 設計系統 |
| UX 缺口分析 | `_bmad-output/planning-artifacts/ux-design-gap-analysis-v4.md` | 設計覆蓋缺口 |
| 畫布慣例（原始） | `.claude/memory/project_pen_flow_layout_convention.md` | A–J 流程/命名/座標 |
| 規則聖經 | `project-context.md`（根目錄） | 含 Rule 21 元件↔設計覆蓋等 |
| 使用者旅程 | `_bmad-output/planning-artifacts/prd/user-journeys.md` | 產品旅程 |
| Web 需求 | `_bmad-output/planning-artifacts/prd/web-application-specific-requirements.md` | Web 專屬需求 |
| 截圖工作流 | `CLAUDE.md`（UX Design Screenshots Workflow 段） | A–J 流程分類與重產流程 |
| 設計系統截圖 | `_bmad-output/screenshots/design-system/` | Design System Reference + 元件庫 |

---

*Generated for the Pencil App Fable 5 agent · source of truth: `apps/web/src/styles.css`、`CLAUDE.md`、`.claude/memory/project_pen_flow_layout_convention.md`. Token 值若與當前 `styles.css` 不符，以程式碼為準。*
