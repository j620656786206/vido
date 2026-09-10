# 「依政策執行」設計定稿 + Pencil Inline AI Agent 提示詞（sub-6-ux-consent-policy）

**核定者：** Sally（UX Designer）　**日期：** 2026-09-10
**Brief：** `_bmad-output/planning-artifacts/consent-policy-brief-2026-09-03.md`（Alexyu 2026-09-04 已確認）
**新增畫面：** F21–F27（flow-f-subtitle-v2）＋ J8-D（flow-j-specs），共 10 張
**執行方式：** Alexyu 在 Pencil 跑 Inline AI Agent → Claude 以 MCP 唯讀複審 → 依 `CLAUDE.md`
重出截圖同 commit（只 stage 真的改到的 PNG）

---

## 〇、執行順序（Alexyu 貼提示詞用的清單）

每一段提示詞是**獨立一次**貼給 Pencil Inline AI Agent。**順序不能亂**：D 建立的畫面是 E、F 的來源。

| # | 提示詞 | 畫面 | 相依 | 貼之前要先做的事 |
| --- | --- | --- | --- | --- |
| 1 | A | F22-D-v2 入口．活動頁（桌機） | — | — |
| 2 | B | F26-D-v2 入口．政策未設定 | — | — |
| 3 | C | F27-D-v2 入口．掃描完成卡 | — | — |
| 4 | **D** | **F21-D-v2 政策確認屏（桌機）** | — | 這張是焦點，跑完先看一眼再往下 |
| 5 | E | F23-D-v2 超限 | **需要 D** | 確認 `f21-d-v2` 已存在 |
| 6 | F | F24-D-v2 無可抽取項目 | **需要 D** | 同上 |
| 7 | G | F25-D-v2 設定頁政策格 | — | — |
| 8 | H | F22-M-v2 入口．活動頁（手機） | — | — |
| 9 | I | F21-M-v2 政策確認屏（手機） | — | — |
| 10 | J | J8-D 決策 spec | — | — |

**十段都跑完之後：** 在 Pencil 按 ⌘S 存檔，然後告訴我一聲 —— 我會用 MCP 逐張複審、
更新 export script、重出截圖、開 PR。**先不要自己跑 `export-pen-screenshots.py`。**

---

## 一、為什麼要有這批稿

正式片庫 2,399 個候選、預設選取 696 部＝ $13.92，已經超過 $5 的出廠上限。今天的流程要求
使用者在一面 2,399 列的牆上逐項決定要不要花錢 —— 這正面違反 `PRODUCT.md` 原則 2「移除操作者」。
政策的做法是：**一條規則 + 一顆按鈕 + 一份可勾掉的清單**。規則是事前同意，執行不需要操作者，
清單退為審計與例外排除。

三個已裁定的邊界（brief §3）在每一張稿上都必須看得到：

1. **政策只自動跑抽取路線**（僅翻譯費）。語音辨識永遠走現有 F15 手動同意，稿上要說出來。
2. **手動按「依政策執行」**。沒有排程器，稿上不得出現「每天自動」「排程」這類字。
3. **清單＝審計＋例外排除**。預設收合成群組，沒動就照跑。

---

## 二、九條裁定

### 裁定 1 — 入口是「主／次」，不是「新／舊」

活動頁標頭與 F17 掃描完成卡上，「依政策執行」是主按鈕，「批次生成字幕」降為次按鈕。
**次按鈕不得消失**：語音辨識、自選片單只能走它，把它藏起來等於把 ASR 路線變成死路。

### 裁定 2 — 政策未設定時，主按鈕說的是下一步，不是錯誤

主按鈕變「設定政策」（仍是主按鈕樣式），旁邊一行 `$text-muted` 說「尚未設定政策」。
不畫成停用態、不畫紅色 —— 沒設定不是錯誤，是還沒開始。

### 裁定 3 — 焦點時刻是「一句話 ＋ 金額」，清單在它下面

F21 頂部固定三行，順序不可換：

1. `將為 46 部可抽取項目產生字幕，約 $1.84（上限 $5.00）` — 18px/600，金額 JetBrains Mono
2. `只處理可抽取內嵌字幕的項目（僅翻譯費）。需要語音辨識的 96 部不在這次範圍。` — 13px `$text-secondary`
3. `使用：Claude（你的金鑰）` — 12px `$text-muted`

第 3 行**只有 Claude 那一半**。政策不跑 ASR，把 OpenAI 也印上去等於承諾一件不會發生的事
（F15 的來源行有兩半，是因為 F15 真的會跑 ASR）。

### 裁定 4 — 清單預設全部收合

F21 的清單只有群組列，沒有展開的影片列。這是政策屏與 F15 最大的差別：F15 要你逐項決定，
F21 只要你確認總量。展開是使用者主動的動作，不是預設。

### 裁定 5 — 「已排除」是一個段落，不是刪除

勾掉的項目移到清單最下方的「已排除（N）」段，每列右邊一個「復原」文字按鈕（`rotate-ccw`）。
列**淡化到 0.6**、不畫成停用樣式 —— 排除是可逆的決定，不是失效的資料。

### 裁定 6 — 超限沿用 sub-6-12 的砍線，一個字都不新造

橫幅文案、砍線列文案、線後列 `opacity: 0.6` 全部沿用 F18（`zBik1`）。政策屏不發明第二套超支語彙。

### 裁定 7 — 空狀態不說謊，而且要說出「不在範圍」

政策的空狀態與 F20 的空狀態**不是同一件事**：F20 是「沒有可產生的項目」，
F21 的空是「有候選，但全部都需要語音辨識，政策不處理」。所以文案是
`沒有可以自動處理的項目`，副句指名去哪裡處理，圖示用 `search-x` `$text-muted`（不是綠勾）。

### 裁定 8 — 政策格放在「字幕設定」

`/settings/subtitle`（字幕設定）新增一格「自動處理政策」，排在「在地化程度」上方。
理由：政策持有的兩個旋鈕（上限金額、翻譯模型）都是「AI 字幕怎麼做出來的」，與在地化程度同源；
「媒體庫掃描」持有的是每個媒體庫的資料夾開關，不是全站花費。
**同時要改**該頁副標，它現在寫「媒體庫的自動字幕請到『媒體庫掃描』」，政策落地後這句會誤導。

