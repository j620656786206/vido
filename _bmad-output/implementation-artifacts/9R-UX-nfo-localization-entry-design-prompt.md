# Pencil Inline AI Agent 提示詞 —— 9R-UX：.nfo 在地化入口與覆寫確認

> **給 Alexyu：** 把下面每一個 `《提示詞 N》` 區塊**整段複製**貼給 Pencil 的 Inline AI Agent，一段一段執行。
> 每段都是自包含的（inline agent 沒有對話上下文）。跑完全部後 ⌘S 存檔，再回報給 Sally review。
>
> **順序很重要**：1 → 2 → 3 → 4 → 5。第 4、5 段會用到第 1 段建立的樣式語彙。

---

## 這次要做什麼（人類看的摘要）

| # | 目標 | 節點 |
|---|---|---|
| 1 | **校正**電影詳情 v2 的動作列（現況三顆是舊的假功能）＋加第四顆「在地化資訊」 | `D2HOt`（在 `uRGu2` / B3p-D 內） |
| 2 | 同上，影集詳情 v2 | `KgifB`（在 `N2fmG6` / B4p-D 內） |
| 3 | 同上，手機版（改成兩排） | `vWLjV`（在 `SzNRb` / B3p-M 內） |
| 4 | **新建** spec 畫面 `J6-D`：覆寫確認對話框（電影版＋影集版） | 新 frame |
| 5 | 在 `J6-D` 內加結果狀態 pill 四態 | 新 frame |

⚠️ **為什麼要校正**：`B3p-D`／`B4p-D`／`B3p-M` 的動作列目前是「播放／加入片單／⋯」，
但 **Vido 沒有播放功能、也沒有片單功能**（`LocalDetailV2.tsx` 檔頭已明文查證過）。
程式碼實際渲染的是「管理字幕／修改資訊／複製檔案路徑」。設計稿是舊的。
⚖️ Alexyu 2026-08-21 裁定：**順手校正整排**，不要在假的三顆旁邊加第四顆。

> **📌 2026-08-21 修正（提示詞 1 執行後）**：字幕圖示的正確名稱是 **`captions`**，不是 `subtitles`。
> Pencil 的 lucide 版本沒有 `subtitles`，而 `captions` **在本檔已被使用 14 次**（是既有的字幕語彙）。
> 提示詞 1 的 inline agent 已自行更正並回報 —— 判斷正確，Sally 追認。提示詞 2、3 已同步改為 `captions`。
> （前端 `lucide-react` 那邊 import 的是 `Subtitles`，那是同一個圖示的舊別名，**不是不一致**。）
>
> **📌 2026-08-21 修正（提示詞 3 執行後）**：`.pen` schema 的對齊值只吃 **`start` / `center` / `end`**
> （`justifyContent` 另有 `space_between` / `space_around`）。**沒有 `stretch`、也沒有 CSS 的 `flex-start` / `flex-end`。**
> 提示詞 3 的 inline agent 踩到 `stretch` 後自行改用 `start` —— 判斷正確，Sally 追認。
> 提示詞 4、5 原本寫的 `flex-start` / `flex-end` 已同步改為 `start` / `end`，避免再踩三次。
>
> **📌 2026-08-21 修正（提示詞 4 執行後）**：任何設了 `width: "fill_container"` 的 **text** 節點，
> **必須同時設 `textGrowth: "fixed-width"`** —— 否則 `.pen` 會忽略寬度、把文字排成不換行的單行，長句直接爆出容器。
> 另外 schema 的 Color **只吃 hex**，沒有 `transparent` 關鍵字（全透明請用 `#00000000`）。
> 兩者都是提示詞 4 的 inline agent 自行發現並更正的 —— 判斷正確，Sally 追認；提示詞 5 已同步補上。

---

## 《提示詞 1》—— 電影詳情 v2 的動作列（桌面）

