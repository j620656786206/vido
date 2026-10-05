# Story infra-optin-usage-report-design: 匿名使用回報的設計稿——精靈多一步、設定頁多一張卡

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** standalone（`infra-optin-usage-report` 家族）· **Priority:** P1 · **Size:** S（設計 only）
**Owner:** ux-designer（Sally）出設計裁定＋節點錨定提示詞 → Alexyu 在 Pencil Inline AI Agent 執行 → Sally 用 MCP 唯讀 review（`feedback-pen-inline-agent-workflow`）。
**Source:** PRD amendment `prd/prd-telemetry-amendment.md`（P1-040-1~3）；後端已上線：`infra-optin-usage-report-a2-reporter`（PR #665，`GET`／`PUT /api/v1/settings/usage-report`、`SetupConfig.usage_report_enabled`、`ValidateStep("usage-report")`）。
**Blocks:** `infra-optin-usage-report-b-frontend`

---

## Story

As a Vido user deciding whether to help the maintainer,
I want the setup wizard and the settings page to show me plainly what would be sent, what never is, and exactly what was sent last time,
so that turning the report on is an informed choice and never a surprise.

---

## Context —— 查到的事（2026-10-04，Sally 以 Pencil MCP 唯讀查證）

- **精靈現有五步**：N1-D `dzgq9`／N2-D `CP7AX`／N3-D `TyjL0`／N4-D `D990CP`（API 金鑰）／N5-D `CWh3E`（完成），都在 `XBRn1`（`N · 桌面 Desktop`）。只有 N3 有手機稿（N3-M `YyaqL`，在 `eYWUL`）。
- **步驟點**：每張的 `step-progress` 是 5 個點＋4 條線（N4-D `B8JksM`：dot-0..3 與 bar-0..2 為 `$accent-primary`，bar-3 與 dot-4 為 `$bg-tertiary`；N5-D `UxNGo` 全部 `$accent-primary`）。程式碼 `StepProgress.tsx` 由 `WIZARD_STEPS` 決定點數，b 加一步後會變 6 個點——**設計稿五張都要跟著變 6 個點**，否則前端驗收會對不上。
- **N5 摘要列**：`xuXAV` 底下五列，最後一列 `row-Claude 金鑰` `qr7Q7`（左 `k`＝`$text-secondary` Body，右 `v`＝已設定 `$text-primary`／未設定 `$text-muted`）。
- **設定分頁列已滿 12 個**（`SettingsLayout.tsx:212-215` 註解：1440 剛好塞下）→ 不加第 13 個分頁。
- **連線設定頁**：C4-D `6UCtX`、往下捲的 C23-D `Qva0y`（Sonarr 卡 `d9AmRx`、Radarr 卡 `Q3IJv`，卡寬 768，`row-enable` 範本 `hCipa`＝左「標題＋提示」、右 `Component/Switch/On` `jqRCX`）；手機 C4-M `2H4OM`、C23-M `p37q9`（在 `v21FYb`）。
- **元件**：`Component/Switch/On` `jqRCX`、`Component/Switch/Off` `Qia26`；沒有 Disabled 開關母版。

---

## 設計裁定（Sally，2026-10-04）

| # | 裁定 | 理由 |
|---|---|---|
| D1 | **設定頁：放在「連線設定」最底下，一張新卡「匿名使用回報」**，不新增分頁 | 分頁列已滿；「連線設定」是「Vido 會連到哪些外部服務」，這是 Vido 唯一一條主動往外送的連線，放這裡最誠實。⚠️ **待 Alexyu 確認**（見文末問題） |
| D2 | **精靈：在「API 金鑰」與「完成」之間插入一步**，成為 6 步中的第 5 步；預設關閉 | 決定要在「看完摘要」之前做；放第一步會像是條件交換 |
| D3 | 精靈這一步**只有「上一步／下一步」，不放「跳過」** | 開關本身就有安全預設（關閉），「跳過」與「下一步」結果相同，留兩個按鈕只會讓人以為有差 |
| D4 | 「會送」與「絕不送」**並列兩個方塊**，絕不送的方塊在右 | 使用者最在意的是不送什麼；兩欄對照一眼看完，不用讀段落 |
| D5 | 「送出的內容（原文）」用**等寬字、原樣一行、可換行**的方塊顯示 | DESIGN.md「比較才用等寬」：這是要讓人跟文件範例逐字對照的讀數 |
| D6 | **三種狀態另開一張規格稿**（不可用／關閉／已開啟但還沒送過） | `feedback_pencil_spec_standalone_screen`：狀態不塞進主稿 |
| D7 | 「不可用」狀態：開關用 `Switch/Off` instance 加 `opacity: 0.4`，旁邊說明原因 | 沒有 Disabled 母版；用 instance override，不手畫 |
| D8 | 金額、狀態色都不用；整張卡維持中性色 | 這不是警告也不是成功，只是一個選擇 |