模型列沿用 sub-6-8b `ModelPicker` 的詞彙（名稱／（預設）／品質 A／時間），但金額改成
**每集約 $0.02** —— 設定頁沒有「這批」，印「這批約 $X」會是假的。

### 裁定 9 — 三個未決在 J8-D 兩案並陳，稿上不偷偷裁

(a) 政策上限與 `AI_RUN_BUDGET_USD`、(b)「已排除」是否持久化、(c) 是否顯示「上次執行花了多少」。
F21 稿面採用的畫法是**其中一案的樣子**，J8-D 必須寫明它是「暫用」而非裁定。

---

## 三、畫面清單

| 代碼 | 圖框名稱 | 位置 (x, y) | 尺寸 | 內容 |
| --- | --- | --- | --- | --- |
| F22-D-v2 | `f22-d-v2` | 66270, -5746 | 1440×900 | 入口．活動頁（桌機），政策已設定 |
| F26-D-v2 | `f26-d-v2` | 67870, -5746 | 1160×140 | 入口．政策未設定（標頭條，獨立規格框） |
| F27-D-v2 | `f27-d-v2` | 69470, -5746 | 480×175 | 入口．掃描完成卡（F17 的政策變體） |
| F21-D-v2 | `f21-d-v2` | 71070, -5746 | 1440×900 | 政策確認屏（桌機） |
| F23-D-v2 | `f23-d-v2` | 72670, -5746 | 1440×900 | 政策確認屏．超限 |
| F24-D-v2 | `f24-d-v2` | 74270, -5746 | 1440×900 | 政策確認屏．無可抽取項目 |
| F25-D-v2 | `f25-d-v2` | 75870, -5746 | 768×470 | 設定頁「自動處理政策」格 |
| F22-M-v2 | `f22-m-v2` | 66270, -4671 | 390×844 | 入口．活動頁（手機） |
| F21-M-v2 | `f21-m-v2` | 71070, -4671 | 390×844 | 政策確認屏（手機 bottom sheet） |
| J8-D | `J8-D` | 25080, 24300 | 1240×~1500 | 政策與手動流程分工 + 三個未決兩案並陳 |

每張桌機框上方 45px 放語意標題（caption），手機框同理；J8-D 的 caption 放在 (25080, 24270)。
caption 樣式：Noto Sans TC 14/600 `#888888`。

| caption 節點名稱 | 位置 | 文字 |
| --- | --- | --- |
| `cap-F22-D-v2` | 66270, -5791 | `F22 · 入口．活動頁（桌機）` |
| `cap-F26-D-v2` | 67870, -5791 | `F26 · 入口．政策未設定（桌機）` |
| `cap-F27-D-v2` | 69470, -5791 | `F27 · 入口．掃描完成卡（桌機）` |
| `cap-F21-D-v2` | 71070, -5791 | `F21 · 依政策執行．確認屏（桌機）` |
| `cap-F23-D-v2` | 72670, -5791 | `F23 · 政策確認屏．超限（桌機）` |
| `cap-F24-D-v2` | 74270, -5791 | `F24 · 政策確認屏．無可抽取項目（桌機）` |
| `cap-F25-D-v2` | 75870, -5791 | `F25 · 設定頁「自動處理政策」格` |
| `cap-F22-M-v2` | 66270, -4716 | `F22 · 入口．活動頁（手機）` |
| `cap-F21-M-v2` | 71070, -4716 | `F21 · 依政策執行．確認屏（手機）` |
| `Caption J8-D` | 25080, 24270 | `J8 · 政策與手動流程分工` |

---

## 四、共用文案與數字（各稿之間不得漂移）

數字沿用 F15 現有樣本，算式對得起來，不是編的：

| 名稱 | 值 | 來源 |
| --- | --- | --- |
| 候選總數 | 142 | F15 `summary-text` |
| 可抽取（政策範圍） | 46 部 · $1.84 | F15 `已選 46 部 · $1.84` |
| 需語音辨識（不在範圍） | 96 部 | 142 − 46 |
| 政策上限 | $5.00 | F15 `budget-value`／`AI_RUN_BUDGET_USD` 出廠值 |
| 已匹配電影群組 | 已選 22/24 · $0.88 | 22 × $0.04 |
| 怪奇物語 · 第 4 季 | 已選 24/24 · $0.96 | 24 × $0.04 |
| 已排除 | 2（星際效應、教父） | 24 + 24 − 46 = 2 |
| 超限樣本（F23） | 696 部 · $13.92，上限 $5.00，約 250 部後暫停 | brief §2；$5.00 ÷ $0.02 |

固定字串（逐字複製）：

- 主按鈕（已設定）：`依政策執行`
- 主按鈕（未設定）：`設定政策`
- 次按鈕：`批次生成字幕`
- 確認屏主按鈕：`開始`
- 來源行：`使用：Claude（你的金鑰）`
- 政策說明行（設定頁與確認屏共用語意，措辭略有不同，見各提示詞）
- 超限橫幅：`預估 $13.92 已超過上限 $5.00 —— 預計可完成約 250 部後暫停（清單中已標示），其餘保留在佇列，可提高上限或稍後續跑。`
- 砍線列：`到此為止約 $5.00，之後的項目會暫停`

---

## 五、執行前必讀的三個 schema 陷阱

1. 對齊值只有 `start` / `center` / `end`（另有 `space_between` / `space_around`）。**沒有 `stretch`**。
   要子元素撐滿，靠子元素自己的 `width: "fill_container"`。
2. `width: "fill_container"` 的 `text` 節點**必須**同時設 `textGrowth: "fixed-width"`，否則長中文會爆出容器。
3. **不要在元件實例（`ref`）裡插入子節點** —— Pencil 不允許。按鈕一律不加圖示；
   需要圖示時在按鈕**外面**包一層 frame 裝 icon + 文字（sub-6-11 已踩過）。

