# 9R-UX 入庫自動生成開關 —— Pencil Inline AI Agent 提示詞

> **給 Alexyu 的使用說明**：整份「## 提示詞正文」以下的內容複製貼給 Pencil 的 Inline AI Agent。
> 它沒有對話上下文，所以正文是自包含的。跑完後 ⌘S 存檔，回報我做 MCP review。

---

## Sally 的裁定摘要（給人看的，不用貼）

### ⚠️ 兩個必須先講的更正

**更正 1 —— story 裡的目標畫面是錯的。**
story 寫目標是 `6UCtX`(C4-D) / `2H4OM`(C4-M)。我實地讀了：`6UCtX` 是 **連線設定（qBittorrent）** 頁，
側欄連「媒體庫掃描」都沒有。媒體資料夾設定的真正所在是 **`KvZSc`(E1-D)** 與 **`uABWl`(E1-M)**，
標題「媒體庫掃描」、副標「設定掃描資料夾、排程，以及手動觸發媒體庫掃描」。
⇒ `LibraryEditModal.tsx:1` 的 `// Design ref: ... (6UCtX)` 是**錯的指向**（Rule 21 缺口）。

**更正 2 —— E1 的設計比程式碼落後一個世代。**
E1-D 畫的是**扁平資料夾列**（`/volume1/media/movies` 兩行 + 編輯/刪除圖示）。
但已出貨的 `MediaLibraryManager` 是**媒體庫卡片**（`LibraryCard`：名稱＋類型圖示＋⋮ 選單＋多條路徑列＋
footer「n 個資料夾 · m 個項目」）＋ `LibraryEditModal`。多媒體庫改版出貨時沒回頭更新 E1。
⇒ 我**不在本 story 重畫 E1**（那是重裁整個多媒體庫 IA，會把 9R-10b 撐爆）。改為：新畫面依**已出貨的程式碼**繪製，
並在 spec 頁上明寫 E1 已過期。E1 重畫另立條目。

### 三個裁定

**裁定 1 — 控制項與文案。** checkbox（不是 toggle switch）。modal 內其他控制項都是樸素原生表單元素，
switch 會是唯一異類；而且 checkbox 讀起來是「我同意這個政策」，switch 讀起來是「開關一台機器」——
這裡的語意是**同意**。位置：**四個欄位的最後一個**，前面加一條 divider（借用 E1-D 自己的 `#374461` 分隔語彙），
讓它讀成一個獨立的承諾而不是又一個屬性。checkbox 與它的兩行說明是**不可分割的一塊**。

**裁定 2 — 歸屬。** 開關**不放在頁面層級**，只放在**每個媒體庫的編輯 modal 裡**。
這一刀就解掉「掃描頁陷阱」的大半：設定的作用域是「這個媒體庫」，不是「掃描」。
E1-D 的頁面副標（正確地只描述掃描）**一個字都不動**。文案主詞一律是「新檔入庫後」，全文**不得出現「掃描」二字**。

**裁定 3 — 卡片要顯示。** 但**不加新徽章** —— 借用 `LibraryCard` footer 既有的頓點語法，
從「2 個資料夾 · 316 個項目」變成「2 個資料夾 · 316 個項目 · **自動處理免費字幕**」，
最後一段用同意流程的 success 綠 `#22C55E`（＝F15「抽取」徽章的顏色，語意就是「這個不用錢」）。
這樣不會和路徑列既有的狀態小圓點打架（Finding 3 的疑慮），也不會重演 `auto_detect` 的隱形布林（Finding 4）。
關閉時整段不出現。

---

## 提示詞正文（以下全部複製貼給 Inline AI Agent）

你要在這份 `.pen` 設計檔裡**新增三個畫面**。全部使用既有的 Midnight Blue v1 設計語彙。
**不要修改任何既有節點**，只做新增。

### 設計 token（一律照用，不要自創）

