# Story 9R-UX: 入庫自動生成的 per-library 開關（design）

Status: done  <!-- Sally 2026-08-20：MCP review 二輪 PASS（MUST-FIX ① 已修並複驗）。交付 PR #246，待 merge。 -->

**Epic:** epic-9R-subtitle-route-c · **Risk: 🟡 UX/DESIGN-ONLY（Sally ux-designer，NOT dev）· 零程式碼 —— 但文案踩在花費紅線上**
**Created:** 2026-08-20（SM Bob, create-story）
**Source:** Rule 24 lane ② from `9R-10b-on-add-autotrigger` create-story（2026-08-20）。
⚖️ **Alexyu 裁定 2026-08-20**：不走 `// Design ref: PENDING` 暫掛，**設計先行**。
**Depends on:** nothing（只需 Pencil.app 執行中）。
**Blocks:** `9R-10b-on-add-autotrigger`（硬阻斷 —— 該 story Status = `blocked`，本 story `done`
且經 Sally MCP review 後才轉回 `ready-for-dev`）。

---

## Story

As Alexyu（想要「晚上丟檔案、早上字幕就好了」的 NAS 擁有者），
I want 在媒體庫設定裡有一個看得懂、不會誤會會被扣錢的開關，
so that 我敢把它打開 —— 而且我打開之後**知道自己同意了什麼、沒同意什麼**。

---

## ⚖️ 這個開關背後的裁定（設計前必讀，文案直接受它管轄）

**2026-08-07 事故**：pipeline 首次上線時 scan-complete 直接掛整庫 sweep，一次掃描 enqueue 1,026 項、
約 2/3 走付費 ASR、估算 ~US$200，使用者全程沒看到數字。
**裁定原文**（`apps/api/internal/cost_consent_test.go` 檔頭）：
「scanning updates metadata and nothing else；paid generation is chosen explicitly on a screen that shows the estimate first」。

**2026-08-19 Alexyu 裁定**：「**9R-10b 花錢須同意**」。落地成兩層：

| 層 | 自動觸發會做什麼 |
|---|---|
| **零花費** | 內嵌**繁中**軌直接沿用、**簡中/混合**軌 OpenCC 簡轉繁、ffprobe 探測與軌道抽取 → ✅ **自動做完** |
| **付費** | LLM 翻譯（英文軌）、ASR 語音辨識（無文字軌） → 🔴 **一律止步**，原地留在「缺繁中」，下次開同意流程（F14–F20）帶著金額等你按 |

⇒ **這個 checkbox 同意的是「免費的自動做」，不是「自動花錢」。**
文案如果讓使用者以為打開就會扣錢 → 沒人敢開，功能等於沒出。
文案如果讓使用者以為打開就全自動包含翻譯 → **直接違反 2026-08-19 裁定**，是本 story 最嚴重的失敗模式。

---

## 🔎 Findings（2026-08-20 create-story 逐檔驗證 —— 設計前必讀的現況事實）

1. **開關的家在「掃描設定」頁，這是個陷阱。**
   `MediaLibraryManager` 唯一的呼叫端是 `ScannerSettings.tsx`（grep 全 repo 僅此一處＋gallery fixture）
   ⇒ 媒體庫卡片與編輯 modal 都長在 `/settings/scanner`。
   **衝突**：sub-4-3 AC #6 白紙黑字「**不得出現任何暗示掃描本身會產生字幕的字樣**」。
   現在要在**掃描設定頁**上放一個字幕開關。⇒ Sally 必須裁定：這個開關的標題／位置／分組要怎麼寫，
   才不會讀成「掃描 = 會產生字幕」。（例如獨立分節、或標題就講清主詞是「新檔入庫後」而非「掃描時」。）

2. **編輯 modal 是手刻的，不是 Radix。**
   `LibraryEditModal.tsx` 自己畫 overlay，沒有 focus trap（對照：F14–F20 全走 `ui/Dialog`）。
   欄位只有三個：名稱（input）／類型（select）／資料夾路徑（input + 路徑列表）。
   ⇒ 新增第四個欄位是**淨新增**，且它是**唯一的 checkbox** —— modal 裡目前沒有任何 checkbox 語彙可抄。