顏色只用既有變數：`$accent-primary` `$accent-text` `$bg-primary` `$bg-secondary` `$bg-tertiary`
`$border-subtle` `$text-primary` `$text-secondary` `$text-muted` `$success` `$success-tint`
`$warning` `$warning-tint` `$error` `$error-text` `$error-tint`。
**沒有** `text-tertiary`、`success-text`、`warning-text` 這些名字。

---

## 提示詞 A — F22-D-v2（入口．活動頁，桌機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡複製畫面 `pwMzT`（`f15-d-v2`）到座標 x=66270、y=-5746，
新畫面命名為 `f22-d-v2`。

複製會產生全新的節點 ID，所以以下都**用名稱**在新畫面裡尋找節點。

1. 刪除新畫面裡名為 `scrim` 的 rectangle，以及名為 `gen-dialog` 的 frame。
   留下的 `backdrop` 就是活動頁本身。

2. 找到 `header`（裡面有 `pageTitle`「活動」與 `pageSub`）。把它改成左右兩欄：
   - 先把 `pageTitle` 與 `pageSub` 一起包進一個新的垂直 frame，名稱 `title-col`，
     `layout: "vertical"`、`gap: 4`、`width: "fill_container"`。
   - 把 `header` 設成 `layout: "horizontal"`、`alignItems: "start"`、
     `justifyContent: "space_between"`、`gap: 16`。
   - 因為 `pageTitle` / `pageSub` 原本是 `width: "fill_container"`，包進 `title-col` 後保持不變即可。

3. 在 `header` 裡、`title-col` 之後新增一個水平 frame，名稱 `header-actions`，
   `gap: 10`、`alignItems: "center"`：

   - 第一顆（主）：`{ type: "ref", ref: "otvKh", name: "btn-policy-run", height: 44,
     descendants: { "DaLcd": { content: "依政策執行", fontFamily: "Noto Sans TC" } } }`
   - 第二顆（次）：`{ type: "ref", ref: "YDPhc", name: "btn-batch-manual", height: 44,
     descendants: { "L9cIf": { content: "批次生成字幕", fontFamily: "Noto Sans TC" } } }`

   兩顆按鈕都**不加圖示**。

4. 在 `header-actions` 下方、仍在 `header` 內不行（`header` 現在是水平的），
   所以改放在 `title-col` 裡、`pageSub` 之後，新增一個文字節點：

   ```
   name: "policy-line"
   type: "text"
   content: "政策：只自動處理可抽取的項目 · 上限 $5.00 · Claude Sonnet 4.5"
   fontFamily: "Noto Sans TC"
   fontSize: 12
   fill: "$text-muted"
   width: "fill_container"
   textGrowth: "fixed-width"
   ```

5. 在畫面上方 45px 處（x=66270、y=-5791）新增 caption 文字節點，名稱 `cap-F22-D-v2`，
   內容 `F22 · 入口．活動頁（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。
   這個節點放在畫布根層，不是畫面裡面。

> 提示詞 A 結束。

---

## 提示詞 B — F26-D-v2（入口．政策未設定，獨立標頭條）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的畫布根層新增一個 frame，座標 x=67870、y=-5746，
名稱 `f26-d-v2`，`width: 1160`、`height: 140`、`fill: "$bg-primary"`、
`layout: "vertical"`、`padding: [24, 24]`、`gap: 12`、`clip: true`。

裡面依序放：

1. 水平 frame `head-row`，`width: "fill_container"`、`alignItems: "start"`、
   `justifyContent: "space_between"`、`gap: 16`：
   - 垂直 frame `title-col`（`gap: 4`、`width: "fill_container"`）：
     - text `pageTitle`：`活動`，Noto Sans TC 28/700，`$text-primary`
     - text `policy-empty`：`尚未設定政策`，Noto Sans TC 12，`$text-muted`，
       `width: "fill_container"`、`textGrowth: "fixed-width"`
   - 水平 frame `header-actions`（`gap: 10`、`alignItems: "center"`）：
     - `{ type: "ref", ref: "otvKh", name: "btn-policy-setup", height: 44,
       descendants: { "DaLcd": { content: "設定政策", fontFamily: "Noto Sans TC" } } }`
     - `{ type: "ref", ref: "YDPhc", name: "btn-batch-manual", height: 44,
       descendants: { "L9cIf": { content: "批次生成字幕", fontFamily: "Noto Sans TC" } } }`

2. text `spec-note`：
   `政策未設定＝還沒開始，不是錯誤：主按鈕仍是主按鈕樣式，不畫停用態、不畫紅色。`
   Noto Sans TC 12，`$text-secondary`，`width: "fill_container"`、`textGrowth: "fixed-width"`。

最後在 x=67870、y=-5791 新增畫布根層 caption 文字節點 `cap-F26-D-v2`，
內容 `F26 · 入口．政策未設定（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 B 結束。

---

## 提示詞 C — F27-D-v2（入口．掃描完成卡）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡複製節點 `fL8DD`（`f17-d-v2` 裡的 `scan-complete-toast`）到畫布根層，
座標 x=69470、y=-5746，新節點命名為 `f27-d-v2`。

在複製出來的節點裡（用名稱尋找，ID 會是新的）：

1. `toast-actions` 目前有三個文字節點：`actionUnmatched`（查看未比對項目）、
   `actionErrors`（查看錯誤）、`actionGenSubs`（產生字幕 →）。
   請把 `actionGenSubs` **移到 `toast-actions` 的第一個位置**（index 0），
   並把它的內容改成 `依政策執行 →`、`fontWeight: "600"`、`fill: "$accent-text"`。

2. 在它後面（index 1）新增一個文字節點：

   ```
   name: "actionManual"
   type: "text"
   content: "批次生成字幕 →"
   fontFamily: "Noto Sans TC"
   fontSize: 13
   fontWeight: "normal"
   fill: "$text-secondary"
   ```

   `actionUnmatched` 與 `actionErrors` 保持原樣，順序排在它們後面。

3. `missingSubsLine` 的內容從 `142 部影片缺繁中字幕` 改成
   `142 部影片缺繁中字幕 · 其中 46 部可自動處理`。