```
在這個 .pen 檔案裡，修改 id 為 D2HOt 的 frame（它的 name 是 "actions"，位於 B3p-D 畫面內）。

這排按鈕目前顯示的是「播放／加入片單／⋯」，但那些功能並不存在。請把它改成產品真正有的四個動作。

保持 D2HOt 本身的屬性不變：gap 12、padding [6,0,0,0]、alignItems "center"、水平排列。

第 1 個按鈕 —— 修改既有的 id YcRA2（name "play"）：
- 把 name 改成 "act-subtitle"
- 保持 height 44、fill "$accent-primary"、cornerRadius 8、gap 8、padding [0,24]、justifyContent "center"、alignItems "center"
- 它的 icon 子節點 id 是 E33Zd：把 icon 從 "play" 改成 "captions"（library 維持 lucide、width/height 維持 18、fill 維持 "$text-on-accent"）
- 它的 text 子節點 id 是 JGyMs：把 content 從「播放」改成「管理字幕」，並把 fill 從 "#FFFFFF" 改成 "$text-on-accent"（其餘維持 Noto Sans TC / fontSize 15 / fontWeight "600"）

第 2 個按鈕 —— 修改既有的 id jAg2N（name "addlist"）：
- 把 name 改成 "act-edit"
- 保持 height 44、fill "#FFFFFF1A"、cornerRadius 8、gap 8、padding [0,20]、justifyContent "center"、alignItems "center"
- 它的 icon 子節點 id 是 RaZNS：把 icon 從 "plus" 改成 "pencil"（其餘不變）
- 它的 text 子節點 id 是 mNMfq：把 content 從「加入片單」改成「修改資訊」，並把 fill 從 "#F2F2F2" 改成 "$text-primary"（其餘不變）

第 3 個按鈕 —— 新增一個 frame，插入到 jAg2N 之後、P1E87 之前：
- name "act-localize"
- height 44、fill "#FFFFFF1A"、cornerRadius 8、gap 8、padding [0,20]、justifyContent "center"、alignItems "center"
- 子節點一：icon，name "i"，library "lucide"，icon "languages"，width 18、height 18、fill "$text-primary"
- 子節點二：text，name "l"，content「在地化資訊」，fontFamily "Noto Sans TC"，fontSize 15，fontWeight "600"，fill "$text-primary"

第 4 個按鈕 —— 修改既有的 id P1E87（name "more"）：
- 把 name 改成 "act-copypath"
- 保持 width 44、height 44、fill "#FFFFFF1A"、cornerRadius 8、justifyContent "center"、alignItems "center"
- 它的 icon 子節點 id 是 g2AKN：把 icon 從 "ellipsis" 改成 "copy"（其餘不變）

最後的順序必須是：act-subtitle → act-edit → act-localize → act-copypath。
不要新增任何其他節點，不要改動 D2HOt 以外的任何 frame。
```

---

## 《提示詞 2》—— 影集詳情 v2 的動作列（桌面）

```
在這個 .pen 檔案裡，修改 id 為 KgifB 的 frame（它的 name 是 "actions"，位於 B4p-D 畫面內）。

這排按鈕目前顯示的是「播放／加入片單／⋯」，但那些功能並不存在。請把它改成產品真正有的四個動作。

保持 KgifB 本身的屬性不變：gap 12、padding [6,0,0,0]、alignItems "center"、水平排列。

第 1 個按鈕 —— 修改既有的 id W92ihe（name "play"）：
- 把 name 改成 "act-subtitle"
- 其餘 frame 屬性全部不變
- 它的 icon 子節點 id 是 BUI7y：把 icon 從 "play" 改成 "captions"
- 它的 text 子節點 id 是 EDQNN：把 content 從「播放」改成「管理字幕」

第 2 個按鈕 —— 修改既有的 id K3jci1（name "addlist"）：
- 把 name 改成 "act-edit"
- 其餘 frame 屬性全部不變
- 它的 icon 子節點 id 是 sTZeH：把 icon 從 "plus" 改成 "pencil"
- 它的 text 子節點 id 是 ckB3X：把 content 從「加入片單」改成「修改資訊」

第 3 個按鈕 —— 新增一個 frame，插入到 K3jci1 之後、HwmjA 之前：
- name "act-localize"
- height 44、fill "#FFFFFF1A"、cornerRadius 8、gap 8、padding [0,20]、justifyContent "center"、alignItems "center"
- 子節點一：icon，name "i"，library "lucide"，icon "languages"，width 18、height 18、fill "$text-primary"
- 子節點二：text，name "l"，content「在地化資訊」，fontFamily "Noto Sans TC"，fontSize 15，fontWeight "600"，fill "$text-primary"

第 4 個按鈕 —— 修改既有的 id HwmjA（name "more"）：
- 把 name 改成 "act-copypath"
- 其餘 frame 屬性全部不變
- 它的 icon 子節點 id 是 ixYBo：把 icon 從 "ellipsis" 改成 "copy"

最後的順序必須是：act-subtitle → act-edit → act-localize → act-copypath。
不要新增任何其他節點，不要改動 KgifB 以外的任何 frame。
```