3. **卡片上看不出開關狀態。**
   `LibraryCard.tsx` 只渲染：類型圖示＋名稱、⋮ 選單（編輯／刪除）、路徑列（含 accessible 狀態小圓點）、
   footer「{n} 個資料夾 · {m} 個項目」。**沒有任何 per-library 布林狀態的顯示語彙。**
   ⇒ Sally 要裁定：開了自動生成的媒體庫，**卡片上要不要看得出來**？
   （若不顯示，使用者要逐一點進 modal 才知道哪個開了；若顯示，需要一個新的 badge/pill 語彙，
   而路徑列已經有一種小圓點狀態語彙了 —— 兩種狀態語彙會不會打架，是設計問題。）

4. **已有一個同型的 per-library 布林，但它在 UI 上是隱形的。**
   `media_libraries.auto_detect` 欄位存在（migration 020）、model 有、repo 四處 CRUD 全通 ——
   但 `UpdateLibraryRequest` 沒有它，FE 也從沒渲染過它。
   ⇒ 這是「加了欄位卻沒有 UI」的**前車之鑑**。本開關不能重蹈：設計必須把「使用者怎麼看到、怎麼改」講死。

5. **目標畫面在 `.pen` 裡是「設定頁整頁」，沒有 modal 的獨立畫面。**
   `LibraryEditModal.tsx:1` 檔頭掛的是 `// Design ref: ux-design.pen Screen 10 Settings Desktop (6UCtX)`
   —— 也就是整頁，不是 modal 本身。⇒ 本 story 可能需要**新增**一張 modal 畫面，而不只是改既有畫面。

6. **同意流程那邊的文案語彙已定稿，應對齊而非另創。**
   F15/F16/F18/F19 已有成熟的「免費 vs 付費」語彙：chip「僅翻譯費」（success 綠）／「付費」（warning 橘）、
   「…本機執行，不會產生費用」、金額前 `≈`。⇒ 本開關的說明文案應**沿用同一套語彙與色彩語意**，
   讓使用者在兩個地方讀到的是同一種話。

---

## 目標畫面（`.pen`）

> 🔴 **2026-08-20 Sally 實地勘查後更正 —— 原表的兩個 node id 是錯的。**

**原表寫的（作廢）**：`6UCtX`(C4-D) / `2H4OM`(C4-M)。
**實況**：`6UCtX` 是 **連線設定（qBittorrent）** 頁 —— 表單欄位是 主機位址／使用者名稱／密碼，
側欄七項（連線設定・快取管理・系統日誌・服務狀態・備份與還原・匯出/匯入・效能監控）**連「媒體庫掃描」都沒有**。
媒體資料夾設定的真正所在是 **E1-D / E1-M**（flow-e-scanner）。
⇒ `LibraryEditModal.tsx:1` 的 `// Design ref: ux-design.pen Screen 10 Settings Desktop (6UCtX)` 是**錯的指向**（Rule 21 缺口，見 Discovery Triage）。

| 畫面 | node id | 檔名 | 座標 | 角色 |
|---|---|---|---|---|
| E1-D 掃描設定（桌面） | `KvZSc` | `flow-e-scanner/e1-d.png` | x=17040, y=10520, 1440×900 | 現況參照 —— **本 story 不修改**（見下方落差說明） |
| E1-M 掃描設定（手機） | `uABWl` | `flow-e-scanner/e1-m.png` | x=17040, y=11530, 390×844 | 同上 |
| **E5-D 媒體庫編輯 Modal（桌面）** | **新增** | `flow-e-scanner/e5-d.png` | x=23200, y=10520, w=520 | ⭐ 開關本體（AC #1） |
| **E5-M 媒體庫編輯 Modal（手機）** | **新增** | `flow-e-scanner/e5-m.png` | x=23200, y=11530, w=350 | AC #4 |
| **J4-D 免費／付費界線規格** | **新增** | `flow-j-specs/j4-d.png` | x=19720, y=24300, w=1240 | AC #3 + AC #5 |

**標準模式來源**：`Paqlk`(H3 · Block 編輯 Modal)、`i74p2`(I3 · 儲存篩選 Modal) —— 本檔已有「只畫 modal 本身、
不畫整頁」的獨立畫面慣例，E5-D/E5-M 照辦。

