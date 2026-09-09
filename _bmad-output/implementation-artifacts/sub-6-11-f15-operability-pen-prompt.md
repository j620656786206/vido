# F15 可操作性版式核定 + Pencil Inline AI Agent 提示詞（sub-6-11）

**核定者：** Sally（UX Designer）　**日期：** 2026-09-09
**目標畫面：** `F15-D-v2`（`pwMzT`）、`F15-M-v2`（`fdu4y`）
**執行方式：** Alexyu 在 Pencil 跑 Inline AI Agent → Claude 以 MCP 複審 → 依 `CLAUDE.md`
重出截圖同 commit（只 stage 真的改到的 PNG）

---

## 為什麼要動這張稿

`/impeccable` critique 的 P1 是「2,399 列不可操作」。這張稿畫的是**六列的理想樣本**，
六列當然可操作 —— 缺陷在稿子上看不見。真實片庫是一整面牆：沒有搜尋、沒有排序、
影集永遠攤開、電影一片平鋪。程式碼這一輪已經補上四件事，稿子要跟上，否則設計稿與
產品從此各說各話。

程式碼已交付（`CandidateListPanel.tsx` / `consentRows.ts`）：

1. **搜尋框**（44px 高）＋ **排序選單**，同一列，在摘要列下方。
2. **只有清單捲動**：摘要／搜尋／chips／工具列變成固定區塊，不再跟著列一起捲走。
3. **影集群組預設收合**，標頭有展開三角；**路線組成永遠顯示**（原本要勾了才出現）。
4. **電影分兩段**：`已匹配` 在前、`未匹配` 在後，兩段都可收合。

---

## 三條裁定（稿子要照這個畫）

### 裁定 1 — 搜尋列放在摘要列與 chips 之間

順序是「這裡有什麼（摘要）→ 找／排（搜尋＋排序）→ 縮小範圍（chips）→ 動作（工具列）→ 清單」。
搜尋是這一版的主要工具，chips 退為次要篩選，所以搜尋在 chips 之上。

### 裁定 2 — 路線組成永遠顯示，且是「整段」的組成不是「已勾」的組成

收合起來的影集，標頭是使用者唯一看得到的東西。「這部劇要花錢嗎」必須在**勾之前**
就能回答；原本的徽章要勾了才出現，等於在問題不再重要之後才作答。

- 徽章數字＝該段**可選取的全部列**的路線組成（＝把標頭勾滿會是什麼）。
- `已選 x/n · $subtotal` 仍然講**已勾的**部分。兩者並列，各自說各自的事。

### 裁定 3 — 電影「已匹配」在前

這一段順序也是**送出順序**（`groupOrder`），超出預算時錢會先花在前面那一段。
錢要先花在片名已經確定的片上；未匹配的沉到下面一段，像 `未知影集` 現在的作法一樣，
但它有自己的標頭，不會被忽略。

---

## 提示詞 A — F15-D-v2（`pwMzT`，桌機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的 `f15-d-v2` 畫面裡，`body` 是節點 `RTmxa`
（`layout: vertical`、`gap: 12`、`padding: [16, 24]`），它現在的子節點依序是：
`njQZt`（summary-bar）、`NiXrE`（route-chips）、`pB5IP`（toolbar）、
`T3rfXQ`（candidate-list）、`LjZL8`（estimate-note）、`UF8ic`（group-note）。

請做以下五件事，全部在既有節點上直接修改或插入，不要重建整個畫面。

**1. 新增「搜尋＋排序」列，插在 `njQZt` 之後、`NiXrE` 之前。**

新增一個 frame，`name: "search-sort-row"`、`width: "fill_container"`、
`gap: 8`、`alignItems: "center"`。裡面放兩個東西：

