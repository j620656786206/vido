# ux3-ai-1 — AI 生成工作區設計 prompt（給 Pencil In-App AI agent）

> 把下方區塊貼進 Pencil.app 的 in-app AI agent，畫 Epic 7（ux3-ai-subtitle）的**生成工作區**。
> 之後由 ux-designer（Sally）用 Pencil MCP 讀圖 / 截圖 / 定點修正 review — 比整條流程都用
> MCP 畫省 token。同路徑：ux3-1-1 / ux3-3-1 / ux3-4-1 / 9R-UX。
> Story 規格：`ux3-ai-1-workspace-design.md`（AC #1–#8）。**能力表以該檔 Dev Notes 為準。**

---

你正在替 Vido NAS 媒體 app 的 **AI 字幕生成** 收尾一個 **全頁沉浸式工作區**（v2 設計語言）。在現有
`ux-design.pen` 內工作。這是既有 `flow-f-subtitle-v2` 區塊的延伸 —— **F1–F10 字幕 dialog 流已上線
（Epic 6）**，這關只補 Epic 7 缺的那塊：**生成工作區**。

**一句話 brief：** 使用者對 38 部缺繁中字幕的清單按下「批次生成」後，需要一個**能離開再回來、關掉不會
消失**的全頁家 —— 佇列進度、即時活動、成本/預算，全部活生生地跑。今天所有生成可見性都被困在 dialog 裡
（關掉就什麼都沒了）。這個工作區就是那個家。

## ⚠️ 起點是 F11 探索稿，但要「重塑」它，不是照抄

現有 `F11-D-v2`（節點 `l8FsB`，x:47590 y:-5746，1440×900）是 inline agent 當初自發畫的
**單一任務·管線工作區**：左 `stage-rail`（階段列 300px）＋ 右 `transcript-pane`（**即時逐字稿**串流）。
party-mode P4 已裁決保留為 Epic 7 起點（見其下方琥珀註記 `rhhQ0`）。**在 `l8FsB` 原地改**成正式規格，
遵守以下兩個硬重塑：

1. **右欄不是逐字稿，是「即時事件日誌」。** 🔴 能力誠實：後端 `transcription_*` SSE 只送
   `{phase, message, percentage}`，**不送逐字對白內容** —— 畫串流對白等於畫一個不存在的能力。把
   `transcript-pane` 重塑成 **即時事件日誌**：每列 = [階段圖示 ＋ 項目標題 ＋ 訊息]，依到達順序累積
   （如 `翻譯中 · 慶餘年 S01E05 · 45%`、`完成 · 沙丘：第二部`、`本次用量 $0.42`）。標註
   `自開啟本頁起累積`＋`即時更新（SSE）`chip。**不畫任何時間戳**（SSE payload 無時間；Rule 23）。
2. **工作區以「批次」為主，單一任務是它的特例。** F11 原本只畫一部片。真正的主戲是 **N 部的批次佇列**。
   左欄改成 **生成佇列**（多列項目：完成✓ / 轉錄中＋階段 stepper / 排隊中 / 已暫停 / 失敗）。同一版面
   在只有 1 列時，就是「詳情觸發的單部生成」case（標註之）。

新畫的 frame（caption 文字在每個 frame 上方，Noto Sans TC 14/600 `#888888`，離 frame ~45px 避免撞
Pencil frame-name chrome，格式對齊同區 F8/F9 caption）：

- **`F11-D-v2 · 生成工作區 — 執行中`**（**改 `l8FsB` 原地**）—— 主畫面 hero。
- **`F11-M-v2 · 生成工作區（手機）`**（新）—— 390px。
- **`F12-D-v2 · 生成工作區 — 預算上限終態`**（新）—— F9 語意逐字。
- **`F13-D-v2 · 生成工作區 — 閒置／中途接入／失敗`**（新）—— 三個稀疏狀態併一框（spec-style）。

用 `FindEmptySpace`（anchor `nodeId: l8FsB`, direction `right`）把 F12/F13 放在 F11 右側續排；
F11-M 放同區手機列（在 F8-M/F3-M 那一排，或 F11 下方、避開琥珀註記 `rhhQ0`）。每個新 frame 建置期間設
`clip: true` + `placeholder: true`，完成一個就移除該 frame 的 `placeholder`。

## ⛔ 不要碰（保持 byte 不變）

- 既有 F1–F10 dialog frame（Epic 6 已上線）、F8-D `i9Nun1`、F9-D `JMqPg`、F8-M `H717g`、
  舊 `F*`/`G*` fetch-era frame。**只改 `l8FsB`，其餘只新增。**
