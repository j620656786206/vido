# F15／F18 金錢陷阱修正 + Pencil Inline AI Agent 提示詞（sub-6-12）

**核定者：** Sally（UX Designer）　**日期：** 2026-09-09
**目標畫面：** `F15-D-v2`（`pwMzT`）、`F15-M-v2`（`fdu4y`）、`F18-D-v2`（`zBik1`）、
新增規格畫面 `F18-SPEC-ERR`
**執行方式：** Alexyu 在 Pencil 跑 Inline AI Agent → Claude 以 MCP 唯讀複審 → 依 `CLAUDE.md`
重出截圖同 commit（只 stage 真的改到的 PNG）

---

## 為什麼要動這三張稿

`/impeccable` critique 的 P1「篩選＋全選＝金錢陷阱」「開始失敗埋在清單底」與 P2「砍線不可見、
總額假精準」都已在程式碼修掉（`consentSelection.ts` / `CandidateListPanel.tsx`）。稿子上還看不到
其中三件事，而這三件都是**關於錢**的：

1. **砍線**。F18 的橫幅寫「預計可完成約 18 部後暫停」，但稿面上的清單沒有任何東西告訴讀者
   那 18 部是哪 18 部。程式碼已經在清單裡畫了一條分隔線，稿子要跟上。
2. **扣誰的錢**。critique 的專案 persona 紅旗：「花錢的事先問」問了多少錢，從來沒問過**誰的錢**。
   摘要條下面現在多一行來源說明。
3. **開始失敗的位置**。`startError` 原本是捲動區的最後一個子節點 —— 在 2,400 列的片庫上，
   按下「開始產生」失敗後，錯誤訊息在按鈕下方約四十個螢幕的地方。現在它固定在頁尾上方。
   依 `feedback_pencil_spec_standalone_screen`，這個位置事實用**獨立規格畫面**表達，
   不塞進 F18 的正常狀態裡（F18 畫的是「超支」，不是「超支又失敗」）。

---

## 三條裁定

### 裁定 1 — 砍線是清單裡的一列，不是覆蓋層

它陳述的是**位置**：線以上會跑，線以下會等。讀者需要它在流裡、在交界上，不是浮在旁邊。
線後的列 `opacity: 0.6`；不畫成停用樣式 —— 使用者仍然可以取消勾選，而且「暫停」不是「拒絕」。

### 裁定 2 — 來源行講「金鑰的歸屬」，不講金鑰本身

一行、12px、`$text-muted`，貼在摘要條下面（同一段落，不是新區塊）。
永遠不顯示金鑰內容或遮碼；只說是「你的金鑰」「環境變數金鑰」「尚未設定金鑰」或「自架」。

### 裁定 3 — 失敗訊息與超支橫幅同一區

兩者都是「按下開始之前你該知道的事」，都固定在頁尾上方、捲動區之外。順序是
**超支橫幅 → 失敗訊息 → 頁尾**：失敗是剛剛才發生的事，離按鈕最近。

---

## 提示詞 A — F15-D-v2（`pwMzT`，桌機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的 `f15-d-v2` 畫面裡，`body` 是節點 `RTmxa`（`layout: vertical`、`gap: 12`、
`padding: [16, 24]`），它的第一個子節點是 `njQZt`（summary-bar）。

請新增一個文字節點，插進 `RTmxa` 裡、**排在 `njQZt` 之後、`UmqHL`（search-sort-row）之前**
（也就是 index 1）：

```
name: "spend-source"
type: "text"
content: "使用：Claude（你的金鑰） · 語音辨識：OpenAI（你的金鑰）"
fontFamily: "Noto Sans TC"
fontSize: 12
fontWeight: "normal"
fill: "$text-muted"
width: "fill_container"
textGrowth: "fixed-width"
```

因為 `RTmxa` 的 `gap` 是 12，這一行會離摘要條稍遠。請把它改成貼著摘要條：
把 `njQZt` 與這個新節點**一起包進一個新的垂直 frame**，名稱 `summary-block`，
`layout: "vertical"`、`gap: 2`、`width: "fill_container"`，並把這個 frame 放在
`RTmxa` 的 index 0（也就是原本 `njQZt` 的位置）。

不要改動 `njQZt` 本身的內容或樣式。

> 提示詞 A 結束。

---

