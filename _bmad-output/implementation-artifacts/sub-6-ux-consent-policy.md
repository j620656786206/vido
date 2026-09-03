# Story 6.UX: 「依政策執行」—— 同意是政策不是清單（UX 設計 story，Sally）

Status: ready-for-dev

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

- [ ] **Task 1 — 入口兩張（AC: #1）**
- [ ] **Task 2 — F21 政策確認屏桌機／手機 + 狀態（AC: #2, #4）**
- [ ] **Task 3 — 設定頁政策格（AC: #3）**
- [ ] **Task 4 — 決策 spec 頁（AC: #5）**
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

### Completion Notes List

### Discovery Triage

- （設計時填）

### File List