4. 卡片高度是自動的，改完後確認沒有內容被裁切；若 `toast-actions` 四個項目在 448 寬度下擠不下，
   把 `toast-actions` 設成 `wrap: true`（或等效的換行設定），**不要**縮字級、不要刪掉任何一個動作。

最後在 x=69470、y=-5791 新增畫布根層 caption 文字節點 `cap-F27-D-v2`，
內容 `F27 · 入口．掃描完成卡（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 C 結束。

---

## 提示詞 D — F21-D-v2（政策確認屏，桌機）★ 本輪的焦點畫面

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡複製畫面 `pwMzT`（`f15-d-v2`）到座標 x=71070、y=-5746，
新畫面命名為 `f21-d-v2`。以下都在**複製出來的新畫面裡**用名稱尋找節點。

**1. 標題列。** `dialog-title` 的內容從 `產生字幕` 改成 `依政策執行`。

**2. 頂部三行取代摘要條。** 把 `summary-block` 的兩個子節點（`summary-bar`、`spend-source`）
整個換掉，`summary-block` 本身保留（`layout: "vertical"`、`gap: 2`、`width: "fill_container"`）。
改成 `gap: 6`，並依序放三個節點：

- 水平 frame `headline`（`gap: 6`、`alignItems: "baseline"`、`width: "fill_container"`）：
  - text `headline-a`：`將為`，Noto Sans TC 18/600，`$text-primary`
  - text `headline-n`：`46`，JetBrains Mono 18/600，`$text-primary`
  - text `headline-b`：`部可抽取項目產生字幕，約`，Noto Sans TC 18/600，`$text-primary`
  - text `headline-usd`：`$1.84`，JetBrains Mono 18/600，`$accent-text`
  - text `headline-cap`：`（上限 $5.00）`，Noto Sans TC 14，`$text-muted`
- text `policy-scope`：
  `只處理可抽取內嵌字幕的項目（僅翻譯費）。需要語音辨識的 96 部不在這次範圍。`
  Noto Sans TC 13，`$text-secondary`，`width: "fill_container"`、`textGrowth: "fixed-width"`
- text `spend-source`：`使用：Claude（你的金鑰）`
  Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`

**3. 刪除不屬於政策屏的控制項。** 刪除 `search-sort-row`、`route-chips`、`toolbar`
（政策屏不是逐項操作的清單，路線 chip 更是錯的 —— 政策只有一條路線）。

**4. 清單改成全部收合。** `candidate-list` 目前有 8 個子節點，全部刪除，改放以下 5 個。
群組列請沿用既有 `group-header-怪奇物語S4` 的樣式：
`fill: "$bg-tertiary"`、`cornerRadius: "$radius-md"`、`gap: 10`、`padding: [8, 12]`、
`alignItems: "center"`、`width: "fill_container"`。

- (a) 群組列 `group-已匹配電影`：
  - `{ type: "ref", ref: "4EHFN", name: "cb" }`（全選勾選框）
  - icon `disclosure`：`chevron-right`，lucide，16×16，`$text-muted`
  - text `group-title`：`已匹配電影`，Noto Sans TC 14/600，`$text-primary`，
    `width: "fill_container"`、`textGrowth: "fixed-width"`
  - 水平 frame `right`（`gap: 8`、`alignItems: "center"`）：
    - frame `route-badge`（`fill: "$success-tint"`、`cornerRadius: "$radius-sm"`、`padding: [2, 6]`）
      內含 text `badge-lbl`：`僅翻譯費 24`，Noto Sans TC 11，`$success`
    - text `group-selected`：`已選 22/24 · $0.88`，JetBrains Mono 12，`$text-muted`

- (b) 群組列 `group-怪奇物語S4`：同樣式，`group-title` 為 `怪奇物語 · 第 4 季`，
  badge 為 `僅翻譯費 24`，`group-selected` 為 `已選 24/24 · $0.96`，
  勾選框用 `{ type: "ref", ref: "4EHFN", name: "cb" }`（全選）。

- (c) 分隔文字 `excluded-header`：水平 frame（`gap: 8`、`alignItems: "center"`、
  `width: "fill_container"`、`padding: [8, 0, 0, 0]`）內含：
  - icon `disclosure`：`chevron-down`，lucide，16×16，`$text-muted`
  - text `excluded-title`：`已排除（2）`，Noto Sans TC 13/600，`$text-secondary`，
    `width: "fill_container"`、`textGrowth: "fixed-width"`
  - text `excluded-note`：`這次不會處理，可復原`，Noto Sans TC 12，`$text-muted`

- (d)(e) 兩列被排除的影片。請**複製既有的 `row-星際效應`（節點 `hxctk`）兩次**放進來，
  然後分別改成：
  - 第一列命名 `row-excluded-星際效應`：`title` = `星際效應`、
    `sub` = `內嵌英文字幕 → 翻譯 · 2 小時 49 分`
  - 第二列命名 `row-excluded-教父`：`title` = `教父`、
    `sub` = `內嵌英文字幕 → 翻譯 · 2 小時 55 分`

  兩列都做以下四件事：
  - 整列 `opacity: 0.6`
  - 勾選框的 `ref` 改成 `Wd9AL`（未勾選樣式）
  - `right` 裡的路線 badge 與金額**保留**（讀者仍要知道排除掉的是多少錢）
  - 在 `right` 的最後新增一個水平 frame `restore-btn`
    （`gap: 4`、`alignItems: "center"`、`cornerRadius: "$radius-sm"`、`padding: [4, 8]`、
    `fill: "$bg-tertiary"`）：icon `rotate-ccw`（lucide，14×14，`$accent-text`）
    ＋ text `restore-lbl`：`復原`，Noto Sans TC 12/600，`$accent-text`

**5. 底部說明。** `estimate-note` 保留原文。`group-note` 的內容改成：
`清單預設全部收合 —— 政策屏只要你確認總量，展開是主動的動作。`

**6. 頁尾。** 在 `footer` 裡：
- `ft-sel-text` 改成 `已選 46 部 · 預估`，`ft-sel-amount` 保持 `$1.84`
- `ft-breakdown` 改成 `抽取 46 部 $1.84 · 已排除 2 部`
- `budget-label` 改成 `政策上限`，`budget-value` 保持 `$5.00`
- `budget-hint` 改成 `達到上限會自動暫停，可稍後續跑`（不變）
- `btn-start` 的 label 從 `開始產生` 改成 `開始`

