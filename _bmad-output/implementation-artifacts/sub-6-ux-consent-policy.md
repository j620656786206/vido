# Story 6.UX: 「依政策執行」—— 同意是政策不是清單（UX 設計 story，Sally）

Status: ready-for-review

<!-- UX design story: deliverable = screens in ux-design.pen + screenshots, not code. -->

## Story

As a BYOK NAS owner,
I want to set one rule for the cheap, safe route and press one button,
so that generating subtitles after a scan no longer means operating a 2,399-row list.

## Brief（已確認，Alexyu 2026-09-04）

`_bmad-output/planning-artifacts/consent-policy-brief-2026-09-03.md` — 三個裁定：**只自動跑抽取路線**、**手動按「依政策執行」**、**清單＝審計＋例外排除**。反目標：不做排程器、不做月上限、不做政策跑 ASR。

## Acceptance Criteria（設計交付）

1. **入口。** 活動頁與 F17 掃描完成卡：「依政策執行」為主按鈕、「批次生成字幕」（現有手動流程）降為次要。政策未設定時主按鈕變「設定政策」。桌機 + 手機各一張。

2. **政策確認屏（F15 的政策變體，命名 F21）。** 頂部一句話＋金額：「將為 N 部可抽取項目產生字幕，約 $X（上限 $Y）」；下方清單**預設收合成群組**（沿用 sub-6-11 的群組樣式），每列可勾掉；勾掉的移到「已排除」段可復原；超限時砍線分隔列（沿用 sub-6-12 AC #3）。主按鈕「開始」。桌機 + 手機（bottom sheet，一句話與「開始」永遠在視野內）。

3. **設定頁「政策」格。** 上限金額、模型（與 sub-6-8b 模型選擇同一元件語彙）、一行說明「只自動處理可抽取（僅翻譯費）的項目；語音辨識仍需逐次同意」。

4. **狀態。** 政策未設定／無可抽取項目（沿用 ConsentEmptyState 不說謊規則）／超限／進行中（沿用 F8）。至少空與超限各一張。

5. **spec 頁（J 系列）。** 一張決策 spec：政策與手動流程的分工圖、「已排除」是否持久化的**兩案並陳**（供裁定 (b)），與 `AI_RUN_BUDGET_USD` 的關係（裁定 (a)）。

6. **交付流程。** `.pen` 新增 frame 於 flow-f-subtitle-v2（或新 block，依 `.claude/memory/project_pen_flow_layout_convention.md`）；`scripts/export-pen-screenshots.py` 的 `SCREENS` 加對應鍵；重出截圖同 commit；標籤不重疊（feedback_pencil_label_overlap）。

## Tasks / Subtasks

- [x] **Task 0 — 設計裁定 + 節點錨定提示詞（本輪產出）**
  - 文件：`sub-6-ux-consent-policy-pen-prompt.md`（九條裁定、10 張畫面規格、共用文案表、提示詞 A–J）
- [x] **Task 1 — 入口（AC: #1）** — F22-D-v2 活動頁（桌機）／F22-M-v2（手機）／F27-D-v2 掃描完成卡
- [x] **Task 2 — F21 政策確認屏桌機／手機 + 狀態（AC: #2, #4）** — F21-D-v2／F21-M-v2／F23-D-v2 超限／F24-D-v2 空／F26-D-v2 政策未設定
- [x] **Task 3 — 設定頁政策格（AC: #3）** — F25-D-v2
- [x] **Task 4 — 決策 spec 頁（AC: #5）** — J8-D
- [x] **Task 5 — export script + 截圖 + commit（AC: #6）**

## Dev Notes

- 這是設計 story；程式 story 由 Bob 在設計 done 後拆（預期：BE 政策設定端點與「依政策執行」批次入口、FE 入口與 F21）。
- 前置零件全部來自 sub-6-1／6-8a/b／6-10a／6-11／6-12，本設計**不另造**列或清單元件。
- 三個未決留給 spec 頁並陳，不在 `.pen` 裡偷偷裁。

### Time-dependent visual coverage

- N/A — 設計交付。

