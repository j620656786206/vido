# Pencil Inline AI Agent 提示詞 —— `J5-D` 部署未啟用狀態規格

> 自包含。貼進 Pencil 的 Inline AI Agent 即可執行，不需要其他上下文。
> 來源裁定：`9R-UX-auto-subtitle-unsupported-state-design.md`

---

## 任務

在 `ux-design.pen` 新增**一個** spec 畫面 `J5-D`，記錄「部署未啟用自動字幕生成時，
編輯 Modal 的欄位與媒體庫卡片各自該長什麼樣」。

**只新增，不修改任何既有節點。** 特別是 `hUVYm`(E5-D)、`P0P82x`(E5-M)、`sPzZT`(J4-D)
必須一個 byte 都不動。

---

## 放哪裡

| 節點 | x | y | w |
|---|---|---|---|
| Caption 文字 | 21060 | 24270 | 自動 |
| `J5-D` frame | 21060 | 24300 | 1240 |

Caption 節點照抄 `b6UOEN` 的樣式：
`fill:"#888888"`, `fontFamily:"Noto Sans TC"`, `fontSize:14`, `fontWeight:"600"`，
內容為 `J5 · 入庫自動生成 · 部署未啟用狀態規格`。

frame 高度由內容決定。背景 **`$bg-secondary`**，圓角 `$radius-lg`，內距 32。

> ⚠️ **2026-08-21 更正**：初版此處寫成 `$bg-primary`（外深內淺），與隔壁 `J4-D` **相反** ——
> J4-D 是 frame `#24304A`（外淺）、強調區塊 `#1B2336`（內深）。兩張 spec 頁並排在同一個 y，
> 內外反過來會看起來像出錯。日後複用本提示詞，**先讀 J4-D 的實際 fill 再決定**。
所有區塊寬 1176（=1240−32×2），縱向間距 28。

**排版一律照 `sPzZT`(J4-D) 的既有慣例**：先用 `Get("sPzZT", {depth:3})` 讀它的
主標／副標／小標／內文字級與顏色，然後沿用同一組值。不要自創字級。

---

## 設計 token（一律用變數，不要寫死色碼）

| 用途 | token |
|---|---|
| 背景 | frame `$bg-secondary` / 強調區塊底 `$bg-primary`（外淺內深，同 J4-D） |
| 分隔線 | `$border-subtle` |
| 主要文字 | `$text-primary` |
| 次要文字 | `$text-secondary` |
| 弱化文字 | `$text-muted` |
| **停用文字** | `$text-disabled` |
| 正在發生 | `$success` |
| **要求了但沒發生** | `$warning` |
| 通知列底／邊 | `$info-tint` / `$info` |

---

## 七個區塊

### 區塊 A —— 標題

- 主標：`部署未啟用時的誠實狀態 —— 控制項與卡片`
- 副標：`9R-10b 補審 M4 · VIDO_SUBTITLE_PIPELINE_MODE 預設 legacy，此時這個開關不可能生效`

### 區塊 B —— 三條事實（`$bg-secondary` 底，圓角 `$radius-md`，內距 16）

小標：`⚙️ 為什麼會有這個狀態`

三行內文，每行 `$text-secondary`：

1. `出貨預設是 legacy（config.go:160）。只有 pipeline 模式才會建構自動生成器（main.go 的 if cfg.SubtitlePipelineEnabled() 區塊）。`
2. `這是 env-only 設定：沒有設定頁入口，改完必須重啟容器（docs/deployment.md:134 明列它是無 UI 的環境變數之一）。`
3. `auto_subtitle 欄位在任何模式都寫得進資料庫 —— 存得下，但執行的人不存在。`

### 區塊 C —— 決策矩陣（四列表格）

小標：`四種組合，四個答案`

表頭：`伺服器模式` ｜ `媒體庫勾選` ｜ `Modal 欄位` ｜ `卡片 footer`

| | | | |
|---|---|---|---|
| `pipeline` | 已勾 | 正常，可變更 | 「· 自動處理免費字幕」`$success` |
| `pipeline` | 未勾 | 正常，可變更 | 整段不出現 |
| `legacy` | 已勾 | **停用 ＋ 通知列** | 「· 自動處理免費字幕（伺服器未啟用）」`$warning` |
| `legacy` | 未勾 | **停用 ＋ 通知列** | 整段不出現 |

後兩列是本頁新增的狀態，用 `$warning-tint` 底色標出來。

表格下方一行註記（`$text-muted`）：
`綠＝正在發生 · 琥珀＝你要求了但沒發生 · 不出現＝你沒要求。琥珀只用在系統與使用者意願不一致的那一格。`

### 區塊 D —— Modal 欄位停用態 Specimen

小標：`編輯媒體庫 Modal —— 停用態（桌面 472 / 行動 318）`

**並排兩個 specimen**，左桌面（寬 472）右行動（寬 318），
各自複製 `hUVYm` 的 `pjrvv`（自動生成開關）與 `P0P82x` 的 `S2kxbY` 結構，
然後套用以下差異：

1. checkbox 換成元件實例 **`Fn5MZ`**（CheckboxDisabledChecked，代表「已勾但停用」）
   —— 桌面右側再放一個小的 **`VSXl5`**（CheckboxDisabledEmpty）標註 `未勾時用這個`