---

## 《提示詞 3》—— 手機版動作列（改成兩排）

```
在這個 .pen 檔案裡，修改 id 為 vWLjV 的 frame（它的 name 是 "actions"，位於 B3p-M 畫面內，畫面寬 390）。

目前它是一排三個（播放／＋／⋯），但那些功能並不存在，而且改成四個動作後一排放不下。
請改成「兩排、每排兩個、每個都有文字標籤」。

第一步：把 vWLjV 本身改成垂直排列
- layout 設為 "vertical"
- width 維持 "fill_container"
- gap 10、padding [12,16] 維持不變
- alignItems 改為 "start"（schema 只吃 start / center / end，沒有 stretch；橫向撐滿是靠兩個 row 各自的 width "fill_container" 達成）

第二步：在 vWLjV 內建立兩個水平的列 frame
列一：name "row-1"，width "fill_container"，gap 10，alignItems "center"（水平排列，不要設 layout）
列二：name "row-2"，width "fill_container"，gap 10，alignItems "center"（水平排列，不要設 layout）

第三步：把四個按鈕放進去。每個按鈕的共通規格是
width "fill_container"、height 46、cornerRadius 8、gap 8、justifyContent "center"、alignItems "center"
icon 子節點 name "i"、library "lucide"、width 18、height 18
text 子節點 name "l"、fontFamily "Noto Sans TC"、fontSize 15、fontWeight "600"

列一放：
1. 既有的 id WelmG（name "play"）移進 row-1，name 改成 "act-subtitle"，fill 維持 "$accent-primary"
   - icon 子節點 id om3jb：icon 從 "play" 改成 "captions"
   - text 子節點 id WPXfz：content 從「播放」改成「管理字幕」
2. 新增 frame，name "act-localize"，fill "$bg-tertiary"
   - icon：icon "languages"，fill "$text-primary"
   - text：content「在地化資訊」，fill "$text-primary"

列二放：
3. 既有的 id WjEQg（name "b-plus"）移進 row-2，name 改成 "act-edit"，fill 維持 "$bg-tertiary"，
   把 width 從 46 改成 "fill_container"，height 46 維持，並加上 gap 8
   - icon 子節點 id b9ZMI：icon 從 "plus" 改成 "pencil"，width/height 從 20 改成 18
   - 新增 text 子節點：name "l"，content「修改資訊」，fill "$text-primary"，fontFamily "Noto Sans TC"，fontSize 15，fontWeight "600"
4. 既有的 id F6DsD（name "b-ellipsis"）移進 row-2，name 改成 "act-copypath"，fill 維持 "$bg-tertiary"，
   把 width 從 46 改成 "fill_container"，height 46 維持，並加上 gap 8
   - icon 子節點 id nHiyy：icon 從 "ellipsis" 改成 "copy"，width/height 從 20 改成 18
   - 新增 text 子節點：name "l"，content「複製路徑」，fill "$text-primary"，fontFamily "Noto Sans TC"，fontSize 15，fontWeight "600"

完成後 row-1 是「管理字幕｜在地化資訊」，row-2 是「修改資訊｜複製路徑」，
四個按鈕都等寬填滿，都有文字標籤，高度都是 46。
不要改動 vWLjV 以外的任何 frame。
```

---

## 《提示詞 4》—— 新建 spec 畫面 J6-D 與兩個確認對話框