### 定稿文案（zh-TW）

**精靈 N6（步驟 5／6）**
- 標題：`匿名使用回報`
- 說明：`要不要每週告訴維護者「Vido 有在幫你做字幕」？只送幾個數字，預設關閉。`
- 開關列：標籤 `每週送一次匿名計數`／提示 `之後隨時可以在「設定 → 連線設定」改。`／`Switch/Off`
- 「會送」方塊標題 `會送`，三行：`一個隨機編號（不是從你的機器算出來的）`、`Vido 版本`、`最近 7 天 Vido 自己做出的字幕數`
- 「絕不送」方塊標題 `絕不送`，三行：`片名、檔名、資料夾路徑`、`API 金鑰與任何設定`、`你看了、想要或下載了什麼`
- 連結：`看完整說明 →`
- 按鈕：`上一步`／`下一步`

**N5 摘要新增一列**：左 `匿名使用回報`，右 `關閉`（`$text-muted`，同「未設定」樣式）

**設定頁卡片（開啟、已送過＝主稿 C25）**
- 卡片標題 `匿名使用回報`，副標 `每週最多一次，把幾個匿名數字送給 Vido 維護者。`
- 開關列：標籤 `每週送一次匿名計數`／提示 `預設關閉。關掉之後就不會再送。`／`Switch/On`
- 讀數列：標籤 `上次送出`，值 `2026-10-04 12:00`（等寬）
- 原文：標籤 `送出的內容（原文）`，方塊內容（等寬、可換行）：
  `{"type":"event","payload":{"website":"11111111-2222-3333-4444-555555555555","hostname":"vido","url":"/usage-report","name":"weekly_usage","id":"3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60","ip":"127.0.0.1","data":{"version":"0.1.2","subtitles_auto_7d":7,"subtitles_embedded_7d":4,"subtitles_online_7d":2,"subtitles_asr_7d":1}}}`
- 連結：`送什麼、不送什麼 →`

**三種狀態（規格稿 C26）**
- 不可用：開關 `Switch/Off`（`opacity 0.4`），提示改為 `這個版本沒有設定接收端，無法開啟。`；沒有讀數列與原文
- 關閉（預設）：開關 `Switch/Off`，提示 `預設關閉。關掉之後就不會再送。`；沒有讀數列與原文
- 已開啟、還沒送過：開關 `Switch/On`；讀數列值 `還沒送過——打開後一小時內會送出第一份`（`$text-muted`）；沒有原文方塊

---

## Acceptance Criteria

1. **N6-D** 新增於 `XBRn1`，在 N5-D 右邊一欄；畫布標題 `N6 · 匿名使用回報（步驟 5／6，在 N5 完成之前）（桌面）`。內容照「定稿文案」；只有上一步／下一步。
2. **N1-D～N5-D 與 N3-M 的步驟點都改為 6 個點＋5 條線**，顏色規則不變（已完成＝`$accent-primary`，未到＝`$bg-tertiary`）；N6-D 亮到第 5 個點；N5-D 全亮。
3. **N5-D 摘要**在「Claude 金鑰」下新增「匿名使用回報｜關閉」一列（`Copy` `qr7Q7`）。
4. **C25-D**（`szz7O`，C24-D 右邊一欄）：連線設定往下捲，看得到 Radarr 卡底部與新卡；**C25-M**（`v21FYb`，C23-M 右邊）同內容手機版，root frame 有 `theme:{bp:"mobile"}`。
5. **C26-D**：三種狀態並排的規格稿，各附一行小標。
6. 全部只用 `$` 變數（顏色、`Space/*`、`Type/*` 三件組）；開關一律用 `jqRCX`／`Qia26` instance；不手畫任何既有元件（DESIGN.md SOP §3–6）。
7. 收工檢查（DESIGN.md SOP §7）：存檔落盤、裁切警告數不高於基準、日巡截圖無黑字壓黑底、最外層與同層重疊 0。
8. `scripts/export-pen-screenshots.py` 的 `SCREENS` 補 N6-D、C25-D、C25-M、C26-D（先數重複 key）；只 stage 真正改動的 PNG：`n1-d`～`n6-d`、`n3-m`、`c25-d`、`c25-m`、`c26-d`。