### ⚠️ 已知設計落差（本 story 不修，另立條目）

E1-D 畫的是**扁平資料夾列**（`/volume1/media/movies`、`/volume1/media/tv-shows` 兩行 + 編輯/刪除圖示）。
但已出貨的 `MediaLibraryManager` 是**媒體庫卡片**（`LibraryCard`：名稱＋類型圖示＋⋮ 選單＋多條路徑列＋
footer「n 個資料夾 · m 個項目」）＋ `LibraryEditModal`。多媒體庫改版出貨時沒回頭更新 E1。
⇒ 重畫 E1 等於重裁整個多媒體庫 IA，會把 9R-10b 撐爆。**新畫面依已出貨的程式碼繪製**，落差在 J4-D 區塊 G 明文標註。

---

## Acceptance Criteria

### AC #1 — 開關本體定稿（modal 內）
- 交付一張畫面（新增或在既有畫面上標註）呈現媒體庫編輯 modal 的**四欄位**版本：名稱／類型／資料夾路徑／**自動生成開關**。
- 定稿內容須包含：控制項型態（checkbox / toggle switch）、標籤文案、說明文案、在四個欄位中的**順序位置**、
  以及開/關兩種狀態的視覺。
- **文案硬性要求**（不可協商，源自 2026-08-19 裁定）：
  - 必須讓使用者讀懂「**只做免費的**」；
  - 必須讓使用者讀懂「**要花錢的不會自動做，會留著等你確認**」；
  - **不得**出現任何暗示「掃描會產生字幕」的字樣（sub-4-3 AC #6）；
  - **不得**承諾「絕不花錢」以外的任何金額保證（軟上限語彙沿用 F18 慣例）。

### AC #2 — Finding 1 的分組裁定
- 明確裁定這個開關在 `/settings/scanner` 頁面上的**歸屬與標題**，並在畫面上標註理由。
- 若裁定「掃描設定頁不是它的家」，須指出替代位置並說明對 `MediaLibraryManager` 唯一呼叫端的影響
  （這會把 9R-10b 的 FE 範圍從「一個 checkbox」放大 —— 屬於合法裁定，但必須明講，SM 會據以重估範圍）。

### AC #3 — Finding 3 的卡片裁定
- 明確裁定 `LibraryCard` 上**是否**顯示自動生成狀態。
- 若「是」：交付 badge/pill 的樣式與文案，並說明它與既有路徑狀態小圓點的視覺層級關係（誰重誰輕）。
- 若「否」：在畫面上以註記寫下理由（讓下一個審查者不用重吵）。

### AC #4 — 行動版
- `2H4OM`（c4-m）對應處理：checkbox＋兩行說明在窄螢幕的排版、44px 觸控目標、說明文字是否縮短。
- **不可略過** —— 說明文案是本開關的安全機制，行動版把它截掉等於拿掉安全機制。

### AC #5 — 規格頁（standalone）
- 「免費層 vs 付費層」的對照與文案理由**另開一張獨立規格頁**（`flow-j-specs` 慣例），
  不得擠進 c4-d 的實作畫面裡。內容至少涵蓋：兩層各含哪些路線、開關同意的是哪一層、
  被止步的項目去哪裡（F15 同意清單）、以及 2026-08-07／2026-08-19 兩次裁定的引用。
- 依 `feedback_pencil_spec_standalone_screen` 慣例。

### AC #6 — 交付與流程
- 依 `.pen` inline-agent 慣例：**Sally 出提示詞 → Alexyu 在 Pencil 跑 Inline AI Agent → Sally 用 MCP review**。
  Sally **不直接**編輯 `.pen`。提示詞落在 `9R-UX-auto-generation-toggle-design-prompt.md`。
- Review 逐條對照本檔 AC #1–#5，PASS 或 PASS-WITH-MUST-FIX（MUST-FIX 須複驗）。
- **標籤不得與其他內容重疊**（`feedback_pencil_label_overlap`）；註記寫在 frame **上方**（flow layout 慣例）。
- 若兩個狀態畫面渲染相同 → 交 Sally 裁決，不可自選方案也不可只丟 backlog（`feedback_identical_rendering_is_sally_decision`）。