```
在這個 .pen 檔案裡新建一個 spec 畫面。座標請完全照下列數值，不要自行選位置。

第一步：建立標題文字（top-level 節點）
- type text，name "Caption J6-D"
- x 22400，y 24270
- content「J6 · .nfo 在地化 · 確認與結果規格」
- fontFamily "Noto Sans TC"、fontSize 14、fontWeight "600"、fill "#888888"

第二步：建立主 frame（top-level 節點）
- type frame，name "J6-D"
- x 22400，y 24300，width 1240
- fill "$bg-primary"、layout "vertical"、gap 28、padding 32

第三步：在 J6-D 內建立區塊 A —— 電影版確認對話框
建立 frame name "sec-movie"，width "fill_container"，layout "vertical"，gap 12
內含：
1. text，name "sh"，content「A · 電影：在地化確認（additive，不覆寫）」，
   fontFamily "Noto Sans TC"、fontSize 13、fontWeight "700"、fill "$text-secondary"
2. frame，name "dialog-movie"，width 520，fill "$bg-secondary"、cornerRadius "$radius-lg"、
   stroke "$border-subtle"、strokeWidth 1、layout "vertical"、gap 16、padding 24
   內含（由上到下）：
   a. text name "title"，content「將資訊在地化為繁體中文」，
      Noto Sans TC、fontSize 18、fontWeight "700"、fill "$text-primary"
   b. text name "body"，content「Vido 會用 AI 把片名、劇情與角色名翻成繁體中文，寫成播放器讀得到的 .nfo 檔。」，
      Noto Sans TC、fontSize 14、fontWeight "400"、fill "$text-secondary"、width "fill_container"
   c. frame name "note-safe"，width "fill_container"，fill "$success-tint"、cornerRadius 8、
      gap 8、padding 12、alignItems "center"（水平）
      內含 icon name "i"（lucide "shield-check"，18x18，fill "$success"）
      與 text name "t"，content「不會覆寫你現有的 .nfo —— 會寫進另一個播放器同樣認得的檔名」，
      Noto Sans TC、fontSize 13、fontWeight "500"、fill "$success"、width "fill_container"
   d. frame name "note-cost"，width "fill_container"，fill "$info-tint"、cornerRadius 8、
      gap 8、padding 12、alignItems "center"（水平）
      內含 icon name "i"（lucide "sparkles"，18x18，fill "$info"）
      與 text name "t"，content「會使用 AI 翻譯額度」，
      Noto Sans TC、fontSize 13、fontWeight "500"、fill "$info"、width "fill_container"
   e. frame name "btnrow"，width "fill_container"，gap 10，justifyContent "end"（水平）
      內含兩個按鈕：
      - frame name "btn-cancel"，height 40、fill "#FFFFFF1A"、cornerRadius 8、padding [0,18]、
        justifyContent "center"、alignItems "center"，
        內含 text content「取消」，Noto Sans TC、fontSize 14、fontWeight "600"、fill "$text-primary"
      - frame name "btn-confirm"，height 40、fill "$accent-primary"、cornerRadius 8、padding [0,18]、
        justifyContent "center"、alignItems "center"，
        內含 text content「開始在地化」，Noto Sans TC、fontSize 14、fontWeight "600"、fill "$text-on-accent"

第四步：在 J6-D 內建立區塊 B —— 影集版確認對話框
建立 frame name "sec-tv"，width "fill_container"，layout "vertical"，gap 12
內含：
1. text，name "sh"，content「B · 影集：覆寫確認（單槽，一定覆寫）」，
   Noto Sans TC、fontSize 13、fontWeight "700"、fill "$text-secondary"
2. frame，name "dialog-tv"，width 520，fill "$bg-secondary"、cornerRadius "$radius-lg"、
   stroke "$border-subtle"、strokeWidth 1、layout "vertical"、gap 16、padding 24
   內含（由上到下）：
   a. text name "title"，content「將資訊在地化為繁體中文」，
      Noto Sans TC、fontSize 18、fontWeight "700"、fill "$text-primary"
   b. text name "body"，content「Vido 會用 AI 把劇名、簡介與角色名翻成繁體中文。」，
      Noto Sans TC、fontSize 14、fontWeight "400"、fill "$text-secondary"、width "fill_container"
   c. frame name "note-replace"，width "fill_container"，fill "$warning-tint"、cornerRadius 8、
      gap 10、padding 12、alignItems "start"（水平）
      內含 icon name "i"（lucide "triangle-alert"，18x18，fill "$warning"）
      與 frame name "txt"，width "fill_container"，layout "vertical"，gap 4，內含兩個 text：
        text 1 content「影集只有一個檔名可用，這會覆寫現有的 tvshow.nfo。」，
              Noto Sans TC、fontSize 13、fontWeight "600"、fill "$warning"、width "fill_container"
        text 2 content「原始檔會先備份成 tvshow.nfo.orig；之後再執行也不會覆蓋這份備份。」，
              Noto Sans TC、fontSize 13、fontWeight "400"、fill "$warning"、width "fill_container"
   d. frame name "note-cost"，width "fill_container"，fill "$info-tint"、cornerRadius 8、
      gap 8、padding 12、alignItems "center"（水平）
      內含 icon name "i"（lucide "sparkles"，18x18，fill "$info"）
      與 text name "t"，content「會使用 AI 翻譯額度」，
      Noto Sans TC、fontSize 13、fontWeight "500"、fill "$info"、width "fill_container"
   e. frame name "opt-episodes"，width "fill_container"，gap 10，padding [12,0,0,0]、
      alignItems "start"（水平）
      內含：
      - frame name "cb"，width 18、height 18、cornerRadius 4、fill "transparent"、
        stroke "$border-subtle"、strokeWidth 1.5（這是「未勾選」狀態，裡面不要放勾號）
      - frame name "txt"，width "fill_container"，layout "vertical"，gap 4，內含兩個 text：
        text 1 content「連同每一集的集名與劇情」，
              Noto Sans TC、fontSize 14、fontWeight "600"、fill "$text-primary"、width "fill_container"
        text 2 content「每一集各翻譯一次，額度用量會明顯增加。」，
              Noto Sans TC、fontSize 12、fontWeight "400"、fill "$text-muted"、width "fill_container"
   f. frame name "btnrow"，width "fill_container"，gap 10，justifyContent "end"（水平）
      內含兩個按鈕：
      - frame name "btn-cancel"，height 40、fill "#FFFFFF1A"、cornerRadius 8、padding [0,18]、
        justifyContent "center"、alignItems "center"，
        內含 text content「取消」，Noto Sans TC、fontSize 14、fontWeight "600"、fill "$text-primary"
      - frame name "btn-confirm"，height 40、fill "$accent-primary"、cornerRadius 8、padding [0,18]、
        justifyContent "center"、alignItems "center"，
        內含 text content「備份並覆寫」，Noto Sans TC、fontSize 14、fontWeight "600"、fill "$text-on-accent"

兩個對話框請左右並排：把 sec-movie 與 sec-tv 放進一個 name "row-dialogs" 的水平 frame，
width "fill_container"、gap 32、alignItems "start"，再把 row-dialogs 放進 J6-D。
```