| 用途 | 值 |
|---|---|
| 畫面底色 | `#1B2336` |
| 卡片／Modal 底色 | `#24304A` |
| 邊框／分隔線 | `#374461` |
| 主要文字 | `#F2F2F2` |
| 次要文字 | `#B3B3B3` |
| 弱化文字 | `#A0AABE` |
| 主色（按鈕／連結） | `#3B82F6` |
| 淺藍連結 | `#60A5FA` |
| 成功綠（＝免費語意） | `#22C55E` |
| 警示橘（＝付費語意） | `#F59E0B` |
| caption 灰 | `#888888` |
| UI 字型 | `Noto Sans TC` |
| 等寬字型（路徑／數字） | `JetBrains Mono` |
| Modal 圓角 | `12` |
| 卡片圓角 | `8` |
| 輸入框／列圓角 | `6` |

既有可重用元件（**用 `ref` 引用，不要重畫**）：
- `4EHFN` = Component/Checkbox（已勾選：20×20，圓角 4，底 `#3B82F6`，內含白色 `check` 圖示）
- `Wd9AL` = Component/CheckboxEmpty（未勾選：20×20，圓角 4，`#A0AABE` 1px 外框，無底色）

---

### 畫面 1 — `E5-D`：媒體庫編輯 Modal（桌面）

**放置**：新增 root frame，`x=23200`、`y=10520`。
**在它上方新增一個 caption 文字節點**：`x=23200`、`y=10475`，內容 `E5 · 媒體庫編輯 Modal（桌面）`，
Noto Sans TC 14/600、`#888888`。

**Frame 規格**：`name="E5-D"`、`width=520`、height 自適應（fit_content）、`fill="#24304A"`、
`cornerRadius=12`、`stroke="#374461"`、`strokeWidth=1`、`layout="vertical"`、`padding=24`、`gap=20`。
（這是「只畫 modal 本身」的慣例，與既有的 `Paqlk`(H3 · Block 編輯 Modal)、`i74p2`(I3 · 儲存篩選 Modal) 一致。）

**內容，由上到下：**

1. **標題列**（horizontal、`justifyContent="space_between"`、`alignItems="center"`、寬度 fill_container）
   - 左：文字 `編輯媒體庫`，Noto Sans TC 17/600、`#F2F2F2`
   - 右：icon，library `lucide`、icon `x`、20×20、`fill="#A0AABE"`

2. **欄位一 — 名稱**（vertical、gap 8、寬 fill_container）
   - 標籤：`名稱`，Noto Sans TC 13/500、`#B3B3B3`
   - 輸入框：frame 寬 fill_container、高 40、`fill="#1B2336"`、`stroke="#374461"`、`strokeWidth=1`、
     `cornerRadius=6`、`padding=[10,14]`；內含文字 `我的電影`，Noto Sans TC 14/normal、`#F2F2F2`

3. **欄位二 — 類型**（同上結構）
   - 標籤：`類型`
   - 下拉框：同輸入框樣式，內部 horizontal + `justifyContent="space_between"` + `alignItems="center"`；
     左側文字 `電影`（Noto Sans TC 14、`#F2F2F2`），右側 icon `lucide`/`chevron-down` 16×16、`#A0AABE`

4. **欄位三 — 資料夾路徑**（vertical、gap 8、寬 fill_container）
   - 標籤：`資料夾路徑`
   - 兩條路徑列，各為 frame：寬 fill_container、`fill="#1B2336"`、`stroke="#374461"`、`strokeWidth=1`、
     `cornerRadius=6`、`padding=[10,14]`、horizontal、`justifyContent="space_between"`、`alignItems="center"`
     - 第一條左側文字 `/volume1/media/movies`，第二條 `/volume1/media/movies-4k`，
       兩者皆 **JetBrains Mono 13/normal、`#B3B3B3`**
     - 右側各放一個 icon `lucide`/`x`、14×14、`#A0AABE`
   - 新增路徑列：frame 寬 fill_container、高 40、`fill="#1B2336"`、`stroke="#374461"`、`cornerRadius=6`、
     `padding=[10,14]`，內含文字 `/media/movies`，JetBrains Mono 13、`#A0AABE`（placeholder 語意）