### AC #7 — 截圖與 commit
- 跑 `python3 scripts/export-pen-screenshots.py`（需 Pencil.app 執行中）。
- **新畫面須先加進 `scripts/export-pen-screenshots.py` 的 `SCREENS` dict**（key = node id，value = `(flow-folder, code)`）。
- ⚠️ **全量重跑是非決定性的** —— 只 stage **設計真的變動**的 PNG，其餘 `git checkout` 掉，避免 commit 重繪雜訊。
- ⚠️ **commit 前必須確認 `.pen` 已存檔**（MCP 匯出讀 app 記憶體不讀磁碟 —— `feedback_verify_pen_saved_before_commit`）。
- `.pen` 與截圖**同一個 commit**。

---

## Tasks / Subtasks

- [x] **Task 1 — 讀裁定與現況**（AC: #1）
  - [x] 讀本檔「⚖️ 裁定」與 Findings 1–6
  - [x] MCP 取得 schema ＋ 實地勘查 → **發現原目標 node id 錯誤**，更正為 E1-D/E1-M（見「目標畫面」節）
  - [x] 擷取 v1 Midnight Blue token 與同意流程（F14/F15）色彩語彙
- [x] **Task 2 — 三個裁定**（AC: #1 #2 #3）—— 見下方「🎨 Sally 的三個裁定」
- [x] **Task 3 — 出提示詞**（AC: #6）
  - [x] `9R-UX-auto-generation-toggle-design-prompt.md`（233 行，自包含，三畫面全節點錨定）
- [x] **Task 4 — 行動版＋規格頁**（AC: #4 #5）—— 已含在提示詞的畫面 2 與畫面 3
- [x] **Task 5 — MCP review**（AC: #6）—— 一輪 PASS-WITH-MUST-FIX → 二輪 **PASS**，見下方「🔍 Sally MCP Review」
  - [x] MUST-FIX ① E5-M 開關列 33 → **45px**（`eIgZf` padding `[6,0]`→`[12,0]`，commit `433d013e`，已複驗）
- [x] **Task 6 — 截圖與 commit**（AC: #7）
  - [x] `SCREENS` dict 補 `hUVYm`/`P0P82x`/`sPzZT` → 匯出 → 只 stage 變動 PNG → `.pen` 落盤驗證 → commit
  - [x] PR **#246**（2 commits）

---

## 🎨 Sally 的三個裁定（2026-08-20）

### 裁定 1 — 控制項與文案（AC #1）

**控制項：checkbox**，不是 toggle switch。兩個理由：modal 內其他控制項都是樸素原生表單元素，switch 會是唯一異類；
更重要的是**語意** —— checkbox 讀起來是「我同意這個政策」，switch 讀起來是「開關一台機器」。這裡要的是**同意**。

**位置**：四個欄位的**最後一個**，前面加一條 `#374461` 分隔線（借用 E1-D 自己的 `xPu8F`/`kg4EN`/`ObQLg` 分隔語彙），
讓它讀成一個獨立的承諾，而不是又一個屬性。
**checkbox 與它的兩段說明是不可分割的一塊** —— 任何情況下不得被捲動邊界或分隔線拆開。

**文案定稿**（三句，逐字不可改）：

| 位置 | 文案 |
|---|---|
| 標籤 | `新檔入庫後，自動完成免費的字幕處理` |
| 說明 ① | `影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。` |
| 說明 ② | `需要 AI 翻譯或語音辨識的影片不會自動處理，它們會留在「產生字幕」清單裡，標好預估金額等你確認。` |

四條硬性要求逐條對照：
- ① 只做免費的 → 「這些都在本機執行，不會產生費用。」**刻意與 F14 畫面的 `L0QAf`「這個步驟在本機執行，不會產生費用。」同語彙**
- ② 要花錢的留著等你確認 → 說明 ②，並以**真實去處**「產生字幕」指名（sub-4-3 已把標題從「批次生成字幕」改為「產生字幕」）
- ③ 不得暗示掃描產生字幕 → 主詞一律「新檔入庫後」，**全文不出現「掃描」二字**
- ④ 不得作金額保證 → 「不會產生費用」只涵蓋零花費那一層，非對整個功能的承諾