---

## Tasks / Subtasks

- [x] Task 0: Sally 查證現況、作設計裁定、寫提示詞（本檔）
- [x] Task 1: Alexyu 執行提示詞 1（設定頁 C25-D／C25-M）——commit 37abbc89；Sally review 通過（見 Completion Notes）
- [ ] Task 2: Alexyu 執行提示詞 2（規格稿 C26-D）
- [x] Task 3: Alexyu 執行提示詞 3（精靈 N6-D）——commit da43f527；Sally review 通過
- [ ] Task 4: Alexyu 執行提示詞 4（N1–N5、N3-M 步驟點＋N5 摘要列）
- [ ] Task 5: Sally MCP review（逐字比對文案、`ctx.problems`、重疊、日巡）
- [ ] Task 6: 存檔驗證 → 匯出截圖 → `SCREENS` → 只 stage 真改動 → commit

---

## 給 Pencil Inline AI Agent 的提示詞（照順序貼；每段自包含）

### 提示詞 1 — 設定頁 C25-D／C25-M

```
在 ux-design.pen 做兩件事，全部只用設計系統變數（顏色 $…、間距 $Space/…、字級 $Type/<角色>/Size + /Line + /Weight 三個一起設），不准寫死 hex 或數字間距。

A. 桌面：Copy 畫面 Qva0y（C23-D）到群組 szz7O 裡，放在 C24-D（L7EVR）右邊一欄（x = C24-D 的 x + 1540，y 與 C24-D 相同），圖框名稱改為 "C25-D"。
   在它上方 45px 放一個畫布標題 text：「C25-D · 設定 — 連線設定（往下捲）· 匿名使用回報（桌面）」，字體與樣式完全比照隔壁 C24-D 的畫布標題（字體變數 $Type/Family/Canvas）。
   在這個副本裡：
   1. 刪除 Sonarr 卡（原 d9AmRx 的副本），保留 Radarr 卡，讓畫面看起來是捲到頁面最底。
   2. 在 Radarr 卡後面新增一張卡，名稱 "usage-report-card"，外觀完全比照 Radarr 卡（寬 768、同樣的底色／圓角／邊框／內距／間距）：
      - card-head：標題「匿名使用回報」（同 Radarr 卡 title 樣式），副標「每週最多一次，把幾個匿名數字送給 Vido 維護者。」（同 sub 樣式）。不要 health 徽章。
      - row-enable：比照 Radarr 卡的 row-enable，標籤「每週送一次匿名計數」，提示「預設關閉。關掉之後就不會再送。」，右邊放 Component/Switch/On（jqRCX）instance。
      - row-last-sent：左右排列，左標籤「上次送出」（$text-secondary、Type/Body），右值「2026-10-04 12:00」（$text-primary，字體 $Type/Family/Mono，Type/Body）。
      - payload：垂直排列，標籤「送出的內容（原文）」（$text-secondary、Type/Body），下面一個方塊（fill $bg-primary、圓角 $radius-md、內距 $Space/md），方塊內一段 text，字體 $Type/Family/Mono、Type/Label 三件組、$text-secondary、寬度 fill_container 可自動換行，內容一字不改：
        {"type":"event","payload":{"website":"11111111-2222-3333-4444-555555555555","hostname":"vido","url":"/usage-report","name":"weekly_usage","id":"3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60","ip":"127.0.0.1","data":{"version":"0.1.2","subtitles_auto_7d":7,"subtitles_embedded_7d":4,"subtitles_online_7d":2,"subtitles_asr_7d":1}}}
      - 最後一行連結文字「送什麼、不送什麼 →」，$accent-text、Type/Body。
   3. 調整圖框高度讓新卡完整露出、底部留 $Space/2xl 以上空白，不要有內容被裁切。

B. 手機：Copy 畫面 p37q9（C23-M）到群組 v21FYb，放在它右邊（x = C23-M 的 x + 490，y 相同），圖框名稱 "C25-M"，root frame 保留 theme:{bp:"mobile"}。上方 45px 畫布標題「C25-M · 設定 — 連線設定（往下捲）· 匿名使用回報（手機）」。比照 A 的做法：刪 Sonarr 卡、保留 Radarr 卡、新增同內容的 usage-report-card（寬度跟隨手機版卡片）。原文方塊必須自動換行、不得水平溢出。

完成後存檔（File ▸ Save）。
```

### 提示詞 2 — 規格稿 C26-D（三種狀態）

