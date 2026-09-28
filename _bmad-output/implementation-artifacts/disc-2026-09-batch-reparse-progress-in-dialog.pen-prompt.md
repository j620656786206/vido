在 `ux-design.pen` 裡新增一張規格稿，畫「批次重新解析」對話框的三個狀態。全部用母版 instance，不要手畫按鈕或進度條。做完請 ⌘S 存檔。

## 位置與命名

1. 父節點是群組 `szz7O`（C · 桌面 Desktop）。在裡面新增一個 frame：
   - `name: "C24-D"`，`x: 52460`，`y: 18481`，`width: 1440`，`height: 900`
   - `fill: "$bg-secondary"`，`layout: "vertical"`，`padding: "$Space/4xl"`，`gap: "$Space/2xl"`，`clip: true`
2. 在同一個群組 `szz7O` 新增標題文字（放在 frame 上方，不在 frame 裡面）：
   - `name: "cap C24-D"`，`x: 52460`，`y: 18436`
   - `content: "C24-D · 批次重新解析 — 比對進度 spec（桌面）"`
   - `fontFamily: "$Type/Family/Canvas"`，`fontSize: "$Type/Body/Size"`，`fontWeight: "600"`，`lineHeight: "$Type/Body/Line"`，`fill: "#888888"`

## frame 內容（由上往下）

A. 標題：`Copy("ZXJVl", frame)`（Text/H2），content 改成「批次重新解析 — 對話框的三個狀態」。
B. 說明：`Copy("O6y7O", frame)`（Text/Body），`width: "fill_container"`，`textGrowth: "fixed-width"`，`fill: "$text-secondary"`，content：
   「按下「重新解析」後，後端會在背景比對。對話框依序經過三個狀態；隨時可以按「關閉」，清單在比對完成後會自動更新。這一輪比對的是所有待整理的片，不只你勾的那幾部，所以總數可能大於你勾的數量。」
C. 一列三張示範卡：frame `name: "states"`，`layout: "horizontal"`，`gap: "$Space/2xl"`，`width: "fill_container"`，`alignItems: "start"`。
   每張卡：frame `fill: "$bg-primary"`，`cornerRadius: "$radius-lg"`，`layout: "vertical"`，`padding: "$Space/xl"`，`gap: "$Space/lg"`，`width: "fill_container"`。卡內第一個是 `Copy("PMg7U", card)`（Text/Label，`fill: "$text-secondary"`）當小標，第二個是對話框 instance：`Insert(card, {type:"ref", ref:"m6KMPr", width:"fill_container"})`。

對話框母版 `m6KMPr`（Component/DialogFrame）的子節點：標題 `CsRIQ`、副標 `lFEZd`、內容文字 `UeqT2`（在 body `InAqu` 裡）、footer 的取消鈕 `NiyZh`（Outline，文字節點 `kgGEW`）、確認鈕 `A3pjr`（Primary，文字節點 `DaLcd`）。
三張卡的 footer 都一樣：把 `A3pjr` 的 `enabled` 設成 `false`（只留一顆鈕），`NiyZh/kgGEW` 的 content 改成「關閉」。

### 卡 ①　小標「① 已排入（第一筆進度來之前）」
- `CsRIQ`：「重新解析中」
- `lFEZd`：「已排入比對 5 項，等待開始…」
- `UeqT2`：「本輪會整理所有待整理的片，不只你勾的 5 部。」

### 卡 ②　小標「② 比對中（收到 enrich_progress）」
- `CsRIQ`：「比對中」
- `lFEZd`：「本輪整理 20 部（含你勾的 5 部）」
- 把 `UeqT2` **換成**進度條 instance：`Replace(instance+"/UeqT2", {type:"ref", ref:"TzhFY", width:"fill_container"})`。進度條母版 `TzhFY`（Component/ProgressBar）的子節點：填色 `ZFxvR`（在 track `MrIoZ` 裡）、左字 `hk8Jr`、右字 `w4P4XP`。覆寫：`hk8Jr` →「目前：你的名字」、`w4P4XP` →「3 / 20」、`ZFxvR` 的 width 改成 track 的 15%（track 若是 fill_container，填色就設 `width: 65`）。
- 在進度條下面再放一行 `Copy("O6y7O", body)`：`fill: "$text-secondary"`，content「成功 2・失敗 1・略過 0」。

### 卡 ③　小標「③ 比對完成（收到 enrich_complete）」
- `CsRIQ`：「比對完成」
- `lFEZd`：「成功 18・失敗 2 — 清單已更新」
- 把 `UeqT2` 換成一個錯誤清單框：frame `fill: "$bg-secondary"`，`cornerRadius: "$radius-md"`，`padding: "$Space/md"`，`layout: "vertical"`，`gap: "$Space/xs"`，`width: "fill_container"`；裡面三行文字（都 `fontSize: "$Type/Label/Size"`，`lineHeight: "$Type/Label/Line"`）：
  - 「2 個項目失敗：」`fontWeight: "500"`，`fill: "$error-text"`
  - 「寄生上流：找不到符合的作品」`fill: "$text-secondary"`
  - 「瀑布：TMDb 沒有回應」`fill: "$text-secondary"`

D. 最下面一段規格文字：`Copy("O6y7O", frame)`，`width: "fill_container"`，`textGrowth: "fixed-width"`，`fill: "$text-secondary"`，content：
「規則：① 三個狀態同一個對話框，標題隨狀態換字（重新解析中 → 比對中 → 比對完成）。② 數字全部來自後端事件（每 5 部送一次），前端不自己算、不用時鐘。③ 完成後若又收到進度（排在別人後面的那一輪），回到「比對中」。④ 一項都沒排到就沒有這三態，維持原本的「操作完成」。⑤ 手機不另外畫：同一個對話框，左右各留 16。」

做完後檢查：三張卡沒有超出 frame（`ctx.problems` 為空），對話框寬度撐滿卡片，然後 ⌘S。