### 裁定 2 — 歸屬（AC #2）

**開關不放頁面層級，只放每個媒體庫的編輯 modal 裡。**

這一刀解掉「掃描頁陷阱」的大半：設定的作用域是「**這個媒體庫**」，不是「掃描」。
E1-D 的頁面標題「媒體庫掃描」與副標「設定掃描資料夾、排程，以及手動觸發媒體庫掃描」**一個字都不動** ——
它們正確地只描述掃描，本來就沒錯。

⇒ **9R-10b 的 FE 範圍不因此放大**（AC #2 要求的「若換位置須明講」不觸發）。

### 裁定 3 — 卡片顯示（AC #3）

**要顯示，但不加新徽章。**

借用 `LibraryCard` footer 既有的頓點語法：
`2 個資料夾 · 316 個項目` → `2 個資料夾 · 316 個項目 · 自動處理免費字幕`
最後一段用 `#22C55E`（＝F15「抽取」徽章的成功綠，語意就是「這個不用錢」），其餘維持 `#A0AABE`。關閉時整段不出現。

兩個理由：
1. **不會和路徑列既有的狀態小圓點打架**（Finding 3 的疑慮）—— 用文字不用彩色圓點，兩種狀態語彙不搶讀。
2. **不會重演 `auto_detect` 的隱形布林**（Finding 4）—— 不必點進 modal 就看得見。

---

---

## 🔍 Sally MCP Review（2026-08-20，唯讀比對 `hUVYm` / `P0P82x` / `sPzZT`）

**判定：PASS-WITH-MUST-FIX** —— 1 項 MUST-FIX，**且該項是我提示詞的算術錯誤，不是執行瑕疵**。

### 逐條對照

| AC | 檢查 | 結果 |
|---|---|---|
| #1 控制項 | checkbox 而非 switch；`ref → 4EHFN`（E5-D `uIl6C` / E5-M `E5BSD4`）；位於四欄位最末；前置分隔線（`YnB3q` / `Sn9MU`） | ✅ |
| #1 文案 | 三句定稿在 **E5-D / E5-M / J4-D 區塊 D** 三處**逐字相同**（`RkDAY`＝`VV4Cx`＝`IrEM1`；`AC8c8`＝`Ydcjx`＝`Po08s`；`JshcP`＝`qTYWW`＝`YYXE0`） | ✅ |
| #1 硬性要求③ | E5-D 含「掃描」節點 **0** 個；E5-M **0** 個 | ✅ |
| #2 歸屬 | 開關只在 modal 內；E1-D／E1-M 零改動（`ux-design.pen` diff = **+1669 / −0**，純新增） | ✅ |
| #3 卡片 | J4-D 區塊 E：`YIbiq`「2 個資料夾 · 316 個項目 · 」`#A0AABE` ＋ `jZiXE`「自動處理免費字幕」`#22C55E` 12/600；關閉態 specimen `Ap0rZ` 無末段 | ✅ |
| #4 行動版 | 兩段說明**一字未刪**（`Ydcjx` bh=38、`qTYWW` bh=38）；按鈕等寬 | ⚠️ **見 MUST-FIX ①** |
| #5 規格頁 | `sPzZT` 獨立 frame 1240×1217，七個區塊齊備（裁定引用／兩層對照／控制項狀態／卡片 footer／文案理由／E1 落差警示） | ✅ |
| 溢出 | 全樹 `ctx.problems` **零**筆（三個 frame 逐節點掃描） | ✅ |
| 標籤重疊 | caption `zgGpe`(y=10475)／`tsJCd`(y=11485) 距 frame **45px**；`b6UOEN`(y=24270) 距 **30px**（同 J1/J2 慣例）；皆 `#888888` 14/600 | ✅ |
| 版面淨空 | J4-D x=19720..20960（J2-D 右緣 19620，留 100）；底緣 25517，下方 `A2p-D`/`A3p-D` 在 y=27501，留 **1984px** | ✅ |
| 元件複用 | 未新增 component；`4EHFN`／`Wd9AL` 以 `ref` 引用 | ✅ |
| 截圖 | 僅 3 張新增（e5-d / e5-m / j4-d），**其餘 153 張未進 commit** | ✅ |
| SCREENS dict | 三個 node id 已登錄，含說明註解 | ✅ |
| `.pen` 落盤 | commit 內 `ux-design.pen` 實際變更 +1669 行 → 存檔已生效 | ✅ |