### References

- brief；critique snapshot `.impeccable/critique/2026-09-03T15-07-46Z__apps-web-src-components-subtitle-consent.md`；PRODUCT.md 原則 2／3

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — Sally（UX Designer）帽

### Design Rulings（2026-09-10）

完整規格與提示詞：`_bmad-output/implementation-artifacts/sub-6-ux-consent-policy-pen-prompt.md`

1. **入口是主／次，不是新／舊。** 「依政策執行」為主、「批次生成字幕」降為次但**不得消失** ——
   語音辨識與自選片單只能走它。
2. **政策未設定時主按鈕說下一步。** 變「設定政策」，仍是主按鈕樣式；不畫停用態、不畫紅色。
3. **焦點＝一句話＋金額。** F21 頂部固定三行：`將為 46 部…約 $1.84（上限 $5.00）` →
   `只處理可抽取…需要語音辨識的 96 部不在這次範圍。` → `使用：Claude（你的金鑰）`。
   來源行**只有 Claude 那一半**：政策不跑 ASR，印 OpenAI 等於承諾不會發生的事。
4. **清單預設全部收合。** F21 只有群組列。展開是使用者主動的動作 —— 這是政策屏與 F15
   最大的差別：F15 要你逐項決定，F21 只要你確認總量。
5. **「已排除」是段落不是刪除。** 移到清單底部「已排除（N）」段，每列一個「復原」；
   列淡化 0.6、不畫停用樣式（排除是可逆的決定，不是失效的資料）。
6. **超限沿用 sub-6-12 砍線，一個字都不新造。**
7. **空狀態不說謊且指路。** 政策的空 ≠ F20 的空：文案是「沒有可以自動處理的項目」，
   副句指名去「批次生成字幕」；`search-x` `$text-muted`，**不畫綠勾**。
8. **政策格放「字幕設定」（`/settings/subtitle`）。** 兩個旋鈕（上限、翻譯模型）都是
   「AI 字幕怎麼做出來的」，與在地化程度同源；「媒體庫掃描」持有的是每個媒體庫的資料夾開關。
   落地時**同時要改**該頁副標（現寫「媒體庫的自動字幕請到『媒體庫掃描』」，政策上線後會誤導）。
   模型列沿用 sub-6-8b `ModelPicker` 詞彙，但金額改「每集約 $0.02」——
   設定頁沒有「這批」，印「這批約 $X」是假的。
9. **三個未決在 J8-D 兩案並陳。** 稿面採用的畫法標明為「暫用」，不是裁定。

**數字全部沿用 F15 既有樣本且算式對得起來**：候選 142／可抽取 46 部 $1.84／需 ASR 96 部／
群組 22×$0.04=$0.88 + 24×$0.04=$0.96 = $1.84／已排除 2（24+24−46）。
超限樣本用 brief §2 的真實數字：696 部 $13.92、上限 $5.00、$5.00÷$0.02 = 約 250 部。

### 分工（Alexyu 2026-09-10 裁定）

`.pen` 由 **Alexyu 貼 Pencil Inline AI Agent 執行**（提示詞 A–J，順序 A→B→C→**D**→E→F→G→H→I→J，
E／F 相依於 D）。Claude 負責提示詞、MCP 唯讀複審、export script、截圖、PR。
沿用 `feedback_pen_inline_agent_workflow`。

### Completion Notes List

**交付：10 張新畫面（F21–F27 於 flow-f-subtitle-v2、J8-D 於 flow-j-specs）＋ 10 個 caption。**
AC #1–#6 全部滿足；三個未決依 Dev Notes 未在稿上偷偷裁，全部進 J8-D 兩案並陳。

**執行分工（Alexyu 2026-09-10 裁定）：** Claude 出節點錨定提示詞 A–L，Alexyu 在 Pencil 跑
Inline AI Agent 並逐段 commit（含 `.pen` 與 8 張截圖），Claude 以 MCP 唯讀複審三輪。
沿用 `feedback_pen_inline_agent_workflow`。

**三輪複審抓到的四個問題，四個都是提示詞本身的錯，不是執行的錯：**

