# Story 6.UX: 「依政策執行」—— 同意是政策不是清單（UX 設計 story，Sally）

Status: in-progress

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
- [ ] **Task 1 — 入口（AC: #1）** — F22-D-v2 活動頁（桌機）／F22-M-v2（手機）／F27-D-v2 掃描完成卡
- [ ] **Task 2 — F21 政策確認屏桌機／手機 + 狀態（AC: #2, #4）** — F21-D-v2／F21-M-v2／F23-D-v2 超限／F24-D-v2 空／F26-D-v2 政策未設定
- [ ] **Task 3 — 設定頁政策格（AC: #3）** — F25-D-v2
- [ ] **Task 4 — 決策 spec 頁（AC: #5）** — J8-D
- [ ] **Task 5 — export script + 截圖 + commit（AC: #6）**

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

### Discovery Triage

- （設計時填）

### File List
