# Story 6.12: 同意畫面金錢陷阱 —— 全選範圍、錯誤位置、砍線、總額 ≈、字級（前端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want 全選 to select what I can see, failures to appear where I am looking, and the budget cut line to be visible in the list,
so that the number I consent to is the number I meant.

## Context

critique P1「篩選＋全選＝金錢陷阱」「開始失敗的錯誤埋在清單底」＋ P2「砍線不可見／總額假精準／10px」。

## Acceptance Criteria

1. **全選作用於可見集合。** `handleToggleAll`（`GenerationConsentView.tsx:277-281`）改為對 `visibleIds`（route chip × 搜尋，sub-6-11）操作；工具列文案「已選 x / n」改為「已選 x / 顯示 n（全部 N）」；全選 aria-label「選取顯示的 n 部」。`onSelectAllExtract` 語意不變（本來就是路線集合）。

2. **錯誤搬到看得到的地方。** `startError`（`CandidateListPanel.tsx:402-406`）移到 sticky footer 上方、與超支橫幅同區（`role="alert"`），並保留在清單捲動之外。

3. **砍線可見。** 超支時，在提交順序的第 `feasibleCount+1` 列**之前**畫分隔列「── 到此為止約 $上限，之後的項目會暫停 ──」，其後列 `opacity-60`；橫幅的「約 N 部」加「（清單中已標示）」。排序改變顯示順序時分隔列跟著提交順序的列走（sub-6-11 AC #2）。

4. **總額 ≈。** 已選項目中任一 `runtime_source=fallback` → 摘要、頁尾、確認框三處總額前加 `≈`（同源 selector 加 `hasEstimatedRows`）；確認框 F16 加一行「其中 n 部片長未知，以 45 分鐘估算」。

5. **字級底線。** `text-[10px]`（`:269`、`:274`）→ `text-xs`；內容用途的 `text-[11px]`（`:99`、`:189`、`:491`）→ `text-xs`；殼層用途維持。偵測器 `design-system-font-size` 歸零。

6. **扣誰的錢。** 摘要條末尾加一行「使用：Claude（你的金鑰）· 語音辨識：OpenAI（你的金鑰）／自架」——來源 `AnalysisSnapshot.self_hosted_asr` 與 keys 狀態（`ApiKeysForm` 已有的 `source` 資料）；critique 專案 persona 紅旗。

7. **設計 + 測試。** `.pen` F15/F18 更新（分隔列、錯誤位置、來源行）；重出截圖。specs：全選只選可見、錯誤在 footer 區、砍線位置＝`feasibleCount`、`≈` 三處同步、字級無 10/11px 內容、來源行三種狀態。

## Tasks / Subtasks

- [ ] **Task 1 — 全選語意 + 文案（AC: #1）**
- [ ] **Task 2 — 錯誤區與砍線（AC: #2, #3）**
- [ ] **Task 3 — `≈` 總額與來源行（AC: #4, #6）**
- [ ] **Task 4 — 字級 + 設計更新 + 截圖（AC: #5, #7）**
- [ ] **Task 5 — 測試（AC: #7）**

（全前端。AC #6 若需後端曝露 key source 至 snapshot，為 additive 欄位，可併入 sub-6-10a。）

## Dev Notes

- 「三處金額同源」與「三序同源」兩條紅線都在 `consentSelection.ts`；新旗標與砍線索引都加在 `ConsentTotals`，不得在元件裡另算。
- Rule 21 header 不變。

### Time-dependent visual coverage

- N/A。

### References

- critique P1/P2；`GenerationConsentView.tsx:277-281`、`CandidateListPanel.tsx:191-199,269,274,402-433`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