| 輪次 | 畫面 | 問題 | 修法 |
| --- | --- | --- | --- |
| 1 | F21-D | 「已排除」兩列帶著複製來源的「未匹配」小標，讀起來像「已排除＝未匹配」；且手機版沒有，同一張稿兩版矛盾 | 提示詞 K：刪掉兩個 `chip-unmatched` |
| 1 | F23-D | 砍線下方的「影集（12 部）」群組沒淡化，與稿子自己的註腳「線以上會跑，線以下會等」矛盾 | 提示詞 K：`group-影集` opacity 0.6 |
| 1 | F23-D | 影片列標籤寫「抽取」，F21 寫「僅翻譯費」——同流程兩套詞 | 提示詞 K：五列統一為「僅翻譯費」 |
| 2 | F27-D | 「批次生成字幕 →」被我指定成灰色，比同卡的「查看錯誤」還淡——等於用顏色藏掉 ASR 唯一的入口，牴觸裁定 1 | 提示詞 L：改 `$accent-primary` |

**驗收證據（MCP 唯讀）：**
- 10 張畫面 + 10 個 caption 座標全對；F 系列 caption 在框上 45px，J8-D 沿用 J 列既有的 30px。
- 文案逐字比對提示詞文件的文案表，全對。
- `ctx.problems` 僅剩活動頁背景「部分裁切」——`f15-d-v2`／`f15-m-v2` 本來就有（頁面比 900 高的視窗長）。
- **F21-M 的硬要求通過**：`headline` 在螢幕 y≈170、`開始` 在 y≈788–832，844 高的視野內同框。
- J8-D 底部 y=25327，離下一個區塊（y=27381）2,054px，無碰撞。
- 數字算式自洽：46×$0.04=$1.84、22+24=46、24+24−46=2；超限 412×$0.02=$8.24、284×$0.02=$5.68、
  合計 696 部 $13.92、$5.00÷$0.02≈250 部。

**已知的樣本差異（不是缺陷，記著避免被當成矛盾）：** F22-D 的政策行寫 Claude Sonnet 4.5，
而 F23 的每列單價是 $0.02（Haiku 價）。F23 用的是 brief §2 的真實痛點數字（696 部＝$13.92），
刻意不為了對齊模型名而改掉真實數字。

**落地時要一起做的事（給 Bob 拆碼 story 用）：**
`/settings/subtitle` 的副標現在寫「媒體庫的自動字幕請到『媒體庫掃描』」，政策格上線後這句會誤導，
必須同時改。另 `LibraryEditModal.tsx:337` 寫「需要 AI 翻譯或語音辨識的影片不會自動處理」——
政策仍是手動按的，所以這句還成立，但措辭要複查一次。

### Discovery Triage

- 無新發現的產品缺陷。三個未決（(a) 政策上限與 `AI_RUN_BUDGET_USD`、(b)「已排除」是否持久化、
  (c) 是否顯示上次花費）不是 discovery，是 brief 明列、本 story 明訂要兩案並陳的待裁項，
  owner = Alexyu，載體 = J8-D。**裁定前不得有任何 story 引用其中一案為既成事實。**

### File List

- `ux-design.pen` — F21-D/F21-M/F22-D/F22-M/F23-D/F24-D/F25-D/F26-D/F27-D + J8-D，共 10 frame + 10 caption
- `_bmad-output/screenshots/flow-f-subtitle-v2/` — f21-d-v2、f21-m-v2、f22-d-v2、f22-m-v2、
  f23-d-v2、f24-d-v2、f25-d-v2、f26-d-v2、f27-d-v2（9 張）
- `_bmad-output/screenshots/flow-j-specs/j8-d.png`
- `scripts/export-pen-screenshots.py` — `SCREENS` +10 筆
- `_bmad-output/implementation-artifacts/sub-6-ux-consent-policy-pen-prompt.md` — 裁定 + 提示詞 A–L + 三輪複審紀錄
- `_bmad-output/implementation-artifacts/sub-6-ux-consent-policy.md` — 本檔
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