5. **分隔線**：rectangle，寬 fill_container、高 1、`fill="#374461"`

6. **⭐ 欄位四 — 自動生成開關（本畫面的重點）**
   frame，vertical、gap 10、寬 fill_container。
   - **第一列**（horizontal、gap 10、`alignItems="start"`）：
     - `ref` 引用 `4EHFN`（**已勾選**狀態）
     - 文字（`textGrowth="fixed-width"`、寬度填滿剩餘空間）：
       `新檔入庫後，自動完成免費的字幕處理`
       Noto Sans TC 14/500、`#F2F2F2`、lineHeight 1.5
   - **第二列 — 說明，兩段**（vertical、gap 6，**左邊縮排 30，對齊上方文字的起點**）：
     - 第一段（`textGrowth="fixed-width"`）：
       `影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。`
       Noto Sans TC 12.5/normal、`#B3B3B3`、lineHeight 1.6
     - 第二段（`textGrowth="fixed-width"`）：
       `需要 AI 翻譯或語音辨識的影片不會自動處理，它們會留在「產生字幕」清單裡，標好預估金額等你確認。`
       Noto Sans TC 12.5/normal、`#B3B3B3`、lineHeight 1.6

   > **這三句是定稿，一個字都不要改寫、不要精簡、不要重排。**
   > 「不會產生費用」與「在本機執行」是刻意與既有 F14 畫面同語彙。
   > **全文不得出現「掃描」二字。**

7. **底部按鈕列**（horizontal、gap 12、`justifyContent="end"`、寬 fill_container、上方留 4px）
   - 取消：frame，`padding=[10,20]`、`cornerRadius=8`、`stroke="#374461"`、`strokeWidth=1`、無底色；
     內含文字 `取消`，Noto Sans TC 14/500、`#B3B3B3`
   - 儲存：frame，`padding=[10,20]`、`cornerRadius=8`、`fill="#3B82F6"`；
     內含文字 `儲存`，Noto Sans TC 14/600、`#FFFFFF`

---

### 畫面 2 — `E5-M`：媒體庫編輯 Modal（手機）

**放置**：root frame，`x=23200`、`y=11530`。
**caption**：`x=23200`、`y=11485`，內容 `E5 · 媒體庫編輯 Modal（手機）`，Noto Sans TC 14/600、`#888888`。

**Frame 規格**：`name="E5-M"`、`width=350`、height 自適應、其餘樣式與 `E5-D` 相同，但 `padding=16`、`gap=16`。

內容與 E5-D **完全一致**，只調整這些尺寸：
- 標題 15/600；欄位標籤 12/500；輸入框高 40、`padding=[10,12]`、內文 13
- 路徑文字 JetBrains Mono 12
- **開關列的觸控目標最小高度 44** —— checkbox 那一列 frame 設 `height="fit_content(44)"`、`alignItems="center"` 之外
  仍維持 `alignItems="start"` 的視覺，請用 `padding=[6,0]` 補足到 44
- **兩段說明文字一字不得刪減或縮短**（這是安全機制，不是裝飾）。字級可降到 12，lineHeight 1.6
- 按鈕列改為兩顆等寬（`justifyContent="space_between"`，各 `width="fill_container"`，gap 10）

---

### 畫面 3 — `J4-D`：入庫自動生成 · 免費／付費界線規格

**放置**：root frame，`x=19720`、`y=24300`。
**caption**：`x=19720`、`y=24270`，內容 `J4 · 入庫自動生成 · 免費／付費界線規格`，Noto Sans TC 14/600、`#888888`。

**Frame 規格**：`name="J4-D"`、`width=1240`、height 自適應、`fill="#24304A"`、`cornerRadius=8`、
`stroke="#374461"`、`strokeWidth=1`、`layout="vertical"`、`padding=32`、`gap=28`。

**內容，由上到下：**