- 琥珀註記 `rhhQ0`（F11 探索說明）不要刪；它現在仍適用（探索→本 story 正式化）。

## 各畫面組成

### F11-D-v2（改 `l8FsB`）—— 執行中（hero）

保留：`sidebar`（`hY6lI`，BDeUS 實例，**活動 active** = accent-tint/accent-text，維持現狀）。
重塑 `main`（`L4RCY`）內容：

- **page-header（`jxF9O`）：**
  - breadcrumb（`O6OQNE`）：`活動` ＞ `生成字幕`（hosted-in-Activity — 見 IA 裁定）。
  - title-row（`BWV0W`）：`批次生成字幕`（單部 case 時 = `生成字幕 · {片名}`）。
  - meta-strip（`whlu3`）改成**總覽列**：`已完成` [Mono 12] `/` [Mono 38]（部字 Noto，TY-3 分割）＋
    細進度條 ＋ `本次用量：` [Mono `$0.42`] ` / 上限 ` [Mono `$5.00`] ＋ `即時更新（SSE）`chip
    （`$info-tint`/`$info` 小圓點）。
- **body（`dAHkG`）兩欄：**
  - **左＝生成佇列（主，寬 ~700 / fill）** —— 取代 `stage-rail`。頂部一列：`全部取消`
    （`Component/ButtonSecondary` `YDPhc`，內嵌確認 per F8 慣例；**只有整批取消，無單項取消**）。下方
    佇列列（poster 縮圖 40×60 ＋ 片名 ＋ 狀態）：
    - 進行中列 = 內嵌 `Component/GenerationProgress-v2`（`XkGvG`）階段 stepper
      `提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → AI校正 → 完成`（凍結 6 段，勿改名/增減）。
    - 完成列 = `$success` 勾；排隊列 = `$text-muted` `排隊中`；失敗列 = `$error-text` `失敗`；
      已暫停列 = `$warning`底 `已暫停 — 下次繼續`（僅預算上限時，見 F12）。
    - 列的 idiom 靠攏 `Component/ActivityRow-v2`（`fF8nX`）—— 別自創第二種任務列長相；若佇列列的
      內嵌 stepper 讓它結構上顯著不同，才建新 `Component/GenQueueRow-v2` 並註冊到 `sJzat`。
  - **右＝即時事件日誌（副，寬 ~400）** —— 取代 `transcript-pane`。標題 `即時活動` ＋
    `自開啟本頁起累積` 小注（`$text-muted`）。事件列（依到達順序，新的在下）：階段圖示（lucide/phosphor
    ＋ `$accent-text`/`$success`/`$error-text`）＋ 項目標題（Noto）＋ 訊息/百分比（數字 Mono）。
    **無時間戳。** 底部一個 `即時更新（SSE）` chip。
- **單部任務 case（標註即可，不另開 frame）**：在佇列上方加一條琥珀小註
  `單部任務（詳情觸發）在此以單列佇列呈現；別處啟動的任務要等下一個 SSE 事件才可見（BE 缺口
  disc-2026-07-transcription-active-jobs）`。

### F12-D-v2（新）—— 預算上限終態

複製 F11-D 的 shell，把狀態切到 `budget_ceiling`（**成功語意，非錯誤 token**）：

- 總覽列上方一條 banner（`$warning-tint` 底、`$warning` circle-alert 圖、`$warning`/`$text-primary` 字，
  **逐字**）：`已達本次預算上限（` [Mono `$5.00`] `）— 已完成` [Mono 12] `部，剩餘` [Mono 26]
  `部下次繼續`。
- 佇列：已完成列 `$success` 勾；剩餘列 `已暫停 — 下次繼續`（`$warning`）。
- footer 動作：`關閉`（`ButtonSecondary`）＋ **`下次繼續`**（`ButtonPrimary` `otvKh` —— = 重開一個
  `scope=missing` 批次，已完成項自動排除；resume-for-free）。
- 加一條琥珀小註：`complete / cancelled / error 三個終態沿用此版面，只換 token（success / muted /
  error-tint）與文案（全部完成 / 已取消 / 批次發生錯誤）；不另畫 frame`。

### F13-D-v2（新）—— 閒置／中途接入／失敗（三稀疏狀態併框）

同 shell，body 放三個各自標籤的面板（spec-style 上下堆疊，各自 caption）：

1. **閒置／空**：`目前沒有進行中的生成`（`$text-secondary`）＋ Mono 缺字幕預覽數
   `缺繁中字幕：` [Mono 38] `部` ＋ 主行動 `批次生成字幕`（`ButtonPrimary` —— **打開 F8 dialog**，
   工作區不自建 scope 選擇，dialog 才是啟動器，見 IA 裁定）。