```
在 ux-design.pen 的群組 szz7O 裡，於 C25-D 右邊一欄（x = C25-D 的 x + 1540，與 C25-D 同 y）新增一個 1440 寬的畫面，圖框名稱 "C26-D"，fill $bg-secondary，內距 $Space/3xl，垂直排列。上方 45px 畫布標題「C26 · 匿名使用回報・三種狀態（規格）」，字體與樣式完全比照同群組其他畫布標題（字體變數 $Type/Family/Canvas）。
內容：
- 頁首 text「匿名使用回報卡的三種狀態」（Type/H3 三件組、$text-primary），下一行「主稿 C25 是『已開啟、已送過』。下面三種狀態的卡片外觀與 C25 完全相同，只差在開關、提示與讀數。」（Type/Body、$text-secondary）。
- 一列三欄（gap $Space/xl），每欄上方一行小標（Type/Label、$text-muted），下面是一張 Copy 自 C25-D 裡 usage-report-card 的卡（寬度 fill_container）：
  1. 小標「不可用（這個版本沒有設定接收端）」：開關換成 Component/Switch/Off（Qia26）instance 並設 opacity 0.4；提示改為「這個版本沒有設定接收端，無法開啟。」；刪掉 row-last-sent 與 payload。
  2. 小標「關閉（預設）」：開關換成 Switch/Off（Qia26）；提示「預設關閉。關掉之後就不會再送。」；刪掉 row-last-sent 與 payload。
  3. 小標「已開啟、還沒送過」：開關 Switch/On（jqRCX）；保留 row-last-sent，但值改為「還沒送過——打開後一小時內會送出第一份」，字色 $text-muted，字體改回 $Type/Family/Primary；刪掉 payload。
- 三張卡都保留最後的「送什麼、不送什麼 →」連結。
- 高度依內容，所有內容不得被裁切。
完成後存檔（File ▸ Save）。
```

### 提示詞 3 — 精靈 N6-D

```
在 ux-design.pen 的群組 XBRn1 裡，Copy 畫面 D990CP（N4-D）到 N5-D（CWh3E）右邊一欄（x = N5-D 的 x + 1540，同 y），圖框名稱 "N6-D"。上方 45px 畫布標題「N6 · 匿名使用回報（步驟 5／6，在 N5 完成之前）（桌面）」，字體與樣式完全比照同群組其他畫布標題（字體變數 $Type/Family/Canvas）。
只改副本裡 wizard-card 的 step-body（原 d5t5D 的副本）與 step-progress：
1. step-body 內容全部換掉（保留 step-body 本身的排版與間距），依序：
   - 標題 text「匿名使用回報」（同 N4 標題「API 金鑰」的樣式）。
   - 說明 text「要不要每週告訴維護者「Vido 有在幫你做字幕」？只送幾個數字，預設關閉。」（同 N4 說明的樣式）。
   - 開關列（左右排列、垂直置中、space_between）：左邊垂直兩行——標籤「每週送一次匿名計數」（$text-secondary、Type/Body、weight 500）、提示「之後隨時可以在「設定 → 連線設定」改。」（$text-muted、Type/Label）；右邊 Component/Switch/Off（Qia26）instance。
   - 兩欄方塊列（gap $Space/md，兩欄等寬）：每個方塊 fill $bg-primary、圓角 $radius-md、內距 $Space/md、垂直 gap $Space/xs。
     左方塊：標題「會送」（$text-primary、Type/Body、weight 600），三行 Type/Label $text-secondary：「一個隨機編號（不是從你的機器算出來的）」「Vido 版本」「最近 7 天 Vido 自己做出的字幕數」。
     右方塊：標題「絕不送」，三行：「片名、檔名、資料夾路徑」「API 金鑰與任何設定」「你看了、想要或下載了什麼」。
   - 連結 text「看完整說明 →」（$accent-text、Type/Label）。
   - nav：保留「上一步」與「下一步」兩顆按鈕，刪掉「跳過」；「下一步」寬度填滿剩餘空間（比照 N5 的 back＋finish 排法）。
   - 刪掉原本的 skip-warn 提示框。
2. step-progress 改為 6 個點＋5 條線（每段 10px 圓點＋24×2 線，維持原間距）：dot-0~dot-4 與 bar-0~bar-3 為 $accent-primary，bar-4 與 dot-5 為 $bg-tertiary。
3. wizard-card 高度依內容自動，所有內容不得被裁切。
完成後存檔（File ▸ Save）。
```

### 提示詞 4 — 既有精靈稿改為 6 步＋N5 摘要列