**7. caption。** 在 x=71070、y=-5791 新增畫布根層文字節點 `cap-F21-D-v2`，
內容 `F21 · 依政策執行．確認屏（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。

改完後請確認對話框內沒有任何節點超出容器（清單變短了，對話框可能需要縮高；
若 `gen-dialog` 明顯留白過多，把它的高度改成自動或縮到內容高度，並讓它在畫面中垂直置中）。

> 提示詞 D 結束。

---

## 提示詞 E — F23-D-v2（政策確認屏．超限）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

**這個提示詞必須在提示詞 D 完成之後才執行。**

複製提示詞 D 做出來的 `f21-d-v2` 到座標 x=72670、y=-5746，命名 `f23-d-v2`。
在複製出來的畫面裡（用名稱尋找）做以下修改：

**1. 頂部數字換成超限樣本。**
- `headline-n` → `696`
- `headline-usd` → `$13.92`，`fill` 改成 `$warning`
- `headline-cap` → `（上限 $5.00）`（不變）
- `policy-scope` → `只處理可抽取內嵌字幕的項目（僅翻譯費）。需要語音辨識的 1,703 部不在這次範圍。`

**2. 群組列換成超限樣本。**
- `group-已匹配電影`：badge `僅翻譯費 412`、`group-selected` `已選 412/412 · $8.24`
- `group-怪奇物語S4` 改名為 `group-影集`，`group-title` 改成 `影集（12 部）`，
  badge `僅翻譯費 284`、`group-selected` `已選 284/284 · $5.68`
- 兩個群組的 `disclosure` 都改成 `chevron-down`（展開），因為砍線必須落在看得見的列之間。

**3. 在 `group-已匹配電影` 之後插入三列影片與一條砍線。**
請複製既有的 `row-沙丘：第二部`（節點 `u4sSoJ`）三次，分別命名
`row-沙丘：第二部`、`row-奧本海默`、`row-全面啟動`，`title` 依名稱設定，
`sub` 分別為 `內嵌英文字幕 → 翻譯 · 2 小時 46 分`、`內嵌英文字幕 → 翻譯 · 3 小時 0 分`、
`內嵌英文字幕 → 翻譯 · 2 小時 28 分`，金額都改成 `$0.02`。

然後在 `row-全面啟動` **之後**插入砍線（沿用 F18 `zBik1` 裡砍線的樣式，
若無法直接複製就照下列規格新建）：

```
name: "budget-cut"
type: "frame"（水平）
width: "fill_container"
fill: "$warning-tint"
cornerRadius: "$radius-sm"
padding: [6, 12]
gap: 8
alignItems: "center"
```
內含 icon `circle-alert`（lucide，14×14，`$warning`）
＋ text `cut-lbl`：`到此為止約 $5.00，之後的項目會暫停`，Noto Sans TC 12，`$warning`。

砍線之後再複製兩列（`row-教父`、`row-星際效應`，`sub` 沿用 F21 的寫法，金額 `$0.02`），
這兩列整列 `opacity: 0.6`，**勾選狀態不動**（它們仍然是已同意，只是會被暫停）。

**4. 加超限橫幅。** 在 `footer` 之前、`body` 之後（也就是與頁尾同一區、捲動區之外）
插入一個水平 frame `limit-warning`：
`fill: "$warning-tint"`、`cornerRadius: "$radius-md"`、`padding: 12`、`gap: 10`、
`alignItems: "center"`、`width: "fill_container"`，左右外距與 `body` 對齊。
內含 icon `circle-alert`（16×16，`$warning`）＋ text `limit-text`：

`預估 $13.92 已超過上限 $5.00 —— 預計可完成約 250 部後暫停（清單中已標示），其餘保留在佇列，可提高上限或稍後續跑。`

Noto Sans TC 13，`$text-primary`，`width: "fill_container"`、`textGrowth: "fixed-width"`。

**5. 頁尾數字。** `ft-sel-text` → `已選 696 部 · 預估`、`ft-sel-amount` → `$13.92`（`fill: "$warning"`）、
`ft-breakdown` → `抽取 696 部 $13.92 · 已排除 0 部`。

**6. 移除「已排除」段。** 這張稿講的是超限，不是排除；刪除 `excluded-header` 與兩列
`row-excluded-*`，把 `group-note` 改成
`線以上會跑，線以下會等。排除與暫停不是同一件事：暫停的項目仍然是已同意的。`

**7. caption。** x=72670、y=-5791，節點 `cap-F23-D-v2`，
內容 `F23 · 政策確認屏．超限（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 E 結束。

---

## 提示詞 F — F24-D-v2（政策確認屏．無可抽取項目）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

**這個提示詞必須在提示詞 D 完成之後才執行。**

複製 `f21-d-v2` 到座標 x=74270、y=-5746，命名 `f24-d-v2`。在複製出來的畫面裡：

1. 把 `body` 的所有子節點刪除（`summary-block`、`candidate-list`、`estimate-note`、`group-note`）。
2. 在 `body` 裡新增一個垂直 frame `empty-state`：
   `width: "fill_container"`、`height: "fill_container"`、`layout: "vertical"`、
   `alignItems: "center"`、`justifyContent: "center"`、`gap: 16`、`padding: [56, 24]`。
   內含（由上而下）：
   - frame `empty-icon-wrap`：56×56、`fill: "$bg-tertiary"`、`cornerRadius: 100`、
     `alignItems: "center"`、`justifyContent: "center"`，內含 icon `search-x`
     （lucide，28×28，`$text-muted`）
   - text `empty-title`：`沒有可以自動處理的項目`，Noto Sans TC 16/600，`$text-primary`
   - text `empty-body`：
     `這次的 96 部候選全部都需要語音辨識 —— 政策只自動處理可抽取內嵌字幕的項目。`
     Noto Sans TC 13，`$text-secondary`，`width: 420`、`textGrowth: "fixed-width"`、
     文字置中（`textAlign: "center"`）
   - text `empty-hint`：`要處理它們，請改用「批次生成字幕」逐次同意。`
     Noto Sans TC 13，`$text-secondary`，`width: 420`、`textGrowth: "fixed-width"`、置中