### 🔴 MUST-FIX ① —— E5-M 開關列觸控目標 33px（AC #4 要求 ≥44px）

- 實測：`eIgZf` 解析後 `bh=33`（checkbox 20 ＋ padding `[6,0]` = 21+12）。
- **成因是我的提示詞算錯**：我寫「用 `padding=[6,0]` 補足到 44」，但 21+12=33。Alexyu 逐字照做，執行無誤。
- 修法：`eIgZf` 的 `padding` 改為 `[12,0]` → 21+24 = **45px** ≥ 44。單一屬性，視覺幾乎不變（上下各多 6px 呼吸）。
- 桌面版 `y6gn3` 維持無 padding —— 44px 觸控目標只約束行動版，滑鼠不需要。

### 📝 記錄：提示詞收尾檢查第 5 條過寬（非 finding，供後續審查者免重吵）

我在提示詞寫「全部三個畫面裡沒有出現『掃描』二字」，但**我自己口述的 J4-D 區塊 B／F 內容就必須含該詞**
（`I18XW` 引用 2026-08-07 事故、`GH05m` 陳述規則本身）。
AC #1 的真正約束是**使用者可見文案**不得暗示掃描會產生字幕 —— E5-D／E5-M／J4-D 區塊 D specimen 皆為 0 命中，**規則成立**。
規格頁的後設敘述使用該詞是正確且必要的。

### ✅ 二輪複驗（2026-08-20，MUST-FIX ① 修後）

| 檢查 | 結果 |
|---|---|
| `eIgZf` padding | `[12,0]` → 解析高度 **45px** ≥ 44 ✅ |
| `y6gn3`（E5-D 對應節點） | `padding=undefined`、高度 21px —— **未動** ✅ |
| 定稿文案 | E5-D 標籤×1／說明1×1／說明2×1；E5-M 同；J4-D 標籤×2（off/on 兩 specimen，符合預期）＋說明各×1 ✅ |
| 含「掃描」 | E5-D **0**／E5-M **0**／J4-D 2（區塊 B 事故引用＋區塊 F 規則陳述，正確且必要）✅ |
| `ctx.problems` | 三個 frame 全樹 **零筆** ✅ |
| 修正 commit 範圍 | `433d013e` = 2 檔（`ux-design.pen` **+1/−1**、`e5-m.png`）—— 一個屬性，零附帶變更 ✅ |
| PR #246 總檔案 | 5 個（3 PNG ＋ 匯出腳本 ＋ `.pen`），無多餘檔案 ✅ |

**判定：PASS。設計交付完成。**

### 看過、決定不改

- **E5-M 開關標籤維持 14/500**（未隨其他欄位降到 12/13）—— 刻意：它是同意行為的主體，比欄位標籤重是對的。
- **E5-D 開關列無 44px padding** —— 見上，桌面不需要。
- **J4-D 卡片 specimen 578px 而非 580px** —— Alexyu 自行改為 `fill_container` 平分以消除 4px 溢出。這是正確的判斷，追認。


## Dev Notes

### 這是設計 story，不是開發 story
- 交付物是 `.pen` 畫面＋截圖，**零程式碼**。`9R-10b` 的 Task 5 才是實作。
- 完成定義（UX Design Story Status 慣例）：`done` = 設計交付完成，**不代表** FE 已實作。

### 給 Sally 的收斂建議（是傾向，不是裁定）
- SM 傾向：**checkbox**（modal 內其他控制項都是樸素的原生表單元素，toggle switch 會是 modal 裡唯一的異類）。
- SM 傾向：說明文字**兩句**——一句講做什麼（免費的自動做完），一句講不做什麼（要花錢的留著等你確認）。
- SM 傾向：卡片上**顯示**狀態（否則 Finding 4 的「隱形布林」歷史會重演）。
- 以上三項 Sally 都可以推翻，但推翻要在畫面上留註記說理由。