2. **中途接入（attach-degraded）**：標題 `已接上進行中的批次` ＋ 只畫 **總覽計數 ＋ 成本 ＋ 進行中單列**，
   佇列其餘以骨架/佔位帶出，一條註 `佇列明細自本頁開啟起顯示（BE 缺口 disc-2026-07-generation-batch-
   status-items：status 探測無 items[]）`。**誠實**，別假裝有完整佇列。
3. **fail-soft 錯誤**：工作區自身資料（活動/預覽）載入失敗 → inline `無法載入生成狀態` ＋ `重試`
   （`role=alert` 語意）；整頁絕不 hard-fail。

## ⚠️ 能力誠實（畫為目標狀態，把缺口標清楚）

在新區塊內加 **一個琥珀 design-note frame**（樣式同 `rhhQ0`：`fill #F59E0B15`、`stroke #F59E0B40`、
radius 6、Noto Sans TC 12/600 `#B45309`），逐條列：

```
BE 能力邊界（本工作區嚴格遵守，違反的控制一律不畫）：
· 無「暫停/續傳」—— 後端任何地方都沒有 pause 端點。
· 取消 = 整批（POST …/generation-batch/cancel）；無單項取消、無單部轉錄取消、無部分取消。
· 無單項重試（失敗只計數、迴圈續跑）。
· 影集不支援（9R-10a 未 merge；批次恆為電影）。
· 成本只來自批次 SSE spent_usd / budget_usd —— 無 HTTP 成本/用量端點（9R-17 backlog）；
  預算為 env-only（AI_RUN_BUDGET_USD，預設 $5），不畫可編輯的預算控制。
· 無歷史 / 無終態重探；status 探測終態後回 {running:false, progress:null}。
· 右欄是事件日誌，非逐字稿（SSE 不送逐字內容）。
· 單部任務載入時可見性 gated on disc-2026-07-transcription-active-jobs（active_jobs 無 transcription kind）。
```

## IA 裁定（已定，畫面要表達出來）

- **hosted-in-Activity**：工作區住在 `活動` destination 內（sidebar 活動 active、breadcrumb 活動＞生成字幕），
  **不是新導覽項**。前端機制 = `/activity?view=generation`（想要清單 `?view=requests` 前例）。
- **dialog = 啟動器，工作區 = 觀看者**：F8/F9 dialog 維持為「開始批次」的啟動器；工作區是離得開、回得來的
  「觀看」場所。閒置狀態的 `批次生成字幕` 主行動打開 F8 dialog；活動中心的 `generation_batch` 列 →
  連到工作區。**不要**在工作區內重建 scope 選擇，也不要畫第二個和活動中心競爭的入口（D4-1）。
- **詳情交叉連結**：F3 生成進度 dialog 內加一個小連結 `前往生成工作區`（單部任務可「彈出」到工作區觀看）。

## 設計語言 v2（全面套用）

- **字型（零容忍）**：所有 CJK＝**Noto Sans TC**；DM Sans 只給 `vido` logo / 純英文 display；**所有數字＝
  JetBrains Mono**（%、N/M、$、計數、預算）。混合字串 `12 部`／`38 部`／`26 部下次繼續` 一律拆成
  Mono 數字節點 ＋ Noto 單位節點（gap 3–4，TY-3）。`$5.00`／`$0.42`／`45%` 的 `$`/`.`/`%` 留在 Mono 節點內。
- **顏色＝只用 token，無 hex 字面**（兩個琥珀 note frame 是唯一例外 —— 它們是畫布註記非 UI）。可用變數：
  `$bg-primary #1B2336` · `$bg-secondary #24304A` · `$bg-tertiary #2E3B56` · `$text-primary` ·
  `$text-secondary` · `$text-muted` · `$text-disabled`（勿承重）· `$accent-primary` · `$accent-text` ·
  `$accent-subtle` · `$accent-tint` · `$success`/`$success-tint` · `$warning`/`$warning-tint` ·
  `$error`/`$error-text`/`$error-tint` · `$info`/`$info-tint` · `$border-subtle` · `$overlay-scrim` ·
  `$radius-sm|md|lg|xl` · `$gap-xs|sm|md|lg|xl|2xl` · `$text-on-accent`。生成狀態沿用 DL-v2 §2.5
  status→token 對照（和生命週期徽章同一套，無自訂調色盤）。
- **彩色內文用 `*-text` AA 變體**（`$accent-text`/`$error-text`），不用 base 色相當內文。
- **觸控目標 ≥ 44×44px**（按鈕、列動作、chip、關閉 ✕）。
- **間距**：區段節奏 24–48；元件內 8–12；平面 chrome（`$border-subtle`），避免過度陰影/圓角。