3. 頁尾 `footer`：把 `ft-left` 與 `ft-budget` 刪除，只留按鈕，並把 `btn-start`
   換成次要按鈕：`{ type: "ref", ref: "YDPhc", name: "btn-close", height: 44,
   descendants: { "L9cIf": { content: "關閉", fontFamily: "Noto Sans TC" } } }`。
   `footer` 設 `justifyContent: "end"`。
4. **不要**畫綠色勾勾。空不等於完成 —— 這一點是 `ConsentEmptyState` 用一次 critique 換來的。
5. caption：x=74270、y=-5791，節點 `cap-F24-D-v2`，
   內容 `F24 · 政策確認屏．無可抽取項目（桌機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 F 結束。

---

## 提示詞 G — F25-D-v2（設定頁「自動處理政策」格）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的畫布根層新增一個 frame，座標 x=75870、y=-5746，
名稱 `f25-d-v2`，`width: 768`、`height: 470`、`fill: "$bg-secondary"`、
`cornerRadius: "$radius-lg"`、`stroke: "$border-subtle"`、`strokeWidth: 1`、
`strokeAlignment: "inner"`、`layout: "vertical"`、`padding: 24`、`gap: 20`、`clip: true`。

（768 = J7-D 裁定的設定頁表單卡片寬度 `max-w-3xl`。）

依序放入：

**1.** 垂直 frame `card-head`（`gap: 6`、`width: "fill_container"`）：
- text `card-title`：`自動處理政策`，Noto Sans TC 18/700，`$text-primary`
- text `card-desc`：
  `只自動處理可抽取（僅翻譯費）的項目；語音辨識仍需逐次同意。`
  Noto Sans TC 13，`$text-secondary`，`width: "fill_container"`、`textGrowth: "fixed-width"`

**2.** 垂直 frame `field-budget`（`gap: 8`、`width: "fill_container"`）：
- text `budget-label`：`每次執行的上限金額`，Noto Sans TC 13/600，`$text-primary`
- 水平 frame `budget-input`：`width: 160`、`height: 40`、`fill: "$bg-primary"`、
  `cornerRadius: "$radius-sm"`、`stroke: "$border-subtle"`、`strokeWidth: 1`、
  `strokeAlignment: "inner"`、`padding: [0, 12]`、`alignItems: "center"`，
  內含 text `budget-value`：`$5.00`，JetBrains Mono 14，`$text-primary`
- text `budget-hint`：`達到上限會自動暫停，其餘保留在佇列，可稍後續跑。`
  Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`

**3.** 垂直 frame `field-model`（`gap: 8`、`width: "fill_container"`）：
- text `model-label`：`翻譯模型`，Noto Sans TC 13/600，`$text-primary`
- 垂直 frame `model-list`：`width: "fill_container"`、`gap: 6`、`padding: 6`、
  `cornerRadius: "$radius-md"`、`stroke: "$border-subtle"`、`strokeWidth: 1`、
  `strokeAlignment: "inner"`。兩列，每列是水平 frame
  （`width: "fill_container"`、`gap: 10`、`alignItems: "center"`、`padding: [8, 10]`、
  `cornerRadius: "$radius-sm"`）：

  - `row-sonnet`（選中，`fill: "$bg-tertiary"`）：
    - `{ type: "ref", ref: "4EHFN", name: "radio" }`
    - text `name`：`Claude Sonnet 4.5（預設）`，Noto Sans TC 13，`$text-primary`，
      `width: "fill_container"`、`textGrowth: "fixed-width"`
    - frame `grade`（`fill: "$success-tint"`、`cornerRadius: "$radius-sm"`、`padding: [2, 6]`）
      內含 text：`品質 A`，Noto Sans TC 11，`$success`
    - text `unit`：`每集約 $0.05`，JetBrains Mono 12，`$text-secondary`
  - `row-haiku`（未選中，無 fill）：
    - `{ type: "ref", ref: "Wd9AL", name: "radio" }`
    - text `name`：`Claude Haiku 4.5`
    - frame `grade`（`fill: "$bg-tertiary"`）內含 text：`品質 B`，`$text-secondary`
    - text `unit`：`每集約 $0.02`

- text `model-hint`：`模型只影響翻譯費用與品質；抽取內嵌字幕本身不花錢。`
  Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`

**4.** 水平 frame `card-foot`（`width: "fill_container"`、`justifyContent: "end"`、`gap: 10`）：
- `{ type: "ref", ref: "otvKh", name: "btn-save", height: 40,
  descendants: { "DaLcd": { content: "儲存", fontFamily: "Noto Sans TC" } } }`

**5.** caption：x=75870、y=-5791，節點 `cap-F25-D-v2`，
內容 `F25 · 設定頁「自動處理政策」格`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 G 結束。

---

## 提示詞 H — F22-M-v2（入口．活動頁，手機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡複製畫面 `fdu4y`（`f15-m-v2`）到座標 x=66270、y=-4671，
新畫面命名為 `f22-m-v2`。在複製出來的畫面裡（用名稱尋找）：

1. 刪除名為 `scrim` 的 rectangle 與名為 `sheet` 的 frame。留下 `backdrop`。
2. 找到 `page-header`（內含 `page-title`「活動」）。把它改成垂直 frame：
   `layout: "vertical"`、`gap: 10`、`padding: [12, 16]`、`height` 改為自動。
3. 在 `page-title` 之後新增文字節點 `policy-line`：
   `政策：只自動處理可抽取的項目 · 上限 $5.00`
   Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`。
4. 再新增一個垂直 frame `header-actions`（`gap: 8`、`width: "fill_container"`），
   兩顆按鈕**上下堆疊、各自滿寬**（手機沒有並排的空間）：
   - `{ type: "ref", ref: "otvKh", name: "btn-policy-run", height: 44,
     width: "fill_container",
     descendants: { "DaLcd": { content: "依政策執行", fontFamily: "Noto Sans TC" } } }`
   - `{ type: "ref", ref: "YDPhc", name: "btn-batch-manual", height: 44,
     width: "fill_container",
     descendants: { "L9cIf": { content: "批次生成字幕", fontFamily: "Noto Sans TC" } } }`