### 關鍵檔案（設計判讀用，不修改）
| 用途 | 路徑 |
|---|---|
| 開關要加進去的 modal | `apps/web/src/components/settings/LibraryEditModal.tsx` |
| 卡片（Finding 3） | `apps/web/src/components/settings/LibraryCard.tsx` |
| 唯一呼叫端（Finding 1） | `apps/web/src/components/settings/ScannerSettings.tsx` |
| 免費/付費語彙來源 | `apps/web/src/components/subtitle/consent/`（F14–F20） |
| 裁定原文 | `apps/api/internal/cost_consent_test.go`（檔頭） |
| 免費/付費的真實分界 | `apps/api/internal/subtitle/router.go:289` `routeForVariant` |

### Time-dependent visual coverage
- **N/A —— 設計 story，零 `apps/web` 程式碼變更。**

### References
- [Source: _bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md] —— 母 story，AC #1 裁定表與 AC #2 欄位設計
- [Source: apps/api/internal/cost_consent_test.go] —— 2026-08-07 事故與裁定原文
- [Source: _bmad-output/implementation-artifacts/sub-4-3-cost-consent-frontend.md] —— F14–F20 語彙、AC #6「不得暗示掃描會產生字幕」
- [Source: _bmad-output/implementation-artifacts/9R-UX-episode-row-cta-design.md] —— 前一個 UX story 的格式與 MCP review 慣例
- [Source: .claude/memory/feedback_pen_inline_agent_workflow.md / feedback_pencil_spec_standalone_screen.md / feedback_pencil_label_overlap.md / feedback_verify_pen_saved_before_commit.md / feedback_identical_rendering_is_sally_decision.md]

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Completion Notes List

- **交付**：`E5-D`(`hUVYm`)／`E5-M`(`P0P82x`)／`J4-D`(`sPzZT`) 三張新畫面，PR #246（2 commits）。
- **三個裁定**全部落地並經 MCP 逐字複驗（見「🎨 Sally 的三個裁定」與「🔍 Sally MCP Review」）。
- **裁定 2 的範圍效果**：開關留在 per-library modal 內 ⇒ **9R-10b 的 FE 範圍不放大**，維持「一個 checkbox」。
- **一輪 MUST-FIX 的責任歸屬**：E5-M 觸控高度不足源自**本人提示詞的算術錯誤**（`padding=[6,0]` 只給 33px），
  非執行瑕疵；已於二輪修正並複驗 45px。
- **提示詞收尾檢查第 5 條過寬**（「三個畫面都不得出現『掃描』」）—— 規格頁的後設敘述必須用該詞，已記錄免後續重吵。
- **Alexyu 於執行中自行修正的 4px 溢出**（J4-D 卡片 specimen 580×2 → `fill_container` 平分 578）：正確判斷，**追認**。
- **⚠️ 新增的存檔坑已進記憶檔**：Pencil 的 ⌘S keystroke 會寫出「mtime 更新但內容是舊快照」的檔案；
  可靠做法是先 `Update()` 標髒再用 AppleScript 點 File → Save，且驗證要 grep 磁碟檔的**實際新值**而非只看 `git status`。
  已補入 `feedback_verify_pen_saved_before_commit`（2026-08-20 補充節）。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **③ `drift-e1-scanner-multi-library`** —— E1-D/E1-M 畫扁平資料夾列，已出貨介面是媒體庫卡片＋編輯 modal，
    設計落後一個世代。非阻塞（新畫面依程式碼繪製並在 J4-D 區塊 G 標註落差）。
  - **③ `bugfix-libraryeditmodal-wrong-design-ref`** —— `LibraryEditModal.tsx:1` 的
    `// Design ref: ... Screen 10 Settings Desktop (6UCtX)` 指向的是**連線設定頁**，Rule 21 traceability 錯誤。
    本 story 交付 E5-D 後即有正確 node 可指，修正併入 9R-10b Task 5。
  - **裁定 2 未觸發範圍放大** —— 開關留在 modal 內，9R-10b 的 FE 範圍維持「一個 checkbox」。

### File List