**區塊 A — 標題**
- 文字 `入庫自動生成 —— 免費／付費界線`，Noto Sans TC 22/600、`#F2F2F2`
- 文字 `9R-10b · 這個開關同意的是「免費的自動做」，不是「自動花錢」`，Noto Sans TC 14/normal、`#A0AABE`

**區塊 B — 裁定引用**（frame，`fill="#1B2336"`、`cornerRadius=6`、`padding=16`、vertical、gap 10、寬 fill_container）
- `⚖️ 2026-08-07 事故與裁定`，Noto Sans TC 13/600、`#F2F2F2`
- `pipeline 首次上線時，掃描完成直接掛整庫生成：一次 enqueue 1,026 項、約 2/3 走付費語音辨識、估算 ~US$200，使用者全程沒看到數字。裁定：掃描只更新中繼資料，付費生成必須在「先顯示金額的畫面」上明示選擇。`
  Noto Sans TC 12.5/normal、`#B3B3B3`、lineHeight 1.6、`textGrowth="fixed-width"` 寬 1160
- `⚖️ 2026-08-19 裁定：花錢須同意`，Noto Sans TC 13/600、`#F2F2F2`
- `付費動作不得自動執行。自動化只能做零花費的部分；需要付費的項目一律止步，留在同意清單等使用者確認。`
  Noto Sans TC 12.5/normal、`#B3B3B3`、lineHeight 1.6、`textGrowth="fixed-width"` 寬 1160

**區塊 C — 兩層對照表**（frame，vertical、gap 0、寬 fill_container）
表頭列（horizontal、`fill="#1B2336"`、`padding=[10,14]`、寬 fill_container）三欄，寬度比 260 / 500 / 400：
`層級`｜`包含哪些路線`｜`自動觸發會做什麼`，Noto Sans TC 12/600、`#A0AABE`

資料列共兩列，各為 horizontal frame、`padding=[14,14]`、`stroke="#374461"`、`strokeWidth={top:1}`：

| 欄1 | 欄2 | 欄3 |
|---|---|---|
| `零花費`（12/600、`#22C55E`） | `內建繁體中文字幕 → 直接沿用`、`內建簡體／混合字幕 → 簡轉繁（OpenCC）`、`軌道探測與抽取（ffprobe / ffmpeg）` 三行，各 12.5、`#B3B3B3` | `✅ 全自動完成。全部在本機執行，不會產生費用。` 12.5、`#B3B3B3` |
| `付費`（12/600、`#F59E0B`） | `內建英文字幕 → AI 翻譯`、`無文字字幕軌 → 語音辨識 ＋ AI 翻譯` 兩行，各 12.5、`#B3B3B3` | `🔴 一律止步。不呼叫、不排隊、不扣款。項目原地留在「缺繁中」，下次開「產生字幕」時帶著預估金額出現。` 12.5、`#B3B3B3` |

**區塊 D — 控制項狀態**（frame，horizontal、gap 40、寬 fill_container）
兩個 specimen，各為 vertical frame、gap 12、寬 560、`fill="#1B2336"`、`cornerRadius=6`、`padding=16`：
1. 標題 `關閉（預設）` 12/600、`#A0AABE`；下方一列 horizontal gap 10：`ref` 引用 **`Wd9AL`**（未勾選）＋
   文字 `新檔入庫後，自動完成免費的字幕處理`（14/500、`#F2F2F2`）；
   再下方註記 `全新媒體庫一律預設關閉。使用者沒開，就什麼都不會自動發生。` 12、`#A0AABE`
2. 標題 `開啟` 12/600、`#22C55E`；下方一列 horizontal gap 10：`ref` 引用 **`4EHFN`**（已勾選）＋同一句標籤；
   再下方**完整**放上那兩段說明文字（12.5、`#B3B3B3`、lineHeight 1.6）