## 提示詞 B — F15-M-v2（`fdu4y`，手機）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的 `f15-m-v2` 畫面裡，`body-top` 是節點 `fusXx`，它的第一個子節點是
`SsLlR`（summary-bar）。

請比照桌機做同一件事：新增文字節點 `spend-source`（內容、字級、顏色與桌機**完全相同**，
見提示詞 A），把它與 `SsLlR` 一起包進新的垂直 frame `summary-block`
（`layout: "vertical"`、`gap: 2`、`width: "fill_container"`），放在 `fusXx` 的 index 0。

手機寬度較窄，這一行會換成兩行，這是預期的 —— **不要**為了塞成一行而縮字級或改文案。

> 提示詞 B 結束。

---

## 提示詞 C — F18-D-v2（`zBik1`，超出上限）

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 的 `f18-d-v2` 畫面裡有這些節點：
`MqUky`（body，vertical、gap 12、padding [16,24]）、`sfG9B`（summary-bar，是 `MqUky` 的
第一個子節點）、`B1SCFw`（candidate-list，vertical、gap 8）、`RrCEy`（limit-warning）、
`tHo1i`（limit-warning 裡的文字）。

`B1SCFw` 目前的子節點依序是：`cEsaG`（怪奇物語 S4E7）、`LZiaq`（怪奇物語 S4E8）、
`mMhPj`（全面啟動）、`t20SOB`（教父）、`FPqBM`（星際效應）。

請做以下四件事，全部在既有節點上修改或插入，不要重建畫面。

**1. 來源行。** 比照 F15：新增文字節點 `spend-source`
（content `"使用：Claude（你的金鑰） · 語音辨識：OpenAI（你的金鑰）"`、
`fontFamily: "Noto Sans TC"`、`fontSize: 12`、`fill: "$text-muted"`、
`width: "fill_container"`、`textGrowth: "fixed-width"`），
與 `sfG9B` 一起包進新的垂直 frame `summary-block`
（`layout: "vertical"`、`gap: 2`、`width: "fill_container"`），放在 `MqUky` 的 index 0。

**2. 預算砍線。** 在 `B1SCFw` 裡，**插在 `mMhPj`（全面啟動）之後、`t20SOB`（教父）之前**
（也就是 index 3）一個新的水平 frame：

```
name: "budget-cut"
type: "frame"
layout: (預設水平)
width: "fill_container"
gap: 12
alignItems: "center"
padding: [4, 0]
```

它有三個子節點，依序：

- `name: "rule-left"`, `type: "rectangle"`, `width: "fill_container"`, `height: 1`, `fill: "$warning"`
- `name: "cut-label"`, `type: "text"`, `content: "到此為止約 $5.00，之後的項目會暫停"`,
  `fontFamily: "Noto Sans TC"`, `fontSize: 12`, `fill: "$warning"`
- `name: "rule-right"`, `type: "rectangle"`, `width: "fill_container"`, `height: 1`, `fill: "$warning"`

**3. 線後的列淡化。** 把 `t20SOB`（教父）與 `FPqBM`（星際效應）兩個 frame 的
`opacity` 設為 `0.6`。**不要**改它們的 fill、stroke 或勾選狀態 —— 這兩列仍然是「已勾選」，
只是會被暫停。

**4. 橫幅文案。** 把 `tHo1i` 的 `content` 改成：

```
預估 $25.80 已超過上限 $5.00 —— 預計可完成約 18 部後暫停（清單中已標示），其餘保留在佇列，可提高上限或稍後續跑。
```

（只加了「（清單中已標示）」六個字，其餘一字不動。）

> 提示詞 C 結束。

---

## 提示詞 D — 新增規格畫面 `F18-SPEC-ERR`

> 貼給 Pencil Inline AI Agent 的內容從這裡開始。

在 `ux-design.pen` 裡新增一個獨立的規格畫面，說明「按下開始產生失敗時，訊息出現在哪裡」。
用 `FindEmptySpace({width: 760, height: 360, padding: 120, nodeId: "zBik1"})` 找位置，
放在 document 根層級，**不要**和其他畫面重疊。

先在畫面上方放一個標題文字節點（與其他畫面的 caption 同款）：

```
name: "cap-F18-SPEC-ERR"
type: "text"
content: "F18 規格 · 開始失敗時訊息的位置"
fontFamily: "Noto Sans TC"
fontSize: 14
fontWeight: "600"
fill: "#888888"
```