```
在 ux-design.pen 修改六張既有精靈稿的步驟點，以及 N5-D 的摘要。只用設計系統變數。
1. 下列每一張的 step-progress，在最後面加一條線（24×2 rectangle，名稱 bar-4）與一個點（10px ellipse，名稱 dot-5），間距與既有一致，變成 6 點 5 線：
   - N1-D（dzgq9）、N2-D（CP7AX）、N3-D（TyjL0）、N3-M（YyaqL）、N4-D（D990CP）、N5-D（CWh3E，其 step-progress 為 UxNGo）。
   新加的 bar-4 與 dot-5 顏色規則：N1～N4 與 N3-M 是「還沒到」→ $bg-tertiary；N5-D 是最後一步「全部完成」→ $accent-primary。其餘既有點與線的顏色一律不動。
2. N5-D 的摘要框 xuXAV：Copy 最後一列 qr7Q7（row-Claude 金鑰）放到它下面，名稱 "row-匿名使用回報"，左邊文字改「匿名使用回報」，右邊文字改「關閉」，右邊字色維持 $text-muted（與「未設定」相同）。
3. 確認六張稿都沒有內容被裁切；N5-D 的 wizard-card 高度跟著摘要多一列而增加。
完成後存檔（File ▸ Save）。
```

---

## Dev Notes

- 提示詞跑完後，依 `feedback-pen-inline-agent-workflow`：先 `git log`／`git hash-object` 確認他是否已經自己 commit；加 `SCREENS` 前先數重複 key。
- 截圖只 stage 真改動的那幾張（CLAUDE.md：整批重產是非決定性的）。
- `-linux` visual 基準不在本單（本單不動 `apps/web`）。

### Time-dependent visual coverage
N/A — no wall-clock-reading components touched（設計稿 only）。

### References
- `DESIGN.md` §怎麼新增一張設計稿（SOP 1–7）
- `.claude/memory/project_pen_flow_layout_convention.md`、`feedback_pencil_spec_standalone_screen.md`、`feedback_verify_pen_saved_before_commit.md`
- 文件範例原文：`docs/usage-report.zh-TW.md`

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-04，SM Bob 建檔＋UX Sally 裁定與提示詞）

### Completion Notes List

- **提示詞 1 review（Sally，2026-10-05，MCP 唯讀）**：C25-D `SHogC`（1440×1346）、C25-M `bG3l3`（390×1468）。卡內 10 段文字與定稿**逐字相同**（含 JSON 原文）；開關為 `jqRCX` instance；讀數與原文用 `$Type/Family/Mono`；桌面 0 個裁切警告；手機唯一一個是分頁列刻意的橫向捲動（既有基準）；`szz7O` 同層重疊 0；`v21FYb` 的 3 組重疊（C5-M 與 dsr-3 規格註記）是既有的，與本單無關。✅ 追認。
- ⚖️ **畫布標題字體**：inline agent 回報我指定的 Noto Sans TC 與隔壁 24 個 caption（`$Type/Family/Canvas`）不一致——DESIGN.md SOP 的文字已經落後於檔案實況。裁定 **B：改成 `$Type/Family/Canvas`**，C25-D／C25-M 的 caption 一併改；提示詞 2–4 已同步改寫。

- **提示詞 3 review（Sally，2026-10-05）**：N6-D `GQae8`。文案 14 段與定稿逐字相同；步驟點 dot-0～4／bar-0～3 為 `$accent-primary`、bar-4／dot-5 為 `$bg-tertiary`；開關 `Qia26`；0 個裁切警告；C25-D／C25-M／C26-D／N6-D 的畫布標題皆已是 `$Type/Family/Canvas`。✅ 追認。
- ⚖️ **兩個對照方塊的底色**：inline agent 回報 `$bg-primary` 與 wizard-card 同色、方塊消失。裁定 **A＋描邊**：方塊改 `$bg-secondary`，並加 `$border-subtle` 1px 描邊——與同一個精靈 N5 摘要框 `xuXAV` 的做法完全一致（`$bg-secondary`＋`$border-subtle`），不另創組合。這是我寫提示詞時沒對照 N5 的疏漏。

### Discovery Triage

- ① DESIGN.md SOP §2 寫的畫布標題字體（Noto Sans TC）與檔案實況（`$Type/Family/Canvas` = DM Sans）不符——已在本單就地更正 DESIGN.md 那一行（2026-10-05）。

### File List

- DESIGN.md（SOP §2 畫布標題字體更正）