**區塊 E — 媒體庫卡片 footer 狀態**（frame，vertical、gap 12、寬 fill_container）
- 小標 `LibraryCard footer —— 借用既有頓點語法，不新增徽章` 13/600、`#F2F2F2`
- 兩個並排 specimen（horizontal、gap 20），各為 frame 寬 580、`fill="#1B2336"`、`cornerRadius=8`、
  `stroke="#374461"`、`padding=16`、vertical、gap 10：
  - specimen 1：文字 `我的電影` 14/600、`#F2F2F2`；下方 footer 文字 `2 個資料夾 · 316 個項目` 12、`#A0AABE`
  - specimen 2：文字 `我的電影` 14/600、`#F2F2F2`；下方 footer 為 horizontal frame gap 0，
    由**兩個相鄰文字節點**組成：`2 個資料夾 · 316 個項目 · `（12、`#A0AABE`）＋
    `自動處理免費字幕`（12/600、**`#22C55E`**）
- 註記：`關閉時最後一段整段不出現。用文字不用彩色圓點，避免和路徑列既有的狀態小圓點搶讀。`
  12、`#A0AABE`、`textGrowth="fixed-width"` 寬 1160

**區塊 F — 文案理由**（frame，vertical、gap 10、寬 fill_container）
- 小標 `文案的四條硬性要求（全部可在上面的定稿裡逐條指認）` 13/600、`#F2F2F2`
- 四行，各 12.5、`#B3B3B3`、`textGrowth="fixed-width"` 寬 1160：
  - `① 讀得懂「只做免費的」 → 「這些都在本機執行，不會產生費用。」（與 F14 畫面同語彙）`
  - `② 讀得懂「要花錢的留著等你確認」 → 「它們會留在『產生字幕』清單裡，標好預估金額等你確認。」`
  - `③ 不得暗示掃描會產生字幕 → 主詞一律是「新檔入庫後」，全文不出現「掃描」二字。`
  - `④ 不得作金額保證 → 「不會產生費用」只涵蓋零花費那一層，不是對整個功能的承諾。`

**區塊 G — ⚠️ 已知設計落差**（frame，`fill="#1B2336"`、`cornerRadius=6`、`padding=16`、vertical、gap 8、寬 fill_container、
`stroke="#F59E0B"`、`strokeWidth=1`）
- `⚠️ E1-D／E1-M 尚未同步多媒體庫改版` 13/600、`#F59E0B`
- `E1 畫的是扁平資料夾列；已出貨的介面是「媒體庫卡片＋編輯 Modal」（MediaLibraryManager / LibraryCard / LibraryEditModal）。本規格頁與 E5-D／E5-M 依已出貨的程式碼繪製，與 E1 不一致是已知且刻意的。E1 重畫另立追蹤條目，不在本次範圍。`
  12.5、`#B3B3B3`、lineHeight 1.6、`textGrowth="fixed-width"` 寬 1160

---

### 收尾檢查（做完請自行核對）

1. 三個新 frame 都有 `name`，且每個子節點都有 `name`。
2. caption 文字節點在對應 frame **上方 30–45px**，**不得與 frame 或 Pencil 圖框名重疊**。
3. `E5-D`／`E5-M`／`J4-D` 內的**所有文字都沒有溢出**父容器（長句一律 `textGrowth="fixed-width"` 並給明確寬度）。
4. 定稿的三句文案（標籤 ＋ 兩段說明）在 `E5-D`、`E5-M`、`J4-D` 區塊 D 三處**逐字相同**。
5. 全部三個畫面裡**沒有出現「掃描」二字**。
6. 沒有修改任何既有節點。

---

# 追加提示詞 —— MUST-FIX ①（Sally MCP review 2026-08-20）

> 貼給 Pencil Inline AI Agent。只改一個屬性。

在這份 `.pen` 檔裡，把節點 ID `eIgZf`（它是 `E5-M` 畫面裡名為「開關列」的 frame）的
`padding` 從 `[6, 0]` 改成 `[12, 0]`。

不要改動任何其他節點、不要改任何文字內容、不要調整 `E5-D` 裡的對應節點 `y6gn3`。

原因：這一列是手機版的觸控目標，必須達到 44px。目前 checkbox 高 20px、上下 padding 各 6px，
合計只有 33px。改成上下各 12px 後為 45px，符合要求。