接著建立主 frame：

```
name: "f18-spec-err"
type: "frame"
layout: "vertical"
width: 720
gap: 0
fill: "$bg-secondary"
cornerRadius: "$radius-lg"
clip: true
```

它由上而下有四個子節點：

**(1) `list-tail`** — 代表捲動區的底部，讓讀者知道以下三塊都在捲動區**之外**。
frame，`layout: "vertical"`、`width: "fill_container"`、`padding: [16, 24]`、`gap: 6`：
- 文字 `name: "tail-hint"`, content `"⋮ 候選清單（可捲動）"`, `fontSize: 12`, `fill: "$text-muted"`
- 文字 `name: "tail-note"`, content `"金額為預估值，實際費用依內容長度而定。"`, `fontSize: 12`,
  `fill: "$text-muted"`

**(2) `limit-warning`** — 直接複製 `zBik1` 底下的 `RrCEy` 到這裡（`Copy`），不改內容。

**(3) `start-error`** — 新的水平 frame，`width: "fill_container"`、`fill: "$error-tint"`、
`cornerRadius: "$radius-md"`、`gap: 8`、`padding: [10, 12]`、`alignItems: "center"`、
外距靠 `margin` 不好處理，請改成把它包在一個 `padding: [0, 24]` 的透明 frame 裡
（`name: "start-error-wrap"`、`fill: "#00000000"`、`width: "fill_container"`）。
`start-error` 的子節點：
- icon `name: "err-icon"`, `icon: "circle-alert"`, `library: "lucide"`, `width: 16`, `height: 16`,
  `fill: "$error"`
- 文字 `name: "err-text"`, content `"無法開始：伺服器沒有回應（502）。請稍後再試。"`,
  `fontFamily: "Noto Sans TC"`, `fontSize: 13`, `fill: "$error-text"`,
  `width: "fill_container"`, `textGrowth: "fixed-width"`

**(4) `footer`** — 直接複製 `zBik1` 底下的 `A9foYw` 到這裡（`Copy`），不改內容。

最後在整個 frame 右側（畫面外）放一段說明文字：

```
name: "spec-note"
type: "text"
width: 300
textGrowth: "fixed-width"
fontFamily: "Noto Sans TC"
fontSize: 12
lineHeight: 1.6
fill: "#888888"
content: "順序固定：超支橫幅 → 失敗訊息 → 頁尾。三者都在捲動區之外，所以 2,400 列的片庫按下開始失敗時，訊息就在按鈕正上方，而不是四十個螢幕以下。失敗訊息帶 role=\"alert\"，會被螢幕閱讀器主動朗讀。"
```

> 提示詞 D 結束。

---

## 複審檢查表（Claude 以 MCP 唯讀執行）

1. `Get("pwMzT", …)` / `Get("fdu4y", …)` / `Get("zBik1", …)`：確認 `spend-source` 存在、
   三張稿文案**逐字相同**、`fontSize` 都是 12、`fill` 都是 `$text-muted`。
2. `Get("B1SCFw", {depth:1}).children` 的順序必須是
   `cEsaG → LZiaq → mMhPj → budget-cut → t20SOB → FPqBM`。
3. `t20SOB` 與 `FPqBM` 的 `opacity === 0.6`；`cEsaG`／`LZiaq`／`mMhPj` **沒有** `opacity`。
4. `tHo1i` 的 content 含「（清單中已標示）」且其餘字元未變。
5. `ctx.problems`：新畫面與三張改稿都不得出現 `partially clipped` / `fully clipped`。
6. 依 `feedback_pencil_label_overlap`：`cap-F18-SPEC-ERR` 與 `spec-note` 不得壓到任何 frame。

## 存檔與截圖（Alexyu）

1. Pencil ⌘S，然後 `git status --porcelain ux-design.pen` **必須**出現 ` M`
   （見 `feedback_verify_pen_saved_before_commit` —— MCP 匯出讀的是記憶體，不是磁碟）。
2. `python3 scripts/export-pen-screenshots.py`
3. 新畫面要先把 node ID 加進 `scripts/export-pen-screenshots.py` 的 `SCREENS` dict
   （`flow-f-subtitle`，代號 `f18-spec-err-d`）。
4. 只 stage `flow-f-subtitle/` 底下真的改到的 PNG；其餘 `git checkout` 還原（全量重出不具決定性）。