## 手機 F11-M-v2（390px）

沿用桌面內容順序，直向堆疊：page-header 總覽（N/M ＋ 成本 ＋ SSE chip）→ **生成佇列**（列全寬、
44px、進行中列 stepper 直向堆疊）→ **即時事件日誌**（可折疊區塊，預設收合以省空間）。`全部取消` 置底或
在標題列。底部 tab bar：bottom-4 `首頁 · 媒體庫 · 活動 · 下載` ＋ More（`Component/MobileTabItem`
`S86VM`）—— 生成字幕不是 tab，是活動內的一個 view。

## 要 REUSE 的元件（別重造）

- `Component/GenerationProgress-v2`（`XkGvG`）—— 進行中佇列列的階段 stepper（凍結 6 段）。
- `Component/ActivityRow-v2`（`fF8nX`）—— 佇列列/日誌列的 idiom 基準。
- `Component/ButtonPrimary`（`otvKh`）/ `ButtonSecondary`（`YDPhc`）—— 別 fork 新按鈕。
- `Component/HomeSidebar-v2`（`BDeUS`）—— 已在 `l8FsB` 內（活動 active），維持。
- `Component/MobileTabItem`（`S86VM`）—— 手機 tab bar。
- 若真的要新元件（`Component/GenQueueRow-v2`），先建成 top-level reusable、再 instance，並**註冊到
  Component Library frame `sJzat`**（progress-v2 列 `luza9` 或 content-cards-v2 列；cell = 直向 frame
  gap 8，ref 實例 ＋ Noto Sans TC 12 `$text-muted` caption）。

## 完成檢查清單

- 只新增/改 `l8FsB`；F1–F10 / F8-D `i9Nun1` / F9-D `JMqPg` / 舊 F*/G* 全 byte 不變；`rhhQ0` 保留。
- 右欄是**事件日誌非逐字稿**；無時間戳；`自開啟本頁起累積` ＋ SSE chip 在。
- 佇列狀態齊全（進行中＋stepper / 完成 / 排隊 / 已暫停 / 失敗）；`全部取消` = 整批、無單項控制。
- F12 預算上限 = 成功語意（warning-tint banner 逐字、下次繼續），非錯誤 token；complete/cancelled/error
  以註記帶出。
- F13 三稀疏狀態齊：閒置＋預覽數＋啟動器、中途接入誠實（no items[]）、fail-soft 重試。
- **無暫停/續傳/單項重試/歷史/預算編輯控制**；琥珀能力邊界 note frame 在，含三個 disc/9R-17/9R-10a 缺口。
- IA 三裁定表達：breadcrumb 活動＞生成字幕、啟動器 vs 觀看者、F3 `前往生成工作區` 連結。
- 全 CJK Noto Sans TC、全數字 Mono（混合字串拆節點）、只用 token、AA `*-text`、44px、無裁切、caption 不撞
  frame chrome。
- 新元件（若有）已註冊 `sJzat`；每個完成的 frame 已移除 `placeholder`。

---

### Inline agent 畫完後（ux-designer via MCP — token-light）

1. `get_screenshot` 每個新/改的 `F11–F13` frame；對照本清單＋DL-v2＋story AC #1–#8。
2. 定點 `batch_design` 修正（不重畫）：hex→token、字型滑落（CJK 落 DM Sans / 數字未拆單位）、逐字稿殘留
   （右欄必須是事件日誌）、缺狀態、44px、能力違反控制（暫停/重試/單項取消/可編輯預算）、琥珀能力 note ＋
   IA 裁定表達、`sJzat` 註冊 cell。
3. 確認 `i9Nun1`/`JMqPg`/舊 F*/G* byte 未動；`rhhQ0` 還在。
4. **Pencil.app 手動 Cmd+S**（MCP/inline 編輯需手動存），更新 `scripts/export-pen-screenshots.py`
   `SCREENS`（新節點 → `("flow-f-subtitle-v2","f11-d-v2")` / `f11-m-v2` / `f12-d-v2` / `f13-d-v2`）→
   `python3 scripts/export-pen-screenshots.py` → 只 stage **真正改動的 PNG**（regen 非確定性）→ 新分支
   off main：`feat(ux3-ai-1): AI generation workspace design — F11 validated spec (.pen flow-f-subtitle-v2)`。
5. 填 story `ux3-ai-1-workspace-design.md` 的 Dev Agent Record（節點 id、決策、偏離）→ status `review`；
   CR/merge 後於 sprint-status 設 `ux3-ai-1-workspace-design` → `done`（並解鎖 `ux3-ai-2` 的 STOP gate）。
```