2. 開關標籤文字 fill 改 **`$text-disabled`**（內容一字不改）
3. 兩句既有說明 fill **維持 `$text-secondary`**（內容一字不改，**且不降色**）

   > ⚖️ **2026-08-21 修訂**：初版寫「降 `$text-disabled`」是錯的。
   > 該 token 在 `styles.css:47` 被註記為 `intentionally sub-AA`，實測 3.55:1，
   > 12px 內文不通過 AA。WCAG 1.4.3 豁免的是「inactive user interface components」——
   > 控制項與其可及名稱（開關標籤）可以降，**旁邊的說明散文不行**。
4. **新增通知列**，位置在 checkbox 列**正下方、兩句說明之上**：
   - 底 `$info-tint`，圓角 `$radius-sm`，內距 12，左緣 4px 實心 `$info` 直條
   - 第一行 `$text-primary` 13px：`字幕生成管線尚未啟用，這個選項無法變更。`
   - 第二行 `$text-secondary` 12px：`請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。`
   - 第二行裡的 `VIDO_SUBTITLE_PIPELINE_MODE` 用 `fontFamily:"JetBrains Mono"`（或既有等寬字）
     與 `$text-primary`，讓它一眼看得出是要照抄的字串

⚠️ **行動版的 checkbox 列高度維持 45**（padding 12 上下），不可回到 33 ——
那是 9R-UX-auto-generation-toggle-design 二輪修掉的觸控目標問題。

### 區塊 E —— 卡片 footer Specimen

小標：`LibraryCard footer —— 三態並陳`

三個 specimen 直排，每個都是 12px 文字、`$text-muted` 起頭：

1. 標籤 `已啟用 · 已勾`：`2 個資料夾 · 316 個項目 · ` ＋ `自動處理免費字幕`（`$success`）
2. 標籤 `未啟用 · 已勾`：`2 個資料夾 · 316 個項目 · ` ＋ `自動處理免費字幕（伺服器未啟用）`（`$warning`）
3. 標籤 `任一 · 未勾`：`2 個資料夾 · 316 個項目`（整段不出現）

下方註記（`$text-muted`）：
`括號裡指名「誰」沒啟用。少了它會被讀成「你沒勾」—— 而使用者明明勾了，那是最糟的誤讀。`

### 區塊 F —— 文案理由（五條，每條一行）

小標：`文案的五條理由（逐條可指認）`

1. `① 兩句通知沿用已出貨的 409 文案，逐字相同（subtitle_pipeline_handler.go:112-113）—— 撞過 API 錯誤的使用者會認得同一句話。`
2. `② 顯示 env var 名稱是 Alexyu 2026-08-21 裁定：那是唯一可行的下一步，藏起來等於請使用者猜一個猜不到的字串。`
3. `③ 凍結的三句（開關標籤＋兩段說明）一字不改，停用態只降 fill —— 「這功能會幫我做什麼」正是使用者決定要不要去改 env var 的依據。`
4. `④ 全文不出現「掃描」二字（sub-4-3 AC #6 / 2026-08-07 誤解的原產地）。`
5. `⑤ 不作金額保證、不暗示自動化會花錢。`

### 區塊 G —— dev 落地備註（`$bg-secondary` 底）

小標：`🛠 落地備註`

1. `Modal：LibraryEditModal.tsx 目前是 autoSubtitleSupported && (...) 整段隱藏（PR #250）→ 改為永遠渲染，不支援時走本頁停用態。`
2. `卡片：LibraryCard.tsx:115 目前只讀 library.autoSubtitle、無 capability 閘門 → 需要把 autoSubtitleSupported 傳進卡片（props 或由 MediaLibraryManager 供給）。`
3. `資料來源已經有了：GET /api/v1/libraries 的 auto_subtitle_supported（PR #250 已出貨）。不需要新 endpoint、不需要新欄位。`
4. `停用時 update payload 仍省略 autoSubtitle（PR #250 既有行為），以免清掉使用者在 pipeline 模式下做過的選擇。`

---

## 完成前自檢（做完直接在同一個 execute 裡跑）

```js
Get("J5-D節點id", (n,c) => c.problems && Print("PROBLEM:", n.name, c.problems))
Get("J5-D節點id", n => typeof n.content === "string" && n.content.includes("掃描") && Print("掃描命中:", n.name, n.content))
```

兩個查詢都必須**零輸出**。然後 `TakeScreenshot` 整個 `J5-D`。

---

## 絕對不要做的事

- ❌ 不要修改 `hUVYm`(E5-D)、`P0P82x`(E5-M)、`sPzZT`(J4-D) 的任何節點
- ❌ 不要新建 checkbox 元件 —— `Fn5MZ` 和 `VSXl5` 已經存在
- ❌ 不要改動凍結的三句文案內容（只能改 fill）
- ❌ 不要在畫面上任何地方寫出「掃描」二字
- ❌ 不要寫死色碼，一律用 `$token`
- ❌ 不要新增 `J5-M`（spec 頁是文件，只有桌面版；J4-D 也沒有 M）