- 搜尋框：`Copy` 既有元件 `6MxLT`（`Component/SearchInput`）成一個 instance，
  `name: "search-input"`、`width: "fill_container"`。把它的 placeholder 文字
  （元件內節點 `iLI6G`）改成 **`搜尋片名或影集`**，`fill: "$text-muted"`、
  `fontFamily: "Noto Sans TC"`。搜尋 icon（`dDHHT`）維持 `search` 不動。
- 排序控制：`Copy` 既有元件 `955EZ`（`Component/SortDropdown`）成一個 instance，
  `name: "sort-select"`。把它的 label（元件內節點 `iEyRL`）改成 **`群組`**，
  `fontFamily: "Noto Sans TC"`。在 label 左側再插入一個 icon 節點，
  `name: "sort-icon"`、`type: "icon"`、`icon: "arrow-up-down"`、
  `width: 14`、`height: 14`、`fill: "$text-muted"`，放在 `iEyRL` 之前。
  chevron（`zdRmF`）維持 `chevron-down` 不動。

搜尋框與排序控制的高度都要是 **44**（可觸擊尺寸）：把兩個 instance 的
`height` 設成 `44`。

**2. 群組標頭 `o7L2K` 加展開三角，並改成收合狀態。**

- 在 `ALgnj`（checkbox）之後、`yQgKh`（group-title）之前，插入一個 icon 節點：
  `name: "disclosure"`、`type: "icon"`、`icon: "chevron-right"`、
  `width: 16`、`height: 16`、`fill: "$text-muted"`。
  （`chevron-right` ＝ 收合；展開時產品裡會轉 90 度變成朝下。）
- 在 `rg04P`（right）裡，於 `i1GmG`（`已選 3/9 · $0.78`）**之前**插入一個徽章
  frame：`name: "route-badge-asr"`、`fill: "$warning-tint"`、
  `cornerRadius: "$radius-sm"`、`padding: [2, 6]`、`alignItems: "center"`，
  裡面一個 text 節點 `name: "badge-lbl"`、`content: "語音辨識 9"`、
  `fontSize: 11`、`fill: "$warning"`、`fontFamily: "Noto Sans TC"`。
  把 `rg04P` 的 `gap` 設成 `8`。
- 因為群組現在是**收合**的，把 `Jht9P`（`row-怪奇物語 S4E7`）的 `enabled` 設成
  `false`，讓那一列不再顯示。

**3. 電影分兩段：加兩個段落標頭。**

在 `T3rfXQ`（candidate-list）裡：

- 在最前面（`u4sSoJ` 之前）`Copy` 一份 `o7L2K` 當作段落標頭，
  `name: "movies-matched-header"`。把複本裡的：
  group-title 文字改成 **`已匹配`**、
  route 徽章文字改成 **`抽取 2`**、徽章 `fill` 改成 `"$success-tint"`、
  徽章文字 `fill` 改成 `"$success"`、
  `已選` 文字改成 **`已選 1/2 · $0.05`**、
  disclosure icon 改成 `chevron-down`（這一段預設展開）。
- 在 `S3aO4`（`row-全面啟動`）**之前**再 `Copy` 一份 `o7L2K`，
  `name: "movies-unmatched-header"`。文字依序是
  **`未匹配`**、route 徽章 **`語音辨識 2`**（維持 `$warning-tint` / `$warning`）、
  **`已選 0/2 · $0.00`**、disclosure icon 改成 `chevron-down`。

**4. 段落標頭與影集標頭要看得出層級一樣。** 三個標頭都用 `o7L2K` 的既有樣式
（`fill: "$bg-tertiary"`、`cornerRadius: "$radius-md"`、`padding: [8, 12]`），
不要另外加框線或陰影。

**5. 更新畫面下方的說明文字 `UF8ic`。**
內容改成 **`群組 checkbox 為三態；影集預設收合，搜尋命中時自動展開。`**

完成後，把 `RTmxa` 的高度確認為自動（`fill_container` / 內容決定），
並確認 `DBPu5`（gen-dialog）沒有內容溢出 —— 需要的話把 `pwMzT` 的高度加大。