---

## 《提示詞 5》—— J6-D 內的結果狀態四態

```
在這個 .pen 檔案裡，找到 name 為 "J6-D" 的 frame，在它的最後面（row-dialogs 之後）新增一個區塊。

建立 frame name "sec-result"，width "fill_container"，layout "vertical"，gap 12
內含：
1. text name "sh"，content「C · 結果狀態（inline pill，就地取代那顆按鈕，不是浮動 toast）」，
   Noto Sans TC、fontSize 13、fontWeight "700"、fill "$text-secondary"
2. frame name "pills"，width "fill_container"，layout "vertical"，gap 10、alignItems "start"

在 pills 內建立四個 pill。每個 pill 的共通規格：
水平 frame、height 40、cornerRadius 999、gap 8、padding [0,16]、alignItems "center"
內含 icon name "i"（lucide，18x18）與 text name "t"（Noto Sans TC、fontSize 13、fontWeight "600"）

pill 1 —— name "pill-movie-ok"
  fill "$success-tint"，icon "check"，icon fill "$success"
  text content「已寫入繁中資訊」，fill "$success"

pill 2 —— name "pill-tv-ok"
  fill "$success-tint"，icon "check"，icon fill "$success"
  text content「已覆寫，原檔已備份為 .nfo.orig」，fill "$success"

pill 3 —— name "pill-batch-partial"
  fill "$warning-tint"，icon "triangle-alert"，icon fill "$warning"
  text content「影集資訊已更新 · 22 集完成、2 集略過」，fill "$warning"

pill 4 —— name "pill-disabled"
  fill "$error-tint"，icon "key-round"，icon fill "$error"
  text content「尚未設定翻譯服務 · 前往設定」，fill "$error"

最後在 sec-result 的最下方加一個註解文字：
text name "note"，width "fill_container"，textGrowth "fixed-width"，
content「「略過」＝資料庫有這一集，但硬碟上找不到對應的影片檔，所以沒有地方可以放 .nfo。」，
Noto Sans TC、fontSize 12、fontWeight "400"、fill "$text-muted"
```

---

## 跑完之後（Alexyu 的收尾清單）

1. **⌘S 存檔**（很重要 —— MCP 讀的是 app 記憶體，不是磁碟；沒存檔 Sally review 會看到舊的）
2. 更新 `scripts/export-pen-screenshots.py` 的 `SCREENS` dict，加入 J6-D：
   在 `"alrIw": ("flow-j-specs", "j5-d"),` 那行後面加一行 —— **node id 請用 Pencil 回傳的真實 id**：
   ```python
   "<J6-D 的實際 node id>": ("flow-j-specs", "j6-d"),
   ```