5. caption：x=66270、y=-4716，節點 `cap-F22-M-v2`，
   內容 `F22 · 入口．活動頁（手機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 H 結束。

---

## 提示詞 I — F21-M-v2（政策確認屏，手機 bottom sheet）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡複製畫面 `fdu4y`（`f15-m-v2`）到座標 x=71070、y=-4671，
新畫面命名為 `f21-m-v2`。在複製出來的畫面裡（用名稱尋找）：

**1. sheet 標題。** `sheet-title` 的內容從 `產生字幕` 改成 `依政策執行`。

**2. 頂部三行。** `body-top` 裡刪除 `search-sort-row`、`chips-scroll`、`toolbar`，
只留 `summary-block`。把 `summary-block` 的子節點全部換掉，改成 `gap: 6` 並依序放：

- 水平 frame `headline`（`gap: 4`、`alignItems: "baseline"`、`width: "fill_container"`、
  `wrap: true`）：
  - text `headline-a`：`將為`，Noto Sans TC 16/600，`$text-primary`
  - text `headline-n`：`46`，JetBrains Mono 16/600，`$text-primary`
  - text `headline-b`：`部產生字幕，約`，Noto Sans TC 16/600，`$text-primary`
  - text `headline-usd`：`$1.84`，JetBrains Mono 16/600，`$accent-text`
  - text `headline-cap`：`（上限 $5.00）`，Noto Sans TC 13，`$text-muted`
- text `policy-scope`：`只處理可抽取的項目（僅翻譯費）。需語音辨識的 96 部不在這次範圍。`
  Noto Sans TC 12，`$text-secondary`，`width: "fill_container"`、`textGrowth: "fixed-width"`
- text `spend-source`：`使用：Claude（你的金鑰）`
  Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`

這一段換行成 3～4 行是**預期的**，不要為了塞成一行縮字級或改文案。

**3. 清單全部收合。** `item-list` 的子節點全部刪除，改放（樣式沿用複製前的
`group-header-怪奇物語S4`，但寬度是 358）：

- 群組列 `group-已匹配電影`：勾選框 `4EHFN`、`disclosure` = `chevron-right`、
  `group-title` = `已匹配電影`、右側 badge `僅翻譯費 24`、`已選 22/24 · $0.88`
- 群組列 `group-怪奇物語S4`：同上，`group-title` = `怪奇物語 · 第 4 季`、
  badge `僅翻譯費 24`、`已選 24/24 · $0.96`
- 水平 frame `excluded-header`（`gap: 8`、`alignItems: "center"`、`width: "fill_container"`、
  `padding: [8, 0, 0, 0]`）：icon `chevron-down`（16×16、`$text-muted`）
  ＋ text `excluded-title`：`已排除（2）`，Noto Sans TC 13/600，`$text-secondary`，
  `width: "fill_container"`、`textGrowth: "fixed-width"`
- 兩列被排除的影片：複製手機版既有列 `row-星際效應`（節點 `ahNAO`）兩次，
  命名 `row-excluded-星際效應`、`row-excluded-教父`，`title` 依名稱設定，
  整列 `opacity: 0.6`，勾選框 `ref` 改成 `Wd9AL`，
  並在列的右側加 `restore-btn`（水平 frame，`gap: 4`、`alignItems: "center"`、
  `padding: [4, 8]`、`cornerRadius: "$radius-sm"`、`fill: "$bg-tertiary"`）：
  icon `rotate-ccw`（14×14、`$accent-text`）＋ text `復原`（Noto Sans TC 12/600、`$accent-text`）

若手機寬度容不下 `restore-btn` 與金額並排，**保留 `restore-btn`、把金額移到標題下方那一行**
（`sub` 之後），不要刪掉任何一個。

**4. 頁尾。** `sheet-footer` 裡：
- `ft-selected` 內的文字改成 `已選 46 部 · 預估 $1.84`
- `budget-row` 的標籤改成 `政策上限`，數值 `$5.00` 不變
- `budget-hint` 保持 `達到上限會自動暫停，可稍後續跑`
- `btn-start` 的 label 從 `開始產生` 改成 `開始`

**5. 驗收條件（brief 的硬要求）：** 頂部的 `headline`（一句話＋金額）與頁尾的 `開始`
**同時在 844 高的視野內**。如果放不下，先縮清單區的高度（清單本來就是可捲的），
**絕不**縮 headline 或把 `開始` 推到捲動區裡。

**6. caption：** x=71070、y=-4716，節點 `cap-F21-M-v2`，
內容 `F21 · 依政策執行．確認屏（手機）`，Noto Sans TC 14/600、`fill: "#888888"`。

> 提示詞 I 結束。

---

## 提示詞 J — J8-D（決策規格：政策與手動流程分工 + 三個未決）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的畫布根層新增一個 frame，座標 x=25080、y=24300，
名稱 `J8-D`，`width: 1240`、`fill: "$bg-secondary"`、`cornerRadius: "$radius-lg"`、
`stroke: "$border-subtle"`、`strokeWidth: 1`、`strokeAlignment: "inner"`、
`layout: "vertical"`、`padding: 32`、`gap: 24`、高度自動、`clip: true`。

（請對照既有的 `J7-D`（節點 `JBKis`）的排版語彙：標題 → 副標 → 分節標題 → 表格 → 註腳。）

**第 1 節 — 標題**
- text：`政策與手動流程的分工 —— 以及三個還沒裁的問題`，Noto Sans TC 22/700，`$text-primary`
- text：`sub-6-ux-consent-policy · brief 2026-09-03（Alexyu 2026-09-04 確認三個裁定）`，
  Noto Sans TC 13，`$text-secondary`