> 提示詞 A 到這裡結束。

---

## 提示詞 B — F15-M-v2（`fdu4y`，手機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的 `f15-m-v2` 畫面裡，`body-top` 是節點 `fusXx`，
它的子節點依序是 `SsLlR`（summary-bar）、`dfupi`（chips-scroll）、`oeZSc`（toolbar）。
清單是 `DXXEC`（item-list）。

**1. 新增「搜尋＋排序」列，插在 `SsLlR` 之後、`dfupi` 之前。**

一個 frame，`name: "search-sort-row"`、`width: "fill_container"`、`gap: 8`、
`alignItems: "center"`、`height: 44`。裡面兩個東西，同一列（手機不換行）：

- `Copy` 元件 `6MxLT` 成 instance，`name: "search-input"`、
  `width: "fill_container"`、`height: 44`，placeholder（`iLI6G`）文字
  **`搜尋片名或影集`**、`fill: "$text-muted"`、`fontFamily: "Noto Sans TC"`。
- `Copy` 元件 `955EZ` 成 instance，`name: "sort-select"`、`height: 44`。
  label（`iEyRL`）文字 **`群組`**、`fontFamily: "Noto Sans TC"`；
  在 label 左邊插入 icon 節點 `name: "sort-icon"`、`icon: "arrow-up-down"`、
  `width: 14`、`height: 14`、`fill: "$text-muted"`。

排序控制在手機上是**系統原生的挑選器**，所以不需要另外畫一張 sheet。

**2. 群組標頭 `zjghQ` 比照桌機：** 在 checkbox 之後插入
`icon: "chevron-right"`（16×16、`fill: "$text-muted"`）當展開三角；右側在
「已選」文字之前插入 `$warning-tint` 徽章、文字 `語音辨識 9`（11px、`$warning`）。
把 `m7KcM`（`row-怪奇物語 S4E7`）的 `enabled` 設成 `false`（收合狀態）。

**3. 手機不畫電影分段標頭** —— 這張稿只有一部未匹配電影（`ahNAO`／星際效應），
分成兩段會讓 390px 的畫面多兩條標頭、少兩列內容。手機上分段仍然存在於產品裡，
稿面以「有一部未匹配」的既有列徽章表達即可。

完成後確認 `tbF3W`（sheet）內容沒有溢出 390×740；`fusXx` 多一列 44px，
需要的話把 `DXXEC`（item-list）縮短或把 sheet 加高。

> 提示詞 B 到這裡結束。

---

## 複審清單（Claude 以 MCP 唯讀檢查）

| # | 檢查 | 方法 |
|---|---|---|
| 1 | 搜尋 placeholder 逐字是 `搜尋片名或影集` | `Get` 讀 `content` |
| 2 | 排序 label 逐字是 `群組`，icon 是 `arrow-up-down` | `Get` 讀 `content` / `icon` |
| 3 | 搜尋框與排序控制高度皆 44 | `ctx.bounds.height` |
| 4 | 三個標頭都有 disclosure icon，影集是 `chevron-right` | `Get` 讀 `icon` |
| 5 | 每個標頭的 route 徽章都在，且不是空的 | `Get` 讀子節點 |
| 6 | 收合的影集列 `enabled: false` | `Get` 讀 `enabled` |
| 7 | 無節點 `partially clipped` / `fully clipped` | `ctx.problems` |
| 8 | 沒有標籤重疊（`feedback_pencil_label_overlap`） | `ctx.bounds` 比對 |

---

## 完成後（Alexyu）

1. Pencil ⌘S 存檔，確認落盤（`git status` 看得到 `ux-design.pen` 有變更）。
2. `python3 scripts/export-pen-screenshots.py`
3. **只 stage 真的改到的 PNG**：`flow-b-detail-interaction/` 底下的 F15 兩張。
   其餘 re-render 噪音用 `git checkout` 還原。
4. `.pen` 與截圖同一個 commit。