3. `python3 scripts/export-pen-screenshots.py`
4. **只 stage 真的有變更的 PNG**：預期是 `flow-b-detail-v2/b3p-d.png`、`b4p-d.png`、`b3p-m.png`
   與新的 `flow-j-specs/j6-d.png`。**其他全部 `git checkout` 掉**（全量重跑是非決定性的，每張都會有 byte diff）
5. 回報給 Sally 做 MCP 唯讀 review

---

## Sally 的裁定紀錄（review 時對照用）

| AC | 裁定 | 理由 |
|---|---|---|
| #1 入口位置 | **選 (a) 直接加第四顆有標籤的 secondary 按鈕**，排在「修改資訊」之後、「複製路徑」之前 | 「在地化資訊」與「修改資訊」語意相鄰（都在改中繼資料），放一起讀得順。桌面 `info` 區寬 962，四顆總寬約 500，空間充裕。**不引入 overflow `⋯`** —— 那會把一個會覆寫檔案的動作藏進選單，風險與可見度不對等 |
| #1 手機 | **改成兩排、每排兩個、四個都有文字標籤** | 390 寬放不下四個有標籤的按鈕；而把它們壓成無標籤 icon 更糟 —— 一個會覆寫檔案的按鈕不該是個猜謎圖示 |
| #2 電影 vs 影集 | **按鈕完全一致**（同位置、同圖示 `languages`、同文字「在地化資訊」），**差異全部收在確認對話框** | 兩種媒體型別在使用者心中是同一件事；在按鈕上分岔會讓人以為是兩個功能。風險揭露交給對話框，那裡才有空間講清楚 |
| #3 電影也要確認 | **⚠️ 推翻 story AC #3.3 的預設** —— 電影**也**要確認對話框 | story 說電影 additive 所以不需確認。但**電影一樣會花錢**（LLM 翻譯）。2026-08-19「花錢須同意」裁定的精神是「付費動作要有明確的同意動作」。一鍵無提示就開始計費不可接受。電影版對話框強調「不會覆寫」＋「會花額度」，是**安心＋誠實**兩件事 |
| #4 `include_episodes` 預設 | **預設不勾** | 勾了 = 使用者第一次嘗試就可能花 24 倍的錢。「先做一部，看滿不滿意，再做整劇」是自然的學習路徑，也符合「更貴的動作要更明確的動作」 |
| #4 主鍵文案 | 電影「**開始在地化**」／影集「**備份並覆寫**」 | 主鍵不得是含糊的「確定」。影集那顆要把兩個動作都說出來 —— 先備份、才覆寫 |
| #5 結果回饋 | **inline 狀態 pill，就地取代那顆按鈕**，不是浮動 toast | repo **沒有共用 Toast 元件**（`ui/` 內查無）；既有語彙是 `RequestButton.tsx` 的 inline pill（`role="status"` + `aria-live="polite"` + tinted rounded-full）。沿用既有語彙，不發明新系統 |
| #5 `skipped` 的講法 | 對使用者說「**略過**」，並在 spec 附註解釋 | 「skipped」的技術意義是「DB 有這集但硬碟沒檔案」—— 使用者需要知道那不是失敗，也不用去修 |

### 兩個順帶發現（給 Bob / 未來的人）

1. **`LocalDetailV2.tsx` 的 Rule 21 檔頭語法與現實不符。**
   它寫 `// Implements: Component/Detail-Movie-v2 (uRGu2) + Component/Detail-TV-v2 (N2fmG6)`，
   但 `uRGu2` / `N2fmG6` **不是 Reusable Component，是 screen frame**（`B3p-D` / `B4p-D`）。
   依 Rule 21 的文法，screen frame 應該用 `// Design ref: ux-design.pen Screen {ScreenName} ({nodeId})`。
   **本 story 不改**（不在範圍），但值得立案。

2. **v2 影集詳情沒有手機版畫面。**
   只有 `B3p-M`（電影版手機），**沒有 `B4p-M`**。影集手機版的季集手風琴、以及本案的動作列兩排排版，
   在手機上目前**無設計覆蓋**。本次《提示詞 3》只能改到 `B3p-M`。值得立案補齊。