**第 2 節 — 分工表**
分節標題 text：`⚖️ 誰負責什麼`，Noto Sans TC 16/700，`$text-primary`。
下面一個 4 欄表格（表頭 `fill: "$bg-tertiary"`，每列 `padding: [10, 12]`、
列間 1px `$border-subtle` 分隔），欄位：`情境` / `走哪條路` / `誰同意` / `畫面`

| 情境 | 走哪條路 | 誰同意 | 畫面 |
| --- | --- | --- | --- |
| 掃到新片、有內嵌字幕 | 政策（抽取＋翻譯） | 規則（事前）＋按一次「依政策執行」 | F22 → F21 |
| 掃到新片、沒有內嵌字幕 | 手動（語音辨識） | 逐項勾選、逐次同意 | F22 →（批次生成字幕）→ F15 |
| 只想處理某幾部 | 手動 | 逐項勾選 | 媒體庫多選 → F15 |
| 政策候選超過上限 | 政策，跑到上限就暫停 | 同上，砍線在清單裡可見 | F23 |
| 沒有可抽取項目 | 不執行 | — | F24 |

**第 3 節 — 三個未決（兩案並陳）**
分節標題 text：`❓ 三個未決 —— 稿上採用的畫法是暫用，不是裁定`，Noto Sans TC 16/700，`$text-primary`。

三個子區塊，每個是一個垂直 frame（`fill: "$bg-primary"`、`cornerRadius: "$radius-md"`、
`padding: 16`、`gap: 10`、`width: "fill_container"`），內含：問題標題（14/600）、
A 案、B 案（各一個水平 frame：左邊 60px 寬的標籤 `A 案`／`B 案`、右邊說明文字），
以及一行 `稿上暫用：…`（12、`$text-muted`）。

- **(a) 政策上限與 `AI_RUN_BUDGET_USD` 是同一個數還是分開？**
  - A 案：**同一個數**。政策上限就是 `AI_RUN_BUDGET_USD`，設定頁那格是它的 UI。
    好處是全站只有一個上限、沒有兩個數字互相矛盾的可能；代價是手動批次也被同一個數綁住。
  - B 案：**分開**。政策有自己的上限，手動批次沿用 `AI_RUN_BUDGET_USD`。
    好處是「無人看管的自動花費」可以設得比「我盯著看的手動花費」更保守；
    代價是兩個數字、兩處說明，而且使用者要理解為什麼有兩個。
  - 稿上暫用：A 案（F21 頁尾寫「政策上限 $5.00」，與 F15 的「預算上限 $5.00」同值）。

- **(b)「已排除」是否持久化？**
  - A 案：**只在這次執行有效**。下次按「依政策執行」時，被排除的項目會再出現。
    好處是沒有隱形狀態，畫面上看到的就是全部；代價是同一部片可能每次都要重新排除一次。
  - B 案：**持久化**。排除過的項目下次不再出現，設定頁要有一個「已排除清單」可以管理。
    好處是排除一次就永久有效；代價是多一份看不見的狀態，
    而「為什麼這部片沒被處理」變成一個要去別的地方查的問題。
  - 稿上暫用：A 案（F21 的「已排除（2）」段就在同一個清單裡，沒有任何「永久」的字眼）。

- **(c) 政策確認屏要不要顯示「上次執行花了多少」？**
  - A 案：**顯示**。頂部三行下方多一行「上次執行：3 天前 · 花了 $2.31」。
    好處是花費有連續性，使用者能建立「一次大概多少錢」的直覺；
    代價是需要後端保存執行歷史（今天沒有這個端點），而且 `Rule 23` 的時間讀數要額外處理。
  - B 案：**不顯示**。確認屏只講這一次。
    好處是不需要新端點、畫面更短；代價是使用者每次都得重新猜「這樣算貴嗎」。
  - 稿上暫用：B 案（F21 沒有這一行）。

**第 4 節 — 註腳**
text：`裁定後請回填本頁，並在 F21/F23/F25 對應處落地；在裁定之前，任何一案都不得被當成既成事實引用。`
Noto Sans TC 12，`$text-muted`，`width: "fill_container"`、`textGrowth: "fixed-width"`

最後在 x=25080、y=24270 新增畫布根層 caption 文字節點 `Caption J8-D`，
內容 `J8 · 政策與手動流程分工`，Noto Sans TC 14/600、`fill: "#888888"`。

**注意：** J 區塊的 spec 畫面是變動高度的長條，做完後請確認 `J8-D` 的底部沒有撞到
下方的區塊（下一個區塊在 y≈27381）。若超過，請把 J 這一列的畫面一起處理，不要單獨移動一張。

> 提示詞 J 結束。

---

## 六、執行完之後（Claude 負責）

1. **MCP 唯讀複審**：逐字比對本文件的文案表、`ctx.bounds` / `ctx.problems` 驗證無溢出、
   標籤不與圖框名重疊（`feedback_pencil_label_overlap`）。
2. **確認 `.pen` 真的存檔**：`git status --porcelain ux-design.pen` 必須出現 ` M`。
   MCP 讀的是 app 記憶體，不是磁碟（`feedback_verify_pen_saved_before_commit`）。
   檔案 size 沒變**不代表**沒存檔。
3. **更新 `scripts/export-pen-screenshots.py` 的 `SCREENS`**（node ID 要等畫面建立後才知道）：

   ```python
   # sub-6-ux-consent-policy — 「依政策執行」政策流程（入口／確認屏／狀態／設定格）
   "<id>": ("flow-f-subtitle-v2", "f21-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f21-m-v2"),
   "<id>": ("flow-f-subtitle-v2", "f22-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f22-m-v2"),
   "<id>": ("flow-f-subtitle-v2", "f23-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f24-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f25-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f26-d-v2"),
   "<id>": ("flow-f-subtitle-v2", "f27-d-v2"),
   "<id>": ("flow-j-specs", "j8-d"),
   ```

4. **重出截圖**：`python3 scripts/export-pen-screenshots.py`。全量重出會動到 170＋ 張 PNG，
   **只 stage 這 10 張新的**，其餘 re-render 雜訊全部 `git checkout` 還原。
5. **同一個 commit** 收 `.pen` ＋ 截圖 ＋ export script ＋ 本文件 ＋ story 檔。
